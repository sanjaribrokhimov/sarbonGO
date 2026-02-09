package store

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrRefreshInvalid = errors.New("refresh invalid")

type RefreshStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRefreshStore(rdb *redis.Client, ttl time.Duration) *RefreshStore {
	return &RefreshStore{rdb: rdb, ttl: ttl}
}

func (s *RefreshStore) key(driverID, jti string) string {
	return "refresh:" + driverID + ":" + jti
}

func (s *RefreshStore) Put(ctx context.Context, driverID, jti string) error {
	return s.rdb.Set(ctx, s.key(driverID, jti), "1", s.ttl).Err()
}

func (s *RefreshStore) Consume(ctx context.Context, driverID, jti string) error {
	key := s.key(driverID, jti)
	n, err := s.rdb.Del(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrRefreshInvalid
	}
	return nil
}

func (s *RefreshStore) RevokeAll(ctx context.Context, driverID string) error {
	// Keep it simple: use a per-driver set of JTIs.
	// Since "SCAN" is not ideal, we model a set key instead.
	// This function is intentionally left as no-op unless we enable the set-based mode.
	_ = driverID
	return nil
}

