package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/dispatchers"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
	"sarbonNew/internal/util"
)

type DispatcherRegistrationHandler struct {
	logger   *zap.Logger
	repo     *dispatchers.Repo
	sessions *store.DispatcherSessionStore
	jwtm     *security.JWTManager
	refresh  *store.RefreshStore
}

func NewDispatcherRegistrationHandler(logger *zap.Logger, repo *dispatchers.Repo, sessions *store.DispatcherSessionStore, jwtm *security.JWTManager, refresh *store.RefreshStore) *DispatcherRegistrationHandler {
	return &DispatcherRegistrationHandler{logger: logger, repo: repo, sessions: sessions, jwtm: jwtm, refresh: refresh}
}

type dispCompleteReq struct {
	SessionID      string  `json:"session_id" binding:"required"`
	Name           string  `json:"name" binding:"required"`
	Password       string  `json:"password" binding:"required"`
	PassportSeries string  `json:"passport_series" binding:"required"`
	PassportNumber string  `json:"passport_number" binding:"required"`
	PINFL          string  `json:"pinfl" binding:"required"`
	Photo          *string `json:"photo,omitempty"`
}

func (h *DispatcherRegistrationHandler) Complete(c *gin.Context) {
	var req dispCompleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	if err := util.ValidatePassword(req.Password); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	phone, err := h.sessions.Consume(c.Request.Context(), strings.TrimSpace(req.SessionID))
	if err != nil {
		if errors.Is(err, store.ErrDispatcherSessionNotFound) {
			resp.Error(c, http.StatusUnauthorized, "session expired or invalid")
			return
		}
		h.logger.Error("consume session failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	existing, err := h.repo.FindByPhone(c.Request.Context(), phone)
	if err == nil {
		id, _ := uuid.Parse(existing.ID)
		tokens, refreshClaims, err := h.jwtm.Issue("dispatcher", id)
		if err != nil {
			resp.Error(c, http.StatusInternalServerError, "token issue failed")
			return
		}
		_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
		resp.OK(c, gin.H{"status": "login", "tokens": tokens, "dispatcher": existing})
		return
	}
	if !errors.Is(err, dispatchers.ErrNotFound) {
		h.logger.Error("find by phone failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	pwHash, err := util.HashPassword(req.Password)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "password hash failed")
		return
	}
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 {
		resp.Error(c, http.StatusBadRequest, "name is too short")
		return
	}
	ps := strings.TrimSpace(req.PassportSeries)
	pn := strings.TrimSpace(req.PassportNumber)
	pinfl := strings.TrimSpace(req.PINFL)
	if ps == "" || pn == "" || pinfl == "" {
		resp.Error(c, http.StatusBadRequest, "passport_series, passport_number, pinfl are required")
		return
	}
	var photo *string
	if req.Photo != nil {
		v := strings.TrimSpace(*req.Photo)
		if v != "" {
			photo = &v
		}
	}

	id, err := h.repo.Create(c.Request.Context(), dispatchers.CreateParams{
		Phone:          phone,
		Name:           name,
		PasswordHash:   pwHash,
		PassportSeries: ps,
		PassportNumber: pn,
		PINFL:          pinfl,
		Photo:          photo,
	})
	if err != nil {
		if errors.Is(err, dispatchers.ErrPhoneAlreadyRegistered) {
			resp.Error(c, http.StatusConflict, "this phone is already registered")
			return
		}
		h.logger.Error("dispatcher create failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	tokens, refreshClaims, err := h.jwtm.Issue("dispatcher", id)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "token issue failed")
		return
	}
	_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)

	disp, _ := h.repo.FindByID(c.Request.Context(), id)
	resp.OK(c, gin.H{"status": "registered", "tokens": tokens, "dispatcher": disp})
}
