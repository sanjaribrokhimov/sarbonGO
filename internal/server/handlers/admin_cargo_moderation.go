package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/cargo"
	"sarbonNew/internal/server/resp"
)

// AdminCargoModerationHandler handles admin moderation of cargo (accept → searching, reject with reason).
type AdminCargoModerationHandler struct {
	logger *zap.Logger
	repo   *cargo.Repo
}

// NewAdminCargoModerationHandler creates the handler.
func NewAdminCargoModerationHandler(logger *zap.Logger, repo *cargo.Repo) *AdminCargoModerationHandler {
	return &AdminCargoModerationHandler{logger: logger, repo: repo}
}

// ListPending returns cargo list with status pending_moderation.
// GET /v1/admin/cargo/moderation
func (h *AdminCargoModerationHandler) ListPending(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	list, total, err := h.repo.ListPendingModeration(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error("admin cargo moderation list", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_list")
		return
	}
	items := make([]gin.H, 0, len(list))
	for i := range list {
		c := &list[i]
		items = append(items, gin.H{
			"id": c.ID.String(), "weight": c.Weight, "volume": c.Volume, "truck_type": c.TruckType,
			"status": c.Status, "created_at": c.CreatedAt, "created_by_type": c.CreatedByType, "created_by_id": c.CreatedByID,
		})
	}
	resp.OKLang(c, "ok", gin.H{"items": items, "total": total})
}

// AcceptReq for POST /v1/admin/cargo/:id/moderation/accept (empty body).
// Accept sets cargo status to searching (drivers can see and send offers).
func (h *AdminCargoModerationHandler) Accept(c *gin.Context) {
	cargoID, err := uuid.Parse(c.Param("id"))
	if err != nil || cargoID == uuid.Nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	if err := h.repo.ModerationAccept(c.Request.Context(), cargoID); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "cargo_not_pending_moderation")
		return
	}
	resp.OKLang(c, "ok", gin.H{"status": cargo.StatusSearching})
}

// RejectReq body: reason (mandatory).
// Reject sets cargo status to rejected and saves reason.
func (h *AdminCargoModerationHandler) Reject(c *gin.Context) {
	cargoID, err := uuid.Parse(c.Param("id"))
	if err != nil || cargoID == uuid.Nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "moderation_rejection_reason_required")
		return
	}
	if err := h.repo.ModerationReject(c.Request.Context(), cargoID, req.Reason); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "cargo_not_pending_moderation")
		return
	}
	resp.OKLang(c, "ok", gin.H{"status": cargo.StatusRejected})
}
