// Middleware: обязательный заголовок X-Client-Type (frontend | mobile).
package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sarbonGO/backend/internal/response"
)

var allowedClientTypes = map[string]bool{
	"frontend": true,
	"mobile":   true,
}

const HeaderXClientType = "X-Client-Type"

// ClientTypeMiddleware требует X-Client-Type: frontend или mobile; иначе 403.
// Для маршрутов /api/v1/auth/* при отсутствии заголовка подставляется mobile (удобно для curl и мобильных клиентов).
func ClientTypeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader(HeaderXClientType)
		if raw == "" {
			if strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth") {
				raw = "mobile"
			} else {
				response.AbortWithError(c, 403, "missing X-Client-Type header")
				return
			}
		}
		if !allowedClientTypes[raw] {
			response.AbortWithError(c, 403, "invalid X-Client-Type; allowed: frontend, mobile")
			return
		}
		c.Set(string(ContextKeyClientType), raw)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ContextKeyClientType, raw))
		c.Next()
	}
}
