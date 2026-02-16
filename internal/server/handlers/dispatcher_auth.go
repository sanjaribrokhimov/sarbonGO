package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/dispatchers"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
	"sarbonNew/internal/telegram"
	"sarbonNew/internal/util"
)

const dispOTPNamespace = "disp:"

type DispatcherAuthHandler struct {
	logger *zap.Logger

	repo         *dispatchers.Repo
	otp          *store.OTPStore
	regSessions  *store.DispatcherSessionStore
	resetActions *store.DispatcherOTPActionStore

	jwtm    *security.JWTManager
	refresh *store.RefreshStore
	tg      *telegram.GatewayClient

	otpTTL time.Duration
	otpLen int
}

func NewDispatcherAuthHandler(
	logger *zap.Logger,
	repo *dispatchers.Repo,
	otpStore *store.OTPStore,
	regSessions *store.DispatcherSessionStore,
	resetActions *store.DispatcherOTPActionStore,
	jwtm *security.JWTManager,
	refresh *store.RefreshStore,
	tg *telegram.GatewayClient,
	otpTTL time.Duration,
	otpLen int,
) *DispatcherAuthHandler {
	return &DispatcherAuthHandler{
		logger:       logger,
		repo:         repo,
		otp:          otpStore,
		regSessions:  regSessions,
		resetActions: resetActions,
		jwtm:         jwtm,
		refresh:      refresh,
		tg:           tg,
		otpTTL:       otpTTL,
		otpLen:       otpLen,
	}
}

type dispSendOTPReq struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *DispatcherAuthHandler) SendOTP(c *gin.Context) {
	var req dispSendOTPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	phone, err := util.NormalizeE164(req.Phone)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	code, err := util.GenerateNumericOTP(h.otpLen)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "otp generation failed")
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	requestID, err := h.tg.SendVerificationMessage(ctx, phone, code, int(h.otpTTL.Seconds()))
	if err != nil {
		h.logger.Warn("telegram sendVerificationMessage failed", zap.Error(err))
		resp.Error(c, http.StatusBadGateway, "otp send failed")
		return
	}

	ip := strings.TrimSpace(c.ClientIP())
	if err := h.otp.SaveOTP(ctx, dispOTPNamespace+phone, code, requestID, ip); err != nil {
		if errors.Is(err, store.ErrOTPCooldown) {
			resp.Error(c, http.StatusTooManyRequests, "otp cooldown")
			return
		}
		if errors.Is(err, store.ErrOTPRateLimited) {
			resp.Error(c, http.StatusTooManyRequests, "otp rate limited")
			return
		}
		h.logger.Error("otp save failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	resp.OK(c, gin.H{"status": "otp_sent", "ttl_seconds": int(h.otpTTL.Seconds())})
}

type dispVerifyOTPReq struct {
	Phone string `json:"phone" binding:"required"`
	OTP   string `json:"otp" binding:"required"`
}

func (h *DispatcherAuthHandler) VerifyOTP(c *gin.Context) {
	var req dispVerifyOTPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	phone, err := util.NormalizeE164(req.Phone)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	otp := strings.TrimSpace(req.OTP)
	if len(otp) < 4 || len(otp) > 8 || !util.IsNumeric(otp) {
		resp.Error(c, http.StatusBadRequest, "otp must be numeric 4..8 digits")
		return
	}

	_, err = h.otp.Verify(c.Request.Context(), dispOTPNamespace+phone, otp)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrOTPExpired):
			resp.Error(c, http.StatusUnauthorized, "otp expired")
		case errors.Is(err, store.ErrOTPInvalid):
			resp.Error(c, http.StatusUnauthorized, "otp invalid")
		case errors.Is(err, store.ErrOTPMaxAttempts):
			resp.Error(c, http.StatusTooManyRequests, "otp max attempts exceeded")
		default:
			h.logger.Error("otp verify error", zap.Error(err))
			resp.Error(c, http.StatusInternalServerError, "internal error")
		}
		return
	}

	d, err := h.repo.FindByPhone(c.Request.Context(), phone)
	if err == nil {
		id, _ := uuid.Parse(d.ID)
		tokens, refreshClaims, err := h.jwtm.Issue("dispatcher", id)
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, "token issue failed")
			return
		}
		_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
		resp.OK(c, gin.H{"status": "login", "tokens": tokens})
		return
	}
	if !errors.Is(err, dispatchers.ErrNotFound) {
		h.logger.Error("find dispatcher by phone failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	sessionID, err := h.regSessions.Create(c.Request.Context(), phone)
	if err != nil {
		h.logger.Error("create dispatcher register session failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "register", "session_id": sessionID})
}

type dispLoginPasswordReq struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *DispatcherAuthHandler) LoginPassword(c *gin.Context) {
	var req dispLoginPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	phone, err := util.NormalizeE164(req.Phone)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	d, err := h.repo.FindByPhone(c.Request.Context(), phone)
	if err != nil {
		resp.Error(c, http.StatusUnauthorized, "invalid phone or password")
		return
	}
	if !util.ComparePassword(d.Password, req.Password) {
		resp.Error(c, http.StatusUnauthorized, "invalid phone or password")
		return
	}
	id, _ := uuid.Parse(d.ID)
	tokens, refreshClaims, err := h.jwtm.Issue("dispatcher", id)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "token issue failed")
		return
	}
	_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
	resp.OK(c, gin.H{"status": "login", "tokens": tokens})
}

type dispResetReq struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *DispatcherAuthHandler) ResetPasswordRequest(c *gin.Context) {
	var req dispResetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	phone, err := util.NormalizeE164(req.Phone)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	d, err := h.repo.FindByPhone(c.Request.Context(), phone)
	if err != nil {
		resp.Error(c, http.StatusNotFound, "dispatcher not found")
		return
	}

	code, err := util.GenerateNumericOTP(h.otpLen)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "otp generation failed")
		return
	}
	if _, err := h.tg.SendVerificationMessage(c.Request.Context(), phone, code, int(h.otpTTL.Seconds())); err != nil {
		h.logger.Warn("telegram sendVerificationMessage failed", zap.Error(err))
		resp.Error(c, http.StatusBadGateway, "otp send failed")
		return
	}
	dispID, _ := uuid.Parse(d.ID)
	sessionID, err := h.resetActions.Create(c.Request.Context(), dispID, phone, "", code)
	if err != nil {
		h.logger.Error("reset password action create failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "otp_sent", "session_id": sessionID, "ttl_seconds": int(h.otpTTL.Seconds())})
}

type dispResetConfirmReq struct {
	SessionID   string `json:"session_id" binding:"required"`
	OTP         string `json:"otp" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (h *DispatcherAuthHandler) ResetPasswordConfirm(c *gin.Context) {
	var req dispResetConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := util.ValidatePassword(req.NewPassword); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	payload, err := h.resetActions.Verify(c.Request.Context(), strings.TrimSpace(req.SessionID), strings.TrimSpace(req.OTP))
	if err != nil {
		switch err {
		case store.ErrDispActionExpired:
			resp.Error(c, http.StatusUnauthorized, "session expired or invalid")
		case store.ErrDispActionInvalidOTP:
			resp.Error(c, http.StatusUnauthorized, "otp invalid")
		case store.ErrDispActionMaxAttempts:
			resp.Error(c, http.StatusTooManyRequests, "otp max attempts exceeded")
		default:
			h.logger.Error("reset password verify failed", zap.Error(err))
			resp.Error(c, http.StatusInternalServerError, "internal error")
		}
		return
	}

	hash, err := util.HashPassword(req.NewPassword)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "password hash failed")
		return
	}
	if err := h.repo.UpdatePasswordHash(c.Request.Context(), payload.DispatcherID, hash); err != nil {
		h.logger.Error("reset password update failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "ok"})
}
