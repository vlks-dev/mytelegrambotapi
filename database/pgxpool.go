package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mytelegrambot/config"
	"log"
	"time"
)

func GetPool(ctx context.Context, config *config.Config) (*pgxpool.Pool, error) {
	ctx, cancelFunc := context.WithTimeout(ctx, 7*time.Second)
	defer cancelFunc()

	parseConfig, err := pgxpool.ParseConfig(config.ConnString)
	if err != nil {
		return nil, fmt.Errorf("error parsing db config: %v", err)
	}
	parseConfig.MaxConns = config.MaxPgxConn
	parseConfig.MaxConnIdleTime = config.MaxPgxConnIdleTime
	parseConfig.MaxConnLifetime = config.MaxPgxConnLifeTime
	parseConfig.HealthCheckPeriod = config.HealthCheckPeriod

	pool, err := pgxpool.NewWithConfig(ctx, parseConfig)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect db\n[ERROR]: %v", err)
	}

	deadline, _ := ctx.Deadline()
	log.Printf("db pool estabilished, time left: %v\n", time.Until(deadline))

	return pool, nil
}
