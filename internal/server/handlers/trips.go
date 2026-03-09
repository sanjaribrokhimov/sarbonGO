package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/trips"
)

type TripsHandler struct {
	logger *zap.Logger
	repo   *trips.Repo
}

func NewTripsHandler(logger *zap.Logger, repo *trips.Repo) *TripsHandler {
	return &TripsHandler{logger: logger, repo: repo}
}

// Get returns trip by id.
func (h *TripsHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil || t == nil {
		resp.Error(c, http.StatusNotFound, "trip not found")
		return
	}
	resp.OK(c, toTripResp(t))
}

// List for GET /api/trips: ?cargo_id= returns single trip for that cargo.
func (h *TripsHandler) List(c *gin.Context) {
	cargoIDStr := c.Query("cargo_id")
	if cargoIDStr == "" {
		resp.Error(c, http.StatusBadRequest, "require cargo_id")
		return
	}
	cargoID, err := uuid.Parse(cargoIDStr)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid cargo_id")
		return
	}
	t, err := h.repo.GetByCargoID(c.Request.Context(), cargoID)
	if err != nil || t == nil {
		resp.OK(c, gin.H{"items": []interface{}{}})
		return
	}
	resp.OK(c, gin.H{"items": []interface{}{toTripResp(t)}})
}

// ListMy for GET /v1/trips (driver): returns trips assigned to current driver.
func (h *TripsHandler) ListMy(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	list, err := h.repo.ListByDriver(c.Request.Context(), driverID, limit)
	if err != nil {
		h.logger.Error("trips list my", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to list")
		return
	}
	out := make([]interface{}, 0, len(list))
	for i := range list {
		out = append(out, toTripResp(&list[i]))
	}
	resp.OK(c, gin.H{"items": out})
}

// AssignDriverReq body for PATCH /api/trips/:id/assign-driver (dispatcher).
type AssignDriverReq struct {
	DriverID string `json:"driver_id" binding:"required,uuid"`
}

// AssignDriver sets driver on trip (dispatcher). Trip must be pending_driver.
func (h *TripsHandler) AssignDriver(c *gin.Context) {
	tripID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	var req AssignDriverReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	driverID, _ := uuid.Parse(req.DriverID)
	if driverID == uuid.Nil {
		resp.Error(c, http.StatusBadRequest, "invalid driver_id")
		return
	}
	if err := h.repo.AssignDriver(c.Request.Context(), tripID, driverID); err != nil {
		if err == trips.ErrNotFound {
			resp.Error(c, http.StatusNotFound, "trip not found or not pending_driver")
			return
		}
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.OK(c, gin.H{"status": "pending_driver", "driver_id": driverID.String()})
}

// DriverConfirm sets trip status to assigned (driver accepted). Driver must be trip.driver_id.
func (h *TripsHandler) DriverConfirm(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	tripID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.DriverConfirm(c.Request.Context(), tripID, driverID); err != nil {
		if err == trips.ErrNotFound {
			resp.Error(c, http.StatusNotFound, "trip not found or not assigned to you")
			return
		}
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.OK(c, gin.H{"status": trips.StatusAssigned})
}

// DriverReject clears driver assignment so dispatcher can assign another.
func (h *TripsHandler) DriverReject(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	tripID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.DriverReject(c.Request.Context(), tripID, driverID); err != nil {
		if err == trips.ErrNotFound {
			resp.Error(c, http.StatusNotFound, "trip not found")
			return
		}
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.OK(c, gin.H{"status": trips.StatusPendingDriver})
}

// PatchStatusReq body for PATCH /api/trips/:id/status (driver: loading, en_route, unloading, completed).
type PatchStatusReq struct {
	Status string `json:"status" binding:"required,oneof=loading en_route unloading completed cancelled"`
}

// PatchStatus updates trip status (driver only).
func (h *TripsHandler) PatchStatus(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	tripID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid id")
		return
	}
	t, _ := h.repo.GetByID(c.Request.Context(), tripID)
	if t == nil || t.DriverID == nil || *t.DriverID != driverID {
		resp.Error(c, http.StatusForbidden, "trip not found or not assigned to you")
		return
	}
	var req PatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	if err := h.repo.SetStatus(c.Request.Context(), tripID, req.Status); err != nil {
		if err == trips.ErrInvalidTransition {
			resp.Error(c, http.StatusBadRequest, "invalid status transition")
			return
		}
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.OK(c, gin.H{"status": req.Status})
}

func toTripResp(t *trips.Trip) gin.H {
	res := gin.H{
		"id":         t.ID.String(),
		"cargo_id":   t.CargoID.String(),
		"offer_id":   t.OfferID.String(),
		"status":     t.Status,
		"created_at": t.CreatedAt,
		"updated_at": t.UpdatedAt,
	}
	if t.DriverID != nil {
		res["driver_id"] = t.DriverID.String()
	}
	return res
}
