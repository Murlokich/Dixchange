package config

import (
	"github.com/dixchange/Monolith/internal/auth/config"
)

type Config struct {
	HTTPConfig HTTPConfig
	DBConfig   DBConfig

	AuthConfig config.AuthConfig
}

type HTTPConfig struct {
	Addr string `envconfig:"HTTP_ADDR" required:"true"`
}

type DBConfig struct {
	URL              string `envconfig:"DB_URL" required:"true"`
	MigrationVersion uint   `envconfig:"DB_MIGRATION_VERSION" required:"true"`
}
