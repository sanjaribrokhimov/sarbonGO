package appusers

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("user not found")
var ErrPhoneExists = errors.New("phone already registered")

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

func (r *Repo) Create(ctx context.Context, phone, passwordHash string, firstName, lastName *string, companyID *uuid.UUID, role string) (*User, error) {
	const q = `
INSERT INTO company_users (phone, password_hash, first_name, last_name, company_id, role)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, phone, password_hash, first_name, last_name, company_id, role, created_at, updated_at`
	var u User
	err := r.pg.QueryRow(ctx, q, phone, passwordHash, firstName, lastName, companyID, role).Scan(
		&u.ID, &u.Phone, &u.PasswordHash, &u.FirstName, &u.LastName, &u.CompanyID, &u.Role, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrPhoneExists
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repo) FindByPhone(ctx context.Context, phone string) (*User, error) {
	const q = `SELECT id, phone, password_hash, first_name, last_name, company_id, role, created_at, updated_at FROM company_users WHERE phone = $1 LIMIT 1`
	var u User
	err := r.pg.QueryRow(ctx, q, phone).Scan(&u.ID, &u.Phone, &u.PasswordHash, &u.FirstName, &u.LastName, &u.CompanyID, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repo) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const q = `SELECT id, phone, password_hash, first_name, last_name, company_id, role, created_at, updated_at FROM company_users WHERE id = $1 LIMIT 1`
	var u User
	err := r.pg.QueryRow(ctx, q, id).Scan(&u.ID, &u.Phone, &u.PasswordHash, &u.FirstName, &u.LastName, &u.CompanyID, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// UpdateCompanyID обновляет company_id у пользователя (для двусторонней связи при назначении владельца).
func (r *Repo) UpdateCompanyID(ctx context.Context, userID uuid.UUID, companyID *uuid.UUID) error {
	const q = `UPDATE company_users SET company_id = $2, updated_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, userID, companyID)
	return err
}

// escapeLike экранирует % и _ для использования в LIKE/ILIKE (ESCAPE '\').
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "%", `\%`)
	s = strings.ReplaceAll(s, "_", `\_`)
	return s
}

// SearchOwners ищет в company_users только пользователей с role = 'owner' по телефону, имени или фамилии.
// Сортировка: сначала точное совпадение, затем "начинается с", затем "содержит" (максимально похожие первые).
func (r *Repo) SearchOwners(ctx context.Context, query string, limit int) ([]User, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	pattern := "%" + escapeLike(query) + "%"
	patternStart := escapeLike(query) + "%"
	queryLower := strings.ToLower(query)
	const q = `
SELECT id, phone, password_hash, first_name, last_name, company_id, role, created_at, updated_at
FROM company_users
WHERE TRIM(COALESCE(role,'')) = 'OWNER'
  AND (phone ILIKE $1 ESCAPE '\' OR first_name ILIKE $1 ESCAPE '\' OR last_name ILIKE $1 ESCAPE '\')
ORDER BY
  (LOWER(TRIM(phone)) = $2 OR (first_name IS NOT NULL AND LOWER(TRIM(first_name)) = $2) OR (last_name IS NOT NULL AND LOWER(TRIM(last_name)) = $2)) DESC,
  (phone ILIKE $3 ESCAPE '\' OR first_name ILIKE $3 ESCAPE '\' OR last_name ILIKE $3 ESCAPE '\') DESC,
  (phone ILIKE $1 ESCAPE '\' OR first_name ILIKE $1 ESCAPE '\' OR last_name ILIKE $1 ESCAPE '\') DESC,
  phone
LIMIT $4`
	rows, err := r.pg.Query(ctx, q, pattern, queryLower, patternStart, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Phone, &u.PasswordHash, &u.FirstName, &u.LastName, &u.CompanyID, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func isUniqueViolation(err error) bool {
	var e *pgconn.PgError
	return errors.As(err, &e) && e.SQLState() == "23505"
}
