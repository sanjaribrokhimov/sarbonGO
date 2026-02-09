package store

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/google/uuid"
)

var ErrSessionNotFound = errors.New("session not found")

type SessionStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewSessionStore(rdb *redis.Client, ttl time.Duration) *SessionStore {
	return &SessionStore{rdb: rdb, ttl: ttl}
}

func (s *SessionStore) key(sessionID string) string { return "regsession:" + sessionID }

func (s *SessionStore) Create(ctx context.Context, phone string) (string, error) {
	id := uuid.NewString()
	if err := s.rdb.Set(ctx, s.key(id), phone, s.ttl).Err(); err != nil {
		return "", err
	}
	return id, nil
}

func (s *SessionStore) Consume(ctx context.Context, sessionID string) (phone string, err error) {
	key := s.key(sessionID)
	phone, err = s.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrSessionNotFound
		}
		return "", err
	}
	_ = s.rdb.Del(ctx, key).Err()
	return phone, nil
}

