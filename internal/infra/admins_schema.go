package infra

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureAdminsTable creates minimal `admins` table if missing.
func EnsureAdminsTable(ctx context.Context, pg *pgxpool.Pool) error {
	if _, err := pg.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`); err != nil {
		return err
	}

	if _, err := pg.Exec(ctx, `
CREATE TABLE IF NOT EXISTS admins (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  login VARCHAR NOT NULL UNIQUE,
  password VARCHAR NOT NULL,
  name VARCHAR NOT NULL,
  status VARCHAR NOT NULL DEFAULT 'active',
  type VARCHAR NOT NULL DEFAULT 'creator'
);
`); err != nil {
		return err
	}

	// Constraint/indexes (idempotent)
	if _, err := pg.Exec(ctx, `
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'admins_status_check') THEN
    ALTER TABLE admins
      ADD CONSTRAINT admins_status_check CHECK (status IN ('active', 'inactive', 'blocked'));
  END IF;
END$$;
`); err != nil {
		return err
	}
	if _, err := pg.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_admins_login ON admins (login);`); err != nil {
		return err
	}
	return nil
}

