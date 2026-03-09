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
const CtxDispatcherCompanyID = "dispatcher_company_id" // optional, set when JWT has company_id (after switch-company)
const CtxUserID = "user_id"   // chat: any authenticated user UUID
const CtxUserRole = "user_role" // chat: driver | dispatcher | admin

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
		id, role, companyID, err := jwtm.ParseAccessWithCompany(raw)
		if err != nil || id == uuid.Nil || role != "dispatcher" {
			resp.Error(c, http.StatusUnauthorized, "invalid X-User-Token")
			c.Abort()
			return
		}
		c.Set(CtxDispatcherID, id)
		if companyID != uuid.Nil {
			c.Set(CtxDispatcherCompanyID, companyID)
		}
		c.Next()
	}
}

// RequireChatUser sets CtxUserID and CtxUserRole. Accepts:
// - X-User-Token (JWT) or header X-User-ID (Swagger)
// - Query ?user_id=uuid or ?token=JWT (for WebSocket, where headers are not always sent)
func RequireChatUser(jwtm *security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tryUserID := func(raw string) (uuid.UUID, bool) {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				return uuid.Nil, false
			}
			id, err := uuid.Parse(raw)
			return id, err == nil && id != uuid.Nil
		}
		tryToken := func(raw string) (uuid.UUID, string, bool) {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				return uuid.Nil, "", false
			}
			id, role, err := jwtm.ParseAccess(raw)
			return id, role, err == nil && id != uuid.Nil
		}
		// 1) Query (for WS)
		if id, ok := tryUserID(c.Query("user_id")); ok {
			c.Set(CtxUserID, id)
			c.Set(CtxUserRole, "user")
			c.Next()
			return
		}
		if id, role, ok := tryToken(c.Query("token")); ok {
			c.Set(CtxUserID, id)
			c.Set(CtxUserRole, role)
			c.Next()
			return
		}
		// 2) Headers
		if id, ok := tryUserID(c.GetHeader(HeaderUserID)); ok {
			c.Set(CtxUserID, id)
			c.Set(CtxUserRole, "user")
			c.Next()
			return
		}
		raw := strings.TrimSpace(c.GetHeader(HeaderUserToken))
		if raw == "" {
			resp.Error(c, http.StatusUnauthorized, "missing X-User-Token or X-User-ID (or query user_id/token)")
			c.Abort()
			return
		}
		id, role, err := jwtm.ParseAccess(raw)
		if err != nil || id == uuid.Nil {
			resp.Error(c, http.StatusUnauthorized, "invalid X-User-Token")
			c.Abort()
			return
		}
		c.Set(CtxUserID, id)
		c.Set(CtxUserRole, role)
		c.Next()
	}
}
