package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/appusers"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/security"
	"sarbonNew/internal/store"
	"sarbonNew/internal/util"
)

type UserAuthHandler struct {
	logger  *zap.Logger
	repo    *appusers.Repo
	jwtm    *security.JWTManager
	refresh *store.RefreshStore
}

func NewUserAuthHandler(logger *zap.Logger, repo *appusers.Repo, jwtm *security.JWTManager, refresh *store.RefreshStore) *UserAuthHandler {
	return &UserAuthHandler{logger: logger, repo: repo, jwtm: jwtm, refresh: refresh}
}

type registerReq struct {
	Email     string  `json:"email" binding:"required,email"`
	Password  string  `json:"password" binding:"required,min=6"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Phone     *string `json:"phone"`
}

func (h *UserAuthHandler) Register(c *gin.Context) {
	var req registerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	hash, err := util.HashPassword(req.Password)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "password hash failed")
		return
	}
	u, err := h.repo.Create(c.Request.Context(), req.Email, hash, req.FirstName, req.LastName, req.Phone)
	if err != nil {
		if err == appusers.ErrEmailExists {
			resp.Error(c, http.StatusConflict, "email already registered")
			return
		}
		h.logger.Error("user register failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "registration failed")
		return
	}
	id, _ := uuid.Parse(u.ID)
	tokens, refreshClaims, err := h.jwtm.IssueWithCompany("user", id, uuid.Nil)
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, "token issue failed")
		return
	}
	_ = h.refresh.Put(c.Request.Context(), refreshClaims.UserID, refreshClaims.JTI)
	resp.Success(c, http.StatusCreated, "created", gin.H{
		"id":         u.ID,
		"email":      u.Email,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
		"phone":      u.Phone,
		"created_at": u.CreatedAt,
		"token":      tokens.AccessToken,
		"expires_in": tokens.ExpiresIn,
	})
}

type loginReq struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *UserAuthHandler) Login(c *gin.Context) {
	var req loginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}
	u, err := h.repo.FindByEmail(c.Request.Context(), req.Email)
	if err != nil || u == nil {
		resp.Error(c, http.StatusUnauthorized, "invalid email or password")
		return
	}
	if !util.ComparePassword(u.PasswordHash, req.Password) {
		resp.Error(c, http.StatusUnauthorized, "invalid email or password")
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
		"token":     tokens.AccessToken,
		"expires_in": tokens.ExpiresIn,
		"user": gin.H{
			"id":         u.ID,
			"email":      u.Email,
			"first_name": u.FirstName,
			"last_name":  u.LastName,
		},
	})
}
