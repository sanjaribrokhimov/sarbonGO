package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/appusers"
	"sarbonNew/internal/approles"
	"sarbonNew/internal/companies"
	"sarbonNew/internal/companytz"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/security"
)

const inviteExpiresIn = 7 * 24 * time.Hour

type CompanyTZHandler struct {
	logger      *zap.Logger
	appusers    *appusers.Repo
	companies   *companies.Repo
	roles       *approles.Repo
	ucr         *companytz.RepoUCR
	invitations *companytz.RepoInvitations
	audit       *companytz.RepoAudit
	jwtm        *security.JWTManager
}

func NewCompanyTZHandler(
	logger *zap.Logger,
	appusers *appusers.Repo,
	companies *companies.Repo,
	roles *approles.Repo,
	ucr *companytz.RepoUCR,
	invitations *companytz.RepoInvitations,
	audit *companytz.RepoAudit,
	jwtm *security.JWTManager,
) *CompanyTZHandler {
	return &CompanyTZHandler{
		logger:      logger,
		appusers:    appusers,
		companies:   companies,
		roles:       roles,
		ucr:         ucr,
		invitations: invitations,
		audit:       audit,
		jwtm:        jwtm,
	}
}

func (h *CompanyTZHandler) appUserID(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get(mw.CtxAppUserID)
	if !ok {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok && id != uuid.Nil
}

func (h *CompanyTZHandler) getCompanyRole(ctx context.Context, userID, companyID uuid.UUID) (roleName string, ok bool) {
	roleID, err := h.ucr.GetRole(ctx, userID, companyID)
	if err == nil {
		ro, _ := h.roles.FindByID(ctx, roleID)
		if ro != nil {
			return ro.Name, true
		}
	}
	comp, _ := h.companies.GetByIDTZ(ctx, companyID)
	if comp != nil && comp.OwnerID != nil && *comp.OwnerID == userID {
		return "Owner", true
	}
	return "", false
}

// CreateCompany POST /companies (TZ 3.2)
func (h *CompanyTZHandler) CreateCompany(c *gin.Context) {
	userID, ok := h.appUserID(c)
	if !ok {
		resp.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Name    string   `json:"name" binding:"required"`
		Type    string   `json:"type" binding:"required,oneof=SHIPPER CARRIER BROKER"`
		Inn     *string  `json:"inn"`
		Phone   *string  `json:"phone"`
		Address *string  `json:"address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	ownerRole, _ := h.roles.FindByName(c.Request.Context(), "Owner")
	if ownerRole == nil {
		resp.Error(c, http.StatusInternalServerError, "roles not configured")
		return
	}
	// В БД храним Shipper, Carrier, Broker (PascalCase)
	companyTypeDB := map[string]string{"SHIPPER": "Shipper", "CARRIER": "Carrier", "BROKER": "Broker"}[strings.ToUpper(req.Type)]
	if companyTypeDB == "" {
		companyTypeDB = req.Type
	}
	companyID, err := h.companies.CreateByOwner(c.Request.Context(), companies.CreateByOwnerParams{
		Name:    req.Name,
		Type:    companyTypeDB,
		OwnerID: userID,
		Inn:     req.Inn,
		Phone:   req.Phone,
		Address: req.Address,
	})
	if err != nil {
		h.logger.Error("company create failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "company create failed")
		return
	}
	if err := h.ucr.Add(c.Request.Context(), userID, companyID, uuid.MustParse(ownerRole.ID), userID); err != nil {
		h.logger.Error("ucr add owner failed", zap.Error(err))
	}
	_ = h.audit.Log(c.Request.Context(), &userID, &companyID, "create", "company", companyID, nil, map[string]interface{}{"name": req.Name, "type": req.Type})
	comp, _ := h.companies.GetByIDTZ(c.Request.Context(), companyID)
	out := gin.H{"id": companyID, "name": req.Name, "type": req.Type, "inn": req.Inn, "phone": req.Phone, "address": req.Address, "owner_id": userID}
	if comp != nil {
		out["created_at"] = comp.CreatedAt
	}
	resp.Success(c, http.StatusCreated, "created", out)
}

// ListMyCompanies GET /auth/companies (TZ 3.4)
func (h *CompanyTZHandler) ListMyCompanies(c *gin.Context) {
	userID, ok := h.appUserID(c)
	if !ok {
		resp.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var currentCompanyID uuid.UUID
	if v, ok := c.Get(mw.CtxAppUserCompanyID); ok && v != nil {
		currentCompanyID, _ = v.(uuid.UUID)
	}
	list, err := h.companies.ListForUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("list companies failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "list failed")
		return
	}
	out := make([]gin.H, 0, len(list))
	for _, co := range list {
		isCurrent := currentCompanyID != uuid.Nil && co.ID == currentCompanyID
		out = append(out, gin.H{"id": co.ID, "name": co.Name, "type": co.Type, "role": co.Role, "is_current": isCurrent})
	}
	resp.OK(c, gin.H{"companies": out})
}

// SwitchCompany POST /auth/switch-company (TZ 3.4)
func (h *CompanyTZHandler) SwitchCompany(c *gin.Context) {
	userID, ok := h.appUserID(c)
	if !ok {
		resp.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		CompanyID string `json:"company_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	companyID, err := uuid.Parse(req.CompanyID)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid company_id")
		return
	}
	roleName, ok := h.getCompanyRole(c.Request.Context(), userID, companyID)
	if !ok {
		resp.Error(c, http.StatusForbidden, "not a member of this company")
		return
	}
	comp, err := h.companies.GetByIDTZ(c.Request.Context(), companyID)
	if err != nil || comp == nil {
		resp.Error(c, http.StatusNotFound, "company not found")
		return
	}
	tokens, _, err := h.jwtm.IssueWithCompany("user", userID, companyID)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "token issue failed")
		return
	}
	resp.OK(c, gin.H{
		"token":   tokens.AccessToken,
		"company": gin.H{"id": comp.ID, "name": comp.Name, "type": comp.Type, "role": roleName},
	})
}

// CreateInvitation POST /companies/:id/invitations (TZ 3.3)
func (h *CompanyTZHandler) CreateInvitation(c *gin.Context) {
	userID, ok := h.appUserID(c)
	if !ok {
		resp.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	companyID, err := uuid.Parse(c.Param("companyId"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid company id")
		return
	}
	actorRole, ok := h.getCompanyRole(c.Request.Context(), userID, companyID)
	if !ok {
		resp.Error(c, http.StatusForbidden, "not a member of this company")
		return
	}
	var req struct {
		Email   string `json:"email" binding:"required,email"`
		RoleID  string `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid role_id")
		return
	}
	role, err := h.roles.FindByID(c.Request.Context(), roleID)
	if err != nil || role == nil {
		resp.Error(c, http.StatusBadRequest, "role not found")
		return
	}
	if !companytz.CanInvite(actorRole, role.Name) {
		resp.Error(c, http.StatusForbidden, "your role cannot invite this role")
		return
	}
	inv, err := h.invitations.Create(c.Request.Context(), companyID, roleID, userID, req.Email, inviteExpiresIn)
	if err != nil {
		h.logger.Error("invitation create failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "invitation failed")
		return
	}
	baseURL := "https://sarbon.me"
	if c.Request != nil && c.Request.Host != "" {
		baseURL = "https://" + c.Request.Host
	}
	inviteLink := baseURL + "/accept-invite?token=" + inv.Token
	_ = h.audit.Log(c.Request.Context(), &userID, &companyID, "create", "invitation", inv.ID, nil, map[string]interface{}{"email": req.Email, "role_id": roleID})
	resp.Success(c, http.StatusCreated, "created", gin.H{
		"invite_link": inviteLink,
		"expires_at":  inv.ExpiresAt.Format(time.RFC3339),
	})
}

// AcceptInvitation POST /invitations/accept — требуется X-User-Token (пользователь компании). Добавляет текущего пользователя в компанию с ролью из приглашения.
func (h *CompanyTZHandler) AcceptInvitation(c *gin.Context) {
	userID, ok := h.appUserID(c)
	if !ok {
		resp.Error(c, http.StatusUnauthorized, "unauthorized: login or register first")
		return
	}
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	inv, err := h.invitations.GetByToken(c.Request.Context(), req.Token)
	if err != nil {
		if err == companytz.ErrInvitationNotFound || err == companytz.ErrInvitationExpired {
			resp.Error(c, http.StatusUnauthorized, "invitation not found or expired")
			return
		}
		resp.Error(c, http.StatusInternalServerError, "failed")
		return
	}
	comp, _ := h.companies.GetByIDTZ(c.Request.Context(), inv.CompanyID)
	role, _ := h.roles.FindByID(c.Request.Context(), inv.RoleID)
	out := gin.H{
		"status":       "accepted",
		"company_id":   inv.CompanyID,
		"company_name": "",
		"role":         "",
	}
	if comp != nil {
		out["company_name"] = comp.Name
	}
	if role != nil {
		out["role"] = role.Name
	}
	if err := h.ucr.Add(c.Request.Context(), userID, inv.CompanyID, inv.RoleID, inv.InvitedBy); err != nil {
		h.logger.Error("ucr add on accept failed", zap.Error(err))
	} else {
		_ = h.audit.Log(c.Request.Context(), &userID, &inv.CompanyID, "create", "user_company_role", userID, nil, map[string]interface{}{"role_id": inv.RoleID})
	}
	_ = h.invitations.Delete(c.Request.Context(), inv.ID)
	resp.OK(c, out)
}

// ListCompanyUsers GET /companies/:id/users (TZ 3.3)
func (h *CompanyTZHandler) ListCompanyUsers(c *gin.Context) {
	userID, ok := h.appUserID(c)
	if !ok {
		resp.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	companyID, err := uuid.Parse(c.Param("companyId"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid company id")
		return
	}
	_, ok = h.getCompanyRole(c.Request.Context(), userID, companyID)
	if !ok {
		resp.Error(c, http.StatusForbidden, "not a member of this company")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * limit
	list, err := h.ucr.ListUsersByCompany(c.Request.Context(), companyID, limit, offset)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "list failed")
		return
	}
	users := make([]gin.H, 0, len(list))
	for _, u := range list {
		users = append(users, gin.H{
			"id": u.UserID, "phone": u.Phone, "first_name": u.FirstName, "last_name": u.LastName,
			"role": gin.H{"id": u.RoleID, "name": u.RoleName, "description": u.RoleDescription},
			"assigned_by": gin.H{"id": u.AssignedBy, "name": u.AssignedByName},
			"assigned_at": u.AssignedAt,
		})
	}
	resp.OK(c, gin.H{"users": users, "total": len(list), "page": page, "limit": limit})
}

// UpdateUserRole PUT /companies/:id/users/:userId/role (TZ 3.3)
func (h *CompanyTZHandler) UpdateUserRole(c *gin.Context) {
	userID, ok := h.appUserID(c)
	if !ok {
		resp.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	companyID, err := uuid.Parse(c.Param("companyId"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid company id")
		return
	}
	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}
	actorRole, ok := h.getCompanyRole(c.Request.Context(), userID, companyID)
	if !ok {
		resp.Error(c, http.StatusForbidden, "not a member of this company")
		return
	}
	var req struct {
		RoleID string `json:"role_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	newRoleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid role_id")
		return
	}
	newRole, _ := h.roles.FindByID(c.Request.Context(), newRoleID)
	if newRole == nil {
		resp.Error(c, http.StatusBadRequest, "role not found")
		return
	}
	if !companytz.CanChangeRole(actorRole, newRole.Name) {
		resp.Error(c, http.StatusForbidden, "your role cannot assign this role")
		return
	}
	oldRoleID, err := h.ucr.GetRole(c.Request.Context(), targetUserID, companyID)
	if err != nil {
		resp.Error(c, http.StatusNotFound, "user not in company")
		return
	}
	oldRole, _ := h.roles.FindByID(c.Request.Context(), oldRoleID)
	if err := h.ucr.UpdateRole(c.Request.Context(), targetUserID, companyID, newRoleID, userID); err != nil {
		resp.Error(c, http.StatusInternalServerError, "update failed")
		return
	}
	oldName := ""
	if oldRole != nil {
		oldName = oldRole.Name
	}
	_ = h.audit.Log(c.Request.Context(), &userID, &companyID, "update", "user_company_role", targetUserID, map[string]interface{}{"role": oldName}, map[string]interface{}{"role": newRole.Name})
	resp.OK(c, gin.H{"user_id": targetUserID, "old_role": oldName, "new_role": newRole.Name})
}

// RemoveUser DELETE /companies/:id/users/:userId (TZ 3.3)
func (h *CompanyTZHandler) RemoveUser(c *gin.Context) {
	userID, ok := h.appUserID(c)
	if !ok {
		resp.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	companyID, err := uuid.Parse(c.Param("companyId"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid company id")
		return
	}
	targetUserID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid user id")
		return
	}
	actorRole, ok := h.getCompanyRole(c.Request.Context(), userID, companyID)
	if !ok {
		resp.Error(c, http.StatusForbidden, "not a member of this company")
		return
	}
	targetRoleID, err := h.ucr.GetRole(c.Request.Context(), targetUserID, companyID)
	if err != nil {
		resp.Error(c, http.StatusNotFound, "user not in company")
		return
	}
	targetRole, _ := h.roles.FindByID(c.Request.Context(), targetRoleID)
	targetRoleName := ""
	if targetRole != nil {
		targetRoleName = targetRole.Name
	}
	if !companytz.CanRemove(actorRole, targetRoleName) {
		resp.Error(c, http.StatusForbidden, "your role cannot remove this user")
		return
	}
	if err := h.ucr.Remove(c.Request.Context(), targetUserID, companyID); err != nil {
		resp.Error(c, http.StatusInternalServerError, "remove failed")
		return
	}
	_ = h.audit.Log(c.Request.Context(), &userID, &companyID, "delete", "user_company_role", targetUserID, map[string]interface{}{"role": targetRoleName}, nil)
	c.Status(http.StatusNoContent)
}
