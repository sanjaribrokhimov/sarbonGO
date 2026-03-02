// Package common — общие хелперы для тестов Driver и Freelance Dispatcher.
// Не зависит от router/config, только от формата ответа API (envelope).
package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// DecodeEnvelope декодирует ответ в envelope (status, code, description, data).
func DecodeEnvelope(rec *httptest.ResponseRecorder) (status string, code int, description string, data map[string]interface{}, err error) {
	var out struct {
		Status      string                 `json:"status"`
		Code        int                    `json:"code"`
		Description string                 `json:"description"`
		Data        map[string]interface{} `json:"data"`
	}
	if err = json.NewDecoder(rec.Body).Decode(&out); err != nil {
		return "", 0, "", nil, err
	}
	return out.Status, out.Code, out.Description, out.Data, nil
}

// AssertSuccess проверяет успешный ответ: HTTP 200, status=success, code=200; при needData — data не nil.
func AssertSuccess(t *testing.T, rec *httptest.ResponseRecorder, needData bool) (data map[string]interface{}) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Errorf("want HTTP 200, got %d body=%s", rec.Code, rec.Body.String())
		return nil
	}
	status, code, desc, data, err := DecodeEnvelope(rec)
	if err != nil {
		t.Fatalf("decode response: %v body=%s", err, rec.Body.String())
	}
	if status != "success" {
		t.Errorf("envelope status: want success, got %q (description=%q)", status, desc)
	}
	if code != http.StatusOK {
		t.Errorf("envelope code: want 200, got %d", code)
	}
	if needData && data == nil {
		t.Error("envelope data: want non-nil")
	}
	return data
}

// AssertError проверяет ответ с ошибкой: HTTP wantCode, status=error, description содержит wantDescSubstr.
func AssertError(t *testing.T, rec *httptest.ResponseRecorder, wantCode int, wantDescSubstr string) {
	t.Helper()
	if rec.Code != wantCode {
		t.Errorf("want HTTP %d, got %d body=%s", wantCode, rec.Code, rec.Body.String())
		return
	}
	status, code, desc, _, err := DecodeEnvelope(rec)
	if err != nil {
		t.Fatalf("decode error response: %v body=%s", err, rec.Body.String())
	}
	if status != "error" {
		t.Errorf("envelope status: want error, got %q", status)
	}
	if code != wantCode {
		t.Errorf("envelope code: want %d, got %d", wantCode, code)
	}
	if wantDescSubstr != "" && !strings.Contains(desc, wantDescSubstr) {
		t.Errorf("envelope description: want substring %q, got %q", wantDescSubstr, desc)
	}
}

// TokensFromData извлекает access_token и refresh_token из data (data.tokens).
func TokensFromData(data map[string]interface{}) (access, refresh string) {
	if data == nil {
		return "", ""
	}
	tokens, _ := data["tokens"].(map[string]interface{})
	if tokens == nil {
		return "", ""
	}
	access, _ = tokens["access_token"].(string)
	refresh, _ = tokens["refresh_token"].(string)
	return access, refresh
}

// DriverFromData извлекает объект driver из data (data.driver).
func DriverFromData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	d, _ := data["driver"].(map[string]interface{})
	return d
}

// DispatcherFromData извлекает объект dispatcher из data (data.dispatcher).
func DispatcherFromData(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	d, _ := data["dispatcher"].(map[string]interface{})
	return d
}
