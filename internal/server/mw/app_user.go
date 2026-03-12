package mw

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
)

const CtxAppUserID = "app_user_id"
const CtxAppUserCompanyID = "app_user_company_id"
const CtxAppUserRole = "app_user_role"

// RequireAppUser requires JWT with role "user" (company app users). Проверяет сессию (после refresh старый access недействителен).
func RequireAppUser(jwtm *security.JWTManager, refreshStore *store.RefreshStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader(HeaderUserToken))
		if raw == "" {
			resp.ErrorLang(c, 401, "missing_user_token")
			c.Abort()
			return
		}
		userID, role, companyID, sid, err := jwtm.ParseAccessWithSID(raw)
		if err != nil || userID == uuid.Nil || role != "user" {
			resp.ErrorLang(c, 401, "invalid_user_token")
			c.Abort()
			return
		}
		if sid != "" && refreshStore != nil && !refreshStore.SessionValid(c.Request.Context(), sid) {
			resp.ErrorLang(c, 401, "invalid_user_token")
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
