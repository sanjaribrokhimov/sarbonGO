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

