package sql

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/pkg"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

	var side OrderSide
	if err := side.Scan(order.Side); err != nil {
		return errors.Wrap(err, "order side scan")
	}

	if err := q.CreateOrder(ctx, CreateOrderParams{
		ID:     order.ID,
		UserID: order.UserID,
		//nolint:gosec //int32 is sqlc gen type + is not expected to overflow
		MarketPairID:    int32(order.MarketPair.ID),
		Side:            side,
		QuantityOrdered: order.QuantityOrdered,
		Price:           order.PricePerUnit,
		Status:          order.Status,
	}); err != nil {
		return errors.Wrap(err, "query create order")
	}

	return nil
}

// FillOrder adds quantity to the filled quantity of the order and marks it filled
// once the ordered quantity is reached, partially filled otherwise.
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

// GetUserOrdersByStatuses returns the user orders with any of the given statuses, newest first.
// The returned orders carry only the market pair ID; the caller resolves the rest of the pair.
func (r *Repository) GetUserOrdersByStatuses(
	ctx context.Context,
	userID uuid.UUID,
	statuses []string,
) ([]entity.Order, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.GetUserOrdersByStatuses(ctx, GetUserOrdersByStatusesParams{
		UserID: userID,
		// sqlc can't infer a name for the ANY($2::text[]) parameter
		Column2: statuses,
	})
	if err != nil {
		return nil, errors.Wrap(err, "query get user orders by statuses")
	}

	orders := make([]entity.Order, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, rowToOrder(row))
	}
	return orders, nil
}

// GetOrderCancelInfo returns what is needed to decide whether the order may be cancelled.
// It returns entity.ErrOrderNotFound when there is no such order.
func (r *Repository) GetOrderCancelInfo(ctx context.Context, orderID uuid.UUID) (entity.OrderCancelInfo, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	row, err := q.GetOrderCancelInfo(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.OrderCancelInfo{}, errors.Wrapf(entity.ErrOrderNotFound, "order %s", orderID)
		}
		return entity.OrderCancelInfo{}, errors.Wrap(err, "query get order cancel info")
	}

	return entity.OrderCancelInfo{UserID: row.UserID, Status: row.Status}, nil
}

// GetOrderForUpdate returns the order and locks its row until the end of the transaction,
// so the order can't change between reading and updating it. It is meant to be called
// within a transaction. The returned order carries only the market pair ID; the caller
// resolves the rest of the pair.
func (r *Repository) GetOrderForUpdate(ctx context.Context, orderID uuid.UUID) (entity.Order, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	row, err := q.GetOrderForUpdate(ctx, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Order{}, errors.Wrapf(entity.ErrOrderNotFound, "order %s", orderID)
		}
		return entity.Order{}, errors.Wrap(err, "query get order for update")
	}

	return rowToOrder(row), nil
}

// CancelOrder marks the order cancelled. It doesn't check the current status: the caller
// is expected to have checked it on the order locked by GetOrderForUpdate.
func (r *Repository) CancelOrder(ctx context.Context, orderID uuid.UUID) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.CancelOrder(ctx, orderID)
	if err != nil {
		return errors.Wrap(err, "query cancel order")
	}
	if rows == noOrderAffected {
		return errors.Errorf("order %s to cancel doesn't exist", orderID)
	}

	return nil
}

func rowToOrder(row OrderOrder) entity.Order {
	return entity.Order{
		ID:              row.ID,
		UserID:          row.UserID,
		MarketPair:      entity.Pair{ID: int(row.MarketPairID)},
		Side:            string(row.Side),
		QuantityOrdered: row.QuantityOrdered,
		QuantityFilled:  row.QuantityFilled,
		PricePerUnit:    row.Price,
		Status:          row.Status,
		CreatedAt:       row.CreatedAt.Time,
		// NULL last_updated_at means the order was never updated, so the zero time is right.
		LastUpdatedAt: row.LastUpdatedAt.Time,
	}
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
