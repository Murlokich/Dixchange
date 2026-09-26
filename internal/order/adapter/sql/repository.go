package sql

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/pkg"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	noOrderAffected = 0
)

type Repository struct {
	db       DBTX
	txGetter pkg.TXGetter
}

func (r *Repository) CreateOrder(ctx context.Context, order entity.Order) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	var action OrderAction
	if err := action.Scan(order.Side); err != nil {
		return errors.Wrap(err, "order side scan")
	}

	if err := q.CreateOrder(ctx, CreateOrderParams{
		ID:     order.ID,
		UserID: order.UserID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		MarketPairID:    int32(order.MarketPair.ID),
		Action:          action,
		QuantityOrdered: order.QuantityOrdered,
		Price:           order.PricePerUnit,
		Status:          order.Status,
	}); err != nil {
		return errors.Wrap(err, "query create order")
	}

	return nil
}

// FillOrder adds quantity to the filled quantity of the order and marks it filled
// once the ordered quantity is reached.
func (r *Repository) FillOrder(ctx context.Context, orderID uuid.UUID, quantity int64) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.FillOrder(ctx, FillOrderParams{
		Quantity: quantity,
		OrderID:  orderID,
	})
	if err != nil {
		return errors.Wrap(err, "query fill order")
	}
	if rows == noOrderAffected {
		return errors.Errorf("order %s to fill doesn't exist", orderID)
	}

	return nil
}

// GetUserOrdersByStatus returns the user orders with the given status, newest first. The returned orders
// carry only the market pair ID; the caller resolves the rest of the pair.
func (r *Repository) GetUserOrdersByStatus(
	ctx context.Context,
	userID uuid.UUID,
	status string,
) ([]entity.Order, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.GetUserOrdersByStatus(ctx, GetUserOrdersByStatusParams{UserID: userID, Status: status})
	if err != nil {
		return nil, errors.Wrap(err, "query get user orders by status")
	}

	orders := make([]entity.Order, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, entity.Order{
			ID:              row.ID,
			UserID:          row.UserID,
			MarketPair:      entity.Pair{ID: int(row.MarketPairID)},
			Side:            string(row.Action),
			QuantityOrdered: row.QuantityOrdered,
			// NULL quantity_filled and last_updated_at mean the order was never filled
			// or updated, so their zero values are the right domain values.
			QuantityFilled: row.QuantityFilled.Int64,
			PricePerUnit:   row.Price,
			Status:         row.Status,
			CreatedAt:      row.CreatedAt.Time,
			LastUpdatedAt:  row.LastUpdatedAt.Time,
		})
	}
	return orders, nil
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
