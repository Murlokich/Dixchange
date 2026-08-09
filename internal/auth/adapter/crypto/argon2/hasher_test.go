package argon2_test

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/dixchange/Monolith/internal/auth/adapter/crypto/argon2"
	"github.com/dixchange/Monolith/internal/auth/config"
	"github.com/stretchr/testify/require"
	cryptoargon2 "golang.org/x/crypto/argon2"
)

var testParams = &config.Argon2Config{ //nolint:gochecknoglobals // acceptable in test
	Memory:     65536,
	Iterations: 1,
	Threads:    1,
	HashLength: 16,
	SaltLength: 16,
}

// hashRX matches the encoded hash format GeneratePasswordHash produces for
// testParams, so it stays correct if the params above ever change.
var hashRX = regexp.MustCompile(fmt.Sprintf( //nolint:gochecknoglobals // acceptable in test
	`^\$argon2id\$v=%d\$m=%d,t=%d,p=%d\$[A-Za-z0-9+/]{%d}\$[A-Za-z0-9+/]{%d}$`,
	cryptoargon2.Version,
	testParams.Memory,
	testParams.Iterations,
	testParams.Threads,
	base64.RawStdEncoding.EncodedLen(int(testParams.SaltLength)),
	base64.RawStdEncoding.EncodedLen(int(testParams.HashLength)),
))

// Pre-computed hash for password "bug", carried over from the reference
// implementation this hasher was adapted from:
// https://github.com/alexedwards/argon2id/blob/master/argon2id_test.go
// ComparePasswords derives its working params from the encoded hash itself,
// so it is independent of testParams above and of p=2/32-byte-key here.
const bugHash = "$argon2id$v=19$m=65536,t=1,p=2$UDk0zEuIzbt0x3bwkf8Bgw$ihSfHWUJpTgDvNWiojrgcN4E0pJdUVmqCEdRZesx9tE"

var memoryParamRX = regexp.MustCompile(`m=\d+`) //nolint:gochecknoglobals // acceptable in test

func TestPasswordHasher_GeneratePasswordHash(t *testing.T) {
	t.Parallel()
	hasher := argon2.NewArgon2(testParams)

	hash1, err := hasher.GeneratePasswordHash("pa$$word")
	require.NoError(t, err)
	require.Regexp(t, hashRX, hash1)

	hash2, err := hasher.GeneratePasswordHash("pa$$word")
	require.NoError(t, err)
	require.Regexp(t, hashRX, hash2)

	require.NotEqual(t, hash1, hash2, "hashes must be unique due to random salt")
}

func TestPasswordHasher_ComparePasswords(t *testing.T) {
	t.Parallel()
	hasher := argon2.NewArgon2(testParams)

	hash, err := hasher.GeneratePasswordHash("pa$$word")
	require.NoError(t, err)

	match, err := hasher.ComparePasswords("pa$$word", hash)
	require.NoError(t, err)
	require.True(t, match)

	match, err = hasher.ComparePasswords("otherPa$$word", hash)
	require.NoError(t, err)
	require.False(t, match)
}

func TestPasswordHasher_ComparePasswords_KnownHash(t *testing.T) {
	t.Parallel()
	hasher := argon2.NewArgon2(testParams)

	match, err := hasher.ComparePasswords("bug", bugHash)
	require.NoError(t, err)
	require.True(t, match)
}

func TestPasswordHasher_ComparePasswords_StrictDecoding(t *testing.T) {
	t.Parallel()
	hasher := argon2.NewArgon2(testParams)

	// changed the last character of the hash's base64 key segment
	corrupted := bugHash[:len(bugHash)-1] + "F"
	_, err := hasher.ComparePasswords("bug", corrupted)
	require.Error(t, err)
}

func TestPasswordHasher_ComparePasswords_MalformedStructure(t *testing.T) {
	t.Parallel()
	hasher := argon2.NewArgon2(testParams)

	testCases := []struct {
		name string
		hash string
	}{
		{name: "missing segment", hash: strings.TrimSuffix(bugHash, "$ihSfHWUJpTgDvNWiojrgcN4E0pJdUVmqCEdRZesx9tE")},
		{name: "extra segment", hash: bugHash + "$extra"},
		{name: "non-numeric memory param", hash: memoryParamRX.ReplaceAllString(bugHash, "m=JUNK")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := hasher.ComparePasswords("bug", tc.hash)
			require.Error(t, err)
		})
	}
}

func TestPasswordHasher_ComparePasswords_IncompatibleVersion(t *testing.T) {
	t.Parallel()
	hasher := argon2.NewArgon2(testParams)

	corrupted := strings.Replace(bugHash, fmt.Sprintf("v=%d", cryptoargon2.Version), "v=1", 1)
	_, err := hasher.ComparePasswords("bug", corrupted)
	require.ErrorIs(t, err, argon2.ErrIncompatibleVersion)
}
