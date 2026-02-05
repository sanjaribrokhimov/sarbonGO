// Drivers Profile: API для водителя по JWT (все операции только над своим профилем). Client token + Bearer.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sarbonGO/backend/internal/middleware"
	"github.com/sarbonGO/backend/internal/response"
)

const profileFileBase = "/api/v1/drivers/profile/files"

// getDriverIDFromContext возвращает driver_id из JWT (в контексте лежит как user_id).
func getDriverIDFromContext(c *gin.Context) string {
	id, _ := c.Get(string(middleware.ContextKeyUserID))
	if s, ok := id.(string); ok {
		return s
	}
	return ""
}

// GetDriverProfile возвращает полный профиль текущего водителя (все поля из таблицы). GET /drivers/profile.
func GetDriverProfile(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := getDriverIDFromContext(c)
		if driverID == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

		row := pool.QueryRow(ctx, `
			SELECT id::text, first_name, last_name, phone_number, passport_series, passport_number,
				company_id::text, freelance_dispatcher_id::text, car_photo_path, adr_document_path, rating, work_status, account_status, language,
				platform, dispatcher_type, last_activated_at, last_activated_latitude, last_activated_longitude,
				created_at::text, updated_at::text, deleted_at
			FROM drivers WHERE id = $1 AND deleted_at IS NULL
		`, driverID)
		d, err := scanDriverWithFileBase(row, profileFileBase)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				response.Error(c, http.StatusNotFound, "driver not found")
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, d)
	}
}

// UpdateDriverProfileRequest — поля для обновления профиля (кроме phone_number).
type UpdateDriverProfileRequest struct {
	FirstName              *string  `json:"first_name"`
	LastName               *string  `json:"last_name"`
	PassportSeries         *string  `json:"passport_series"`
	PassportNumber         *string  `json:"passport_number"`
	CompanyID              *string  `json:"company_id"`
	FreelanceDispatcherID  *string  `json:"freelance_dispatcher_id"`
	Rating                 *float64 `json:"rating"`
	WorkStatus             *string  `json:"work_status"`
	AccountStatus          *string  `json:"account_status"`
	Language               *string  `json:"language"`
	Platform               *string  `json:"platform"`
	DispatcherType         *string  `json:"dispatcher_type"`
}

// UpdateDriverProfile обновляет профиль текущего водителя (все поля кроме phone_number). PUT /drivers/profile. JSON или multipart (файлы car_photo, adr_document).
func UpdateDriverProfile(pool *pgxpool.Pool, storageRoot string) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := getDriverIDFromContext(c)
		if driverID == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		id, _ := uuid.Parse(driverID)
		var req UpdateDriverProfileRequest
		isMultipart := strings.HasPrefix(c.GetHeader("Content-Type"), "multipart/form-data")
		if isMultipart {
			req = bindUpdateDriverProfileFromForm(c)
		} else {
			if err := c.ShouldBindJSON(&req); err != nil {
				response.Error(c, http.StatusBadRequest, "invalid request body")
				return
			}
		}
		// Язык и платформа из заголовков, если не переданы
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
		ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
		defer cancel()

		updates, args := buildDriverProfileUpdate(&req)
		if len(updates) > 0 {
			args = append(args, driverID)
			q := `UPDATE drivers SET ` + updates + `, updated_at = now() WHERE id = $` + itoa(len(args)) + ` AND deleted_at IS NULL`
			_, err := pool.Exec(ctx, q, args...)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, "internal error")
				return
			}
		}
		if isMultipart {
			handleDriverProfileFiles(c, ctx, pool, storageRoot, id)
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, nil)
	}
}

func buildDriverProfileUpdate(req *UpdateDriverProfileRequest) (string, []interface{}) {
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
	if req.CompanyID != nil {
		set = append(set, "company_id = $"+itoa(n))
		if *req.CompanyID == "" {
			args = append(args, nil)
		} else {
			args = append(args, *req.CompanyID)
		}
		n++
	}
	if req.FreelanceDispatcherID != nil {
		set = append(set, "freelance_dispatcher_id = $"+itoa(n))
		if *req.FreelanceDispatcherID == "" {
			args = append(args, nil)
		} else {
			args = append(args, *req.FreelanceDispatcherID)
		}
		n++
	}
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
	if req.DispatcherType != nil {
		set = append(set, "dispatcher_type = $"+itoa(n))
		args = append(args, *req.DispatcherType)
		n++
	}
	return stringsJoin(set, ", "), args
}

func bindUpdateDriverProfileFromForm(c *gin.Context) UpdateDriverProfileRequest {
	var req UpdateDriverProfileRequest
	if v := c.PostForm("first_name"); v != "" {
		req.FirstName = &v
	}
	if v := c.PostForm("last_name"); v != "" {
		req.LastName = &v
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
	if v := c.PostForm("dispatcher_type"); v != "" {
		req.DispatcherType = &v
	}
	return req
}

func handleDriverProfileFiles(c *gin.Context, ctx context.Context, pool *pgxpool.Pool, storageRoot string, driverID uuid.UUID) {
	if adrFile, _ := c.FormFile("adr_document"); adrFile != nil {
		_ = SaveDriverFileFromMultipart(ctx, pool, storageRoot, driverID, adrFile, "adr_document_path", "adr_document", adrMaxSize, adrAllowed)
	}
	if carFile, _ := c.FormFile("car_photo"); carFile != nil {
		_ = SaveDriverFileFromMultipart(ctx, pool, storageRoot, driverID, carFile, "car_photo_path", "car_photo", carPhotoMaxSize, carAllowed)
	}
	if c.PostForm("remove_adr_document") == "1" || c.PostForm("remove_adr_document") == "true" {
		_ = RemoveDriverFile(ctx, pool, storageRoot, driverID, "adr_document_path")
	}
	if c.PostForm("remove_car_photo") == "1" || c.PostForm("remove_car_photo") == "true" {
		_ = RemoveDriverFile(ctx, pool, storageRoot, driverID, "car_photo_path")
	}
}

// DeleteDriverProfile удаляет свой профиль: перенос в deleted_drivers, затем удаление из drivers. DELETE /drivers/profile.
func DeleteDriverProfile(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := getDriverIDFromContext(c)
		if driverID == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		id, err := uuid.Parse(driverID)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid id")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()

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

// GetDriverProfileLastActivate возвращает время и координаты последней активации. GET /drivers/profile/last-activate.
func GetDriverProfileLastActivate(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := getDriverIDFromContext(c)
		if driverID == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var activatedAt *time.Time
		var lat, lng *float64
		err := pool.QueryRow(ctx, `
			SELECT last_activated_at, last_activated_latitude, last_activated_longitude
			FROM drivers WHERE id = $1 AND deleted_at IS NULL
		`, driverID).Scan(&activatedAt, &lat, &lng)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				response.Error(c, http.StatusNotFound, "driver not found")
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		out := gin.H{}
		if activatedAt != nil {
			out["last_activated_at"] = activatedAt.Format(time.RFC3339)
		}
		if lat != nil {
			out["last_activated_latitude"] = *lat
		}
		if lng != nil {
			out["last_activated_longitude"] = *lng
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, out)
	}
}

// PatchDriverProfileLastActivate обновляет последнюю активацию текущего водителя. PATCH /drivers/profile/last-activate.
func PatchDriverProfileLastActivate(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := getDriverIDFromContext(c)
		if driverID == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
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
				response.Error(c, http.StatusBadRequest, "activated_at must be RFC3339")
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
		args = append(args, driverID)
		_, err := pool.Exec(ctx, `UPDATE drivers SET `+stringsJoin(set, ", ")+`, updated_at = now() WHERE id = $`+itoa(n)+` AND deleted_at IS NULL`, args...)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		response.Success(c, http.StatusOK, response.MsgSuccess, nil)
	}
}

// GetDriverProfileCarPhoto отдаёт файл фото машины текущего водителя. GET /drivers/profile/files/car-photo.
func GetDriverProfileCarPhoto(pool *pgxpool.Pool, storageRoot string) gin.HandlerFunc {
	return getDriverProfileFile(pool, storageRoot, "car_photo_path")
}

// GetDriverProfileAdrDocument отдаёт файл ADR текущего водителя. GET /drivers/profile/files/adr-document.
func GetDriverProfileAdrDocument(pool *pgxpool.Pool, storageRoot string) gin.HandlerFunc {
	return getDriverProfileFile(pool, storageRoot, "adr_document_path")
}

func getDriverProfileFile(pool *pgxpool.Pool, storageRoot, pathColumn string) gin.HandlerFunc {
	return func(c *gin.Context) {
		driverID := getDriverIDFromContext(c)
		if driverID == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized")
			return
		}
		id, err := uuid.Parse(driverID)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "invalid id")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var relPath string
		err = pool.QueryRow(ctx, `SELECT `+pathColumn+` FROM drivers WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&relPath)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				response.Error(c, http.StatusNotFound, "driver or file not found")
				return
			}
			response.Error(c, http.StatusInternalServerError, "internal error")
			return
		}
		if relPath == "" {
			response.Error(c, http.StatusNotFound, "file not found")
			return
		}
		absPath := filepath.Join(storageRoot, relPath)
		c.File(absPath)
	}
}
