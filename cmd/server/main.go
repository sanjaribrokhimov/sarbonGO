// Точка входа сервера: загрузка конфига, БД, Redis, миграции, роутер, graceful shutdown.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/sarbonGO/backend/internal/auth"
	"github.com/sarbonGO/backend/internal/config"
	"github.com/sarbonGO/backend/internal/db"
	"github.com/sarbonGO/backend/internal/i18n"
	"github.com/sarbonGO/backend/internal/migrations"
	"github.com/sarbonGO/backend/internal/redis"
	"github.com/sarbonGO/backend/internal/router"
)

func main() {
	// Локальный запуск без Docker: поднять Postgres, Redis, открыть pgAdmin (только macOS), затем .env.
	startLocalServices()

	// Подгружаем .env из deploy/docker/.env или .env в текущей/корневой папке.
	tryLoadEnv()

	// Загрузка конфигурации из переменных окружения (секреты не в репозитории).
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Загрузка переводов i18n (ru, en, uz, tr, zh) из встроенных JSON.
	if err := i18n.Load(); err != nil {
		log.Fatalf("i18n: %v", err)
	}

	ctx := context.Background()

	// Подключение к PostgreSQL (пул, таймауты, graceful close при выходе).
	pool, err := db.NewPostgres(ctx, cfg.Postgres)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close(pool, cfg.Server.ShutdownTimeout)

	// Запуск миграций при старте (только Go-код, без SQL-файлов).
	if err := migrations.NewRunner(pool).Up(ctx); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// Подключение к Redis (пул, таймауты; для rate limit и будущего кэша/сессий).
	rdb, err := redis.New(cfg.Redis)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redis.Close(rdb)

	// Валидатор Bearer-токена пользователя (заглушка; в production — JWT с проверкой подписи).
	validator := auth.NewStubValidator(cfg.Security)
	deps := router.Dependencies{
		FrontendClientToken: cfg.Security.FrontendClientToken,
		MobileClientToken:   cfg.Security.MobileClientToken,
		AuthValidator:       validator,
		RateLimitRPS:        cfg.Security.RateLimitRPS,
		Redis:               rdb,
		Pool:                pool,
		StoragePath:         cfg.Storage.Root,
		Security:            cfg.Security,
	}
	r := router.New(deps)

	srv := &http.Server{
		Addr:         ":" + fmt.Sprintf("%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	// Ожидание SIGINT/SIGTERM для корректного завершения.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown: завершение активных запросов в пределах таймаута.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("server stopped")
}

// startLocalServices при go run поднимает локальные Postgres, Redis и открывает pgAdmin (без Docker; только macOS).
func startLocalServices() {
	if runtime.GOOS != "darwin" {
		return
	}
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err == nil {
			log.Printf("[local] %s %v — запущено", name, args)
		}
	}
	run("brew", "services", "start", "postgresql@16")
	run("brew", "services", "start", "postgresql")
	run("brew", "services", "start", "redis")
	// pgAdmin: пробуем разные имена приложения на macOS
	if _, err := os.Stat("/Applications/pgAdmin 4.app"); err == nil {
		run("open", "/Applications/pgAdmin 4.app")
	} else if _, err := os.Stat("/Applications/pgAdmin4.app"); err == nil {
		run("open", "/Applications/pgAdmin4.app")
	} else {
		run("open", "-a", "pgAdmin 4")
		run("open", "-a", "pgAdmin4")
	}
}

// tryLoadEnv загружает .env из известных путей (для go run без ручного export).
func tryLoadEnv() {
	cwd, _ := os.Getwd()
	for _, path := range []string{
		filepath.Join(cwd, "deploy", "docker", ".env"),
		filepath.Join(cwd, ".env"),
		filepath.Join(cwd, "..", "..", "deploy", "docker", ".env"), // из cmd/server
	} {
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
			return
		}
	}
}
