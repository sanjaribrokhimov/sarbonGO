// HTTP-обработчики; ответы готовы к i18n (язык из контекста).
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sarbonGO/backend/internal/i18n"
	"github.com/sarbonGO/backend/internal/middleware"
	"github.com/sarbonGO/backend/internal/response"
)

// Health возвращает 200 в едином формате (status, message, data).
func Health(c *gin.Context) {
	lang := middleware.LanguageFrom(c.Request.Context())
	msg := i18n.T(lang, "ok")
	response.Success(c, http.StatusOK, msg, nil)
}

// StatusCodeItem — код и сообщение для справочника.
type StatusCodeItem struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// statusCodesList — все коды статуса API с сообщениями (один GET для справки).
var statusCodesList = []StatusCodeItem{
	{http.StatusOK, "success"},
	{http.StatusCreated, "created"},
	{http.StatusBadRequest, "bad request"},
	{http.StatusUnauthorized, "unauthorized"},
	{http.StatusForbidden, "forbidden"},
	{http.StatusNotFound, "not found"},
	{http.StatusTooManyRequests, "too many requests"},
	{http.StatusInternalServerError, "internal server error"},
}

// StatusCodes возвращает список всех кодов статуса и сообщений (GET /status-codes).
func StatusCodes(c *gin.Context) {
	response.Success(c, http.StatusOK, response.MsgSuccess, statusCodesList)
}
