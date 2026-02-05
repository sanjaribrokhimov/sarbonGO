// Telegram Gateway API: отправка кода верификации по номеру телефона (без chat_id).
// Документация: https://core.telegram.org/gateway/api
package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const gatewayAPIURL = "https://gatewayapi.telegram.org/sendVerificationMessage"

// SendOTPViaGateway отправляет OTP через Telegram Gateway API (Verification Codes).
// Номер в формате E.164, code — только цифры 4–8 символов. chat_id не нужен.
// Токен берётся с https://gateway.telegram.org/account/api
func SendOTPViaGateway(gatewayToken, phone, code string, ttlSec int) error {
	if gatewayToken == "" || phone == "" || code == "" {
		return fmt.Errorf("gateway token, phone and code required")
	}
	if ttlSec < 30 {
		ttlSec = 30
	}
	if ttlSec > 3600 {
		ttlSec = 3600
	}
	body := map[string]interface{}{
		"phone_number": phone,
		"code":         code,
		"ttl":          ttlSec,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, gatewayAPIURL, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+gatewayToken)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	var out struct {
		OK     bool   `json:"ok"`
		Error  string `json:"error,omitempty"`
		Result *struct {
			RequestID string `json:"request_id"`
		} `json:"result,omitempty"`
	}
	_ = json.Unmarshal(rb, &out)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("gateway api %d: %s", resp.StatusCode, string(rb))
	}
	if !out.OK {
		return fmt.Errorf("gateway api error: %s", out.Error)
	}
	return nil
}
