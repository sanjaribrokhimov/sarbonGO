package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/companies"
	"sarbonNew/internal/dispatchercompanies"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
)

type DispatcherCompaniesHandler struct {
	logger *zap.Logger
	repo   *companies.Repo
	dcr    *dispatchercompanies.Repo
	jwtm   *security.JWTManager
}

func NewDispatcherCompaniesHandler(logger *zap.Logger, repo *companies.Repo, dcr *dispatchercompanies.Repo, jwtm *security.JWTManager) *DispatcherCompaniesHandler {
	return &DispatcherCompaniesHandler{logger: logger, repo: repo, dcr: dcr, jwtm: jwtm}
}

// CreateCompanyReq body for POST /v1/dispatchers/companies
type CreateCompanyReq struct {
	Name string `json:"name" binding:"required,max=255"`
}

// CreateCompany creates a Broker company owned by the current dispatcher (freelance "own company" flow).
func (h *DispatcherCompaniesHandler) CreateCompany(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	var req CreateCompanyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		resp.Error(c, http.StatusBadRequest, "name is required")
		return
	}
	companyID, err := h.repo.CreateWithOwnerDispatcher(c.Request.Context(), name, dispatcherID)
	if err != nil {
		h.logger.Error("dispatcher create company", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to create company")
		return
	}
	if err := h.dcr.Add(c.Request.Context(), dispatcherID, companyID, "owner"); err != nil {
		h.logger.Error("dispatcher_company_roles add", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to link company")
		return
	}
	resp.Success(c, http.StatusCreated, "created", gin.H{"id": companyID.String(), "name": name})
}

// ListMyCompanies returns companies where dispatcher is owner or has role (GET /v1/dispatchers/companies).
func (h *DispatcherCompaniesHandler) ListMyCompanies(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	list, err := h.repo.ListForDispatcher(c.Request.Context(), dispatcherID)
	if err != nil {
		h.logger.Error("list for dispatcher", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to list companies")
		return
	}
	if list == nil {
		list = []companies.CompanyWithRole{}
	}
	resp.Success(c, http.StatusOK, "ok", gin.H{"items": list})
}

// SwitchCompanyReq body for POST /v1/dispatchers/auth/switch-company
type SwitchCompanyReq struct {
	CompanyID string `json:"company_id" binding:"required,uuid"`
}

// SwitchCompany issues new access token with company_id so further requests are scoped to that company.
func (h *DispatcherCompaniesHandler) SwitchCompany(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	var req SwitchCompanyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	companyID, err := uuid.Parse(req.CompanyID)
	if err != nil || companyID == uuid.Nil {
		resp.Error(c, http.StatusBadRequest, "invalid company_id")
		return
	}
	ok, err := h.dcr.HasAccess(c.Request.Context(), dispatcherID, companyID)
	if err != nil || !ok {
		resp.Error(c, http.StatusForbidden, "company not found or access denied")
		return
	}
	tokens, _, err := h.jwtm.IssueWithCompany("dispatcher", dispatcherID, companyID)
	if err != nil {
		h.logger.Error("issue token with company", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to issue token")
		return
	}
	resp.Success(c, http.StatusOK, "ok", gin.H{
		"access_token":        tokens.AccessToken,
		"refresh_token":       tokens.RefreshToken,
		"expires_in":          tokens.ExpiresIn,
		"expires_at":          tokens.ExpiresAt,
		"refresh_expires_in":  tokens.RefreshExpiresIn,
		"refresh_expires_at":  tokens.RefreshExpiresAt,
	})
}
