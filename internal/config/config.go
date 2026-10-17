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
	sections := []struct {
		name string
		cfg  interface{ Validate() error }
	}{
		{"http", &c.HTTPConfig},
		{"argon2", &c.Argon2Config},
		{"command bus", &c.CommandBusConfig},
		{"order book", &c.OrderBookConfig},
		{"event bus", &c.EventBusConfig},
		{"trade history", &c.TradeHistory},
	}
	for _, s := range sections {
		if err := s.cfg.Validate(); err != nil {
			return errors.Wrap(err, s.name)
		}
	}
	return nil
}

type HTTPConfig struct {
	Addr                     string `required:"true"`
	ReadHeaderTimeoutSeconds int64  `default:"10" split_words:"true"`
	FrontendHost             string `default:"http://localhost:5173" split_words:"true"`
}

func (h *HTTPConfig) Validate() error {
	if h.ReadHeaderTimeoutSeconds < 1 {
		return errors.Errorf("read header timeout seconds must be positive, got %d", h.ReadHeaderTimeoutSeconds)
	}
	return nil
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

// Validate only checks for zeros: the fields are unsigned, so envconfig already rejects negatives.
func (a *Argon2Config) Validate() error {
	fields := []struct {
		name  string
		value uint32
	}{
		{"memory", a.Memory},
		{"iterations", a.Iterations},
		{"threads", uint32(a.Threads)},
		{"hash length", a.HashLength},
		{"salt length", a.SaltLength},
	}
	for _, f := range fields {
		if f.value == 0 {
			return errors.Errorf("%s must be positive", f.name)
		}
	}
	return nil
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

func (o *OrderBook) Validate() error {
	const minBTreeDegree = 2

	if o.B < minBTreeDegree {
		return errors.Errorf("B must be at least %d, got %d", minBTreeDegree, o.B)
	}
	return nil
}

type CommandBusConfig struct {
	BufferSize  int           `default:"100" split_words:"true"`
	SendTimeout time.Duration `default:"100ms" split_words:"true"`
}

// Validate requires a buffer: with no capacity the slot semaphore can never be acquired, so
// every request would time out as overloaded.
func (c *CommandBusConfig) Validate() error {
	if c.BufferSize < 1 {
		return errors.Errorf("buffer size must be positive, got %d", c.BufferSize)
	}
	return nil
}

type EventBusConfig struct {
	BufferSize int `default:"100" split_words:"true"`
}

// Validate allows 0: the bus then hands each event over synchronously.
func (e *EventBusConfig) Validate() error {
	if e.BufferSize < 0 {
		return errors.Errorf("buffer size must not be negative, got %d", e.BufferSize)
	}
	return nil
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
