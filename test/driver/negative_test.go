// Негативные тесты Driver: отсутствие/неверные заголовки, неверная сессия, неверный user token, невалидные поля.
//
// --- Flow этого файла (как проходит тест) ---
// negative_test.go: каждый тест делает один запрос с «неправильными» данными и проверяет, что API возвращает ошибку (4xx).
//   • MissingClientToken: запрос без X-Client-Token → 400, "missing required headers".
//   • InvalidClientToken: неверный X-Client-Token (если задан CLIENT_TOKEN_EXPECTED) → 401, "invalid X-Client-Token".
//   • MissingBaseHeaders: запрос без заголовков → 400.
//   • InvalidDeviceType: X-Device-Type=desktop → 400, "invalid X-Device-Type".
//   • RegistrationStart_InvalidSession: несуществующий session_id → 401, "session".
//   • RegistrationStart_OfertaNotAccepted: oferta_accepted=false → 400.
//   • Profile_NoToken: GET /v1/profile без X-User-Token → 401, "missing X-User-Token".
//   • Profile_InvalidToken: невалидный JWT → 401, "invalid X-User-Token".
//   • ProfilePatch_InvalidWorkStatus / TransportType_InvalidDriverType: неверные значения полей → 400.
package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"sarbonNew/test/common"
)

func TestDriverFlow_MissingClientToken_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	t.Log("GET /v1/transport-options без X-Client-Token (только X-Device-Type, X-Language)")
	h := map[string]string{
		"X-Device-Type": "android",
		"X-Language":     "ru",
		"Content-Type":   "application/json",
	}
	r := req(http.MethodGet, "/v1/transport-options", nil, h, "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusBadRequest, "missing required headers")
}

func TestDriverFlow_InvalidClientToken_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	if testCfg.ClientTokenExpected == "" {
		t.Skip("CLIENT_TOKEN_EXPECTED not set — проверка неверного client token пропущена")
	}
	t.Log("GET /v1/transport-options с неверным X-Client-Token (ожидается 401 invalid X-Client-Token)")
	h := baseHeaders()
	h["X-Client-Token"] = "wrong-client-token"
	r := req(http.MethodGet, "/v1/transport-options", nil, h, "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusUnauthorized, "invalid X-Client-Token")
}

func TestDriverFlow_MissingBaseHeaders_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	t.Log("GET /v1/transport-options без заголовков")
	r := httptest.NewRequest(http.MethodGet, "/v1/transport-options", nil)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusBadRequest, "missing required headers")
}

func TestDriverFlow_InvalidDeviceType_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	h := baseHeaders()
	h["X-Device-Type"] = "desktop"
	r := req(http.MethodGet, "/v1/transport-options", nil, h, "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusBadRequest, "invalid X-Device-Type")
}

func TestDriverFlow_RegistrationStart_InvalidSession_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	t.Log("POST /v1/registration/start с несуществующим session_id — ожидается 401")
	body, _ := json.Marshal(map[string]interface{}{
		"session_id": uuid.NewString(), "name": "Test", "oferta_accepted": true,
	})
	r := req(http.MethodPost, "/v1/registration/start", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusUnauthorized, "session")
}

func TestDriverFlow_RegistrationStart_OfertaNotAccepted_Negative(t *testing.T) {
	if testRouter == nil || testSession == nil {
		t.Skip("router/session not initialized")
	}
	sessionID, _ := testSession.Create(context.Background(), "+998909998887")
	body, _ := json.Marshal(map[string]interface{}{
		"session_id": sessionID, "name": "Test", "oferta_accepted": false,
	})
	r := req(http.MethodPost, "/v1/registration/start", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusBadRequest, "")
}

func TestDriverFlow_Profile_NoToken_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	t.Log("GET /v1/profile без X-User-Token — ожидается 401 missing X-User-Token")
	r := req(http.MethodGet, "/v1/profile", nil, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusUnauthorized, "missing X-User-Token")
}

func TestDriverFlow_Profile_InvalidToken_Negative(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	r := req(http.MethodGet, "/v1/profile", nil, baseHeaders(), "invalid-jwt-token")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusUnauthorized, "invalid X-User-Token")
}

func TestDriverFlow_ProfilePatch_InvalidWorkStatus_Negative(t *testing.T) {
	if testRouter == nil || testSession == nil {
		t.Skip("router/session not initialized")
	}
	sessionID, _ := testSession.Create(context.Background(), "+998907776665")
	body, _ := json.Marshal(map[string]interface{}{
		"session_id": sessionID, "name": "Bad Status", "oferta_accepted": true,
	})
	r := req(http.MethodPost, "/v1/registration/start", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	data := common.AssertSuccess(t, rec, true)
	access, _ := common.TokensFromData(data)
	if access == "" {
		t.Skip("no token")
	}
	patchBody, _ := json.Marshal(map[string]string{"work_status": "invalid_status"})
	r2 := req(http.MethodPatch, "/v1/profile/driver", patchBody, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	common.AssertError(t, rec2, http.StatusBadRequest, "work_status")
}

func TestDriverFlow_TransportType_InvalidDriverType_Negative(t *testing.T) {
	if testRouter == nil || testSession == nil {
		t.Skip("router/session not initialized")
	}
	sessionID, _ := testSession.Create(context.Background(), "+998906665554")
	body, _ := json.Marshal(map[string]interface{}{
		"session_id": sessionID, "name": "Bad Type", "oferta_accepted": true,
	})
	r := req(http.MethodPost, "/v1/registration/start", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	data := common.AssertSuccess(t, rec, true)
	access, _ := common.TokensFromData(data)
	if access == "" {
		t.Skip("no token")
	}
	transportBody, _ := json.Marshal(map[string]string{
		"driver_type": "invalid_type", "power_plate_type": "TRUCK", "trailer_plate_type": "TENTED",
	})
	r2 := req(http.MethodPatch, "/v1/registration/transport-type", transportBody, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	common.AssertError(t, rec2, http.StatusBadRequest, "driver_type")
}
