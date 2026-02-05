// Auth Driver: OTP send → verify (session_id или токены) → complete-register по session_id.
package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/sarbonGO/backend/internal/auth"
	"github.com/sarbonGO/backend/internal/config"
	"github.com/sarbonGO/backend/internal/middleware"
	"github.com/sarbonGO/backend/internal/response"
)

// --- POST /auth/otp/send ---

type SendOTPRequestV2 struct {
	Phone string `json:"phone" binding:"required"`
}

func SendOTPV2(pool *pgxpool.Pool, rdb *redis.Client, cfg config.Security) gin.HandlerFunc {
	ttlSec := cfg.OTPTTLSec
	if ttlSec < 180 {
		ttlSec = 180
	}
	if ttlSec > 300 {
		ttlSec = 300
	}
	rateLimit := cfg.OTPRateLimitPerPhone
	if rateLimit <= 0 {
		rateLimit = 3
	}
	const rateWindow = 15 * time.Minute
	return func(c *gin.Context) {
		var req SendOTPRequestV2
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "phone is required")
			return
		}
		phone := auth.NormalizePhone(req.Phone)
		if phone == "" {
			response.Error(c, http.StatusBadRequest, "invalid phone")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		if rdb != nil {
			key := "otp_send:" + phone
			n, err := rdb.Incr(ctx, key).Result()
			if err == nil {
				if n == 1 {
					rdb.Expire(ctx, key, rateWindow)
				}
				if n > int64(rateLimit) {
					response.Error(c, http.StatusTooManyRequests, "too many OTP requests for this phone")
					return
				}
			}
		}

		code, err := auth.CreateOTP(ctx, pool, phone, ttlSec)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		if cfg.TelegramGatewayToken != "" {
			_ = auth.SendOTPViaGateway(cfg.TelegramGatewayToken, phone, code, ttlSec)
		}

		response.Success(c, http.StatusOK, "OTP sent", gin.H{"message": "If the number is registered in Telegram, you will receive a code."})
	}
}

// --- POST /auth/otp/verify ---

type VerifyOTPRequestV2 struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// VerifyOTPResponseV2 — при наличии водителя: токены + message "login"; при отсутствии: session_id + phone + message "register".
type VerifyOTPResponseV2 struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
	IsNew        bool   `json:"is_new"`
	Message      string `json:"message"` // "login" или "register"
	SessionID    string `json:"session_id,omitempty"`
	Phone        string `json:"phone,omitempty"`
}

func VerifyOTPV2(pool *pgxpool.Pool, cfg config.Security) gin.HandlerFunc {
	maxAttempts := cfg.OTPAttemptsMax
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	return func(c *gin.Context) {
		var req VerifyOTPRequestV2
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "phone and code are required")
			return
		}
		phone := auth.NormalizePhone(req.Phone)
		if phone == "" {
			response.Error(c, http.StatusBadRequest, "invalid phone")
			return
		}
		code := strings.TrimSpace(req.Code)
		if code == "" {
			response.Error(c, http.StatusBadRequest, "code is required")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		ok, err := auth.ValidateAndConsumeOTP(ctx, pool, phone, code, maxAttempts)
		if err != nil {
			log.Printf("[auth] verify OTP: %v", err)
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		if !ok {
			response.Error(c, http.StatusBadRequest, "invalid or expired code")
			return
		}

		driver, err := auth.GetDriverByPhone(ctx, pool, phone)
		if err != nil {
			log.Printf("[auth] verify OTP GetDriverByPhone: %v", err)
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}

		if driver != nil && driver.AccountStatus == AccountStatusBlocked {
			response.Error(c, http.StatusForbidden, "account blocked")
			return
		}

		if driver != nil {
			tp, err := auth.CreateTokenPair(ctx, pool, driver.ID, cfg.JWTSecret)
			if err != nil {
				log.Printf("[auth] verify OTP CreateTokenPair: %v", err)
				response.Error(c, http.StatusInternalServerError, "internal error")
				return
			}
			response.Success(c, http.StatusOK, "success", VerifyOTPResponseV2{
				AccessToken:  tp.AccessToken,
				RefreshToken: tp.RefreshToken,
				ExpiresIn:    tp.ExpiresIn,
				IsNew:        false,
				Message:      "login",
			})
			return
		}

		sessionID, err := auth.CreateRegistrationSession(ctx, pool, phone)
		if err != nil {
			log.Printf("[auth] verify OTP CreateRegistrationSession: %v", err)
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		response.Success(c, http.StatusOK, "success", VerifyOTPResponseV2{
			IsNew:     true,
			Message:   "register",
			SessionID: sessionID,
			Phone:     phone,
		})
	}
}

// --- POST /auth/complete-register (multipart: session_id + поля + car_photo, adr_document) ---

func CompleteRegister(pool *pgxpool.Pool, storageRoot string, cfg config.Security) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := strings.TrimSpace(c.PostForm("session_id"))
		if sessionID == "" {
			response.Error(c, http.StatusBadRequest, "session_id is required")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()

		phone, ok, err := auth.GetRegistrationSessionByID(ctx, pool, sessionID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		if !ok {
			response.Error(c, http.StatusBadRequest, "invalid or expired session")
			return
		}

		firstName := strings.TrimSpace(c.PostForm("first_name"))
		lastName := strings.TrimSpace(c.PostForm("last_name"))
		passportSeries := strings.TrimSpace(c.PostForm("passport_series"))
		passportNumber := strings.TrimSpace(c.PostForm("passport_number"))
		language := strings.TrimSpace(c.PostForm("language"))
		if language == "" {
			language = middleware.LanguageFrom(c.Request.Context())
			if language == "" || !validLanguage[language] {
				language = LangRU
			}
		}
		if !validLanguage[language] {
			response.Error(c, http.StatusBadRequest, "language must be ru, uz, en, tr or zh")
			return
		}
		platform := middleware.PlatformFrom(c.Request.Context())

		if firstName == "" || lastName == "" || passportSeries == "" || passportNumber == "" {
			response.Error(c, http.StatusBadRequest, "first_name, last_name, passport_series, passport_number are required")
			return
		}

		carFile, _ := c.FormFile("car_photo")
		adrFile, _ := c.FormFile("adr_document")
		if carFile == nil || carFile.Size == 0 {
			response.Error(c, http.StatusBadRequest, "car_photo is required (image file)")
			return
		}
		if adrFile == nil || adrFile.Size == 0 {
			response.Error(c, http.StatusBadRequest, "adr_document is required (PDF or image)")
			return
		}

		// Валидация файлов ПЕРЕД добавлением в БД
		if err := ValidateDriverFile(adrFile, adrMaxSize, adrAllowed); err != nil {
			if errors.Is(err, errFileTooLarge) {
				response.Error(c, http.StatusBadRequest, "adr_document: file too large (max 10 MB)")
				return
			}
			if errors.Is(err, errInvalidFileType) {
				response.Error(c, http.StatusBadRequest, "adr_document: invalid type (PDF, JPG, PNG only)")
				return
			}
			response.Error(c, http.StatusBadRequest, "adr_document: invalid file")
			return
		}
		if err := ValidateDriverFile(carFile, carPhotoMaxSize, carAllowed); err != nil {
			if errors.Is(err, errFileTooLarge) {
				response.Error(c, http.StatusBadRequest, "car_photo: file too large (max 5 MB)")
				return
			}
			if errors.Is(err, errInvalidFileType) {
				response.Error(c, http.StatusBadRequest, "car_photo: invalid type (JPG, PNG only)")
				return
			}
			response.Error(c, http.StatusBadRequest, "car_photo: invalid file")
			return
		}

		var companyID, freelanceDispatcherID *uuid.UUID
		if v := c.PostForm("company_id"); v != "" {
			parsed, err := uuid.Parse(v)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "company_id must be a valid UUID")
				return
			}
			companyID = &parsed
		}
		if v := c.PostForm("freelance_dispatcher_id"); v != "" {
			parsed, err := uuid.Parse(v)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "freelance_dispatcher_id must be a valid UUID")
				return
			}
			freelanceDispatcherID = &parsed
		}

		// INSERT только после успешной валидации всех данных
		var driverID string
		err = pool.QueryRow(ctx, `
			INSERT INTO drivers (
				first_name, last_name, phone_number, passport_series, passport_number,
				company_id, freelance_dispatcher_id, rating, work_status, account_status, language, platform
			) VALUES ($1, $2, $3, $4, $5, $6, $7, 0, 'free', 'pending', $8, $9)
			RETURNING id::text
		`, firstName, lastName, phone, passportSeries, passportNumber, companyID, freelanceDispatcherID, language, platform).Scan(&driverID)
		if err != nil {
			if isUniqueViolation(err) {
				response.Error(c, http.StatusBadRequest, "phone_number already registered")
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}

		// Сохранение файлов после успешного INSERT
		driverUUID, _ := uuid.Parse(driverID)
		if err := SaveDriverFileFromMultipart(ctx, pool, storageRoot, driverUUID, adrFile, "adr_document_path", "adr_document", adrMaxSize, adrAllowed); err != nil {
			// Если сохранение файла не удалось, удаляем водителя из БД
			_, _ = pool.Exec(ctx, `DELETE FROM drivers WHERE id = $1`, driverUUID)
			response.Error(c, http.StatusInternalServerError, "internal error saving adr_document")
			return
		}
		if err := SaveDriverFileFromMultipart(ctx, pool, storageRoot, driverUUID, carFile, "car_photo_path", "car_photo", carPhotoMaxSize, carAllowed); err != nil {
			// Если сохранение файла не удалось, удаляем водителя из БД
			_, _ = pool.Exec(ctx, `DELETE FROM drivers WHERE id = $1`, driverUUID)
			response.Error(c, http.StatusInternalServerError, "internal error saving car_photo")
			return
		}

		_ = auth.ConsumeRegistrationSession(ctx, pool, sessionID)

		tp, err := auth.CreateTokenPair(ctx, pool, driverID, cfg.JWTSecret)
		if err != nil {
			log.Printf("[auth] complete-register CreateTokenPair: %v", err)
		}
		resp := gin.H{
			"id":             driverID,
			"account_status": "pending",
		}
		if tp != nil {
			resp["access_token"] = tp.AccessToken
			resp["refresh_token"] = tp.RefreshToken
			resp["expires_in"] = tp.ExpiresIn
		}
		response.Success(c, http.StatusCreated, response.MsgCreated, resp)
	}
}
