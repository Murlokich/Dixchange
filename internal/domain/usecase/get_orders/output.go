package getorders

import "github.com/dixchange/Monolith/internal/domain/entity"

type Output struct {
	// Orders carry their full market pair so amounts can be formatted at its precisions.
	Orders []entity.Order
}
