// Негативные тесты: cargo API и chat — невалидные запросы, несуществующие id.
//
// --- Flow этого файла ---
// • Cargo: создание без load/unload → 400; GET /api/cargo/:id с несуществующим id → 404 или 500; accept несуществующего оффера → 400.
// • Chat: peer_id = свой id (cannot chat with yourself) → 400; невалидный peer_id → 400.
package cargo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"sarbonNew/test/common"
)

func TestCargoFlow_Create_NoLoadUnload_Negative(t *testing.T) {
	dispToken, _ := createDispatcherToken(t)
	if dispToken == "" {
		return
	}
	// Только load, без unload → 400
	body, _ := json.Marshal(map[string]interface{}{
		"title":       "Invalid",
		"weight":      100,
		"truck_type":  "tent",
		"capacity":    10,
		"route_points": []map[string]interface{}{
			{"type": "load", "address": "A", "lat": 41, "lng": 69, "point_order": 1},
		},
	})
	r := req(http.MethodPost, "/api/cargo", body, baseHeaders(), dispToken)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusBadRequest, "load")
}

func TestCargoFlow_GetByID_NotFound_Negative(t *testing.T) {
	driverToken, _ := createDriverToken(t)
	if driverToken == "" {
		return
	}
	fakeID := uuid.New().String()
	r := req(http.MethodGet, "/api/cargo/"+fakeID, nil, baseHeaders(), driverToken)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Errorf("GET /api/cargo/:id (fake): want 404 or 500, got %d", rec.Code)
	}
}

func TestCargoFlow_AcceptOffer_NotFound_Negative(t *testing.T) {
	dispToken, _ := createDispatcherToken(t)
	if dispToken == "" {
		return
	}
	fakeOfferID := uuid.New().String()
	r := req(http.MethodPost, "/api/offers/"+fakeOfferID+"/accept", nil, baseHeaders(), dispToken)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusBadRequest, "")
}

func TestChatFlow_SameUser_Negative(t *testing.T) {
	driverToken, driverID := createDriverToken(t)
	if driverToken == "" || driverID == "" {
		return
	}
	// Водитель создаёт диалог с peer_id = свой id → 400 "cannot chat with yourself"
	body, _ := json.Marshal(map[string]string{"peer_id": driverID})
	r := req(http.MethodPost, "/v1/chat/conversations", body, baseHeaders(), driverToken)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	common.AssertError(t, rec, http.StatusBadRequest, "yourself")
}
