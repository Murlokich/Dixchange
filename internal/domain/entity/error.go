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
	ErrNoUserFound         = errors.New("No such user was found")
	ErrInsufficientBalance = errors.New("User has insufficient balance for the corresponding action")
	ErrBalanceNotFound     = errors.New("No balance record was found for the user and symbol")
	ErrNotTradeParticipant = errors.New("User is not a participant of the trade")
)
