package entity

import (
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// TradeParty holds the facts about one side of a trade that were true at match time
// and stay true afterwards. Mutable order state, such as status or filled quantity,
// is deliberately left out: it is reached through OrderID instead.
type TradeParty struct {
	OrderID uuid.UUID
	UserID  uuid.UUID
	// OrderPrice is the price the order was placed at, which is not necessarily the
	// price the trade executed at: a trade takes the resting order price.
	OrderPrice int64
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

// Notional returns what the buyer pays for the trade, in quote minor units.
func (t *Trade) Notional() (int64, error) {
	amount, err := t.MarketPair.Notional(t.Quantity, t.Price)
	if err != nil {
		return 0, errors.Wrap(err, "market pair notional")
	}
	return amount, nil
}

// GetExcessReservedAmount returns the quote funds the buyer reserved on top of what the
// trade actually cost, in quote minor units. A buy order reserves the quote asset at
// its own price, but the trade executes at the resting order price, which is never
// higher. Only the bid side can hold a remainder: sellers reserve the base asset in
// quantity units, which a fill consumes exactly.
func (t *Trade) GetExcessReservedAmount() (int64, error) {
	excessPerUnit := t.Bid.OrderPrice - t.Price
	if excessPerUnit <= 0 {
		return 0, nil
	}
	amount, err := t.MarketPair.Notional(t.Quantity, excessPerUnit)
	if err != nil {
		return 0, errors.Wrap(err, "market pair notional")
	}
	return amount, nil
}
