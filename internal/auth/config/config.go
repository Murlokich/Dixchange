package config

type AuthConfig struct {
	Argon2Config Argon2Config `envconfig:"ARGON2"`
}

type Argon2Config struct {
	Memory     uint32 `default:"65536"` // 64 * 1024 - default suggestion for argon2
	Iterations uint32 `default:"1"`
	Threads    uint8  `default:"1"`
	HashLength uint32 `default:"16" split_words:"true"`
}
