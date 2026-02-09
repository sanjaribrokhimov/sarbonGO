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

	"sarbonNew/internal/drivers"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
	"sarbonNew/internal/telegram"
	"sarbonNew/internal/util"
)

type AuthHandler struct {
	logger *zap.Logger

	drivers *drivers.Repo
	otp     *store.OTPStore
	sessions *store.SessionStore
	refresh *store.RefreshStore
	jwtm    *security.JWTManager
	tg      *telegram.GatewayClient

	otpTTL time.Duration
	otpLen int
}

func NewAuthHandler(
	logger *zap.Logger,
	driversRepo *drivers.Repo,
	otpStore *store.OTPStore,
	sessionStore *store.SessionStore,
	refreshStore *store.RefreshStore,
	jwtm *security.JWTManager,
	tg *telegram.GatewayClient,
	otpTTL time.Duration,
	otpLen int,
) *AuthHandler {
	return &AuthHandler{
		logger:  logger,
		drivers: driversRepo,
		otp:     otpStore,
		sessions: sessionStore,
		refresh: refreshStore,
		jwtm:    jwtm,
		tg:      tg,
		otpTTL:  otpTTL,
		otpLen:  otpLen,
	}
}

type sendOTPReq struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *AuthHandler) SendOTP(c *gin.Context) {
	var req sendOTPReq
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
	if err := h.otp.SaveOTP(ctx, phone, code, requestID, ip); err != nil {
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

	resp.OK(c, gin.H{
		"event":       "otp_sent",
		"ttl_seconds": int(h.otpTTL.Seconds()),
	})
}

type verifyOTPReq struct {
	Phone string `json:"phone" binding:"required"`
	OTP   string `json:"otp" binding:"required"`
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req verifyOTPReq
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

	rec, err := h.otp.Verify(c.Request.Context(), phone, otp)
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
	_ = rec // request_id can be used for optional Telegram checkVerificationStatus

	d, err := h.drivers.FindByPhone(c.Request.Context(), phone)
	if err == nil {
		driverUUID, _ := uuid.Parse(d.ID)
		tokens, refreshClaims, err := h.jwtm.Issue(driverUUID)
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, "token issue failed")
			return
		}
		_ = h.refresh.Put(c.Request.Context(), refreshClaims.DriverID, refreshClaims.JTI)

		resp.OK(c, gin.H{
			"event":  "login",
			"tokens": tokens,
		})
		return
	}
	if !errors.Is(err, drivers.ErrNotFound) {
		h.logger.Error("find driver by phone failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	sessionID, err := h.sessions.Create(c.Request.Context(), phone)
	if err != nil {
		h.logger.Error("create register session failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	resp.OK(c, gin.H{
		"event":      "register",
		"session_id": sessionID,
	})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	claims, err := h.jwtm.ParseRefresh(strings.TrimSpace(req.RefreshToken))
	if err != nil {
		resp.Error(c, http.StatusUnauthorized, "invalid refresh_token")
		return
	}
	// rotate: old jti must exist
	if err := h.refresh.Consume(c.Request.Context(), claims.DriverID, claims.JTI); err != nil {
		resp.Error(c, http.StatusUnauthorized, "invalid refresh_token")
		return
	}

	driverUUID, err := uuid.Parse(claims.DriverID)
	if err != nil {
		resp.Error(c, http.StatusUnauthorized, "invalid refresh_token")
		return
	}
	tokens, newRefreshClaims, err := h.jwtm.Issue(driverUUID)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "token issue failed")
		return
	}
	_ = h.refresh.Put(c.Request.Context(), newRefreshClaims.DriverID, newRefreshClaims.JTI)

	resp.OK(c, gin.H{"tokens": tokens})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	claims, err := h.jwtm.ParseRefresh(strings.TrimSpace(req.RefreshToken))
	if err != nil {
		resp.OK(c, gin.H{"event": "logout"})
		return
	}
	_ = h.refresh.Consume(c.Request.Context(), claims.DriverID, claims.JTI)
	resp.OK(c, gin.H{"event": "logout"})
}

