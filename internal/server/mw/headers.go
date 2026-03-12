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
	HeaderUserID      = "X-User-ID" // optional; for chat Swagger testing — overrides JWT user when set
)

func RequireBaseHeaders(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		device := strings.ToLower(strings.TrimSpace(c.GetHeader(HeaderDeviceType)))
		lang := strings.ToLower(strings.TrimSpace(c.GetHeader(HeaderLanguage)))
		clientToken := strings.TrimSpace(c.GetHeader(HeaderClientToken))

		if device == "" || lang == "" || clientToken == "" {
			resp.ErrorLang(c, http.StatusBadRequest, "missing_required_headers")
			c.Abort()
			return
		}

		switch device {
		case "ios", "android", "web":
		default:
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_x_device_type")
			c.Abort()
			return
		}

		switch lang {
		case "ru", "uz", "en", "tr", "zh":
		default:
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_x_language")
			c.Abort()
			return
		}

		if cfg.ClientTokenExpected != "" && clientToken != cfg.ClientTokenExpected {
			resp.ErrorLang(c, http.StatusUnauthorized, "invalid_x_client_token")
			c.Abort()
			return
		}

		c.Next()
	}
}

