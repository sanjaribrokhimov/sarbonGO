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
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	phone, err := util.NormalizeE164StrictPlus(req.Phone)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}

	ttlSec := int(h.otpTTL.Seconds())
	code, requestID, err := SendOTP(c.Request.Context(), h.tg, phone, ttlSec, h.otpLen)
	if err != nil {
		if WriteOTPSendError(c, err, h.logger, "telegram sendVerificationMessage failed") {
			return
		}
		resp.ErrorLang(c, http.StatusInternalServerError, "otp_generation_failed")
		return
	}

	ctx := c.Request.Context()
	ip := strings.TrimSpace(c.ClientIP())
	if err := h.otp.SaveOTP(ctx, dispOTPNamespace+phone, code, requestID, ip); err != nil {
		if errors.Is(err, store.ErrOTPCooldown) {
			resp.ErrorLang(c, http.StatusTooManyRequests, "otp_cooldown")
			return
		}
		if errors.Is(err, store.ErrOTPRateLimited) {
			resp.ErrorLang(c, http.StatusTooManyRequests, "otp_rate_limited")
			return
		}
		h.logger.Error("otp save failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}

	resp.OKLang(c, "otp_sent", gin.H{"status": "otp_sent", "ttl_seconds": ttlSec})
}

type dispVerifyOTPReq struct {
	Phone string `json:"phone" binding:"required"`
	OTP   string `json:"otp" binding:"required"`
}

func (h *DispatcherAuthHandler) VerifyOTP(c *gin.Context) {
	var req dispVerifyOTPReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	phone, err := util.NormalizeE164StrictPlus(req.Phone)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	otp := strings.TrimSpace(req.OTP)
	if len(otp) < 4 || len(otp) > 8 || !util.IsNumeric(otp) {
		resp.ErrorLang(c, http.StatusBadRequest, "otp_must_be_numeric")
		return
	}

	_, err = h.otp.Verify(c.Request.Context(), dispOTPNamespace+phone, otp)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrOTPExpired):
			resp.ErrorLang(c, http.StatusUnauthorized, "otp_expired")
		case errors.Is(err, store.ErrOTPInvalid):
			resp.ErrorLang(c, http.StatusUnauthorized, "otp_invalid")
		case errors.Is(err, store.ErrOTPMaxAttempts):
			resp.ErrorLang(c, http.StatusTooManyRequests, "otp_max_attempts_exceeded")
		case errors.Is(err, store.ErrOTPVerifyRateLimited):
			resp.ErrorLang(c, http.StatusTooManyRequests, "otp_verify_attempts_exceeded")
		default:
			h.logger.Error("dispatchers auth otp verify failed", zap.String("path", "dispatchers/auth/otp/verify"), zap.Error(err))
			resp.ErrorLang(c, http.StatusInternalServerError, "verification_failed")
		}
		return
	}

	d, err := h.repo.FindByPhone(c.Request.Context(), phone)
	if err == nil {
		id, _ := uuid.Parse(d.ID)
		tokens, refreshClaims, err := h.jwtm.Issue("dispatcher", id)
		if err != nil {
			h.logger.Error("dispatcher token issue failed", zap.Error(err))
			resp.ErrorLang(c, http.StatusInternalServerError, "token_issue_failed")
			return
		}
		_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
		_ = h.refresh.PutSession(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
		resp.OKLang(c, "login", gin.H{"status": "login", "tokens": tokens})
		return
	}
	if !errors.Is(err, dispatchers.ErrNotFound) {
		h.logger.Error("find dispatcher by phone failed", zap.String("path", "dispatchers/auth/otp/verify"), zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "verification_failed")
		return
	}

	sessionID, err := h.regSessions.Create(c.Request.Context(), phone)
	if err != nil {
		h.logger.Error("create dispatcher register session failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "register", gin.H{"status": "register", "session_id": sessionID})
}

type dispLoginPasswordReq struct {
	Phone    string `json:"phone" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *DispatcherAuthHandler) LoginPassword(c *gin.Context) {
	var req dispLoginPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	phone, err := util.NormalizeE164(req.Phone)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	d, err := h.repo.FindByPhone(c.Request.Context(), phone)
	if err != nil {
		resp.ErrorLang(c, http.StatusUnauthorized, "invalid_phone_or_password")
		return
	}
	if !util.ComparePassword(d.Password, req.Password) {
		resp.ErrorLang(c, http.StatusUnauthorized, "invalid_phone_or_password")
		return
	}
	id, _ := uuid.Parse(d.ID)
	tokens, refreshClaims, err := h.jwtm.Issue("dispatcher", id)
	if err != nil {
		resp.ErrorLang(c, http.StatusInternalServerError, "token_issue_failed")
		return
	}
	_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
	_ = h.refresh.PutSession(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
	resp.OKLang(c, "login", gin.H{"status": "login", "tokens": tokens})
}

type dispResetReq struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *DispatcherAuthHandler) ResetPasswordRequest(c *gin.Context) {
	var req dispResetReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	phone, err := util.NormalizeE164StrictPlus(req.Phone)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	d, err := h.repo.FindByPhone(c.Request.Context(), phone)
	if err != nil {
		resp.ErrorLang(c, http.StatusNotFound, "dispatcher_not_found")
		return
	}

	ttlSec := int(h.otpTTL.Seconds())
	code, _, err := SendOTP(c.Request.Context(), h.tg, phone, ttlSec, h.otpLen)
	if err != nil {
		if WriteOTPSendError(c, err, h.logger, "telegram sendVerificationMessage failed") {
			return
		}
		resp.ErrorLang(c, http.StatusInternalServerError, "otp_generation_failed")
		return
	}
	dispID, _ := uuid.Parse(d.ID)
	sessionID, err := h.resetActions.Create(c.Request.Context(), dispID, phone, "", code)
	if err != nil {
		h.logger.Error("reset password action create failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "otp_sent", gin.H{"status": "otp_sent", "session_id": sessionID, "ttl_seconds": ttlSec})
}

type dispResetConfirmReq struct {
	SessionID   string `json:"session_id" binding:"required"`
	OTP         string `json:"otp" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (h *DispatcherAuthHandler) ResetPasswordConfirm(c *gin.Context) {
	var req dispResetConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	if err := util.ValidatePassword(req.NewPassword); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}

	payload, err := h.resetActions.Verify(c.Request.Context(), strings.TrimSpace(req.SessionID), strings.TrimSpace(req.OTP))
	if err != nil {
		switch err {
		case store.ErrDispActionExpired:
			resp.ErrorLang(c, http.StatusUnauthorized, "session_expired_or_invalid")
		case store.ErrDispActionInvalidOTP:
			resp.ErrorLang(c, http.StatusUnauthorized, "otp_invalid")
		case store.ErrDispActionMaxAttempts:
			resp.ErrorLang(c, http.StatusTooManyRequests, "otp_max_attempts_exceeded")
		default:
			h.logger.Error("reset password verify failed", zap.Error(err))
			resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		}
		return
	}

	hash, err := util.HashPassword(req.NewPassword)
	if err != nil {
		resp.ErrorLang(c, http.StatusInternalServerError, "password_hash_failed")
		return
	}
	if err := h.repo.UpdatePasswordHash(c.Request.Context(), payload.DispatcherID, hash); err != nil {
		h.logger.Error("reset password update failed", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "internal_error")
		return
	}
	resp.OKLang(c, "ok", gin.H{"status": "ok"})
}

type dispLogoutReq struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

// Logout инвалидирует сессию. Тело: { "refresh_token": "..." } — отзыв одной сессии; { "access_token": "..." } — отзыв всех сессий диспетчера.
// Валидация: при невалидном/истёкшем токене возвращается 401.
func (h *DispatcherAuthHandler) Logout(c *gin.Context) {
	var req dispLogoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
		return
	}
	refreshToken := strings.TrimSpace(req.RefreshToken)
	accessToken := strings.TrimSpace(req.AccessToken)
	if refreshToken == "" && accessToken == "" {
		resp.ErrorLang(c, http.StatusBadRequest, "refresh_token_or_access_token_required")
		return
	}

	if refreshToken != "" {
		claims, err := h.jwtm.ParseRefresh(refreshToken)
		if err != nil {
			resp.ErrorLang(c, http.StatusUnauthorized, "invalid_or_expired_refresh_token")
			return
		}
		if claims.Role != "dispatcher" {
			resp.ErrorLang(c, http.StatusUnauthorized, "invalid_refresh_token_for_dispatcher")
			return
		}
		if err := h.refresh.Consume(c.Request.Context(), claims.UserID, claims.JTI); err != nil {
			resp.ErrorLang(c, http.StatusUnauthorized, "refresh_token_already_used")
			return
		}
		resp.OKLang(c, "ok", gin.H{"status": "ok"})
		return
	}

	// logout по access_token — отзываем все сессии этого диспетчера
	userID, role, err := h.jwtm.ParseAccess(accessToken)
	if err != nil {
		resp.ErrorLang(c, http.StatusUnauthorized, "invalid_or_expired_access_token")
		return
	}
	if role != "dispatcher" {
		resp.ErrorLang(c, http.StatusUnauthorized, "access_token_not_for_dispatcher")
		return
	}
	_ = h.refresh.RevokeAll(c.Request.Context(), userID.String())
	resp.OKLang(c, "ok", gin.H{"status": "ok"})
}
