package chat

import (
	"time"

	"github.com/google/uuid"
)

// Conversation between two users (user_a_id < user_b_id in DB).
type Conversation struct {
	ID        uuid.UUID `json:"id"`
	UserAID   uuid.UUID `json:"user_a_id"`
	UserBID   uuid.UUID `json:"user_b_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Message in a conversation.
type Message struct {
	ID             uuid.UUID  `json:"id"`
	ConversationID uuid.UUID  `json:"conversation_id"`
	SenderID       uuid.UUID  `json:"sender_id"`
	Body           string     `json:"body"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}

// PeerID returns the other participant's ID for the given user.
func (c *Conversation) PeerID(me uuid.UUID) uuid.UUID {
	if c.UserAID == me {
		return c.UserBID
	}
	return c.UserAID
}
