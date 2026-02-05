// Middleware: обязательный заголовок X-Platform (web | ios | android).
package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sarbonGO/backend/internal/response"
)

var allowedPlatforms = map[string]bool{
	"web": true, "ios": true, "android": true,
}

const HeaderXPlatform = "X-Platform"

// PlatformMiddleware требует X-Platform: web, ios или android; иначе 403.
// Для /api/v1/auth/* при отсутствии заголовка подставляется web.
func PlatformMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader(HeaderXPlatform)
		if raw == "" {
			if strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth") {
				raw = "web"
			} else {
				response.AbortWithError(c, 403, "missing X-Platform header")
				return
			}
		}
		if !allowedPlatforms[raw] {
			response.AbortWithError(c, 403, "invalid X-Platform; allowed: web, ios, android")
			return
		}
		c.Set(string(ContextKeyPlatform), raw)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ContextKeyPlatform, raw))
		c.Next()
	}
}
