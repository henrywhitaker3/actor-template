// Package config
package config

import (
	"fmt"

	"github.com/henrywhitaker3/windowframe/v2/config"
	"github.com/joho/godotenv"
)

type Runtime struct {
	Hostname                 string                  `env:"HOST"`
	PSK                      config.RedactedString   `env:"PSK"`
	ManagementTokens         []config.RedactedString `env:"MANAGEMENT_TOKENS"`
	ManagementAllowedOrigins []string                `env:"MANAGEMENT_ALLOWED_ORIGINS"`
}

type Database struct {
	Driver string                `env:"DRIVER, default=sqlite"`
	URL    config.RedactedString `env:"URL, default=db.sqlite"`
}

type App struct {
	Name string `env:"NAME"`
	URL  string `env:"URL"`
}

type Config struct {
	App      App      `env:",prefix=APP_"`
	Runtime  Runtime  `env:",prefix=RUNTIME_"`
	Database Database `env:",prefix=DB_"`
	LogLevel string   `env:"LOG_LEVEL, default=info"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()
	conf, err := config.NewParser[Config]().
		WithExtractors(
			config.NewEnvExtractor[Config](),
		).Parse()
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &conf, nil
}
