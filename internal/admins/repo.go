package admins

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("admin not found")

type Repo struct {
	pg *pgxpool.Pool
}

func NewRepo(pg *pgxpool.Pool) *Repo {
	return &Repo{pg: pg}
}

func (r *Repo) FindByLogin(ctx context.Context, login string) (*Admin, error) {
	const q = `
SELECT id, login, password, name, status, type
FROM admins
WHERE login = $1
LIMIT 1`
	var a Admin
	err := r.pg.QueryRow(ctx, q, login).Scan(&a.ID, &a.Login, &a.Password, &a.Name, &a.Status, &a.Type)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

