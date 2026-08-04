package deposit

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type Input struct {
	UserID uuid.UUID
	Amount decimal.Decimal
}

func (i *Input) Validate(ctx context.Context) error {
	const errWrapMessage = "register input validate"

	if err := utils.ValidateUserID(ctx, i.UserID); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}

	if err := utils.ValidatePositiveDecimal(ctx, i.Amount); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}

	return nil
}
