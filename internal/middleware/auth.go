// Middleware: обязательный заголовок Authorization: Bearer <token> и валидация токена.
package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sarbonGO/backend/internal/response"
)

const HeaderAuthorization = "Authorization"
const BearerPrefix = "Bearer "

// TokenValidator — интерфейс проверки Bearer-токена (например JWT). В production — реализация с проверкой подписи.
type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (userID string, err error)
}

// AuthMiddleware требует Authorization: Bearer <token> и при наличии validator проверяет токен; иначе 403/401.
func AuthMiddleware(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader(HeaderAuthorization)
		if raw == "" {
			response.AbortWithError(c, 403, "missing Authorization header")
			return
		}
		if !strings.HasPrefix(raw, BearerPrefix) {
			response.AbortWithError(c, 403, "invalid Authorization; expected Bearer <token>")
			return
		}
		token := strings.TrimPrefix(raw, BearerPrefix)
		if token == "" {
			response.AbortWithError(c, 403, "missing Bearer token")
			return
		}
		if validator != nil {
			userID, err := validator.ValidateToken(c.Request.Context(), token)
			if err != nil {
				response.AbortWithError(c, 401, "invalid or expired token")
				return
			}
			c.Set(string(ContextKeyUserID), userID)
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ContextKeyUserID, userID))
		}
		c.Next()
	}
}
