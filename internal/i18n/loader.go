// i18n: загрузка переводов из встроенных JSON (ru, en, uz, tr, zh); язык только из Accept-Language.
package i18n

import (
	"embed"
	"encoding/json"
	"sync"
)

//go:embed ru/*.json en/*.json uz/*.json tr/*.json zh/*.json
var fs embed.FS

var (
	mu    sync.RWMutex
	packs = make(map[string]map[string]string) // язык -> ключ -> сообщение
)

// Поддерживаемые языки (только эти; язык берётся только из заголовка Accept-Language).
const (
	LangRU = "ru"
	LangEN = "en"
	LangUZ = "uz"
	LangTR = "tr"
	LangZH = "zh"
)

// Load загружает встроенные JSON по каждому языку; ключи: error.unauthorized, error.forbidden и т.д.
func Load() error {
	mu.Lock()
	defer mu.Unlock()
	for _, lang := range []string{LangRU, LangEN, LangUZ, LangTR, LangZH} {
		data, err := fs.ReadFile(lang + "/messages.json")
		if err != nil {
			packs[lang] = defaultMessages(lang)
			continue
		}
		var m map[string]string
		if err := json.Unmarshal(data, &m); err != nil {
			packs[lang] = defaultMessages(lang)
			continue
		}
		packs[lang] = m
	}
	return nil
}

// defaultMessages — запасные сообщения при отсутствии или ошибке JSON; ключи для ошибок и ответов.
func defaultMessages(lang string) map[string]string {
	return map[string]string{
		"error.unauthorized":  "unauthorized",
		"error.forbidden":     "forbidden",
		"error.not_found":     "not found",
		"error.rate_limit":    "rate limit exceeded",
		"error.internal":     "internal server error",
		"ok":                  "ok",
	}
}

// T возвращает сообщение по языку и ключу; при отсутствии — fallback на en, иначе сам ключ.
func T(lang, key string) string {
	mu.RLock()
	defer mu.RUnlock()
	if m, ok := packs[lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if m, ok := packs[LangEN]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	return key
}
