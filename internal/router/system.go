// System router: health and other system endpoints (client token only).
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/sarbonGO/backend/internal/handlers"
)

// RegisterSystem mounts system routes on the given v1 group (e.g. /api/v1/health).
func RegisterSystem(v1 *gin.RouterGroup) {
	v1.GET("/health", handlers.Health)
	v1.GET("/status-codes", handlers.StatusCodes)
}
