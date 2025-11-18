package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vlks-dev/mytelegrambotapi/config"
	"go.uber.org/zap"
)

// GetRedisClient создает и возвращает Redis клиент
func GetRedisClient(ctx context.Context, cfg *config.Config, logger *zap.SugaredLogger) (*redis.Client, error) {
	logger = logger.Named("redis")

	// Если адрес Redis не указан, возвращаем nil (опциональное использование)
	if cfg.RedisAddr == "" {
		logger.Warn("Redis address not configured, Redis features will be disabled")
		return nil, nil
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Проверяем подключение
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := rdb.Ping(ctx).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Infow("Redis client connected",
		"addr", cfg.RedisAddr,
		"db", cfg.RedisDB,
	)

	return rdb, nil
}
