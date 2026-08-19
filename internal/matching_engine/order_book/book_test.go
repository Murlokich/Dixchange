package orderbook

import (
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	orderType = "limit"
)

var (
	noTrades = []Trade{}
)

type testCase struct {
	steps          []step
	finalBookState finalBookState
}

type step struct {
	order          *Order
	wantTrades     []Trade
	topAskQuantity *int64 // if nil - don't check it
	topBidQuantity *int64 // if nil - don't check it
}

type finalBookState struct {
	asksLen  int
	bidsLen  int
	orderIDs []uuid.UUID
}

type orderInput struct {
	side     string
	price    int64
	quantity int64
}

// This test inserts 9 sell orders across three different price leves: [78, 79, 80]
// Then appends big limit order with price 81 - having exceeding quantity over all existing orders
// Expected to match all the appended orders, clear the order book and keep single resting buying order
func TestBook_AddOrder_MultiplePriceLevelsConsumption(t *testing.T) {
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
			AskOrderID: orders[idx].ID,
			BidOrderID: orders[9].ID,
			Price:      orderInputs[idx].price,
			Quantity:   orderInputs[idx].quantity,
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
			asksLen:  0,
			bidsLen:  1,
			orderIDs: []uuid.UUID{orders[9].ID},
		},
	}

	testSteps(t, &tc)
}

// Main logic which checks order book state after each step and its final state
func testSteps(t *testing.T, tc *testCase) {
	book := NewBook(8)

	for i, step := range tc.steps {
		t.Run(fmt.Sprintf("step %d", i), func(t *testing.T) {
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
	assert.Equal(t, tc.finalBookState.asksLen, book.asks.Len())
	assert.Equal(t, tc.finalBookState.bidsLen, book.bids.Len())
	assert.Len(t, book.orders, tc.finalBookState.asksLen+tc.finalBookState.bidsLen)
	assert.ElementsMatch(t, tc.finalBookState.orderIDs, slices.Collect(maps.Keys(book.orders)))
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
			OrderType:         orderType,
			Side:              orderInput[i].side,
			Price:             orderInput[i].price,
			RemainingQuantity: orderInput[i].quantity,
		}
	}
	return orders
}
