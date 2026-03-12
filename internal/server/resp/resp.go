package resp

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope is the unified response structure for ALL API endpoints.
type Envelope struct {
	Status      string `json:"status"`      // success | error
	Code        int    `json:"code"`        // usually HTTP status code
	Description string `json:"description"` // human readable
	Data        any    `json:"data"`        // object | array | null
}

func Success(c *gin.Context, httpCode int, description string, data any) {
	c.JSON(httpCode, Envelope{
		Status:      "success",
		Code:        httpCode,
		Description: description,
		Data:        data,
	})
}

func OK(c *gin.Context, data any) {
	Success(c, http.StatusOK, "ok", data)
}

func Error(c *gin.Context, httpCode int, description string) {
	c.JSON(httpCode, Envelope{
		Status:      "error",
		Code:        httpCode,
		Description: description,
		Data:        nil,
	})
}

// ErrorWithData sends error response with optional data (e.g. limit, current count).
func ErrorWithData(c *gin.Context, httpCode int, description string, data any) {
	c.JSON(httpCode, Envelope{
		Status:      "error",
		Code:        httpCode,
		Description: description,
		Data:        data,
	})
}

// Lang returns X-Language from request (ru, uz, en, tr, zh). Default "en".
func Lang(c *gin.Context) string {
	return LangFromContext(c)
}

// OKLang sends success 200 with description by message key and X-Language. status stays "success" (English).
func OKLang(c *gin.Context, messageKey string, data any) {
	desc := Msg(messageKey, Lang(c))
	Success(c, http.StatusOK, desc, data)
}

// SuccessLang sends success with code and localized description by key.
func SuccessLang(c *gin.Context, httpCode int, messageKey string, data any) {
	desc := Msg(messageKey, Lang(c))
	Success(c, httpCode, desc, data)
}

// ErrorLang sends error response with description by message key and X-Language. status stays "error" (English).
func ErrorLang(c *gin.Context, httpCode int, messageKey string) {
	desc := Msg(messageKey, Lang(c))
	Error(c, httpCode, desc)
}

