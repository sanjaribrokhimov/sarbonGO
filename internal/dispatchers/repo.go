package dispatchers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound              = errors.New("dispatcher not found")
	ErrPhoneAlreadyRegistered = errors.New("phone already registered")
)

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

func (r *Repo) FindByPhone(ctx context.Context, phone string) (*Dispatcher, error) {
	const q = `
SELECT
  id, name, phone, password,
  passport_series, passport_number, pinfl,
  cargo_id, driver_id,
  rating, work_status, account_status AS status,
  photo_path AS photo,
  created_at, updated_at, deleted_at
FROM freelance_dispatchers
WHERE phone = $1 AND deleted_at IS NULL
LIMIT 1`
	d, err := scanDispatcher(r.pg.QueryRow(ctx, q, phone))
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (r *Repo) FindByID(ctx context.Context, id uuid.UUID) (*Dispatcher, error) {
	const q = `
SELECT
  id, name, phone, password,
  passport_series, passport_number, pinfl,
  cargo_id, driver_id,
  rating, work_status, account_status AS status,
  photo_path AS photo,
  created_at, updated_at, deleted_at
FROM freelance_dispatchers
WHERE id = $1 AND deleted_at IS NULL
LIMIT 1`
	d, err := scanDispatcher(r.pg.QueryRow(ctx, q, id))
	if err != nil {
		return nil, err
	}
	return d, nil
}

func scanDispatcher(row pgx.Row) (*Dispatcher, error) {
	var d Dispatcher
	err := row.Scan(
		&d.ID, &d.Name, &d.Phone, &d.Password,
		&d.PassportSeries, &d.PassportNumber, &d.PINFL,
		&d.CargoID, &d.DriverID,
		&d.Rating, &d.WorkStatus, &d.Status,
		&d.Photo,
		&d.CreatedAt, &d.UpdatedAt, &d.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &d, nil
}

type CreateParams struct {
	Phone          string
	Name           string
	PasswordHash   string
	PassportSeries string
	PassportNumber string
	PINFL          string
	Photo          *string
}

func (r *Repo) Create(ctx context.Context, p CreateParams) (uuid.UUID, error) {
	const q = `
INSERT INTO freelance_dispatchers (
  phone, name, password,
  passport_series, passport_number, pinfl,
  photo_path,
  rating, work_status, account_status,
  created_at, updated_at, deleted_at
) VALUES (
  $1, $2, $3,
  $4, $5, $6,
  $7,
  0, 'available', 'active',
  now(), now(), NULL
) RETURNING id`

	var id uuid.UUID
	err := r.pg.QueryRow(ctx, q,
		p.Phone, p.Name, p.PasswordHash,
		p.PassportSeries, p.PassportNumber, p.PINFL,
		p.Photo,
	).Scan(&id)
	if err != nil {
		if e, ok := err.(*pgconn.PgError); ok && e.SQLState() == "23505" {
			return uuid.Nil, ErrPhoneAlreadyRegistered
		}
		return uuid.Nil, err
	}
	return id, nil
}

type UpdateProfileParams struct {
	Name           *string
	PassportSeries *string
	PassportNumber *string
	PINFL          *string
	Photo          *string
}

func (r *Repo) UpdateProfile(ctx context.Context, id uuid.UUID, p UpdateProfileParams) error {
	const q = `
UPDATE freelance_dispatchers
SET name = COALESCE($2, name),
    passport_series = COALESCE($3, passport_series),
    passport_number = COALESCE($4, passport_number),
    pinfl = COALESCE($5, pinfl),
    photo_path = COALESCE($6, photo_path),
    updated_at = now()
WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, p.Name, p.PassportSeries, p.PassportNumber, p.PINFL, p.Photo)
	return err
}

func (r *Repo) UpdatePasswordHash(ctx context.Context, id uuid.UUID, passwordHash string) error {
	const q = `UPDATE freelance_dispatchers SET password = $2, updated_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, passwordHash)
	return err
}

func (r *Repo) UpdatePhone(ctx context.Context, id uuid.UUID, newPhone string) error {
	const q = `UPDATE freelance_dispatchers SET phone = $2, updated_at = now() WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, newPhone)
	if err != nil {
		if e, ok := err.(*pgconn.PgError); ok && e.SQLState() == "23505" {
			return ErrPhoneAlreadyRegistered
		}
		return err
	}
	return nil
}

var ErrDeleteNotFound = errors.New("dispatcher to delete not found")

func (r *Repo) DeleteAndArchive(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `UPDATE freelance_dispatchers SET deleted_at = now(), updated_at = now() WHERE id = $1`, id); err != nil {
		return err
	}

	tag, err := tx.Exec(ctx, `INSERT INTO deleted_freelance_dispatchers SELECT * FROM freelance_dispatchers WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrDeleteNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM freelance_dispatchers WHERE id = $1`, id); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Repo) Touch(ctx context.Context, id uuid.UUID, t time.Time) error {
	const q = `UPDATE freelance_dispatchers SET updated_at = $2 WHERE id = $1`
	_, err := r.pg.Exec(ctx, q, id, t)
	return err
}
