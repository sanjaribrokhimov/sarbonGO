package handlers

import (
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
	phone, err := util.NormalizeE164StrictPlus(req.Phone)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	ttlSec := int(h.otpTTL.Seconds())
	code, requestID, err := SendOTP(c.Request.Context(), h.tg, phone, ttlSec, h.otpLen)
	if err != nil {
		if WriteOTPSendError(c, err, h.logger, "telegram sendVerificationMessage failed") {
			return
		}
		resp.Error(c, http.StatusInternalServerError, "otp generation failed")
		return
	}

	ctx := c.Request.Context()
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

	resp.OK(c, gin.H{"status": "otp_sent", "ttl_seconds": ttlSec})
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
		case errors.Is(err, store.ErrOTPVerifyRateLimited):
			resp.Error(c, http.StatusTooManyRequests, "otp verify attempts exceeded for this phone, try again later")
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
		tokens, refreshClaims, err := h.jwtm.Issue("driver", driverUUID)
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, "token issue failed")
			return
		}
		_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
		_ = h.refresh.PutSession(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)

		resp.OK(c, gin.H{
			"status": "login",
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
		"status":     "register",
		"session_id": sessionID,
	})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

// logoutReq — для logout принимаем refresh_token или access_token (хотя бы один).
type logoutReq struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	refreshToken := strings.TrimSpace(req.RefreshToken)
	if refreshToken == "" {
		resp.Error(c, http.StatusBadRequest, "refresh_token is required")
		return
	}
	// Проверка формата JWT (три части, разделённые точками)
	if !isJWTFormat(refreshToken) {
		resp.Error(c, http.StatusUnauthorized, "invalid refresh_token")
		return
	}
	claims, err := h.jwtm.ParseRefresh(refreshToken)
	if err != nil {
		resp.Error(c, http.StatusUnauthorized, "invalid refresh_token")
		return
	}
	// rotate: old jti must exist
	if err := h.refresh.Consume(c.Request.Context(), claims.UserID, claims.JTI); err != nil {
		resp.Error(c, http.StatusUnauthorized, "invalid refresh_token")
		return
	}
	// Старый access-токен (с этим sid) больше не действителен
	_ = h.refresh.InvalidateSession(c.Request.Context(), claims.JTI)

	userUUID, err := uuid.Parse(claims.UserID)
	if err != nil {
		resp.Error(c, http.StatusUnauthorized, "invalid refresh_token")
		return
	}
	tokens, newRefreshClaims, err := h.jwtm.Issue(claims.Role, userUUID)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "token issue failed")
		return
	}
	_ = h.refresh.Put(c.Request.Context(), newRefreshClaims.UserID, newRefreshClaims.JTI)
	_ = h.refresh.PutSession(c.Request.Context(), newRefreshClaims.UserID, newRefreshClaims.JTI)

	resp.OK(c, gin.H{"tokens": tokens})
}

// Logout инвалидирует сессию. Тело: { "refresh_token": "..." } — отзыв одной сессии; { "access_token": "..." } — отзыв всех сессий водителя.
// Валидация: при невалидном/истёкшем токене возвращается 401.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	refreshToken := strings.TrimSpace(req.RefreshToken)
	accessToken := strings.TrimSpace(req.AccessToken)
	if refreshToken == "" && accessToken == "" {
		resp.Error(c, http.StatusBadRequest, "refresh_token or access_token required")
		return
	}

	if refreshToken != "" {
		claims, err := h.jwtm.ParseRefresh(refreshToken)
		if err != nil {
			resp.Error(c, http.StatusUnauthorized, "invalid or expired refresh_token")
			return
		}
		if claims.Role != "driver" {
			resp.Error(c, http.StatusUnauthorized, "invalid refresh_token for driver")
			return
		}
		if err := h.refresh.Consume(c.Request.Context(), claims.UserID, claims.JTI); err != nil {
			resp.Error(c, http.StatusUnauthorized, "refresh_token already used or invalid")
			return
		}
		resp.OK(c, gin.H{"status": "ok"})
		return
	}

	userID, role, err := h.jwtm.ParseAccess(accessToken)
	if err != nil {
		resp.Error(c, http.StatusUnauthorized, "invalid or expired access_token")
		return
	}
	if role != "driver" {
		resp.Error(c, http.StatusUnauthorized, "access_token is not for driver")
		return
	}
	_ = h.refresh.RevokeAll(c.Request.Context(), userID.String())
	resp.OK(c, gin.H{"status": "ok"})
}

// isJWTFormat проверяет, что строка похожа на JWT (три части через точку).
func isJWTFormat(s string) bool {
	const jwtParts = 3
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			n++
		}
	}
	return n == jwtParts-1
}


