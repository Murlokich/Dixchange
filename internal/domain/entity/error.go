package entity

import (
	"fmt"

	"github.com/pkg/errors"
)

type ValidationError struct {
	Err error
}

func (v ValidationError) Error() string {
	return v.Err.Error()
}

type InvalidCommandTypeError struct {
	CommandType string
}

func (e InvalidCommandTypeError) Error() string {
	return fmt.Sprintf("command has invalid type: %s", e.CommandType)
}

var (
	ErrNoUserFound          = errors.New("No such user was found")
	ErrInsufficientBalance  = errors.New("User has insufficient balance for the corresponding action")
	ErrBalanceNotFound      = errors.New("No balance record was found for the user and symbol")
	ErrNotTradeParticipant  = errors.New("User is not a participant of the trade")
	ErrOrderNotFound        = errors.New("No such order was found")
	ErrOrderNotOwned        = errors.New("Order doesn't belong to the user")
	ErrOrderNotCancellable  = errors.New("Order is no longer active and can't be cancelled")
	ErrCommandBusOverloaded = errors.New("Matching engine is overloaded, try again later")
)
