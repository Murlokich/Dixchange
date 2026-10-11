package orderbook

import (
	"errors"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
)

const (
	LimitOrder = "limit"
)

var (
	ErrOrderOverfill = errors.New("Fill for more than remaining quantity called for order")
)

type Order struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	OrderType         string
	Side              string
	Price             int64
	RemainingQuantity int64
}

func (o *Order) Fill(quantity int64) error {
	if quantity > o.RemainingQuantity {
		return ErrOrderOverfill
	}
	o.RemainingQuantity -= quantity
	return nil
}

func (o *Order) IsSatisfactoryPrice(price int64) bool {
	if o.Side == entity.BuySide {
		return price <= o.Price
	}
	return price >= o.Price
}

func (o *Order) IsFilled() bool {
	return o.RemainingQuantity == 0
}

func NewOrderFromEntity(order entity.Order) Order {
	return Order{
		ID:                order.ID,
		UserID:            order.UserID,
		OrderType:         LimitOrder,
		Side:              order.Side,
		Price:             order.PricePerUnit,
		RemainingQuantity: order.RemainingQuantity(),
	}
}
