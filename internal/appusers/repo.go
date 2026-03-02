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
var ErrEmailExists = errors.New("email already registered")

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

func (r *Repo) Create(ctx context.Context, email, passwordHash string, firstName, lastName, phone *string) (*User, error) {
	const q = `
INSERT INTO app_users (email, password_hash, first_name, last_name, phone)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, phone, password_hash, first_name, last_name, created_at, updated_at`
	var u User
	err := r.pg.QueryRow(ctx, q, email, passwordHash, firstName, lastName, phone).Scan(
		&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailExists
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repo) FindByEmail(ctx context.Context, email string) (*User, error) {
	const q = `SELECT id, email, phone, password_hash, first_name, last_name, created_at, updated_at FROM app_users WHERE email = $1 LIMIT 1`
	var u User
	err := r.pg.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *Repo) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	const q = `SELECT id, email, phone, password_hash, first_name, last_name, created_at, updated_at FROM app_users WHERE id = $1 LIMIT 1`
	var u User
	err := r.pg.QueryRow(ctx, q, id).Scan(&u.ID, &u.Email, &u.Phone, &u.PasswordHash, &u.FirstName, &u.LastName, &u.CreatedAt, &u.UpdatedAt)
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
