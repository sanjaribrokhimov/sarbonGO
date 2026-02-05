// Middleware: обязательный заголовок X-App-Version в формате x.y.z (семантическая версия).
package middleware

import (
	"context"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/sarbonGO/backend/internal/response"
)

// Регулярка для версии вида x.y.z.
var appVersionRe = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

const HeaderXAppVersion = "X-App-Version"

// AppVersionMiddleware требует X-App-Version в формате x.y.z; иначе 403.
func AppVersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader(HeaderXAppVersion)
		if raw == "" {
			response.AbortWithError(c, 403, "missing X-App-Version header")
			return
		}
		if !appVersionRe.MatchString(raw) {
			response.AbortWithError(c, 403, "invalid X-App-Version; expected x.y.z")
			return
		}
		c.Set(string(ContextKeyAppVersion), raw)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ContextKeyAppVersion, raw))
		c.Next()
	}
}
