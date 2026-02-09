package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

