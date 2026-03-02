package chat

import (
	"context"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	presencePrefix   = "chat:presence:"
	presenceTTL      = 65 * time.Second
	typingPrefix     = "chat:typing:"
	typingTTL        = 15 * time.Second
	lastSeenPrefix   = "chat:lastseen:"
	lastSeenKeepSec  = 30 * 24 * 3600 // 30 days
)

// PresenceStore holds online/typing/last_seen in Redis.
type PresenceStore struct {
	rdb *redis.Client
}

func NewPresenceStore(rdb *redis.Client) *PresenceStore {
	return &PresenceStore{rdb: rdb}
}

// SetOnline marks user as online. Call on WS connect; TTL 65s, client should heartbeat.
func (s *PresenceStore) SetOnline(ctx context.Context, userID uuid.UUID) error {
	k := presencePrefix + userID.String()
	return s.rdb.Set(ctx, k, "1", presenceTTL).Err()
}

// SetOffline removes online and sets last_seen. Call on WS disconnect.
func (s *PresenceStore) SetOffline(ctx context.Context, userID uuid.UUID) error {
	now := time.Now().Unix()
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, presencePrefix+userID.String())
	pipe.Set(ctx, lastSeenPrefix+userID.String(), now, lastSeenKeepSec*time.Second)
	_, err := pipe.Exec(ctx)
	return err
}

// Heartbeat refreshes online TTL. Call every ~30s from connected client.
func (s *PresenceStore) Heartbeat(ctx context.Context, userID uuid.UUID) error {
	return s.rdb.Expire(ctx, presencePrefix+userID.String(), presenceTTL).Err()
}

// IsOnline returns true if user has recent presence key.
func (s *PresenceStore) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	n, err := s.rdb.Exists(ctx, presencePrefix+userID.String()).Result()
	return n > 0, err
}

// LastSeen returns last seen unix ts; 0 if never or error.
func (s *PresenceStore) LastSeen(ctx context.Context, userID uuid.UUID) (int64, error) {
	v, err := s.rdb.Get(ctx, lastSeenPrefix+userID.String()).Result()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	ts, _ := strconv.ParseInt(v, 10, 64)
	return ts, nil
}

// SetTyping sets typing indicator for user in conversation. TTL 15s.
func (s *PresenceStore) SetTyping(ctx context.Context, conversationID, userID uuid.UUID) error {
	k := typingPrefix + conversationID.String() + ":" + userID.String()
	return s.rdb.Set(ctx, k, "1", typingTTL).Err()
}

// ClearTyping removes typing for user in conversation.
func (s *PresenceStore) ClearTyping(ctx context.Context, conversationID, userID uuid.UUID) error {
	return s.rdb.Del(ctx, typingPrefix+conversationID.String()+":"+userID.String()).Err()
}

// IsTyping returns true if user has typing key in conversation.
func (s *PresenceStore) IsTyping(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	n, err := s.rdb.Exists(ctx, typingPrefix+conversationID.String()+":"+userID.String()).Result()
	return n > 0, err
}

// PresenceResponse for API.
type PresenceResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Online    bool      `json:"online"`
	Typing    bool      `json:"typing,omitempty"`
	LastSeen  int64     `json:"last_seen,omitempty"` // unix sec
}

// GetPresence returns online and last_seen for a user (typing optional, per conversation).
func (s *PresenceStore) GetPresence(ctx context.Context, userID uuid.UUID, conversationID *uuid.UUID) (PresenceResponse, error) {
	out := PresenceResponse{UserID: userID}
	online, err := s.IsOnline(ctx, userID)
	if err != nil {
		return out, err
	}
	out.Online = online
	if !online {
		ls, err := s.LastSeen(ctx, userID)
		if err != nil {
			return out, err
		}
		out.LastSeen = ls
	}
	if conversationID != nil {
		typing, _ := s.IsTyping(ctx, *conversationID, userID)
		out.Typing = typing
	}
	return out, nil
}
