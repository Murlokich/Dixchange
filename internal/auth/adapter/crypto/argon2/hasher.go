package argon2

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/dixchange/Monolith/internal/config"
	"github.com/pkg/errors"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash         = errors.New("the encoded hash is not in the correct format")
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
)

type Params struct {
	memory     uint32
	iter       uint32
	threads    uint8
	hashLength uint32
	saltLength uint32
}

type PasswordHasher struct {
	params Params
}

func (p *PasswordHasher) GeneratePasswordHash(password string) (string, error) {
	salt, err := p.genSalt()
	if err != nil {
		return "", errors.Wrap(err, "password hasher gen salt")
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		p.params.iter,
		p.params.memory,
		p.params.threads,
		p.params.hashLength,
	)

	b64Salt := base64.RawStdEncoding.Strict().EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.Strict().EncodeToString(hash)

	// remember parameters and the algorithm to support potential migration
	// https://www.alexedwards.net/blog/how-to-hash-and-verify-passwords-with-argon2-in-go#storingpasswords
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.params.memory,
		p.params.iter,
		p.params.threads,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

func (p *PasswordHasher) ComparePasswords(password string, passwordHash string) (bool, error) {
	// Extract the parameters, salt and derived key from the encoded password
	// hash.
	hashParams, salt, hash, err := decodeHash(passwordHash)
	if err != nil {
		return false, errors.Wrap(err, "decode password hash")
	}

	// Derive the key from the other password using the same parameters.
	otherHash := argon2.IDKey(
		[]byte(password),
		salt,
		hashParams.iter,
		hashParams.memory,
		hashParams.threads,
		hashParams.hashLength,
	)

	// Check that the contents of the hashed passwords are identical. Note
	// that we are using the subtle.ConstantTimeCompare() function for this
	// to help prevent timing attacks.
	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}
	return false, nil
}

func decodeHash(encodedHash string) (*Params, []byte, []byte, error) {
	vals := strings.Split(encodedHash, "$")
	const passwordHashElementsNumber = 6
	if len(vals) != passwordHashElementsNumber {
		return nil, nil, nil, errors.Wrap(ErrInvalidHash, "password hash invalid structure")
	}

	var version int
	_, err := fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "scan password hash algo version")
	}
	if version != argon2.Version {
		return nil, nil, nil, errors.Wrap(ErrIncompatibleVersion, "password hash unsupported version")
	}

	p := &Params{}
	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.memory, &p.iter, &p.threads)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "scan password hash algo settings")
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "scan password hash salt")
	}
	p.saltLength = uint32(len(salt)) // #nosec G115 - uint32 type required by argon2

	hash, err := base64.RawStdEncoding.Strict().DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "scan password hash len")
	}
	p.hashLength = uint32(len(hash)) // #nosec G115 - uint32 type required by argon2

	return p, salt, hash, nil
}

func (p *PasswordHasher) genSalt() ([]byte, error) {
	salt := make([]byte, p.params.saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, errors.Wrap(err, "gen salt")
	}
	return salt, nil
}

// TODO: check non-negative values
func NewArgon2(c *config.Argon2Config) *PasswordHasher {
	return &PasswordHasher{
		params: Params{
			memory:     c.Memory,
			iter:       c.Iterations,
			threads:    c.Threads,
			hashLength: c.HashLength,
			saltLength: c.SaltLength,
		},
	}
}
