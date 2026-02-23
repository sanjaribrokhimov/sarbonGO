package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/companies"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
)

type AdminCompaniesHandler struct {
	logger *zap.Logger
	repo   *companies.Repo
}

func NewAdminCompaniesHandler(logger *zap.Logger, repo *companies.Repo) *AdminCompaniesHandler {
	return &AdminCompaniesHandler{logger: logger, repo: repo}
}

type adminCreateCompanyReq struct {
	Name          string  `json:"name" binding:"required"`
	Inn           *string `json:"inn"`
	Address       *string `json:"address"`
	Phone         *string `json:"phone"`
	Email         *string `json:"email"`
	Website       *string `json:"website"`
	LicenseNumber *string `json:"license_number"`
	Status        *string `json:"status"`

	MaxVehicles       int `json:"max_vehicles"`
	MaxDrivers        int `json:"max_drivers"`
	MaxCargo          int `json:"max_cargo"`
	MaxDispatchers    int `json:"max_dispatchers"`
	MaxManagers       int `json:"max_managers"`
	MaxTopDispatchers int `json:"max_top_dispatchers"`
	MaxTopManagers    int `json:"max_top_managers"`
}

func (h *AdminCompaniesHandler) Create(c *gin.Context) {
	var req adminCreateCompanyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		resp.Error(c, http.StatusBadRequest, "name is required")
		return
	}

	rawAdminID, ok := c.Get(mw.CtxAdminID)
	if !ok {
		resp.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	adminID, ok := rawAdminID.(uuid.UUID)
	if !ok || adminID == uuid.Nil {
		resp.Error(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := h.repo.Create(c.Request.Context(), companies.CreateParams{
		Name:             req.Name,
		Inn:              req.Inn,
		Address:          req.Address,
		Phone:            req.Phone,
		Email:            req.Email,
		Website:          req.Website,
		LicenseNumber:    req.LicenseNumber,
		Status:           req.Status,
		MaxVehicles:      req.MaxVehicles,
		MaxDrivers:       req.MaxDrivers,
		MaxCargo:         req.MaxCargo,
		MaxDispatchers:   req.MaxDispatchers,
		MaxManagers:      req.MaxManagers,
		MaxTopDispatchers: req.MaxTopDispatchers,
		MaxTopManagers:   req.MaxTopManagers,
		CreatedBy:        adminID,
	})
	if err != nil {
		h.logger.Error("company create failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "company create failed")
		return
	}

	resp.OK(c, gin.H{"company_id": id})
}

