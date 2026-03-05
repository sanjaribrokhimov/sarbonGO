package companytz

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type UserCompanyRole struct {
	UserID     uuid.UUID `json:"user_id"`
	CompanyID  uuid.UUID `json:"company_id"`
	RoleID     uuid.UUID `json:"role_id"`
	AssignedBy *uuid.UUID `json:"assigned_by"`
	AssignedAt string    `json:"assigned_at"`
}

type RepoUCR struct {
	pg *pgxpool.Pool
}

func NewRepoUCR(pg *pgxpool.Pool) *RepoUCR {
	return &RepoUCR{pg: pg}
}

func (r *RepoUCR) Add(ctx context.Context, userID, companyID, roleID, assignedBy uuid.UUID) error {
	_, err := r.pg.Exec(ctx, `
INSERT INTO user_company_roles (user_id, company_id, role_id, assigned_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, company_id, role_id) DO UPDATE SET role_id = EXCLUDED.role_id, assigned_by = EXCLUDED.assigned_by, assigned_at = CURRENT_TIMESTAMP`,
		userID, companyID, roleID, assignedBy)
	return err
}

func (r *RepoUCR) GetRole(ctx context.Context, userID, companyID uuid.UUID) (roleID uuid.UUID, err error) {
	err = r.pg.QueryRow(ctx, `SELECT role_id FROM user_company_roles WHERE user_id = $1 AND company_id = $2 LIMIT 1`, userID, companyID).Scan(&roleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, ErrNotFound
		}
		return uuid.Nil, err
	}
	return roleID, nil
}

func (r *RepoUCR) Remove(ctx context.Context, userID, companyID uuid.UUID) error {
	cmd, err := r.pg.Exec(ctx, `DELETE FROM user_company_roles WHERE user_id = $1 AND company_id = $2`, userID, companyID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RepoUCR) UpdateRole(ctx context.Context, userID, companyID, newRoleID, assignedBy uuid.UUID) error {
	cmd, err := r.pg.Exec(ctx, `
UPDATE user_company_roles SET role_id = $3, assigned_by = $4, assigned_at = CURRENT_TIMESTAMP
WHERE user_id = $1 AND company_id = $2`,
		userID, companyID, newRoleID, assignedBy)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *RepoUCR) ListUsersByCompany(ctx context.Context, companyID uuid.UUID, limit, offset int) ([]UserCompanyRoleWithUser, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pg.Query(ctx, `
SELECT ucr.user_id, ucr.company_id, ucr.role_id, ucr.assigned_by, ucr.assigned_at,
  u.phone, u.first_name, u.last_name,
  r.name as role_name, r.description as role_description,
  ab.first_name as assigned_by_first_name, ab.last_name as assigned_by_last_name
FROM user_company_roles ucr
JOIN company_users u ON u.id = ucr.user_id
JOIN app_roles r ON r.id = ucr.role_id
LEFT JOIN company_users ab ON ab.id = ucr.assigned_by
WHERE ucr.company_id = $1
ORDER BY ucr.assigned_at DESC
LIMIT $2 OFFSET $3`,
		companyID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []UserCompanyRoleWithUser
	for rows.Next() {
		var row UserCompanyRoleWithUser
		var abFn, abLn *string
		err := rows.Scan(&row.UserID, &row.CompanyID, &row.RoleID, &row.AssignedBy, &row.AssignedAt,
			&row.Phone, &row.FirstName, &row.LastName,
			&row.RoleName, &row.RoleDescription,
			&abFn, &abLn)
		if err != nil {
			return nil, err
		}
		if abFn != nil && abLn != nil {
			s := *abFn + " " + *abLn
			row.AssignedByName = &s
		}
		list = append(list, row)
	}
	return list, rows.Err()
}

type UserCompanyRoleWithUser struct {
	UserID          uuid.UUID `json:"user_id"`
	CompanyID       uuid.UUID `json:"company_id"`
	RoleID          uuid.UUID `json:"role_id"`
	AssignedBy      *uuid.UUID `json:"assigned_by"`
	AssignedAt      string    `json:"assigned_at"`
	Phone           string    `json:"phone"`
	FirstName       *string   `json:"first_name"`
	LastName        *string   `json:"last_name"`
	RoleName        string    `json:"role_name"`
	RoleDescription *string   `json:"role_description"`
	AssignedByName  *string   `json:"assigned_by_name"`
}
