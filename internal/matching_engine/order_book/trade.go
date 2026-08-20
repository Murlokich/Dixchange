package orderbook

import "github.com/google/uuid"

type Trade struct {
	AskOrderID uuid.UUID
	BidOrderID uuid.UUID
	Price      int64
	Quantity   int64
}
