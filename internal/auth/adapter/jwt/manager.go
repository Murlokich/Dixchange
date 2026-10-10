package jwt

import (
	"encoding/base64"
	"time"

	"crypto/rsa"

	"github.com/dixchange/Monolith/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	tokenValidityPeriod = 24 * 7
)

type Manager struct {
	issuer     string
	audience   string
	publicKey  rsa.PublicKey
	privateKey rsa.PrivateKey
}

func (m *Manager) IssueToken(userID uuid.UUID) (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", errors.Wrap(err, "uuid NewV7")
	}

	claims := jwt.RegisteredClaims{
		Issuer:    m.issuer,
		Subject:   userID.String(),
		Audience:  jwt.ClaimStrings{m.audience},
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenValidityPeriod * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ID:        id.String(),
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(&m.privateKey)
	if err != nil {
		return "", errors.Wrap(err, "jwt NewWithClaims SignedString")
	}
	return token, nil
}

func (m *Manager) ValidateToken(accessToken string) (*jwt.Token, error) {
	parser := jwt.NewParser()

	// return manager's public key to validate token
	getKey := func(*jwt.Token) (any, error) {
		return &m.publicKey, nil
	}
	token, err := parser.Parse(accessToken, getKey)
	if err != nil {
		return nil, errors.Wrap(err, "parser Parse")
	}
	return token, nil
}

func NewManager(c *config.JWTConfig) (*Manager, error) {
	rawPublicKey, err := base64.StdEncoding.DecodeString(c.PublicKeyB64)
	if err != nil {
		return nil, errors.Wrap(err, "base64 decode public key")
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(rawPublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "jwt ParseRSAPublicKeyFromPEM")
	}
	rawPrivateKey, err := base64.StdEncoding.DecodeString(c.PrivateKeyB64)
	if err != nil {
		return nil, errors.Wrap(err, "base64 decode private key")
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(rawPrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "jwt ParseRSAPrivateKeyFromPEM")
	}

	return &Manager{
		issuer:     c.Issuer,
		audience:   c.Audience,
		publicKey:  *publicKey,
		privateKey: *privateKey,
	}, nil
}
