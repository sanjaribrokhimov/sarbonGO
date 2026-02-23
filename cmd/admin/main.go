package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"sarbonNew/internal/config"
	"sarbonNew/internal/infra"
	"sarbonNew/internal/util"
)

func main() {
	config.LoadDotEnvUp(8)

	var (
		login  = flag.String("login", "", "admin login (unique)")
		pw     = flag.String("password", "", "admin password (plain text)")
		name   = flag.String("name", "", "admin name")
		status = flag.String("status", "active", "active|inactive|blocked")
		typ    = flag.String("type", "creator", "admin type")
	)
	flag.Parse()

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(2)
	}

	l := strings.TrimSpace(*login)
	p := strings.TrimSpace(*pw)
	n := strings.TrimSpace(*name)
	s := strings.TrimSpace(*status)
	t := strings.TrimSpace(*typ)
	if l == "" || p == "" || n == "" {
		fmt.Fprintln(os.Stderr, "flags -login, -password, -name are required")
		os.Exit(2)
	}
	if err := util.ValidatePassword(p); err != nil {
		fmt.Fprintln(os.Stderr, "invalid -password:", err)
		os.Exit(2)
	}

	hash, err := util.HashPassword(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, "password hash failed:", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "db connect failed:", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := infra.EnsureAdminsTable(ctx, pool); err != nil {
		fmt.Fprintln(os.Stderr, "ensure admins table failed:", err)
		os.Exit(1)
	}

	const q = `
INSERT INTO admins (login, password, name, status, type)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (login) DO UPDATE
SET password = EXCLUDED.password,
    name = EXCLUDED.name,
    status = EXCLUDED.status,
    type = EXCLUDED.type
RETURNING id`
	var id uuid.UUID
	if err := pool.QueryRow(ctx, q, l, hash, n, s, t).Scan(&id); err != nil {
		fmt.Fprintln(os.Stderr, "admin upsert failed:", err)
		os.Exit(1)
	}

	fmt.Println("admin id:", id.String())
}

