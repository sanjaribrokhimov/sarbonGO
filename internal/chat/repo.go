package chat

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
SELECT id, conversation_id, sender_id, body, created_at, updated_at, deleted_at
FROM chat_messages m
JOIN chat_conversations c ON c.id = m.conversation_id AND (c.user_a_id = $2 OR c.user_b_id = $2)
WHERE m.conversation_id = $1 AND m.deleted_at IS NULL
ORDER BY m.created_at DESC
LIMIT $3
`, conversationID, userID, limit)
	} else {
		rows, err = r.pg.Query(ctx, `
SELECT m.id, m.conversation_id, m.sender_id, m.body, m.created_at, m.updated_at, m.deleted_at
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
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// CreateMessage inserts a message and returns it.
func (r *Repo) CreateMessage(ctx context.Context, conversationID, senderID uuid.UUID, body string) (*Message, error) {
	var m Message
	err := r.pg.QueryRow(ctx, `
INSERT INTO chat_messages (conversation_id, sender_id, body)
SELECT $1, $2, $3
FROM chat_conversations
WHERE id = $1 AND (user_a_id = $2 OR user_b_id = $2)
RETURNING id, conversation_id, sender_id, body, created_at, updated_at, deleted_at
`, conversationID, senderID, body).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateMessage updates body if message belongs to sender.
func (r *Repo) UpdateMessage(ctx context.Context, messageID, senderID uuid.UUID, body string) (*Message, error) {
	var m Message
	err := r.pg.QueryRow(ctx, `
UPDATE chat_messages SET body = $3, updated_at = now()
WHERE id = $1 AND sender_id = $2 AND deleted_at IS NULL
RETURNING id, conversation_id, sender_id, body, created_at, updated_at, deleted_at
`, messageID, senderID, body).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
	if err != nil {
		return nil, err
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
	err := r.pg.QueryRow(ctx, `
SELECT m.id, m.conversation_id, m.sender_id, m.body, m.created_at, m.updated_at, m.deleted_at
FROM chat_messages m
JOIN chat_conversations c ON c.id = m.conversation_id
WHERE m.id = $1 AND (c.user_a_id = $2 OR c.user_b_id = $2)
`, messageID, userID).Scan(&m.ID, &m.ConversationID, &m.SenderID, &m.Body, &m.CreatedAt, &m.UpdatedAt, &m.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &m, nil
}
