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
		resp.Error(c, http.StatusUnauthorized, "dispatcher not found")
		return
	}
	resp.OK(c, gin.H{"dispatcher": d})
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
		resp.Error(c, http.StatusBadRequest, "invalid payload")
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
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	d, _ := h.repo.FindByID(c.Request.Context(), id)
	resp.OK(c, gin.H{"status": "ok", "dispatcher": d})
}

type dispChangePasswordReq struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (h *DispatcherProfileHandler) ChangePassword(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	var req dispChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := util.ValidatePassword(req.NewPassword); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	d, err := h.repo.FindByID(c.Request.Context(), id)
	if err != nil || !util.ComparePassword(d.Password, req.CurrentPassword) {
		resp.Error(c, http.StatusUnauthorized, "invalid current_password")
		return
	}
	hash, err := util.HashPassword(req.NewPassword)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "password hash failed")
		return
	}
	if err := h.repo.UpdatePasswordHash(c.Request.Context(), id, hash); err != nil {
		h.logger.Error("dispatcher password update failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "ok"})
}

type dispPhoneChangeReq struct {
	NewPhone string `json:"new_phone" binding:"required"`
}

func (h *DispatcherProfileHandler) PhoneChangeRequest(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	var req dispPhoneChangeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	newPhone, err := util.ValidateUzPhoneStrict(req.NewPhone)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if _, err := h.repo.FindByPhone(c.Request.Context(), newPhone); err == nil {
		resp.Error(c, http.StatusConflict, "this phone is already registered")
		return
	}

	ttlSec := int(h.otpTTL.Seconds())
	code, _, err := SendOTP(c.Request.Context(), h.tg, newPhone, ttlSec, h.otpLen)
	if err != nil {
		if WriteOTPSendError(c, err, h.logger, "telegram sendVerificationMessage failed") {
			return
		}
		resp.Error(c, http.StatusInternalServerError, "otp generation failed")
		return
	}
	sessionID, err := h.phoneChange.Create(c.Request.Context(), id, "", newPhone, code)
	if err != nil {
		h.logger.Error("dispatcher phone change action create failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "otp_sent", "session_id": sessionID, "ttl_seconds": ttlSec})
}

type dispPhoneChangeVerifyReq struct {
	SessionID string `json:"session_id" binding:"required"`
	OTP       string `json:"otp" binding:"required"`
}

func (h *DispatcherProfileHandler) PhoneChangeVerify(c *gin.Context) {
	var req dispPhoneChangeVerifyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	payload, err := h.phoneChange.Verify(c.Request.Context(), strings.TrimSpace(req.SessionID), strings.TrimSpace(req.OTP))
	if err != nil {
		switch err {
		case store.ErrDispActionExpired:
			resp.Error(c, http.StatusUnauthorized, "session expired or invalid")
		case store.ErrDispActionInvalidOTP:
			resp.Error(c, http.StatusUnauthorized, "otp invalid")
		case store.ErrDispActionMaxAttempts:
			resp.Error(c, http.StatusTooManyRequests, "otp max attempts exceeded")
		default:
			h.logger.Error("dispatcher phone change verify failed", zap.Error(err))
			resp.Error(c, http.StatusInternalServerError, "internal error")
		}
		return
	}

	if payload.NewPhone == "" {
		resp.Error(c, http.StatusUnauthorized, "session expired or invalid")
		return
	}

	if err := h.repo.UpdatePhone(c.Request.Context(), payload.DispatcherID, payload.NewPhone); err != nil {
		if errors.Is(err, dispatchers.ErrPhoneAlreadyRegistered) {
			resp.Error(c, http.StatusConflict, "this phone is already registered")
			return
		}
		h.logger.Error("dispatcher phone update failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	d, _ := h.repo.FindByID(c.Request.Context(), payload.DispatcherID)
	resp.OK(c, gin.H{"status": "ok", "dispatcher": d})
}

// UploadPhoto — POST multipart/form-data с полем "photo" (файл). Фото сохраняется в БД.
func (h *DispatcherProfileHandler) UploadPhoto(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	file, err := c.FormFile("photo")
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "missing or invalid file: use form field 'photo'")
		return
	}
	if file.Size > maxDispatcherPhotoSize {
		resp.Error(c, http.StatusBadRequest, "file too large (max 5 MB)")
		return
	}
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	if !allowedPhotoTypes[contentType] {
		resp.Error(c, http.StatusBadRequest, "allowed types: image/jpeg, image/png")
		return
	}
	f, err := file.Open()
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "cannot read file")
		return
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "cannot read file")
		return
	}
	if err := h.repo.UpdatePhoto(c.Request.Context(), id, data, contentType); err != nil {
		h.logger.Error("dispatcher photo update failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "ok", "event": "photo_uploaded"})
}

// GetPhoto — GET фото диспетчера (бинарный ответ с Content-Type). 404 если фото нет.
func (h *DispatcherProfileHandler) GetPhoto(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	data, contentType, err := h.repo.GetPhoto(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, dispatchers.ErrNotFound) {
			resp.Error(c, http.StatusNotFound, "photo not found")
			return
		}
		h.logger.Error("dispatcher get photo failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	c.Data(http.StatusOK, contentType, data)
}

func (h *DispatcherProfileHandler) Delete(c *gin.Context) {
	id := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	if err := h.repo.DeleteAndArchive(c.Request.Context(), id); err != nil {
		if errors.Is(err, dispatchers.ErrDeleteNotFound) {
			resp.Error(c, http.StatusNotFound, "dispatcher not found")
			return
		}
		h.logger.Error("dispatcher delete failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "ok"})
}
