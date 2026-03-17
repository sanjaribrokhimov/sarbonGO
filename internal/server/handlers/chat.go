package handlers

import (
	"context"
	"fmt"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"sarbonNew/internal/chat"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type ChatHandler struct {
	logger   *zap.Logger
	repo     *chat.Repo
	presence *chat.PresenceStore
	hub      *chat.Hub
}

func NewChatHandler(logger *zap.Logger, repo *chat.Repo, presence *chat.PresenceStore, hub *chat.Hub) *ChatHandler {
	h := &ChatHandler{logger: logger, repo: repo, presence: presence, hub: hub}
	hub.SetOnTyping(func(conversationID, fromUserID uuid.UUID) (uuid.UUID, bool) {
		ctx := context.Background()
		conv, err := repo.GetConversation(ctx, conversationID, fromUserID)
		if err != nil || conv == nil {
			return uuid.Nil, false
		}
		_ = presence.SetTyping(ctx, conversationID, fromUserID)
		return conv.PeerID(fromUserID), true
	})
	return h
}

func (h *ChatHandler) getUserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(mw.CtxUserID)
	if !ok {
		return uuid.Nil, false
	}
	id, _ := v.(uuid.UUID)
	return id, id != uuid.Nil
}

// ListConversations returns conversations for the current user.
// GET /v1/chat/conversations
func (h *ChatHandler) ListConversations(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		resp.ErrorLang(c, http.StatusUnauthorized, "user_not_identified")
		return
	}
	limit := getIntQuery(c, "limit", 50)
	list, err := h.repo.ListConversations(c.Request.Context(), userID, limit)
	if err != nil {
		h.logger.Error("chat list conversations", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_list_conversations")
		return
	}
	// Enrich with peer_id for each
	out := make([]gin.H, 0, len(list))
	for _, conv := range list {
		peerID := conv.PeerID(userID)
		out = append(out, gin.H{
			"id":         conv.ID,
			"peer_id":   peerID,
			"created_at": conv.CreatedAt,
		})
	}
	resp.OKLang(c, "ok", gin.H{"conversations": out})
}

// GetOrCreateConversation gets or creates a conversation with peer_id.
// POST /v1/chat/conversations body: { "peer_id": "uuid" }
func (h *ChatHandler) GetOrCreateConversation(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		resp.ErrorLang(c, http.StatusUnauthorized, "user_not_identified")
		return
	}
	var req struct {
		PeerID string `json:"peer_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	peerID, err := uuid.Parse(req.PeerID)
	if err != nil || peerID == uuid.Nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_peer_id")
		return
	}
	conv, err := h.repo.GetOrCreateConversation(c.Request.Context(), userID, peerID)
	if err != nil {
		if err == chat.ErrSameUser {
			resp.ErrorLang(c, http.StatusBadRequest, "cannot_chat_with_yourself")
			return
		}
		h.logger.Error("chat get or create conversation", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_create_conversation")
		return
	}
	peerID = conv.PeerID(userID)
	resp.OKLang(c, "ok", gin.H{
		"id":         conv.ID,
		"peer_id":   peerID,
		"created_at": conv.CreatedAt,
	})
}

// ListMessages returns messages for a conversation (paginated by cursor).
// GET /v1/chat/conversations/:id/messages?limit=20&cursor=uuid
func (h *ChatHandler) ListMessages(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		resp.ErrorLang(c, http.StatusUnauthorized, "user_not_identified")
		return
	}
	convIDStr := c.Param("id")
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_conversation_id")
		return
	}
	var cursor *uuid.UUID
	if c := c.Query("cursor"); c != "" {
		u, err := uuid.Parse(c)
		if err == nil {
			cursor = &u
		}
	}
	limit := getIntQuery(c, "limit", 50)
	list, err := h.repo.ListMessages(c.Request.Context(), convID, userID, cursor, limit)
	if err != nil {
		h.logger.Error("chat list messages", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_list_messages")
		return
	}
	resp.OKLang(c, "ok", gin.H{"messages": list})
}

// SendMessage creates a message and broadcasts via WebSocket.
// POST /v1/chat/conversations/:id/messages body: { "body": "text" }
func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		resp.ErrorLang(c, http.StatusUnauthorized, "user_not_identified")
		return
	}
	convIDStr := c.Param("id")
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_conversation_id")
		return
	}
	var req struct {
		Body string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	if len(req.Body) > 64*1024 {
		resp.ErrorLang(c, http.StatusBadRequest, "message_too_long")
		return
	}
	conv, err := h.repo.GetConversation(c.Request.Context(), convID, userID)
	if err != nil || conv == nil {
		resp.ErrorLang(c, http.StatusNotFound, "conversation_not_found")
		return
	}
	msg, err := h.repo.CreateTextMessage(c.Request.Context(), convID, userID, req.Body)
	if err != nil {
		h.logger.Error("chat create message", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_send_message")
		return
	}
	h.hub.BroadcastMessage(conv.UserAID, conv.UserBID, msg)
	resp.SuccessLang(c, http.StatusCreated, "ok", msg)
}

type mediaMsgPayload struct {
	AttachmentID string `json:"attachment_id"`
	URL          string `json:"url"`
	ThumbURL     string `json:"thumb_url,omitempty"`
	Mime         string `json:"mime"`
	SizeBytes    int64  `json:"size_bytes"`
	DurationMs   *int   `json:"duration_ms,omitempty"`
	Width        *int   `json:"width,omitempty"`
	Height       *int   `json:"height,omitempty"`
}

// SendMediaMessage uploads media (photo/voice/video/video_note), creates message and broadcasts.
// POST /v1/chat/conversations/:id/messages/media (multipart/form-data)
func (h *ChatHandler) SendMediaMessage(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		resp.ErrorLang(c, http.StatusUnauthorized, "user_not_identified")
		return
	}
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_conversation_id")
		return
	}
	conv, err := h.repo.GetConversation(c.Request.Context(), convID, userID)
	if err != nil || conv == nil {
		resp.ErrorLang(c, http.StatusNotFound, "conversation_not_found")
		return
	}

	msgType := strings.ToUpper(strings.TrimSpace(c.PostForm("type")))
	switch msgType {
	case "PHOTO", "VOICE", "VIDEO", "VIDEO_NOTE":
	default:
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}

	var caption *string
	if v := strings.TrimSpace(c.PostForm("body")); v != "" {
		if len(v) > 64*1024 {
			resp.ErrorLang(c, http.StatusBadRequest, "message_too_long")
			return
		}
		caption = &v
	}

	fh, err := c.FormFile("file")
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	if fh.Size <= 0 {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}

	storageRoot := strings.TrimSpace(os.Getenv("CHAT_STORAGE_DIR"))
	if storageRoot == "" {
		storageRoot = "storage"
	}
	baseDir := filepath.Join(storageRoot, "chat", convID.String())
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		h.logger.Error("chat mkdir", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}

	inPath := filepath.Join(baseDir, "upload_"+uuid.New().String()+"_"+filepath.Base(fh.Filename))
	src, err := fh.Open()
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "cannot_read_file")
		return
	}
	defer src.Close()
	dst, err := os.Create(inPath)
	if err != nil {
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	_ = dst.Close()

	outExt := map[string]string{
		"PHOTO":      ".jpg",
		"VOICE":      ".ogg",
		"VIDEO":      ".mp4",
		"VIDEO_NOTE": ".mp4",
	}[msgType]
	outPath := filepath.Join(baseDir, msgType+"_"+uuid.New().String()+outExt)
	thumbPath := ""

	var durationMs *int
	var width, height *int
	var mime string

	switch msgType {
	case "VOICE":
		if err := ffmpegVoiceToOggOpus(inPath, outPath); err != nil {
			h.logger.Error("ffmpeg voice", zap.Error(err))
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
			return
		}
		mime = "audio/ogg"
		if ms, _ := ffprobeDurationMs(outPath); ms != nil {
			durationMs = ms
		}
	case "VIDEO":
		if err := ffmpegVideoToMp4(inPath, outPath, false); err != nil {
			h.logger.Error("ffmpeg video", zap.Error(err))
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
			return
		}
		mime = "video/mp4"
		if ms, _ := ffprobeDurationMs(outPath); ms != nil {
			durationMs = ms
		}
		thumbPath = filepath.Join(baseDir, "thumb_"+uuid.New().String()+".jpg")
		_ = ffmpegThumbnail(outPath, thumbPath)
	case "VIDEO_NOTE":
		if err := ffmpegVideoToMp4(inPath, outPath, true); err != nil {
			h.logger.Error("ffmpeg video_note", zap.Error(err))
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
			return
		}
		mime = "video/mp4"
		if ms, _ := ffprobeDurationMs(outPath); ms != nil {
			durationMs = ms
		}
		thumbPath = filepath.Join(baseDir, "thumb_"+uuid.New().String()+".jpg")
		_ = ffmpegThumbnail(outPath, thumbPath)
	case "PHOTO":
		if err := ffmpegImageToJpeg(inPath, outPath); err != nil {
			h.logger.Error("ffmpeg photo", zap.Error(err))
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
			return
		}
		mime = "image/jpeg"
	}

	stat, err := os.Stat(outPath)
	if err != nil {
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}

	hashHex, err := sha256FileHex(outPath)
	if err != nil {
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}

	// Store deduplicated file path: storage/chat/media/<prefix>/<hash>.<ext>
	mediaDir := filepath.Join(storageRoot, "chat", "media", hashHex[:2])
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	finalRelPath := filepath.Join(storageRoot, "chat", "media", hashHex[:2], hashHex+outExt)
	finalPath := finalRelPath

	mediaID, inserted, err := h.repo.UpsertMediaFile(c.Request.Context(), chat.MediaFile{
		ContentHash: hashHex,
		Kind:        msgType,
		Mime:        mime,
		SizeBytes:   stat.Size(),
		Path:        finalRelPath,
	})
	if err != nil {
		h.logger.Error("chat upsert media file", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_send_message")
		return
	}
	if inserted {
		// Move optimized file into hash-based path.
		_ = os.Rename(outPath, finalPath)
	} else {
		// Duplicate: remove newly created file to save disk.
		_ = os.Remove(outPath)
	}

	var thumbMediaID *uuid.UUID
	var thumbRel *string
	if thumbPath != "" {
		// Thumbnail is optional; we keep it per-upload (no dedup required), but store in legacy column and optionally media_files too.
		rel := thumbPath
		thumbRel = &rel
	}
	rel := finalRelPath
	attID, err := h.repo.CreateAttachment(c.Request.Context(), chat.Attachment{
		MessageID:      nil,
		ConversationID: convID,
		UploaderID:     userID,
		Kind:           msgType,
		Mime:           mime,
		SizeBytes:      stat.Size(),
		Path:           rel,
		ThumbPath:      thumbRel,
		Width:          width,
		Height:         height,
		DurationMs:     durationMs,
		MediaFileID:    &mediaID,
		ThumbMediaFileID: thumbMediaID,
	})
	if err != nil {
		h.logger.Error("chat create attachment", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_send_message")
		return
	}

	payload := mediaMsgPayload{
		AttachmentID: attID.String(),
		URL:          "/v1/chat/files/" + attID.String(),
		Mime:         mime,
		SizeBytes:    stat.Size(),
		DurationMs:   durationMs,
		Width:        width,
		Height:       height,
	}
	if thumbRel != nil {
		payload.ThumbURL = "/v1/chat/files/" + attID.String() + "?thumb=1"
	}

	msg, err := h.repo.CreateMessage(c.Request.Context(), convID, userID, msgType, caption, payload)
	if err != nil {
		h.logger.Error("chat create media message", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_send_message")
		return
	}
	_ = h.repo.LinkAttachment(c.Request.Context(), attID, msg.ID)

	h.hub.BroadcastMessage(conv.UserAID, conv.UserBID, msg)
	resp.SuccessLang(c, http.StatusCreated, "ok", msg)
}

// GetFile serves a chat attachment file (or thumbnail) if requester is participant.
// GET /v1/chat/files/:id?thumb=1
func (h *ChatHandler) GetFile(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		resp.ErrorLang(c, http.StatusUnauthorized, "user_not_identified")
		return
	}
	attID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	a, err := h.repo.GetAttachmentForUser(c.Request.Context(), attID, userID)
	if err != nil {
		resp.ErrorLang(c, http.StatusNotFound, "photo_not_found")
		return
	}
	path := a.Path
	etag := ""
	if a.MediaFileID != nil {
		if mf, err := h.repo.GetMediaFileByID(c.Request.Context(), *a.MediaFileID); err == nil && mf != nil {
			path = mf.Path
			etag = mf.ContentHash
		}
	}
	if c.Query("thumb") == "1" && a.ThumbPath != nil && *a.ThumbPath != "" {
		path = *a.ThumbPath
	}
	if _, err := os.Stat(path); err != nil {
		resp.ErrorLang(c, http.StatusNotFound, "photo_not_found")
		return
	}

	// Caching headers (immutable for stored media).
	if etag != "" {
		c.Header("ETag", `"`+etag+`"`)
		if inm := strings.TrimSpace(c.GetHeader("If-None-Match")); inm != "" && inm == `"`+etag+`"` {
			c.Status(http.StatusNotModified)
			return
		}
	}
	c.Header("Cache-Control", "public, max-age=31536000, immutable")

	// Prefer Nginx X-Accel-Redirect for efficient range/caching.
	if strings.TrimSpace(os.Getenv("CHAT_USE_X_ACCEL")) == "1" {
		prefix := strings.TrimSpace(os.Getenv("CHAT_X_ACCEL_PREFIX"))
		if prefix == "" {
			prefix = "/_protected"
		}
		// Important: Nginx should map prefix + "/" + path to filesystem via alias.
		c.Header("X-Accel-Redirect", prefix+"/"+strings.TrimLeft(path, "/"))
		c.Status(http.StatusOK)
		return
	}

	// Fallback: serve from Go (slower than Nginx for large video).
	c.File(path)
}

func ffmpegVoiceToOggOpus(inPath, outPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", inPath, "-vn", "-c:a", "libopus", "-b:a", "32k", "-vbr", "on", "-compression_level", "10", "-ac", "1", "-ar", "48000", outPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, string(out))
	}
	return nil
}

func ffmpegVideoToMp4(inPath, outPath string, square bool) error {
	args := []string{"-y", "-i", inPath, "-c:v", "libx264", "-preset", "veryfast", "-crf", "28", "-c:a", "aac", "-b:a", "96k", "-movflags", "+faststart"}
	if square {
		// Make a square "video note"-like output.
		// scale to min dim, then crop center to 480x480.
		args = append(args, "-vf", "scale='if(gt(iw,ih),-2,480)':'if(gt(iw,ih),480,-2)',crop=480:480")
	} else {
		args = append(args, "-vf", "scale='min(1280,iw)':-2")
	}
	args = append(args, outPath)
	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, string(out))
	}
	return nil
}

func ffmpegImageToJpeg(inPath, outPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", inPath, "-q:v", "3", outPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, string(out))
	}
	return nil
}

func ffmpegThumbnail(inPath, outPath string) error {
	cmd := exec.Command("ffmpeg", "-y", "-ss", "00:00:01", "-i", inPath, "-vframes", "1", "-q:v", "4", outPath)
	_, _ = cmd.CombinedOutput()
	return nil
}

func ffprobeDurationMs(path string) (*int, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	ms := int(f * 1000)
	return &ms, nil
}

func sha256FileHex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// EditMessage updates a message.
// PATCH /v1/chat/messages/:id body: { "body": "new text" }
func (h *ChatHandler) EditMessage(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		resp.ErrorLang(c, http.StatusUnauthorized, "user_not_identified")
		return
	}
	msgIDStr := c.Param("id")
	msgID, err := uuid.Parse(msgIDStr)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_message_id")
		return
	}
	var req struct {
		Body string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	if len(req.Body) > 64*1024 {
		resp.ErrorLang(c, http.StatusBadRequest, "message_too_long")
		return
	}
	msg, err := h.repo.UpdateMessage(c.Request.Context(), msgID, userID, req.Body)
	if err != nil {
		resp.ErrorLang(c, http.StatusNotFound, "message_not_found")
		return
	}
	resp.OKLang(c, "ok", msg)
}

// DeleteMessage soft-deletes a message.
// DELETE /v1/chat/messages/:id
func (h *ChatHandler) DeleteMessage(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		resp.ErrorLang(c, http.StatusUnauthorized, "user_not_identified")
		return
	}
	msgIDStr := c.Param("id")
	msgID, err := uuid.Parse(msgIDStr)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_message_id")
		return
	}
	if err := h.repo.DeleteMessage(c.Request.Context(), msgID, userID); err != nil {
		if err == chat.ErrNotFound {
			resp.ErrorLang(c, http.StatusNotFound, "message_not_found")
			return
		}
		h.logger.Error("chat delete message", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_delete_message")
		return
	}
	resp.OKLang(c, "ok", gin.H{"deleted": true})
}

// GetPresence returns online/last_seen (and optionally typing) for a user.
// GET /v1/chat/presence/:user_id?conversation_id=uuid
func (h *ChatHandler) GetPresence(c *gin.Context) {
	userIDStr := c.Param("user_id")
	targetID, err := uuid.Parse(userIDStr)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_user_id")
		return
	}
	var convID *uuid.UUID
	if c := c.Query("conversation_id"); c != "" {
		u, err := uuid.Parse(c)
		if err == nil {
			convID = &u
		}
	}
	pres, err := h.presence.GetPresence(c.Request.Context(), targetID, convID)
	if err != nil {
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_get_presence")
		return
	}
	resp.OKLang(c, "ok", pres)
}

// ServeWS upgrades connection to WebSocket and runs the client (read/write pumps).
// GET /v1/chat/ws?token=JWT or ?user_id=UUID (for Swagger)
func (h *ChatHandler) ServeWS(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		resp.ErrorLang(c, http.StatusUnauthorized, "user_not_identified")
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Debug("chat ws upgrade failed", zap.Error(err))
		return
	}
	client := h.hub.Register(userID, conn)
	go client.WritePump()
	client.ReadPump()
}
