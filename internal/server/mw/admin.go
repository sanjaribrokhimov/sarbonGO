package mw

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
)

const CtxAdminID = "admin_id"

// RequireAdmin проверяет X-User-Token (JWT) и допускает только роль admin.
// Для всех /v1/admin/* (кроме auth/login) также уже проверены X-Client-Token, X-Device-Type, X-Language через RequireBaseHeaders на v1.
func RequireAdmin(jwtm *security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader(HeaderUserToken))
		if raw == "" {
			resp.Error(c, http.StatusUnauthorized, "missing X-User-Token")
			c.Abort()
			return
		}
		id, role, err := jwtm.ParseAccess(raw)
		if err != nil || id == uuid.Nil || role != "admin" {
			resp.Error(c, http.StatusUnauthorized, "invalid X-User-Token")
			c.Abort()
			return
		}
		c.Set(CtxAdminID, id)
		c.Next()
	}
}

