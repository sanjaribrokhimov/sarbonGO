// Reference router: reference (spravochniki) APIs — user-categories and future reference endpoints.
package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sarbonGO/backend/internal/handlers"
)

// RegisterReference mounts reference routes on the given group (e.g. /api/v1/reference).
// All reference APIs require X-Client-Token only (no user JWT).
func RegisterReference(ref *gin.RouterGroup, pool *pgxpool.Pool) {
	if pool == nil {
		return
	}
	ref.GET("/user-categories", handlers.UserCategories(pool))
}
