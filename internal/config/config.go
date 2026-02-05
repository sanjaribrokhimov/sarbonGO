// Конфигурация приложения только из переменных окружения (секреты не в репозитории).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config — корневая структура конфигурации (env-only).
type Config struct {
	Server   Server
	Postgres Postgres
	Redis    Redis
	Security Security
	Storage  Storage
}

// Storage — корень папки для файлов (водители: storage/drivers/{id}/).
type Storage struct {
	Root string
}

// Server — настройки HTTP-сервера (порт, таймауты, время на shutdown).
type Server struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// Postgres — DSN, размер пула, таймауты подключения и жизни соединений.
type Postgres struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
}

// Redis — адрес, пароль, пул, таймауты (для rate limit, кэша, сессий).
type Redis struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Security — лимиты запросов, секрет JWT пользователя и отдельные токены для frontend/mobile (защита доступа к API).
type Security struct {
	RateLimitRPS        int
	RateLimitBurst      int
	JWTSecret           string
	FrontendClientToken string // токен приложения frontend — только с ним запросы от фронта принимаются
	MobileClientToken   string // токен приложения mobile — только с ним запросы от мобилок принимаются
	// Telegram OTP: Gateway API (Verification Codes) — по номеру, без chat_id
	TelegramGatewayToken string // токен с https://gateway.telegram.org/account/api
	OTPTTLSec            int    // время жизни OTP в БД (сек); 180–300 (3–5 мин)
	OTPAttemptsMax       int    // макс. попыток ввода кода (3–5)
	OTPRateLimitPerPhone int    // макс. отправок OTP на один номер за окно (например 3 за 15 мин)
}

// Load читает конфиг из env; JWT_SECRET обязателен.
func Load() (*Config, error) {
	cfg := &Config{
		Server: Server{
			Port:            getInt("SERVER_PORT", 8080),
			ReadTimeout:     getDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Postgres: Postgres{
			DSN:             getEnv("POSTGRES_DSN", "postgres://sarbon:sarbon@localhost:5432/sarbon?sslmode=disable"),
			MaxConns:        int32(getInt("POSTGRES_MAX_CONNS", 25)),
			MinConns:        int32(getInt("POSTGRES_MIN_CONNS", 5)),
			MaxConnLifetime: getDuration("POSTGRES_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime: getDuration("POSTGRES_MAX_CONN_IDLE_TIME", 30*time.Minute),
			ConnectTimeout:  getDuration("POSTGRES_CONNECT_TIMEOUT", 5*time.Second),
		},
		Redis: Redis{
			Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getInt("REDIS_DB", 0),
			PoolSize:     getInt("REDIS_POOL_SIZE", 10),
			MinIdleConns: getInt("REDIS_MIN_IDLE", 2),
			DialTimeout:  getDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		},
		Security: Security{
			RateLimitRPS:        getInt("RATE_LIMIT_RPS", 100),
			RateLimitBurst:      getInt("RATE_LIMIT_BURST", 200),
			JWTSecret:           getEnv("JWT_SECRET", ""),
			FrontendClientToken: getEnv("FRONTEND_CLIENT_TOKEN", ""),
			MobileClientToken:   getEnv("MOBILE_CLIENT_TOKEN", ""),
			TelegramGatewayToken: getEnv("TELEGRAM_GATEWAY_TOKEN", ""),
			OTPTTLSec:            getInt("OTP_TTL_SEC", 300),
			OTPAttemptsMax:       getInt("OTP_ATTEMPTS_MAX", 5),
			OTPRateLimitPerPhone: getInt("OTP_RATE_LIMIT_PER_PHONE", 3),
		},
		Storage: Storage{
			Root: getEnv("STORAGE_PATH", "storage"),
		},
	}
	if cfg.Security.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.Security.FrontendClientToken == "" {
		return nil, fmt.Errorf("FRONTEND_CLIENT_TOKEN is required")
	}
	if cfg.Security.MobileClientToken == "" {
		return nil, fmt.Errorf("MOBILE_CLIENT_TOKEN is required")
	}
	return cfg, nil
}

// getEnv возвращает значение переменной окружения или значение по умолчанию.
func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getInt парсит целое из env или возвращает def.
func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

// getBool парсит 1/true/yes как true, иначе false.
func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		switch strings.ToLower(v) {
		case "1", "true", "yes":
			return true
		case "0", "false", "no":
			return false
		}
	}
	return def
}

// getDuration парсит длительность из env или возвращает def.
func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
