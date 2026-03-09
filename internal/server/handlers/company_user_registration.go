package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/appusers"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/store"
	"sarbonNew/internal/util"
)

const companyUserRoleOwner = "OWNER"

type CompanyUserRegistrationHandler struct {
	logger  *zap.Logger
	repo    *appusers.Repo
	sessions *store.DispatcherSessionStore
	jwtm    *security.JWTManager
	refresh *store.RefreshStore
}

func NewCompanyUserRegistrationHandler(
	logger *zap.Logger,
	repo *appusers.Repo,
	sessions *store.DispatcherSessionStore,
	jwtm *security.JWTManager,
	refresh *store.RefreshStore,
) *CompanyUserRegistrationHandler {
	return &CompanyUserRegistrationHandler{
		logger:   logger,
		repo:     repo,
		sessions: sessions,
		jwtm:     jwtm,
		refresh:  refresh,
	}
}

type companyUserCompleteReq struct {
	SessionID string `json:"session_id" binding:"required"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name"`
	Password  string `json:"password" binding:"required"`
}

func (h *CompanyUserRegistrationHandler) Complete(c *gin.Context) {
	var req companyUserCompleteReq
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
		h.logger.Error("company user session consume failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	phone = strings.TrimSpace(phone)
	firstName := strings.TrimSpace(req.FirstName)
	if len(firstName) < 2 {
		resp.Error(c, http.StatusBadRequest, "first_name is too short")
		return
	}
	var lastName *string
	if s := strings.TrimSpace(req.LastName); s != "" {
		lastName = &s
	}
	pwHash, err := util.HashPassword(req.Password)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "password hash failed")
		return
	}
	u, err := h.repo.Create(c.Request.Context(), phone, pwHash, &firstName, lastName, nil, companyUserRoleOwner)
	if err != nil {
		if errors.Is(err, appusers.ErrPhoneExists) {
			resp.Error(c, http.StatusConflict, "this phone is already registered")
			return
		}
		h.logger.Error("company user create failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}
	id, _ := uuid.Parse(u.ID)
	tokens, refreshClaims, err := h.jwtm.IssueWithCompany("user", id, uuid.Nil)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "token issue failed")
		return
	}
	_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
	resp.OK(c, gin.H{
		"status": "registered",
		"tokens": tokens,
		"user": gin.H{
			"id":         u.ID,
			"phone":      u.Phone,
			"first_name": u.FirstName,
			"last_name":  u.LastName,
			"role":       companyUserRoleOwner,
			"company_id": u.CompanyID,
		},
	})
}
