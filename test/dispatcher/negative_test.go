// Негативные тесты Freelance Dispatcher: отсутствие/неверные заголовки, неверная сессия, неверный user token.
//
// --- Flow этого файла (как проходит тест) ---
// negative_test.go: каждый тест делает один запрос с «неправильными» данными и проверяет ответ с ошибкой (4xx).
//   • MissingClientToken: GET profile без X-Client-Token → 400, "missing required headers".
//   • RegistrationComplete_InvalidSession: complete с несуществующим session_id → 401, "session".
//   • Profile_NoToken: GET profile без X-User-Token → 401, "missing X-User-Token".
//   • Profile_InvalidToken: невалидный X-User-Token → 401, "invalid X-User-Token".
//   • LoginPassword_InvalidCredentials: неверный пароль → 401, "invalid phone or password".
package dispatcher

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"sarbonNew/test/common"
)

func TestDispatcherFlow_MissingClientToken_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	t.Log("GET /v1/dispatchers/profile без X-Client-Token (только Device-Type, Language)")
	h := map[string]string{
		"X-Device-Type": "android",
		"X-Language":    "ru",
		"Content-Type":  "application/json",
	}
	r := req(http.MethodGet, "/v1/dispatchers/profile", nil, h, "dummy-token")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusBadRequest, "missing required headers")
}

func TestDispatcherFlow_RegistrationComplete_InvalidSession_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	t.Log("POST /v1/dispatchers/registration/complete с несуществующим session_id — ожидается 401")
	body, _ := json.Marshal(map[string]interface{}{
		"session_id":       uuid.NewString(),
		"name":             "Test",
		"password":         "Pass123!",
		"passport_series":  "AB",
		"passport_number":  "1234567",
		"pinfl":            "12345678901234",
	})
	r := req(http.MethodPost, "/v1/dispatchers/registration/complete", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusUnauthorized, "session")
}

func TestDispatcherFlow_Profile_NoToken_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	t.Log("GET /v1/dispatchers/profile без X-User-Token — ожидается 401 missing X-User-Token")
	r := req(http.MethodGet, "/v1/dispatchers/profile", nil, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusUnauthorized, "missing X-User-Token")
}

func TestDispatcherFlow_Profile_InvalidToken_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	t.Log("GET /v1/dispatchers/profile с невалидным X-User-Token — ожидается 401 invalid X-User-Token")
	r := req(http.MethodGet, "/v1/dispatchers/profile", nil, baseHeaders(), "invalid-jwt-token")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusUnauthorized, "invalid X-User-Token")
}

func TestDispatcherFlow_LoginPassword_InvalidCredentials_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	t.Log("POST /v1/dispatchers/auth/login/password с неверным паролем — ожидается 401")
	body, _ := json.Marshal(map[string]interface{}{
		"phone":    "+998900000099",
		"password": "WrongPassword",
	})
	r := req(http.MethodPost, "/v1/dispatchers/auth/login/password", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusUnauthorized, "invalid phone or password")
}
