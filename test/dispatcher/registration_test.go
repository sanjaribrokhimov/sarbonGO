// Тесты: регистрация Freelance Dispatcher (POST /v1/dispatchers/registration/complete).
//
// --- Flow этого файла (как проходит тест) ---
// registration_test.go:
//   • В тесте создаётся session диспетчера в Redis (имитация того, что auth/otp/verify для нового номера вернул session_id).
//   • Шаг 1: POST /v1/dispatchers/registration/complete с session_id, name, password, passport_series, passport_number, pinfl → 200.
//   • Проверка: data.status=registered|login, data.tokens (access + refresh), data.dispatcher.name.
//   • Шаг 2: GET /v1/dispatchers/profile с X-User-Token=access_token → 200.
//   • Проверка: data.dispatcher.name совпадает — авторизация по токену работает.
package dispatcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sarbonNew/test/common"
)

// TestDispatcherFlow_RegistrationComplete_Positive — полный сценарий: session → complete → токены → GET /v1/dispatchers/profile.
// Правильный flow: 1) Приложение вызывает auth/phone → otp_sent; auth/otp/verify (новый номер) → status=register, session_id.
// 2) POST /v1/dispatchers/registration/complete с session_id, name, password, passport_series, passport_number, pinfl.
// 3) Ответ: data.status=registered, data.tokens (access + refresh), data.dispatcher. 4) Запросы с X-User-Token=access_token.
func TestDispatcherFlow_RegistrationComplete_Positive(t *testing.T) {
	if testRouter == nil || testDispSession == nil {
		t.Skip("router or dispatcher session store not initialized")
	}
	phone := "+998901234500"
	sessionID, err := testDispSession.Create(context.Background(), phone)
	if err != nil {
		t.Fatalf("create dispatcher session: %v", err)
	}

	t.Log("POST /v1/dispatchers/registration/complete с session_id, name, password, passport, pinfl")
	body, _ := json.Marshal(map[string]interface{}{
		"session_id":       sessionID,
		"name":             "Test Dispatcher",
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
		t.Error("data.tokens.refresh_token: want non-empty")
	}
	disp := common.DispatcherFromData(data)
	if disp == nil {
		t.Error("data.dispatcher: want non-nil")
	} else if name, _ := disp["name"].(string); name != "Test Dispatcher" {
		t.Errorf("data.dispatcher.name: want Test Dispatcher, got %q", name)
	}

	t.Log("GET /v1/dispatchers/profile с X-User-Token — проверка авторизации по user token")
	r2 := req(http.MethodGet, "/v1/dispatchers/profile", nil, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	data2 := common.AssertSuccess(t, rec2, true)
	disp2 := common.DispatcherFromData(data2)
	if disp2 == nil {
		t.Fatal("GET /v1/dispatchers/profile: data.dispatcher want non-nil")
	}
	if name, _ := disp2["name"].(string); name != "Test Dispatcher" {
		t.Errorf("GET /v1/dispatchers/profile data.dispatcher.name: want Test Dispatcher, got %q", name)
	}
}
