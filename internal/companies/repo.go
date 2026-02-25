package companies

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

type CreateParams struct {
	Name          string
	Inn           *string
	Address       *string
	Phone         *string
	Email         *string
	Website       *string
	LicenseNumber *string
	Status        *string

	MaxVehicles       int
	MaxDrivers        int
	MaxCargo          int
	MaxDispatchers    int
	MaxManagers       int
	MaxTopDispatchers int
	MaxTopManagers    int

	CreatedBy uuid.UUID
}

func (r *Repo) Create(ctx context.Context, p CreateParams) (uuid.UUID, error) {
	const q = `
INSERT INTO companies (
  name, inn, address, phone, email, website, license_number,
  status,
  max_vehicles, max_drivers, max_cargo, max_dispatchers, max_managers, max_top_dispatchers, max_top_managers,
  completed_orders, cancelled_orders, total_revenue,
  created_by, created_at, updated_at, deleted_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7,
  COALESCE($8, 'active'),
  $9, $10, $11, $12, $13, $14, $15,
  0, 0, 0,
  $16, now(), now(), NULL
) RETURNING id`

	var id uuid.UUID
	err := r.pg.QueryRow(ctx, q,
		p.Name, p.Inn, p.Address, p.Phone, p.Email, p.Website, p.LicenseNumber,
		p.Status,
		p.MaxVehicles, p.MaxDrivers, p.MaxCargo, p.MaxDispatchers, p.MaxManagers, p.MaxTopDispatchers, p.MaxTopManagers,
		p.CreatedBy,
	).Scan(&id)
	return id, err
}

