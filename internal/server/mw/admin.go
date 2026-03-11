package mw

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
)

const CtxAdminID = "admin_id"

// RequireAdmin проверяет X-User-Token (JWT) и допускает только роль admin.
func RequireAdmin(jwtm *security.JWTManager, refreshStore *store.RefreshStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader(HeaderUserToken))
		if raw == "" {
			resp.Error(c, 401, "missing X-User-Token")
			c.Abort()
			return
		}
		id, role, _, sid, err := jwtm.ParseAccessWithSID(raw)
		if err != nil || id == uuid.Nil || role != "admin" {
			resp.Error(c, 401, "invalid X-User-Token")
			c.Abort()
			return
		}
		if sid != "" && refreshStore != nil && !refreshStore.SessionValid(c.Request.Context(), sid) {
			resp.Error(c, 401, "invalid X-User-Token")
			c.Abort()
			return
		}
		c.Set(CtxAdminID, id)
		c.Next()
	}
}

