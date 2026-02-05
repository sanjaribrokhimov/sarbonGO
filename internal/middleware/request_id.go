// Middleware: обязательный заголовок X-Request-ID в формате UUID.
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sarbonGO/backend/internal/response"
)

const HeaderXRequestID = "X-Request-ID"

// RequestIDMiddleware проверяет наличие и валидность X-Request-ID (UUID); при отсутствии или неверном формате — 403.
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader(HeaderXRequestID)
		if raw == "" {
			response.AbortWithError(c, 403, "missing X-Request-ID header")
			return
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			response.AbortWithError(c, 403, "invalid X-Request-ID: must be UUID")
			return
		}
		rid := id.String()
		c.Set(string(ContextKeyRequestID), rid)
		c.Header(HeaderXRequestID, rid)
		c.Next()
	}
}
