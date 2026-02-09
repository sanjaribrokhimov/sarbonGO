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
	ErrPhoneChangeSessionNotFound = errors.New("phone change session not found")
	ErrPhoneChangeOTPExpired      = errors.New("phone change otp expired")
	ErrPhoneChangeOTPInvalid      = errors.New("phone change otp invalid")
	ErrPhoneChangeMaxAttempts     = errors.New("phone change max attempts exceeded")
)

type PhoneChangeStore struct {
	rdb         *redis.Client
	secret      string
	ttl         time.Duration
	maxAttempts int
}

func NewPhoneChangeStore(rdb *redis.Client, secret string, ttl time.Duration, maxAttempts int) *PhoneChangeStore {
	return &PhoneChangeStore{rdb: rdb, secret: secret, ttl: ttl, maxAttempts: maxAttempts}
}

func (s *PhoneChangeStore) key(sessionID string) string { return "phonechange:" + sessionID }

func (s *PhoneChangeStore) hash(sessionID, otp string) string {
	sum := sha256.Sum256([]byte(sessionID + ":" + otp + ":" + s.secret))
	return hex.EncodeToString(sum[:])
}

func (s *PhoneChangeStore) Create(ctx context.Context, driverID uuid.UUID, newPhone string, otp string) (sessionID string, err error) {
	sessionID = uuid.NewString()
	key := s.key(sessionID)
	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key,
		"driver_id", driverID.String(),
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

type PhoneChangeRecord struct {
	DriverID uuid.UUID
	NewPhone string
}

func (s *PhoneChangeStore) Verify(ctx context.Context, sessionID string, otp string) (PhoneChangeRecord, error) {
	key := s.key(sessionID)
	vals, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return PhoneChangeRecord{}, err
	}
	if len(vals) == 0 {
		return PhoneChangeRecord{}, ErrPhoneChangeOTPExpired
	}

	attempts, _ := strconv.Atoi(vals["attempts"])
	if attempts >= s.maxAttempts {
		return PhoneChangeRecord{}, ErrPhoneChangeMaxAttempts
	}

	want := vals["hash"]
	got := s.hash(sessionID, otp)
	if want == "" || got != want {
		attempts++
		_ = s.rdb.HSet(ctx, key, "attempts", fmt.Sprintf("%d", attempts)).Err()
		if attempts >= s.maxAttempts {
			return PhoneChangeRecord{}, ErrPhoneChangeMaxAttempts
		}
		return PhoneChangeRecord{}, ErrPhoneChangeOTPInvalid
	}

	driverID, err := uuid.Parse(vals["driver_id"])
	if err != nil {
		return PhoneChangeRecord{}, ErrPhoneChangeSessionNotFound
	}
	newPhone := vals["new_phone"]
	if newPhone == "" {
		return PhoneChangeRecord{}, ErrPhoneChangeSessionNotFound
	}

	// consume
	_ = s.rdb.Del(ctx, key).Err()
	return PhoneChangeRecord{DriverID: driverID, NewPhone: newPhone}, nil
}

