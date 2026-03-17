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

// Поток: 1) POST /v1/admin/companies — только создание компании (без владельца).
//        2) PATCH /v1/admin/companies/:id/owner — привязать владельца из company_users.

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

	CompanyType *string `json:"company_type"` // CARGO_OWNER, CARRIER, EXPEDITOR; владельца (CEO) назначают отдельно через PATCH .../owner

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
		resp.ErrorLang(c, http.StatusBadRequest, "invalid payload")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		resp.ErrorLang(c, http.StatusBadRequest, "name_required")
		return
	}

	rawAdminID, ok := c.Get(mw.CtxAdminID)
	if !ok {
		resp.ErrorLang(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	adminID, ok := rawAdminID.(uuid.UUID)
	if !ok || adminID == uuid.Nil {
		resp.ErrorLang(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	var companyType *string
	if req.CompanyType != nil {
		s := strings.TrimSpace(strings.ToUpper(*req.CompanyType))
		if s != "" && (s == "CARGO_OWNER" || s == "CARRIER" || s == "EXPEDITOR") {
			// В БД храним PascalCase
			dbVal := map[string]string{
				"CARGO_OWNER": "CargoOwner",
				"CARRIER":     "Carrier",
				"EXPEDITOR":   "Expeditor",
			}[s]
			companyType = &dbVal
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
		OwnerID:           nil, // владелец назначается отдельно через PATCH /v1/admin/companies/:id/owner
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
		resp.ErrorLang(c, http.StatusInternalServerError, "company_create_failed")
		return
	}

	resp.OKLang(c, "created", gin.H{"company_id": id})
}

// adminSetOwnerReq body for PATCH /admin/companies/:id/owner
type adminSetOwnerReq struct {
	OwnerID string `json:"owner_id" binding:"required"` // company_users.id
}

// SetOwner PATCH /admin/companies/:id/owner — привязывает владельца к компании и переводит status в active.
func (h *AdminCompaniesHandler) SetOwner(c *gin.Context) {
	companyIDStr := c.Param("id")
	if companyIDStr == "" {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_company_id")
		return
	}
	companyID, err := uuid.Parse(companyIDStr)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_company_id")
		return
	}
	var req adminSetOwnerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid payload")
		return
	}
	ownerID, err := uuid.Parse(strings.TrimSpace(req.OwnerID))
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	user, err := h.usersRepo.FindByID(c.Request.Context(), ownerID)
	if err != nil {
		if errors.Is(err, appusers.ErrNotFound) {
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_user_id")
			return
		}
		h.logger.Error("company user find by id failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	// Основателем/владельцем (CEO) может быть только пользователь с ролью CEO в company_users
	if user.Role == nil || strings.TrimSpace(*user.Role) != "CEO" {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	if err := h.repo.SetOwner(c.Request.Context(), companyID, ownerID); err != nil {
		h.logger.Error("company set owner failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	// Двусторонняя связь: у владельца в company_users тоже проставляем company_id
	if err := h.usersRepo.UpdateCompanyID(c.Request.Context(), ownerID, &companyID); err != nil {
		h.logger.Error("company user company_id update failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "ok", gin.H{"status": "ok", "message": "owner set, status set to active"})
}

// SearchOwners GET /admin/company-users/owners/search?q=... — поиск владельцев (company_users с role=owner) по телефону, имени или фамилии. Сортировка: точное совпадение → начинается с → содержит.
func (h *AdminCompaniesHandler) SearchOwners(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		resp.OKLang(c, "ok", []any{})
		return
	}
	list, err := h.usersRepo.SearchOwners(c.Request.Context(), q, 50)
	if err != nil {
		h.logger.Error("search owners failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	// Без password_hash в ответе
	out := make([]gin.H, 0, len(list))
	for _, u := range list {
		out = append(out, gin.H{
			"id":          u.ID,
			"phone":       u.Phone,
			"first_name": u.FirstName,
			"last_name":   u.LastName,
			"company_id":  u.CompanyID,
			"role":        u.Role,
			"created_at":  u.CreatedAt,
			"updated_at":  u.UpdatedAt,
		})
	}
	resp.OKLang(c, "ok", out)
}

