package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"

	"sarbonNew/internal/config"
	"sarbonNew/internal/infra"
	"sarbonNew/internal/logger"
	"sarbonNew/internal/server"
)

func main() {
	config.LoadDotEnvUp(8)

	var log *zap.Logger
	if os.Getenv("APP_ENV") == "local" {
		log = logger.NewDevelopment()
	} else {
		var err error
		log, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	}
	defer func() { _ = log.Sync() }()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatal("config load failed", zap.Error(err))
	}
	log.Info("otp config",
		zap.Bool("telegram_gateway_bypass", cfg.TelegramGatewayBypass),
		zap.Int("otp_len", cfg.OTPLength),
		zap.Duration("otp_ttl", cfg.OTPTTL),
		zap.Duration("otp_resend_cooldown", cfg.OTPResendCooldown),
		zap.Int("otp_send_limit_phone_per_hour", cfg.OTPSendLimitPerPhonePerHour),
		zap.Int("otp_send_limit_ip_per_hour", cfg.OTPSendLimitPerIPPerHour),
		zap.Duration("otp_send_window", cfg.OTPSendWindow),
	)

	// Авто-миграции при старте API.
	// Это заменяет ручной запуск `cmd/migrate` в dev/stage окружениях.
	if err := runMigrationsUp(cfg.DatabaseURL); err != nil {
		log.Fatal("migrations up failed", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	infraDeps, err := infra.New(ctx, cfg, log)
	if err != nil {
		log.Fatal("infra init failed", zap.Error(err))
	}
	defer infraDeps.Close()

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.NewRouter(cfg, infraDeps, log),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("http server starting", zap.String("addr", cfg.HTTPAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("http server error", zap.Error(err))
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpServer.Shutdown(shutdownCtx)
}

func runMigrationsUp(dbURL string) error {
	if strings.TrimSpace(dbURL) == "" {
		return fmt.Errorf("DATABASE_URL is empty")
	}
	// golang-migrate pgx/v5 driver registers as "pgx5".
	if strings.HasPrefix(dbURL, "postgres://") {
		dbURL = "pgx5://" + strings.TrimPrefix(dbURL, "postgres://")
	}
	if strings.HasPrefix(dbURL, "pgx://") {
		dbURL = "pgx5://" + strings.TrimPrefix(dbURL, "pgx://")
	}

	sourceURL, err := findMigrationsSourceURL()
	if err != nil {
		return err
	}

	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		return fmt.Errorf("migrate init error: %w", err)
	}
	defer func() {
		_, _ = m.Close()
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		// Если миграция прервалась (Dirty database version N), сбрасываем на предыдущую и повторяем Up
		if strings.Contains(err.Error(), "Dirty database") {
			const prevVersion = 28 // версия до 000029
			if forceErr := m.Force(prevVersion); forceErr != nil {
				return fmt.Errorf("force version after dirty failed: %w", forceErr)
			}
			if retryErr := m.Up(); retryErr != nil && retryErr != migrate.ErrNoChange {
				return retryErr
			}
			return nil
		}
		return err
	}
	return nil
}

func findMigrationsSourceURL() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for i := 0; i < 12; i++ {
		migDir := filepath.Join(dir, "migrations")
		if st, err := os.Stat(migDir); err == nil && st.IsDir() {
			return "file://" + migDir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("migrations directory not found from cwd: %s", wd)
}
