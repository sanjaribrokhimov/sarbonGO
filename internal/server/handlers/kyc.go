package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/domain"
	"sarbonNew/internal/drivers"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
)

type KYCHandler struct {
	logger *zap.Logger
	drivers *drivers.Repo
}

func NewKYCHandler(logger *zap.Logger, driversRepo *drivers.Repo) *KYCHandler {
	return &KYCHandler{logger: logger, drivers: driversRepo}
}

type kycReq struct {
	DriverData driverData `json:"driver_data" binding:"required"`
	PowerData  powerData  `json:"power_data" binding:"required"`
	TrailerData trailerData `json:"trailer_data" binding:"required"`

	DriverOwner *bool `json:"driver_owner" binding:"required"` // required (true/false)
}

type driverData struct {
	PassportSeries string `json:"driver_passport_series" binding:"required"`
	PassportNumber string `json:"driver_passport_number" binding:"required"`
	PINFL          string `json:"driver_pinfl" binding:"required"`
	ScanStatus     *bool  `json:"driver_scan_status" binding:"required"` // required (true/false)
}

type powerData struct {
	PlateNumber string `json:"power_plate_number" binding:"required"`
	TechSeries  string `json:"power_tech_series" binding:"required"`
	TechNumber  string `json:"power_tech_number" binding:"required"`
	OwnerID     string `json:"power_owner_id" binding:"required"`     // INN or PINFL (value)
	OwnerName   string `json:"power_owner_name" binding:"required"`   // name or organization
	ScanStatus  *bool  `json:"power_scan_status" binding:"required"`  // required (true/false)
}

type trailerData struct {
	PlateNumber string `json:"trailer_plate_number" binding:"required"`
	TechSeries  string `json:"trailer_tech_series" binding:"required"`
	TechNumber  string `json:"trailer_tech_number" binding:"required"`
	OwnerID     string `json:"trailer_owner_id" binding:"required"`   // INN or PINFL (value)
	OwnerName   string `json:"trailer_owner_name" binding:"required"` // name or organization
	ScanStatus  *bool  `json:"trailer_scan_status" binding:"required"`// required (true/false)
}

func (h *KYCHandler) Submit(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)

	var req kycReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload")
		return
	}

	trim := func(s string) string { return strings.TrimSpace(s) }
	req.DriverData.PassportSeries = trim(req.DriverData.PassportSeries)
	req.DriverData.PassportNumber = trim(req.DriverData.PassportNumber)
	req.DriverData.PINFL = trim(req.DriverData.PINFL)
	req.PowerData.PlateNumber = trim(req.PowerData.PlateNumber)
	req.PowerData.TechSeries = trim(req.PowerData.TechSeries)
	req.PowerData.TechNumber = trim(req.PowerData.TechNumber)
	req.PowerData.OwnerID = trim(req.PowerData.OwnerID)
	req.PowerData.OwnerName = trim(req.PowerData.OwnerName)
	req.TrailerData.PlateNumber = trim(req.TrailerData.PlateNumber)
	req.TrailerData.TechSeries = trim(req.TrailerData.TechSeries)
	req.TrailerData.TechNumber = trim(req.TrailerData.TechNumber)
	req.TrailerData.OwnerID = trim(req.TrailerData.OwnerID)
	req.TrailerData.OwnerName = trim(req.TrailerData.OwnerName)

	// Completion rules (for StatusFull): all required fields are present + scan statuses are true.
	driverOK := req.DriverData.PassportSeries != "" && req.DriverData.PassportNumber != "" && req.DriverData.PINFL != "" &&
		req.DriverData.ScanStatus != nil && *req.DriverData.ScanStatus

	powerOK := req.PowerData.PlateNumber != "" && req.PowerData.TechSeries != "" && req.PowerData.TechNumber != "" &&
		req.PowerData.OwnerID != "" && req.PowerData.OwnerName != "" &&
		req.PowerData.ScanStatus != nil && *req.PowerData.ScanStatus

	trailerOK := req.TrailerData.PlateNumber != "" && req.TrailerData.TechSeries != "" && req.TrailerData.TechNumber != "" &&
		req.TrailerData.OwnerID != "" && req.TrailerData.OwnerName != "" &&
		req.TrailerData.ScanStatus != nil && *req.TrailerData.ScanStatus

	driverOwnerOK := req.DriverOwner != nil

	isFull := driverOK && powerOK && trailerOK && driverOwnerOK

	// Keep current status unless we can upgrade to full.
	current, err := h.drivers.FindByID(c.Request.Context(), driverID)
	if err != nil {
		resp.Error(c, http.StatusUnauthorized, "driver not found")
		return
	}
	nextStatus := current.RegistrationStatus
	regStatus := ""
	if isFull {
		regStatus = string(domain.StatusFull)
	} else if nextStatus != nil && *nextStatus != "" {
		regStatus = *nextStatus
	} else {
		// If registration isn't started properly, keep it as-is.
		regStatus = string(domain.StatusBasic)
	}

	kycStatus := "pending"
	if isFull {
		kycStatus = "approved"
	}

	u := drivers.KYCUpdate{
		DriverPassportSeries: req.DriverData.PassportSeries,
		DriverPassportNumber: req.DriverData.PassportNumber,
		DriverPINFL:          req.DriverData.PINFL,
		DriverScanStatus:     req.DriverData.ScanStatus,
		PowerPlateNumber:     req.PowerData.PlateNumber,
		PowerTechSeries:      req.PowerData.TechSeries,
		PowerTechNumber:      req.PowerData.TechNumber,
		PowerOwnerID:         req.PowerData.OwnerID,
		PowerOwnerName:       req.PowerData.OwnerName,
		PowerScanStatus:      req.PowerData.ScanStatus,
		TrailerPlateNumber:   req.TrailerData.PlateNumber,
		TrailerTechSeries:    req.TrailerData.TechSeries,
		TrailerTechNumber:    req.TrailerData.TechNumber,
		TrailerOwnerID:       req.TrailerData.OwnerID,
		TrailerOwnerName:     req.TrailerData.OwnerName,
		TrailerScanStatus:    req.TrailerData.ScanStatus,
		DriverOwner:          req.DriverOwner,
		KYCStatus:            kycStatus,
		RegistrationStatus:   regStatus,
	}

	if err := h.drivers.UpdateKYC(c.Request.Context(), driverID, u); err != nil {
		h.logger.Error("update kyc failed", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "internal error")
		return
	}

	// When registration becomes FULL, auto-fill default columns.
	if isFull {
		if err := h.drivers.ApplyFullDefaults(c.Request.Context(), driverID); err != nil {
			h.logger.Error("apply full defaults failed", zap.Error(err))
			resp.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
	}
	updated, _ := h.drivers.FindByID(c.Request.Context(), driverID)
	resp.OK(c, gin.H{
		"event":   "updated",
		"is_full": isFull,
		"driver":  updated,
	})
}

