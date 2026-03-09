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

	OwnerID     *uuid.UUID // app_users id; when set, status is set to active
	CompanyType *string    // CargoOwner, Carrier, Expeditor (or Shipper, Broker, Fleet, OwnerOperator)
	CreatedBy   uuid.UUID
}

func (r *Repo) Create(ctx context.Context, p CreateParams) (uuid.UUID, error) {
	status := "pending"
	if p.Status != nil && *p.Status != "" {
		status = *p.Status
	}
	if p.OwnerID != nil {
		status = "active" // после добавления owner статус переводится в active
	}
	const q = `
INSERT INTO companies (
  name, inn, address, phone, email, website, license_number,
  status, owner_id, company_type,
  max_vehicles, max_drivers, max_cargo, max_dispatchers, max_managers, max_top_dispatchers, max_top_managers,
  completed_orders, cancelled_orders, total_revenue,
  created_by, created_at, updated_at, deleted_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7,
  $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17,
  0, 0, 0,
  $18, now(), now(), NULL
) RETURNING id`

	var id uuid.UUID
	err := r.pg.QueryRow(ctx, q,
		p.Name, p.Inn, p.Address, p.Phone, p.Email, p.Website, p.LicenseNumber,
		status, p.OwnerID, p.CompanyType,
		p.MaxVehicles, p.MaxDrivers, p.MaxCargo, p.MaxDispatchers, p.MaxManagers, p.MaxTopDispatchers, p.MaxTopManagers,
		p.CreatedBy,
	).Scan(&id)
	return id, err
}

// CompanyTZ is the TZ company row (owner_id, company_type, auto_approve_limit).
type CompanyTZ struct {
	ID                uuid.UUID  `json:"id"`
	Name              string     `json:"name"`
	Type              *string    `json:"type"`
	AutoApproveLimit  *float64   `json:"auto_approve_limit"`
	OwnerID           *uuid.UUID `json:"owner_id"`
	Phone             *string    `json:"phone"`
	Address           *string    `json:"address"`
	Inn               *string    `json:"inn"`
	CreatedAt         string     `json:"created_at"`
	UpdatedAt         string     `json:"updated_at"`
}

// CreateByOwnerParams for TZ: user creates company and becomes Owner.
type CreateByOwnerParams struct {
	Name             string
	Type             string // Shipper, Broker, Fleet, OwnerOperator
	AutoApproveLimit *float64
	OwnerID          uuid.UUID
	Phone            *string
	Address          *string
	Inn              *string
}

// CreateByOwner inserts a company with owner_id and company_type (TZ flow).
func (r *Repo) CreateByOwner(ctx context.Context, p CreateByOwnerParams) (uuid.UUID, error) {
	const q = `
INSERT INTO companies (name, company_type, auto_approve_limit, owner_id, phone, address, inn, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, 'active', now(), now())
RETURNING id`
	var id uuid.UUID
	err := r.pg.QueryRow(ctx, q, p.Name, p.Type, p.AutoApproveLimit, p.OwnerID, p.Phone, p.Address, p.Inn).Scan(&id)
	return id, err
}

// GetByIDTZ returns company by id (for TZ: id, name, type, owner_id, etc.).
func (r *Repo) GetByIDTZ(ctx context.Context, id uuid.UUID) (*CompanyTZ, error) {
	const q = `SELECT id, name, company_type, auto_approve_limit, owner_id, phone, address, inn, created_at::text, updated_at::text
FROM companies WHERE id = $1 AND deleted_at IS NULL LIMIT 1`
	var c CompanyTZ
	err := r.pg.QueryRow(ctx, q, id).Scan(&c.ID, &c.Name, &c.Type, &c.AutoApproveLimit, &c.OwnerID, &c.Phone, &c.Address, &c.Inn, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// SetOwner sets company owner_id and sets status to active (for admin: link owner to company).
func (r *Repo) SetOwner(ctx context.Context, companyID, ownerID uuid.UUID) error {
	const q = `UPDATE companies SET owner_id = $2, status = 'active', updated_at = now() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.pg.Exec(ctx, q, companyID, ownerID)
	return err
}

// CreateWithOwnerDispatcher creates a company owned by a freelance dispatcher (Broker). Used for "create own company" flow.
func (r *Repo) CreateWithOwnerDispatcher(ctx context.Context, name string, ownerDispatcherID uuid.UUID) (uuid.UUID, error) {
	const q = `
INSERT INTO companies (name, company_type, status, owner_dispatcher_id, created_at, updated_at)
VALUES ($1, 'Broker', 'active', $2, now(), now())
RETURNING id`
	var id uuid.UUID
	err := r.pg.QueryRow(ctx, q, name, ownerDispatcherID).Scan(&id)
	return id, err
}

// ListForDispatcher returns companies where dispatcher is owner (owner_dispatcher_id) or has role in dispatcher_company_roles.
func (r *Repo) ListForDispatcher(ctx context.Context, dispatcherID uuid.UUID) ([]CompanyWithRole, error) {
	const q = `
SELECT c.id, c.name, c.company_type, c.owner_id, COALESCE(dcr.role, 'owner') as role_name
FROM companies c
LEFT JOIN dispatcher_company_roles dcr ON dcr.company_id = c.id AND dcr.dispatcher_id = $1
WHERE c.deleted_at IS NULL AND (c.owner_dispatcher_id = $1 OR dcr.dispatcher_id = $1)
ORDER BY (c.owner_dispatcher_id = $1) DESC, c.name`
	rows, err := r.pg.Query(ctx, q, dispatcherID, dispatcherID, dispatcherID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []CompanyWithRole
	for rows.Next() {
		var row CompanyWithRole
		var roleName string
		var ownerID *uuid.UUID
		err := rows.Scan(&row.ID, &row.Name, &row.Type, &ownerID, &roleName)
		if err != nil {
			return nil, err
		}
		if ownerID != nil {
			row.OwnerID = *ownerID
		}
		row.Role = roleName
		list = append(list, row)
	}
	return list, rows.Err()
}

// ListForUser returns companies where user is owner or has a role (for GET /auth/companies).
func (r *Repo) ListForUser(ctx context.Context, userID uuid.UUID) ([]CompanyWithRole, error) {
	const q = `
SELECT c.id, c.name, c.company_type, c.owner_id, r.name as role_name
FROM companies c
LEFT JOIN user_company_roles ucr ON ucr.company_id = c.id AND ucr.user_id = $1
LEFT JOIN app_roles r ON r.id = ucr.role_id
WHERE c.deleted_at IS NULL AND (c.owner_id = $1 OR ucr.user_id = $1)
ORDER BY (c.owner_id = $1) DESC, c.name`
	rows, err := r.pg.Query(ctx, q, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []CompanyWithRole
	for rows.Next() {
		var row CompanyWithRole
		var roleName *string
		err := rows.Scan(&row.ID, &row.Name, &row.Type, &row.OwnerID, &roleName)
		if err != nil {
			return nil, err
		}
		if roleName != nil {
			row.Role = *roleName
		} else if row.OwnerID == userID {
			row.Role = "Owner"
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

type CompanyWithRole struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	Type    *string   `json:"type"`
	OwnerID uuid.UUID `json:"owner_id"`
	Role    string    `json:"role"`
}

