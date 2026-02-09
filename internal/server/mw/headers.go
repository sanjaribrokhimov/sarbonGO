package mw

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"sarbonNew/internal/config"
	"sarbonNew/internal/server/resp"
)

const (
	HeaderDeviceType  = "X-Device-Type"
	HeaderLanguage    = "X-Language"
	HeaderClientToken = "X-Client-Token"
	HeaderUserToken   = "X-User-Token"
)

func RequireBaseHeaders(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		device := strings.ToLower(strings.TrimSpace(c.GetHeader(HeaderDeviceType)))
		lang := strings.ToLower(strings.TrimSpace(c.GetHeader(HeaderLanguage)))
		clientToken := strings.TrimSpace(c.GetHeader(HeaderClientToken))

		if device == "" || lang == "" || clientToken == "" {
			resp.Error(c, http.StatusBadRequest, "missing required headers: X-Device-Type, X-Language, X-Client-Token")
			c.Abort()
			return
		}

		switch device {
		case "ios", "android", "web":
		default:
			resp.Error(c, http.StatusBadRequest, "invalid X-Device-Type (allowed: ios, android, web)")
			c.Abort()
			return
		}

		switch lang {
		case "ru", "uz", "en", "tr", "zh":
		default:
			resp.Error(c, http.StatusBadRequest, "invalid X-Language (allowed: ru, uz, en, tr, zh)")
			c.Abort()
			return
		}

		if cfg.ClientTokenExpected != "" && clientToken != cfg.ClientTokenExpected {
			resp.Error(c, http.StatusUnauthorized, "invalid X-Client-Token")
			c.Abort()
			return
		}

		c.Next()
	}
}

