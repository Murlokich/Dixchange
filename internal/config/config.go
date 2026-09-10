package config

import "time"

type Config struct {
	HTTPConfig        HTTPConfig        `envconfig:"HTTP"`
	DBConfig          DBConfig          `envconfig:"DB"`
	Argon2Config      Argon2Config      `envconfig:"ARGON2"`
	JWTConfig         JWTConfig         `envconfig:"JWT"`
	BookWriterConfig  BookWriterConfig  `envconfig:"BOOK_WRITER"`
	OrderBookConfig   OrderBook         `envconfig:"ORDER_BOOK"`
	TradeWriterConfig TradeWriterConfig `envconfig:"TRADE_WRITER"`
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

type BookWriterConfig struct {
	BufferSize   int           `default:"100" split_words:"true"`
	WriteTimeout time.Duration `default:"100ms" split_words:"true"`
}

type TradeWriterConfig struct {
	BufferSize int `default:"100" split_words:"true"`
}
