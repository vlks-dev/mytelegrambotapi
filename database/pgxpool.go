package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mytelegrambot/config"
	"go.uber.org/zap"
	"time"
)

func GetPool(ctx context.Context, config *config.Config, logger *zap.SugaredLogger) (*pgxpool.Pool, error) {
	logger = logger.Named("database")
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
	logger.Infof("db pool estabilished, time left: %v", time.Until(deadline))

	return pool, nil
}
