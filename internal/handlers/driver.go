// Driver CRUD handlers (no auth, pure CRUD).
package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sarbonGO/backend/internal/middleware"
	"github.com/sarbonGO/backend/internal/response"
)

const (
	WorkStatusFree = "free"
	WorkStatusBusy = "busy"

	AccountStatusPending  = "pending"
	AccountStatusApproved = "approved"
	AccountStatusBlocked  = "blocked"

	LangRU = "ru"
	LangUZ = "uz"
	LangEN = "en"
	LangTR = "tr"
	LangZH = "zh"
)

var (
	validWorkStatus = map[string]bool{WorkStatusFree: true, WorkStatusBusy: true}
	validAccountStatus = map[string]bool{
		AccountStatusPending: true, AccountStatusApproved: true, AccountStatusBlocked: true,
	}
	validLanguage = map[string]bool{LangRU: true, LangUZ: true, LangEN: true, LangTR: true, LangZH: true}
)

// Driver is the full driver entity as returned by API.
type Driver struct {
	ID                     string   `json:"id"`
	FirstName              string   `json:"first_name"`
	LastName               string   `json:"last_name"`
	PhoneNumber            string   `json:"phone_number"`
	PassportSeries         string   `json:"passport_series"`
	PassportNumber         string   `json:"passport_number"`
	CompanyID              *string  `json:"company_id,omitempty"`
	FreelanceDispatcherID  *string  `json:"freelance_dispatcher_id,omitempty"`
	CarPhotoURL            *string  `json:"car_photo_url,omitempty"`
	AdrDocumentURL         *string  `json:"adr_document_url,omitempty"`
	Rating                 float64  `json:"rating"`
	WorkStatus             string   `json:"work_status"`
	AccountStatus          string   `json:"account_status"`
	Language               string   `json:"language"`
	Platform               string   `json:"platform,omitempty"`
	DispatcherType         string   `json:"dispatcher_type,omitempty"`
	LastActivatedAt        *string  `json:"last_activated_at,omitempty"`
	LastActivatedLatitude   *float64 `json:"last_activated_latitude,omitempty"`
	LastActivatedLongitude  *float64 `json:"last_activated_longitude,omitempty"`
	CreatedAt              string   `json:"created_at"`
	UpdatedAt              string   `json:"updated_at"`
	DeletedAt              *string  `json:"deleted_at,omitempty"`
}

// CreateDriverRequest is the body for POST /drivers.
type CreateDriverRequest struct {
	FirstName       string   `json:"first_name" binding:"required"`
	LastName        string   `json:"last_name" binding:"required"`
	PhoneNumber     string   `json:"phone_number" binding:"required"`
	PassportSeries  string   `json:"passport_series" binding:"required"`
	PassportNumber  string   `json:"passport_number" binding:"required"`
	CompanyID             *string  `json:"company_id"`
	FreelanceDispatcherID *string  `json:"freelance_dispatcher_id"`
	CarPhotoURL            *string  `json:"car_photo_url"`
	AdrDocumentURL         *string  `json:"adr_document_url"`
	Rating                 *float64 `json:"rating"`
	WorkStatus             *string  `json:"work_status"`
	AccountStatus          *string  `json:"account_status"`
	Language               *string  `json:"language"`
}

// UpdateDriverRequest is the body for PUT /drivers/:id (all fields optional for PATCH). Language/platform берутся из заголовков, если не переданы.
type UpdateDriverRequest struct {
	FirstName       *string  `json:"first_name"`
	LastName        *string  `json:"last_name"`
	PhoneNumber     *string  `json:"phone_number"`
	PassportSeries  *string  `json:"passport_series"`
	PassportNumber  *string  `json:"passport_number"`
	CompanyID             *string  `json:"company_id"`
	FreelanceDispatcherID *string  `json:"freelance_dispatcher_id"`
	CarPhotoURL            *string  `json:"car_photo_url"`
	AdrDocumentURL         *string  `json:"adr_document_url"`
	Rating                 *float64 `json:"rating"`
	WorkStatus             *string  `json:"work_status"`
	AccountStatus          *string  `json:"account_status"`
	Language               *string  `json:"language"`
	Platform               *string  `json:"platform"`
}

func validateRating(r float64) bool { return r >= 0 && r <= 5 }

type scanner interface {
	Scan(dest ...any) error
}

// fileBaseURL для ссылок на файлы: для профиля "/api/v1/drivers/profile/files", для по id — "/api/v1/drivers/"+id+"/files".
func scanDriverWithFileBase(row scanner, fileBaseURL string) (Driver, error) {
	var d Driver
	var companyID, freelanceDispatcherID, carPhotoPath, adrDocumentPath, platform, dispatcherType *string
	var deletedAt, lastActivatedAt *time.Time
	var lastLat, lastLng *float64
	err := row.Scan(
		&d.ID, &d.FirstName, &d.LastName, &d.PhoneNumber,
		&d.PassportSeries, &d.PassportNumber,
		&companyID, &freelanceDispatcherID, &carPhotoPath, &adrDocumentPath,
		&d.Rating, &d.WorkStatus, &d.AccountStatus, &d.Language,
		&platform, &dispatcherType, &lastActivatedAt, &lastLat, &lastLng,
		&d.CreatedAt, &d.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return Driver{}, err
	}
	d.CompanyID = companyID
	d.FreelanceDispatcherID = freelanceDispatcherID
	if platform != nil {
		d.Platform = *platform
	}
	if dispatcherType != nil {
		d.DispatcherType = *dispatcherType
	}
	if lastActivatedAt != nil {
		t := lastActivatedAt.Format(time.RFC3339)
		d.LastActivatedAt = &t
	}
	d.LastActivatedLatitude = lastLat
	d.LastActivatedLongitude = lastLng
	if deletedAt != nil {
		t := deletedAt.Format(time.RFC3339)
		d.DeletedAt = &t
	}
	if fileBaseURL == "" && d.ID != "" {
		fileBaseURL = "/api/v1/drivers/" + d.ID + "/files"
	}
	if carPhotoPath != nil && *carPhotoPath != "" && fileBaseURL != "" {
		u := fileBaseURL + "/car-photo"
		d.CarPhotoURL = &u
	}
	if adrDocumentPath != nil && *adrDocumentPath != "" && fileBaseURL != "" {
		u := fileBaseURL + "/adr-document"
		d.AdrDocumentURL = &u
	}
	return d, nil
}

func scanDriver(row scanner) (Driver, error) {
	return scanDriverWithFileBase(row, "")
}

// CreateDriver creates a new driver. POST /drivers. Accepts JSON or multipart/form-data (fields + optional files adr_document, car_photo).
func CreateDriver(pool *pgxpool.Pool, storageRoot string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateDriverRequest
		isMultipart := strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data")
		if isMultipart {
			req = bindCreateDriverFromForm(c)
			if req.FirstName == "" || req.LastName == "" || req.PhoneNumber == "" || req.PassportSeries == "" || req.PassportNumber == "" {
				response.Error(c, http.StatusBadRequest, "first_name, last_name, phone_number, passport_series, passport_number are required")
				return
			}
		} else {
			if err := c.ShouldBindJSON(&req); err != nil {
				response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
				return
			}
		}
		rating := 0.0
		if req.Rating != nil {
			rating = *req.Rating
			if !validateRating(rating) {
				response.Error(c, http.StatusBadRequest, "rating must be between 0 and 5")
				return
			}
		}
		workStatus := WorkStatusFree
		if req.WorkStatus != nil {
			if !validWorkStatus[*req.WorkStatus] {
				response.Error(c, http.StatusBadRequest, "work_status must be free or busy")
				return
			}
			workStatus = *req.WorkStatus
		}
		accountStatus := AccountStatusPending
		if req.AccountStatus != nil {
			if !validAccountStatus[*req.AccountStatus] {
				response.Error(c, http.StatusBadRequest, "account_status must be pending, approved or blocked")
				return
			}
			accountStatus = *req.AccountStatus
		}
		language := middleware.LanguageFrom(c.Request.Context())
		if language == "" || !validLanguage[language] {
			language = LangRU
		}
		if req.Language != nil {
			if !validLanguage[*req.Language] {
				response.Error(c, http.StatusBadRequest, "language must be ru, uz, en, tr or zh")
				return
			}
			language = *req.Language
		}
		platform := middleware.PlatformFrom(c.Request.Context())
		var companyID, freelanceDispatcherID *uuid.UUID
		if req.CompanyID != nil && *req.CompanyID != "" {
			parsed, err := uuid.Parse(*req.CompanyID)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "company_id must be a valid UUID")
				return
			}
			companyID = &parsed
		}
		if req.FreelanceDispatcherID != nil && *req.FreelanceDispatcherID != "" {
			parsed, err := uuid.Parse(*req.FreelanceDispatcherID)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "freelance_dispatcher_id must be a valid UUID")
				return
			}
			freelanceDispatcherID = &parsed
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO drivers (
				first_name, last_name, phone_number, passport_series, passport_number,
				company_id, freelance_dispatcher_id, rating, work_status, account_status, language, platform
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id::text
		`, req.FirstName, req.LastName, req.PhoneNumber, req.PassportSeries, req.PassportNumber,
			companyID, freelanceDispatcherID, rating, workStatus, accountStatus, language, platform).Scan(&id)
		if err != nil {
			if isUniqueViolation(err) {
				response.Error(c, http.StatusBadRequest, "phone_number already exists")
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		driverID, _ := uuid.Parse(id)
		if isMultipart {
			if adrFile, _ := c.FormFile("adr_document"); adrFile != nil {
				if err := SaveDriverFileFromMultipart(ctx, pool, storageRoot, driverID, adrFile, "adr_document_path", "adr_document", adrMaxSize, adrAllowed); err != nil {
					if errors.Is(err, errFileTooLarge) {
						response.Error(c, http.StatusBadRequest, "adr_document: file too large (max 10 MB)")
						return
					}
					if errors.Is(err, errInvalidFileType) {
						response.Error(c, http.StatusBadRequest, "adr_document: invalid type (PDF, JPG, PNG only)")
						return
					}
					response.Error(c, http.StatusInternalServerError, "internal error")
					return
				}
			}
			if carFile, _ := c.FormFile("car_photo"); carFile != nil {
				if err := SaveDriverFileFromMultipart(ctx, pool, storageRoot, driverID, carFile, "car_photo_path", "car_photo", carPhotoMaxSize, carAllowed); err != nil {
					if errors.Is(err, errFileTooLarge) {
						response.Error(c, http.StatusBadRequest, "car_photo: file too large (max 5 MB)")
						return
					}
					if errors.Is(err, errInvalidFileType) {
						response.Error(c, http.StatusBadRequest, "car_photo: invalid type (JPG, PNG only)")
						return
					}
					response.Error(c, http.StatusInternalServerError, "internal error")
					return
				}
			}
		}
		response.Success(c, http.StatusCreated, response.MsgCreated, gin.H{"id": id})
	}
}

func bindCreateDriverFromForm(c *gin.Context) CreateDriverRequest {
	var req CreateDriverRequest
	req.FirstName = c.PostForm("first_name")
	req.LastName = c.PostForm("last_name")
	req.PhoneNumber = c.PostForm("phone_number")
	req.PassportSeries = c.PostForm("passport_series")
	req.PassportNumber = c.PostForm("passport_number")
	if v := c.PostForm("company_id"); v != "" {
		req.CompanyID = &v
	}
	if v := c.PostForm("freelance_dispatcher_id"); v != "" {
		req.FreelanceDispatcherID = &v
	}
	if v := c.PostForm("rating"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			req.Rating = &f
		}
	}
	if v := c.PostForm("work_status"); v != "" {
		req.WorkStatus = &v
	}
	if v := c.PostForm("account_status"); v != "" {
		req.AccountStatus = &v
	}
	if v := c.PostForm("language"); v != "" {
		req.Language = &v
	}
	return req
}

// GetDriverByID returns one driver by ID. GET /drivers/:id.
func GetDriverByID(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid id")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		row := pool.QueryRow(ctx, `
			SELECT id::text, first_name, last_name, phone_number, passport_series, passport_number,
				company_id::text, freelance_dispatcher_id::text, car_photo_path, adr_document_path, rating, work_status, account_status, language,
				platform, dispatcher_type, last_activated_at, last_activated_latitude, last_activated_longitude,
				created_at::text, updated_at::text, deleted_at
			FROM drivers WHERE id = $1
		`, id)
		d, err := scanDriver(row)
		if err != nil {
			if isNoRows(err) {
				response.Error(c, http.StatusNotFound, "driver not found")
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		if d.DeletedAt != nil {
			response.Error(c, http.StatusNotFound, "driver not found")
			return
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, d)
	}
}

// ListDrivers returns paginated list of drivers (non-deleted). GET /drivers.
func ListDrivers(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := 20
		offset := 0
		if l := c.Query("limit"); l != "" {
			if n, err := parseInt(l); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}
		if o := c.Query("offset"); o != "" {
			if n, err := parseInt(o); err == nil && n >= 0 {
				offset = n
			}
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		rows, err := pool.Query(ctx, `
			SELECT id::text, first_name, last_name, phone_number, passport_series, passport_number,
				company_id::text, freelance_dispatcher_id::text, car_photo_path, adr_document_path, rating, work_status, account_status, language,
				platform, dispatcher_type, last_activated_at, last_activated_latitude, last_activated_longitude,
				created_at::text, updated_at::text, deleted_at
			FROM drivers WHERE deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`, limit, offset)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		defer rows.Close()

		var list []Driver
		for rows.Next() {
			d, err := scanDriver(rows)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, "internal error")
				return
			}
			list = append(list, d)
		}
		if list == nil {
			list = []Driver{}
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, list)
	}
}

// GetDriverByPhone returns one driver by phone_number. GET /drivers/phone/:phone.
func GetDriverByPhone(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		phone := c.Param("phone")
		if phone == "" {
			response.Error(c, http.StatusBadRequest, "phone is required")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		row := pool.QueryRow(ctx, `
			SELECT id::text, first_name, last_name, phone_number, passport_series, passport_number,
				company_id::text, freelance_dispatcher_id::text, car_photo_path, adr_document_path, rating, work_status, account_status, language,
				platform, dispatcher_type, last_activated_at, last_activated_latitude, last_activated_longitude,
				created_at::text, updated_at::text, deleted_at
			FROM drivers WHERE phone_number = $1 AND deleted_at IS NULL
		`, phone)
		d, err := scanDriver(row)
		if err != nil {
			if isNoRows(err) {
				response.Error(c, http.StatusNotFound, "driver not found")
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, d)
	}
}

// UpdateDriverFull full update. PUT /drivers/:id. Accepts JSON or multipart (fields + optional files adr_document, car_photo; remove_adr_document, remove_car_photo to clear).
func UpdateDriverFull(pool *pgxpool.Pool, storageRoot string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid id")
			return
		}
		var req UpdateDriverRequest
		isMultipart := strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data")
		if isMultipart {
			req = bindUpdateDriverFromForm(c)
			if req.FirstName == nil || req.LastName == nil || req.PhoneNumber == nil || req.PassportSeries == nil || req.PassportNumber == nil {
				response.Error(c, http.StatusBadRequest, "first_name, last_name, phone_number, passport_series, passport_number are required")
				return
			}
		} else {
			if err := c.ShouldBindJSON(&req); err != nil {
				response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
				return
			}
			if req.FirstName == nil || req.LastName == nil || req.PhoneNumber == nil ||
				req.PassportSeries == nil || req.PassportNumber == nil {
				response.Error(c, http.StatusBadRequest, "first_name, last_name, phone_number, passport_series, passport_number are required")
				return
			}
		}
		rating := 0.0
		if req.Rating != nil {
			rating = *req.Rating
			if !validateRating(rating) {
				response.Error(c, http.StatusBadRequest, "rating must be between 0 and 5")
				return
			}
		}
		if req.WorkStatus != nil && !validWorkStatus[*req.WorkStatus] {
			response.Error(c, http.StatusBadRequest, "work_status must be free or busy")
			return
		}
		if req.AccountStatus != nil && !validAccountStatus[*req.AccountStatus] {
			response.Error(c, http.StatusBadRequest, "account_status must be pending, approved or blocked")
			return
		}
		if req.Language != nil && !validLanguage[*req.Language] {
			response.Error(c, http.StatusBadRequest, "language must be ru, uz, en, tr or zh")
			return
		}
		workStatus := WorkStatusFree
		if req.WorkStatus != nil {
			workStatus = *req.WorkStatus
		}
		accountStatus := AccountStatusPending
		if req.AccountStatus != nil {
			accountStatus = *req.AccountStatus
		}
		language := middleware.LanguageFrom(c.Request.Context())
		if language == "" || !validLanguage[language] {
			language = LangRU
		}
		if req.Language != nil {
			language = *req.Language
		}
		platform := middleware.PlatformFrom(c.Request.Context())
		if req.Platform != nil {
			platform = *req.Platform
		}
		var companyID, freelanceDispatcherID *uuid.UUID
		if req.CompanyID != nil && *req.CompanyID != "" {
			parsed, err := uuid.Parse(*req.CompanyID)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "company_id must be a valid UUID")
				return
			}
			companyID = &parsed
		}
		if req.FreelanceDispatcherID != nil && *req.FreelanceDispatcherID != "" {
			parsed, err := uuid.Parse(*req.FreelanceDispatcherID)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "freelance_dispatcher_id must be a valid UUID")
				return
			}
			freelanceDispatcherID = &parsed
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		cmd, err := pool.Exec(ctx, `
			UPDATE drivers SET
				first_name = $1, last_name = $2, phone_number = $3, passport_series = $4, passport_number = $5,
				company_id = $6, freelance_dispatcher_id = $7, rating = $8, work_status = $9, account_status = $10, language = $11, platform = $12,
				updated_at = now()
			WHERE id = $13 AND deleted_at IS NULL
		`, *req.FirstName, *req.LastName, *req.PhoneNumber, *req.PassportSeries, *req.PassportNumber,
			companyID, freelanceDispatcherID, rating, workStatus, accountStatus, language, platform, id)
		if err != nil {
			if isUniqueViolation(err) {
				response.Error(c, http.StatusBadRequest, "phone_number already exists")
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		if cmd.RowsAffected() == 0 {
			response.Error(c, http.StatusNotFound, "driver not found")
			return
		}
		if isMultipart && handleDriverFilesAfterUpdate(c, ctx, pool, storageRoot, id) {
			return
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, nil)
	}
}

// UpdateDriverPartial partial update. PATCH /drivers/:id. Accepts JSON or multipart (optional fields + optional files; remove_* to clear).
func UpdateDriverPartial(pool *pgxpool.Pool, storageRoot string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid id")
			return
		}
		var req UpdateDriverRequest
		isMultipart := strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data")
		if isMultipart {
			req = bindUpdateDriverFromForm(c)
		} else {
			if err := c.ShouldBindJSON(&req); err != nil {
				response.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
				return
			}
		}
		if req.Rating != nil && !validateRating(*req.Rating) {
			response.Error(c, http.StatusBadRequest, "rating must be between 0 and 5")
			return
		}
		if req.WorkStatus != nil && !validWorkStatus[*req.WorkStatus] {
			response.Error(c, http.StatusBadRequest, "work_status must be free or busy")
			return
		}
		if req.AccountStatus != nil && !validAccountStatus[*req.AccountStatus] {
			response.Error(c, http.StatusBadRequest, "account_status must be pending, approved or blocked")
			return
		}
		if req.Language != nil && !validLanguage[*req.Language] {
			response.Error(c, http.StatusBadRequest, "language must be ru, uz, en, tr or zh")
			return
		}
		// Язык и платформа из заголовков, если не переданы в теле
		if req.Language == nil {
			if lang := middleware.LanguageFrom(c.Request.Context()); validLanguage[lang] {
				req.Language = &lang
			}
		}
		if req.Platform == nil {
			if p := middleware.PlatformFrom(c.Request.Context()); p != "" {
				req.Platform = &p
			}
		}
		var companyID, freelanceDispatcherID *uuid.UUID
		if req.CompanyID != nil && *req.CompanyID != "" {
			parsed, err := uuid.Parse(*req.CompanyID)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "company_id must be a valid UUID")
				return
			}
			companyID = &parsed
		}
		if req.FreelanceDispatcherID != nil && *req.FreelanceDispatcherID != "" {
			parsed, err := uuid.Parse(*req.FreelanceDispatcherID)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "freelance_dispatcher_id must be a valid UUID")
				return
			}
			freelanceDispatcherID = &parsed
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		// Build dynamic update (only non-nil fields). If only files sent (multipart), updates may be empty.
		updates, args := buildDriverUpdate(&req, companyID, freelanceDispatcherID)
		if len(updates) > 0 {
			args = append(args, id)
			q := `UPDATE drivers SET ` + updates + `, updated_at = now() WHERE id = $` + itoa(len(args)) + ` AND deleted_at IS NULL`
			cmd, err := pool.Exec(ctx, q, args...)
			if err != nil {
				if isUniqueViolation(err) {
					response.Error(c, http.StatusBadRequest, "phone_number already exists")
					return
				}
				response.Error(c, http.StatusInternalServerError, "internal error")
				return
			}
			if cmd.RowsAffected() == 0 {
				response.Error(c, http.StatusNotFound, "driver not found")
				return
			}
		} else if isMultipart {
			// Only files: verify driver exists
			var one int
			if err := pool.QueryRow(ctx, `SELECT 1 FROM drivers WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&one); err != nil {
				response.Error(c, http.StatusNotFound, "driver not found")
				return
			}
		} else {
			response.Success(c, http.StatusOK, response.MsgSuccess, nil)
			return
		}
		if isMultipart && handleDriverFilesAfterUpdate(c, ctx, pool, storageRoot, id) {
			return
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, nil)
	}
}

func bindUpdateDriverFromForm(c *gin.Context) UpdateDriverRequest {
	var req UpdateDriverRequest
	if v := c.PostForm("first_name"); v != "" {
		req.FirstName = &v
	}
	if v := c.PostForm("last_name"); v != "" {
		req.LastName = &v
	}
	if v := c.PostForm("phone_number"); v != "" {
		req.PhoneNumber = &v
	}
	if v := c.PostForm("passport_series"); v != "" {
		req.PassportSeries = &v
	}
	if v := c.PostForm("passport_number"); v != "" {
		req.PassportNumber = &v
	}
	if v := c.PostForm("company_id"); v != "" {
		req.CompanyID = &v
	}
	if v := c.PostForm("freelance_dispatcher_id"); v != "" {
		req.FreelanceDispatcherID = &v
	}
	if v := c.PostForm("rating"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			req.Rating = &f
		}
	}
	if v := c.PostForm("work_status"); v != "" {
		req.WorkStatus = &v
	}
	if v := c.PostForm("account_status"); v != "" {
		req.AccountStatus = &v
	}
	if v := c.PostForm("language"); v != "" {
		req.Language = &v
	}
	if v := c.PostForm("platform"); v != "" {
		req.Platform = &v
	}
	return req
}

// handleDriverFilesAfterUpdate processes remove_* flags and file uploads after PUT/PATCH (multipart). Returns true if response was sent (error).
func handleDriverFilesAfterUpdate(c *gin.Context, ctx context.Context, pool *pgxpool.Pool, storageRoot string, id uuid.UUID) bool {
	removeADR := c.PostForm("remove_adr_document")
	if removeADR == "1" || removeADR == "true" || removeADR == "yes" {
		_ = RemoveDriverFile(ctx, pool, storageRoot, id, "adr_document_path")
	}
	removeCar := c.PostForm("remove_car_photo")
	if removeCar == "1" || removeCar == "true" || removeCar == "yes" {
		_ = RemoveDriverFile(ctx, pool, storageRoot, id, "car_photo_path")
	}
	if adrFile, _ := c.FormFile("adr_document"); adrFile != nil {
		if err := SaveDriverFileFromMultipart(ctx, pool, storageRoot, id, adrFile, "adr_document_path", "adr_document", adrMaxSize, adrAllowed); err != nil {
			if errors.Is(err, errFileTooLarge) {
				response.Error(c, http.StatusBadRequest, "adr_document: file too large (max 10 MB)")
			} else if errors.Is(err, errInvalidFileType) {
				response.Error(c, http.StatusBadRequest, "adr_document: invalid type (PDF, JPG, PNG only)")
			} else {
				response.Error(c, http.StatusInternalServerError, "internal error")
			}
			return true
		}
	}
	if carFile, _ := c.FormFile("car_photo"); carFile != nil {
		if err := SaveDriverFileFromMultipart(ctx, pool, storageRoot, id, carFile, "car_photo_path", "car_photo", carPhotoMaxSize, carAllowed); err != nil {
			if errors.Is(err, errFileTooLarge) {
				response.Error(c, http.StatusBadRequest, "car_photo: file too large (max 5 MB)")
			} else if errors.Is(err, errInvalidFileType) {
				response.Error(c, http.StatusBadRequest, "car_photo: invalid type (JPG, PNG only)")
			} else {
				response.Error(c, http.StatusInternalServerError, "internal error")
			}
			return true
		}
	}
	return false
}

func buildDriverUpdate(req *UpdateDriverRequest, companyID, freelanceDispatcherID *uuid.UUID) (string, []interface{}) {
	var set []string
	var args []interface{}
	n := 1
	if req.FirstName != nil {
		set = append(set, "first_name = $"+itoa(n))
		args = append(args, *req.FirstName)
		n++
	}
	if req.LastName != nil {
		set = append(set, "last_name = $"+itoa(n))
		args = append(args, *req.LastName)
		n++
	}
	if req.PhoneNumber != nil {
		set = append(set, "phone_number = $"+itoa(n))
		args = append(args, *req.PhoneNumber)
		n++
	}
	if req.PassportSeries != nil {
		set = append(set, "passport_series = $"+itoa(n))
		args = append(args, *req.PassportSeries)
		n++
	}
	if req.PassportNumber != nil {
		set = append(set, "passport_number = $"+itoa(n))
		args = append(args, *req.PassportNumber)
		n++
	}
	if companyID != nil || (req.CompanyID != nil && *req.CompanyID == "") {
		set = append(set, "company_id = $"+itoa(n))
		if companyID != nil {
			args = append(args, *companyID)
		} else {
			args = append(args, nil)
		}
		n++
	}
	if freelanceDispatcherID != nil || (req.FreelanceDispatcherID != nil && *req.FreelanceDispatcherID == "") {
		set = append(set, "freelance_dispatcher_id = $"+itoa(n))
		if freelanceDispatcherID != nil {
			args = append(args, *freelanceDispatcherID)
		} else {
			args = append(args, nil)
		}
		n++
	}
	// car_photo and adr_document are set only via multipart file upload, not JSON
	if req.Rating != nil {
		set = append(set, "rating = $"+itoa(n))
		args = append(args, *req.Rating)
		n++
	}
	if req.WorkStatus != nil {
		set = append(set, "work_status = $"+itoa(n))
		args = append(args, *req.WorkStatus)
		n++
	}
	if req.AccountStatus != nil {
		set = append(set, "account_status = $"+itoa(n))
		args = append(args, *req.AccountStatus)
		n++
	}
	if req.Language != nil {
		set = append(set, "language = $"+itoa(n))
		args = append(args, *req.Language)
		n++
	}
	if req.Platform != nil {
		set = append(set, "platform = $"+itoa(n))
		args = append(args, *req.Platform)
		n++
	}
	if len(set) == 0 {
		return "", nil
	}
	return stringsJoin(set, ", "), args
}

// LastActivateRequest — тело PATCH /drivers/:id/last-activate (время и координаты последней активации на карте).
type LastActivateRequest struct {
	ActivatedAt *string   `json:"activated_at"` // RFC3339
	Latitude    *float64  `json:"latitude"`
	Longitude   *float64  `json:"longitude"`
}

// PatchDriverLastActivate обновляет последнюю активацию водителя: время и координаты. Мобильное/фронт передаёт дату-время и координаты на карте.
func PatchDriverLastActivate(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid id")
			return
		}
		var req LastActivateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.ActivatedAt == nil && req.Latitude == nil && req.Longitude == nil {
			response.Error(c, http.StatusBadRequest, "at least one of activated_at, latitude, longitude is required")
			return
		}
		var activatedAt *time.Time
		if req.ActivatedAt != nil && *req.ActivatedAt != "" {
			t, err := time.Parse(time.RFC3339, *req.ActivatedAt)
			if err != nil {
				response.Error(c, http.StatusBadRequest, "activated_at must be RFC3339 (e.g. 2025-02-02T12:00:00Z)")
				return
			}
			activatedAt = &t
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		var set []string
		var args []interface{}
		n := 1
		if activatedAt != nil {
			set = append(set, "last_activated_at = $"+itoa(n))
			args = append(args, activatedAt)
			n++
		}
		if req.Latitude != nil {
			set = append(set, "last_activated_latitude = $"+itoa(n))
			args = append(args, *req.Latitude)
			n++
		}
		if req.Longitude != nil {
			set = append(set, "last_activated_longitude = $"+itoa(n))
			args = append(args, *req.Longitude)
			n++
		}
		if len(set) == 0 {
			response.Success(c, http.StatusOK, response.MsgSuccess, nil)
			return
		}
		args = append(args, id)
		_, err = pool.Exec(ctx, `UPDATE drivers SET `+stringsJoin(set, ", ")+`, updated_at = now() WHERE id = $`+itoa(n)+` AND deleted_at IS NULL`, args...)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, nil)
	}
}

// DeleteDriver переносит водителя в deleted_drivers и удаляет из drivers (hard delete).
func DeleteDriver(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid id")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		// Перенос в архив (deleted_drivers), затем hard delete из drivers
		cmd, err := pool.Exec(ctx, `
			INSERT INTO deleted_drivers (
				id, first_name, last_name, phone_number, passport_series, passport_number,
				company_id, freelance_dispatcher_id, car_photo_path, adr_document_path,
				rating, work_status, account_status, language, platform, dispatcher_type,
				last_activated_at, last_activated_latitude, last_activated_longitude,
				created_at, updated_at, archived_at
			)
			SELECT id, first_name, last_name, phone_number, passport_series, passport_number,
				company_id, freelance_dispatcher_id, car_photo_path, adr_document_path,
				rating, work_status, account_status, language, platform, dispatcher_type,
				last_activated_at, last_activated_latitude, last_activated_longitude,
				created_at, updated_at, now()
			FROM drivers WHERE id = $1 AND deleted_at IS NULL
		`, id)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		if cmd.RowsAffected() == 0 {
			response.Error(c, http.StatusNotFound, "driver not found")
			return
		}
		_, err = pool.Exec(ctx, `DELETE FROM drivers WHERE id = $1`, id)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, nil)
	}
}

// helpers
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
func itoa(n int) string { return strconv.Itoa(n) }
func stringsJoin(a []string, sep string) string { return strings.Join(a, sep) }

func isNoRows(err error) bool   { return errors.Is(err, pgx.ErrNoRows) }
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
