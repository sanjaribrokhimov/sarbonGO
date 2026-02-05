// Registration token: одноразовый токен после проверки OTP (register), чтобы потом завершить регистрацию данными водителя.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/redis/go-redis/v9"
)

const regTokenKeyPrefix = "reg_token:"
const regTokenTTL = 15 * time.Minute

// CreateRegistrationToken создаёт токен и сохраняет в Redis (значение = phone), TTL 15 мин.
func CreateRegistrationToken(ctx context.Context, rdb *redis.Client, phone string, ttl time.Duration) (string, error) {
	if ttl <= 0 {
		ttl = regTokenTTL
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	key := regTokenKeyPrefix + token
	if err := rdb.Set(ctx, key, phone, ttl).Err(); err != nil {
		return "", err
	}
	return token, nil
}

// ValidateRegistrationToken возвращает phone по токену и удаляет ключ (одноразовый).
func ValidateRegistrationToken(ctx context.Context, rdb *redis.Client, token string) (phone string, err error) {
	key := regTokenKeyPrefix + token
	phone, err = rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", ErrInvalidToken
		}
		return "", err
	}
	_ = rdb.Del(ctx, key).Err()
	return phone, nil
}
