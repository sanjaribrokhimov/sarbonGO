package infra

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"sarbonNew/internal/config"
)

type Infra struct {
	PG    *pgxpool.Pool
	Redis *redis.Client
}

func New(ctx context.Context, cfg config.Config, logger *zap.Logger) (*Infra, error) {
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	// Self-bootstrap schema: ensure `drivers` exists before serving requests.
	if err := EnsureDriversTable(ctx, pool); err != nil {
		pool.Close()
		return nil, err
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		pool.Close()
		return nil, err
	}

	logger.Info("infra ready")
	return &Infra{PG: pool, Redis: rdb}, nil
}

func (i *Infra) Close() {
	if i == nil {
		return
	}
	if i.PG != nil {
		i.PG.Close()
	}
	if i.Redis != nil {
		_ = i.Redis.Close()
	}
}

