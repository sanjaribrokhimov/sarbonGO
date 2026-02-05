// Middleware: язык только из заголовка Accept-Language (не из query/body/cookies).
package middleware

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sarbonGO/backend/internal/response"
)

// Допустимые языки — только из Accept-Language; не из query, body или cookies.
var supportedLanguages = map[string]bool{
	"ru": true, "en": true, "uz": true, "tr": true, "zh": true,
}

const HeaderAcceptLanguage = "Accept-Language"

// LanguageMiddleware проверяет Accept-Language; допускаются только ru, en, uz, tr, zh; иначе 403.
// Для /api/v1/auth/* при отсутствии заголовка подставляется ru.
func LanguageMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader(HeaderAcceptLanguage)
		if raw == "" {
			if strings.HasPrefix(c.Request.URL.Path, "/api/v1/auth") {
				raw = "ru"
			} else {
				response.AbortWithError(c, 403, "missing Accept-Language header")
				return
			}
		}
		// Берём только первый тег (например "ru" из "ru-RU,en;q=0.9").
		lang := parseAcceptLanguage(raw)
		if !supportedLanguages[lang] {
			response.AbortWithError(c, 403, "unsupported Accept-Language; allowed: ru, en, uz, tr, zh")
			return
		}
		c.Set(string(ContextKeyLanguage), lang)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ContextKeyLanguage, lang))
		c.Next()
	}
}

// parseAcceptLanguage извлекает первый языковой тег из значения заголовка.
func parseAcceptLanguage(h string) string {
	for i, r := range h {
		if r == '-' || r == ',' || r == ';' || r == ' ' {
			if i > 0 {
				return h[:i]
			}
			break
		}
	}
	if len(h) >= 2 {
		return h[:2]
	}
	return h
}

// IsLanguageSupported возвращает true для ru, en, uz, tr, zh.
func IsLanguageSupported(lang string) bool {
	return supportedLanguages[lang]
}
