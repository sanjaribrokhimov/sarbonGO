// Middleware: распределённый лимит запросов через Redis (по IP клиента).
package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sarbonGO/backend/internal/response"
)

const (
	rateLimitKeyPrefix = "ratelimit:"
	rateLimitWindow    = time.Second
)

// RateLimitMiddleware ограничивает число запросов в секунду на клиента (Redis); при превышении — 429.
func RateLimitMiddleware(rdb *redis.Client, limitPerSec int) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := rateLimitKeyPrefix + c.ClientIP()
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			response.AbortWithError(c, http.StatusServiceUnavailable, "service unavailable")
			return
		}
		if count == 1 {
			rdb.Expire(ctx, key, rateLimitWindow)
		}
		ttl, _ := rdb.TTL(ctx, key).Result()
		if ttl < 0 {
			rdb.Expire(ctx, key, rateLimitWindow)
		}

		if count > int64(limitPerSec) {
			c.Header("Retry-After", "1")
			response.AbortWithError(c, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limitPerSec))
		c.Next()
	}
}
