// Интеграционный тест API создания компании (POST /v1/companies): проверка, что
// компания создаётся и текущий пользователь назначается владельцем (owner_id в ответе).
// Запуск с БД: TEST_DATABASE_URL или DATABASE_URL заданы.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"sarbonNew/internal/appusers"
	"sarbonNew/internal/approles"
	"sarbonNew/internal/companies"
	"sarbonNew/internal/companytz"
	"sarbonNew/internal/config"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/mw"
)

func testPoolAPI(t *testing.T) *pgxpool.Pool {
	t.Helper()
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = os.Getenv("DATABASE_URL")
	}
	if connStr == "" {
		t.Skip("TEST_DATABASE_URL or DATABASE_URL required for API integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("pool.Ping: %v", err)
	}
	return pool
}

// TestCreateCompanyAPI проверяет API создания компании (POST /v1/companies):
// авторизованный пользователь создаёт компанию и назначается владельцем (owner_id в ответе).
func TestCreateCompanyAPI(t *testing.T) {
	pool := testPoolAPI(t)
	defer pool.Close()
	ctx := context.Background()
	logger := zap.NewNop()

	cfg := config.Config{
		ClientTokenExpected: "", // в тесте не проверяем клиентский токен
	}
	companiesRepo := companies.NewRepo(pool)
	appusersRepo := appusers.NewRepo(pool)
	approlesRepo := approles.NewRepo(pool)
	ucrRepo := companytz.NewRepoUCR(pool)
	invitationsRepo := companytz.NewRepoInvitations(pool)
	auditRepo := companytz.NewRepoAudit(pool)
	jwtm := security.NewJWTManager("test-secret-key", 10*time.Minute, 24*time.Hour)
	companyTZH := NewCompanyTZHandler(logger, appusersRepo, companiesRepo, approlesRepo, ucrRepo, invitationsRepo, auditRepo, jwtm)

	// Пользователь компании (должен существовать и быть в company_users)
	user, err := appusersRepo.Create(ctx, "+7000"+uuid.New().String()[:8], "hash", nil, nil, nil, "owner")
	if err != nil {
		t.Fatalf("create company user: %v", err)
	}
	userID := uuid.MustParse(user.ID)

	// JWT с ролью "user" для app user
	tokens, _, err := jwtm.IssueWithCompany("user", userID, uuid.Nil)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	v1 := r.Group("/v1")
	v1.Use(mw.RequireBaseHeaders(cfg))
	authed := v1.Group("")
	authed.Use(mw.RequireAppUser(jwtm, nil))
	authed.POST("/companies", companyTZH.CreateCompany)

	body := map[string]any{
		"name": "API Test Company " + uuid.New().String(),
		"type": "SHIPPER",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/v1/companies", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Token", tokens.AccessToken)
	req.Header.Set("X-Device-Type", "web")
	req.Header.Set("X-Language", "ru")
	req.Header.Set("X-Client-Token", "test-client")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("POST /v1/companies: status %d, body %s", rec.Code, rec.Body.String())
		return
	}

	var envelope struct {
		Status string `json:"status"`
		Code   int    `json:"code"`
		Data   struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Type    string `json:"type"`
			OwnerID string `json:"owner_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if envelope.Status != "success" {
		t.Errorf("status: got %q want success", envelope.Status)
	}
	if envelope.Data.OwnerID != userID.String() {
		t.Errorf("owner_id: got %q want %s", envelope.Data.OwnerID, userID)
	}
	if envelope.Data.ID == "" || envelope.Data.ID == uuid.Nil.String() {
		t.Error("data.id must be set")
	}
	if envelope.Data.Name != body["name"] {
		t.Errorf("data.name: got %q want %q", envelope.Data.Name, body["name"])
	}
	if envelope.Data.Type != "SHIPPER" {
		t.Errorf("data.type: got %q want SHIPPER", envelope.Data.Type)
	}
}
