package orderbook

import "github.com/dixchange/Monolith/internal/domain/entity"

type Trade struct {
	Ask      entity.TradeParty
	Bid      entity.TradeParty
	Price    int64
	Quantity int64
}
