package approles

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("role not found")

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

func (r *Repo) FindByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	const q = `SELECT id, name, description FROM app_roles WHERE id = $1 LIMIT 1`
	var ro Role
	err := r.pg.QueryRow(ctx, q, id).Scan(&ro.ID, &ro.Name, &ro.Description)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ro, nil
}

func (r *Repo) FindByName(ctx context.Context, name string) (*Role, error) {
	const q = `SELECT id, name, description FROM app_roles WHERE name = $1 LIMIT 1`
	var ro Role
	err := r.pg.QueryRow(ctx, q, name).Scan(&ro.ID, &ro.Name, &ro.Description)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &ro, nil
}

func (r *Repo) ListAll(ctx context.Context) ([]Role, error) {
	rows, err := r.pg.Query(ctx, `SELECT id, name, description FROM app_roles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Role
	for rows.Next() {
		var ro Role
		if err := rows.Scan(&ro.ID, &ro.Name, &ro.Description); err != nil {
			return nil, err
		}
		list = append(list, ro)
	}
	return list, rows.Err()
}
