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
cwd := dir
dir = filepath.Dir(dir)   // начать с уровня выше

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

// не нашли выше — проверить саму директорию (cwd)
p := filepath.Join(cwd, ".env")
if _, err := os.Stat(p); err == nil {
    _ = godotenv.Load(p)
    return
}
_ = godotenv.Load()
}

