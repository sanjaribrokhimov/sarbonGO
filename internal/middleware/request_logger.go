// Middleware: логирование каждого запроса API — метод, путь, код ответа и время выполнения.
package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLoggerMiddleware логирует каждый запрос после обработки: метод, путь, статус, длительность.
// Пример в консоли: [API] POST /api/v1/auth/complete-registration -> 200 (15ms)
func RequestLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		if path == "" {
			path = "/"
		}
		c.Next()
		status := c.Writer.Status()
		log.Printf("[API] %s %s -> %d (%v)", method, path, status, time.Since(start).Round(time.Millisecond))
	}
}
