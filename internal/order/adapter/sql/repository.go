package sql

import (
	"context"
	"encoding/json"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	"github.com/Murlokich/Dixchange/pkg"
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

// CreateOrderCommand appends the command to the command log.
func (r *Repository) CreateOrderCommand(ctx context.Context, command entity.OrderCommand) error {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	var orderDetails []byte
	if command.OrderDetails != nil {
		var err error
		orderDetails, err = json.Marshal(command.OrderDetails)
		if err != nil {
			return errors.Wrap(err, "marshal order details")
		}
	}

	if err := q.CreateOrderCommand(ctx, CreateOrderCommandParams{
		ID:           command.ID,
		Type:         OrderCommandType(command.Type),
		OrderID:      command.OrderID,
		OrderDetails: orderDetails,
	}); err != nil {
		return errors.Wrap(err, "query create order command")
	}

	return nil
}

// TakeOrderCommand deletes the command from the command log and returns it. It is meant
// to be called within the transaction that applies the command, so the command is gone
// exactly when its effect is in the orders.
func (r *Repository) TakeOrderCommand(ctx context.Context, commandID uuid.UUID) (entity.OrderCommand, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	row, err := q.TakeOrderCommand(ctx, commandID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.OrderCommand{}, errors.Errorf("order command %s doesn't exist", commandID)
		}
		return entity.OrderCommand{}, errors.Wrap(err, "query take order command")
	}

	command, err := rowToCommand(row)
	if err != nil {
		return entity.OrderCommand{}, errors.Wrap(err, "rowToCommand")
	}

	return command, nil
}

func (r *Repository) GetAllOpenOrders(ctx context.Context) ([]entity.Order, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.GetAllOpenOrders(ctx)
	if err != nil {
		return []entity.Order{}, errors.Wrap(err, "query GetAllOpenOrders")
	}

	return rowsToOrder(rows), nil
}

func (r *Repository) GetAllCommands(ctx context.Context) ([]entity.OrderCommand, error) {
	db := r.txGetter(ctx, r.db)
	q := New(db)

	rows, err := q.GetAllCommands(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "query GetAllCommands")
	}

	cmds, err := rowsToCommands(rows)
	if err != nil {
		return nil, errors.Wrap(err, "rowsToCommands")
	}
	return cmds, nil
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

func rowsToOrder(rows []OrderOrder) []entity.Order {
	orders := make([]entity.Order, len(rows))
	for i := range rows {
		orders[i] = rowToOrder(rows[i])
	}
	return orders
}

func rowToCommand(row OrderCommandLogUsdKcn) (entity.OrderCommand, error) {
	command := entity.OrderCommand{
		ID:      row.ID,
		Type:    entity.OrderCommandType(string(row.Type)),
		OrderID: row.OrderID,
	}

	// fill the order details if command is for order creation
	if command.Type == entity.OrderCommandCreate {
		var orderDetails entity.OrderDetails
		if err := json.Unmarshal(row.OrderDetails, &orderDetails); err != nil {
			return entity.OrderCommand{}, errors.Wrapf(err, "unmarshal order details of command %s", row.ID)
		}
		command.OrderDetails = &orderDetails
	}

	return command, nil
}

func rowsToCommands(rows []OrderCommandLogUsdKcn) ([]entity.OrderCommand, error) {
	cmds := make([]entity.OrderCommand, len(rows))

	for i := range rows {
		cmd, err := rowToCommand(rows[i])
		if err != nil {
			return nil, errors.Wrap(err, "rowToCommand")
		}
		cmds[i] = cmd
	}
	return cmds, nil
}

func NewRepository(db DBTX, txGetter pkg.TXGetter) *Repository {
	return &Repository{db: db, txGetter: txGetter}
}
