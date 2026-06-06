package register

import (
	"context"
	"regexp"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/pkg/errors"
)

type Input struct {
	Email    string
	Password string
}

func (i *Input) Validate(ctx context.Context) error {
	const errWrapMessage = "register input validate"
	if err := validation.Validate(i.Email, validation.Required, is.EmailFormat); err != nil {
		return errors.Wrap(err, errWrapMessage)
	}
	if err := validation.Validate(
		i.Password,
		validation.Required,
		validation.Length(8, 256),
		validation.Match(regexp.MustCompile(".*[A-Z].*")),                                 // Contains upper case letter
		validation.Match(regexp.MustCompile(".*[a-z].*")),                                 // Contains lower case letter
		validation.Match(regexp.MustCompile(".*[0-9].*")),                                 // Contains digit
		validation.Match(regexp.MustCompile(".*[!@#$%^&*()_+\\-=\\[\\]{};:'\",.<>?/].*")), // contains special character
	); err != nil {
		return errors.Wrap(err, errWrapMessage)
	}

	return nil
}
