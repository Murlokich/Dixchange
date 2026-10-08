package entity

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type OrderCommandType string

const (
	OrderCommandCreate OrderCommandType = "create"
	OrderCommandCancel OrderCommandType = "cancel"
)

// OrderCommand is a request accepted for the matching engine whose effect hasn't been
// applied to the orders yet. OrderDetails is set only for a create command; a cancel
// command only names the order to cancel.
type OrderCommand struct {
	ID           uuid.UUID
	Type         OrderCommandType
	OrderID      uuid.UUID
	OrderDetails *OrderDetails
}

// OrderDetails is the order a create command creates, apart from what OrderCommand
// already carries. It is stored in the command log as JSON: renaming a JSON key breaks
// reading the commands already stored.
type OrderDetails struct {
	UserID       uuid.UUID `json:"user_id"`
	PairID       int       `json:"pair_id"`
	Side         string    `json:"side"`
	PricePerUnit int64     `json:"price"`
	Quantity     int64     `json:"quantity"`
}

func NewCreateOrderCommand(id uuid.UUID, order Order) OrderCommand {
	return OrderCommand{
		ID:      id,
		Type:    OrderCommandCreate,
		OrderID: order.ID,
		OrderDetails: &OrderDetails{
			UserID:       order.UserID,
			PairID:       order.MarketPair.ID,
			Side:         order.Side,
			PricePerUnit: order.PricePerUnit,
			Quantity:     order.QuantityOrdered,
		},
	}
}

func NewCancelOrderCommand(id, orderID uuid.UUID) OrderCommand {
	return OrderCommand{
		ID:      id,
		Type:    OrderCommandCancel,
		OrderID: orderID,
	}
}

// CreatedOrder returns the order a create command creates, as it is before any fill. The
// order carries only the market pair ID; the caller resolves the rest of the pair.
func (c *OrderCommand) CreatedOrder() (Order, error) {
	if c.Type != OrderCommandCreate || c.OrderDetails == nil {
		return Order{}, errors.Errorf("order command %s is %s, not create", c.ID, c.Type)
	}
	return Order{
		ID:              c.OrderID,
		UserID:          c.OrderDetails.UserID,
		MarketPair:      Pair{ID: c.OrderDetails.PairID},
		Side:            c.OrderDetails.Side,
		QuantityOrdered: c.OrderDetails.Quantity,
		PricePerUnit:    c.OrderDetails.PricePerUnit,
		Status:          StatusOpen,
	}, nil
}
