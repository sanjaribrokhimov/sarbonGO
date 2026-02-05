// Валидация Bearer-токена (заглушка; в production — JWT с проверкой подписи).
package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/sarbonGO/backend/internal/config"
)

// StubValidator — заглушка проверки токена; в production заменить на реализацию с JWT.
type StubValidator struct {
	Secret string
}

// NewStubValidator создаёт заглушку валидатора (секрет из конфига для будущей проверки JWT).
func NewStubValidator(cfg config.Security) *StubValidator {
	return &StubValidator{Secret: cfg.JWTSecret}
}

// ValidateToken реализует middleware.TokenValidator: проверка JWT (HMAC-SHA256), возврат sub как userID.
func (v *StubValidator) ValidateToken(ctx context.Context, token string) (userID string, err error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrInvalidToken
	}
	if v.Secret != "" {
		return ValidateToken(v.Secret, token)
	}
	return "stub-user-id", nil
}

// ErrInvalidToken — ошибка невалидного или пустого токена.
var ErrInvalidToken = errors.New("invalid token")
