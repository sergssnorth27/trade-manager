package config

import (
	"fmt"
	"os"
)

type Config struct {
	PGDSN         string
	MARKETCSGOKey string
	LogLevel      string
	DryRun        bool
}

func Load() (*Config, error) {
	pgDSN := os.Getenv("PG_DSN")
	if pgDSN == "" {
		return nil, fmt.Errorf("PG_DSN is required")
	}

	return &Config{
		PGDSN:         pgDSN,
		MARKETCSGOKey: os.Getenv("MARKETCSGO_KEY"),
		LogLevel:      os.Getenv("LOG_LEVEL"),
		DryRun:        os.Getenv("DRY_RUN") == "true",
	}, nil
}
