package getorders

import (
	"context"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	"github.com/Murlokich/Dixchange/internal/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Input struct {
	UserID   uuid.UUID
	Statuses []string
}

func (i *Input) Validate(ctx context.Context) error {
	if err := utils.ValidateUserID(ctx, i.UserID); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, "validate user id")
	}

	// only active orders can be fetched for now
	if err := validation.ValidateWithContext(
		ctx,
		i.Statuses,
		validation.Required,
		validation.Each(
			// In accepts empty values, so Required rejects an empty status in the list
			validation.Required,
			validation.In(entity.StatusOpen, entity.StatusPartiallyFilled).Error("is not supported status"),
		),
	); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, "validate statuses")
	}

	return nil
}
