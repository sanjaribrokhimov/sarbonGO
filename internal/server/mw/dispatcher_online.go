package mw

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sarbonNew/internal/dispatchers"
)

// UpdateDispatcherLastOnline обновляет last_online_at диспетчера при каждом запросе к защищённым эндпоинтам.
// Ставится после RequireDispatcher.
func UpdateDispatcherLastOnline(repo *dispatchers.Repo) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		raw, ok := c.Get(CtxDispatcherID)
		if !ok || raw == nil {
			return
		}
		dispatcherID, ok := raw.(uuid.UUID)
		if !ok || dispatcherID == uuid.Nil {
			return
		}
		_ = repo.UpdateLastOnlineAt(c.Request.Context(), dispatcherID, time.Now().UTC())
	}
}
