package cancelorder

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var ErrEmptyOrderID = errors.New("order id cannot be empty")

type Input struct {
	UserID  uuid.UUID
	OrderID uuid.UUID
}

func (i *Input) Validate(ctx context.Context) error {
	if err := utils.ValidateUserID(ctx, i.UserID); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, "validate user id")
	}
	if i.OrderID == uuid.Nil {
		return errors.Wrap(entity.ValidationError{Err: ErrEmptyOrderID}, "validate order id")
	}
	return nil
}
