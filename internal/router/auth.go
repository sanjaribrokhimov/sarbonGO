// Auth router: OTP send → verify (session_id или токены) → complete-register по session_id.
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/sarbonGO/backend/internal/config"
	"github.com/sarbonGO/backend/internal/handlers"
)

// RegisterAuth регистрирует POST /auth/otp/send, POST /auth/otp/verify, POST /auth/complete-register.
func RegisterAuth(v1 *gin.RouterGroup, pool *pgxpool.Pool, rdb *redis.Client, storagePath string, security config.Security) {
	if pool == nil {
		return
	}
	v1.POST("/otp/send", handlers.SendOTPV2(pool, rdb, security))
	v1.POST("/otp/verify", handlers.VerifyOTPV2(pool, security))
	v1.POST("/complete-register", handlers.CompleteRegister(pool, storagePath, security))
}
