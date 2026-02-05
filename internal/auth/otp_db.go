package auth

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const otpDigits = 6

// GenerateOTPCode возвращает 6-значный цифровой код.
func GenerateOTPCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// CreateOTP сохраняет новый OTP для номера: инвалидирует предыдущие (used_at = now()), вставляет запись с expires_at и attempts_count = 0.
func CreateOTP(ctx context.Context, pool *pgxpool.Pool, phone string, ttlSec int) (code string, err error) {
	code, err = GenerateOTPCode()
	if err != nil {
		return "", err
	}
	if ttlSec < 60 {
		ttlSec = 180
	}
	if ttlSec > 600 {
		ttlSec = 300
	}
	expiresAt := time.Now().Add(time.Duration(ttlSec) * time.Second)
	// Пометить старые OTP по этому номеру как использованные (сброс)
	_, _ = pool.Exec(ctx, `UPDATE otp_codes SET used_at = now() WHERE phone = $1 AND used_at IS NULL`, phone)
	_, err = pool.Exec(ctx, `
		INSERT INTO otp_codes (phone, code, expires_at, attempts_count)
		VALUES ($1, $2, $3, 0)
	`, phone, code, expiresAt)
	if err != nil {
		return "", err
	}
	return code, nil
}

// ValidateAndConsumeOTP проверяет код: не истёк, не использован, attempts_count < maxAttempts; при совпадении кода помечает used_at и возвращает true.
func ValidateAndConsumeOTP(ctx context.Context, pool *pgxpool.Pool, phone, code string, maxAttempts int) (ok bool, err error) {
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	var id string
	var attempts int
	var usedAt *time.Time
	var expiresAt time.Time
	err = pool.QueryRow(ctx, `
		SELECT id, attempts_count, used_at, expires_at
		FROM otp_codes
		WHERE phone = $1 AND used_at IS NULL
		ORDER BY expires_at DESC
		LIMIT 1
	`, phone).Scan(&id, &attempts, &usedAt, &expiresAt)
	if err != nil {
		return false, nil
	}
	if time.Now().After(expiresAt) {
		return false, nil
	}
	if attempts >= maxAttempts {
		return false, nil
	}
	var storedCode string
	err = pool.QueryRow(ctx, `SELECT code FROM otp_codes WHERE id = $1`, id).Scan(&storedCode)
	if err != nil {
		return false, nil
	}
	codeNorm := trimDigits(code)
	storedNorm := trimDigits(storedCode)
	if codeNorm != storedNorm {
		_, _ = pool.Exec(ctx, `UPDATE otp_codes SET attempts_count = attempts_count + 1 WHERE id = $1`, id)
		return false, nil
	}
	// Код верный — пометить использованным
	_, err = pool.Exec(ctx, `UPDATE otp_codes SET used_at = now() WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	return true, nil
}

func trimDigits(s string) string {
	var b []byte
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b = append(b, byte(r))
		}
	}
	return string(b)
}
