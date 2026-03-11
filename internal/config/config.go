package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv   string
	HTTPAddr string

	ClientTokenExpected string

	DatabaseURL string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSigningKey               string
	JWTAccessTTL                time.Duration
	JWTRefreshTTL               time.Duration
	OTPLength                   int
	OTPTTL                      time.Duration
	OTPResendCooldown           time.Duration
	OTPMaxAttempts              int
	OTPSendLimitPerPhonePerHour int
	OTPSendLimitPerIPPerHour    int
	OTPSendWindow               time.Duration
	OTPVerifyAttemptsPerPhone   int           // макс. попыток ввода OTP на один номер в окне (0 = только maxAttempts на один код)
	OTPVerifyWindowSeconds     int           // окно в секундах для OTPVerifyAttemptsPerPhone

	TelegramGatewayBaseURL  string
	TelegramGatewayToken    string
	TelegramGatewaySenderID string
	TelegramGatewayBypass   bool // dev: do not call gateway, just log OTP

	// FreelanceDispatcherCargoLimit — макс. число грузов на одного фриланс-диспетчера (0 = без лимита)
	FreelanceDispatcherCargoLimit int
}

func LoadFromEnv() (Config, error) {
	var cfg Config

	cfg.AppEnv = getEnv("APP_ENV", "local")
	cfg.HTTPAddr = getEnv("HTTP_ADDR", ":8080")
	cfg.ClientTokenExpected = os.Getenv("CLIENT_TOKEN_EXPECTED")

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	cfg.RedisAddr = getEnv("REDIS_ADDR", "localhost:6379")
	cfg.RedisPassword = os.Getenv("REDIS_PASSWORD")
	cfg.RedisDB = mustAtoi(getEnv("REDIS_DB", "0"))

	cfg.JWTSigningKey = os.Getenv("JWT_SIGNING_KEY")
	if cfg.JWTSigningKey == "" {
		return Config{}, fmt.Errorf("JWT_SIGNING_KEY is required")
	}
	cfg.JWTAccessTTL = time.Duration(mustAtoi(getEnv("JWT_ACCESS_TTL_SECONDS", "900"))) * time.Second
	cfg.JWTRefreshTTL = time.Duration(mustAtoi(getEnv("JWT_REFRESH_TTL_SECONDS", "2592000"))) * time.Second

	cfg.OTPLength = mustAtoi(getEnv("OTP_LENGTH", "6"))
	cfg.OTPTTL = time.Duration(mustAtoi(getEnv("OTP_TTL_SECONDS", "120"))) * time.Second
	cfg.OTPResendCooldown = time.Duration(mustAtoi(getEnv("OTP_RESEND_COOLDOWN_SECONDS", "30"))) * time.Second
	cfg.OTPMaxAttempts = mustAtoi(getEnv("OTP_MAX_ATTEMPTS", "5"))
	cfg.OTPSendLimitPerPhonePerHour = mustAtoi(getEnv("OTP_SEND_LIMIT_PER_PHONE_PER_HOUR", "10"))
	cfg.OTPSendLimitPerIPPerHour = mustAtoi(getEnv("OTP_SEND_LIMIT_PER_IP_PER_HOUR", "30"))
	cfg.OTPSendWindow = time.Duration(mustAtoi(getEnv("OTP_SEND_WINDOW_SECONDS", "3600"))) * time.Second
	cfg.OTPVerifyAttemptsPerPhone = mustAtoi(getEnv("OTP_VERIFY_ATTEMPTS_PER_PHONE", "10"))
	cfg.OTPVerifyWindowSeconds = mustAtoi(getEnv("OTP_VERIFY_WINDOW_SECONDS", "900"))

	cfg.TelegramGatewayBaseURL = getEnv("TELEGRAM_GATEWAY_BASE_URL", "https://gatewayapi.telegram.org")
	cfg.TelegramGatewayToken = os.Getenv("TELEGRAM_GATEWAY_TOKEN")
	cfg.TelegramGatewaySenderID = os.Getenv("TELEGRAM_GATEWAY_SENDER_ID")
	cfg.TelegramGatewayBypass = mustBool(getEnv("TELEGRAM_GATEWAY_BYPASS", "false"))

	// Normalize base URL (no trailing slash)
	cfg.TelegramGatewayBaseURL = strings.TrimRight(cfg.TelegramGatewayBaseURL, "/")

	cfg.FreelanceDispatcherCargoLimit = mustAtoi(getEnv("FREELANCE_DISPATCHER_CARGO_LIMIT", "0"))

	return cfg, nil
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func mustAtoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return n
}

func mustBool(s string) bool {
	b, err := strconv.ParseBool(strings.TrimSpace(s))
	if err != nil {
		panic(err)
	}
	return b
}
