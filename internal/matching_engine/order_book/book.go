package orderbook

import (
	"cmp"
	"container/list"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/btree"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	ErrInvalidTypeInList      = errors.New("List doesn't store Order type object")
	ErrBestAskInvariantBroken = errors.New("No best ask found inside the loop")
)

type PriceLevel struct {
	Price  int64
	Orders *list.List
}

func less(x, y PriceLevel) bool {
	return cmp.Less(x.Price, y.Price)
}

type Book struct {
	asks   *asks
	bids   *bids
	orders map[uuid.UUID]*list.Element // reverse lookup map to list nodes stored in price levels of asks/bids
}

func (b *Book) AddOrder(order Order) []Trade {
	return nil
}

func (b *Book) matchOrder(order *Order) ([]Trade, error) {
	trades := make([]Trade, 0)
	oppositeSide := b.oppositeSide(order.Side)
	// filling loop - iterate until can still produce trades
	// while there are price levels and order is not filled yet
	for oppositeSide.Len() > 0 && order.RemainingQuantity > 0 {
		bestPriceLevel, ok := oppositeSide.getBestPriceLevel()
		if !ok {
			return nil, ErrBestAskInvariantBroken
		}
		// price doesn't satisfy the order
		if !order.IsSatisfactoryPrice(bestPriceLevel.Price) {
			break
		}
		// loop over orders on the price level for trades creation
		for bestPriceLevel.Orders.Len() > 0 && order.RemainingQuantity > 0 {
			// extract first order in the queue
			topNode := bestPriceLevel.Orders.Front()
			topOrder, ok := topNode.Value.(*Order)
			if !ok {
				return nil, ErrInvalidTypeInList
			}
			trade, err := executeTrade(topOrder, order)
			if err != nil {
				return nil, errors.Wrap(err, "executeTrade")
			}
			trades = append(trades, trade)
			// if resting order was filled - remove from the order book
			if topOrder.RemainingQuantity == 0 {
				bestPriceLevel.Orders.Remove(topNode)
				delete(b.orders, topOrder.ID)
			}
		}
		// if best price level was fully exhausted - delete it
		if bestPriceLevel.Orders.Len() == 0 {
			oppositeSide.Delete(bestPriceLevel)
		}
	}
	// no more trades can be produced

	if order.RemainingQuantity == 0 {
		return trades, nil
	}
	sameSide := b.sameSide(order.Side)
	// insert order to the resting queue
	priceLvl, ok := sameSide.Get(PriceLevel{order.Price, nil})

	var nodeWithNewOrder *list.Element
	// price level doesn't exist in bids, create new one
	if !ok {
		// initialize new list with provided order
		orderList := list.New()
		nodeWithNewOrder = orderList.PushBack(order)
		// create new price level
		priceLvl = PriceLevel{order.Price, orderList}
		sameSide.ReplaceOrInsert(priceLvl)
	} else {
		// price level exists, just add order to the end of the list
		nodeWithNewOrder = priceLvl.Orders.PushBack(order)
	}
	b.orders[order.ID] = nodeWithNewOrder
	return trades, nil
}

// returns pointer to the requested side
func (b *Book) sameSide(side string) bookSide {
	if side == entity.BuyAction {
		return b.bids
	}
	return b.asks
}

// returns pointer to the opposite of the requested side
func (b *Book) oppositeSide(side string) bookSide {
	if side == entity.BuyAction {
		return b.asks
	}
	return b.bids
}

// Takes resting order and new one and creates trade with
// min quanity and resting order's price and updates the orders state
func executeTrade(restingOrder *Order, newOrder *Order) (Trade, error) {
	var askOrder, bidOrder *Order
	// determine which order is ask and bid
	if restingOrder.Side == entity.BuyAction {
		askOrder, bidOrder = newOrder, restingOrder
	} else {
		askOrder, bidOrder = restingOrder, newOrder
	}
	// form the trade with resting order price and min quantity
	tradeQuantity := min(restingOrder.RemainingQuantity, newOrder.RemainingQuantity)
	// update orders
	if err := restingOrder.Fill(tradeQuantity); err != nil {
		return Trade{}, errors.Wrap(err, "resting order fill")
	}
	if err := newOrder.Fill(tradeQuantity); err != nil {
		return Trade{}, errors.Wrap(err, "new order fill")
	}
	return Trade{
		AskOrderID: askOrder.ID,
		BidOrderID: bidOrder.ID,
		Price:      restingOrder.Price,
		Quantity:   tradeQuantity,
	}, nil
}

func NewBook(b int) *Book {
	return &Book{
		asks:   &asks{btree.NewG[PriceLevel](b, less)},
		bids:   &bids{btree.NewG[PriceLevel](b, less)},
		orders: make(map[uuid.UUID]*list.Element),
	}
}
