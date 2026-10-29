package sql

import (
	"context"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	"github.com/Murlokich/Dixchange/pkg"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Repository struct {
	db       DBTX
	txGetter pkg.TXGetter
}

func (r *Repository) CreateTrade(ctx context.Context, trade entity.Trade) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	if err := q.CreateTrade(ctx, CreateTradeParams{
		ID:          trade.ID,
		BuyOrderID:  trade.Bid.OrderID,
		SellOrderID: trade.Ask.OrderID,
		BuyerID:     trade.Bid.UserID,
		SellerID:    trade.Ask.UserID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		PairID:   int32(trade.MarketPair.ID),
		Price:    trade.Price,
		Quantity: trade.Quantity,
	}); err != nil {
		return errors.Wrap(err, "query create trade")
	}

	return nil
}

// GetUserTrades returns up to limit trades the user took part in, newest first, that are
// older than the trade with the cursor ID. A nil cursor starts from the newest trade.
// Trade IDs are UUIDv7, so the ID order is the creation order. The returned trades carry
// only the market pair ID and no order prices; the caller resolves the pair.
func (r *Repository) GetUserTrades(
	ctx context.Context,
	userID uuid.UUID,
	cursor *uuid.UUID,
	limit int,
) ([]entity.Trade, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	// every trade ID is below the max UUID, so the first page needs no special query
	before := uuid.Max
	if cursor != nil {
		before = *cursor
	}

	rows, err := q.GetUserTrades(ctx, GetUserTradesParams{
		// sqlc names the parameter after buyer_id, but it matches seller_id as well
		BuyerID: userID,
		ID:      before,
		//nolint:gosec //int32 is sqlc gen type + page size is small
		Limit: int32(limit),
	})
	if err != nil {
		return nil, errors.Wrap(err, "query get user trades")
	}

	trades := make([]entity.Trade, 0, len(rows))
	for _, row := range rows {
		trades = append(trades, entity.Trade{
			ID:         row.ID,
			MarketPair: entity.Pair{ID: int(row.PairID)},
			Bid:        entity.TradeParty{OrderID: row.BuyOrderID, UserID: row.BuyerID},
			Ask:        entity.TradeParty{OrderID: row.SellOrderID, UserID: row.SellerID},
			Price:      row.Price,
			Quantity:   row.Quantity,
			CreatedAt:  row.CreatedAt.Time,
		})
	}
	return trades, nil
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
