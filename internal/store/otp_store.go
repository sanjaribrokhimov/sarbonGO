package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrOTPCooldown     = errors.New("otp cooldown")
	ErrOTPExpired      = errors.New("otp expired")
	ErrOTPInvalid      = errors.New("otp invalid")
	ErrOTPMaxAttempts  = errors.New("otp max attempts exceeded")
	ErrOTPRateLimited  = errors.New("otp rate limited")
)

type OTPStore struct {
	rdb *redis.Client
	// secret is used only to hash OTP in Redis (not a JWT secret).
	secret string

	ttl            time.Duration
	cooldown       time.Duration
	maxAttempts    int
}

func NewOTPStore(rdb *redis.Client, secret string, ttl, cooldown time.Duration, maxAttempts int) *OTPStore {
	return &OTPStore{
		rdb:         rdb,
		secret:      secret,
		ttl:         ttl,
		cooldown:    cooldown,
		maxAttempts: maxAttempts,
	}
}

func (s *OTPStore) otpKey(phone string) string      { return "otp:" + phone }
func (s *OTPStore) cooldownKey(phone string) string { return "otp:cooldown:" + phone }
func (s *OTPStore) sendCountKey(phone string) string { return "otp:send_count:" + phone }
func (s *OTPStore) sendCountIPKey(ip string) string  { return "otp:send_count_ip:" + ip }

func (s *OTPStore) hash(phone, code string) string {
	sum := sha256.Sum256([]byte(phone + ":" + code + ":" + s.secret))
	return hex.EncodeToString(sum[:])
}

type OTPRecord struct {
	RequestID string
}

// SaveOTP stores OTP hash + request_id with TTL and starts resend cooldown.
func (s *OTPStore) SaveOTP(ctx context.Context, phone, code, requestID, ip string) error {
	// cooldown check
	if ok, _ := s.rdb.Exists(ctx, s.cooldownKey(phone)).Result(); ok > 0 {
		return ErrOTPCooldown
	}

	// basic rate limit: per phone and per ip, 10/hour
	if err := s.incrWithLimit(ctx, s.sendCountKey(phone), 10, time.Hour); err != nil {
		return err
	}
	if ip != "" {
		if err := s.incrWithLimit(ctx, s.sendCountIPKey(ip), 30, time.Hour); err != nil {
			return err
		}
	}

	key := s.otpKey(phone)
	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key,
		"hash", s.hash(phone, code),
		"attempts", "0",
		"request_id", requestID,
	)
	pipe.Expire(ctx, key, s.ttl)
	pipe.Set(ctx, s.cooldownKey(phone), "1", s.cooldown)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *OTPStore) Verify(ctx context.Context, phone, code string) (OTPRecord, error) {
	key := s.otpKey(phone)
	vals, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return OTPRecord{}, err
	}
	if len(vals) == 0 {
		return OTPRecord{}, ErrOTPExpired
	}

	attempts, _ := strconv.Atoi(vals["attempts"])
	if attempts >= s.maxAttempts {
		return OTPRecord{}, ErrOTPMaxAttempts
	}

	want := vals["hash"]
	got := s.hash(phone, code)
	if want == "" || got != want {
		attempts++
		_ = s.rdb.HSet(ctx, key, "attempts", fmt.Sprintf("%d", attempts)).Err()
		if attempts >= s.maxAttempts {
			return OTPRecord{}, ErrOTPMaxAttempts
		}
		return OTPRecord{}, ErrOTPInvalid
	}

	// success: delete OTP record
	_ = s.rdb.Del(ctx, key).Err()

	return OTPRecord{RequestID: vals["request_id"]}, nil
}

func (s *OTPStore) incrWithLimit(ctx context.Context, key string, limit int64, window time.Duration) error {
	n, err := s.rdb.Incr(ctx, key).Result()
	if err != nil {
		return err
	}
	if n == 1 {
		_ = s.rdb.Expire(ctx, key, window).Err()
	}
	if n > limit {
		return ErrOTPRateLimited
	}
	return nil
}

