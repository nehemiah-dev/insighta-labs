package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL	string
	Port       	string
}

// Load reads configuration from environment variables.
// It fails loudly if anything required is missing — better to crash at
// startup than to limp along with a zero-value config.
func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		DatabaseURL: dbURL,
		Port:        port,
	}, nil
}