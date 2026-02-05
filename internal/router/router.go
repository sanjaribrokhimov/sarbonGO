// Роутер: сборка Gin с recovery, security headers, Swagger и /api/v1 с полным набором middleware.
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/sarbonGO/backend/internal/config"
	"github.com/sarbonGO/backend/internal/docs"
	"github.com/sarbonGO/backend/internal/middleware"
)

// Dependencies — зависимости для роутера (клиентские токены, валидатор JWT пользователя, Redis, Pool, StoragePath, лимит RPS, Security для OTP).
type Dependencies struct {
	FrontendClientToken string
	MobileClientToken   string
	AuthValidator       middleware.TokenValidator
	RateLimitRPS        int
	Redis               *redis.Client
	Pool                *pgxpool.Pool
	StoragePath         string
	Security            config.Security
}

// New создаёт движок Gin: глобально recovery и security headers, /swagger без обязательных заголовков, /api/v1 с полной цепочкой middleware.
func New(deps Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Глобально: сначала recovery, затем security headers на каждый ответ.
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.SecurityHeadersMiddleware())

	// Swagger — без обязательных client-headers, раздача из embed (/swagger/index.html и /swagger/openapi.yaml).
	r.StaticFS("/swagger", docs.SwaggerFS)

	// API v1: для каждого запроса обязательны X-Client-Type, X-Platform, X-Client-Token, Accept-Language; JWT пользователя — только там, где явно подключён AuthMiddleware.
	v1 := r.Group("/api/v1")
	{
		v1.Use(middleware.RequestLoggerMiddleware())
		v1.Use(middleware.ClientTypeMiddleware())
		v1.Use(middleware.PlatformMiddleware())
		v1.Use(middleware.ClientTokenMiddleware(deps.FrontendClientToken, deps.MobileClientToken))
		v1.Use(middleware.LanguageMiddleware())
		if deps.Redis != nil && deps.RateLimitRPS > 0 {
			v1.Use(middleware.RateLimitMiddleware(deps.Redis, deps.RateLimitRPS))
		}
		// System (health, etc.) — client token only
		RegisterSystem(v1)
		// Reference (spravochniki) — client token only
		refGroup := v1.Group("/reference")
		RegisterReference(refGroup, deps.Pool)
		driversGroup := v1.Group("/drivers")
		RegisterDriverProfile(driversGroup, deps.Pool, deps.StoragePath, deps.AuthValidator)
		// Auth: OTP send → verify (session_id или токены) → complete-register
		authGroup := v1.Group("/auth")
		RegisterAuth(authGroup, deps.Pool, deps.Redis, deps.StoragePath, deps.Security)
		// Routes requiring user JWT — AuthMiddleware
		v1Auth := v1.Group("")
		v1Auth.Use(middleware.AuthMiddleware(deps.AuthValidator))
		{
			// v1Auth.GET("/me", handlers.Me) — пример: здесь уже нужен Authorization: Bearer <JWT>
		}
	}

	return r
}
