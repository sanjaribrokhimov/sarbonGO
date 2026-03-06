package telegram

import "errors"

// ErrNoAccount means the phone number has no Telegram account (user not found / not registered).
// Handlers should return HTTP 400.
var ErrNoAccount = errors.New("no account")

// ErrRateLimited means Telegram gateway returned FLOOD_WAIT_* or similar rate limit.
// Handlers should return HTTP 429 with gateway message in description.
var ErrRateLimited = errors.New("rate limited")

// GatewayError is a normalized error returned by Telegram Gateway client.
// Message is suitable to return to API clients in resp.Envelope.description.
type GatewayError struct {
	Kind       error  // optional classification, e.g. ErrNoAccount
	StatusCode int    // HTTP status from gateway, if available
	Message    string // parsed error like PHONE_NUMBER_NOT_AVAILABLE
	RawBody    string // raw response body (may be empty/truncated)
}

func (e *GatewayError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Kind != nil {
		return e.Kind.Error()
	}
	return "telegram gateway error"
}

func (e *GatewayError) Unwrap() error { return e.Kind }
