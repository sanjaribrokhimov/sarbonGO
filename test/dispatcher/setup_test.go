// Пакет dispatcher: общий setup для тестов flow Freelance Dispatcher (регистрация, логин по паролю, профиль).
// Запуск: go test -v ./test/dispatcher/
package dispatcher

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"sarbonNew/internal/config"
	"sarbonNew/internal/infra"
	"sarbonNew/internal/logger"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server"
	"sarbonNew/internal/store"
)

var (
	testRouter     http.Handler
	testCfg        config.Config
	testDispSession *store.DispatcherSessionStore
	testJWT        *security.JWTManager
)

func TestMain(m *testing.M) {
	config.LoadDotEnvUp(8)
	cfg, err := config.LoadFromEnv()
	if err != nil {
		os.Exit(m.Run())
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	zlog := logger.NewDevelopment()
	infraDeps, err := infra.New(ctx, cfg, zlog)
	if err != nil {
		os.Exit(m.Run())
		return
	}
	defer infraDeps.Close()

	testCfg = cfg
	testRouter = server.NewRouter(cfg, infraDeps, zlog)
	testDispSession = store.NewDispatcherSessionStore(infraDeps.Redis, "disp_regsession", 15*time.Minute)
	testJWT = security.NewJWTManager(cfg.JWTSigningKey, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	os.Exit(m.Run())
}

func baseHeaders() map[string]string {
	h := map[string]string{
		"X-Device-Type":  "android",
		"X-Language":     "ru",
		"X-Client-Token": "test-client-token",
		"Content-Type":   "application/json",
	}
	if testCfg.ClientTokenExpected != "" {
		h["X-Client-Token"] = testCfg.ClientTokenExpected
	}
	return h
}

func req(method, path string, body []byte, headers map[string]string, userToken string) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	if userToken != "" {
		r.Header.Set("X-User-Token", userToken)
	}
	return r
}
