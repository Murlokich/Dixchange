package entity

import (
	"time"

	"github.com/google/uuid"
)

// TradeParty holds the facts about one side of a trade that were true at match time
// and stay true afterwards. Mutable order state, such as status or filled quantity,
// is deliberately left out: it is reached through OrderID instead.
type TradeParty struct {
	OrderID uuid.UUID
	UserID  uuid.UUID
}

type Trade struct {
	ID uuid.UUID
	// MarketPair is always fully populated: the matching engine stamps it from the
	// book that produced the trade, which is the only place trades are created.
	MarketPair Pair
	Ask        TradeParty
	Bid        TradeParty
	Price      int64
	Quantity   int64
	CreatedAt  time.Time
}
