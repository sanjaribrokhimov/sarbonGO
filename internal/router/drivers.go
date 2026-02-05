// Drivers Profile: все маршруты под JWT (текущий водитель). Client token + Bearer.
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sarbonGO/backend/internal/handlers"
	"github.com/sarbonGO/backend/internal/middleware"
)

// RegisterDriverProfile регистрирует API профиля водителя. Все маршруты требуют Authorization: Bearer (JWT).
func RegisterDriverProfile(drivers *gin.RouterGroup, pool *pgxpool.Pool, storagePath string, auth middleware.TokenValidator) {
	if pool == nil || auth == nil {
		return
	}
	drivers.Use(middleware.AuthMiddleware(auth))
	drivers.GET("/profile", handlers.GetDriverProfile(pool))
	drivers.PUT("/profile", handlers.UpdateDriverProfile(pool, storagePath))
	drivers.PATCH("/profile", handlers.UpdateDriverProfile(pool, storagePath))
	drivers.DELETE("/profile", handlers.DeleteDriverProfile(pool))
	drivers.GET("/profile/last-activate", handlers.GetDriverProfileLastActivate(pool))
	drivers.PATCH("/profile/last-activate", handlers.PatchDriverProfileLastActivate(pool))
	drivers.GET("/profile/files/car-photo", handlers.GetDriverProfileCarPhoto(pool, storagePath))
	drivers.GET("/profile/files/adr-document", handlers.GetDriverProfileAdrDocument(pool, storagePath))
}
