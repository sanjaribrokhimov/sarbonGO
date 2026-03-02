// Пакет cargo: тесты flow грузов (диспетчер создаёт → водитель видит → оффер → согласие/отказ) и чата.
// Запуск: go test -v ./test/cargo/
package cargo

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"sarbonNew/internal/config"
	"sarbonNew/internal/infra"
	"sarbonNew/internal/logger"
	"sarbonNew/internal/server"
	"sarbonNew/internal/security"
	"sarbonNew/internal/store"
	"sarbonNew/test/common"
)

var (
	testRouter       http.Handler
	testCfg          config.Config
	testSession      *store.SessionStore
	testDispSession  *store.DispatcherSessionStore
	testJWT          *security.JWTManager
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
	testSession = store.NewSessionStore(infraDeps.Redis, 15*time.Minute)
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

// createDriverToken создаёт водителя (session → registration/start) и возвращает access_token и driver.id (UUID string).
func createDriverToken(t *testing.T) (accessToken, driverID string) {
	t.Helper()
	if testRouter == nil || testSession == nil {
		t.Skip("router/session not initialized")
	}
	phone := "+9989020" + uuid.New().String()[:7] // уникальный номер
	sessionID, err := testSession.Create(context.Background(), phone)
	if err != nil {
		t.Fatalf("create driver session: %v", err)
	}
	body, _ := json.Marshal(map[string]interface{}{
		"session_id":      sessionID,
		"name":            "Cargo Test Driver",
		"oferta_accepted": true,
	})
	r := req(http.MethodPost, "/v1/registration/start", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	data := common.AssertSuccess(t, rec, true)
	if data == nil {
		return "", ""
	}
	access, _ := common.TokensFromData(data)
	if access == "" {
		t.Fatal("driver: no access_token")
	}
	// GET /v1/profile чтобы взять driver.id (carrier_id для оффера)
	r2 := req(http.MethodGet, "/v1/profile", nil, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	data2 := common.AssertSuccess(t, rec2, true)
	drv := common.DriverFromData(data2)
	if drv == nil {
		t.Fatal("driver: GET /v1/profile no data.driver")
	}
	id, _ := drv["id"].(string)
	if id == "" {
		t.Fatal("driver: data.driver.id empty")
	}
	return access, id
}

// createDispatcherToken создаёт диспетчера (session → registration/complete) и возвращает access_token и dispatcher.id (UUID string).
func createDispatcherToken(t *testing.T) (accessToken, dispatcherID string) {
	t.Helper()
	if testRouter == nil || testDispSession == nil {
		t.Skip("router/dispatcher session not initialized")
	}
	phone := "+9989021" + uuid.New().String()[:7]
	sessionID, err := testDispSession.Create(context.Background(), phone)
	if err != nil {
		t.Fatalf("create dispatcher session: %v", err)
	}
	body, _ := json.Marshal(map[string]interface{}{
		"session_id":       sessionID,
		"name":             "Cargo Test Dispatcher",
		"password":         "SecurePass123!",
		"passport_series":  "AB",
		"passport_number":  "1234567",
		"pinfl":            "12345678901234",
	})
	r := req(http.MethodPost, "/v1/dispatchers/registration/complete", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	data := common.AssertSuccess(t, rec, true)
	if data == nil {
		return "", ""
	}
	access, _ := common.TokensFromData(data)
	if access == "" {
		t.Fatal("dispatcher: no access_token")
	}
	r2 := req(http.MethodGet, "/v1/dispatchers/profile", nil, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	data2 := common.AssertSuccess(t, rec2, true)
	disp := common.DispatcherFromData(data2)
	if disp == nil {
		t.Fatal("dispatcher: GET /v1/dispatchers/profile no data.dispatcher")
	}
	id, _ := disp["id"].(string)
	if id == "" {
		t.Fatal("dispatcher: data.dispatcher.id empty")
	}
	return access, id
}
