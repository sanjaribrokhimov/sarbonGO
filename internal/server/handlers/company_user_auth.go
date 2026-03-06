package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/appusers"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
	"sarbonNew/internal/telegram"
	"sarbonNew/internal/util"
)

type CompanyUserAuthHandler struct {
	logger       *zap.Logger
	repo         *appusers.Repo
	otp          *store.OTPStore
	regSessions  *store.DispatcherSessionStore
	jwtm         *security.JWTManager
	refresh      *store.RefreshStore
	tg           *telegram.GatewayClient
	otpTTL       time.Duration
	otpLen       int
}

func NewCompanyUserAuthHandler(
	logger *zap.Logger,
	repo *appusers.Repo,
	otpStore *store.OTPStore,
	regSessions *store.DispatcherSessionStore,
	jwtm *security.JWTManager,
	refresh *store.RefreshStore,
	tg *telegram.GatewayClient,
	otpTTL time.Duration,
	otpLen int,
) *CompanyUserAuthHandler {
	return &CompanyUserAuthHandler{
		logger:      logger,
		repo:        repo,
		otp:         otpStore,
		regSessions: regSessions,
		jwtm:        jwtm,
		refresh:     refresh,
		tg:          tg,
		otpTTL:      otpTTL,
		otpLen:      otpLen,
	}
}

type companyUserSendOTPReq struct {
	Phone string `json:"phone" binding:"required"`
}

func (h *CompanyUserAuthHandler) SendOTP(c *gin.Context) {
	var req companyUserSendOTPReq
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
		if WriteOTPSendError(c, err, h.logger, "company user otp send failed") {
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
		h.logger.Error("company user otp save failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "otp_sent", "ttl_seconds": ttlSec})
}

type companyUserVerifyOTPReq struct {
	Phone string `json:"phone" binding:"required"`
	OTP   string `json:"otp" binding:"required"`
}

func (h *CompanyUserAuthHandler) VerifyOTP(c *gin.Context) {
	var req companyUserVerifyOTPReq
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
	_, err = h.otp.Verify(c.Request.Context(), phone, otp)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrOTPExpired):
			resp.Error(c, http.StatusUnauthorized, "otp expired")
		case errors.Is(err, store.ErrOTPInvalid):
			resp.Error(c, http.StatusUnauthorized, "otp invalid")
		case errors.Is(err, store.ErrOTPMaxAttempts):
			resp.Error(c, http.StatusTooManyRequests, "otp max attempts exceeded")
		default:
			h.logger.Error("company user otp verify failed", zap.Error(err))
			resp.Error(c, http.StatusInternalServerError, "verification failed")
		}
		return
	}
	u, err := h.repo.FindByPhone(c.Request.Context(), phone)
	if err == nil {
		id, _ := uuid.Parse(u.ID)
		tokens, refreshClaims, err := h.jwtm.IssueWithCompany("user", id, uuid.Nil)
		if err != nil {
			h.logger.Error("company user token issue failed", zap.Error(err))
			resp.Error(c, http.StatusInternalServerError, "token issue failed")
			return
		}
		_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
		resp.OK(c, gin.H{"status": "login", "tokens": tokens, "user": userToMap(u)})
		return
	}
	if !errors.Is(err, appusers.ErrNotFound) {
		h.logger.Error("company user find by phone failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "verification failed")
		return
	}
	sessionID, err := h.regSessions.Create(c.Request.Context(), phone)
	if err != nil {
		h.logger.Error("company user reg session create failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	resp.OK(c, gin.H{"status": "register", "session_id": sessionID})
}

func userToMap(u *appusers.User) map[string]interface{} {
	m := map[string]interface{}{
		"id":         u.ID,
		"phone":      u.Phone,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
		"role":       u.Role,
		"company_id": u.CompanyID,
	}
	if u.CompanyID != nil {
		m["company_id"] = u.CompanyID.String()
	}
	return m
}
