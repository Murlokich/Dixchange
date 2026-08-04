package utils

import (
	"context"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

const (
	passwordMinLength = 8
	passwordMaxLength = 256
)

var (
	ErrEmptyUserUID          = errors.New("user uid cannot be empty")
	ErrUserUIDTypeConversion = errors.New("userUID type is no longer uuid.UUID")

	ErrDecimalNonPositive    = errors.New("not a positive decimal amount")
	ErrDecimalTypeConversion = errors.New("amount type is no longer decimal.Decimal")
)

func ValidateUserID(ctx context.Context, userID uuid.UUID) error {
	return validation.ValidateWithContext(
		ctx,
		userID,
		validation.By(func(value interface{}) error {
			userUID, ok := value.(uuid.UUID)
			if !ok {
				return ErrUserUIDTypeConversion
			}
			if userUID == uuid.Nil {
				return ErrEmptyUserUID
			}
			return nil
		}),
	)
}

func ValidateEmail(ctx context.Context, email string) error {
	return validation.ValidateWithContext(ctx, email, validation.Required, is.EmailFormat)
}

func ValidatePassword(ctx context.Context, password string) error {
	return validation.ValidateWithContext(
		ctx,
		password,
		validation.Required,
		validation.Length(passwordMinLength, passwordMaxLength),
		validation.Match(regexp.MustCompile(".*[A-Z].*")),                                 // Contains upper case letter
		validation.Match(regexp.MustCompile(".*[a-z].*")),                                 // Contains lower case letter
		validation.Match(regexp.MustCompile(".*[0-9].*")),                                 // Contains digit
		validation.Match(regexp.MustCompile(".*[!@#$%^&*()_+\\-=\\[\\]{};:'\",.<>?/].*")), // contains special character
	)
}

func ValidatePositiveDecimal(ctx context.Context, dec decimal.Decimal) error {
	return validation.ValidateWithContext(
		ctx,
		dec,
		validation.By(func(value interface{}) error {
			amount, ok := value.(decimal.Decimal)
			if !ok {
				return ErrDecimalTypeConversion
			}
			if !amount.IsPositive() {
				return ErrDecimalNonPositive
			}
			return nil
		}),
	)
}
