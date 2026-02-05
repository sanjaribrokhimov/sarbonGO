// Driver file upload/download: ADR document and car photo. Paths stored in DB, files in storage/drivers/{id}/.
package handlers

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sarbonGO/backend/internal/response"
)

const (
	adrMaxSize      = 10 << 20 // 10 MB
	carPhotoMaxSize = 5 << 20  // 5 MB
)

var (
	adrAllowed = map[string]bool{"pdf": true, "jpg": true, "jpeg": true, "png": true}
	carAllowed = map[string]bool{"jpg": true, "jpeg": true, "png": true}
)

func extFromFilename(name string) string {
	e := strings.ToLower(filepath.Ext(name))
	if e == "" {
		return ""
	}
	return strings.TrimPrefix(e, ".")
}

// ValidateDriverFile проверяет файл (размер и тип) без сохранения. Возвращает errFileTooLarge или errInvalidFileType при ошибке.
func ValidateDriverFile(file *multipart.FileHeader, maxBytes int64, allowedExt map[string]bool) error {
	if file == nil || file.Size <= 0 {
		return errors.New("file is required")
	}
	if file.Size > maxBytes {
		return errFileTooLarge
	}
	ext := extFromFilename(file.Filename)
	if ext == "" || !allowedExt[ext] {
		return errInvalidFileType
	}
	return nil
}

// SaveDriverFileFromMultipart saves a multipart file for a driver: deletes old file if any, saves new, updates path in DB.
// Used from CreateDriver/UpdateDriver when frontend sends adr_document or car_photo in the same request.
func SaveDriverFileFromMultipart(ctx context.Context, pool *pgxpool.Pool, storageRoot string, driverID uuid.UUID, file *multipart.FileHeader, pathColumn, baseName string, maxBytes int64, allowedExt map[string]bool) error {
	if file == nil || file.Size <= 0 {
		return nil
	}
	if file.Size > maxBytes {
		return errFileTooLarge
	}
	ext := extFromFilename(file.Filename)
	if ext == "" || !allowedExt[ext] {
		return errInvalidFileType
	}
	// Delete old file if exists
	var oldPath string
	_ = pool.QueryRow(ctx, `SELECT `+pathColumn+` FROM drivers WHERE id = $1`, driverID).Scan(&oldPath)
	if oldPath != "" {
		_ = deleteFileByRelativePath(storageRoot, oldPath)
	}
	dir := filepath.Join(storageRoot, "drivers", driverID.String())
	if err := mkdirAll(dir); err != nil {
		return err
	}
	storagePath := filepath.Join(dir, baseName+"."+ext)
	relativePath := filepath.Join("drivers", driverID.String(), baseName+"."+ext)
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	if err := saveToFile(storagePath, src, file.Size); err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `UPDATE drivers SET `+pathColumn+` = $1, updated_at = now() WHERE id = $2`, relativePath, driverID)
	return err
}

// RemoveDriverFile deletes file from disk and clears path in DB. Used when frontend sends remove_adr_document or remove_car_photo.
func RemoveDriverFile(ctx context.Context, pool *pgxpool.Pool, storageRoot string, driverID uuid.UUID, pathColumn string) error {
	var relPath string
	err := pool.QueryRow(ctx, `SELECT `+pathColumn+` FROM drivers WHERE id = $1`, driverID).Scan(&relPath)
	if err != nil {
		return err
	}
	if relPath != "" {
		_ = deleteFileByRelativePath(storageRoot, relPath)
	}
	_, err = pool.Exec(ctx, `UPDATE drivers SET `+pathColumn+` = NULL, updated_at = now() WHERE id = $1`, driverID)
	return err
}

var errFileTooLarge = errors.New("file too large")
var errInvalidFileType = errors.New("invalid file type")

func deleteFileByRelativePath(storageRoot, relPath string) error {
	if relPath == "" {
		return nil
	}
	absPath := filepath.Join(storageRoot, relPath)
	cleanAbs := filepath.Clean(absPath)
	cleanRoot := filepath.Clean(storageRoot)
	if cleanRoot != "" && !strings.HasPrefix(cleanAbs, cleanRoot) {
		return nil
	}
	return os.Remove(absPath)
}

// GetADRDocument serves the ADR file. GET /drivers/:id/files/adr-document.
func GetADRDocument(pool *pgxpool.Pool, storageRoot string) gin.HandlerFunc {
	return getDriverFile(pool, storageRoot, "adr_document_path")
}

// GetCarPhoto serves the car photo. GET /drivers/:id/files/car-photo.
func GetCarPhoto(pool *pgxpool.Pool, storageRoot string) gin.HandlerFunc {
	return getDriverFile(pool, storageRoot, "car_photo_path")
}

func getDriverFile(pool *pgxpool.Pool, storageRoot, pathColumn string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := uuid.Parse(idStr)
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
				response.Error(c, http.StatusNotFound, "driver not found")
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
		if !strings.HasPrefix(filepath.Clean(absPath), filepath.Clean(storageRoot)) {
			response.Error(c, http.StatusNotFound, "not found")
			return
		}
		c.File(absPath)
	}
}

func mkdirAll(dir string) error {
	return os.MkdirAll(dir, 0755)
}

func saveToFile(dest string, src io.Reader, maxBytes int64) error {
	dst, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, io.LimitReader(src, maxBytes))
	return err
}
