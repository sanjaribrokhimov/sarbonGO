package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"sarbonNew/internal/dispatchercompanies"
	"sarbonNew/internal/driverinvitations"
	"sarbonNew/internal/drivers"
	"sarbonNew/internal/server/mw"
	"sarbonNew/internal/server/resp"
)

type DriverInvitationsHandler struct {
	logger *zap.Logger
	repo   *driverinvitations.Repo
	dcr    *dispatchercompanies.Repo
	drv    *drivers.Repo
}

func NewDriverInvitationsHandler(logger *zap.Logger, repo *driverinvitations.Repo, dcr *dispatchercompanies.Repo, drv *drivers.Repo) *DriverInvitationsHandler {
	return &DriverInvitationsHandler{logger: logger, repo: repo, dcr: dcr, drv: drv}
}

// CreateDriverInvitationReq body for POST /v1/dispatchers/companies/:companyId/driver-invitations
type CreateDriverInvitationReq struct {
	Phone string `json:"phone" binding:"required"`
}

// CreateDriverInvitation creates invitation for driver by phone (dispatcher with company access).
func (h *DriverInvitationsHandler) Create(c *gin.Context) {
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
	var req CreateDriverInvitationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	phone := strings.TrimSpace(req.Phone)
	if phone == "" {
		resp.Error(c, http.StatusBadRequest, "phone is required")
		return
	}
	token, err := h.repo.Create(c.Request.Context(), companyID, phone, dispatcherID, 7*24*time.Hour)
	if err != nil {
		h.logger.Error("driver invitation create", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to create invitation")
		return
	}
	resp.Success(c, http.StatusCreated, "created", gin.H{"token": token, "expires_in_hours": 168})
}

// CreateForFreelanceReq body for POST /v1/dispatchers/driver-invitations — phone или driver_id (найти водителя через GET .../drivers/find).
type CreateForFreelanceReq struct {
	Phone    string     `json:"phone"`
	DriverID *uuid.UUID `json:"driver_id"`
}

// CreateForFreelance creates driver invitation as freelance (no company). Можно передать phone или driver_id (после поиска через find).
func (h *DriverInvitationsHandler) CreateForFreelance(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	var req CreateForFreelanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	phone := strings.TrimSpace(req.Phone)
	if req.DriverID != nil && *req.DriverID != uuid.Nil {
		drv, err := h.drv.FindByID(c.Request.Context(), *req.DriverID)
		if err != nil || drv == nil {
			resp.Error(c, http.StatusBadRequest, "driver not found")
			return
		}
		phone = strings.TrimSpace(drv.Phone)
	}
	if phone == "" {
		resp.Error(c, http.StatusBadRequest, "phone or driver_id is required")
		return
	}
	token, err := h.repo.CreateForFreelance(c.Request.Context(), dispatcherID, phone, 7*24*time.Hour)
	if err != nil {
		h.logger.Error("driver invitation create freelance", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to create invitation")
		return
	}
	resp.Success(c, http.StatusCreated, "created", gin.H{"token": token, "expires_in_hours": 168})
}

// FindDrivers returns drivers matching phone search (для диспетчера: найти водителя и пригласить по driver_id). Совпадения сверху.
func (h *DriverInvitationsHandler) FindDrivers(c *gin.Context) {
	_ = c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	phoneSearch := strings.TrimSpace(c.Query("phone"))
	if phoneSearch == "" {
		resp.OK(c, gin.H{"items": []gin.H{}})
		return
	}
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	list, err := h.drv.SearchByPhone(c.Request.Context(), phoneSearch, limit)
	if err != nil {
		h.logger.Error("drivers find", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to search drivers")
		return
	}
	if list == nil {
		list = []*drivers.Driver{}
	}
	items := make([]gin.H, 0, len(list))
	for _, d := range list {
		items = append(items, gin.H{
			"id": d.ID, "phone": d.Phone, "name": d.Name,
			"work_status": d.WorkStatus, "driver_type": d.DriverType,
			"freelancer_id": d.FreelancerID, "company_id": d.CompanyID,
		})
	}
	resp.OK(c, gin.H{"items": items})
}

// ListSent returns invitations sent by the current dispatcher (company and freelance). Диспетчер видит кому отправил приглашения.
func (h *DriverInvitationsHandler) ListSent(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	list, err := h.repo.ListByInvitedBy(c.Request.Context(), dispatcherID)
	if err != nil {
		h.logger.Error("driver invitations list sent", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to list invitations")
		return
	}
	if list == nil {
		list = []driverinvitations.Invitation{}
	}
	items := make([]gin.H, 0, len(list))
	for _, inv := range list {
		item := gin.H{
			"token":      inv.Token,
			"phone":      inv.Phone,
			"expires_at": inv.ExpiresAt,
			"created_at": inv.CreatedAt,
		}
		if inv.CompanyID != nil && *inv.CompanyID != uuid.Nil {
			item["type"] = "company"
			item["company_id"] = inv.CompanyID.String()
		} else {
			item["type"] = "freelance"
			if inv.InvitedByDispatcherID != nil {
				item["dispatcher_id"] = inv.InvitedByDispatcherID.String()
			}
		}
		items = append(items, item)
	}
	resp.OK(c, gin.H{"items": items})
}

// UnlinkDriver removes driver from dispatcher's list (sets driver.freelancer_id = NULL). Водитель должен быть принят по приглашению (freelancer_id = я).
func (h *DriverInvitationsHandler) UnlinkDriver(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	driverID, err := uuid.Parse(c.Param("driverId"))
	if err != nil || driverID == uuid.Nil {
		resp.Error(c, http.StatusBadRequest, "invalid driver_id")
		return
	}
	ok, err := h.drv.UnlinkFromFreelancer(c.Request.Context(), driverID, dispatcherID)
	if err != nil {
		h.logger.Error("unlink driver", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to unlink")
		return
	}
	if !ok {
		resp.Error(c, http.StatusForbidden, "driver not found or not linked to you")
		return
	}
	resp.Success(c, http.StatusOK, "unlinked", nil)
}

// SetDriverPowerReq body for PUT /v1/dispatchers/drivers/:driverId/power
type SetDriverPowerReq struct {
	PowerPlateType   *string `json:"power_plate_type,omitempty"`
	PowerPlateNumber *string `json:"power_plate_number,omitempty"`
	PowerTechSeries  *string `json:"power_tech_series,omitempty"`
	PowerTechNumber  *string `json:"power_tech_number,omitempty"`
	PowerOwnerID     *string `json:"power_owner_id,omitempty"`
	PowerOwnerName   *string `json:"power_owner_name,omitempty"`
	PowerScanStatus  *bool   `json:"power_scan_status,omitempty"`
}

// SetDriverPower adds or updates тягач for a driver. Водитель должен быть принят по приглашению (freelancer_id = я).
func (h *DriverInvitationsHandler) SetDriverPower(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	driverID, err := uuid.Parse(c.Param("driverId"))
	if err != nil || driverID == uuid.Nil {
		resp.Error(c, http.StatusBadRequest, "invalid driver_id")
		return
	}
	drv, err := h.drv.FindByID(c.Request.Context(), driverID)
	if err != nil || drv == nil {
		resp.Error(c, http.StatusNotFound, "driver not found")
		return
	}
	if drv.FreelancerID == nil || *drv.FreelancerID != dispatcherID.String() {
		resp.Error(c, http.StatusForbidden, "driver must have accepted your invitation")
		return
	}
	var req SetDriverPowerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	trimPtr := func(p **string) {
		if p == nil || *p == nil {
			return
		}
		v := strings.TrimSpace(**p)
		if v == "" {
			*p = nil
			return
		}
		*p = &v
	}
	trimPtr(&req.PowerPlateType)
	trimPtr(&req.PowerPlateNumber)
	trimPtr(&req.PowerTechSeries)
	trimPtr(&req.PowerTechNumber)
	trimPtr(&req.PowerOwnerID)
	trimPtr(&req.PowerOwnerName)
	if err := h.drv.UpdatePowerProfile(c.Request.Context(), driverID, drivers.UpdatePowerProfile{
		PowerPlateType:   req.PowerPlateType,
		PowerPlateNumber: req.PowerPlateNumber,
		PowerTechSeries:  req.PowerTechSeries,
		PowerTechNumber:  req.PowerTechNumber,
		PowerOwnerID:     req.PowerOwnerID,
		PowerOwnerName:   req.PowerOwnerName,
		PowerScanStatus:  req.PowerScanStatus,
	}); err != nil {
		h.logger.Error("dispatcher set driver power", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to update power")
		return
	}
	updated, _ := h.drv.FindByID(c.Request.Context(), driverID)
	resp.OK(c, gin.H{"event": "updated", "driver": updated})
}

// SetDriverTrailerReq body for PUT /v1/dispatchers/drivers/:driverId/trailer
type SetDriverTrailerReq struct {
	TrailerPlateType   *string `json:"trailer_plate_type,omitempty"`
	TrailerPlateNumber *string `json:"trailer_plate_number,omitempty"`
	TrailerTechSeries  *string `json:"trailer_tech_series,omitempty"`
	TrailerTechNumber  *string `json:"trailer_tech_number,omitempty"`
	TrailerOwnerID     *string `json:"trailer_owner_id,omitempty"`
	TrailerOwnerName   *string `json:"trailer_owner_name,omitempty"`
	TrailerScanStatus  *bool   `json:"trailer_scan_status,omitempty"`
}

// SetDriverTrailer adds or updates прицеп for a driver. Водитель должен быть принят по приглашению (freelancer_id = я).
func (h *DriverInvitationsHandler) SetDriverTrailer(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	driverID, err := uuid.Parse(c.Param("driverId"))
	if err != nil || driverID == uuid.Nil {
		resp.Error(c, http.StatusBadRequest, "invalid driver_id")
		return
	}
	drv, err := h.drv.FindByID(c.Request.Context(), driverID)
	if err != nil || drv == nil {
		resp.Error(c, http.StatusNotFound, "driver not found")
		return
	}
	if drv.FreelancerID == nil || *drv.FreelancerID != dispatcherID.String() {
		resp.Error(c, http.StatusForbidden, "driver must have accepted your invitation")
		return
	}
	var req SetDriverTrailerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	trimPtr := func(p **string) {
		if p == nil || *p == nil {
			return
		}
		v := strings.TrimSpace(**p)
		if v == "" {
			*p = nil
			return
		}
		*p = &v
	}
	trimPtr(&req.TrailerPlateType)
	trimPtr(&req.TrailerPlateNumber)
	trimPtr(&req.TrailerTechSeries)
	trimPtr(&req.TrailerTechNumber)
	trimPtr(&req.TrailerOwnerID)
	trimPtr(&req.TrailerOwnerName)
	if err := h.drv.UpdateTrailerProfile(c.Request.Context(), driverID, drivers.UpdateTrailerProfile{
		TrailerPlateType:   req.TrailerPlateType,
		TrailerPlateNumber: req.TrailerPlateNumber,
		TrailerTechSeries:  req.TrailerTechSeries,
		TrailerTechNumber:  req.TrailerTechNumber,
		TrailerOwnerID:     req.TrailerOwnerID,
		TrailerOwnerName:   req.TrailerOwnerName,
		TrailerScanStatus:  req.TrailerScanStatus,
	}); err != nil {
		h.logger.Error("dispatcher set driver trailer", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to update trailer")
		return
	}
	updated, _ := h.drv.FindByID(c.Request.Context(), driverID)
	resp.OK(c, gin.H{"event": "updated", "driver": updated})
}

// CancelInvitation cancels (revokes) an invitation sent by the current dispatcher. Только свои приглашения.
func (h *DriverInvitationsHandler) CancelInvitation(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	token := strings.TrimSpace(c.Param("token"))
	if token == "" {
		resp.Error(c, http.StatusBadRequest, "token is required")
		return
	}
	inv, err := h.repo.GetByToken(c.Request.Context(), token)
	if err != nil || inv == nil {
		resp.Error(c, http.StatusNotFound, "invitation not found or expired")
		return
	}
	if inv.InvitedBy != dispatcherID {
		resp.Error(c, http.StatusForbidden, "not your invitation")
		return
	}
	if err := h.repo.Delete(c.Request.Context(), token); err != nil {
		h.logger.Error("driver invitation cancel", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to cancel invitation")
		return
	}
	resp.Success(c, http.StatusOK, "cancelled", nil)
}

// ListInvitations returns pending invitations for the current driver (by phone). Водитель видит приглашения в чате/разделе приглашений.
func (h *DriverInvitationsHandler) ListInvitations(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	drv, err := h.drv.FindByID(c.Request.Context(), driverID)
	if err != nil || drv == nil {
		resp.Error(c, http.StatusUnauthorized, "driver not found")
		return
	}
	list, err := h.repo.ListByPhone(c.Request.Context(), drv.Phone)
	if err != nil {
		h.logger.Error("driver invitations list", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to list invitations")
		return
	}
	if list == nil {
		list = []driverinvitations.Invitation{}
	}
	items := make([]gin.H, 0, len(list))
	for _, inv := range list {
		item := gin.H{
			"token":      inv.Token,
			"phone":      inv.Phone,
			"expires_at": inv.ExpiresAt,
			"created_at": inv.CreatedAt,
		}
		if inv.CompanyID != nil && *inv.CompanyID != uuid.Nil {
			item["type"] = "company"
			item["company_id"] = inv.CompanyID.String()
		} else if inv.InvitedByDispatcherID != nil && *inv.InvitedByDispatcherID != uuid.Nil {
			item["type"] = "freelance"
			item["dispatcher_id"] = inv.InvitedByDispatcherID.String()
		} else {
			item["type"] = "unknown"
		}
		items = append(items, item)
	}
	resp.OK(c, gin.H{"items": items})
}

// AcceptDriverInvitationReq body for POST /v1/driver/driver-invitations/accept
type AcceptDriverInvitationReq struct {
	Token string `json:"token" binding:"required"`
}

// AcceptDriverInvitation links driver to company or to freelance dispatcher (driver's phone must match invitation).
func (h *DriverInvitationsHandler) Accept(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req AcceptDriverInvitationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	inv, err := h.repo.GetByToken(c.Request.Context(), strings.TrimSpace(req.Token))
	if err != nil || inv == nil {
		resp.Error(c, http.StatusBadRequest, "invitation not found or expired")
		return
	}
	drv, err := h.drv.FindByID(c.Request.Context(), driverID)
	if err != nil || drv == nil {
		resp.Error(c, http.StatusUnauthorized, "driver not found")
		return
	}
	if strings.TrimSpace(strings.ReplaceAll(inv.Phone, " ", "")) != strings.TrimSpace(strings.ReplaceAll(drv.Phone, " ", "")) {
		resp.Error(c, http.StatusForbidden, "invitation was sent to another phone")
		return
	}
	if inv.CompanyID != nil && *inv.CompanyID != uuid.Nil {
		if err := h.drv.SetCompanyID(c.Request.Context(), driverID, *inv.CompanyID); err != nil {
			h.logger.Error("driver set company", zap.Error(err))
			resp.Error(c, http.StatusInternalServerError, "failed to accept")
			return
		}
		_ = h.repo.Delete(c.Request.Context(), inv.Token)
		resp.Success(c, http.StatusOK, "accepted", gin.H{"company_id": inv.CompanyID.String()})
		return
	}
	if inv.InvitedByDispatcherID != nil && *inv.InvitedByDispatcherID != uuid.Nil {
		if err := h.drv.SetFreelancerID(c.Request.Context(), driverID, *inv.InvitedByDispatcherID); err != nil {
			h.logger.Error("driver set freelancer", zap.Error(err))
			resp.Error(c, http.StatusInternalServerError, "failed to accept")
			return
		}
		_ = h.repo.Delete(c.Request.Context(), inv.Token)
		resp.Success(c, http.StatusOK, "accepted", gin.H{"freelancer_id": inv.InvitedByDispatcherID.String()})
		return
	}
	resp.Error(c, http.StatusBadRequest, "invitation invalid")
}

// DeclineDriverInvitationReq body for POST /v1/driver/driver-invitations/decline
type DeclineDriverInvitationReq struct {
	Token string `json:"token" binding:"required"`
}

// DeclineDriverInvitation удаляет приглашение (водитель отказывается). Проверяем, что приглашение было на этот номер.
func (h *DriverInvitationsHandler) Decline(c *gin.Context) {
	driverID := c.MustGet(mw.CtxDriverID).(uuid.UUID)
	var req DeclineDriverInvitationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, "invalid payload: "+err.Error())
		return
	}
	token := strings.TrimSpace(req.Token)
	inv, err := h.repo.GetByToken(c.Request.Context(), token)
	if err != nil || inv == nil {
		resp.Error(c, http.StatusBadRequest, "invitation not found or expired")
		return
	}
	drv, err := h.drv.FindByID(c.Request.Context(), driverID)
	if err != nil || drv == nil {
		resp.Error(c, http.StatusUnauthorized, "driver not found")
		return
	}
	if strings.TrimSpace(strings.ReplaceAll(inv.Phone, " ", "")) != strings.TrimSpace(strings.ReplaceAll(drv.Phone, " ", "")) {
		resp.Error(c, http.StatusForbidden, "invitation was sent to another phone")
		return
	}
	_ = h.repo.Delete(c.Request.Context(), token)
	resp.OK(c, gin.H{"status": "declined"})
}

// ListMyDrivers returns drivers linked to the current freelance dispatcher (freelancer_id = me).
func (h *DriverInvitationsHandler) ListMyDrivers(c *gin.Context) {
	dispatcherID := c.MustGet(mw.CtxDispatcherID).(uuid.UUID)
	limit := 100
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	list, err := h.drv.ListByFreelancerID(c.Request.Context(), dispatcherID, limit)
	if err != nil {
		h.logger.Error("list my drivers", zap.Error(err))
		resp.Error(c, http.StatusInternalServerError, "failed to list drivers")
		return
	}
	if list == nil {
		list = []*drivers.Driver{}
	}
	resp.Success(c, http.StatusOK, "ok", gin.H{"items": list})
}
