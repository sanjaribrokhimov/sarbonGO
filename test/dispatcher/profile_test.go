// Тесты: профиль Freelance Dispatcher (GET/PATCH /v1/dispatchers/profile).
//
// --- Flow этого файла (как проходит тест) ---
// profile_test.go:
//   • Шаг 0: создаётся session → POST /v1/dispatchers/registration/complete → получаем access_token.
//   • Шаг 1: PATCH /v1/dispatchers/profile с X-User-Token и body { "name": "Updated Disp Name" } → 200.
//   • Проверка: data.status=ok, data.dispatcher.name=Updated Disp Name — обновление профиля работает.
package dispatcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sarbonNew/test/common"
)

// TestDispatcherFlow_ProfilePatch_Positive — PATCH профиля диспетчера; проверка data.status=ok и data.dispatcher.
func TestDispatcherFlow_ProfilePatch_Positive(t *testing.T) {
	if testRouter == nil || testDispSession == nil {
		t.Skip("router or dispatcher session not initialized")
	}
	phone := "+998902222200"
	sessionID, _ := testDispSession.Create(context.Background(), phone)
	regBody, _ := json.Marshal(map[string]interface{}{
		"session_id":       sessionID,
		"name":             "Patch Disp",
		"password":         "Pass123!",
		"passport_series":  "EF",
		"passport_number":  "1111111",
		"pinfl":            "11112222333344",
	})
	rReg := req(http.MethodPost, "/v1/dispatchers/registration/complete", regBody, baseHeaders(), "")
	recReg := httptest.NewRecorder()
	testRouter.ServeHTTP(recReg, rReg)
	dataReg := common.AssertSuccess(t, recReg, true)
	access, _ := common.TokensFromData(dataReg)
	if access == "" {
		t.Skip("no access token from registration")
	}

	t.Log("PATCH /v1/dispatchers/profile с X-User-Token: name")
	patchBody, _ := json.Marshal(map[string]string{"name": "Updated Disp Name"})
	r2 := req(http.MethodPatch, "/v1/dispatchers/profile", patchBody, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	data2 := common.AssertSuccess(t, rec2, true)
	disp := common.DispatcherFromData(data2)
	if disp == nil {
		return
	}
	if status, _ := data2["status"].(string); status != "ok" {
		t.Errorf("data.status: want ok, got %q", status)
	}
	if name, _ := disp["name"].(string); name != "Updated Disp Name" {
		t.Errorf("data.dispatcher.name: want Updated Disp Name, got %q", name)
	}
}
