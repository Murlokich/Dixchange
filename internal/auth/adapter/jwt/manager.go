package jwt

import (
	"encoding/base64"
	"time"

	"crypto/rsa"

	"github.com/dixchange/Monolith/internal/auth/config"
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
	const errWrapMessage = "jwt manager IssueToken"
	id, err := uuid.NewV7()
	if err != nil {
		return "", errors.Wrap(err, errWrapMessage)
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
		return "", errors.Wrap(err, errWrapMessage)
	}
	return token, nil
}

func (m *Manager) ValidateToken(accessToken string) (*jwt.Token, error) {
	const errWrapMessage = "jwt manager ValidateToken"

	parser := jwt.NewParser()

	// return manager's public key to validate token
	getKey := func(*jwt.Token) (any, error) {
		return &m.publicKey, nil
	}
	token, err := parser.Parse(accessToken, getKey)
	if err != nil {
		return nil, errors.Wrap(err, errWrapMessage)
	}
	return token, nil
}

func NewManager(c *config.JWTConfig) (*Manager, error) {
	rawPublicKey, err := base64.StdEncoding.DecodeString(c.PublicKeyB64)
	if err != nil {
		return nil, err
	}
	publicKey, err := jwt.ParseRSAPublicKeyFromPEM(rawPublicKey)
	if err != nil {
		return nil, err
	}
	rawPrivateKey, err := base64.StdEncoding.DecodeString(c.PrivateKeyB64)
	if err != nil {
		return nil, err
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(rawPrivateKey)
	if err != nil {
		return nil, err
	}

	return &Manager{
		issuer:     c.Issuer,
		audience:   c.Audience,
		publicKey:  *publicKey,
		privateKey: *privateKey,
	}, nil
}
