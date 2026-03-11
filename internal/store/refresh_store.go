package store

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrRefreshInvalid = errors.New("refresh invalid")

type RefreshStore struct {
	rdb        *redis.Client
	ttl        time.Duration
	accessTTL  time.Duration // TTL для сессии access-токена (инвалидация после refresh)
}

func NewRefreshStore(rdb *redis.Client, refreshTTL, accessTTL time.Duration) *RefreshStore {
	return &RefreshStore{rdb: rdb, ttl: refreshTTL, accessTTL: accessTTL}
}

func (s *RefreshStore) key(driverID, jti string) string {
	return "refresh:" + driverID + ":" + jti
}

func (s *RefreshStore) sessionKey(sid string) string {
	return "session:" + sid
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

// RevokeAll удаляет все refresh-сессии пользователя (по user_id). Используется при logout по access_token.
func (s *RefreshStore) RevokeAll(ctx context.Context, userID string) error {
	pattern := "refresh:" + userID + ":*"
	keys, err := s.rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	if err := s.rdb.Del(ctx, keys...).Err(); err != nil {
		return err
	}
	// Инвалидировать все access-сессии этого пользователя (session:sid)
	userSessionsKey := "user_sessions:" + userID
	sids, err := s.rdb.SMembers(ctx, userSessionsKey).Result()
	if err != nil {
		return nil
	}
	for _, sid := range sids {
		_ = s.rdb.Del(ctx, s.sessionKey(sid)).Err()
	}
	_ = s.rdb.Del(ctx, userSessionsKey).Err()
	return nil
}

// PutSession сохраняет сессию access-токена (sid = JTI refresh). После refresh старый sid инвалидируется.
func (s *RefreshStore) PutSession(ctx context.Context, userID, sid string) error {
	k := s.sessionKey(sid)
	if err := s.rdb.Set(ctx, k, userID, s.accessTTL).Err(); err != nil {
		return err
	}
	userKey := "user_sessions:" + userID
	if err := s.rdb.SAdd(ctx, userKey, sid).Err(); err != nil {
		return err
	}
	return s.rdb.Expire(ctx, userKey, s.ttl).Err()
}

// InvalidateSession удаляет сессию (вызывается при refresh — старый access перестаёт работать).
func (s *RefreshStore) InvalidateSession(ctx context.Context, sid string) error {
	return s.rdb.Del(ctx, s.sessionKey(sid)).Err()
}

// SessionValid проверяет, что сессия access-токена ещё действительна (не была отозвана при refresh).
func (s *RefreshStore) SessionValid(ctx context.Context, sid string) bool {
	if sid == "" {
		return false
	}
	n, err := s.rdb.Exists(ctx, s.sessionKey(sid)).Result()
	return err == nil && n > 0
}

