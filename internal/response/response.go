// Package response provides a unified API response format: status (HTTP code), message, data.
package response

import (
	"github.com/gin-gonic/gin"
)

// Body is the unified response structure for all API responses.
type Body struct {
	Status  int         `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// Success sends a successful response with status code, message and optional data.
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	if message == "" {
		message = "success"
	}
	c.JSON(statusCode, Body{Status: statusCode, Message: message, Data: data})
}

// Error sends an error response with status code and message; data is nil.
func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, Body{Status: statusCode, Message: message, Data: nil})
}

// AbortWithError aborts the chain and sends the unified error response (for middleware).
func AbortWithError(c *gin.Context, statusCode int, message string) {
	c.AbortWithStatusJSON(statusCode, Body{Status: statusCode, Message: message, Data: nil})
}

// Common messages.
const (
	MsgSuccess = "success"
	MsgCreated = "created"
)
