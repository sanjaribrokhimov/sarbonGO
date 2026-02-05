// Middleware: проверка X-Client-Token — отдельный токен для frontend и для mobile (доступ к API только доверенным приложениям).
package middleware

import (
	"crypto/subtle"

	"github.com/gin-gonic/gin"
	"github.com/sarbonGO/backend/internal/response"
)

const HeaderXClientToken = "X-Client-Token"

// ClientTokenMiddleware проверяет заголовок X-Client-Token: для X-Client-Type frontend — один токен, для mobile — другой; иначе 403.
// Если X-Client-Type не передан, используется значение из контекста (для /auth по умолчанию mobile).
func ClientTokenMiddleware(frontendToken, mobileToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientType := c.GetHeader(HeaderXClientType)
		if clientType == "" {
			if v, ok := c.Get(string(ContextKeyClientType)); ok {
				clientType = v.(string)
			}
		}
		raw := c.GetHeader(HeaderXClientToken)
		if raw == "" {
			response.AbortWithError(c, 403, "missing X-Client-Token header")
			return
		}
		var expected string
		switch clientType {
		case "frontend":
			expected = frontendToken
		case "mobile":
			expected = mobileToken
		default:
			response.AbortWithError(c, 403, "invalid or missing X-Client-Type")
			return
		}
		if subtle.ConstantTimeCompare([]byte(raw), []byte(expected)) != 1 {
			response.AbortWithError(c, 403, "invalid X-Client-Token")
			return
		}
		c.Next()
	}
}
