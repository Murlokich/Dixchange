package entity

import (
	"github.com/pkg/errors"
)

type ValidationError struct {
	Err error
}

func (v ValidationError) Error() string {
	return v.Err.Error()
}

var (
	ErrNoUserFound = errors.New("No such user was found")
)
