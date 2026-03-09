package trips

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("trip not found")
var ErrInvalidTransition = errors.New("invalid status transition")

var allowedTransitions = map[string][]string{
	StatusPendingDriver: {StatusAssigned, StatusCancelled},
	StatusAssigned:      {StatusLoading, StatusCancelled},
	StatusLoading:       {StatusEnRoute, StatusCancelled},
	StatusEnRoute:       {StatusUnloading},
	StatusUnloading:     {StatusCompleted},
	StatusCompleted:     nil,
	StatusCancelled:     nil,
}

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

// Create creates trip with status pending_driver (after offer accepted).
func (r *Repo) Create(ctx context.Context, cargoID, offerID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.pg.QueryRow(ctx,
		`INSERT INTO trips (cargo_id, offer_id, status) VALUES ($1, $2, $3) RETURNING id`,
		cargoID, offerID, StatusPendingDriver).Scan(&id)
	return id, err
}

// GetByID returns trip by id.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*Trip, error) {
	var t Trip
	err := r.pg.QueryRow(ctx,
		`SELECT id, cargo_id, offer_id, driver_id, status, created_at, updated_at FROM trips WHERE id = $1`,
		id).Scan(&t.ID, &t.CargoID, &t.OfferID, &t.DriverID, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// GetByOfferID returns trip by offer_id (unique).
func (r *Repo) GetByOfferID(ctx context.Context, offerID uuid.UUID) (*Trip, error) {
	var t Trip
	err := r.pg.QueryRow(ctx,
		`SELECT id, cargo_id, offer_id, driver_id, status, created_at, updated_at FROM trips WHERE offer_id = $1`,
		offerID).Scan(&t.ID, &t.CargoID, &t.OfferID, &t.DriverID, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// GetByCargoID returns trip for cargo (at most one active).
func (r *Repo) GetByCargoID(ctx context.Context, cargoID uuid.UUID) (*Trip, error) {
	var t Trip
	err := r.pg.QueryRow(ctx,
		`SELECT id, cargo_id, offer_id, driver_id, status, created_at, updated_at FROM trips WHERE cargo_id = $1 ORDER BY created_at DESC LIMIT 1`,
		cargoID).Scan(&t.ID, &t.CargoID, &t.OfferID, &t.DriverID, &t.Status, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

// AssignDriver sets driver_id (dispatcher assigns driver). Trip must be pending_driver.
func (r *Repo) AssignDriver(ctx context.Context, tripID, driverID uuid.UUID) error {
	res, err := r.pg.Exec(ctx,
		`UPDATE trips SET driver_id = $2, updated_at = now() WHERE id = $1 AND status = $3`,
		tripID, driverID, StatusPendingDriver)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DriverConfirm sets status to assigned (driver accepted the assignment). Trip must have driver_id = caller and stay pending_driver until driver confirms.
// So actually: when dispatcher assigns driver, we only set driver_id. Driver then "confirms" and we set status = assigned.
func (r *Repo) DriverConfirm(ctx context.Context, tripID, driverID uuid.UUID) error {
	res, err := r.pg.Exec(ctx,
		`UPDATE trips SET status = $2, updated_at = now() WHERE id = $1 AND driver_id = $3 AND status = $4`,
		tripID, StatusAssigned, driverID, StatusPendingDriver)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DriverReject clears driver_id so dispatcher can assign another driver.
func (r *Repo) DriverReject(ctx context.Context, tripID, driverID uuid.UUID) error {
	res, err := r.pg.Exec(ctx,
		`UPDATE trips SET driver_id = NULL, updated_at = now() WHERE id = $1 AND driver_id = $2 AND status = $3`,
		tripID, driverID, StatusPendingDriver)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetStatus updates trip status (driver: loading -> en_route -> unloading -> completed).
func (r *Repo) SetStatus(ctx context.Context, tripID uuid.UUID, newStatus string) error {
	t, err := r.GetByID(ctx, tripID)
	if err != nil || t == nil {
		return ErrNotFound
	}
	allowed := allowedTransitions[t.Status]
	for _, s := range allowed {
		if s == newStatus {
			_, err = r.pg.Exec(ctx, `UPDATE trips SET status = $1, updated_at = now() WHERE id = $2`, newStatus, tripID)
			return err
		}
	}
	return ErrInvalidTransition
}

// ListByDriver returns trips for driver (where driver_id = driverID).
func (r *Repo) ListByDriver(ctx context.Context, driverID uuid.UUID, limit int) ([]Trip, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.pg.Query(ctx,
		`SELECT id, cargo_id, offer_id, driver_id, status, created_at, updated_at FROM trips WHERE driver_id = $1 ORDER BY created_at DESC LIMIT $2`,
		driverID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Trip
	for rows.Next() {
		var t Trip
		err := rows.Scan(&t.ID, &t.CargoID, &t.OfferID, &t.DriverID, &t.Status, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}

// ListByCargoIDs returns trips for given cargo IDs (for dispatcher listing by cargo).
func (r *Repo) ListByCargoIDs(ctx context.Context, cargoIDs []uuid.UUID) ([]Trip, error) {
	if len(cargoIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pg.Query(ctx,
		`SELECT id, cargo_id, offer_id, driver_id, status, created_at, updated_at FROM trips WHERE cargo_id = ANY($1) ORDER BY created_at DESC`,
		cargoIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Trip
	for rows.Next() {
		var t Trip
		err := rows.Scan(&t.ID, &t.CargoID, &t.OfferID, &t.DriverID, &t.Status, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
	}
	return list, rows.Err()
}
