package appusers

import (
	"context"
	"errors"

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

func isUniqueViolation(err error) bool {
	var e *pgconn.PgError
	return errors.As(err, &e) && e.SQLState() == "23505"
}
