package chat

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Attachment struct {
	ID             uuid.UUID
	MessageID      *uuid.UUID
	ConversationID uuid.UUID
	UploaderID     uuid.UUID
	Kind           string
	Mime           string
	SizeBytes      int64
	Path           string
	ThumbPath      *string
	Width          *int
	Height         *int
	DurationMs     *int
	MediaFileID    *uuid.UUID
	ThumbMediaFileID *uuid.UUID
}

type MediaFile struct {
	ID          uuid.UUID
	ContentHash string
	Kind        string
	Mime        string
	SizeBytes   int64
	Path        string
}

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

// GetOrCreateConversation returns existing conversation or creates one. userID = current user, peerID = other.
func (r *Repo) GetOrCreateConversation(ctx context.Context, userID, peerID uuid.UUID) (*Conversation, error) {
	if userID == peerID {
		return nil, ErrSameUser
	}
	u1, u2 := userID, peerID
	if u1.String() > u2.String() {
		u1, u2 = u2, u1
	}
	var c Conversation
	err := r.pg.QueryRow(ctx, `
INSERT INTO chat_conversations (user_a_id, user_b_id)
VALUES ($1, $2)
ON CONFLICT (user_a_id, user_b_id) DO UPDATE SET user_a_id = chat_conversations.user_a_id
RETURNING id, user_a_id, user_b_id, created_at
`, u1, u2).Scan(&c.ID, &c.UserAID, &c.UserBID, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListConversations returns conversations for user (newest first by last message).
func (r *Repo) ListConversations(ctx context.Context, userID uuid.UUID, limit int) ([]Conversation, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.pg.Query(ctx, `
SELECT c.id, c.user_a_id, c.user_b_id, c.created_at
FROM chat_conversations c
WHERE c.user_a_id = $1 OR c.user_b_id = $1
ORDER BY (
  SELECT COALESCE(MAX(m.created_at), c.created_at)
  FROM chat_messages m
  WHERE m.conversation_id = c.id AND m.deleted_at IS NULL
) DESC
LIMIT $2
`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.UserAID, &c.UserBID, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// GetConversation loads one conversation by ID if user is participant.
func (r *Repo) GetConversation(ctx context.Context, conversationID, userID uuid.UUID) (*Conversation, error) {
	var c Conversation
	err := r.pg.QueryRow(ctx, `
SELECT id, user_a_id, user_b_id, created_at
FROM chat_conversations
WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2)
`, conversationID, userID).Scan(&c.ID, &c.UserAID, &c.UserBID, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListMessages returns messages for conversation (desc created_at), cursor = message ID for next page.
func (r *Repo) ListMessages(ctx context.Context, conversationID, userID uuid.UUID, cursor *uuid.UUID, limit int) ([]Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows pgx.Rows
	var err error
	if cursor == nil || *cursor == uuid.Nil {
		rows, err = r.pg.Query(ctx, `
SELECT m.id, m.conversation_id, m.sender_id, m.type, m.body, m.payload, m.created_at, m.updated_at, m.deleted_at
FROM chat_messages m
JOIN chat_conversations c ON c.id = m.conversation_id AND (c.user_a_id = $2 OR c.user_b_id = $2)
WHERE m.conversation_id = $1 AND m.deleted_at IS NULL
ORDER BY m.created_at DESC
LIMIT $3
`, conversationID, userID, limit)
	} else {
		rows, err = r.pg.Query(ctx, `
SELECT m.id, m.conversation_id, m.sender_id, m.type, m.body, m.payload, m.created_at, m.updated_at, m.deleted_at
FROM chat_messages m
JOIN chat_conversations c ON c.id = m.conversation_id AND (c.user_a_id = $2 OR c.user_b_id = $2)
WHERE m.conversation_id = $1 AND m.deleted_at IS NULL
AND m.created_at < (SELECT created_at FROM chat_messages WHERE id = $4)
ORDER BY m.created_at DESC
LIMIT $3
`, conversationID, userID, limit, *cursor)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Message
	for rows.Next() {
		var m Message
		var payloadBytes []byte
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Type, &m.Body, &payloadBytes, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt); err != nil {
			return nil, err
		}
		if len(payloadBytes) > 0 {
			m.Payload = json.RawMessage(payloadBytes)
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// CreateTextMessage inserts a TEXT message and returns it.
func (r *Repo) CreateTextMessage(ctx context.Context, conversationID, senderID uuid.UUID, body string) (*Message, error) {
	if strings.TrimSpace(body) == "" {
		return nil, ErrInvalidBody
	}
	return r.CreateMessage(ctx, conversationID, senderID, "TEXT", &body, nil)
}

// CreateMessage inserts a message with type/body/payload and returns it.
func (r *Repo) CreateMessage(ctx context.Context, conversationID, senderID uuid.UUID, msgType string, body *string, payload any) (*Message, error) {
	var m Message
	var payloadJSON []byte
	if payload != nil {
		payloadJSON, _ = json.Marshal(payload)
	}
	err := r.pg.QueryRow(ctx, `
INSERT INTO chat_messages (conversation_id, sender_id, type, body, payload)
SELECT $1, $2, $3, $4, $5
FROM chat_conversations
WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2)
RETURNING id, conversation_id, sender_id, type, body, payload, created_at, updated_at, deleted_at
`, conversationID, senderID, msgType, body, payloadJSON).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Type, &m.Body, &payloadJSON, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
	if err != nil {
		return nil, err
	}
	if len(payloadJSON) > 0 {
		m.Payload = json.RawMessage(payloadJSON)
	}
	return &m, nil
}

// UpdateMessage updates body if message belongs to sender.
func (r *Repo) UpdateMessage(ctx context.Context, messageID, senderID uuid.UUID, body string) (*Message, error) {
	var m Message
	var payloadBytes []byte
	err := r.pg.QueryRow(ctx, `
UPDATE chat_messages SET body = $3, updated_at = now()
WHERE id = $1 AND sender_id = $2 AND deleted_at IS NULL
RETURNING id, conversation_id, sender_id, type, body, payload, created_at, updated_at, deleted_at
`, messageID, senderID, body).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Type, &m.Body, &payloadBytes, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
	if err != nil {
		return nil, err
	}
	if len(payloadBytes) > 0 {
		m.Payload = json.RawMessage(payloadBytes)
	}
	return &m, nil
}

// DeleteMessage soft-deletes message if sender.
func (r *Repo) DeleteMessage(ctx context.Context, messageID, senderID uuid.UUID) error {
	cmd, err := r.pg.Exec(ctx, `
UPDATE chat_messages SET deleted_at = now(), updated_at = now()
WHERE id = $1 AND sender_id = $2 AND deleted_at IS NULL
`, messageID, senderID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetMessageByID returns message if it exists and user is in conversation.
func (r *Repo) GetMessageByID(ctx context.Context, messageID, userID uuid.UUID) (*Message, error) {
	var m Message
	var payloadBytes []byte
	err := r.pg.QueryRow(ctx, `
SELECT m.id, m.conversation_id, m.sender_id, m.type, m.body, m.payload, m.created_at, m.updated_at, m.deleted_at
FROM chat_messages m
JOIN chat_conversations c ON c.id = m.conversation_id
WHERE m.id = $1 AND (c.user_a_id = $2 OR c.user_b_id = $2)
`, messageID, userID).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Type, &m.Body, &payloadBytes, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
	if err != nil {
		return nil, err
	}
	if len(payloadBytes) > 0 {
		m.Payload = json.RawMessage(payloadBytes)
	}
	return &m, nil
}

func (r *Repo) CreateAttachment(ctx context.Context, a Attachment) (uuid.UUID, error) {
	const q = `
INSERT INTO chat_attachments (message_id, conversation_id, uploader_id, kind, mime, size_bytes, path, thumb_path, width, height, duration_ms, media_file_id, thumb_media_file_id)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
RETURNING id`
	var id uuid.UUID
	err := r.pg.QueryRow(ctx, q,
		a.MessageID, a.ConversationID, a.UploaderID, a.Kind, a.Mime, a.SizeBytes, a.Path, a.ThumbPath, a.Width, a.Height, a.DurationMs, a.MediaFileID, a.ThumbMediaFileID,
	).Scan(&id)
	return id, err
}

// LinkAttachment sets attachment.message_id after message is created.
func (r *Repo) LinkAttachment(ctx context.Context, attachmentID, messageID uuid.UUID) error {
	_, err := r.pg.Exec(ctx, `UPDATE chat_attachments SET message_id = $2 WHERE id = $1`, attachmentID, messageID)
	return err
}

// GetAttachmentForUser returns attachment if user is participant in its conversation.
func (r *Repo) GetAttachmentForUser(ctx context.Context, attachmentID, userID uuid.UUID) (*Attachment, error) {
	const q = `
SELECT a.id, a.message_id, a.conversation_id, a.uploader_id, a.kind, a.mime, a.size_bytes, a.path, a.thumb_path, a.width, a.height, a.duration_ms, a.media_file_id, a.thumb_media_file_id
FROM chat_attachments a
JOIN chat_conversations c ON c.id = a.conversation_id
WHERE a.id = $1 AND (c.user_a_id = $2 OR c.user_b_id = $2)
LIMIT 1`
	var a Attachment
	err := r.pg.QueryRow(ctx, q, attachmentID, userID).Scan(
		&a.ID, &a.MessageID, &a.ConversationID, &a.UploaderID, &a.Kind, &a.Mime, &a.SizeBytes, &a.Path, &a.ThumbPath, &a.Width, &a.Height, &a.DurationMs, &a.MediaFileID, &a.ThumbMediaFileID,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repo) UpsertMediaFile(ctx context.Context, f MediaFile) (uuid.UUID, bool, error) {
	const q = `
INSERT INTO media_files (content_hash, kind, mime, size_bytes, path)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (content_hash) DO UPDATE SET content_hash = media_files.content_hash
RETURNING id, (xmax = 0) AS inserted`
	var id uuid.UUID
	var inserted bool
	err := r.pg.QueryRow(ctx, q, f.ContentHash, f.Kind, f.Mime, f.SizeBytes, f.Path).Scan(&id, &inserted)
	return id, inserted, err
}

func (r *Repo) GetMediaFileByID(ctx context.Context, id uuid.UUID) (*MediaFile, error) {
	const q = `SELECT id, content_hash, kind, mime, size_bytes, path FROM media_files WHERE id=$1 LIMIT 1`
	var f MediaFile
	err := r.pg.QueryRow(ctx, q, id).Scan(&f.ID, &f.ContentHash, &f.Kind, &f.Mime, &f.SizeBytes, &f.Path)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ListOrphanMediaFiles returns media files not referenced by any attachment.
func (r *Repo) ListOrphanMediaFiles(ctx context.Context, olderThanDays int, limit int) ([]MediaFile, error) {
	if olderThanDays <= 0 {
		olderThanDays = 30
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	const q = `
SELECT f.id, f.content_hash, f.kind, f.mime, f.size_bytes, f.path
FROM media_files f
LEFT JOIN chat_attachments a ON a.media_file_id = f.id OR a.thumb_media_file_id = f.id
WHERE a.id IS NULL AND f.created_at < now() - ($1::int || ' days')::interval
ORDER BY f.created_at ASC
LIMIT $2`
	rows, err := r.pg.Query(ctx, q, olderThanDays, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MediaFile
	for rows.Next() {
		var f MediaFile
		if err := rows.Scan(&f.ID, &f.ContentHash, &f.Kind, &f.Mime, &f.SizeBytes, &f.Path); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repo) DeleteMediaFile(ctx context.Context, id uuid.UUID) error {
	_, err := r.pg.Exec(ctx, `DELETE FROM media_files WHERE id=$1`, id)
	return err
}

type ExpiredAttachment struct {
	AttachmentID uuid.UUID
	MediaFileID  *uuid.UUID
	ThumbMediaFileID *uuid.UUID
	Path         string
	ThumbPath    *string
}

// ListExpiredDeletedMessageAttachments returns attachments whose message was soft-deleted long ago.
func (r *Repo) ListExpiredDeletedMessageAttachments(ctx context.Context, olderThanDays int, limit int) ([]ExpiredAttachment, error) {
	if olderThanDays <= 0 {
		olderThanDays = 30
	}
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	const q = `
SELECT a.id, a.media_file_id, a.thumb_media_file_id, a.path, a.thumb_path
FROM chat_attachments a
JOIN chat_messages m ON m.id = a.message_id
WHERE m.deleted_at IS NOT NULL
  AND m.deleted_at < now() - ($1::int || ' days')::interval
ORDER BY m.deleted_at ASC
LIMIT $2`
	rows, err := r.pg.Query(ctx, q, olderThanDays, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExpiredAttachment
	for rows.Next() {
		var e ExpiredAttachment
		if err := rows.Scan(&e.AttachmentID, &e.MediaFileID, &e.ThumbMediaFileID, &e.Path, &e.ThumbPath); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repo) DeleteAttachment(ctx context.Context, attachmentID uuid.UUID) error {
	_, err := r.pg.Exec(ctx, `DELETE FROM chat_attachments WHERE id=$1`, attachmentID)
	return err
}
