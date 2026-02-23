package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"sarbonNew/internal/admins"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
	"sarbonNew/internal/util"
)

type AdminAuthHandler struct {
	logger  *zap.Logger
	repo    *admins.Repo
	jwtm    *security.JWTManager
	refresh *store.RefreshStore
}

func NewAdminAuthHandler(logger *zap.Logger, repo *admins.Repo, jwtm *security.JWTManager, refresh *store.RefreshStore) *AdminAuthHandler {
	return &AdminAuthHandler{logger: logger, repo: repo, jwtm: jwtm, refresh: refresh}
}

type adminLoginPasswordReq struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AdminAuthHandler) LoginPassword(c *gin.Context) {
	var req adminLoginPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	login := strings.TrimSpace(req.Login)
	pw := strings.TrimSpace(req.Password)
	if login == "" || pw == "" {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}

	a, err := h.repo.FindByLogin(c.Request.Context(), login)
	if err != nil {
		if errors.Is(err, admins.ErrNotFound) {
			resp.Error(c, http.StatusUnauthorized, "invalid login or password")
			return
		}
		h.logger.Error("admin findByLogin failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	if strings.ToLower(strings.TrimSpace(a.Status)) != "active" {
		resp.Error(c, http.StatusUnauthorized, "admin inactive")
		return
	}
	if !util.ComparePassword(a.Password, pw) {
		resp.Error(c, http.StatusUnauthorized, "invalid login or password")
		return
	}

	tokens, refreshClaims, err := h.jwtm.Issue("admin", a.ID)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "token issue failed")
		return
	}
	_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)

	resp.OK(c, gin.H{
		"tokens": tokens,
		"admin": gin.H{
			"id":     a.ID,
			"login":  a.Login,
			"name":   a.Name,
			"status": a.Status,
			"type":   a.Type,
		},
	})
}

