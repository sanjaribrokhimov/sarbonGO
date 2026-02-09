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

	JWTSigningKey    string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration
	OTPLength        int
	OTPTTL           time.Duration
	OTPResendCooldown time.Duration
	OTPMaxAttempts   int

	TelegramGatewayBaseURL string
	TelegramGatewayToken   string
	TelegramGatewaySenderID string
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
	cfg.OTPTTL = time.Duration(mustAtoi(getEnv("OTP_TTL_SECONDS", "180"))) * time.Second
	cfg.OTPResendCooldown = time.Duration(mustAtoi(getEnv("OTP_RESEND_COOLDOWN_SECONDS", "30"))) * time.Second
	cfg.OTPMaxAttempts = mustAtoi(getEnv("OTP_MAX_ATTEMPTS", "5"))

	cfg.TelegramGatewayBaseURL = getEnv("TELEGRAM_GATEWAY_BASE_URL", "https://gatewayapi.telegram.org")
	cfg.TelegramGatewayToken = os.Getenv("TELEGRAM_GATEWAY_TOKEN")
	cfg.TelegramGatewaySenderID = os.Getenv("TELEGRAM_GATEWAY_SENDER_ID")

	// Normalize base URL (no trailing slash)
	cfg.TelegramGatewayBaseURL = strings.TrimRight(cfg.TelegramGatewayBaseURL, "/")

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

