// Middleware: перехват panic, ответ 500 без утечки стека клиенту, лог с request_id.
package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sarbonGO/backend/internal/response"
)

// RecoveryMiddleware перехватывает panic, логирует с request_id и возвращает 500 без раскрытия деталей клиенту.
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] path=%s err=%v", c.Request.URL.Path, err)
				response.AbortWithError(c, http.StatusInternalServerError, "internal server error")
			}
		}()
		c.Next()
	}
}
