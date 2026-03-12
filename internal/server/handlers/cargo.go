package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/cargo"
	"sarbonNew/internal/config"
	"sarbonNew/internal/reference"
	"sarbonNew/internal/security"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
	"sarbonNew/internal/trips"
)

type CargoHandler struct {
	logger    *zap.Logger
	repo      *cargo.Repo
	tripsRepo *trips.Repo
	jwtm      *security.JWTManager
	cfg       config.Config
}

func NewCargoHandler(logger *zap.Logger, repo *cargo.Repo, tripsRepo *trips.Repo, jwtm *security.JWTManager, cfg config.Config) *CargoHandler {
	return &CargoHandler{logger: logger, repo: repo, tripsRepo: tripsRepo, jwtm: jwtm, cfg: cfg}
}

// CreateCargoReq body for POST /api/cargo.
type CreateCargoReq struct {
	Weight       float64                `json:"weight" binding:"required,gt=0"`
	Volume       float64                `json:"volume" binding:"required,gt=0"` // объём груза (м³)
	ReadyEnabled bool                   `json:"ready_enabled"`
	ReadyAt      *string                `json:"ready_at"`
	LoadComment  *string                `json:"load_comment"`
	TruckType    string                 `json:"truck_type" binding:"required"`
	TempMin      *float64               `json:"temp_min"`
	TempMax      *float64               `json:"temp_max"`
	ADREnabled   bool                   `json:"adr_enabled"`
	ADRClass     *string                `json:"adr_class"`
	LoadingTypes []string               `json:"loading_types"`
	Requirements []string               `json:"requirements"`
	ShipmentType *string                `json:"shipment_type"`
	BeltsCount   *int                   `json:"belts_count"`
	Documents    *cargo.Documents        `json:"documents"`
	ContactName  *string                `json:"contact_name"`
	ContactPhone *string                `json:"contact_phone"`
	RoutePoints  []RoutePointReq        `json:"route_points" binding:"required,dive"`
	Payment      *PaymentReq            `json:"payment"`
	CompanyID    *uuid.UUID             `json:"company_id"`
}

type RoutePointReq struct {
	Type         string   `json:"type" binding:"required,oneof=load unload customs transit"`
	CityCode     string   `json:"city_code" binding:"required"`   // код города (TAS, SAM, DXB) — из справочника
	RegionCode   string   `json:"region_code"`                    // код региона/области (опционально)
	Address      string   `json:"address" binding:"required"`      // адрес (улица, дом)
	Orientir     string   `json:"orientir"`                      // ориентир для водителя
	Lat          float64  `json:"lat" binding:"required"`
	Lng          float64  `json:"lng" binding:"required"`
	Comment      *string  `json:"comment"`
	PointOrder   int      `json:"point_order" binding:"required"`
	IsMainLoad   bool     `json:"is_main_load"`
	IsMainUnload bool     `json:"is_main_unload"`
}

type PaymentReq struct {
	IsNegotiable       bool     `json:"is_negotiable"`
	PriceRequest       bool     `json:"price_request"`
	TotalAmount        *float64 `json:"total_amount"`
	TotalCurrency      *string  `json:"total_currency"`
	WithPrepayment     bool     `json:"with_prepayment"`
	WithoutPrepayment  bool     `json:"without_prepayment"`
	PrepaymentAmount   *float64 `json:"prepayment_amount"`
	PrepaymentCurrency *string  `json:"prepayment_currency"`
	PrepaymentType     *string  `json:"prepayment_type"`
	RemainingAmount    *float64 `json:"remaining_amount"`
	RemainingCurrency  *string  `json:"remaining_currency"`
	RemainingType      *string  `json:"remaining_type"`
}

func (h *CargoHandler) Create(c *gin.Context) {
	var req CreateCargoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	if err := validateCargoCreate(req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	params := toCreateParams(req)
	params.CompanyID = req.CompanyID
	// Автоматически записываем, кто создал груз: admin, dispatcher или company
	raw := strings.TrimSpace(c.GetHeader(mw.HeaderUserToken))
	if raw != "" && h.jwtm != nil {
		if userID, role, err := h.jwtm.ParseAccess(raw); err == nil {
			switch role {
			case "admin":
				params.CreatedByType = strPtr("admin")
				params.CreatedByID = &userID
			case "dispatcher":
				params.CreatedByType = strPtr("dispatcher")
				params.CreatedByID = &userID
				// Лимит грузов для фриланс-диспетчера (из env)
				if h.cfg.FreelanceDispatcherCargoLimit > 0 {
					count, err := h.repo.CountByDispatcher(c.Request.Context(), userID)
					if err != nil {
						h.logger.Error("cargo count by dispatcher", zap.Error(err))
						resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_check_cargo_limit")
						return
					}
					if count >= h.cfg.FreelanceDispatcherCargoLimit {
						resp.ErrorWithData(c, http.StatusForbidden, "cargo limit reached for freelance dispatcher", gin.H{
							"limit":  h.cfg.FreelanceDispatcherCargoLimit,
							"current": count,
						})
						return
					}
				}
			}
		}
	}
	// Если создатель не определён по JWT, но передан company_id — считаем создателем компанию
	if params.CreatedByType == nil && req.CompanyID != nil {
		params.CreatedByType = strPtr("company")
		params.CreatedByID = req.CompanyID
		params.CompanyID = req.CompanyID
	}
	id, err := h.repo.Create(c.Request.Context(), params)
	if err != nil {
		h.logger.Error("cargo create", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_create_cargo")
		return
	}
	// Возвращаем полный объект груза (как GET /api/cargo/:id), чтобы клиент видел все сохранённые данные
	obj, err := h.repo.GetByID(c.Request.Context(), id, false)
	if err != nil || obj == nil {
		resp.SuccessLang(c, http.StatusCreated, "created", gin.H{"id": id.String()})
		return
	}
	points, _ := h.repo.GetRoutePoints(c.Request.Context(), id)
	pay, _ := h.repo.GetPayment(c.Request.Context(), id)
	resp.SuccessLang(c, http.StatusCreated, "created", toCargoDetail(obj, points, pay))
}

func (h *CargoHandler) List(c *gin.Context) {
	f := cargo.ListFilter{
		Page:   getIntQuery(c, "page", 1),
		Limit:  getIntQuery(c, "limit", 20),
		Sort:   c.DefaultQuery("sort", "created_at:desc"),
		TruckType: strings.TrimSpace(c.Query("truck_type")),
		CreatedFrom: strings.TrimSpace(c.Query("created_from")),
		CreatedTo:   strings.TrimSpace(c.Query("created_to")),
	}
	if v := c.Query("status"); v != "" {
		f.Status = strings.Split(v, ",")
		for i := range f.Status {
			f.Status[i] = strings.TrimSpace(f.Status[i])
		}
	}
	if v := c.Query("weight_min"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			f.WeightMin = &n
		}
	}
	if v := c.Query("weight_max"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			f.WeightMax = &n
		}
	}
	if v := c.Query("with_offers"); v != "" {
		b := strings.ToLower(v) == "true" || v == "1"
		f.WithOffers = &b
	}
	result, err := h.repo.List(c.Request.Context(), f)
	if err != nil {
		h.logger.Error("cargo list", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_list")
		return
	}
	resp.OKLang(c, "ok", gin.H{
		"items": toCargoListItems(result.Items),
		"total": result.Total,
	})
}

func (h *CargoHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	obj, err := h.repo.GetByID(c.Request.Context(), id, false)
	if err != nil {
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_get_cargo")
		return
	}
	if obj == nil {
		resp.ErrorLang(c, http.StatusNotFound, "cargo_not_found")
		return
	}
	points, _ := h.repo.GetRoutePoints(c.Request.Context(), id)
	pay, _ := h.repo.GetPayment(c.Request.Context(), id)
	resp.OKLang(c, "ok", toCargoDetail(obj, points, pay))
}

func (h *CargoHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	var req UpdateCargoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	if err := validateCargoUpdate(req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	params := toUpdateParams(req)
	if err := h.repo.Update(c.Request.Context(), id, params); err != nil {
		if err == cargo.ErrCannotEditAfterAssigned {
			resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
			return
		}
		h.logger.Error("cargo update", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_update_cargo")
		return
	}
	resp.OKLang(c, "ok", gin.H{"id": id.String()})
}

func (h *CargoHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("cargo delete", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed_to_delete_cargo")
		return
	}
	resp.OKLang(c, "ok", gin.H{"id": id.String()})
}

func (h *CargoHandler) PatchStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	var req struct {
		Status string `json:"status" binding:"required,oneof=created searching assigned in_transit delivered cancelled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	if err := h.repo.SetStatus(c.Request.Context(), id, req.Status); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	resp.OKLang(c, "updated", gin.H{"id": id.String(), "status": req.Status})
}

func (h *CargoHandler) CreateOffer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	var req struct {
		CarrierID uuid.UUID `json:"carrier_id" binding:"required"`
		Price     float64   `json:"price" binding:"required"`
		Currency  string    `json:"currency" binding:"required"`
		Comment   string    `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	offerID, err := h.repo.CreateOffer(c.Request.Context(), id, req.CarrierID, req.Price, req.Currency, req.Comment)
	if err != nil {
		h.logger.Error("cargo create offer", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed to create offer")
		return
	}
	resp.SuccessLang(c, http.StatusCreated, "created", gin.H{"id": offerID.String()})
}

func (h *CargoHandler) ListOffers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_id")
		return
	}
	offers, err := h.repo.GetOffers(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("cargo list offers", zap.Error(err))
		resp.ErrorLang(c, http.StatusInternalServerError, "failed to list offers")
		return
	}
	resp.OKLang(c, "ok", gin.H{"items": toOfferList(offers)})
}

func (h *CargoHandler) AcceptOffer(c *gin.Context) {
	offerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid offer id")
		return
	}
	cargoID, err := h.repo.AcceptOffer(c.Request.Context(), offerID)
	if err != nil {
		resp.ErrorLang(c, http.StatusBadRequest, "invalid_payload_detail")
		return
	}
	if h.tripsRepo != nil {
		tripID, _ := h.tripsRepo.Create(c.Request.Context(), cargoID, offerID)
		if tripID != uuid.Nil {
			resp.OKLang(c, "ok", gin.H{"cargo_id": cargoID.String(), "offer_id": offerID.String(), "trip_id": tripID.String(), "status": "accepted"})
			return
		}
	}
	resp.OKLang(c, "ok", gin.H{"cargo_id": cargoID.String(), "offer_id": offerID.String(), "status": "accepted"})
}

// UpdateCargoReq for PUT /api/cargo/:id (all optional).
type UpdateCargoReq struct {
	Weight       *float64         `json:"weight"`
	Volume       *float64         `json:"volume"`
	ReadyEnabled *bool            `json:"ready_enabled"`
	ReadyAt      *string          `json:"ready_at"`
	LoadComment  *string          `json:"load_comment"`
	TruckType    *string          `json:"truck_type"`
	TempMin      *float64         `json:"temp_min"`
	TempMax      *float64         `json:"temp_max"`
	ADREnabled   *bool            `json:"adr_enabled"`
	ADRClass     *string          `json:"adr_class"`
	LoadingTypes []string         `json:"loading_types"`
	Requirements []string         `json:"requirements"`
	ShipmentType *string          `json:"shipment_type"`
	BeltsCount   *int             `json:"belts_count"`
	Documents    *cargo.Documents  `json:"documents"`
	ContactName  *string          `json:"contact_name"`
	ContactPhone *string          `json:"contact_phone"`
	RoutePoints  []RoutePointReq  `json:"route_points"`
	Payment      *PaymentReq      `json:"payment"`
}

func validateCargoCreate(req CreateCargoReq) error {
	if !reference.IsAllowed(req.TruckType, reference.AllowedTruckTypes()) {
		return errors.New("truck_type must be from reference: " + strings.Join(reference.AllowedTruckTypes(), ", "))
	}
	hasLoad, hasUnload := false, false
	for _, rp := range req.RoutePoints {
		if !reference.IsAllowed(rp.Type, reference.AllowedRoutePointTypes()) {
			return errors.New("route_points[].type must be one of: load, unload, customs, transit")
		}
		if rp.Type == "load" {
			hasLoad = true
		}
		if rp.Type == "unload" {
			hasUnload = true
		}
	}
	if !hasLoad || !hasUnload {
		return errors.New("at least one load and one unload point required")
	}
	if (req.TempMin != nil || req.TempMax != nil) && req.TruckType != "refrigerator" {
		return errors.New("temp_min/temp_max require truck_type refrigerator")
	}
	if req.ADREnabled && (req.ADRClass == nil || *req.ADRClass == "") {
		return errors.New("adr_class required when adr_enabled is true")
	}
	if req.ReadyEnabled && (req.ReadyAt == nil || *req.ReadyAt == "") {
		return errors.New("ready_at required when ready_enabled is true")
	}
	if req.ShipmentType != nil && *req.ShipmentType != "" && !reference.IsAllowed(*req.ShipmentType, reference.AllowedShipmentTypes()) {
		return errors.New("shipment_type must be from reference GET /v1/reference/cargo → shipment_type")
	}
	for i, v := range req.LoadingTypes {
		if v != "" && !reference.IsAllowed(v, reference.AllowedLoadingTypes()) {
			return errors.New("loading_types[" + strconv.Itoa(i) + "] must be from reference GET /v1/reference/cargo → loading_type")
		}
	}
	if req.Payment != nil {
		if !req.Payment.PriceRequest && req.Payment.TotalAmount == nil {
			return errors.New("total_amount or price_request required in payment")
		}
		if req.Payment.TotalCurrency != nil && *req.Payment.TotalCurrency != "" && !reference.IsAllowed(*req.Payment.TotalCurrency, reference.AllowedCurrencies()) {
			return errors.New("payment.total_currency must be from reference GET /v1/reference/cargo → currency")
		}
		if req.Payment.PrepaymentCurrency != nil && *req.Payment.PrepaymentCurrency != "" && !reference.IsAllowed(*req.Payment.PrepaymentCurrency, reference.AllowedCurrencies()) {
			return errors.New("payment.prepayment_currency must be from reference GET /v1/reference/cargo → currency")
		}
		if req.Payment.RemainingCurrency != nil && *req.Payment.RemainingCurrency != "" && !reference.IsAllowed(*req.Payment.RemainingCurrency, reference.AllowedCurrencies()) {
			return errors.New("payment.remaining_currency must be from reference GET /v1/reference/cargo → currency")
		}
		if req.Payment.PrepaymentType != nil && *req.Payment.PrepaymentType != "" && !reference.IsAllowed(*req.Payment.PrepaymentType, reference.AllowedPrepaymentTypes()) {
			return errors.New("payment.prepayment_type must be from reference GET /v1/reference/cargo → prepayment_type")
		}
		if req.Payment.RemainingType != nil && *req.Payment.RemainingType != "" && !reference.IsAllowed(*req.Payment.RemainingType, reference.AllowedRemainingTypes()) {
			return errors.New("payment.remaining_type must be from reference GET /v1/reference/cargo → remaining_type")
		}
	}
	return nil
}

func validateCargoUpdate(req UpdateCargoReq) error {
	if req.Weight != nil && *req.Weight <= 0 {
		return errors.New("weight must be > 0")
	}
	if req.TruckType != nil && *req.TruckType != "" && !reference.IsAllowed(*req.TruckType, reference.AllowedTruckTypes()) {
		return errors.New("truck_type must be from reference GET /v1/reference/cargo → truck_type")
	}
	if req.TempMin != nil || req.TempMax != nil {
		if req.TruckType == nil || *req.TruckType != "refrigerator" {
			return errors.New("temp_min/temp_max require truck_type refrigerator")
		}
	}
	if req.ADREnabled != nil && *req.ADREnabled && (req.ADRClass == nil || *req.ADRClass == "") {
		return errors.New("adr_class required when adr_enabled is true")
	}
	if req.ReadyEnabled != nil && *req.ReadyEnabled && (req.ReadyAt == nil || *req.ReadyAt == "") {
		return errors.New("ready_at required when ready_enabled is true")
	}
	if req.ShipmentType != nil && *req.ShipmentType != "" && !reference.IsAllowed(*req.ShipmentType, reference.AllowedShipmentTypes()) {
		return errors.New("shipment_type must be from reference GET /v1/reference/cargo → shipment_type")
	}
	for i, v := range req.LoadingTypes {
		if v != "" && !reference.IsAllowed(v, reference.AllowedLoadingTypes()) {
			return errors.New("loading_types[" + strconv.Itoa(i) + "] must be from reference GET /v1/reference/cargo → loading_type")
		}
	}
	for i, rp := range req.RoutePoints {
		if rp.Type != "" && !reference.IsAllowed(rp.Type, reference.AllowedRoutePointTypes()) {
			return errors.New("route_points[" + strconv.Itoa(i) + "].type must be one of: load, unload, customs, transit")
		}
	}
	if req.Payment != nil {
		if req.Payment.TotalCurrency != nil && *req.Payment.TotalCurrency != "" && !reference.IsAllowed(*req.Payment.TotalCurrency, reference.AllowedCurrencies()) {
			return errors.New("payment.total_currency must be from reference GET /v1/reference/cargo → currency")
		}
		if req.Payment.PrepaymentCurrency != nil && *req.Payment.PrepaymentCurrency != "" && !reference.IsAllowed(*req.Payment.PrepaymentCurrency, reference.AllowedCurrencies()) {
			return errors.New("payment.prepayment_currency must be from reference GET /v1/reference/cargo → currency")
		}
		if req.Payment.RemainingCurrency != nil && *req.Payment.RemainingCurrency != "" && !reference.IsAllowed(*req.Payment.RemainingCurrency, reference.AllowedCurrencies()) {
			return errors.New("payment.remaining_currency must be from reference GET /v1/reference/cargo → currency")
		}
		if req.Payment.PrepaymentType != nil && *req.Payment.PrepaymentType != "" && !reference.IsAllowed(*req.Payment.PrepaymentType, reference.AllowedPrepaymentTypes()) {
			return errors.New("payment.prepayment_type must be from reference GET /v1/reference/cargo → prepayment_type")
		}
		if req.Payment.RemainingType != nil && *req.Payment.RemainingType != "" && !reference.IsAllowed(*req.Payment.RemainingType, reference.AllowedRemainingTypes()) {
			return errors.New("payment.remaining_type must be from reference GET /v1/reference/cargo → remaining_type")
		}
	}
	return nil
}

func toCreateParams(req CreateCargoReq) cargo.CreateParams {
	p := cargo.CreateParams{
		Weight:        req.Weight,
		Volume:        req.Volume,
		ReadyEnabled:  req.ReadyEnabled,
		ReadyAt:       req.ReadyAt,
		LoadComment:   req.LoadComment,
		TruckType:     req.TruckType,
		TempMin:       req.TempMin,
		TempMax:       req.TempMax,
		ADREnabled:    req.ADREnabled,
		ADRClass:      req.ADRClass,
		LoadingTypes:  req.LoadingTypes,
		Requirements:  req.Requirements,
		ShipmentType:  req.ShipmentType,
		BeltsCount:    req.BeltsCount,
		Documents:     req.Documents,
		ContactName:   req.ContactName,
		ContactPhone:  req.ContactPhone,
		Status:        cargo.StatusCreated, // при создании всегда created; смена на searching — через PATCH .../status
	}
	for _, rp := range req.RoutePoints {
		p.RoutePoints = append(p.RoutePoints, cargo.RoutePointInput{
			Type:         rp.Type,
			CityCode:     rp.CityCode,
			RegionCode:   rp.RegionCode,
			Address:      rp.Address,
			Orientir:     rp.Orientir,
			Lat:          rp.Lat,
			Lng:          rp.Lng,
			Comment:      rp.Comment,
			PointOrder:   rp.PointOrder,
			IsMainLoad:   rp.IsMainLoad,
			IsMainUnload: rp.IsMainUnload,
		})
	}
	if req.Payment != nil {
		p.Payment = &cargo.PaymentInput{
			IsNegotiable:       req.Payment.IsNegotiable,
			PriceRequest:       req.Payment.PriceRequest,
			TotalAmount:        req.Payment.TotalAmount,
			TotalCurrency:      req.Payment.TotalCurrency,
			WithPrepayment:     req.Payment.WithPrepayment,
			WithoutPrepayment:  req.Payment.WithoutPrepayment,
			PrepaymentAmount:   req.Payment.PrepaymentAmount,
			PrepaymentCurrency: req.Payment.PrepaymentCurrency,
			PrepaymentType:     req.Payment.PrepaymentType,
			RemainingAmount:    req.Payment.RemainingAmount,
			RemainingCurrency:  req.Payment.RemainingCurrency,
			RemainingType:      req.Payment.RemainingType,
		}
	}
	return p
}

func toUpdateParams(req UpdateCargoReq) cargo.UpdateParams {
	p := cargo.UpdateParams{}
	p.Weight = req.Weight
	p.Volume = req.Volume
	p.ReadyEnabled = req.ReadyEnabled
	p.ReadyAt = req.ReadyAt
	p.LoadComment = req.LoadComment
	p.TruckType = req.TruckType
	p.TempMin = req.TempMin
	p.TempMax = req.TempMax
	p.ADREnabled = req.ADREnabled
	p.ADRClass = req.ADRClass
	p.LoadingTypes = req.LoadingTypes
	p.Requirements = req.Requirements
	p.ShipmentType = req.ShipmentType
	p.BeltsCount = req.BeltsCount
	p.Documents = req.Documents
	p.ContactName = req.ContactName
	p.ContactPhone = req.ContactPhone
	for _, rp := range req.RoutePoints {
		p.RoutePoints = append(p.RoutePoints, cargo.RoutePointInput{
			Type: rp.Type, CityCode: rp.CityCode, RegionCode: rp.RegionCode, Address: rp.Address, Orientir: rp.Orientir,
			Lat: rp.Lat, Lng: rp.Lng, Comment: rp.Comment, PointOrder: rp.PointOrder, IsMainLoad: rp.IsMainLoad, IsMainUnload: rp.IsMainUnload,
		})
	}
	if req.Payment != nil {
		p.Payment = &cargo.PaymentInput{
			IsNegotiable: req.Payment.IsNegotiable, PriceRequest: req.Payment.PriceRequest,
			TotalAmount: req.Payment.TotalAmount, TotalCurrency: req.Payment.TotalCurrency,
			WithPrepayment: req.Payment.WithPrepayment, WithoutPrepayment: req.Payment.WithoutPrepayment,
			PrepaymentAmount: req.Payment.PrepaymentAmount, PrepaymentCurrency: req.Payment.PrepaymentCurrency,
			PrepaymentType: req.Payment.PrepaymentType, RemainingAmount: req.Payment.RemainingAmount,
			RemainingCurrency: req.Payment.RemainingCurrency, RemainingType: req.Payment.RemainingType,
		}
	}
	return p
}

func toCargoListItems(items []cargo.Cargo) []gin.H {
	out := make([]gin.H, 0, len(items))
	for _, c := range items {
		out = append(out, toCargoItem(&c))
	}
	return out
}

func toCargoItem(c *cargo.Cargo) gin.H {
	out := gin.H{
		"id": c.ID.String(), "weight": c.Weight, "volume": c.Volume,
		"ready_enabled": c.ReadyEnabled, "ready_at": c.ReadyAt, "load_comment": c.LoadComment,
		"truck_type": c.TruckType, "temp_min": c.TempMin, "temp_max": c.TempMax,
		"adr_enabled": c.ADREnabled, "adr_class": c.ADRClass, "loading_types": c.LoadingTypes, "requirements": c.Requirements,
		"shipment_type": c.ShipmentType, "belts_count": c.BeltsCount, "documents": c.Documents,
		"contact_name": c.ContactName, "contact_phone": c.ContactPhone, "status": c.Status,
		"created_at": c.CreatedAt, "updated_at": c.UpdatedAt,
	}
	if c.CreatedByType != nil {
		out["created_by_type"] = *c.CreatedByType
	}
	if c.CreatedByID != nil {
		out["created_by_id"] = c.CreatedByID.String()
	}
	if c.CompanyID != nil {
		out["company_id"] = c.CompanyID.String()
	}
	return out
}

func toCargoDetail(c *cargo.Cargo, points []cargo.RoutePoint, pay *cargo.Payment) gin.H {
	detail := toCargoItem(c)
	detail["route_points"] = toRoutePointsResp(points)
	detail["payment"] = toPaymentResp(pay)
	return detail
}

func toRoutePointsResp(p []cargo.RoutePoint) []gin.H {
	out := make([]gin.H, 0, len(p))
	for _, rp := range p {
		out = append(out, gin.H{
			"id": rp.ID.String(), "cargo_id": rp.CargoID.String(), "type": rp.Type,
			"city_code": rp.CityCode, "region_code": rp.RegionCode, "address": rp.Address, "orientir": rp.Orientir,
			"lat": rp.Lat, "lng": rp.Lng, "comment": rp.Comment,
			"point_order": rp.PointOrder, "is_main_load": rp.IsMainLoad, "is_main_unload": rp.IsMainUnload,
		})
	}
	return out
}

func toPaymentResp(p *cargo.Payment) gin.H {
	if p == nil {
		return nil
	}
	return gin.H{
		"id": p.ID.String(), "cargo_id": p.CargoID.String(), "is_negotiable": p.IsNegotiable, "price_request": p.PriceRequest,
		"total_amount": p.TotalAmount, "total_currency": p.TotalCurrency,
		"with_prepayment": p.WithPrepayment, "without_prepayment": p.WithoutPrepayment,
		"prepayment_amount": p.PrepaymentAmount, "prepayment_currency": p.PrepaymentCurrency, "prepayment_type": p.PrepaymentType,
		"remaining_amount": p.RemainingAmount, "remaining_currency": p.RemainingCurrency, "remaining_type": p.RemainingType,
	}
}

func toOfferList(offers []cargo.Offer) []gin.H {
	out := make([]gin.H, 0, len(offers))
	for _, o := range offers {
		out = append(out, gin.H{
			"id": o.ID.String(), "cargo_id": o.CargoID.String(), "carrier_id": o.CarrierID.String(),
			"price": o.Price, "currency": o.Currency, "comment": o.Comment, "status": o.Status, "created_at": o.CreatedAt,
		})
	}
	return out
}

func getIntQuery(c *gin.Context, key string, defaultVal int) int {
	v := c.Query(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

func strPtr(s string) *string { return &s }