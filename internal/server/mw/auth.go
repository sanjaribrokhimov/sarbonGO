package mw

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
)

const CtxDriverID = "driver_id"
const CtxDispatcherID = "dispatcher_id"

func RequireDriver(jwtm *security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader(HeaderUserToken))
		if raw == "" {
			resp.Error(c, http.StatusUnauthorized, "missing X-User-Token")
			c.Abort()
			return
		}
		id, role, err := jwtm.ParseAccess(raw)
		if err != nil || id == uuid.Nil || role != "driver" {
			resp.Error(c, http.StatusUnauthorized, "invalid X-User-Token")
			c.Abort()
			return
		}
		c.Set(CtxDriverID, id)
		c.Next()
	}
}

func RequireDispatcher(jwtm *security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader(HeaderUserToken))
		if raw == "" {
			resp.Error(c, http.StatusUnauthorized, "missing X-User-Token")
			c.Abort()
			return
		}
		id, role, err := jwtm.ParseAccess(raw)
		if err != nil || id == uuid.Nil || role != "dispatcher" {
			resp.Error(c, http.StatusUnauthorized, "invalid X-User-Token")
			c.Abort()
			return
		}
		c.Set(CtxDispatcherID, id)
		c.Next()
	}
}
