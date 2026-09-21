package sql

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/pkg"
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
		BuyOrderID:  trade.BidOrderID,
		SellOrderID: trade.AskOrderID,
		Price:       trade.Price,
		Quantity:    trade.Quantity,
	}); err != nil {
		return errors.Wrap(err, "query create trade")
	}

	return nil
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
