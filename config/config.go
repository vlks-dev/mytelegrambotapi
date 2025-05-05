package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Token              string
	ConnString         string
	MaxPgxConn         int32
	MaxPgxConnIdleTime time.Duration
	MaxPgxConnLifeTime time.Duration
	HealthCheckPeriod  time.Duration
	R1Token            string
	R1ProToken         string
	BotEnv             bool
	Logger             Logger
}

type Logger struct {
	Development      bool
	OutputPaths      []string
	ErrorOutputPaths []string
}

var (
	cfg            *Config
	botDebug       bool
	logDevelopment bool
)

func LoadEnvCfg(source string) (*Config, error) {
	err := godotenv.Load(source)
	if err != nil {
		return nil, fmt.Errorf("error getting enviroment file, in %v, err: %v", source, err)

	}

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

	if os.Getenv("BOT_ENV") == "debug" {
		botDebug = true
	} else {
		botDebug = false
	}

	if os.Getenv("LOG_DEVELOPMENT") == "true" {
		logDevelopment = true
	} else {
		logDevelopment = false
	}

	cfg = &Config{
		Token:              os.Getenv("TOKEN"),
		ConnString:         os.Getenv("CONNECTION_STRING"),
		MaxPgxConn:         int32(parsedPgxConns),
		MaxPgxConnIdleTime: time.Duration(parsedPgxConnIdleTime),
		MaxPgxConnLifeTime: time.Duration(parsedMaxPgxConnLifeTime),
		HealthCheckPeriod:  time.Duration(parsedHealthCheckPeriod),
		R1Token:            os.Getenv("R1_TOKEN"),
		R1ProToken:         os.Getenv("R1_PRO_TOKEN"),
		BotEnv:             botDebug,
		Logger: Logger{
			Development:      logDevelopment,
			OutputPaths:      strings.Split(os.Getenv("LOG_OUTPUT_PATHS"), ","),
			ErrorOutputPaths: strings.Split(os.Getenv("LOG_ERROR_OUTPUT_PATHS"), ","),
		},
	}
	return cfg, nil
}
