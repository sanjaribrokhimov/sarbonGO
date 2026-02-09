package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadDotEnvUp searches for ".env" in current dir and parents,
// then loads the first match. Safe to call in production (no-op if not found).
func LoadDotEnvUp(maxDepth int) {
	if maxDepth <= 0 {
		maxDepth = 6
	}

	dir, err := os.Getwd()
	if err != nil {
		_ = godotenv.Load()
		return
	}

	for i := 0; i <= maxDepth; i++ {
		p := filepath.Join(dir, ".env")
		if _, err := os.Stat(p); err == nil {
			_ = godotenv.Load(p)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// fallback: current directory
	_ = godotenv.Load()
}

