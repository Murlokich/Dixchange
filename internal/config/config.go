package config

import (
	"time"

	"github.com/pkg/errors"
)

type Config struct {
	HTTPConfig       HTTPConfig       `envconfig:"HTTP"`
	DBConfig         DBConfig         `envconfig:"DB"`
	Argon2Config     Argon2Config     `envconfig:"ARGON2"`
	JWTConfig        JWTConfig        `envconfig:"JWT"`
	CommandBusConfig CommandBusConfig `envconfig:"COMMAND_BUS"`
	OrderBookConfig  OrderBook        `envconfig:"ORDER_BOOK"`
	EventBusConfig   EventBusConfig   `envconfig:"EVENT_BUS"`
	TradeHistory     TradeHistory     `envconfig:"TRADE_HISTORY"`
}

// Validate checks the values envconfig can't: those that parse fine but make no sense.
func (c *Config) Validate() error {
	if err := c.TradeHistory.Validate(); err != nil {
		return errors.Wrap(err, "trade history")
	}
	return nil
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

type Argon2Config struct {
	Memory     uint32 `default:"65536"` // 64 * 1024 - default suggestion for argon2
	Iterations uint32 `default:"1"`
	Threads    uint8  `default:"1"`
	HashLength uint32 `default:"16" split_words:"true"`
	SaltLength uint32 `default:"16" split_words:"true"`
}

type JWTConfig struct {
	Issuer        string `default:"monolith-auth"`
	Audience      string `default:"monolith-api"`
	PublicKeyB64  string `split_words:"true"`
	PrivateKeyB64 string `split_words:"true"`
}

type OrderBook struct {
	B int `default:"8"`
}

type CommandBusConfig struct {
	BufferSize  int           `default:"100" split_words:"true"`
	SendTimeout time.Duration `default:"100ms" split_words:"true"`
}

type EventBusConfig struct {
	BufferSize int `default:"100" split_words:"true"`
}

type TradeHistory struct {
	// PageSize is how many trades a single trade history page holds. It is fixed on the
	// server so a client can't request arbitrarily large pages.
	PageSize int `default:"20" split_words:"true"`
}

func (t *TradeHistory) Validate() error {
	if t.PageSize < 1 {
		return errors.Errorf("page size must be positive, got %d", t.PageSize)
	}
	return nil
}
