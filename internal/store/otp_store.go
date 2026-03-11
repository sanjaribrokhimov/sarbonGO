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
	ErrOTPCooldown         = errors.New("otp cooldown")
	ErrOTPExpired          = errors.New("otp expired")
	ErrOTPInvalid          = errors.New("otp invalid")
	ErrOTPMaxAttempts      = errors.New("otp max attempts exceeded")
	ErrOTPRateLimited      = errors.New("otp rate limited")
	ErrOTPVerifyRateLimited = errors.New("otp verify attempts per phone exceeded")
)

type OTPStore struct {
	rdb                      *redis.Client
	secret                   string
	ttl                      time.Duration
	cooldown                 time.Duration
	maxAttempts              int
	sendLimitPerPhone        int64
	sendLimitPerIP           int64
	sendWindow               time.Duration
	verifyAttemptsPerPhone   int64         // лимит попыток ввода OTP на номер в окне (0 = только maxAttempts на один код)
	verifyAttemptsWindow     time.Duration // окно для verifyAttemptsPerPhone
	prefix                   string
}

func NewOTPStore(rdb *redis.Client, secret string, ttl, cooldown time.Duration, maxAttempts int, sendLimitPerPhone, sendLimitPerIP int64, sendWindow time.Duration, verifyAttemptsPerPhone int64, verifyAttemptsWindow time.Duration) *OTPStore {
	if sendWindow <= 0 {
		sendWindow = time.Hour
	}
	return &OTPStore{
		rdb:                    rdb,
		secret:                 secret,
		ttl:                    ttl,
		cooldown:               cooldown,
		maxAttempts:            maxAttempts,
		sendLimitPerPhone:      sendLimitPerPhone,
		sendLimitPerIP:         sendLimitPerIP,
		sendWindow:             sendWindow,
		verifyAttemptsPerPhone: verifyAttemptsPerPhone,
		verifyAttemptsWindow:   verifyAttemptsWindow,
		prefix:                 "",
	}
}

func NewOTPStoreWithPrefix(rdb *redis.Client, secret string, ttl, cooldown time.Duration, maxAttempts int, sendLimitPerPhone, sendLimitPerIP int64, sendWindow time.Duration, verifyAttemptsPerPhone int64, verifyAttemptsWindow time.Duration, prefix string) *OTPStore {
	if sendWindow <= 0 {
		sendWindow = time.Hour
	}
	return &OTPStore{
		rdb:                    rdb,
		secret:                 secret,
		ttl:                    ttl,
		cooldown:               cooldown,
		maxAttempts:            maxAttempts,
		sendLimitPerPhone:      sendLimitPerPhone,
		sendLimitPerIP:         sendLimitPerIP,
		sendWindow:             sendWindow,
		verifyAttemptsPerPhone: verifyAttemptsPerPhone,
		verifyAttemptsWindow:   verifyAttemptsWindow,
		prefix:                 prefix,
	}
}

func (s *OTPStore) otpKey(phone string) string           { return s.prefix + "otp:" + phone }
func (s *OTPStore) cooldownKey(phone string) string      { return s.prefix + "otp:cooldown:" + phone }
func (s *OTPStore) sendCountKey(phone string) string     { return s.prefix + "otp:send_count:" + phone }
func (s *OTPStore) sendCountIPKey(ip string) string     { return s.prefix + "otp:send_count_ip:" + ip }
func (s *OTPStore) verifyAttemptsKey(phone string) string { return s.prefix + "otp:verify_attempts:" + phone }

func (s *OTPStore) hash(phone, code string) string {
	sum := sha256.Sum256([]byte(phone + ":" + code + ":" + s.secret))
	return hex.EncodeToString(sum[:])
}

type OTPRecord struct {
	RequestID string
}

// SaveOTP stores OTP hash + request_id with TTL and starts resend cooldown.
func (s *OTPStore) SaveOTP(ctx context.Context, phone, code, requestID, ip string) error {
	// cooldown check (if enabled). When cooldown == 0, Redis SET would persist forever,
	// so we must skip cooldown entirely.
	if s.cooldown > 0 {
		if ok, _ := s.rdb.Exists(ctx, s.cooldownKey(phone)).Result(); ok > 0 {
			return ErrOTPCooldown
		}
	}

	// basic rate limit: per phone and per ip (configurable)
	if s.sendLimitPerPhone > 0 {
		if err := s.incrWithLimit(ctx, s.sendCountKey(phone), s.sendLimitPerPhone, s.sendWindow); err != nil {
			return err
		}
	}
	if ip != "" && s.sendLimitPerIP > 0 {
		if err := s.incrWithLimit(ctx, s.sendCountIPKey(ip), s.sendLimitPerIP, s.sendWindow); err != nil {
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
	if s.cooldown > 0 {
		pipe.Set(ctx, s.cooldownKey(phone), "1", s.cooldown)
	}
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
		// Неверный код: учитываем в лимите попыток на номер (защита от подбора)
		if s.verifyAttemptsPerPhone > 0 && s.verifyAttemptsWindow > 0 {
			vkey := s.verifyAttemptsKey(phone)
			n, err := s.rdb.Incr(ctx, vkey).Result()
			if err != nil {
				return OTPRecord{}, err
			}
			if n == 1 {
				_ = s.rdb.Expire(ctx, vkey, s.verifyAttemptsWindow).Err()
			}
			if n > s.verifyAttemptsPerPhone {
				return OTPRecord{}, ErrOTPVerifyRateLimited
			}
		}
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
