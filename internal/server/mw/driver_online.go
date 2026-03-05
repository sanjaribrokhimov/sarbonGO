package mw

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sarbonNew/internal/drivers"
)

// UpdateDriverLastOnline обновляет last_online_at водителя при каждом запросе к защищённым эндпоинтам.
// Ставится после RequireDriver, чтобы в контексте был CtxDriverID.
func UpdateDriverLastOnline(repo *drivers.Repo) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		raw, ok := c.Get(CtxDriverID)
		if !ok || raw == nil {
			return
		}
		driverID, ok := raw.(uuid.UUID)
		if !ok || driverID == uuid.Nil {
			return
		}
		_ = repo.UpdateOnlineAt(c.Request.Context(), driverID, time.Now().UTC())
	}
}
