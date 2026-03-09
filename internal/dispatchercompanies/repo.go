package dispatchercompanies

import (
	"context"
	"errors"

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

// Add links dispatcher to company with role (owner, dispatcher, top_dispatcher). Sets accepted_at = now().
func (r *Repo) Add(ctx context.Context, dispatcherID, companyID uuid.UUID, role string) error {
	const q = `
INSERT INTO dispatcher_company_roles (dispatcher_id, company_id, role, accepted_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (dispatcher_id, company_id) DO UPDATE SET role = $3, accepted_at = COALESCE(dispatcher_company_roles.accepted_at, now())`
	_, err := r.pg.Exec(ctx, q, dispatcherID, companyID, role)
	return err
}

// HasAccess returns true if dispatcher has any role in company (or company.owner_dispatcher_id = dispatcher).
func (r *Repo) HasAccess(ctx context.Context, dispatcherID, companyID uuid.UUID) (bool, error) {
	var n int
	err := r.pg.QueryRow(ctx,
		`SELECT 1 FROM companies c
WHERE c.id = $2 AND c.deleted_at IS NULL AND (c.owner_dispatcher_id = $1 OR EXISTS (SELECT 1 FROM dispatcher_company_roles d WHERE d.dispatcher_id = $1 AND d.company_id = $2))`,
		dispatcherID, companyID).Scan(&n)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
