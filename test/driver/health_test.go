// Тесты: публичные эндпоинты и обязательные base headers для Driver API.
//
// --- Flow этого файла (как проходит тест) ---
// health_test.go:
//   • TestDriverFlow_Health: GET /health без заголовков → 200 → проверка data.status=ok.
//   • TestDriverFlow_BaseHeadersRequired_Positive: GET /v1/transport-options с X-Device-Type, X-Language, X-Client-Token → 200.
package driver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"sarbonNew/test/common"
)

// TestDriverFlow_Health — GET /health без заголовков (публичный эндпоинт).
// Flow: приложение проверяет доступность сервиса; ответ 200, data.status=ok.
func TestDriverFlow_Health(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized (no DATABASE_URL/JWT_SIGNING_KEY)")
	}
	t.Log("GET /health без заголовков (публичный эндпоинт)")
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	data := common.AssertSuccess(t, rec, true)
	if data != nil && data["status"] != "ok" {
		t.Errorf("data.status: want ok, got %v", data["status"])
	}
}

// TestDriverFlow_BaseHeadersRequired_Positive — запрос с полным набором base headers.
// Flow: все запросы к /v1/* требуют X-Device-Type, X-Language, X-Client-Token; при их наличии — 200.
func TestDriverFlow_BaseHeadersRequired_Positive(t *testing.T) {
	if testRouter == nil {
		t.Skip("router not initialized")
	}
	t.Log("GET /v1/transport-options с полными base headers (X-Device-Type, X-Language, X-Client-Token)")
	r := req(http.MethodGet, "/v1/transport-options", nil, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertSuccess(t, rec, true)
}
