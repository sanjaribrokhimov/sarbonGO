// Все миграции в одном файле; порядок задаётся списком в migrations.go.
package migrations

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// 1 — schema_version + user_categories
func UpUserCategories(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_version (
			version INT PRIMARY KEY,
			name    TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_categories (
			id         SMALLSERIAL PRIMARY KEY,
			code       TEXT UNIQUE NOT NULL,
			name       TEXT NOT NULL,
			sort_order SMALLINT NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO user_categories (code, name, sort_order) VALUES
			('super_admin',   'Super Admin',    1),
			('admin',         'Admin',          2),
			('carrier',       'Carrier',        3),
			('top_dispatcher','Top Dispatcher', 4),
			('dispatcher',    'Dispatcher',     5),
			('driver',        'Driver',         6),
			('vehicle',       'Vehicle',        7),
			('manager',       'Manager',        8)
		ON CONFLICT (code) DO NOTHING
	`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO schema_version (version, name) VALUES (1, 'create_user_categories')
		ON CONFLICT (version) DO NOTHING
	`)
	return err
}

// 2 — drivers
func UpDrivers(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS drivers (
			id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			first_name        TEXT NOT NULL,
			last_name         TEXT NOT NULL,
			phone_number      TEXT NOT NULL UNIQUE,
			passport_series   TEXT NOT NULL,
			passport_number   TEXT NOT NULL,
			company_id            UUID,
			freelance_dispatcher_id UUID,
			car_photo_path    TEXT,
			adr_document_path TEXT,
			rating            REAL NOT NULL DEFAULT 0 CHECK (rating >= 0 AND rating <= 5),
			work_status       TEXT NOT NULL DEFAULT 'free' CHECK (work_status IN ('free', 'busy')),
			account_status    TEXT NOT NULL DEFAULT 'pending' CHECK (account_status IN ('pending', 'approved', 'blocked')),
			language          TEXT NOT NULL DEFAULT 'ru' CHECK (language IN ('ru', 'uz', 'en', 'tr', 'zh')),
			created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
			deleted_at        TIMESTAMPTZ
		)
	`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO schema_version (version, name) VALUES (2, 'create_drivers_table')
		ON CONFLICT (version) DO NOTHING
	`)
	return err
}

// 3 — drivers.freelance_dispatcher_id
func UpDriversFreelanceDispatcherID(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		ALTER TABLE drivers ADD COLUMN IF NOT EXISTS freelance_dispatcher_id UUID
	`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO schema_version (version, name) VALUES (3, 'drivers_freelance_dispatcher_id')
		ON CONFLICT (version) DO NOTHING
	`)
	return err
}

// 4 — drivers file paths
func UpDriversFilePaths(ctx context.Context, pool *pgxpool.Pool) error {
	for _, q := range []string{
		`ALTER TABLE drivers ADD COLUMN IF NOT EXISTS adr_document_path TEXT`,
		`ALTER TABLE drivers ADD COLUMN IF NOT EXISTS car_photo_path TEXT`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO schema_version (version, name) VALUES (4, 'drivers_file_paths')
		ON CONFLICT (version) DO NOTHING
	`)
	return err
}

// 5 — otp_codes, registration_sessions (session после OTP для complete-register), auth_tokens по driver_id (таблицы users нет)
func UpAuthSchema(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS otp_codes (
			id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			phone          TEXT NOT NULL,
			code           TEXT NOT NULL,
			expires_at     TIMESTAMPTZ NOT NULL,
			used_at        TIMESTAMPTZ,
			attempts_count INT NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		return err
	}
	_, _ = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_otp_codes_phone_active ON otp_codes (phone) WHERE used_at IS NULL`)

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS registration_sessions (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			phone      TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, _ = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_registration_sessions_phone ON registration_sessions (phone)`)

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS auth_tokens (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			driver_id     UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
			access_token  TEXT NOT NULL,
			refresh_token TEXT NOT NULL UNIQUE,
			expires_at    TIMESTAMPTZ NOT NULL
		)
	`)
	if err != nil {
		return err
	}
	_, _ = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_auth_tokens_driver_id ON auth_tokens (driver_id)`)

	_, err = pool.Exec(ctx, `
		INSERT INTO schema_version (version, name) VALUES (6, 'auth_schema')
		ON CONFLICT (version) DO NOTHING
	`)
	return err
}

// 6 — drivers: только platform, dispatcher_type (без user_id; таблицы users нет)
func UpDriversUserID(ctx context.Context, pool *pgxpool.Pool) error {
	for _, q := range []string{
		`ALTER TABLE drivers ADD COLUMN IF NOT EXISTS platform TEXT`,
		`ALTER TABLE drivers ADD COLUMN IF NOT EXISTS dispatcher_type TEXT`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO schema_version (version, name) VALUES (7, 'drivers_user_id')
		ON CONFLICT (version) DO NOTHING
	`)
	return err
}

// 7 — deleted_drivers (архив при hard delete)
func UpDeletedDrivers(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS deleted_drivers (
			id                     UUID PRIMARY KEY,
			first_name             TEXT NOT NULL,
			last_name              TEXT NOT NULL,
			phone_number           TEXT,
			passport_series        TEXT NOT NULL,
			passport_number        TEXT NOT NULL,
			company_id             UUID,
			freelance_dispatcher_id UUID,
			car_photo_path         TEXT,
			adr_document_path      TEXT,
			rating                 REAL NOT NULL DEFAULT 0,
			work_status            TEXT NOT NULL DEFAULT 'free',
			account_status         TEXT NOT NULL DEFAULT 'pending',
			language               TEXT NOT NULL DEFAULT 'ru',
			platform               TEXT,
			dispatcher_type        TEXT,
			created_at             TIMESTAMPTZ NOT NULL,
			updated_at             TIMESTAMPTZ NOT NULL,
			archived_at            TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO schema_version (version, name) VALUES (8, 'deleted_drivers')
		ON CONFLICT (version) DO NOTHING
	`)
	return err
}

// 8 — удалить таблицу users; мигрировать auth_tokens на driver_id при необходимости; убрать user_id из drivers
func UpUsersDropEmail(ctx context.Context, pool *pgxpool.Pool) error {
	// registration_sessions мог быть создан в auth_schema; на старых БД создаём если нет
	_, _ = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS registration_sessions (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			phone      TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL
		)
	`)
	_, _ = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_registration_sessions_phone ON registration_sessions (phone)`)

	// Миграция auth_tokens: user_id -> driver_id только если есть колонка user_id (старая БД)
	_, _ = pool.Exec(ctx, `ALTER TABLE auth_tokens ADD COLUMN IF NOT EXISTS driver_id UUID REFERENCES drivers(id) ON DELETE CASCADE`)
	_, _ = pool.Exec(ctx, `
		DO $$ BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='auth_tokens' AND column_name='user_id') THEN
				UPDATE auth_tokens SET driver_id = (SELECT id FROM drivers WHERE drivers.user_id = auth_tokens.user_id LIMIT 1) WHERE driver_id IS NULL AND user_id IS NOT NULL;
				DELETE FROM auth_tokens WHERE driver_id IS NULL;
				ALTER TABLE auth_tokens DROP COLUMN user_id;
				ALTER TABLE auth_tokens ALTER COLUMN driver_id SET NOT NULL;
			END IF;
		END $$
	`)

	_, _ = pool.Exec(ctx, `DROP TABLE IF EXISTS users CASCADE`)
	_, _ = pool.Exec(ctx, `ALTER TABLE drivers DROP COLUMN IF EXISTS user_id`)
	_, _ = pool.Exec(ctx, `ALTER TABLE deleted_drivers DROP COLUMN IF EXISTS user_id`)

	// Файлы только по path; ссылки для скачивания строятся в API (GET /drivers/:id/files/...)
	_, _ = pool.Exec(ctx, `ALTER TABLE drivers DROP COLUMN IF EXISTS car_photo_url`)
	_, _ = pool.Exec(ctx, `ALTER TABLE drivers DROP COLUMN IF EXISTS adr_document_url`)
	_, _ = pool.Exec(ctx, `ALTER TABLE deleted_drivers DROP COLUMN IF EXISTS car_photo_url`)
	_, _ = pool.Exec(ctx, `ALTER TABLE deleted_drivers DROP COLUMN IF EXISTS adr_document_url`)

	_, err := pool.Exec(ctx, `
		INSERT INTO schema_version (version, name) VALUES (9, 'users_drop_email')
		ON CONFLICT (version) DO NOTHING
	`)
	return err
}

// 9 — последняя активация водителя: время и координаты на карте
func UpDriversLastActivate(ctx context.Context, pool *pgxpool.Pool) error {
	for _, q := range []string{
		`ALTER TABLE drivers ADD COLUMN IF NOT EXISTS last_activated_at TIMESTAMPTZ`,
		`ALTER TABLE drivers ADD COLUMN IF NOT EXISTS last_activated_latitude DOUBLE PRECISION`,
		`ALTER TABLE drivers ADD COLUMN IF NOT EXISTS last_activated_longitude DOUBLE PRECISION`,
		`ALTER TABLE deleted_drivers ADD COLUMN IF NOT EXISTS last_activated_at TIMESTAMPTZ`,
		`ALTER TABLE deleted_drivers ADD COLUMN IF NOT EXISTS last_activated_latitude DOUBLE PRECISION`,
		`ALTER TABLE deleted_drivers ADD COLUMN IF NOT EXISTS last_activated_longitude DOUBLE PRECISION`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			return err
		}
	}
	_, err := pool.Exec(ctx, `
		INSERT INTO schema_version (version, name) VALUES (10, 'drivers_last_activate')
		ON CONFLICT (version) DO NOTHING
	`)
	return err
}
