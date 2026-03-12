package driverinvitations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// Create creates driver invitation by company (company_id set). invitedBy = dispatcher or company user id.
func (r *Repo) Create(ctx context.Context, companyID uuid.UUID, phone string, invitedBy uuid.UUID, expiresIn time.Duration) (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token = hex.EncodeToString(b)
	expiresAt := time.Now().Add(expiresIn)
	_, err = r.pg.Exec(ctx,
		`INSERT INTO driver_invitations (token, company_id, phone, invited_by, expires_at) VALUES ($1, $2, $3, $4, $5)`,
		token, companyID, phone, invitedBy, expiresAt)
	return token, err
}

// CreateForFreelance creates driver invitation by freelance dispatcher (no company). Driver will get freelancer_id on accept.
func (r *Repo) CreateForFreelance(ctx context.Context, dispatcherID uuid.UUID, phone string, expiresIn time.Duration) (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token = hex.EncodeToString(b)
	expiresAt := time.Now().Add(expiresIn)
	_, err = r.pg.Exec(ctx,
		`INSERT INTO driver_invitations (token, company_id, phone, invited_by, invited_by_dispatcher_id, expires_at) VALUES ($1, NULL, $2, $3, $3, $4)`,
		token, phone, dispatcherID, expiresAt)
	return token, err
}

// GetByToken returns invitation if not expired.
func (r *Repo) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	var i Invitation
	var companyID *uuid.UUID
	var invDispatcherID *uuid.UUID
	err := r.pg.QueryRow(ctx,
		`SELECT id, token, company_id, phone, invited_by, invited_by_dispatcher_id, expires_at, created_at FROM driver_invitations WHERE token = $1`,
		token).Scan(&i.ID, &i.Token, &companyID, &i.Phone, &i.InvitedBy, &invDispatcherID, &i.ExpiresAt, &i.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if time.Now().After(i.ExpiresAt) {
		return nil, nil
	}
	i.CompanyID = companyID
	i.InvitedByDispatcherID = invDispatcherID
	return &i, nil
}

type Invitation struct {
	ID                     uuid.UUID
	Token                  string
	CompanyID              *uuid.UUID // nil when invited by freelance dispatcher
	Phone                  string
	InvitedBy              uuid.UUID
	InvitedByDispatcherID  *uuid.UUID // set when freelance dispatcher invites (no company)
	ExpiresAt              time.Time
	CreatedAt              time.Time
}

// Delete removes invitation after accept or decline.
func (r *Repo) Delete(ctx context.Context, token string) error {
	_, err := r.pg.Exec(ctx, `DELETE FROM driver_invitations WHERE token = $1`, token)
	return err
}

// ListByPhone returns non-expired invitations for the given phone (для водителя: список приглашений в чате).
func (r *Repo) ListByPhone(ctx context.Context, phone string) ([]Invitation, error) {
	rows, err := r.pg.Query(ctx,
		`SELECT id, token, company_id, phone, invited_by, invited_by_dispatcher_id, expires_at, created_at
		 FROM driver_invitations WHERE expires_at > now() AND trim(replace(phone, ' ', '')) = trim(replace($1, ' ', ''))
		 ORDER BY created_at DESC`,
		phone)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Invitation
	for rows.Next() {
		var i Invitation
		var companyID, invDispID *uuid.UUID
		if err := rows.Scan(&i.ID, &i.Token, &companyID, &i.Phone, &i.InvitedBy, &invDispID, &i.ExpiresAt, &i.CreatedAt); err != nil {
			return nil, err
		}
		i.CompanyID = companyID
		i.InvitedByDispatcherID = invDispID
		list = append(list, i)
	}
	return list, rows.Err()
}
