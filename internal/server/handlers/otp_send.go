package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/telegram"
	"sarbonNew/internal/util"
)

const otpSendTimeout = 10 * time.Second

// SendOTP generates a code, sends it via Telegram Gateway, and returns (code, requestID, ttlSeconds, err).
// If err != nil, the caller should use WriteOTPSendError and return.
func SendOTP(ctx context.Context, tg *telegram.GatewayClient, phone string, ttlSeconds int, otpLen int) (code, requestID string, err error) {
	code, err = util.GenerateNumericOTP(otpLen)
	if err != nil {
		return "", "", err
	}
	reqCtx, cancel := context.WithTimeout(ctx, otpSendTimeout)
	defer cancel()
	requestID, err = tg.SendVerificationMessage(reqCtx, phone, code, ttlSeconds)
	if err != nil {
		return code, "", err
	}
	return code, requestID, nil
}

// WriteOTPSendError writes the appropriate HTTP response for a SendOTP error and returns true.
// Returns false if err == nil (caller should continue).
func WriteOTPSendError(c *gin.Context, err error, logger *zap.Logger, logMsg string) bool {
	if err == nil {
		return false
	}
	var tgErr *telegram.GatewayError
	if errors.As(err, &tgErr) {
		if errors.Is(err, telegram.ErrNoAccount) {
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload")
			return true
		}
		if errors.Is(err, telegram.ErrRateLimited) {
			resp.ErrorLang(c, http.StatusTooManyRequests, "otp_rate_limited")
			return true
		}
		logger.Warn(logMsg, zap.Error(err))
		resp.ErrorLang(c, http.StatusBadGateway, "internal_error")
		return true
	}
	logger.Warn(logMsg, zap.Error(err))
	resp.ErrorLang(c, http.StatusBadGateway, "internal_error")
	return true
}
