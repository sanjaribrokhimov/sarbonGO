package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DriverRow — водитель по номеру (для проверки после OTP).
type DriverRow struct {
	ID            string
	PhoneNumber   string
	AccountStatus string
}

// GetDriverByPhone возвращает водителя по phone_number в drivers или nil.
func GetDriverByPhone(ctx context.Context, pool *pgxpool.Pool, phone string) (*DriverRow, error) {
	var d DriverRow
	err := pool.QueryRow(ctx, `
		SELECT id::text, phone_number, COALESCE(account_status, 'pending')
		FROM drivers
		WHERE phone_number = $1 AND deleted_at IS NULL
		LIMIT 1
	`, phone).Scan(&d.ID, &d.PhoneNumber, &d.AccountStatus)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

const registrationSessionTTL = 15 * time.Minute

// CreateRegistrationSession создаёт сессию после OTP для complete-register; возвращает session_id.
func CreateRegistrationSession(ctx context.Context, pool *pgxpool.Pool, phone string) (sessionID string, err error) {
	expiresAt := time.Now().Add(registrationSessionTTL)
	err = pool.QueryRow(ctx, `
		INSERT INTO registration_sessions (phone, expires_at)
		VALUES ($1, $2)
		RETURNING id::text
	`, phone, expiresAt).Scan(&sessionID)
	return sessionID, err
}

// GetRegistrationSessionByID возвращает phone и true, если сессия валидна (не истекла). После использования сессию удаляют.
func GetRegistrationSessionByID(ctx context.Context, pool *pgxpool.Pool, sessionID string) (phone string, ok bool, err error) {
	var expiresAt time.Time
	err = pool.QueryRow(ctx, `
		SELECT phone, expires_at FROM registration_sessions WHERE id = $1
	`, sessionID).Scan(&phone, &expiresAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	if time.Now().After(expiresAt) {
		_, _ = pool.Exec(ctx, `DELETE FROM registration_sessions WHERE id = $1`, sessionID)
		return "", false, nil
	}
	return phone, true, nil
}

// ConsumeRegistrationSession удаляет сессию после успешного complete-register.
func ConsumeRegistrationSession(ctx context.Context, pool *pgxpool.Pool, sessionID string) error {
	_, err := pool.Exec(ctx, `DELETE FROM registration_sessions WHERE id = $1`, sessionID)
	return err
}
