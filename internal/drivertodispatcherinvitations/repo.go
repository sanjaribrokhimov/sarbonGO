package drivertodispatcherinvitations

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
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

func normalizePhone(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "-", ""))
	return strings.TrimPrefix(s, "+")
}

// Create creates an invitation from driver to dispatcher (by dispatcher phone). Returns token.
func (r *Repo) Create(ctx context.Context, driverID uuid.UUID, dispatcherPhone string, expiresIn time.Duration) (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token = hex.EncodeToString(b)
	expiresAt := time.Now().Add(expiresIn)
	_, err = r.pg.Exec(ctx,
		`INSERT INTO driver_to_dispatcher_invitations (token, driver_id, dispatcher_phone, expires_at) VALUES ($1, $2, $3, $4)`,
		token, driverID, strings.TrimSpace(dispatcherPhone), expiresAt)
	return token, err
}

// Invitation is a driver-to-dispatcher invitation row.
type Invitation struct {
	ID              uuid.UUID
	Token           string
	DriverID        uuid.UUID
	DispatcherPhone string
	ExpiresAt       time.Time
	CreatedAt       time.Time
}

// GetByToken returns invitation if not expired.
func (r *Repo) GetByToken(ctx context.Context, token string) (*Invitation, error) {
	var i Invitation
	err := r.pg.QueryRow(ctx,
		`SELECT id, token, driver_id, dispatcher_phone, expires_at, created_at FROM driver_to_dispatcher_invitations WHERE token = $1`,
		token).Scan(&i.ID, &i.Token, &i.DriverID, &i.DispatcherPhone, &i.ExpiresAt, &i.CreatedAt)
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

// Delete removes invitation after accept or decline.
func (r *Repo) Delete(ctx context.Context, token string) error {
	_, err := r.pg.Exec(ctx, `DELETE FROM driver_to_dispatcher_invitations WHERE token = $1`, token)
	return err
}

// ListByDriverID returns non-expired invitations sent by this driver.
func (r *Repo) ListByDriverID(ctx context.Context, driverID uuid.UUID) ([]Invitation, error) {
	rows, err := r.pg.Query(ctx,
		`SELECT id, token, driver_id, dispatcher_phone, expires_at, created_at
		 FROM driver_to_dispatcher_invitations WHERE expires_at > now() AND driver_id = $1 ORDER BY created_at DESC`,
		driverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Invitation
	for rows.Next() {
		var i Invitation
		if err := rows.Scan(&i.ID, &i.Token, &i.DriverID, &i.DispatcherPhone, &i.ExpiresAt, &i.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

// ListByDispatcherPhone returns non-expired invitations sent TO this dispatcher phone (normalized match).
func (r *Repo) ListByDispatcherPhone(ctx context.Context, dispatcherPhone string) ([]Invitation, error) {
	norm := normalizePhone(dispatcherPhone)
	if norm == "" {
		return []Invitation{}, nil
	}
	rows, err := r.pg.Query(ctx,
		`SELECT id, token, driver_id, dispatcher_phone, expires_at, created_at
		 FROM driver_to_dispatcher_invitations
		 WHERE expires_at > now() AND replace(replace(replace(trim(dispatcher_phone), ' ', ''), '-', ''), '+', '') = $1
		 ORDER BY created_at DESC`,
		norm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Invitation
	for rows.Next() {
		var i Invitation
		if err := rows.Scan(&i.ID, &i.Token, &i.DriverID, &i.DispatcherPhone, &i.ExpiresAt, &i.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, i)
	}
	return list, rows.Err()
}

// PhoneMatches returns true if invitation's dispatcher_phone matches the given phone (normalized).
func (i *Invitation) PhoneMatches(phone string) bool {
	return normalizePhone(i.DispatcherPhone) == normalizePhone(phone)
}
