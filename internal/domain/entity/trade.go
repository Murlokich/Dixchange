package entity

import (
	"time"

	"github.com/google/uuid"
)

type Trade struct {
	ID         uuid.UUID
	AskOrderID uuid.UUID
	BidOrderID uuid.UUID
	Price      int64
	Quantity   int64
	CreatedAt  time.Time
}
