package cargorecommendations

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

const statusPending = "PENDING"
const statusAccepted = "ACCEPTED"
const statusDeclined = "DECLINED"

// Create creates a recommendation (dispatcher recommends cargo to driver). Unique (cargo_id, driver_id).
func (r *Repo) Create(ctx context.Context, cargoID, driverID, dispatcherID uuid.UUID) error {
	_, err := r.pg.Exec(ctx,
		`INSERT INTO cargo_driver_recommendations (cargo_id, driver_id, invited_by_dispatcher_id, status)
VALUES ($1, $2, $3, $4)
ON CONFLICT (cargo_id, driver_id) DO UPDATE SET invited_by_dispatcher_id = $3, status = $4`,
		cargoID, driverID, dispatcherID, statusPending)
	return err
}

// Recommendation row.
type Recommendation struct {
	ID                    uuid.UUID
	CargoID               uuid.UUID
	DriverID              uuid.UUID
	InvitedByDispatcherID uuid.UUID
	Status                string
	CreatedAt             time.Time
}

// GetByCargoAndDriver returns recommendation if exists.
func (r *Repo) GetByCargoAndDriver(ctx context.Context, cargoID, driverID uuid.UUID) (*Recommendation, error) {
	var rec Recommendation
	var createdAt time.Time
	err := r.pg.QueryRow(ctx,
		`SELECT id, cargo_id, driver_id, invited_by_dispatcher_id, status, created_at
FROM cargo_driver_recommendations WHERE cargo_id = $1 AND driver_id = $2`,
		cargoID, driverID).Scan(&rec.ID, &rec.CargoID, &rec.DriverID, &rec.InvitedByDispatcherID, &rec.Status, &createdAt)
	if err == nil {
		rec.CreatedAt = createdAt
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

// ListByDriver returns pending recommendations for the driver.
func (r *Repo) ListByDriver(ctx context.Context, driverID uuid.UUID, limit int) ([]Recommendation, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pg.Query(ctx,
		`SELECT id, cargo_id, driver_id, invited_by_dispatcher_id, status, created_at
FROM cargo_driver_recommendations WHERE driver_id = $1 AND status = $2 ORDER BY created_at DESC LIMIT $3`,
		driverID, statusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Recommendation
	for rows.Next() {
		var rec Recommendation
		if err := rows.Scan(&rec.ID, &rec.CargoID, &rec.DriverID, &rec.InvitedByDispatcherID, &rec.Status, &rec.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, rec)
	}
	return list, rows.Err()
}

// Accept sets status to accepted. Returns cargo_id. Fails if not pending.
func (r *Repo) Accept(ctx context.Context, cargoID, driverID uuid.UUID) (bool, error) {
	res, err := r.pg.Exec(ctx,
		`UPDATE cargo_driver_recommendations SET status = $1 WHERE cargo_id = $2 AND driver_id = $3 AND status = $4`,
		statusAccepted, cargoID, driverID, statusPending)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}

// Decline sets status to declined.
func (r *Repo) Decline(ctx context.Context, cargoID, driverID uuid.UUID) (bool, error) {
	res, err := r.pg.Exec(ctx,
		`UPDATE cargo_driver_recommendations SET status = $1 WHERE cargo_id = $2 AND driver_id = $3 AND status = $4`,
		statusDeclined, cargoID, driverID, statusPending)
	if err != nil {
		return false, err
	}
	return res.RowsAffected() > 0, nil
}
