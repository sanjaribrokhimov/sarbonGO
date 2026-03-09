package dispatcherinvitations

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

type Invitation struct {
	ID         uuid.UUID
	Token      string
	CompanyID  uuid.UUID
	Role       string
	Phone      string
	InvitedBy  uuid.UUID
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

// Create creates invitation for dispatcher by phone. Role: dispatcher, top_dispatcher.
func (r *Repo) Create(ctx context.Context, companyID uuid.UUID, role, phone string, invitedBy uuid.UUID, expiresIn time.Duration) (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token = hex.EncodeToString(b)
	expiresAt := time.Now().Add(expiresIn)
	_, err = r.pg.Exec(ctx,
		`INSERT INTO dispatcher_invitations (token, company_id, role, phone, invited_by, expires_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		token, companyID, role, phone, invitedBy, expiresAt)
	return token, err
}

// GetByToken returns invitation if valid (not expired).
func (r *Repo) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	var i Invitation
	err := r.pg.QueryRow(ctx,
		`SELECT id, token, company_id, role, phone, invited_by, expires_at, created_at FROM dispatcher_invitations WHERE token = $1`,
		token).Scan(&i.ID, &i.Token, &i.CompanyID, &i.Role, &i.Phone, &i.InvitedBy, &i.ExpiresAt, &i.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if time.Now().After(i.ExpiresAt) {
		return nil, nil
	}
	return &i, nil
}

// Delete removes invitation (after accept or decline).
func (r *Repo) Delete(ctx context.Context, token string) error {
	_, err := r.pg.Exec(ctx, `DELETE FROM dispatcher_invitations WHERE token = $1`, token)
	return err
}
