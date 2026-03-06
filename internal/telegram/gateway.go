package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const maxGatewayRawLogBytes = 4096

type GatewayClient struct {
	baseURL string
	token   string
	sender  string
	bypass  bool
	http    *http.Client
}

func NewGatewayClient(baseURL, token, sender string, bypass bool) *GatewayClient {
	return &GatewayClient{
		baseURL: baseURL,
		token:   token,
		sender:  sender,
		bypass:  bypass,
		http: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

type sendVerificationMessageReq struct {
	PhoneNumber    string `json:"phone_number"`
	Code           string `json:"code,omitempty"`
	CodeLength     int    `json:"code_length,omitempty"`
	TTL            int    `json:"ttl,omitempty"`
	Payload        string `json:"payload,omitempty"`
	SenderUsername string `json:"sender_username,omitempty"`
}

type requestStatus struct {
	RequestID string `json:"request_id"`
}

type gatewayResp struct {
	OK           bool          `json:"ok"`
	Result       requestStatus `json:"result"`
	Error        string        `json:"error"`
	ErrorMessage string        `json:"error_message"`
	Message      string        `json:"message"`
	Reason       string        `json:"reason"`
	Code         string        `json:"code"`
}

func (c *GatewayClient) SendVerificationMessage(ctx context.Context, phoneE164 string, code string, ttlSeconds int) (requestID string, err error) {
	if c.bypass {
		log.Printf("telegram gateway bypass: phone=%s code=%s ttl=%ds", phoneE164, code, ttlSeconds)
		return "bypass", nil
	}
	if c.token == "" {
		return "", &GatewayError{Message: "telegram gateway token is not configured"}
	}

	body := sendVerificationMessageReq{
		PhoneNumber: phoneE164,
		Code:        code,
		TTL:         ttlSeconds,
		Payload:     "sarbon-otp",
	}
	if c.sender != "" {
		body.SenderUsername = c.sender
	}

	b, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sendVerificationMessage", bytes.NewReader(b))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", &GatewayError{Message: err.Error()}
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var gr gatewayResp
	_ = json.Unmarshal(raw, &gr)
	errMsg := firstNonEmpty(gr.Error, gr.ErrorMessage, gr.Message, gr.Reason, gr.Code)
	if resp.StatusCode != http.StatusOK {
		if errMsg == "" {
			errMsg = string(raw)
		}
		log.Printf("telegram gateway raw response (http!=200): status=%d body=%s", resp.StatusCode, truncateForLog(string(raw), maxGatewayRawLogBytes))
		kind := error(nil)
		if isNoAccountError(errMsg) {
			kind = ErrNoAccount
		} else if isRateLimitError(errMsg) {
			kind = ErrRateLimited
		}
		return "", &GatewayError{Kind: kind, StatusCode: resp.StatusCode, Message: strings.TrimSpace(errMsg), RawBody: truncateForLog(string(raw), maxGatewayRawLogBytes)}
	}

	if err := json.Unmarshal(raw, &gr); err != nil {
		log.Printf("telegram gateway raw response (decode error): status=%d body=%s", resp.StatusCode, truncateForLog(string(raw), maxGatewayRawLogBytes))
		return "", &GatewayError{StatusCode: resp.StatusCode, Message: "telegram gateway decode error", RawBody: truncateForLog(string(raw), maxGatewayRawLogBytes)}
	}
	if !gr.OK {
		errMsg = firstNonEmpty(gr.Error, gr.ErrorMessage, gr.Message, gr.Reason, gr.Code)
		log.Printf("telegram gateway raw response (ok=false): status=%d body=%s", resp.StatusCode, truncateForLog(string(raw), maxGatewayRawLogBytes))
		kind := error(nil)
		if isNoAccountError(errMsg) {
			kind = ErrNoAccount
		} else if isRateLimitError(errMsg) {
			kind = ErrRateLimited
		}
		return "", &GatewayError{Kind: kind, StatusCode: resp.StatusCode, Message: strings.TrimSpace(errMsg), RawBody: truncateForLog(string(raw), maxGatewayRawLogBytes)}
	}
	if gr.Result.RequestID == "" {
		log.Printf("telegram gateway raw response (empty request_id): status=%d body=%s", resp.StatusCode, truncateForLog(string(raw), maxGatewayRawLogBytes))
		return "", &GatewayError{StatusCode: resp.StatusCode, Message: "telegram gateway empty request_id", RawBody: truncateForLog(string(raw), maxGatewayRawLogBytes)}
	}
	return gr.Result.RequestID, nil
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func truncateForLog(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "...(truncated)"
}

// isNoAccountError returns true if the gateway error indicates the phone has no Telegram account.
func isNoAccountError(errText string) bool {
	s := strings.ToLower(strings.TrimSpace(errText))
	if s == "" {
		return false
	}
	patterns := []string{
		"no account", "no telegram", "user not found", "user_not_found", "not found",
		"phone_number_unoccupied", "phone_number_invalid", "no_account",
		"phone_number_not_available", "not_available",
		"invalid phone", "unoccupied", "not registered",
		"contact not found", "contact_not_found", "recipient not found",
		"account not found", "phone not found",
		"peer_not_found", "user_not_registered", "not_registered",
		"нет аккаунта", "аккаунт не найден", "не найден", "пользователь не найден",
	}
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

// isRateLimitError returns true if the gateway error is Telegram rate limit (e.g. FLOOD_WAIT_1038).
func isRateLimitError(errText string) bool {
	s := strings.ToLower(strings.TrimSpace(errText))
	if s == "" {
		return false
	}
	return strings.HasPrefix(s, "flood_wait") || strings.Contains(s, "flood_wait") ||
		strings.Contains(s, "rate_limit") || strings.Contains(s, "too many requests")
}
