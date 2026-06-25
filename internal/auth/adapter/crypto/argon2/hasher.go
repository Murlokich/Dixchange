package argon2

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/dixchange/Monolith/internal/auth/config"
	"github.com/pkg/errors"
	"golang.org/x/crypto/argon2"
)

type PasswordHasher struct {
	memory     uint32
	iter       uint32
	threads    uint8
	hashLength uint32
}

func (p *PasswordHasher) GeneratePasswordHash(password string) (string, error) {
	const errWrapMessage = "argon2 PasswordHasher GeneratePasswordHash"

	salt, err := p.genSalt()
	if err != nil {
		return "", errors.Wrap(err, errWrapMessage)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		p.iter,
		p.memory,
		p.threads,
		p.hashLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// remember parameters and the algorithm to support potential migration
	// https://www.alexedwards.net/blog/how-to-hash-and-verify-passwords-with-argon2-in-go#storingpasswords
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.memory,
		p.iter,
		p.threads,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

func (p *PasswordHasher) genSalt() ([]byte, error) {
	const errWrapMessage = "argon2 PasswordHasher genSalt"
	const saltLength = 16
	salt := make([]byte, saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, errors.Wrap(err, errWrapMessage)
	}
	return salt, nil
}

// TODO: check non-negative values
func NewArgon2(c *config.Argon2Config) *PasswordHasher {
	return &PasswordHasher{
		memory:     c.Memory,
		iter:       c.Iterations,
		threads:    c.Threads,
		hashLength: c.HashLength,
	}
}
