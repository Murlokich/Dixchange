package orderbook

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/dixchange/Monolith/internal/config"
	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	noTrades = []Trade{}    //nolint:gochecknoglobals // acceptable in test
	testPair = entity.Pair{ //nolint:gochecknoglobals // acceptable in test
		ID:        1,
		Base:      entity.Symbol{ID: 1, Code: "USD", Precision: 2},
		Quote:     entity.Symbol{ID: 2, Code: "KCN", Precision: 0},
		LotSize:   1,
		PriceTick: 1,
	}
)

type testCase struct {
	steps          []step
	finalBookState finalBookState
}

type step struct {
	order          *Order
	cancelID       *uuid.UUID
	wantTrades     []Trade
	topAskQuantity *int64 // if nil - don't check it
	topBidQuantity *int64 // if nil - don't check it
}

type finalBookState struct {
	asksShape map[int64]int // map from price level to number of orders
	bidsShape map[int64]int // map from price level to number of orders
	orderIDs  []uuid.UUID
}

type orderInput struct {
	side     string
	price    int64
	quantity int64
}

// This test inserts 9 sell orders across three different price leves: [78, 79, 80]
// Then appends big limit order with price 81 - having exceeding quantity over all existing orders
// Expected to match all the appended orders, clear the order book and keep single resting buying order.
func TestBook_AddOrder_BigOrderMatchesMultiplePriceLevels(t *testing.T) {
	orderInputs := []orderInput{
		{
			side:     entity.SellAction,
			price:    80,
			quantity: 10,
		},
		{
			side:     entity.SellAction,
			price:    80,
			quantity: 15,
		}, {
			side:     entity.SellAction,
			price:    80,
			quantity: 1,
		},
		{
			side:     entity.SellAction,
			price:    80,
			quantity: 7,
		},
		{
			side:     entity.SellAction,
			price:    79,
			quantity: 10,
		},
		{
			side:     entity.SellAction,
			price:    79,
			quantity: 10,
		},
		{
			side:     entity.SellAction,
			price:    78,
			quantity: 10,
		},
		{
			side:     entity.SellAction,
			price:    78,
			quantity: 5,
		},
		{
			side:     entity.SellAction,
			price:    78,
			quantity: 2,
		},
		{
			side:     entity.BuyAction,
			price:    81,
			quantity: 100,
		},
	}
	orders := prepOrders(orderInputs)

	steps := make([]step, 10)
	for i := range 9 {
		steps[i] = step{
			order:      orders[i],
			wantTrades: noTrades,
		}
	}
	// price priority: level 78 first (indices 6,7,8), then 79 (4,5), then 80 (0,1,2,3)
	matchOrder := []int{6, 7, 8, 4, 5, 0, 1, 2, 3}
	wantTrades := make([]Trade, len(matchOrder))
	for i, idx := range matchOrder {
		wantTrades[i] = Trade{
			Ask:      entity.TradeParty{OrderID: orders[idx].ID, UserID: orders[idx].UserID},
			Bid:      entity.TradeParty{OrderID: orders[9].ID, UserID: orders[9].UserID},
			Price:    orderInputs[idx].price,
			Quantity: orderInputs[idx].quantity,
		}
	}
	steps[9] = step{
		order:          orders[9],
		wantTrades:     wantTrades,
		topBidQuantity: new(int64(30)),
	}

	tc := testCase{
		steps: steps,
		finalBookState: finalBookState{
			asksShape: map[int64]int{},
			bidsShape: map[int64]int{orders[9].Price: 1},
			orderIDs:  []uuid.UUID{orders[9].ID},
		},
	}

	testSteps(t, &tc)
}

// Create one big buy order
// Create multiple small sell orders to match big one
// End with two non-matching orders and remaining big one.
func TestBook_AddOrder_RestingBigOrderMatchesMultipleSmall(t *testing.T) {
	orderInputs := []orderInput{
		{
			side:     entity.BuyAction,
			price:    90,
			quantity: 400,
		},
	}

	// create two orders of 10 and 20 quantity across price levels [85, ..., 91]
	for price := 91; price >= 85; price-- {
		orderInputs = append(orderInputs,
			orderInput{
				side:     entity.SellAction,
				price:    int64(price),
				quantity: 10,
			},
			orderInput{
				side:     entity.SellAction,
				price:    int64(price),
				quantity: 20,
			},
		)
	}

	orders := prepOrders(orderInputs)
	steps := make([]step, len(orders))
	// create big buy order
	steps[0] = step{
		order:      orders[0],
		wantTrades: noTrades,
	}
	// first two orders with price 91 - no match
	steps[1], steps[2] = step{
		order:      orders[1],
		wantTrades: noTrades,
	}, step{
		order:      orders[2],
		wantTrades: noTrades,
	}

	bigOrderRemainingQuantity := []int64{390, 370, 360, 340, 330, 310, 300, 280, 270, 250, 240, 220}
	// create steps for matching small orders
	for i := 3; i < len(orders); i++ {
		steps[i] = step{
			order: orders[i],
			wantTrades: []Trade{
				{
					Ask:      entity.TradeParty{OrderID: orders[i].ID, UserID: orders[i].UserID},
					Bid:      entity.TradeParty{OrderID: orders[0].ID, UserID: orders[0].UserID},
					Price:    orders[0].Price,
					Quantity: orders[i].RemainingQuantity,
				},
			},
			topBidQuantity: &bigOrderRemainingQuantity[i-3],
		}
	}

	tc := testCase{
		steps: steps,
		finalBookState: finalBookState{
			asksShape: map[int64]int{orders[1].Price: 2},
			bidsShape: map[int64]int{orders[0].Price: 1},
			orderIDs:  []uuid.UUID{orders[0].ID, orders[1].ID, orders[2].ID},
		},
	}

	testSteps(t, &tc)
}

// Create multiple resting orders on the same price level
// Validate that orders are filled in FIFO order.
func TestBook_AddOrder_FIFO(t *testing.T) {
	const (
		fixedPrice          = 100
		fixedQuantity       = 10
		ordersPerSideNumber = 10
	)

	orderInputs := make([]orderInput, ordersPerSideNumber*2)
	for i := range ordersPerSideNumber {
		orderInputs[i] = orderInput{
			side:     entity.BuyAction,
			price:    fixedPrice,
			quantity: fixedQuantity,
		}
	}
	for i := range ordersPerSideNumber {
		orderInputs[ordersPerSideNumber+i] = orderInput{
			side:     entity.SellAction,
			price:    fixedPrice,
			quantity: fixedQuantity,
		}
	}

	orders := prepOrders(orderInputs)
	steps := make([]step, ordersPerSideNumber*2)

	// buy steps
	for i := range ordersPerSideNumber {
		steps[i] = step{
			order:      orders[i],
			wantTrades: noTrades,
		}
	}
	// sell steps
	for i := range ordersPerSideNumber {
		steps[ordersPerSideNumber+i] = step{
			order: orders[ordersPerSideNumber+i],
			wantTrades: []Trade{
				{
					Ask: entity.TradeParty{
						OrderID: orders[ordersPerSideNumber+i].ID,
						UserID:  orders[ordersPerSideNumber+i].UserID,
					},
					Bid:      entity.TradeParty{OrderID: orders[i].ID, UserID: orders[i].UserID},
					Price:    fixedPrice,
					Quantity: fixedQuantity,
				},
			},
		}
	}

	tc := testCase{
		steps: steps,
		finalBookState: finalBookState{
			asksShape: nil,
			bidsShape: nil,
			orderIDs:  nil,
		},
	}

	testSteps(t, &tc)
}

// Create two orders on the same price level
// Cancel one of the orders
// Check that price level remained.
func TestBook_AddOrder_CancelOrder_PriceLevelSurvives(t *testing.T) {
	const (
		fixedPrice    = 100
		fixedQuantity = 10
	)

	orderInputs := []orderInput{
		{
			side:     entity.SellAction,
			price:    fixedPrice,
			quantity: fixedQuantity,
		},
		{
			side:     entity.SellAction,
			price:    fixedPrice,
			quantity: fixedQuantity,
		},
	}

	orders := prepOrders(orderInputs)
	steps := make([]step, len(orders)+1)
	for i := range orders {
		steps[i] = step{
			order:      orders[i],
			wantTrades: noTrades,
		}
	}
	steps[len(steps)-1] = step{
		cancelID: &orders[0].ID,
	}
	tc := testCase{
		steps: steps,
		finalBookState: finalBookState{
			asksShape: map[int64]int{orders[1].Price: 1},
			bidsShape: nil,
			orderIDs:  []uuid.UUID{orders[1].ID},
		},
	}
	testSteps(t, &tc)
}

// Create two orders on the same price level
// Cancel both of them
// Check that book is clear.
func TestBook_AddOrder_CancelOrder_PriceLevelRemoved(t *testing.T) {
	const (
		fixedPrice    = 100
		fixedQuantity = 10
	)

	orderInputs := []orderInput{
		{
			side:     entity.SellAction,
			price:    fixedPrice,
			quantity: fixedQuantity,
		},
		{
			side:     entity.SellAction,
			price:    fixedPrice,
			quantity: fixedQuantity,
		},
	}

	orders := prepOrders(orderInputs)
	steps := make([]step, len(orders)*2)
	for i := range orders {
		steps[i] = step{
			order:      orders[i],
			wantTrades: noTrades,
		}
	}
	for i := range orders {
		steps[len(orders)+i] = step{
			cancelID: &orders[i].ID,
		}
	}

	tc := testCase{
		steps: steps,
		finalBookState: finalBookState{
			asksShape: nil,
			bidsShape: nil,
			orderIDs:  nil,
		},
	}
	testSteps(t, &tc)
}

// Create multiple buy/sell orders with not matching prices
// Expected not to see any trades as a result.
func TestBook_AddOrder_NonMatchingOrders(t *testing.T) {
	const fixedQuantity = 10

	buyPrices := []int64{200, 201, 205, 210, 211}
	mxBuyPrice := slices.Max(buyPrices) + 1
	ordersPerSideNumber := len(buyPrices) * 2

	orderInputs := make([]orderInput, 0, ordersPerSideNumber*2)
	for _, price := range buyPrices {
		for range 2 {
			orderInputs = append(orderInputs,
				orderInput{
					side:     entity.BuyAction,
					price:    price,
					quantity: fixedQuantity,
				},
				orderInput{
					side:     entity.SellAction,
					price:    mxBuyPrice,
					quantity: fixedQuantity,
				})
		}
	}
	orders := prepOrders(orderInputs)
	steps := make([]step, ordersPerSideNumber*2)
	for i := range steps {
		steps[i] = step{
			order:      orders[i],
			wantTrades: noTrades,
		}
	}
	bidsShape := map[int64]int{}
	for _, price := range buyPrices {
		bidsShape[price] = 2
	}
	orderIDs := make([]uuid.UUID, len(orders))
	for i := range orders {
		orderIDs[i] = orders[i].ID
	}
	tc := testCase{
		steps: steps,
		finalBookState: finalBookState{
			asksShape: map[int64]int{mxBuyPrice: ordersPerSideNumber},
			bidsShape: bidsShape,
			orderIDs:  orderIDs,
		},
	}
	testSteps(t, &tc)
}

func TestBook_CancelOrder_DoubleCancellation(t *testing.T) {
	book := NewBook(&config.OrderBook{B: 8}, testPair)
	id, err := uuid.NewV7()
	require.NoError(t, err)
	order := Order{
		ID:                id,
		OrderType:         LimitOrder,
		Side:              entity.BuyAction,
		Price:             80,
		RemainingQuantity: 10,
	}
	trades, err := book.AddOrder(&order)
	require.NoError(t, err)
	require.Empty(t, trades)

	ok, err := book.CancelOrder(order.ID)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = book.CancelOrder(order.ID)
	require.NoError(t, err)
	assert.False(t, ok)
}

// Main logic which checks order book state after each step and its final state.
func testSteps(t *testing.T, tc *testCase) {
	book := NewBook(&config.OrderBook{B: 8}, testPair)

	for i, step := range tc.steps {
		t.Run(fmt.Sprintf("step %d", i), func(t *testing.T) {
			if step.cancelID != nil {
				ok, err := book.CancelOrder(*step.cancelID)
				require.NoError(t, err)
				require.True(t, ok)
				return
			}
			trades, err := book.AddOrder(step.order)
			require.NoError(t, err)
			assert.Equal(t, step.wantTrades, trades)
			if step.topAskQuantity != nil {
				checkStepTopSideQuantity(t, *book, *step.topAskQuantity, entity.SellAction)
			}
			if step.topBidQuantity != nil {
				checkStepTopSideQuantity(t, *book, *step.topBidQuantity, entity.BuyAction)
			}
		})
	}
	assert.Equal(t, len(tc.finalBookState.asksShape), book.asks.Len())
	assert.Equal(t, len(tc.finalBookState.bidsShape), book.bids.Len())
	assert.ElementsMatch(t, tc.finalBookState.orderIDs, slices.Collect(maps.Keys(book.orders)))

	// validate orders number per price level for asks
	for price, orderCount := range tc.finalBookState.asksShape {
		pl, ok := book.asks.Get(PriceLevel{Price: price})
		require.True(t, ok)
		require.Equal(t, orderCount, pl.Orders.Len())
	}
	// validate orders number per price level for asks
	for price, orderCount := range tc.finalBookState.bidsShape {
		pl, ok := book.bids.Get(PriceLevel{Price: price})
		require.True(t, ok)
		require.Equal(t, orderCount, pl.Orders.Len())
	}
}

// Checks if top order from the appropriate resting side has correct remaining quantity
func checkStepTopSideQuantity(t *testing.T, book Book, quantity int64, side string) {
	restingOrders := book.sameSide(side)
	pl, ok := restingOrders.getBestPriceLevel()
	require.True(t, ok)
	el := pl.Orders.Front()
	require.NotNil(t, el)
	order, ok := el.Value.(*Order)
	require.True(t, ok)
	assert.Equal(t, quantity, order.RemainingQuantity)
}

func prepOrders(orderInput []orderInput) []*Order {
	orders := make([]*Order, len(orderInput))
	for i := range orderInput {
		id, _ := uuid.NewV7()
		orders[i] = &Order{
			ID:                id,
			OrderType:         LimitOrder,
			Side:              orderInput[i].side,
			Price:             orderInput[i].price,
			RemainingQuantity: orderInput[i].quantity,
		}
	}
	return orders
}
