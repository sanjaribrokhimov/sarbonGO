// Тесты: авторизация Freelance Dispatcher (POST /v1/dispatchers/auth/login/password).
//
// --- Flow этого файла (как проходит тест) ---
// auth_test.go:
//   • Шаг 0: создаётся session → POST /v1/dispatchers/registration/complete → диспетчер зарегистрирован.
//   • Шаг 1: POST /v1/dispatchers/auth/login/password с phone и password → 200.
//   • Проверка: data.status=login, data.tokens.access_token не пустой.
//   • Шаг 2: GET /v1/dispatchers/profile с X-User-Token=access_token → 200.
//   • Проверка: data.dispatcher.name совпадает с зарегистрированным — вход по паролю и доступ по токену работают.
package dispatcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sarbonNew/test/common"
)

// TestDispatcherFlow_LoginPassword_Positive — логин по phone + password после регистрации; проверка токенов и GET profile.
// Правильный flow: 1) Диспетчер уже зарегистрирован (registration/complete). 2) POST /v1/dispatchers/auth/login/password
// с phone и password. 3) Ответ: data.status=login, data.tokens. 4) Дальнейшие запросы с X-User-Token.
func TestDispatcherFlow_LoginPassword_Positive(t *testing.T) {
	if testRouter == nil || testDispSession == nil {
		t.Skip("router or dispatcher session not initialized")
	}
	phone := "+998901111100"
	sessionID, _ := testDispSession.Create(context.Background(), phone)
	regBody, _ := json.Marshal(map[string]interface{}{
		"session_id":       sessionID,
		"name":             "Login Test Disp",
		"password":         "MyPassword123!",
		"passport_series":  "CD",
		"passport_number":  "7654321",
		"pinfl":            "98765432101234",
	})
	rReg := req(http.MethodPost, "/v1/dispatchers/registration/complete", regBody, baseHeaders(), "")
	recReg := httptest.NewRecorder()
	testRouter.ServeHTTP(recReg, rReg)
	if recReg.Code != http.StatusOK {
		t.Skipf("registration/complete failed: %d", recReg.Code)
	}

	t.Log("POST /v1/dispatchers/auth/login/password с phone и password")
	loginBody, _ := json.Marshal(map[string]interface{}{
		"phone":    phone,
		"password": "MyPassword123!",
	})
	r := req(http.MethodPost, "/v1/dispatchers/auth/login/password", loginBody, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	data := common.AssertSuccess(t, rec, true)
	if data == nil {
		return
	}
	if data["status"] != "login" {
		t.Errorf("data.status: want login, got %v", data["status"])
	}
	access, _ := common.TokensFromData(data)
	if access == "" {
		t.Fatal("data.tokens.access_token: want non-empty")
	}

	t.Log("GET /v1/dispatchers/profile с полученным X-User-Token")
	r2 := req(http.MethodGet, "/v1/dispatchers/profile", nil, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	data2 := common.AssertSuccess(t, rec2, true)
	disp := common.DispatcherFromData(data2)
	if disp == nil {
		t.Fatal("GET profile: data.dispatcher want non-nil")
	}
	if name, _ := disp["name"].(string); name != "Login Test Disp" {
		t.Errorf("data.dispatcher.name: want Login Test Disp, got %q", name)
	}
}
