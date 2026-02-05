// Middleware: OWASP-рекомендованные заголовки безопасности в каждый ответ.
package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware добавляет заголовки безопасности (X-Content-Type-Options, X-Frame-Options и т.д.) в каждый ответ.
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		// Для /swagger разрешаем CDN (Swagger UI с unpkg), иначе страница пустая из‑за блокировки скриптов/стилей.
		if strings.HasPrefix(c.Request.URL.Path, "/swagger") {
			c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' https://unpkg.com; style-src 'self' 'unsafe-inline' https://unpkg.com; img-src 'self' data: https:; font-src 'self' https://unpkg.com; connect-src 'self'")
		} else {
			c.Header("Content-Security-Policy", "default-src 'none'")
		}
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		c.Next()
	}
}
