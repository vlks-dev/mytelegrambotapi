package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Token              string
	ConnString         string
	MaxPgxConn         int32
	MaxPgxConnIdleTime time.Duration
	MaxPgxConnLifeTime time.Duration
	HealthCheckPeriod  time.Duration
}

func LoadEnvCfg(source string) (*Config, error) {
	err := godotenv.Load(source)
	if err != nil {
		return nil, fmt.Errorf("error getting enviroment file, in %v, err: %v", source, err)

	}
	var cfg *Config
	parsedPgxConns, err := strconv.ParseInt(os.Getenv("MAX_PGX_CONN"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf(
			"error parsing MAX_PGX_CONNS string: %v, err: %v",
			os.Getenv("MAX_PGX_CONN"),
			err,
		)
	}
	parsedPgxConnIdleTime, err := strconv.ParseInt(os.Getenv("MAX_PGX_CONN_IDLE_TIME"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf(
			"error parsing MAX_PGX_CONN_IDLE_TIME string: %v, err: %v",
			os.Getenv("MAX_PGX_CONN_IDLE_TIME"),
			err,
		)
	}
	parsedMaxPgxConnLifeTime, err := strconv.ParseInt(os.Getenv("MAX_PGX_CONN_LIFETIME"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf(
			"error parsing MAX_PGX_CONN_LIFETIME string: %v, err: %v",
			os.Getenv("MAX_PGX_CONN_LIFETIME"),
			err,
		)
	}
	parsedHealthCheckPeriod, err := strconv.ParseInt(os.Getenv("HEALTH_CHECK_PERIOD"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf(
			"error parsing HEALTH_CHECK_PERIOD string: %v, err: %v",
			os.Getenv("HEALTH_CHECK_PERIOD"),
			err,
		)
	}
	cfg = &Config{
		Token:              os.Getenv("TOKEN"),
		ConnString:         os.Getenv("CONNECTION_STRING"),
		MaxPgxConn:         int32(parsedPgxConns),
		MaxPgxConnIdleTime: time.Duration(parsedPgxConnIdleTime),
		MaxPgxConnLifeTime: time.Duration(parsedMaxPgxConnLifeTime),
		HealthCheckPeriod:  time.Duration(parsedHealthCheckPeriod),
	}
	return cfg, nil
}
