package orderbook

import (
	"container/list"
	"fmt"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/btree"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	ErrInvalidTypeInList      = errors.New("List doesn't store Order type object")
	ErrBestAskInvariantBroken = errors.New("No best ask found inside the loop")
)

type deletableNode struct {
	node *list.Element
	list *list.List
}

type Book struct {
	asks   *asks
	bids   *bids
	orders map[uuid.UUID]deletableNode // reverse lookup map to list nodes stored in price levels of asks/bids
}

// Matches the orders and forms the trades; If Order cannot be fully filled immediately
// appends it to the appropriate resting queue
func (b *Book) AddOrder(order *Order) ([]Trade, error) {
	trades := make([]Trade, 0)
	oppositeSide := b.oppositeSide(order.Side)

	for oppositeSide.Len() > 0 && !order.IsFilled() {
		bestPriceLevel, ok := oppositeSide.getBestPriceLevel()
		if !ok {
			return nil, ErrBestAskInvariantBroken
		}
		if !order.IsSatisfactoryPrice(bestPriceLevel.Price) {
			break
		}
		plTrades, err := b.createTradesForPriceLevel(bestPriceLevel, order, trades, oppositeSide)
		if err != nil {
			return nil, errors.Wrap(err, "match price level")
		}
		trades = append(trades, plTrades...)
	}
	// no more trades can be produced
	if order.IsFilled() {
		return trades, nil
	}
	b.insertRestingOrder(order)
	return trades, nil
}

func (b *Book) CancelOrder(id uuid.UUID) (bool, error) {
	nodeInfo, ok := b.orders[id]
	if !ok {
		return false, nil
	}
	order, ok := nodeInfo.node.Value.(*Order)
	if !ok {
		return false, ErrInvalidTypeInList
	}
	nodeInfo.list.Remove(nodeInfo.node)
	delete(b.orders, id)
	if nodeInfo.list.Len() > 0 {
		return true, nil
	}
	// if it was last order - delete price level
	_, ok = b.sameSide(order.Side).Delete(PriceLevel{Price: order.Price})
	if !ok {
		return false, fmt.Errorf("delete price level: %d", order.Price)
	}
	return true, nil
}

func NewBook(b int) *Book {
	return &Book{
		asks:   &asks{btree.NewG[PriceLevel](b, less)},
		bids:   &bids{btree.NewG[PriceLevel](b, less)},
		orders: make(map[uuid.UUID]deletableNode),
	}
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

func (b *Book) insertRestingOrder(order *Order) {
	sameSide := b.sameSide(order.Side)
	priceLvl, ok := sameSide.Get(PriceLevel{Price: order.Price})

	var nodeWithNewOrder *list.Element
	// price level doesn't exist in bids, create new one
	if !ok {
		orderList := list.New()
		nodeWithNewOrder = orderList.PushBack(order)
		priceLvl = PriceLevel{order.Price, orderList}
		sameSide.ReplaceOrInsert(priceLvl)
	} else {
		// price level exists, just add order to the end of the list
		nodeWithNewOrder = priceLvl.Orders.PushBack(order)
	}
	b.orders[order.ID] = deletableNode{
		node: nodeWithNewOrder,
		list: priceLvl.Orders,
	}
}

// Iterates over price level and creates trades; created for main matching function simplification;
// Must be used only on already matched price level (this function doesn't intend to check matching condition)
func (b *Book) createTradesForPriceLevel(bestPriceLevel PriceLevel, order *Order, trades []Trade, oppositeSide bookSide) ([]Trade, error) {
	// iterate within price level
	for bestPriceLevel.Orders.Len() > 0 && !order.IsFilled() {
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
		if topOrder.IsFilled() {
			bestPriceLevel.Orders.Remove(topNode)
			delete(b.orders, topOrder.ID)
		}
	}
	// if best price level was fully exhausted - delete it
	if bestPriceLevel.Orders.Len() == 0 {
		oppositeSide.Delete(bestPriceLevel)
	}
	return trades, nil
}
