package createorder

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Input struct {
	UserID          uuid.UUID
	MarketPairID    int
	Action          string
	QuantityOrdered decimal.Decimal
	PricePerUnit    decimal.Decimal
}

func (i *Input) Validate(ctx context.Context) error {
	if err := utils.ValidateUserID(ctx, i.UserID); err != nil {
		return entity.ValidationError{Err: err}
	}
	if err := validation.ValidateWithContext(ctx, i.MarketPairID, validation.Required, validation.Min(0)); err != nil {
		return entity.ValidationError{Err: err}
	}
	if err := validation.ValidateWithContext(
		ctx,
		i.Action,
		validation.Required.Error("status cannot be blank"),
		validation.In(entity.BuyAction, entity.SellAction).Error("is not valid status"),
	); err != nil {
		return entity.ValidationError{Err: err}
	}
	if err := utils.ValidatePositiveDecimal(ctx, i.QuantityOrdered); err != nil {
		return entity.ValidationError{Err: err}
	}
	if err := utils.ValidatePositiveDecimal(ctx, i.PricePerUnit); err != nil {
		return entity.ValidationError{Err: err}
	}
	return nil
}
