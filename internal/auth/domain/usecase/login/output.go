package login

import (
	"github.com/pkg/errors"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Output struct {
	AccessToken string
}
