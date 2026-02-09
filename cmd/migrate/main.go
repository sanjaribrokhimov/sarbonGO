package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"sarbonNew/internal/config"
)

func main() {
	config.LoadDotEnvUp(8)

	var (
		direction = flag.String("direction", "up", "up|down")
		steps     = flag.Int("steps", 0, "number of steps (0 = all)")
	)
	flag.Parse()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is required")
		os.Exit(2)
	}
	// golang-migrate pgx/v5 driver registers as "pgx5".
	if strings.HasPrefix(dbURL, "postgres://") {
		dbURL = "pgx5://" + strings.TrimPrefix(dbURL, "postgres://")
	}
	if strings.HasPrefix(dbURL, "pgx://") {
		dbURL = "pgx5://" + strings.TrimPrefix(dbURL, "pgx://")
	}

	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migrate init error:", err)
		os.Exit(1)
	}

	switch *direction {
	case "up":
		if *steps > 0 {
			err = m.Steps(*steps)
		} else {
			err = m.Up()
		}
	case "down":
		if *steps > 0 {
			err = m.Steps(-*steps)
		} else {
			err = m.Down()
		}
	default:
		fmt.Fprintln(os.Stderr, "invalid -direction, must be up|down")
		os.Exit(2)
	}

	if err != nil && err != migrate.ErrNoChange {
		fmt.Fprintln(os.Stderr, "migration error:", err)
		os.Exit(1)
	}

	fmt.Println("migrations:", *direction, "ok")
}

