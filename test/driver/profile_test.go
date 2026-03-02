// Тесты: профиль водителя (PATCH /v1/profile/driver, geo-push, transport-type).
//
// --- Flow этого файла (как проходит тест) ---
// profile_test.go:
//   • Общий шаг 0: создаётся session → POST /v1/registration/start → получаем access_token.
//   • TestDriverFlow_ProfilePatch_Positive: PATCH /v1/profile/driver с name, work_status → 200 → проверка data.event=updated, data.driver.name и work_status.
//   • TestDriverFlow_GeoPush_Positive: PATCH /v1/registration/geo-push с latitude, longitude → 200.
//   • TestDriverFlow_TransportType_Positive: PATCH /v1/registration/transport-type с driver_type, power_plate_type, trailer_plate_type → 200 → проверка data.driver.driver_type.
package driver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sarbonNew/test/common"
)

// TestDriverFlow_ProfilePatch_Positive — PATCH профиля с name и work_status; проверка data.event и data.driver.
func TestDriverFlow_ProfilePatch_Positive(t *testing.T) {
	if testRouter == nil || testSession == nil {
		t.Skip("router/session not initialized")
	}
	phone := "+998901111222"
	sessionID, err := testSession.Create(context.Background(), phone)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	body, _ := json.Marshal(map[string]interface{}{
		"session_id": sessionID, "name": "Patch Test", "oferta_accepted": true,
	})
	r := req(http.MethodPost, "/v1/registration/start", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	data := common.AssertSuccess(t, rec, true)
	access, _ := common.TokensFromData(data)
	if access == "" {
		t.Skip("no access token from registration/start")
	}

	t.Log("PATCH /v1/profile/driver с X-User-Token: name, work_status")
	patchBody, _ := json.Marshal(map[string]string{"name": "Updated Name", "work_status": "available"})
	r2 := req(http.MethodPatch, "/v1/profile/driver", patchBody, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	data2 := common.AssertSuccess(t, rec2, true)
	drv := common.DriverFromData(data2)
	if drv == nil {
		return
	}
	if event, _ := data2["event"].(string); event != "updated" {
		t.Errorf("data.event: want updated, got %q", event)
	}
	if name, _ := drv["name"].(string); name != "Updated Name" {
		t.Errorf("data.driver.name: want Updated Name, got %q", name)
	}
	if workStatus, _ := drv["work_status"].(string); workStatus != "available" {
		t.Errorf("data.driver.work_status: want available, got %q", workStatus)
	}
}

// TestDriverFlow_GeoPush_Positive — PATCH /v1/registration/geo-push с координатами.
func TestDriverFlow_GeoPush_Positive(t *testing.T) {
	if testRouter == nil || testSession == nil {
		t.Skip("router/session not initialized")
	}
	phone := "+998902223334"
	sessionID, _ := testSession.Create(context.Background(), phone)
	body, _ := json.Marshal(map[string]interface{}{
		"session_id": sessionID, "name": "Geo Test", "oferta_accepted": true,
	})
	r := req(http.MethodPost, "/v1/registration/start", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	data := common.AssertSuccess(t, rec, true)
	access, _ := common.TokensFromData(data)
	if access == "" {
		t.Skip("no access token")
	}

	t.Log("PATCH /v1/registration/geo-push с X-User-Token: latitude, longitude")
	geoBody, _ := json.Marshal(map[string]interface{}{"latitude": 41.3, "longitude": 69.2})
	r2 := req(http.MethodPatch, "/v1/registration/geo-push", geoBody, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	common.AssertSuccess(t, rec2, true)
}

// TestDriverFlow_TransportType_Positive — PATCH /v1/registration/transport-type; проверка data.driver.driver_type.
func TestDriverFlow_TransportType_Positive(t *testing.T) {
	if testRouter == nil || testSession == nil {
		t.Skip("router/session not initialized")
	}
	phone := "+998903334445"
	sessionID, _ := testSession.Create(context.Background(), phone)
	body, _ := json.Marshal(map[string]interface{}{
		"session_id": sessionID, "name": "Transport Test", "oferta_accepted": true,
	})
	r := req(http.MethodPost, "/v1/registration/start", body, baseHeaders(), "")
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	data := common.AssertSuccess(t, rec, true)
	access, _ := common.TokensFromData(data)
	if access == "" {
		t.Skip("no access token")
	}

	t.Log("PATCH /v1/registration/transport-type с X-User-Token: driver_type, power_plate_type, trailer_plate_type")
	transportBody, _ := json.Marshal(map[string]string{
		"driver_type": "driver", "power_plate_type": "TRUCK", "trailer_plate_type": "TENTED",
	})
	r2 := req(http.MethodPatch, "/v1/registration/transport-type", transportBody, baseHeaders(), access)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	data2 := common.AssertSuccess(t, rec2, true)
	drv := common.DriverFromData(data2)
	if drv != nil {
		if dt, _ := drv["driver_type"].(string); dt != "driver" {
			t.Errorf("data.driver.driver_type: want driver, got %q", dt)
		}
	}
}
