// Тесты: полный flow грузов — диспетчер создаёт груз, водитель видит, подаёт оффер, диспетчер принимает.
//
// --- Flow этого файла ---
// 1) Dispatcher создаёт груз POST /api/cargo (X-User-Token dispatcher).
// 2) Driver: GET /api/cargo → список, GET /api/cargo/:id → детали.
// 3) Driver: POST /api/cargo/:id/offers (carrier_id=driver.id, price, currency).
// 4) Dispatcher: GET /api/cargo/:id/offers → список офферов, POST /api/offers/:id/accept → принятие.
// 5) Ситуация «отказ»: водитель только list/get, не создаёт оффер.
// 6) Ситуация два водителя: два оффера по одному грузу, accept один → второй rejected.
package cargo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sarbonNew/test/common"
)

func TestCargoFlow_DispatcherCreates_DriverSees_Offer_Accept(t *testing.T) {
	dispToken, _ := createDispatcherToken(t)
	driverToken, driverID := createDriverToken(t)
	if dispToken == "" || driverToken == "" || driverID == "" {
		return
	}

	// 1) Диспетчер создаёт груз
	t.Log("POST /api/cargo (dispatcher)")
	cargoBody, _ := json.Marshal(map[string]interface{}{
		"title":       "Груз тест",
		"weight":      1000,
		"truck_type":  "tent",
		"capacity":    20,
		"route_points": []map[string]interface{}{
			{"type": "load", "address": "Ташкент", "lat": 41.3, "lng": 69.2, "point_order": 1},
			{"type": "unload", "address": "Самарканд", "lat": 39.65, "lng": 66.95, "point_order": 2},
		},
		"payment": map[string]interface{}{"price_request": true},
	})
	r := req(http.MethodPost, "/api/cargo", cargoBody, baseHeaders(), dispToken)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	if rec.Code != http.StatusCreated {
		t.Errorf("POST /api/cargo: want 201, got %d body=%s", rec.Code, rec.Body.String())
		return
	}
	_, _, _, data, _ := common.DecodeEnvelope(rec)
	if data == nil {
		t.Fatal("POST /api/cargo: data nil")
	}
	cargoID, _ := data["id"].(string)
	if cargoID == "" {
		t.Fatal("POST /api/cargo: data.id empty")
	}

	// 2) Водитель видит груз: list и get
	t.Log("GET /api/cargo (driver)")
	r2 := req(http.MethodGet, "/api/cargo?limit=10", nil, baseHeaders(), driverToken)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	data2 := common.AssertSuccess(t, rec2, true)
	if data2 == nil {
		return
	}
	items, _ := data2["items"].([]interface{})
	if len(items) == 0 {
		t.Error("GET /api/cargo: want at least one item")
	}
	t.Log("GET /api/cargo/:id (driver)")
	r3 := req(http.MethodGet, "/api/cargo/"+cargoID, nil, baseHeaders(), driverToken)
	rec3 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec3, r3)
	data3 := common.AssertSuccess(t, rec3, true)
	if data3 == nil {
		return
	}
	if data3["title"] != "Груз тест" {
		t.Errorf("GET /api/cargo/:id title: want Груз тест, got %v", data3["title"])
	}

	// 3) Водитель подаёт оффер
	t.Log("POST /api/cargo/:id/offers (driver)")
	offerBody, _ := json.Marshal(map[string]interface{}{
		"carrier_id": driverID,
		"price":      5000000,
		"currency":   "UZS",
		"comment":    "Готов везти",
	})
	r4 := req(http.MethodPost, "/api/cargo/"+cargoID+"/offers", offerBody, baseHeaders(), driverToken)
	rec4 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec4, r4)
	if rec4.Code != http.StatusCreated {
		t.Errorf("POST /api/cargo/:id/offers: want 201, got %d body=%s", rec4.Code, rec4.Body.String())
		return
	}
	_, _, _, data4, _ := common.DecodeEnvelope(rec4)
	offerID, _ := data4["id"].(string)
	if offerID == "" {
		t.Fatal("POST offers: data.id empty")
	}

	// 4) Диспетчер смотрит офферы и принимает
	t.Log("GET /api/cargo/:id/offers (dispatcher)")
	r5 := req(http.MethodGet, "/api/cargo/"+cargoID+"/offers", nil, baseHeaders(), dispToken)
	rec5 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec5, r5)
	data5 := common.AssertSuccess(t, rec5, true)
	if data5 == nil {
		return
	}
	offers, _ := data5["items"].([]interface{})
	if len(offers) == 0 {
		t.Error("GET offers: want at least one")
	}
	t.Log("POST /api/offers/:id/accept (dispatcher)")
	r6 := req(http.MethodPost, "/api/offers/"+offerID+"/accept", nil, baseHeaders(), dispToken)
	rec6 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec6, r6)
	data6 := common.AssertSuccess(t, rec6, true)
	if data6 == nil {
		return
	}
	if data6["status"] != "accepted" {
		t.Errorf("accept offer: want status=accepted, got %v", data6["status"])
	}

	// 5) Груз в статусе assigned
	r7 := req(http.MethodGet, "/api/cargo/"+cargoID, nil, baseHeaders(), driverToken)
	rec7 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec7, r7)
	data7 := common.AssertSuccess(t, rec7, true)
	if data7 != nil && data7["status"] != "assigned" {
		t.Errorf("cargo status after accept: want assigned, got %v", data7["status"])
	}
}

// Ситуация «отказ»: водитель видит груз, но не подаёт оффер.
func TestCargoFlow_DriverSeesCargo_NoOffer(t *testing.T) {
	dispToken, _ := createDispatcherToken(t)
	driverToken, _ := createDriverToken(t)
	if dispToken == "" || driverToken == "" {
		return
	}
	cargoBody, _ := json.Marshal(map[string]interface{}{
		"title":       "Груз без оффера",
		"weight":      500,
		"truck_type":  "tent",
		"capacity":    10,
		"route_points": []map[string]interface{}{
			{"type": "load", "address": "A", "lat": 41, "lng": 69, "point_order": 1},
			{"type": "unload", "address": "B", "lat": 39, "lng": 66, "point_order": 2},
		},
		"payment": map[string]interface{}{"price_request": true},
	})
	r := req(http.MethodPost, "/api/cargo", cargoBody, baseHeaders(), dispToken)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create cargo: %d %s", rec.Code, rec.Body.String())
	}
	_, _, _, data, _ := common.DecodeEnvelope(rec)
	cargoID, _ := data["id"].(string)
	if cargoID == "" {
		t.Fatal("no cargo id")
	}
	// Водитель только list и get — оффер не создаёт
	r2 := req(http.MethodGet, "/api/cargo", nil, baseHeaders(), driverToken)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	common.AssertSuccess(t, rec2, true)
	r3 := req(http.MethodGet, "/api/cargo/"+cargoID, nil, baseHeaders(), driverToken)
	rec3 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec3, r3)
	data3 := common.AssertSuccess(t, rec3, true)
	if data3 != nil && data3["status"] != "created" && data3["status"] != "searching" {
		t.Logf("cargo status (no offer): %v", data3["status"])
	}
	// Список офферов пустой
	r4 := req(http.MethodGet, "/api/cargo/"+cargoID+"/offers", nil, baseHeaders(), dispToken)
	rec4 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec4, r4)
	data4 := common.AssertSuccess(t, rec4, true)
	items, _ := data4["items"].([]interface{})
	if len(items) != 0 {
		t.Errorf("offers without driver offer: want 0, got %d", len(items))
	}
}

// Два водителя подают офферы; диспетчер принимает один — второй автоматически rejected.
func TestCargoFlow_TwoOffers_AcceptOne(t *testing.T) {
	dispToken, _ := createDispatcherToken(t)
	driver1Token, driver1ID := createDriverToken(t)
	driver2Token, driver2ID := createDriverToken(t)
	if dispToken == "" || driver1ID == "" || driver2ID == "" {
		return
	}
	cargoBody, _ := json.Marshal(map[string]interface{}{
		"title":       "Груз два оффера",
		"weight":      2000,
		"truck_type":  "tent",
		"capacity":    25,
		"route_points": []map[string]interface{}{
			{"type": "load", "address": "X", "lat": 41, "lng": 69, "point_order": 1},
			{"type": "unload", "address": "Y", "lat": 39, "lng": 66, "point_order": 2},
		},
		"payment": map[string]interface{}{"price_request": true},
	})
	r := req(http.MethodPost, "/api/cargo", cargoBody, baseHeaders(), dispToken)
	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, r)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create cargo: %d", rec.Code)
	}
	_, _, _, data, _ := common.DecodeEnvelope(rec)
	cargoID, _ := data["id"].(string)
	if cargoID == "" {
		t.Fatal("no cargo id")
	}
	// Оффер от водителя 1
	offer1, _ := json.Marshal(map[string]interface{}{"carrier_id": driver1ID, "price": 1000, "currency": "UZS"})
	r1 := req(http.MethodPost, "/api/cargo/"+cargoID+"/offers", offer1, baseHeaders(), driver1Token)
	rec1 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec1, r1)
	if rec1.Code != http.StatusCreated {
		t.Fatalf("offer 1: %d", rec1.Code)
	}
	_, _, _, d1, _ := common.DecodeEnvelope(rec1)
	offer1ID, _ := d1["id"].(string)
	// Оффер от водителя 2
	offer2, _ := json.Marshal(map[string]interface{}{"carrier_id": driver2ID, "price": 1200, "currency": "UZS"})
	r2 := req(http.MethodPost, "/api/cargo/"+cargoID+"/offers", offer2, baseHeaders(), driver2Token)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	if rec2.Code != http.StatusCreated {
		t.Fatalf("offer 2: %d", rec2.Code)
	}
	_, _, _, d2, _ := common.DecodeEnvelope(rec2)
	offer2ID, _ := d2["id"].(string)
	// Принимаем первый оффер
	r3 := req(http.MethodPost, "/api/offers/"+offer1ID+"/accept", nil, baseHeaders(), dispToken)
	rec3 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec3, r3)
	common.AssertSuccess(t, rec3, true)
	// Список офферов: один accepted, второй rejected
	r4 := req(http.MethodGet, "/api/cargo/"+cargoID+"/offers", nil, baseHeaders(), dispToken)
	rec4 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec4, r4)
	data4 := common.AssertSuccess(t, rec4, true)
	items, _ := data4["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("offers count: want 2, got %d", len(items))
	}
	var accepted, rejected int
	for _, it := range items {
		m, _ := it.(map[string]interface{})
		s, _ := m["status"].(string)
		if s == "accepted" {
			accepted++
		}
		if s == "rejected" {
			rejected++
		}
	}
	if accepted != 1 || rejected != 1 {
		t.Errorf("after accept one: want 1 accepted, 1 rejected; got accepted=%d rejected=%d", accepted, rejected)
	}
	// Повторный accept того же оффера или второго — ошибка (offer not found or not pending)
	r5 := req(http.MethodPost, "/api/offers/"+offer2ID+"/accept", nil, baseHeaders(), dispToken)
	rec5 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec5, r5)
	if rec5.Code == http.StatusOK {
		t.Error("accept already-rejected offer: want error")
	}
}
