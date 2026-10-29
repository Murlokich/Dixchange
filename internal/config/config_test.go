package config_test

import (
	"testing"

	"github.com/Murlokich/Dixchange/internal/config"
	"github.com/kelseyhightower/envconfig"
	"github.com/stretchr/testify/require"
)

func validConfig() config.Config {
	return config.Config{
		HTTPConfig: config.HTTPConfig{ReadHeaderTimeoutSeconds: 10},
		Argon2Config: config.Argon2Config{
			Memory:     64 * 1024,
			Iterations: 1,
			Threads:    1,
			HashLength: 16,
			SaltLength: 16,
		},
		CommandBusConfig: config.CommandBusConfig{BufferSize: 100},
		OrderBookConfig:  config.OrderBook{B: 8},
		EventBusConfig:   config.EventBusConfig{BufferSize: 100},
		TradeHistory:     config.TradeHistory{PageSize: 20},
	}
}

func TestConfig_Validate_Valid(t *testing.T) {
	t.Parallel()

	cfg := validConfig()
	require.NoError(t, cfg.Validate())
}

func TestConfig_Validate_TradeHistoryPageSize(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		pageSize int
		wantErr  bool
	}{
		{name: "positive", pageSize: 20},
		{name: "one", pageSize: 1},
		{name: "zero", pageSize: 0, wantErr: true},
		{name: "negative", pageSize: -5, wantErr: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			cfg.TradeHistory.PageSize = tc.pageSize
			err := cfg.Validate()

			if tc.wantErr {
				require.ErrorContains(t, err, "trade history: page size must be positive")
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfig_Validate_Bounds(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		set     func(c *config.Config)
		wantErr string
	}{
		{
			name: "http timeout one",
			set:  func(c *config.Config) { c.HTTPConfig.ReadHeaderTimeoutSeconds = 1 },
		},
		{
			name:    "http timeout zero",
			set:     func(c *config.Config) { c.HTTPConfig.ReadHeaderTimeoutSeconds = 0 },
			wantErr: "http: read header timeout",
		},
		{
			name:    "http timeout negative",
			set:     func(c *config.Config) { c.HTTPConfig.ReadHeaderTimeoutSeconds = -1 },
			wantErr: "http: read header timeout",
		},

		{
			name: "order book B two",
			set:  func(c *config.Config) { c.OrderBookConfig.B = 2 },
		},
		{
			name:    "order book B one",
			set:     func(c *config.Config) { c.OrderBookConfig.B = 1 },
			wantErr: "order book: B must be at least 2",
		},
		{
			name:    "order book B zero",
			set:     func(c *config.Config) { c.OrderBookConfig.B = 0 },
			wantErr: "order book: B must be at least 2",
		},
		{
			name:    "order book B negative",
			set:     func(c *config.Config) { c.OrderBookConfig.B = -1 },
			wantErr: "order book: B must be at least 2",
		},

		{
			name: "command bus buffer one",
			set:  func(c *config.Config) { c.CommandBusConfig.BufferSize = 1 },
		},
		{
			name:    "command bus buffer zero",
			set:     func(c *config.Config) { c.CommandBusConfig.BufferSize = 0 },
			wantErr: "command bus: buffer size",
		},
		{
			name:    "command bus buffer negative",
			set:     func(c *config.Config) { c.CommandBusConfig.BufferSize = -1 },
			wantErr: "command bus: buffer size",
		},

		{
			name: "event bus buffer zero",
			set:  func(c *config.Config) { c.EventBusConfig.BufferSize = 0 },
		},
		{
			name:    "event bus buffer negative",
			set:     func(c *config.Config) { c.EventBusConfig.BufferSize = -1 },
			wantErr: "event bus: buffer size",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tc.set(&cfg)
			err := cfg.Validate()

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfig_Validate_Argon2Zero(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		zero    func(c *config.Argon2Config)
		wantErr string
	}{
		{name: "memory", zero: func(c *config.Argon2Config) { c.Memory = 0 }, wantErr: "argon2: memory"},
		{name: "iterations", zero: func(c *config.Argon2Config) { c.Iterations = 0 }, wantErr: "argon2: iterations"},
		{name: "threads", zero: func(c *config.Argon2Config) { c.Threads = 0 }, wantErr: "argon2: threads"},
		{name: "hash length", zero: func(c *config.Argon2Config) { c.HashLength = 0 }, wantErr: "argon2: hash length"},
		{name: "salt length", zero: func(c *config.Argon2Config) { c.SaltLength = 0 }, wantErr: "argon2: salt length"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := validConfig()
			tc.zero(&cfg.Argon2Config)

			require.ErrorContains(t, cfg.Validate(), tc.wantErr)
		})
	}
}

// Negative values never reach Validate: the fields are unsigned, so envconfig fails to parse them.
func TestArgon2Config_NegativeRejectedByEnvconfig(t *testing.T) {
	t.Setenv("ARGON2_ITERATIONS", "-1")

	var cfg config.Argon2Config
	require.Error(t, envconfig.Process("ARGON2", &cfg))
}
