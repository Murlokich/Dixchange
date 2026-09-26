package getorders

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
	Status string
}

func (i *Input) Validate(ctx context.Context) error {
	if err := utils.ValidateUserID(ctx, i.UserID); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, "validate user id")
	}

	// only open orders can be fetched for now
	if err := validation.ValidateWithContext(
		ctx,
		i.Status,
		validation.Required,
		validation.In(entity.StatusOpen).Error("is not supported status"),
	); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, "validate status")
	}

	return nil
}
