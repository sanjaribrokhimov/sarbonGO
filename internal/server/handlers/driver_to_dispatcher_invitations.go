package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/dispatchers"
	"sarbonNew/internal/drivers"
	"sarbonNew/internal/drivertodispatcherinvitations"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
)

// DriverToDispatcherInvitationsHandler handles invitations FROM driver TO dispatcher (by phone). Driver sends, dispatcher accepts/declines.
type DriverToDispatcherInvitationsHandler struct {
	logger *zap.Logger
	repo   *drivertodispatcherinvitations.Repo
	drv    *drivers.Repo
	disp   *dispatchers.Repo
}

// NewDriverToDispatcherInvitationsHandler creates the handler.
func NewDriverToDispatcherInvitationsHandler(logger *zap.Logger, repo *drivertodispatcherinvitations.Repo, drv *drivers.Repo, disp *dispatchers.Repo) *DriverToDispatcherInvitationsHandler {
	return &DriverToDispatcherInvitationsHandler{logger: logger, repo: repo, drv: drv, disp: disp}
}

// CreateDriverToDispatcherReq body for POST /v1/driver/dispatcher-invitations
type CreateDriverToDispatcherReq struct {
	Phone string `json:"phone" binding:"required"`
}

// CreateFromDriver creates invitation from current driver to dispatcher (by phone). GET /v1/driver/dispatcher-invitations (list) and POST (create).
func (h *DriverToDispatcherInvitationsHandler) CreateFromDriver(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req CreateDriverToDispatcherReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		resp.ErrorLang(c, http.StatusBadRequest, "phone_required")
		return
	}
	// Optional: check dispatcher exists by phone (so we don't invite non-existent)
	disp, _ := h.disp.FindByPhone(c.Request.Context(), phone)
	if disp == nil {
		// still allow sending (dispatcher might register later)
	}
	// If driver already linked to this dispatcher, no need to invite
	if disp != nil {
		drv, _ := h.drv.FindByID(c.Request.Context(), driverID)
		if drv != nil && drv.FreelancerID != nil && *drv.FreelancerID == disp.ID {
			resp.ErrorLang(c, http.StatusConflict, "already_linked_to_this_dispatcher")
			return
		}
	}
	token, err := h.repo.Create(c.Request.Context(), driverID, phone, 7*24*time.Hour)
	if err != nil {
		h.logger.Error("driver to dispatcher invitation create", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_create_invitation")
		return
	}
	resp.SuccessLang(c, http.StatusCreated, "created", gin.H{"token": token, "expires_in_hours": 168})
}

// ListSentByDriver returns invitations sent by the current driver (to dispatchers).
func (h *DriverToDispatcherInvitationsHandler) ListSentByDriver(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	list, err := h.repo.ListByDriverID(c.Request.Context(), driverID)
	if err != nil {
		h.logger.Error("driver to dispatcher invitations list sent", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_list_invitations")
		return
	}
	if list == nil {
		list = []drivertodispatcherinvitations.Invitation{}
	}
	items := make([]gin.H, 0, len(list))
	for _, inv := range list {
		items = append(items, gin.H{
			"token":             inv.Token,
			"dispatcher_phone":  inv.DispatcherPhone,
			"expires_at":       inv.ExpiresAt,
			"created_at":       inv.CreatedAt,
		})
	}
	resp.OKLang(c, "ok", gin.H{"items": items})
}

// CancelByDriver cancels an invitation sent by the current driver.
func (h *DriverToDispatcherInvitationsHandler) CancelByDriver(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		resp.ErrorLang(c, http.StatusBadRequest, "token_required")
		return
	}
	inv, err := h.repo.GetByToken(c.Request.Context(), token)
	if err != nil || inv == nil {
		resp.ErrorLang(c, http.StatusNotFound, "invitation_not_found_or_expired")
		return
	}
	if inv.DriverID != driverID {
		resp.ErrorLang(c, http.StatusForbidden, "not_your_invitation")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), token); err != nil {
		h.logger.Error("driver to dispatcher invitation cancel", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_cancel_invitation")
		return
	}
	resp.OKLang(c, "ok", nil)
}

// ListReceivedByDispatcher returns invitations sent TO the current dispatcher (by their phone).
func (h *DriverToDispatcherInvitationsHandler) ListReceivedByDispatcher(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	disp, err := h.disp.FindByID(c.Request.Context(), dispatcherID)
	if err != nil || disp == nil {
		resp.ErrorLang(c, http.StatusUnauthorized, "dispatcher_not_found")
		return
	}
	list, err := h.repo.ListByDispatcherPhone(c.Request.Context(), disp.Phone)
	if err != nil {
		h.logger.Error("driver to dispatcher invitations list received", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_list_invitations")
		return
	}
	if list == nil {
		list = []drivertodispatcherinvitations.Invitation{}
	}
	items := make([]gin.H, 0, len(list))
	for _, inv := range list {
		item := gin.H{
			"token":             inv.Token,
			"driver_id":        inv.DriverID.String(),
			"dispatcher_phone": inv.DispatcherPhone,
			"expires_at":       inv.ExpiresAt,
			"created_at":       inv.CreatedAt,
		}
		drv, _ := h.drv.FindByID(c.Request.Context(), inv.DriverID)
		if drv != nil {
			item["driver_name"] = drv.Name
			item["driver_phone"] = drv.Phone
		}
		items = append(items, item)
	}
	resp.OKLang(c, "ok", gin.H{"items": items})
}

// AcceptByDispatcherReq body for POST /v1/dispatchers/invitations-from-drivers/accept
type AcceptByDispatcherReq struct {
	Token string `json:"token" binding:"required"`
}

// AcceptByDispatcher dispatcher accepts driver's invitation; driver.freelancer_id = dispatcher.
func (h *DriverToDispatcherInvitationsHandler) AcceptByDispatcher(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	disp, err := h.disp.FindByID(c.Request.Context(), dispatcherID)
	if err != nil || disp == nil {
		resp.ErrorLang(c, http.StatusUnauthorized, "dispatcher_not_found")
		return
	}
	var req AcceptByDispatcherReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	token := strings.TrimSpace(req.Token)
	inv, err := h.repo.GetByToken(c.Request.Context(), token)
	if err != nil || inv == nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invitation_not_found_or_expired")
		return
	}
	if !inv.PhoneMatches(disp.Phone) {
		resp.ErrorLang(c, http.StatusForbidden, "invitation_sent_to_another_phone")
		return
	}
	if err := h.drv.SetFreelancerID(c.Request.Context(), inv.DriverID, dispatcherID); err != nil {
		h.logger.Error("dispatcher accept driver invitation", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_accept")
		return
	}
	_ = h.repo.Delete(c.Request.Context(), token)
	resp.SuccessLang(c, http.StatusOK, "accepted", gin.H{"driver_id": inv.DriverID.String()})
}

// DeclineByDispatcherReq body for POST /v1/dispatchers/invitations-from-drivers/decline
type DeclineByDispatcherReq struct {
	Token string `json:"token" binding:"required"`
}

// DeclineByDispatcher dispatcher declines driver's invitation; invitation is deleted.
func (h *DriverToDispatcherInvitationsHandler) DeclineByDispatcher(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	disp, err := h.disp.FindByID(c.Request.Context(), dispatcherID)
	if err != nil || disp == nil {
		resp.ErrorLang(c, http.StatusUnauthorized, "dispatcher_not_found")
		return
	}
	var req DeclineByDispatcherReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	token := strings.TrimSpace(req.Token)
	inv, err := h.repo.GetByToken(c.Request.Context(), token)
	if err != nil || inv == nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invitation_not_found_or_expired")
		return
	}
	if !inv.PhoneMatches(disp.Phone) {
		resp.ErrorLang(c, http.StatusForbidden, "invitation_sent_to_another_phone")
		return
	}
	_ = h.repo.Delete(c.Request.Context(), token)
	resp.OKLang(c, "declined", gin.H{"status": "declined"})
}
