package login

import (
	"context"
	"regexp"

	"github.com/dixchange/Monolith/internal/domain/entity"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
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
	if err := validation.ValidateWithContext(ctx, i.Email, validation.Required, is.EmailFormat); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}
	if err := validation.ValidateWithContext(
		ctx,
		i.Password,
		validation.Required,
		validation.Length(passwordMinLength, passwordMaxLength),
		validation.Match(regexp.MustCompile(".*[A-Z].*")),                                 // Contains upper case letter
		validation.Match(regexp.MustCompile(".*[a-z].*")),                                 // Contains lower case letter
		validation.Match(regexp.MustCompile(".*[0-9].*")),                                 // Contains digit
		validation.Match(regexp.MustCompile(".*[!@#$%^&*()_+\\-=\\[\\]{};:'\",.<>?/].*")), // contains special character
	); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, errWrapMessage)
	}

	return nil
}
