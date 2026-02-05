// Ключи контекста и геттеры для request_id, язык, тип клиента, платформа, версия приложения, user_id.
package middleware

import "context"

type contextKey string

const (
	ContextKeyRequestID  contextKey = "request_id"
	ContextKeyLanguage   contextKey = "language"
	ContextKeyClientType contextKey = "client_type"
	ContextKeyPlatform   contextKey = "platform"
	ContextKeyAppVersion contextKey = "app_version"
	ContextKeyUserID     contextKey = "user_id" // заполняется AuthMiddleware
)

// UserIDFrom возвращает ID пользователя из контекста (после AuthMiddleware).
func UserIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyUserID).(string); ok {
		return v
	}
	return ""
}

// RequestIDFrom возвращает X-Request-ID из контекста.
func RequestIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyRequestID).(string); ok {
		return v
	}
	return ""
}

// LanguageFrom возвращает язык из контекста (Accept-Language); по умолчанию "en".
func LanguageFrom(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyLanguage).(string); ok {
		return v
	}
	return "en"
}

// ClientTypeFrom возвращает тип клиента (frontend/mobile) из контекста.
func ClientTypeFrom(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyClientType).(string); ok {
		return v
	}
	return ""
}

// PlatformFrom возвращает платформу (web/ios/android) из контекста.
func PlatformFrom(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyPlatform).(string); ok {
		return v
	}
	return ""
}

// AppVersionFrom возвращает версию приложения (x.y.z) из контекста.
func AppVersionFrom(ctx context.Context) string {
	if v, ok := ctx.Value(ContextKeyAppVersion).(string); ok {
		return v
	}
	return ""
}
