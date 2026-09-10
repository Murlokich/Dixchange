package sql

import (
	"context"
	"encoding/json"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/pkg"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Repository struct {
	db       DBTX
	txGetter pkg.TXGetter
}

// tradeRow is the JSON shape consumed by the CreateTrades query. jsonb_to_recordset
// matches object keys to the column definition list of that query, so these tags must
// stay in sync with it.
type tradeRow struct {
	ID          uuid.UUID `json:"id"`
	BuyOrderID  uuid.UUID `json:"buy_order_id"`
	SellOrderID uuid.UUID `json:"sell_order_id"`
	Price       int64     `json:"price"`
	Quantity    int64     `json:"quantity"`
}

func (r *Repository) CreateTrades(ctx context.Context, trades []entity.Trade) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows := make([]tradeRow, len(trades))
	for i := range trades {
		rows[i] = tradeRow{
			ID:          trades[i].ID,
			BuyOrderID:  trades[i].BidOrderID,
			SellOrderID: trades[i].AskOrderID,
			Price:       trades[i].Price,
			Quantity:    trades[i].Quantity,
		}
	}

	payload, err := json.Marshal(rows)
	if err != nil {
		return errors.Wrap(err, "marshal trades")
	}

	if err = q.CreateTrades(ctx, payload); err != nil {
		return errors.Wrap(err, "query create trades")
	}

	return nil
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
