package handlers

import (
	"errors"
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
	sessionID, err := h.phoneChange.Create(c.Request.Context(), id, "", newPhone, code)
	if err != nil {
		h.logger.Error("dispatcher phone change action create failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "otp_sent", "session_id": sessionID, "ttl_seconds": int(h.otpTTL.Seconds())})
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
