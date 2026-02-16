package store

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var ErrDispatcherSessionNotFound = errors.New("dispatcher session not found")

type DispatcherSessionStore struct {
	rdb    *redis.Client
	prefix string
	ttl    time.Duration
}

func NewDispatcherSessionStore(rdb *redis.Client, prefix string, ttl time.Duration) *DispatcherSessionStore {
	return &DispatcherSessionStore{rdb: rdb, prefix: prefix, ttl: ttl}
}

func (s *DispatcherSessionStore) key(id string) string { return s.prefix + ":" + id }

func (s *DispatcherSessionStore) Create(ctx context.Context, payload string) (string, error) {
	id := uuid.NewString()
	if err := s.rdb.Set(ctx, s.key(id), payload, s.ttl).Err(); err != nil {
		return "", err
	}
	return id, nil
}

func (s *DispatcherSessionStore) Consume(ctx context.Context, sessionID string) (string, error) {
	key := s.key(sessionID)
	val, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrDispatcherSessionNotFound
		}
		return "", err
	}
	_ = s.rdb.Del(ctx, key).Err()
	return val, nil
}
