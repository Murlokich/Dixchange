package deposit

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

var (
	ErrEmptyUserUID          = errors.New("user uid cannot be empty")
	ErrUserUIDTypeConversion = errors.New("userUID type is no longer uuid.UUID")

	ErrUSDAmountNonPositive    = errors.New("not a positive deposit amount")
	ErrUSDAmountTypeConversion = errors.New("amount type is no longer decimal.Decimal")
)

type Input struct {
	UserID    uuid.UUID
	USDAmount decimal.Decimal
}

func (i *Input) Validate(ctx context.Context) error {
	const errWrapMessage = "register input validate"
	if err := validation.ValidateWithContext(
		ctx,
		i.UserID,
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
	); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}
	if err := validation.ValidateWithContext(
		ctx,
		i.USDAmount,
		validation.By(func(value interface{}) error {
			amount, ok := value.(decimal.Decimal)
			if !ok {
				return ErrUSDAmountTypeConversion
			}
			if !amount.IsPositive() {
				return ErrUSDAmountNonPositive
			}
			return nil
		}),
	); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}

	return nil
}
