package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GatewayClient struct {
	baseURL string
	token   string
	sender  string
	http    *http.Client
}

func NewGatewayClient(baseURL, token, sender string) *GatewayClient {
	return &GatewayClient{
		baseURL: baseURL,
		token:   token,
		sender:  sender,
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
	OK     bool          `json:"ok"`
	Result requestStatus `json:"result"`
	Error  string        `json:"error"`
}

func (c *GatewayClient) SendVerificationMessage(ctx context.Context, phoneE164 string, code string, ttlSeconds int) (requestID string, err error) {
	if c.token == "" {
		return "", fmt.Errorf("telegram gateway token is not configured")
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
		return "", err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("telegram gateway http %d: %s", resp.StatusCode, string(raw))
	}

	var gr gatewayResp
	if err := json.Unmarshal(raw, &gr); err != nil {
		return "", fmt.Errorf("telegram gateway decode error: %w", err)
	}
	if !gr.OK {
		return "", fmt.Errorf("telegram gateway error: %s", gr.Error)
	}
	if gr.Result.RequestID == "" {
		return "", fmt.Errorf("telegram gateway: empty request_id")
	}
	return gr.Result.RequestID, nil
}

