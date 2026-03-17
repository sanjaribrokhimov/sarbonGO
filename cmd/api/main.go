package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"

	"sarbonNew/internal/config"
	"sarbonNew/internal/chat"
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

	startChatMediaGC(ctx, log, infraDeps)

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

func startChatMediaGC(ctx context.Context, log *zap.Logger, deps *infra.Infra) {
	if strings.TrimSpace(os.Getenv("CHAT_MEDIA_GC_ENABLED")) != "1" {
		return
	}
	days := 30
	if v := strings.TrimSpace(os.Getenv("CHAT_MEDIA_GC_DAYS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}
	storageRoot := strings.TrimSpace(os.Getenv("CHAT_STORAGE_DIR"))
	if storageRoot == "" {
		storageRoot = "storage"
	}
	repo := chat.NewRepo(deps.PG)

	go func() {
		ticker := time.NewTicker(6 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// 1) delete attachments whose messages were deleted long ago
				exp, err := repo.ListExpiredDeletedMessageAttachments(ctx, days, 500)
				if err != nil {
					log.Warn("chat media gc: list expired attachments", zap.Error(err))
					continue
				}
				for _, e := range exp {
					_ = repo.DeleteAttachment(ctx, e.AttachmentID)
					// legacy paths cleanup (best effort)
					if e.Path != "" {
						_ = os.Remove(e.Path)
					}
					if e.ThumbPath != nil && *e.ThumbPath != "" {
						_ = os.Remove(*e.ThumbPath)
					}
				}

				// 2) delete orphan media files (not referenced by any attachment)
				orphan, err := repo.ListOrphanMediaFiles(ctx, days, 500)
				if err != nil {
					log.Warn("chat media gc: list orphans", zap.Error(err))
					continue
				}
				for _, f := range orphan {
					// only delete inside storageRoot for safety
					p := f.Path
					if strings.HasPrefix(p, storageRoot) {
						_ = os.Remove(p)
					}
					_ = repo.DeleteMediaFile(ctx, f.ID)
				}
				if len(exp) > 0 || len(orphan) > 0 {
					log.Info("chat media gc done", zap.Int("expired_attachments", len(exp)), zap.Int("orphan_files", len(orphan)))
				}
			}
		}
	}()
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
		// Если миграция прервалась (Dirty database version N), сбрасываем на N-1 и повторяем Up
		if strings.Contains(err.Error(), "Dirty database") {
			prevVersion := parseDirtyVersion(err.Error())
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
	// 1) Ищем migrations от текущей рабочей директории (go run . из корня или из cmd/api)
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if u, err := findMigrationsInDir(wd); err == nil {
		return u, nil
	}
	// 2) Fallback: ищем от директории main.go (чтобы работало при запуске из любой папки)
	if _, file, _, ok := runtime.Caller(0); ok {
		dir := filepath.Dir(file)
		if u, err := findMigrationsInDir(dir); err == nil {
			return u, nil
		}
	}
	return "", fmt.Errorf("migrations directory not found (cwd: %s)", wd)
}

// findMigrationsInDir ищет папку migrations в dir или выше по дереву.
func findMigrationsInDir(start string) (string, error) {
	dir := start
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
	return "", fmt.Errorf("not found from %s", start)
}

// parseDirtyVersion извлекает номер версии из сообщения "Dirty database version N" и возвращает N-1 для force.
func parseDirtyVersion(errMsg string) int {
	re := regexp.MustCompile(`(?i)dirty database version\s+(\d+)`)
	if m := re.FindStringSubmatch(errMsg); len(m) == 2 {
		if v, err := strconv.Atoi(m[1]); err == nil && v > 0 {
			return v - 1
		}
	}
	return 28 // fallback: перезапустить миграции с 29
}
