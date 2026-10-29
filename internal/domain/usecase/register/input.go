package register

import (
	"context"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	"github.com/Murlokich/Dixchange/internal/utils"
	"github.com/pkg/errors"
)

type Input struct {
	Email    string
	Password string
}

func (i *Input) Validate(ctx context.Context) error {
	if err := utils.ValidateEmail(ctx, i.Email); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, "validate email")
	}
	if err := utils.ValidatePassword(ctx, i.Password); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, "validate password")
	}

	return nil
}
