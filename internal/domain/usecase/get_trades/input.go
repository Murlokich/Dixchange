package gettrades

import (
	"context"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	"github.com/Murlokich/Dixchange/internal/utils"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Input struct {
	UserID uuid.UUID
	// Cursor is the ID of the last trade of the previous page; nil means the first page.
	Cursor *uuid.UUID
}

func (i *Input) Validate(ctx context.Context) error {
	if err := utils.ValidateUserID(ctx, i.UserID); err != nil {
		return errors.Wrap(entity.ValidationError{Err: err}, "validate user id")
	}
	return nil
}
