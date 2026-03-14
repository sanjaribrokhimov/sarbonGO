package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/cargo"
	"sarbonNew/internal/cargorecommendations"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/trips"
)

// CargoRecommendationsHandler: dispatcher recommends cargo to driver; driver accept/decline.
type CargoRecommendationsHandler struct {
	logger     *zap.Logger
	recRepo    *cargorecommendations.Repo
	cargoRepo  *cargo.Repo
	tripsRepo  *trips.Repo
}

// NewCargoRecommendationsHandler creates the handler.
func NewCargoRecommendationsHandler(logger *zap.Logger, recRepo *cargorecommendations.Repo, cargoRepo *cargo.Repo, tripsRepo *trips.Repo) *CargoRecommendationsHandler {
	return &CargoRecommendationsHandler{logger: logger, recRepo: recRepo, cargoRepo: cargoRepo, tripsRepo: tripsRepo}
}

// RecommendReq body: driver_id. Dispatcher recommends cargo to one driver.
func (h *CargoRecommendationsHandler) Recommend(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	cargoID, err := uuid.Parse(c.Param("id"))
	if err != nil || cargoID == uuid.Nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	var req struct {
		DriverID uuid.UUID `json:"driver_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	obj, _ := h.cargoRepo.GetByID(c.Request.Context(), cargoID, false)
	if obj == nil {
		resp.ErrorLang(c, http.StatusNotFound, "cargo_not_found")
		return
	}
	if !cargo.IsSearching(obj.Status) {
		resp.ErrorLang(c, http.StatusBadRequest, "cargo_not_searching")
		return
	}
	if obj.CreatedByType == nil || *obj.CreatedByType != "DISPATCHER" || obj.CreatedByID == nil || *obj.CreatedByID != dispatcherID {
		resp.ErrorLang(c, http.StatusForbidden, "not_your_cargo")
		return
	}
	if err := h.recRepo.Create(c.Request.Context(), cargoID, req.DriverID, dispatcherID); err != nil {
		h.logger.Error("cargo recommend create", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_create_recommendation")
		return
	}
	resp.SuccessLang(c, http.StatusCreated, "created", gin.H{"cargo_id": cargoID.String(), "driver_id": req.DriverID.String()})
}

// ListRecommendedForDriver returns cargos recommended to the current driver (pending only).
func (h *CargoRecommendationsHandler) ListRecommendedForDriver(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	limit := 30
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	list, err := h.recRepo.ListByDriver(c.Request.Context(), driverID, limit)
	if err != nil {
		h.logger.Error("cargo recommendations list", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_list")
		return
	}
	items := make([]gin.H, 0, len(list))
	for i := range list {
		rec := &list[i]
		cargoObj, _ := h.cargoRepo.GetByID(c.Request.Context(), rec.CargoID, false)
		item := gin.H{
			"cargo_id": rec.CargoID.String(), "driver_id": rec.DriverID.String(),
			"invited_by_dispatcher_id": rec.InvitedByDispatcherID.String(),
			"status": rec.Status, "created_at": rec.CreatedAt,
		}
		if cargoObj != nil {
			item["cargo"] = gin.H{"id": cargoObj.ID, "weight": cargoObj.Weight, "volume": cargoObj.Volume, "truck_type": cargoObj.TruckType, "status": cargoObj.Status}
		}
		items = append(items, item)
	}
	resp.OKLang(c, "ok", gin.H{"items": items})
}

// AcceptRecommendation driver accepts recommended cargo: create offer at cargo price and auto-accept, create trip.
// Order: validate cargo (searching) and recommendation (pending), create offer, accept offer, then mark recommendation accepted.
func (h *CargoRecommendationsHandler) AcceptRecommendation(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	cargoID, err := uuid.Parse(c.Param("cargoId"))
	if err != nil || cargoID == uuid.Nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	rec, _ := h.recRepo.GetByCargoAndDriver(c.Request.Context(), cargoID, driverID)
	if rec == nil || rec.Status != "PENDING" {
		resp.ErrorLang(c, http.StatusBadRequest, "recommendation_not_found_or_not_pending")
		return
	}
	obj, _ := h.cargoRepo.GetByID(c.Request.Context(), cargoID, false)
	if obj == nil {
		resp.ErrorLang(c, http.StatusNotFound, "cargo_not_found")
		return
	}
	if !cargo.IsSearching(obj.Status) {
		resp.ErrorLang(c, http.StatusConflict, "cargo_already_assigned")
		return
	}
	price, currency := 0.0, "USD"
	if pay, _ := h.cargoRepo.GetPayment(c.Request.Context(), cargoID); pay != nil && pay.TotalAmount != nil && pay.TotalCurrency != nil {
		price = *pay.TotalAmount
		currency = *pay.TotalCurrency
	}
	offerID, err := h.cargoRepo.CreateOffer(c.Request.Context(), cargoID, driverID, price, currency, "")
	if err != nil {
		h.logger.Error("cargo recommend accept create offer", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_accept")
		return
	}
	_, carrierID, err := h.cargoRepo.AcceptOffer(c.Request.Context(), offerID)
	if err != nil {
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_accept")
		return
	}
	_, _ = h.recRepo.Accept(c.Request.Context(), cargoID, driverID)
	if h.tripsRepo != nil {
		tripID, _ := h.tripsRepo.Create(c.Request.Context(), cargoID, offerID)
		if tripID != uuid.Nil {
			_ = h.tripsRepo.AssignDriver(c.Request.Context(), tripID, carrierID)
			resp.SuccessLang(c, http.StatusOK, "accepted", gin.H{"cargo_id": cargoID.String(), "trip_id": tripID.String(), "driver_id": carrierID.String()})
			return
		}
	}
	resp.SuccessLang(c, http.StatusOK, "accepted", gin.H{"cargo_id": cargoID.String()})
}

// DeclineRecommendation driver declines recommended cargo.
func (h *CargoRecommendationsHandler) DeclineRecommendation(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	cargoID, err := uuid.Parse(c.Param("cargoId"))
	if err != nil || cargoID == uuid.Nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	ok, err := h.recRepo.Decline(c.Request.Context(), cargoID, driverID)
	if err != nil || !ok {
		resp.ErrorLang(c, http.StatusBadRequest, "recommendation_not_found_or_not_pending")
		return
	}
	resp.OKLang(c, "declined", gin.H{"status": "declined"})
}
