// Клиент Redis: пул, таймауты, Ping при старте; используется для rate limit, кэша, сессий.
package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/sarbonGO/backend/internal/config"
)

// New создаёт клиент Redis с пулом и таймаутами (Dial, Read, Write); при старте выполняется Ping.
func New(cfg config.Redis) (*redis.Client, error) {
	cli := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()
	if err := cli.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return cli, nil
}

// Close закрывает клиент Redis (вызывать при graceful shutdown).
func Close(cli *redis.Client) {
	_ = cli.Close()
}
