// Миграции в Go; порядок задаётся списком. Все Up-функции в up.go.
// schema_version создаётся в первой миграции.
package migrations

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Runner запускает миграции по порядку.
type Runner struct {
	pool *pgxpool.Pool
}

// NewRunner создаёт раннер миграций для данного пула.
func NewRunner(pool *pgxpool.Pool) *Runner {
	return &Runner{pool: pool}
}

// Up выполняет все миграции по порядку.
func (r *Runner) Up(ctx context.Context) error {
	for i, m := range migrations {
		if err := m.Up(ctx, r.pool); err != nil {
			return fmt.Errorf("migration %d (%s): %w", i, m.Name, err)
		}
	}
	return nil
}

type migration struct {
	Name string
	Up   func(ctx context.Context, pool *pgxpool.Pool) error
}

// Список миграций: порядок важен.
var migrations = []migration{
	{Name: "create_user_categories", Up: UpUserCategories},
	{Name: "create_drivers_table", Up: UpDrivers},
	{Name: "drivers_freelance_dispatcher_id", Up: UpDriversFreelanceDispatcherID},
	{Name: "drivers_file_paths", Up: UpDriversFilePaths},
	{Name: "auth_schema", Up: UpAuthSchema},
	{Name: "drivers_user_id", Up: UpDriversUserID},
	{Name: "deleted_drivers", Up: UpDeletedDrivers},
	{Name: "users_drop_email", Up: UpUsersDropEmail},
	{Name: "drivers_last_activate", Up: UpDriversLastActivate},
}
