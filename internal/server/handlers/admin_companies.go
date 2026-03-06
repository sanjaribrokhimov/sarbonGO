package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/appusers"
	"sarbonNew/internal/companies"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
)

type AdminCompaniesHandler struct {
	logger   *zap.Logger
	repo     *companies.Repo
	usersRepo *appusers.Repo
}

func NewAdminCompaniesHandler(logger *zap.Logger, repo *companies.Repo, usersRepo *appusers.Repo) *AdminCompaniesHandler {
	return &AdminCompaniesHandler{logger: logger, repo: repo, usersRepo: usersRepo}
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

	// Owner (app_users id): при указании компания получает owner и status = active
	OwnerID     *string `json:"owner_id"`     // UUID
	CompanyType *string `json:"company_type"` // CargoOwner, Carrier, Expeditor (грузовладелец, перевозчик, экспедитор)

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

	var ownerID *uuid.UUID
	if req.OwnerID != nil {
		s := strings.TrimSpace(*req.OwnerID)
		if s != "" {
			parsed, err := uuid.Parse(s)
			if err != nil {
				resp.Error(c, http.StatusBadRequest, "invalid owner_id: must be UUID")
				return
			}
			_, err = h.usersRepo.FindByID(c.Request.Context(), parsed)
			if err != nil {
				if errors.Is(err, appusers.ErrNotFound) {
					resp.Error(c, http.StatusBadRequest, "owner_id must be an existing company user (register via POST /v1/company-users/auth/phone and registration/complete first)")
					return
				}
				h.logger.Error("company user find by id failed", zap.Error(err))
				resp.Error(c, http.StatusInternalServerError, "company create failed")
				return
			}
			ownerID = &parsed
		}
	}
	var companyType *string
	if req.CompanyType != nil {
		s := strings.TrimSpace(*req.CompanyType)
		if s != "" {
			companyType = &s
		}
	}

	id, err := h.repo.Create(c.Request.Context(), companies.CreateParams{
		Name:              req.Name,
		Inn:               req.Inn,
		Address:           req.Address,
		Phone:             req.Phone,
		Email:             req.Email,
		Website:           req.Website,
		LicenseNumber:     req.LicenseNumber,
		Status:            req.Status,
		OwnerID:           ownerID,
		CompanyType:       companyType,
		MaxVehicles:       req.MaxVehicles,
		MaxDrivers:        req.MaxDrivers,
		MaxCargo:          req.MaxCargo,
		MaxDispatchers:    req.MaxDispatchers,
		MaxManagers:       req.MaxManagers,
		MaxTopDispatchers: req.MaxTopDispatchers,
		MaxTopManagers:    req.MaxTopManagers,
		CreatedBy:         adminID,
	})
	if err != nil {
		h.logger.Error("company create failed", zap.Error(err))
		msg := "company create failed"
		if errStr := err.Error(); errStr != "" {
			msg = msg + ": " + errStr
		}
		resp.Error(c, http.StatusInternalServerError, msg)
		return
	}

	resp.OK(c, gin.H{"company_id": id})
}

// adminSetOwnerReq body for PATCH /admin/companies/:id/owner
type adminSetOwnerReq struct {
	OwnerID string `json:"owner_id" binding:"required"` // company_users.id
}

// SetOwner PATCH /admin/companies/:id/owner — привязывает владельца к компании и переводит status в active.
func (h *AdminCompaniesHandler) SetOwner(c *gin.Context) {
	companyIDStr := c.Param("id")
	if companyIDStr == "" {
		resp.Error(c, http.StatusBadRequest, "company id required")
		return
	}
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid company id")
		return
	}
	var req adminSetOwnerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	ownerID, err := uuid.Parse(strings.TrimSpace(req.OwnerID))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid owner_id: must be UUID")
		return
	}
	_, err = h.usersRepo.FindByID(c.Request.Context(), ownerID)
	if err != nil {
		if errors.Is(err, appusers.ErrNotFound) {
			resp.Error(c, http.StatusBadRequest, "owner_id must be an existing company user (register via POST /v1/company-users/auth/phone and registration/complete first)")
			return
		}
		h.logger.Error("company user find by id failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "set owner failed")
		return
	}
	if err := h.repo.SetOwner(c.Request.Context(), companyID, ownerID); err != nil {
		h.logger.Error("company set owner failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "set owner failed")
		return
	}
	resp.OK(c, gin.H{"status": "ok", "message": "owner set, status set to active"})
}

