package config_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/config"
	"github.com/stretchr/testify/require"
)

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

			cfg := config.Config{TradeHistory: config.TradeHistory{PageSize: tc.pageSize}}
			err := cfg.Validate()

			if tc.wantErr {
				require.ErrorContains(t, err, "trade history: page size must be positive")
				return
			}
			require.NoError(t, err)
		})
	}
}
