package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/dispatchercompanies"
	"sarbonNew/internal/dispatcherinvitations"
	"sarbonNew/internal/dispatchers"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
)

type DispatcherInvitationsHandler struct {
	logger *zap.Logger
	repo   *dispatcherinvitations.Repo
	dcr    *dispatchercompanies.Repo
	disp   *dispatchers.Repo
}

func NewDispatcherInvitationsHandler(logger *zap.Logger, repo *dispatcherinvitations.Repo, dcr *dispatchercompanies.Repo, disp *dispatchers.Repo) *DispatcherInvitationsHandler {
	return &DispatcherInvitationsHandler{logger: logger, repo: repo, dcr: dcr, disp: disp}
}

// CreateInvitationReq body for POST /v1/dispatchers/companies/:companyId/invitations
type CreateInvitationReq struct {
	Phone string `json:"phone" binding:"required"`
	Role  string `json:"role" binding:"required,oneof=dispatcher top_dispatcher"`
}

// CreateInvitation creates invitation for a dispatcher (by phone). Caller must have access to company (owner).
func (h *DispatcherInvitationsHandler) CreateInvitation(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	companyID, _ := uuid.Parse(c.Param("companyId"))
	if companyID == uuid.Nil {
		resp.Error(c, http.StatusBadRequest, "invalid company_id")
		return
	}
	ok, err := h.dcr.HasAccess(c.Request.Context(), dispatcherID, companyID)
	if err != nil || !ok {
		resp.Error(c, http.StatusForbidden, "company not found or access denied")
		return
	}
	var req CreateInvitationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		resp.Error(c, http.StatusBadRequest, "phone is required")
		return
	}
	token, err := h.repo.Create(c.Request.Context(), companyID, req.Role, phone, dispatcherID, 7*24*time.Hour)
	if err != nil {
		h.logger.Error("dispatcher invitation create", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to create invitation")
		return
	}
	resp.Success(c, http.StatusCreated, "created", gin.H{"token": token, "expires_in_hours": 168})
}

// AcceptReq body for POST /v1/dispatchers/invitations/accept
type AcceptReq struct {
	Token string `json:"token" binding:"required"`
}

func normPhone(s string) string { return strings.TrimSpace(strings.ReplaceAll(s, " ", "")) }

// Accept adds dispatcher to company and deletes invitation. Phone must match.
func (h *DispatcherInvitationsHandler) Accept(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	var req AcceptReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	inv, err := h.repo.GetByToken(c.Request.Context(), strings.TrimSpace(req.Token))
	if err != nil || inv == nil {
		resp.Error(c, http.StatusBadRequest, "invitation not found or expired")
		return
	}
	disp, err := h.disp.FindByID(c.Request.Context(), dispatcherID)
	if err != nil || disp == nil {
		resp.Error(c, http.StatusUnauthorized, "dispatcher not found")
		return
	}
	if normPhone(inv.Phone) != normPhone(disp.Phone) {
		resp.Error(c, http.StatusForbidden, "invitation was sent to another phone")
		return
	}
	if err := h.dcr.Add(c.Request.Context(), dispatcherID, inv.CompanyID, inv.Role); err != nil {
		h.logger.Error("dcr add on accept", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to accept")
		return
	}
	_ = h.repo.Delete(c.Request.Context(), inv.Token)
	resp.Success(c, http.StatusOK, "accepted", gin.H{"company_id": inv.CompanyID.String()})
}

// DeclineReq body for POST /v1/dispatchers/invitations/decline
type DeclineReq struct {
	Token string `json:"token" binding:"required"`
}

// Decline deletes invitation.
func (h *DispatcherInvitationsHandler) Decline(c *gin.Context) {
	var req DeclineReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	_ = h.repo.Delete(c.Request.Context(), strings.TrimSpace(req.Token))
	resp.Success(c, http.StatusOK, "declined", nil)
}
