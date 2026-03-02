package mw

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
)

const CtxAppUserID = "app_user_id"
const CtxAppUserCompanyID = "app_user_company_id"
const CtxAppUserRole = "app_user_role"

// RequireAppUser requires JWT with role "user" (app users from auth/register). Sets CtxAppUserID, CtxAppUserCompanyID (optional), CtxAppUserRole.
func RequireAppUser(jwtm *security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader(HeaderUserToken))
		if raw == "" {
			resp.Error(c, http.StatusUnauthorized, "missing X-User-Token")
			c.Abort()
			return
		}
		userID, role, companyID, err := jwtm.ParseAccessWithCompany(raw)
		if err != nil || userID == uuid.Nil || role != "user" {
			resp.Error(c, http.StatusUnauthorized, "invalid X-User-Token")
			c.Abort()
			return
		}
		c.Set(CtxAppUserID, userID)
		c.Set(CtxAppUserRole, role)
		if companyID != uuid.Nil {
			c.Set(CtxAppUserCompanyID, companyID)
		}
		c.Next()
	}
}
