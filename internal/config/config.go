package config

import (
	"github.com/dixchange/Monolith/internal/auth/config"
)

type Config struct {
	HTTPConfig HTTPConfig `envconfig:"HTTP"`
	DBConfig   DBConfig   `envconfig:"DB"`

	AuthConfig config.AuthConfig `envconfig:"AUTH"`
}

type HTTPConfig struct {
	Addr                     string `required:"true"`
	ReadHeaderTimeoutSeconds int64  `default:"10" split_words:"true"`
	FrontendHost             string `default:"http://localhost:5173" split_words:"true"`
}

type DBConfig struct {
	URL              string `required:"true"`
	MigrationVersion uint   `required:"true" split_words:"true"`
}
