package getpairbalance

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Input struct {
	UserID uuid.UUID
	PairID int
}

func (i *Input) Validate(ctx context.Context) error {
	if err := utils.ValidateUserID(ctx, i.UserID); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, "validate user id")
	}

	if err := validation.ValidateWithContext(ctx, i.PairID, validation.Required, validation.Min(0)); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, "validate pair id")
	}

	return nil
}
