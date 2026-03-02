package companytz

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

var ErrInvitationNotFound = errors.New("invitation not found")
var ErrInvitationExpired = errors.New("invitation expired")

type Invitation struct {
	ID         uuid.UUID `json:"id"`
	Token      string    `json:"token"`
	CompanyID  uuid.UUID `json:"company_id"`
	RoleID     uuid.UUID `json:"role_id"`
	Email      string    `json:"email"`
	InvitedBy  uuid.UUID `json:"invited_by"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type RepoInvitations struct {
	pg *pgxpool.Pool
}

func NewRepoInvitations(pg *pgxpool.Pool) *RepoInvitations {
	return &RepoInvitations{pg: pg}
}

func genToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (r *RepoInvitations) Create(ctx context.Context, companyID, roleID, invitedBy uuid.UUID, email string, expiresIn time.Duration) (*Invitation, error) {
	token, err := genToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(expiresIn)
	const q = `
INSERT INTO invitations (token, company_id, role_id, email, invited_by, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, token, company_id, role_id, email, invited_by, expires_at, created_at`
	var inv Invitation
	err = r.pg.QueryRow(ctx, q, token, companyID, roleID, email, invitedBy, expiresAt).Scan(
		&inv.ID, &inv.Token, &inv.CompanyID, &inv.RoleID, &inv.Email, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *RepoInvitations) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	const q = `SELECT id, token, company_id, role_id, email, invited_by, expires_at, created_at FROM invitations WHERE token = $1 LIMIT 1`
	var inv Invitation
	err := r.pg.QueryRow(ctx, q, token).Scan(&inv.ID, &inv.Token, &inv.CompanyID, &inv.RoleID, &inv.Email, &inv.InvitedBy, &inv.ExpiresAt, &inv.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvitationNotFound
		}
		return nil, err
	}
	if time.Now().After(inv.ExpiresAt) {
		return nil, ErrInvitationExpired
	}
	return &inv, nil
}

func (r *RepoInvitations) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pg.Exec(ctx, `DELETE FROM invitations WHERE id = $1`, id)
	return err
}
