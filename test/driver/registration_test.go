// Тесты: регистрация/старт водителя (POST /v1/registration/start).
//
// --- Flow этого файла (как проходит тест) ---
// registration_test.go:
//   • В тесте создаётся session в Redis (имитация успешного OTP).
//   • Шаг 1: POST /v1/registration/start с session_id, name, oferta_accepted=true → 200.
//   • Проверка: data.status=registered|login, data.tokens.access_token и refresh_token, data.driver.name.
//   • Шаг 2: GET /v1/profile с заголовком X-User-Token=access_token → 200.
//   • Проверка: в ответе тот же data.driver.name — авторизация по токену работает.
package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sarbonNew/test/common"
)

// TestDriverFlow_RegistrationStart_Positive — полный сценарий: session → start → токены → GET /v1/profile по X-User-Token.
// Правильный flow: 1) Мобильное приложение получает session_id после OTP. 2) POST /v1/registration/start с session_id, name, oferta_accepted=true.
// 3) Ответ: data.status=registered|login, data.tokens.access_token, data.tokens.refresh_token, data.driver. 4) Дальнейшие запросы с заголовком X-User-Token=access_token.
func TestDriverFlow_RegistrationStart_Positive(t *testing.T) {
	if testRouter == nil || testSession == nil {
		t.Skip("router/session not initialized")
	}
	phone := "+998901234567"
	sessionID, err := testSession.Create(context.Background(), phone)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	t.Log("POST /v1/registration/start с session_id, name, oferta_accepted=true")
	body, _ := json.Marshal(map[string]interface{}{
		"session_id":      sessionID,
		"name":            "Test Driver",
		"oferta_accepted": true,
	})
	r := req(http.MethodPost, "/v1/registration/start", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	data := common.AssertSuccess(t, rec, true)
	if data == nil {
		return
	}
	if data["status"] != "registered" && data["status"] != "login" {
		t.Errorf("data.status: want registered or login, got %v", data["status"])
	}
	access, refresh := common.TokensFromData(data)
	if access == "" {
		t.Fatal("data.tokens.access_token: want non-empty")
	}
	if refresh == "" {
		t.Error("data.tokens.refresh_token: want non-empty (user token pair)")
	}
	drv := common.DriverFromData(data)
	if drv == nil {
		t.Error("data.driver: want non-nil")
	} else if name, _ := drv["name"].(string); name != "Test Driver" {
		t.Errorf("data.driver.name: want Test Driver, got %q", name)
	}

	t.Log("GET /v1/profile с X-User-Token (access_token) — проверка авторизации по user token")
	r2 := req(http.MethodGet, "/v1/profile", nil, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	data2 := common.AssertSuccess(t, rec2, true)
	drv2 := common.DriverFromData(data2)
	if drv2 == nil {
		t.Fatal("GET /v1/profile: data.driver want non-nil")
	}
	if name, _ := drv2["name"].(string); name != "Test Driver" {
		t.Errorf("GET /v1/profile data.driver.name: want Test Driver, got %q", name)
	}
}
