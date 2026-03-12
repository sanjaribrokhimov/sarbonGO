package handlers

import (
	"errors"
	"io"
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

const maxDriverPhotoSize = 5 * 1024 * 1024 // 5 MB
var allowedDriverPhotoTypes = map[string]bool{"image/jpeg": true, "image/png": true}

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

// GET /v1/driver/profile
func (h *ProfileHandler) Get(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	_ = h.drivers.TouchOnline(c.Request.Context(), driverID)
	d, err := h.drivers.FindByID(c.Request.Context(), driverID)
	if err != nil {
		resp.ErrorLang(c, http.StatusUnauthorized, "driver_not_found")
		return
	}
	resp.OKLang(c, "ok", gin.H{"driver": d})
}

type patchDriverReq struct {
	Name                 *string `json:"name,omitempty"`
	WorkStatus           *string `json:"work_status,omitempty"` // available|loaded|busy
	DriverPassportSeries *string `json:"driver_passport_series,omitempty"`
	DriverPassportNumber *string `json:"driver_passport_number,omitempty"`
	DriverPINFL          *string `json:"driver_pinfl,omitempty"`
}

// PATCH /v1/driver/profile/driver
func (h *ProfileHandler) PatchDriver(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req patchDriverReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}

	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		if len(v) < 2 {
			resp.ErrorLang(c, http.StatusBadRequest, "name_too_short")
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
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_work_status")
			return
		}
	}
	if req.DriverPassportSeries != nil {
		v := strings.TrimSpace(*req.DriverPassportSeries)
		if v == "" {
			resp.ErrorLang(c, http.StatusBadRequest, "driver_passport_series_required")
			return
		}
		req.DriverPassportSeries = &v
	}
	if req.DriverPassportNumber != nil {
		v := strings.TrimSpace(*req.DriverPassportNumber)
		if v == "" {
			resp.ErrorLang(c, http.StatusBadRequest, "driver_passport_number_required")
			return
		}
		req.DriverPassportNumber = &v
	}
	if req.DriverPINFL != nil {
		v := strings.TrimSpace(*req.DriverPINFL)
		if v == "" {
			resp.ErrorLang(c, http.StatusBadRequest, "driver_pinfl_required")
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
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}

	d, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OKLang(c, "updated", gin.H{"event": "updated", "driver": d})
}

type heartbeatReq struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

// PUT /v1/driver/profile/heartbeat — только latitude и longitude; last_online_at всегда обновляется на сервере автоматически.
func (h *ProfileHandler) Heartbeat(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req heartbeatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	if err := h.drivers.UpdateHeartbeat(c.Request.Context(), driverID, req.Latitude, req.Longitude, time.Now().UTC()); err != nil {
		h.logger.Error("heartbeat update failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	d, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OKLang(c, "heartbeat", gin.H{"event": "heartbeat", "driver": d})
}

type phoneChangeRequestReq struct {
	NewPhone string `json:"new_phone" binding:"required"`
}

// POST /v1/driver/profile/phone-change/request
func (h *ProfileHandler) PhoneChangeRequest(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req phoneChangeRequestReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	newPhone, err := util.ValidateUzPhoneStrict(req.NewPhone)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	// Check uniqueness
	if _, err := h.drivers.FindByPhone(c.Request.Context(), newPhone); err == nil {
		resp.ErrorLang(c, http.StatusConflict, "phone_already_registered")
		return
	}

	ttlSec := int(h.otpTTL.Seconds())
	code, _, err := SendOTP(c.Request.Context(), h.tg, newPhone, ttlSec, h.otpLen)
	if err != nil {
		if WriteOTPSendError(c, err, h.logger, "telegram sendVerificationMessage failed") {
			return
		}
		resp.ErrorLang(c, http.StatusInternalServerError, "otp_generation_failed")
		return
	}
	sessionID, err := h.phoneChange.Create(c.Request.Context(), driverID, newPhone, code)
	if err != nil {
		h.logger.Error("phone change session create failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "otp_sent", gin.H{"event": "otp_sent", "session_id": sessionID, "ttl_seconds": ttlSec})
}

type phoneChangeVerifyReq struct {
	SessionID string `json:"session_id" binding:"required"`
	OTP       string `json:"otp" binding:"required"`
}

// POST /v1/driver/profile/phone-change/verify
func (h *ProfileHandler) PhoneChangeVerify(c *gin.Context) {
	var req phoneChangeVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	sessionID := strings.TrimSpace(req.SessionID)
	otp := strings.TrimSpace(req.OTP)
	if otp == "" {
		resp.ErrorLang(c, http.StatusBadRequest, "otp_required")
		return
	}

	rec, err := h.phoneChange.Verify(c.Request.Context(), sessionID, otp)
	if err != nil {
		switch err {
		case store.ErrPhoneChangeOTPExpired:
			resp.ErrorLang(c, http.StatusUnauthorized, "session_expired_or_invalid")
		case store.ErrPhoneChangeOTPInvalid:
			resp.ErrorLang(c, http.StatusUnauthorized, "otp_invalid")
		case store.ErrPhoneChangeMaxAttempts:
			resp.ErrorLang(c, http.StatusTooManyRequests, "otp_max_attempts_exceeded")
		default:
			h.logger.Error("phone change verify failed", zap.Error(err))
			resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	if err := h.drivers.UpdatePhone(c.Request.Context(), rec.DriverID, rec.NewPhone); err != nil {
		if err == drivers.ErrPhoneAlreadyRegistered {
			resp.ErrorLang(c, http.StatusConflict, "phone_already_registered")
			return
		}
		h.logger.Error("phone update failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	d, _ := h.drivers.FindByID(c.Request.Context(), rec.DriverID)
	resp.OKLang(c, "phone_updated", gin.H{"event": "phone_updated", "driver": d})
}

type patchPowerReq struct {
	PowerPlateNumber *string `json:"power_plate_number,omitempty"`
	PowerTechSeries  *string `json:"power_tech_series,omitempty"`
	PowerTechNumber  *string `json:"power_tech_number,omitempty"`
	PowerOwnerID     *string `json:"power_owner_id,omitempty"`
	PowerOwnerName   *string `json:"power_owner_name,omitempty"`
	PowerScanStatus  *bool   `json:"power_scan_status,omitempty"`
}

// PATCH /v1/driver/profile/power
func (h *ProfileHandler) PatchPower(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req patchPowerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
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
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	d, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OKLang(c, "updated", gin.H{"event": "updated", "driver": d})
}

type patchTrailerReq struct {
	TrailerPlateNumber *string `json:"trailer_plate_number,omitempty"`
	TrailerTechSeries  *string `json:"trailer_tech_series,omitempty"`
	TrailerTechNumber  *string `json:"trailer_tech_number,omitempty"`
	TrailerOwnerID     *string `json:"trailer_owner_id,omitempty"`
	TrailerOwnerName   *string `json:"trailer_owner_name,omitempty"`
	TrailerScanStatus  *bool   `json:"trailer_scan_status,omitempty"`
}

// PATCH /v1/driver/profile/trailer
func (h *ProfileHandler) PatchTrailer(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req patchTrailerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
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
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	d, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OKLang(c, "updated", gin.H{"event": "updated", "driver": d})
}

// DELETE /v1/driver/profile
func (h *ProfileHandler) Delete(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	if err := h.drivers.DeleteAndArchive(c.Request.Context(), driverID); err != nil {
		if err == drivers.ErrDeleteNotFound {
			resp.ErrorLang(c, http.StatusNotFound, "driver_not_found")
			return
		}
		h.logger.Error("delete profile failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "ok", gin.H{"status": "ok"})
}

// UploadPhoto — POST multipart/form-data с полем "photo". Фото необязательно при регистрации; можно добавить/обновить когда угодно.
func (h *ProfileHandler) UploadPhoto(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	file, err := c.FormFile("photo")
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "photo_file_required")
		return
	}
	if file.Size > maxDriverPhotoSize {
		resp.ErrorLang(c, http.StatusBadRequest, "file_too_large")
		return
	}
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	if !allowedDriverPhotoTypes[contentType] {
		resp.ErrorLang(c, http.StatusBadRequest, "allowed_image_types")
		return
	}
	f, err := file.Open()
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "cannot_read_file")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "cannot_read_file")
		return
	}
	if err := h.drivers.UpdatePhoto(c.Request.Context(), driverID, data, contentType); err != nil {
		h.logger.Error("driver photo update failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "photo_uploaded", gin.H{"status": "ok", "event": "photo_uploaded"})
}

// GetPhoto — GET фото водителя (бинарный ответ с Content-Type). 404 если фото нет.
func (h *ProfileHandler) GetPhoto(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	data, contentType, err := h.drivers.GetPhoto(c.Request.Context(), driverID)
	if err != nil {
		if errors.Is(err, drivers.ErrNotFound) {
			resp.ErrorLang(c, http.StatusNotFound, "photo_not_found")
			return
		}
		h.logger.Error("driver get photo failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	c.Data(http.StatusOK, contentType, data)
}

// DeletePhoto — DELETE фото водителя. Можно удалить когда угодно.
func (h *ProfileHandler) DeletePhoto(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	if err := h.drivers.DeletePhoto(c.Request.Context(), driverID); err != nil {
		h.logger.Error("driver delete photo failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "photo_deleted", gin.H{"status": "ok", "event": "photo_deleted"})
}

