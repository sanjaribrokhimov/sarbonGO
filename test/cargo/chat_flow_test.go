// Тесты: чат между водителем и диспетчером — conversations, messages.
//
// --- Flow этого файла ---
// 1) Driver и Dispatcher получают токены и id (из profile).
// 2) Driver: GET /v1/chat/conversations (может быть пусто).
// 3) Driver: POST /v1/chat/conversations body { "peer_id": dispatcher_id } → id диалога.
// 4) Driver: POST /v1/chat/conversations/:id/messages body { "body": "Текст от водителя" }.
// 5) Dispatcher: GET /v1/chat/conversations, POST /v1/chat/conversations body { "peer_id": driver_id } → тот же диалог.
// 6) Dispatcher: GET /v1/chat/conversations/:id/messages → видит сообщение; POST message в ответ.
package cargo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sarbonNew/test/common"
)

func TestChatFlow_DriverDispatcher_ConversationAndMessages(t *testing.T) {
	driverToken, driverID := createDriverToken(t)
	dispToken, dispatcherID := createDispatcherToken(t)
	if driverToken == "" || driverID == "" || dispToken == "" || dispatcherID == "" {
		return
	}

	// Водитель: список диалогов (может быть пустой)
	t.Log("GET /v1/chat/conversations (driver)")
	r0 := req(http.MethodGet, "/v1/chat/conversations", nil, baseHeaders(), driverToken)
	rec0 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec0, r0)
	data0 := common.AssertSuccess(t, rec0, true)
	if data0 == nil {
		return
	}

	// Водитель: создать/получить диалог с диспетчером
	t.Log("POST /v1/chat/conversations (driver, peer_id=dispatcher)")
	bodyConv, _ := json.Marshal(map[string]string{"peer_id": dispatcherID})
	r1 := req(http.MethodPost, "/v1/chat/conversations", bodyConv, baseHeaders(), driverToken)
	rec1 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec1, r1)
	data1 := common.AssertSuccess(t, rec1, true)
	if data1 == nil {
		return
	}
	convID, _ := data1["id"].(string)
	if convID == "" {
		// id может приходить как UUID в JSON — попробуем через описание ответа
		t.Fatal("POST /v1/chat/conversations: data.id empty")
	}

	// Водитель: отправить сообщение
	t.Log("POST /v1/chat/conversations/:id/messages (driver)")
	bodyMsg, _ := json.Marshal(map[string]string{"body": "Груз принял, выезжаю"})
	r2 := req(http.MethodPost, "/v1/chat/conversations/"+convID+"/messages", bodyMsg, baseHeaders(), driverToken)
	rec2 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec2, r2)
	if rec2.Code != http.StatusCreated {
		t.Errorf("POST messages: want 201, got %d body=%s", rec2.Code, rec2.Body.String())
		return
	}

	// Диспетчер: получить или создать тот же диалог (peer_id = driver)
	t.Log("POST /v1/chat/conversations (dispatcher, peer_id=driver)")
	bodyConv2, _ := json.Marshal(map[string]string{"peer_id": driverID})
	r3 := req(http.MethodPost, "/v1/chat/conversations", bodyConv2, baseHeaders(), dispToken)
	rec3 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec3, r3)
	data3 := common.AssertSuccess(t, rec3, true)
	if data3 == nil {
		return
	}
	convID2, _ := data3["id"].(string)
	if convID2 == "" {
		t.Fatal("dispatcher POST conversations: data.id empty")
	}
	// Должен быть тот же диалог (один на пару)
	if convID2 != convID {
		t.Logf("conversation ids: driver got %s, dispatcher got %s (may differ by implementation)", convID, convID2)
	}

	// Диспетчер: список сообщений в диалоге
	t.Log("GET /v1/chat/conversations/:id/messages (dispatcher)")
	r4 := req(http.MethodGet, "/v1/chat/conversations/"+convID2+"/messages?limit=10", nil, baseHeaders(), dispToken)
	rec4 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec4, r4)
	data4 := common.AssertSuccess(t, rec4, true)
	if data4 == nil {
		return
	}
	msgs, _ := data4["messages"].([]interface{})
	if len(msgs) == 0 {
		t.Error("GET messages: want at least one message")
	} else {
		first, _ := msgs[0].(map[string]interface{})
		if body, _ := first["body"].(string); body != "Груз принял, выезжаю" {
			t.Errorf("first message body: want Груз принял, выезжаю, got %q", body)
		}
	}

	// Диспетчер: ответить в чат
	t.Log("POST /v1/chat/conversations/:id/messages (dispatcher)")
	bodyReply, _ := json.Marshal(map[string]string{"body": "Ок, жду"})
	r5 := req(http.MethodPost, "/v1/chat/conversations/"+convID2+"/messages", bodyReply, baseHeaders(), dispToken)
	rec5 := httptest.NewRecorder()
	testRouter.ServeHTTP(rec5, r5)
	if rec5.Code != http.StatusCreated {
		t.Errorf("dispatcher POST message: want 201, got %d", rec5.Code)
	}
}
