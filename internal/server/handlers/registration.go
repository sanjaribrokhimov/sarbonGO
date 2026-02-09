package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/domain"
	"sarbonNew/internal/drivers"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
)

type RegistrationHandler struct {
	logger *zap.Logger

	drivers  *drivers.Repo
	sessions *store.SessionStore
	jwtm     *security.JWTManager
	refresh  *store.RefreshStore
}

func NewRegistrationHandler(logger *zap.Logger, driversRepo *drivers.Repo, sessions *store.SessionStore, jwtm *security.JWTManager, refresh *store.RefreshStore) *RegistrationHandler {
	return &RegistrationHandler{
		logger:  logger,
		drivers: driversRepo,
		sessions: sessions,
		jwtm:    jwtm,
		refresh: refresh,
	}
}

type startReq struct {
	SessionID      string `json:"session_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	OfertaAccepted bool   `json:"oferta_accepted" binding:"required"`
}

func (h *RegistrationHandler) Start(c *gin.Context) {
	var req startReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if !req.OfertaAccepted {
		resp.Error(c, http.StatusBadRequest, "oferta_accepted must be true")
		return
	}
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 {
		resp.Error(c, http.StatusBadRequest, "name is too short")
		return
	}

	phone, err := h.sessions.Consume(c.Request.Context(), strings.TrimSpace(req.SessionID))
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) {
			resp.Error(c, http.StatusUnauthorized, "session_id expired or invalid")
			return
		}
		h.logger.Error("consume session failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	// Idempotency / race: if already exists, return login tokens.
	existing, err := h.drivers.FindByPhone(c.Request.Context(), phone)
	if err == nil {
		driverUUID, _ := uuid.Parse(existing.ID)
		tokens, refreshClaims, err := h.jwtm.Issue(driverUUID)
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, "token issue failed")
			return
		}
		_ = h.refresh.Put(c.Request.Context(), refreshClaims.DriverID, refreshClaims.JTI)

		resp.OK(c, gin.H{
			"event":  "login",
			"tokens": tokens,
			"driver": existing,
		})
		return
	}
	if !errors.Is(err, drivers.ErrNotFound) {
		h.logger.Error("find by phone failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	id, err := h.drivers.CreateStart(c.Request.Context(), phone, name)
	if err != nil {
		h.logger.Error("create driver failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	tokens, refreshClaims, err := h.jwtm.Issue(id)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "token issue failed")
		return
	}
	_ = h.refresh.Put(c.Request.Context(), refreshClaims.DriverID, refreshClaims.JTI)

	drv, _ := h.drivers.FindByID(c.Request.Context(), id)
	resp.OK(c, gin.H{
		"event":               "registered",
		"tokens":              tokens,
		"driver":              drv,
		"registration_status": domain.StatusStart,
		"registration_step":   domain.StepNameOferta,
	})
}

type geoReq struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	PushToken *string `json:"push_token,omitempty"`
}

func (h *RegistrationHandler) GeoPush(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)

	var req geoReq
	_ = c.ShouldBindJSON(&req) // payload is optional for no-op

	d, err := h.drivers.FindByID(c.Request.Context(), driverID)
	if err != nil {
		resp.Error(c, http.StatusUnauthorized, "driver not found")
		return
	}

	// If latitude/longitude not provided => do not advance step.
	if req.Latitude == nil || req.Longitude == nil {
		if req.PushToken != nil && strings.TrimSpace(*req.PushToken) != "" {
			if err := h.drivers.UpdatePushToken(c.Request.Context(), driverID, strings.TrimSpace(*req.PushToken)); err != nil {
				h.logger.Error("update push token failed", zap.Error(err))
				resp.Error(c, http.StatusInternalServerError, "internal error")
				return
			}
			updated, _ := h.drivers.FindByID(c.Request.Context(), driverID)
			resp.OK(c, gin.H{"event": "updated", "driver": updated})
			return
		}
		_ = h.drivers.TouchOnline(c.Request.Context(), driverID)
		resp.OK(c, gin.H{"event": "noop", "driver": d})
		return
	}

	nextStep := domain.StepGeoPush
	if d.RegistrationStep != nil && *d.RegistrationStep == string(domain.StepTransportType) {
		nextStep = domain.StepTransportType
	}

	if err := h.drivers.UpdateGeo(c.Request.Context(), driverID, *req.Latitude, *req.Longitude, string(nextStep), req.PushToken); err != nil {
		h.logger.Error("update geo failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	updated, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OK(c, gin.H{"event": "updated", "driver": updated})
}

type transportReq struct {
	DriverType      string  `json:"driver_type" binding:"required"`       // company|freelancer|driver
	PowerPlateType  string  `json:"power_plate_type" binding:"required"`  // required for completing step
	TrailerPlateType string `json:"trailer_plate_type" binding:"required"` // required for completing step
	FreelancerID    *string `json:"freelancer_id,omitempty"`              // uuid (optional)
	CompanyID       *string `json:"company_id,omitempty"`                 // uuid (optional)
}

func (h *RegistrationHandler) TransportType(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)

	var req transportReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}

	d, err := h.drivers.FindByID(c.Request.Context(), driverID)
	if err != nil {
		resp.Error(c, http.StatusUnauthorized, "driver not found")
		return
	}

	dt := strings.ToLower(strings.TrimSpace(req.DriverType))
	switch domain.DriverType(dt) {
	case domain.DriverTypeCompany, domain.DriverTypeFreelancer, domain.DriverTypeDriver:
	default:
		resp.Error(c, http.StatusBadRequest, "invalid driver_type")
		return
	}

	ppt := strings.TrimSpace(req.PowerPlateType)
	tpt := strings.TrimSpace(req.TrailerPlateType)
	if ppt == "" {
		resp.Error(c, http.StatusBadRequest, "power_plate_type is required")
		return
	}
	if tpt == "" {
		resp.Error(c, http.StatusBadRequest, "trailer_plate_type is required")
		return
	}

	var freelancerUUID *uuid.UUID
	if req.FreelancerID != nil && strings.TrimSpace(*req.FreelancerID) != "" {
		u, err := uuid.Parse(strings.TrimSpace(*req.FreelancerID))
		if err != nil {
			resp.Error(c, http.StatusBadRequest, "invalid freelancer_id")
			return
		}
		freelancerUUID = &u
	}
	var companyUUID *uuid.UUID
	if req.CompanyID != nil && strings.TrimSpace(*req.CompanyID) != "" {
		u, err := uuid.Parse(strings.TrimSpace(*req.CompanyID))
		if err != nil {
			resp.Error(c, http.StatusBadRequest, "invalid company_id")
			return
		}
		companyUUID = &u
	}

	nextStatus := string(domain.StatusBasic)
	if d.RegistrationStatus != nil && *d.RegistrationStatus == string(domain.StatusFull) {
		nextStatus = string(domain.StatusFull)
	}

	if err := h.drivers.UpdateTransportType(c.Request.Context(), driverID, dt, freelancerUUID, companyUUID, ppt, tpt, string(domain.StepTransportType), nextStatus); err != nil {
		h.logger.Error("update transport type failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	updated, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OK(c, gin.H{"event": "updated", "driver": updated})
}

