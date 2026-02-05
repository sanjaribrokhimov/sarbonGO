package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const accessTokenExpire = 30 * 24 * time.Hour
const refreshTokenExpire = 7 * 24 * time.Hour

// TokenPair — пара access (JWT) и refresh (случайная строка в БД).
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

// RevokeDriverTokens удаляет все токены водителя (ротация при повторном логине).
func RevokeDriverTokens(ctx context.Context, pool *pgxpool.Pool, driverID string) error {
	_, err := pool.Exec(ctx, `DELETE FROM auth_tokens WHERE driver_id = $1`, driverID)
	return err
}

// CreateTokenPair создаёт JWT и refresh_token, сохраняет в auth_tokens по driver_id, возвращает пару.
func CreateTokenPair(ctx context.Context, pool *pgxpool.Pool, driverID string, jwtSecret string) (*TokenPair, error) {
	if err := RevokeDriverTokens(ctx, pool, driverID); err != nil {
		return nil, err
	}
	accessToken, err := CreateToken(jwtSecret, driverID, accessTokenExpire)
	if err != nil {
		return nil, err
	}
	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(refreshTokenExpire)
	_, err = pool.Exec(ctx, `
		INSERT INTO auth_tokens (driver_id, access_token, refresh_token, expires_at)
		VALUES ($1, $2, $3, $4)
	`, driverID, accessToken, refreshToken, expiresAt)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessTokenExpire.Seconds()),
	}, nil
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
