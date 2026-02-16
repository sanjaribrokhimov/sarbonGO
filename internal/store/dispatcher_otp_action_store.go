package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrDispActionExpired     = errors.New("dispatcher action expired")
	ErrDispActionInvalidOTP  = errors.New("dispatcher action invalid otp")
	ErrDispActionMaxAttempts = errors.New("dispatcher action max attempts exceeded")
	ErrDispActionNotFound    = errors.New("dispatcher action not found")
)

type DispatcherOTPActionStore struct {
	rdb         *redis.Client
	secret      string
	prefix      string
	ttl         time.Duration
	maxAttempts int
}

func NewDispatcherOTPActionStore(rdb *redis.Client, secret, prefix string, ttl time.Duration, maxAttempts int) *DispatcherOTPActionStore {
	return &DispatcherOTPActionStore{rdb: rdb, secret: secret, prefix: prefix, ttl: ttl, maxAttempts: maxAttempts}
}

func (s *DispatcherOTPActionStore) key(sessionID string) string { return s.prefix + ":" + sessionID }

func (s *DispatcherOTPActionStore) hash(sessionID, otp string) string {
	sum := sha256.Sum256([]byte(sessionID + ":" + otp + ":" + s.secret))
	return hex.EncodeToString(sum[:])
}

type DispActionPayload struct {
	DispatcherID uuid.UUID
	Phone        string
	NewPhone     string
}

func (s *DispatcherOTPActionStore) Create(ctx context.Context, dispatcherID uuid.UUID, phone string, newPhone string, otp string) (sessionID string, err error) {
	sessionID = uuid.NewString()
	key := s.key(sessionID)
	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key,
		"dispatcher_id", dispatcherID.String(),
		"phone", phone,
		"new_phone", newPhone,
		"hash", s.hash(sessionID, otp),
		"attempts", "0",
	)
	pipe.Expire(ctx, key, s.ttl)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func (s *DispatcherOTPActionStore) Verify(ctx context.Context, sessionID string, otp string) (DispActionPayload, error) {
	key := s.key(sessionID)
	vals, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return DispActionPayload{}, err
	}
	if len(vals) == 0 {
		return DispActionPayload{}, ErrDispActionExpired
	}

	attempts, _ := strconv.Atoi(vals["attempts"])
	if attempts >= s.maxAttempts {
		return DispActionPayload{}, ErrDispActionMaxAttempts
	}

	want := vals["hash"]
	got := s.hash(sessionID, otp)
	if want == "" || got != want {
		attempts++
		_ = s.rdb.HSet(ctx, key, "attempts", fmt.Sprintf("%d", attempts)).Err()
		if attempts >= s.maxAttempts {
			return DispActionPayload{}, ErrDispActionMaxAttempts
		}
		return DispActionPayload{}, ErrDispActionInvalidOTP
	}

	did, err := uuid.Parse(vals["dispatcher_id"])
	if err != nil {
		return DispActionPayload{}, ErrDispActionNotFound
	}

	phone := vals["phone"]
	newPhone := vals["new_phone"]

	_ = s.rdb.Del(ctx, key).Err()
	return DispActionPayload{DispatcherID: did, Phone: phone, NewPhone: newPhone}, nil
}
