package config

import (
	"errors"
	"os"
)

type Config struct {
	DatabaseURL string
	HTTPAddr    string
}

func Load() (Config, error) {
	dbURL := os.Getenv("DATABASE_URL")

	if dbURL == "" {
		return Config{}, errors.New("DATABASE_URL environment variable not set")
	}

	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	return Config{HTTPAddr: httpAddr, DatabaseURL: dbURL}, nil
}
