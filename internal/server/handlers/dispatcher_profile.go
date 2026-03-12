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

	"sarbonNew/internal/dispatchers"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
	"sarbonNew/internal/telegram"
	"sarbonNew/internal/util"
)

const maxDispatcherPhotoSize = 5 * 1024 * 1024 // 5 MB
var allowedPhotoTypes = map[string]bool{"image/jpeg": true, "image/png": true}

type DispatcherProfileHandler struct {
	logger      *zap.Logger
	repo        *dispatchers.Repo
	phoneChange *store.DispatcherOTPActionStore
	tg          *telegram.GatewayClient
	otpTTL      time.Duration
	otpLen      int
}

func NewDispatcherProfileHandler(logger *zap.Logger, repo *dispatchers.Repo, phoneChange *store.DispatcherOTPActionStore, tg *telegram.GatewayClient, otpTTL time.Duration, otpLen int) *DispatcherProfileHandler {
	return &DispatcherProfileHandler{logger: logger, repo: repo, phoneChange: phoneChange, tg: tg, otpTTL: otpTTL, otpLen: otpLen}
}

func (h *DispatcherProfileHandler) Get(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	d, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		resp.ErrorLang(c, http.StatusUnauthorized, "dispatcher_not_found")
		return
	}
	resp.OKLang(c, "ok", gin.H{"dispatcher": d})
}

type dispPatchReq struct {
	Name           *string `json:"name,omitempty"`
	PassportSeries *string `json:"passport_series,omitempty"`
	PassportNumber *string `json:"passport_number,omitempty"`
	PINFL          *string `json:"pinfl,omitempty"`
	Photo          *string `json:"photo,omitempty"`
}

func (h *DispatcherProfileHandler) Patch(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	var req dispPatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	trim := func(p **string) {
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
	trim(&req.Name)
	trim(&req.PassportSeries)
	trim(&req.PassportNumber)
	trim(&req.PINFL)
	trim(&req.Photo)

	if err := h.repo.UpdateProfile(c.Request.Context(), id, dispatchers.UpdateProfileParams{
		Name: req.Name, PassportSeries: req.PassportSeries, PassportNumber: req.PassportNumber, PINFL: req.PINFL, Photo: req.Photo,
	}); err != nil {
		h.logger.Error("dispatcher profile update failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	d, _ := h.repo.FindByID(c.Request.Context(), id)
	resp.OKLang(c, "ok", gin.H{"status": "ok", "dispatcher": d})
}

type dispChangePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (h *DispatcherProfileHandler) ChangePassword(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	var req dispChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	if err := util.ValidatePassword(req.NewPassword); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	d, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil || !util.ComparePassword(d.Password, req.CurrentPassword) {
		resp.ErrorLang(c, http.StatusUnauthorized, "invalid_current_password")
		return
	}
	hash, err := util.HashPassword(req.NewPassword)
	if err != nil {
		resp.ErrorLang(c, http.StatusInternalServerError, "password_hash_failed")
		return
	}
	if err := h.repo.UpdatePasswordHash(c.Request.Context(), id, hash); err != nil {
		h.logger.Error("dispatcher password update failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "ok", gin.H{"status": "ok"})
}

type dispPhoneChangeReq struct {
	NewPhone string `json:"new_phone" binding:"required"`
}

func (h *DispatcherProfileHandler) PhoneChangeRequest(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	var req dispPhoneChangeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	newPhone, err := util.ValidateUzPhoneStrict(req.NewPhone)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	if _, err := h.repo.FindByPhone(c.Request.Context(), newPhone); err == nil {
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
	sessionID, err := h.phoneChange.Create(c.Request.Context(), id, "", newPhone, code)
	if err != nil {
		h.logger.Error("dispatcher phone change action create failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "otp_sent", gin.H{"status": "otp_sent", "session_id": sessionID, "ttl_seconds": ttlSec})
}

type dispPhoneChangeVerifyReq struct {
	SessionID string `json:"session_id" binding:"required"`
	OTP       string `json:"otp" binding:"required"`
}

func (h *DispatcherProfileHandler) PhoneChangeVerify(c *gin.Context) {
	var req dispPhoneChangeVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	payload, err := h.phoneChange.Verify(c.Request.Context(), strings.TrimSpace(req.SessionID), strings.TrimSpace(req.OTP))
	if err != nil {
		switch err {
		case store.ErrDispActionExpired:
			resp.ErrorLang(c, http.StatusUnauthorized, "session_expired_or_invalid")
		case store.ErrDispActionInvalidOTP:
			resp.ErrorLang(c, http.StatusUnauthorized, "otp_invalid")
		case store.ErrDispActionMaxAttempts:
			resp.ErrorLang(c, http.StatusTooManyRequests, "otp_max_attempts_exceeded")
		default:
			h.logger.Error("dispatcher phone change verify failed", zap.Error(err))
			resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	if payload.NewPhone == "" {
		resp.ErrorLang(c, http.StatusUnauthorized, "session_expired_or_invalid")
		return
	}

	if err := h.repo.UpdatePhone(c.Request.Context(), payload.DispatcherID, payload.NewPhone); err != nil {
		if errors.Is(err, dispatchers.ErrPhoneAlreadyRegistered) {
			resp.ErrorLang(c, http.StatusConflict, "phone_already_registered")
			return
		}
		h.logger.Error("dispatcher phone update failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	d, _ := h.repo.FindByID(c.Request.Context(), payload.DispatcherID)
	resp.OKLang(c, "ok", gin.H{"status": "ok", "dispatcher": d})
}

// UploadPhoto — POST multipart/form-data с полем "photo" (файл). Фото сохраняется в БД.
func (h *DispatcherProfileHandler) UploadPhoto(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	file, err := c.FormFile("photo")
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "photo_file_required")
		return
	}
	if file.Size > maxDispatcherPhotoSize {
		resp.ErrorLang(c, http.StatusBadRequest, "file_too_large")
		return
	}
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	if !allowedPhotoTypes[contentType] {
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
	if err := h.repo.UpdatePhoto(c.Request.Context(), id, data, contentType); err != nil {
		h.logger.Error("dispatcher photo update failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "photo_uploaded", gin.H{"status": "ok", "event": "photo_uploaded"})
}

// GetPhoto — GET фото диспетчера (бинарный ответ с Content-Type). 404 если фото нет.
func (h *DispatcherProfileHandler) GetPhoto(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	data, contentType, err := h.repo.GetPhoto(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, dispatchers.ErrNotFound) {
			resp.ErrorLang(c, http.StatusNotFound, "photo_not_found")
			return
		}
		h.logger.Error("dispatcher get photo failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	c.Data(http.StatusOK, contentType, data)
}

func (h *DispatcherProfileHandler) Delete(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	if err := h.repo.DeleteAndArchive(c.Request.Context(), id); err != nil {
		if errors.Is(err, dispatchers.ErrDeleteNotFound) {
			resp.ErrorLang(c, http.StatusNotFound, "dispatcher_not_found")
			return
		}
		h.logger.Error("dispatcher delete failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "ok", gin.H{"status": "ok"})
}
