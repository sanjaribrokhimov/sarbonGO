package handlers

import (
	"context"
	"net/http"

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
	msg, err := h.repo.CreateMessage(c.Request.Context(), convID, userID, req.Body)
	if err != nil {
		h.logger.Error("chat create message", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_send_message")
		return
	}
	h.hub.BroadcastMessage(conv.UserAID, conv.UserBID, msg)
	resp.SuccessLang(c, http.StatusCreated, "ok", msg)
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
