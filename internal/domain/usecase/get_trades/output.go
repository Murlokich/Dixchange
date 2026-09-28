package gettrades

import (
	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
)

type Output struct {
	// Trades are ordered newest first.
	Trades []UserTrade
	// NextCursor is the ID of the last returned trade, where the next page starts;
	// nil when there are no more trades.
	NextCursor *uuid.UUID
}

// UserTrade is a trade as seen by one of its participants.
type UserTrade struct {
	// Trade carries its full market pair so amounts can be formatted at its precisions.
	Trade entity.Trade
	Party entity.UserTradeParty
	// Total is the trade notional in quote minor units.
	Total int64
}
