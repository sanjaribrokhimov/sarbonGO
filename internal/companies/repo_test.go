// Интеграционные тесты репозитория компаний: создание компании с владельцем (CreateByOwner)
// и назначение владельца (SetOwner). Запуск с БД: TEST_DATABASE_URL или DATABASE_URL заданы.
package companies

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sarbonNew/internal/appusers"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		connStr = os.Getenv("DATABASE_URL")
	}
	if connStr == "" {
		t.Skip("TEST_DATABASE_URL or DATABASE_URL required for integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("pool.Ping: %v", err)
	}
	return pool
}

// TestCreateByOwner_AndGetByIDTZ проверяет, что создание компании с владельцем
// и получение по ID возвращают корректные name, type, owner_id.
func TestCreateByOwner_AndGetByIDTZ(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	repo := NewRepo(pool)
	usersRepo := appusers.NewRepo(pool)

	// Владелец должен существовать в company_users (FK)
	owner, err := usersRepo.Create(ctx, "+7999"+uuid.New().String()[:8], "hash", nil, nil, nil, "OWNER")
	if err != nil {
		t.Fatalf("create company user: %v", err)
	}
	ownerID := uuid.MustParse(owner.ID)
	name := "Test Company " + uuid.New().String()
	companyType := "Shipper" // в БД PascalCase
	inn := "1234567890"

	companyID, err := repo.CreateByOwner(ctx, CreateByOwnerParams{
		Name:    name,
		Type:    companyType,
		OwnerID: ownerID,
		Inn:     &inn,
	})
	if err != nil {
		t.Fatalf("CreateByOwner: %v", err)
	}
	if companyID == uuid.Nil {
		t.Fatal("CreateByOwner returned nil id")
	}

	comp, err := repo.GetByIDTZ(ctx, companyID)
	if err != nil {
		t.Fatalf("GetByIDTZ: %v", err)
	}
	if comp == nil {
		t.Fatal("GetByIDTZ returned nil")
	}
	if comp.Name != name {
		t.Errorf("name: got %q want %q", comp.Name, name)
	}
	if comp.Type == nil || *comp.Type != companyType {
		t.Errorf("type: got %v want %q", comp.Type, companyType)
	}
	if comp.OwnerID == nil || *comp.OwnerID != ownerID {
		t.Errorf("owner_id: got %v want %s", comp.OwnerID, ownerID)
	}
	if comp.Inn == nil || *comp.Inn != inn {
		t.Errorf("inn: got %v want %q", comp.Inn, inn)
	}
}

// TestSetOwner проверяет, что назначение владельца компании (PATCH /admin/companies/:id/owner)
// корректно устанавливает owner_id и статус active (проверяем через GetByIDTZ — owner_id).
func TestSetOwner(t *testing.T) {
	pool := testPool(t)
	defer pool.Close()
	ctx := context.Background()
	repo := NewRepo(pool)
	usersRepo := appusers.NewRepo(pool)

	// Владелец должен существовать в company_users (FK)
	owner, err := usersRepo.Create(ctx, "+7998"+uuid.New().String()[:8], "hash", nil, nil, nil, "OWNER")
	if err != nil {
		t.Fatalf("create company user: %v", err)
	}
	ownerID := uuid.MustParse(owner.ID)

	// Создаём компанию без владельца (прямой INSERT, чтобы не зависеть от таблицы admins)
	name := "Company Without Owner " + uuid.New().String()
	var companyID uuid.UUID
	const insertQ = `
INSERT INTO companies (name, status, created_at, updated_at, max_vehicles, max_drivers, max_cargo, max_dispatchers, max_managers, max_top_dispatchers, max_top_managers, completed_orders, cancelled_orders, total_revenue)
VALUES ($1, 'pending', now(), now(), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
RETURNING id`
	err = pool.QueryRow(ctx, insertQ, name).Scan(&companyID)
	if err != nil {
		t.Fatalf("insert company: %v", err)
	}

	compBefore, _ := repo.GetByIDTZ(ctx, companyID)
	if compBefore != nil && compBefore.OwnerID != nil {
		t.Fatalf("company should have no owner before SetOwner, got %v", compBefore.OwnerID)
	}

	err = repo.SetOwner(ctx, companyID, ownerID)
	if err != nil {
		t.Fatalf("SetOwner: %v", err)
	}

	comp, err := repo.GetByIDTZ(ctx, companyID)
	if err != nil {
		t.Fatalf("GetByIDTZ after SetOwner: %v", err)
	}
	if comp == nil {
		t.Fatal("GetByIDTZ returned nil")
	}
	if comp.OwnerID == nil || *comp.OwnerID != ownerID {
		t.Errorf("owner_id after SetOwner: got %v want %s", comp.OwnerID, ownerID)
	}
}
