package entity

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type OrderCommandType string

const (
	OrderCommandPlace  OrderCommandType = "place"
	OrderCommandCancel OrderCommandType = "cancel"
)

// OrderCommand is a request accepted for the matching engine whose effect hasn't been
// applied to the orders yet. Place is set only for a place command; a cancel command
// only names the order to cancel.
type OrderCommand struct {
	ID      uuid.UUID
	Type    OrderCommandType
	OrderID uuid.UUID
	Place   *PlaceOrderDetails
}

// PlaceOrderDetails is the order a place command creates, apart from what OrderCommand
// already carries.
type PlaceOrderDetails struct {
	UserID       uuid.UUID
	PairID       int
	Side         string
	PricePerUnit int64
	Quantity     int64
}

func NewPlaceOrderCommand(id uuid.UUID, order Order) OrderCommand {
	return OrderCommand{
		ID:      id,
		Type:    OrderCommandPlace,
		OrderID: order.ID,
		Place: &PlaceOrderDetails{
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

// PlacedOrder returns the order a place command creates, as it is before any fill. The
// order carries only the market pair ID; the caller resolves the rest of the pair.
func (c *OrderCommand) PlacedOrder() (Order, error) {
	if c.Type != OrderCommandPlace || c.Place == nil {
		return Order{}, errors.Errorf("order command %s is %s, not place", c.ID, c.Type)
	}
	return Order{
		ID:              c.OrderID,
		UserID:          c.Place.UserID,
		MarketPair:      Pair{ID: c.Place.PairID},
		Side:            c.Place.Side,
		QuantityOrdered: c.Place.Quantity,
		PricePerUnit:    c.Place.PricePerUnit,
		Status:          StatusOpen,
	}, nil
}
