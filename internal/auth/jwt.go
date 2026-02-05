// JWT: создание и проверка токена (HMAC-SHA256), без внешних зависимостей.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const defaultExpire = 30 * 24 * time.Hour // 30 дней

// CreateToken создаёт JWT с claim sub = userID и exp.
func CreateToken(secret, userID string, expire time.Duration) (string, error) {
	if expire <= 0 {
		expire = defaultExpire
	}
	exp := time.Now().Add(expire).Unix()
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	payload := map[string]interface{}{"sub": userID, "exp": exp}
	hdrB, _ := json.Marshal(header)
	payB, _ := json.Marshal(payload)
	b64H := base64.RawURLEncoding.EncodeToString(hdrB)
	b64P := base64.RawURLEncoding.EncodeToString(payB)
	unsigned := b64H + "." + b64P
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + sig, nil
}

// ValidateToken проверяет подпись и exp, возвращает sub (userID).
func ValidateToken(secret, token string) (userID string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", ErrInvalidToken
	}
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(parts[2]), []byte(expected)) {
		return "", ErrInvalidToken
	}
	payB, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrInvalidToken
	}
	var payload struct {
		Sub string `json:"sub"`
		Exp int64  `json:"exp"`
	}
	if err := json.Unmarshal(payB, &payload); err != nil {
		return "", ErrInvalidToken
	}
	if time.Now().Unix() > payload.Exp {
		return "", ErrExpired
	}
	if payload.Sub == "" {
		return "", ErrInvalidToken
	}
	return payload.Sub, nil
}

// UserIDFromToken — обёртка для middleware (возвращает только userID по secret и token).
func UserIDFromToken(secret, token string) (string, error) {
	return ValidateToken(secret, token)
}

// ErrExpired — токен просрочен.
var ErrExpired = errors.New("token expired")