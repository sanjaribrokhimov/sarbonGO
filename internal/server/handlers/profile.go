package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/drivers"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
	"sarbonNew/internal/telegram"
	"sarbonNew/internal/util"
)

type ProfileHandler struct {
	logger *zap.Logger
	drivers *drivers.Repo
	phoneChange *store.PhoneChangeStore
	tg *telegram.GatewayClient
	otpTTL time.Duration
	otpLen int
}

func NewProfileHandler(logger *zap.Logger, driversRepo *drivers.Repo, phoneChange *store.PhoneChangeStore, tg *telegram.GatewayClient, otpTTL time.Duration, otpLen int) *ProfileHandler {
	return &ProfileHandler{logger: logger, drivers: driversRepo, phoneChange: phoneChange, tg: tg, otpTTL: otpTTL, otpLen: otpLen}
}

// GET /v1/profile
func (h *ProfileHandler) Get(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	_ = h.drivers.TouchOnline(c.Request.Context(), driverID)
	d, err := h.drivers.FindByID(c.Request.Context(), driverID)
	if err != nil {
		resp.Error(c, http.StatusUnauthorized, "driver not found")
		return
	}
	resp.OK(c, gin.H{"driver": d})
}

type patchDriverReq struct {
	Name                 *string `json:"name,omitempty"`
	WorkStatus           *string `json:"work_status,omitempty"` // available|loaded|busy
	DriverPassportSeries *string `json:"driver_passport_series,omitempty"`
	DriverPassportNumber *string `json:"driver_passport_number,omitempty"`
	DriverPINFL          *string `json:"driver_pinfl,omitempty"`
}

// PATCH /v1/profile/driver
func (h *ProfileHandler) PatchDriver(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req patchDriverReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}

	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		if len(v) < 2 {
			resp.Error(c, http.StatusBadRequest, "name is too short")
			return
		}
		req.Name = &v
	}
	if req.WorkStatus != nil {
		v := strings.ToLower(strings.TrimSpace(*req.WorkStatus))
		switch v {
		case "available", "loaded", "busy":
			req.WorkStatus = &v
		default:
			resp.Error(c, http.StatusBadRequest, "invalid work_status (allowed: available, loaded, busy)")
			return
		}
	}
	if req.DriverPassportSeries != nil {
		v := strings.TrimSpace(*req.DriverPassportSeries)
		if v == "" {
			resp.Error(c, http.StatusBadRequest, "driver_passport_series cannot be empty")
			return
		}
		req.DriverPassportSeries = &v
	}
	if req.DriverPassportNumber != nil {
		v := strings.TrimSpace(*req.DriverPassportNumber)
		if v == "" {
			resp.Error(c, http.StatusBadRequest, "driver_passport_number cannot be empty")
			return
		}
		req.DriverPassportNumber = &v
	}
	if req.DriverPINFL != nil {
		v := strings.TrimSpace(*req.DriverPINFL)
		if v == "" {
			resp.Error(c, http.StatusBadRequest, "driver_pinfl cannot be empty")
			return
		}
		req.DriverPINFL = &v
	}

	if err := h.drivers.UpdateDriverEditable(c.Request.Context(), driverID, drivers.UpdateDriverEditable{
		Name: req.Name, WorkStatus: req.WorkStatus,
		DriverPassportSeries: req.DriverPassportSeries,
		DriverPassportNumber: req.DriverPassportNumber,
		DriverPINFL:          req.DriverPINFL,
	}); err != nil {
		h.logger.Error("update driver profile failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	d, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OK(c, gin.H{"event": "updated", "driver": d})
}

type heartbeatReq struct {
	Latitude     float64 `json:"latitude" binding:"required"`
	Longitude    float64 `json:"longitude" binding:"required"`
	LastOnlineAt string  `json:"last_online_at" binding:"required"` // RFC3339
}

// PUT /v1/profile/heartbeat
func (h *ProfileHandler) Heartbeat(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req heartbeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(req.LastOnlineAt))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "last_online_at must be RFC3339")
		return
	}
	if err := h.drivers.UpdateHeartbeat(c.Request.Context(), driverID, req.Latitude, req.Longitude, t); err != nil {
		h.logger.Error("heartbeat update failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	d, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OK(c, gin.H{"event": "heartbeat", "driver": d})
}

type phoneChangeRequestReq struct {
	NewPhone string `json:"new_phone" binding:"required"`
}

// POST /v1/profile/phone-change/request
func (h *ProfileHandler) PhoneChangeRequest(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req phoneChangeRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	newPhone, err := util.ValidateUzPhoneStrict(req.NewPhone)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	// Check uniqueness
	if _, err := h.drivers.FindByPhone(c.Request.Context(), newPhone); err == nil {
		resp.Error(c, http.StatusConflict, "this phone is already registered")
		return
	}

	code, err := util.GenerateNumericOTP(h.otpLen)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "otp generation failed")
		return
	}
	if _, err := h.tg.SendVerificationMessage(c.Request.Context(), newPhone, code, int(h.otpTTL.Seconds())); err != nil {
		h.logger.Warn("telegram sendVerificationMessage failed", zap.Error(err))
		resp.Error(c, http.StatusBadGateway, "otp send failed")
		return
	}
	sessionID, err := h.phoneChange.Create(c.Request.Context(), driverID, newPhone, code)
	if err != nil {
		h.logger.Error("phone change session create failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"event": "otp_sent", "session_id": sessionID, "ttl_seconds": int(h.otpTTL.Seconds())})
}

type phoneChangeVerifyReq struct {
	SessionID string `json:"session_id" binding:"required"`
	OTP       string `json:"otp" binding:"required"`
}

// POST /v1/profile/phone-change/verify
func (h *ProfileHandler) PhoneChangeVerify(c *gin.Context) {
	var req phoneChangeVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	sessionID := strings.TrimSpace(req.SessionID)
	otp := strings.TrimSpace(req.OTP)
	if otp == "" {
		resp.Error(c, http.StatusBadRequest, "otp is required")
		return
	}

	rec, err := h.phoneChange.Verify(c.Request.Context(), sessionID, otp)
	if err != nil {
		switch err {
		case store.ErrPhoneChangeOTPExpired:
			resp.Error(c, http.StatusUnauthorized, "session expired or invalid")
		case store.ErrPhoneChangeOTPInvalid:
			resp.Error(c, http.StatusUnauthorized, "otp invalid")
		case store.ErrPhoneChangeMaxAttempts:
			resp.Error(c, http.StatusTooManyRequests, "otp max attempts exceeded")
		default:
			h.logger.Error("phone change verify failed", zap.Error(err))
			resp.Error(c, http.StatusInternalServerError, "internal error")
		}
		return
	}

	if err := h.drivers.UpdatePhone(c.Request.Context(), rec.DriverID, rec.NewPhone); err != nil {
		if err == drivers.ErrPhoneAlreadyRegistered {
			resp.Error(c, http.StatusConflict, "this phone is already registered")
			return
		}
		h.logger.Error("phone update failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	d, _ := h.drivers.FindByID(c.Request.Context(), rec.DriverID)
	resp.OK(c, gin.H{"event": "phone_updated", "driver": d})
}

type patchPowerReq struct {
	PowerPlateNumber *string `json:"power_plate_number,omitempty"`
	PowerTechSeries  *string `json:"power_tech_series,omitempty"`
	PowerTechNumber  *string `json:"power_tech_number,omitempty"`
	PowerOwnerID     *string `json:"power_owner_id,omitempty"`
	PowerOwnerName   *string `json:"power_owner_name,omitempty"`
	PowerScanStatus  *bool   `json:"power_scan_status,omitempty"`
}

// PATCH /v1/profile/power
func (h *ProfileHandler) PatchPower(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req patchPowerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}

	trimPtr := func(p **string) {
		if *p == nil {
			return
		}
		v := strings.TrimSpace(**p)
		if v == "" {
			*p = nil
			return
		}
		*p = &v
	}
	trimPtr(&req.PowerPlateNumber)
	trimPtr(&req.PowerTechSeries)
	trimPtr(&req.PowerTechNumber)
	trimPtr(&req.PowerOwnerID)
	trimPtr(&req.PowerOwnerName)

	if err := h.drivers.UpdatePowerProfile(c.Request.Context(), driverID, drivers.UpdatePowerProfile{
		PowerPlateNumber: req.PowerPlateNumber,
		PowerTechSeries:  req.PowerTechSeries,
		PowerTechNumber:  req.PowerTechNumber,
		PowerOwnerID:     req.PowerOwnerID,
		PowerOwnerName:   req.PowerOwnerName,
		PowerScanStatus:  req.PowerScanStatus,
	}); err != nil {
		h.logger.Error("update power profile failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	d, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OK(c, gin.H{"event": "updated", "driver": d})
}

type patchTrailerReq struct {
	TrailerPlateNumber *string `json:"trailer_plate_number,omitempty"`
	TrailerTechSeries  *string `json:"trailer_tech_series,omitempty"`
	TrailerTechNumber  *string `json:"trailer_tech_number,omitempty"`
	TrailerOwnerID     *string `json:"trailer_owner_id,omitempty"`
	TrailerOwnerName   *string `json:"trailer_owner_name,omitempty"`
	TrailerScanStatus  *bool   `json:"trailer_scan_status,omitempty"`
}

// PATCH /v1/profile/trailer
func (h *ProfileHandler) PatchTrailer(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req patchTrailerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}

	trimPtr := func(p **string) {
		if *p == nil {
			return
		}
		v := strings.TrimSpace(**p)
		if v == "" {
			*p = nil
			return
		}
		*p = &v
	}
	trimPtr(&req.TrailerPlateNumber)
	trimPtr(&req.TrailerTechSeries)
	trimPtr(&req.TrailerTechNumber)
	trimPtr(&req.TrailerOwnerID)
	trimPtr(&req.TrailerOwnerName)

	if err := h.drivers.UpdateTrailerProfile(c.Request.Context(), driverID, drivers.UpdateTrailerProfile{
		TrailerPlateNumber: req.TrailerPlateNumber,
		TrailerTechSeries:  req.TrailerTechSeries,
		TrailerTechNumber:  req.TrailerTechNumber,
		TrailerOwnerID:     req.TrailerOwnerID,
		TrailerOwnerName:   req.TrailerOwnerName,
		TrailerScanStatus:  req.TrailerScanStatus,
	}); err != nil {
		h.logger.Error("update trailer profile failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	d, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OK(c, gin.H{"event": "updated", "driver": d})
}

// DELETE /v1/profile
func (h *ProfileHandler) Delete(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	if err := h.drivers.DeleteAndArchive(c.Request.Context(), driverID); err != nil {
		if err == drivers.ErrDeleteNotFound {
			resp.Error(c, http.StatusNotFound, "driver not found")
			return
		}
		h.logger.Error("delete profile failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "ok"})
}

