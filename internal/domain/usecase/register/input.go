package register

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/utils"
	"github.com/pkg/errors"
)

const (
	passwordMinLength = 8
	passwordMaxLength = 256
)

type Input struct {
	Email    string
	Password string
}

func (i *Input) Validate(ctx context.Context) error {
	const errWrapMessage = "register input validate"
	if err := utils.ValidateEmail(ctx, i.Email); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}
	if err := utils.ValidatePassword(ctx, i.Password); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}

	return nil
}
