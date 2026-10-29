package getorders

import "github.com/Murlokich/Dixchange/internal/domain/entity"

type Output struct {
	// Orders carry their full market pair so amounts can be formatted at its precisions.
	Orders []entity.Order
}
