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
	"sarbonNew/internal/server"
)

func main() {
	config.LoadDotEnvUp(8)

	logger, _ := zap.NewProduction()
	if os.Getenv("APP_ENV") == "local" {
		logger, _ = zap.NewDevelopment()
	}
	defer func() { _ = logger.Sync() }()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		logger.Fatal("config load failed", zap.Error(err))
	}

	// Авто-миграции при старте API.
	// Это заменяет ручной запуск `cmd/migrate` в dev/stage окружениях.
	if err := runMigrationsUp(cfg.DatabaseURL); err != nil {
		logger.Fatal("migrations up failed", zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	infraDeps, err := infra.New(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("infra init failed", zap.Error(err))
	}
	defer infraDeps.Close()

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.NewRouter(cfg, infraDeps, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("http server starting", zap.String("addr", cfg.HTTPAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server error", zap.Error(err))
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

