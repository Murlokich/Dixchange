package login

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/utils"
	"github.com/pkg/errors"
)

type Input struct {
	Email    string
	Password string
}

func (i *Input) Validate(ctx context.Context) error {
	const errWrapMessage = "login input validate"
	if err := utils.ValidateEmail(ctx, i.Email); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}
	if err := utils.ValidatePassword(ctx, i.Password); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}

	return nil
}
