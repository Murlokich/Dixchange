package deposit

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type Input struct {
	UserID   uuid.UUID
	SymbolID int
	Amount   decimal.Decimal
}

func (i *Input) Validate(ctx context.Context) error {
	const errWrapMessage = "register input validate"

	if err := utils.ValidateUserID(ctx, i.UserID); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}

	if err := validation.ValidateWithContext(ctx, i.SymbolID, validation.Required, validation.Min(0)); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}

	if err := utils.ValidatePositiveDecimal(ctx, i.Amount); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}

	return nil
}
