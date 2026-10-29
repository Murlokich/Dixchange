package createorder_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	createorder "github.com/Murlokich/Dixchange/internal/domain/usecase/create_order"
	"github.com/Murlokich/Dixchange/pkg"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// fakeTicket records what the usecase does with its access ticket. mockery only covers
// the usecase packages, and the ticket interface lives in pkg.
type fakeTicket struct {
	sendErr    error
	releaseErr error

	sends         int
	releases      int
	sentCommandID uuid.UUID
	sentOrder     entity.Order
}

var _ pkg.AccessTicket = (*fakeTicket)(nil)

func (f *fakeTicket) SendPlaceOrder(commandID uuid.UUID, order entity.Order) error {
	f.sends++
	f.sentCommandID = commandID
	f.sentOrder = order
	return f.sendErr
}

func (f *fakeTicket) SendCancelOrder(uuid.UUID, uuid.UUID) error {
	return errors.New("create order must not send a cancel")
}

func (f *fakeTicket) ReleaseWriteAccess() error {
	f.releases++
	return f.releaseErr
}

func validInput() createorder.Input {
	return createorder.Input{
		UserID:          validUUID,
		MarketPairID:    validMarketPairID,
		Side:            entity.BuySide,
		QuantityOrdered: validQuantityDecimal,
		PricePerUnit:    validPriceDecimal,
	}
}

func testPair() entity.Pair {
	return entity.Pair{
		ID:             validMarketPairID,
		Base:           entity.Symbol{ID: 1, Code: "KCN", Precision: 0},
		Quote:          entity.Symbol{ID: 2, Code: "USD", Precision: 2},
		PricePrecision: 2,
	}
}

// expectOrderDerived sets up the pair lookup and the conversion, which every test that
// gets past validation goes through.
func expectOrderDerived(marketRepo *createorder.MockMarketRepository, converter *createorder.MockConverter) {
	marketRepo.EXPECT().GetPair(validMarketPairID).Return(testPair(), true).Once()
	converter.EXPECT().
		DeriveOrderWithQuantitativeFields(mock.Anything, validQuantityDecimal, validPriceDecimal).
		RunAndReturn(func(order entity.Order, _ decimal.Decimal, _ decimal.Decimal) (entity.Order, error) {
			order.QuantityOrdered = validQuantityOrdered
			order.PricePerUnit = validPricePerUnit
			return order, nil
		}).
		Once()
}

func TestUseCase_Execute_HappyPath(t *testing.T) {
	t.Parallel()

	orderRepo := createorder.NewMockOrderRepository(t)
	walletRepo := createorder.NewMockWalletRepository(t)
	marketRepo := createorder.NewMockMarketRepository(t)
	txManager := createorder.NewMockTXManager(t)
	converter := createorder.NewMockConverter(t)
	commandBus := createorder.NewMockCommandBus(t)

	pair := entity.Pair{
		ID:             validMarketPairID,
		Base:           entity.Symbol{ID: 1, Code: "KCN", Precision: 0},
		Quote:          entity.Symbol{ID: 2, Code: "USD", Precision: 2},
		PricePrecision: 2,
	}
	// buy debits the quote asset; base precision + price precision == quote precision,
	// so the notional is plain price times quantity
	wantTotalPrice := validPricePerUnit * validQuantityOrdered

	marketRepo.EXPECT().GetPair(validMarketPairID).Return(pair, true).Once()

	// the conversion math itself is covered by the converter's own tests; here we just
	// pass the quantitative fields through so the rest of the usecase can be exercised.
	converter.EXPECT().
		DeriveOrderWithQuantitativeFields(mock.Anything, validQuantityDecimal, validPriceDecimal).
		RunAndReturn(func(order entity.Order, _ decimal.Decimal, _ decimal.Decimal) (entity.Order, error) {
			order.QuantityOrdered = validQuantityOrdered
			order.PricePerUnit = validPricePerUnit
			return order, nil
		}).
		Once()

	walletRepo.EXPECT().
		ReserveFunds(mock.Anything, validUUID, pair.Quote.ID, wantTotalPrice).
		Return(nil).
		Once()

	matchesExpectedOrder := func(order entity.Order) bool {
		return order.ID != uuid.Nil &&
			order.UserID == validUUID &&
			order.MarketPair == pair &&
			order.Side == entity.BuySide &&
			order.QuantityOrdered == validQuantityOrdered &&
			order.PricePerUnit == validPricePerUnit &&
			order.Status == entity.StatusOpen
	}

	// the order goes to the command log, not to the orders
	var loggedCommandID uuid.UUID
	orderRepo.EXPECT().
		CreateOrderCommand(mock.Anything, mock.MatchedBy(func(command entity.OrderCommand) bool {
			return command.ID != uuid.Nil &&
				command.Type == entity.OrderCommandCreate &&
				command.OrderID != uuid.Nil &&
				command.OrderDetails != nil &&
				command.OrderDetails.UserID == validUUID &&
				command.OrderDetails.PairID == validMarketPairID &&
				command.OrderDetails.Side == entity.BuySide &&
				command.OrderDetails.PricePerUnit == validPricePerUnit &&
				command.OrderDetails.Quantity == validQuantityOrdered
		})).
		Run(func(_ context.Context, command entity.OrderCommand) {
			loggedCommandID = command.ID
		}).
		Return(nil).
		Once()

	ticket := &fakeTicket{}
	commandBus.EXPECT().GetWriteAccess(mock.Anything).Return(ticket, nil).Once()

	txManager.EXPECT().
		WithTX(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).
		Once()

	uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager, converter, commandBus)
	output, err := uc.Execute(t.Context(), createorder.Input{
		UserID:          validUUID,
		MarketPairID:    validMarketPairID,
		Side:            entity.BuySide,
		QuantityOrdered: validQuantityDecimal,
		PricePerUnit:    validPriceDecimal,
	})

	require.NoError(t, err)
	require.Equal(t, createorder.Output{}, output)

	// the same command ID is sent to the engine, so settlement can find the command
	require.Equal(t, 1, ticket.sends)
	require.Equal(t, loggedCommandID, ticket.sentCommandID)
	require.True(t, matchesExpectedOrder(ticket.sentOrder))
	// the release is deferred unconditionally; the real ticket ignores it once sent
	require.Equal(t, 1, ticket.releases)
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		userID uuid.UUID
		side   string
	}{
		{name: "nil user id", userID: uuid.Nil, side: entity.BuySide},
		{name: "invalid side", userID: validUUID, side: "gibberish"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// no expectations set up: the mocks panic if the usecase
			// calls them, proving Validate short-circuits Execute.
			orderRepo := createorder.NewMockOrderRepository(t)
			walletRepo := createorder.NewMockWalletRepository(t)
			marketRepo := createorder.NewMockMarketRepository(t)
			txManager := createorder.NewMockTXManager(t)
			converter := createorder.NewMockConverter(t)
			commandBus := createorder.NewMockCommandBus(t)

			uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager, converter, commandBus)
			_, err := uc.Execute(t.Context(), createorder.Input{
				UserID:          tc.userID,
				MarketPairID:    validMarketPairID,
				Side:            tc.side,
				QuantityOrdered: validQuantityDecimal,
				PricePerUnit:    validPriceDecimal,
			})

			var valErr entity.ValidationError
			require.ErrorAs(t, err, &valErr)
		})
	}
}

func TestUseCase_Execute_PairDoesNotExist(t *testing.T) {
	t.Parallel()

	orderRepo := createorder.NewMockOrderRepository(t)
	walletRepo := createorder.NewMockWalletRepository(t)
	marketRepo := createorder.NewMockMarketRepository(t)
	txManager := createorder.NewMockTXManager(t)
	converter := createorder.NewMockConverter(t)
	commandBus := createorder.NewMockCommandBus(t)

	marketRepo.EXPECT().GetPair(validMarketPairID).Return(entity.Pair{}, false).Once()

	uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager, converter, commandBus)
	_, err := uc.Execute(t.Context(), createorder.Input{
		UserID:          validUUID,
		MarketPairID:    validMarketPairID,
		Side:            entity.BuySide,
		QuantityOrdered: validQuantityDecimal,
		PricePerUnit:    validPriceDecimal,
	})

	var valErr entity.ValidationError
	require.ErrorAs(t, err, &valErr)
	require.ErrorContains(t, err, "market repo pair doesn't exist")
}

// An overloaded bus rejects the request before anything is written.
func TestUseCase_Execute_NoWriteAccess(t *testing.T) {
	t.Parallel()

	orderRepo := createorder.NewMockOrderRepository(t)
	walletRepo := createorder.NewMockWalletRepository(t)
	marketRepo := createorder.NewMockMarketRepository(t)
	txManager := createorder.NewMockTXManager(t)
	converter := createorder.NewMockConverter(t)
	commandBus := createorder.NewMockCommandBus(t)

	expectOrderDerived(marketRepo, converter)
	commandBus.EXPECT().GetWriteAccess(mock.Anything).Return(nil, entity.ErrCommandBusOverloaded).Once()

	// no tx expectation: the mocks fail the test if the usecase writes anything
	uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager, converter, commandBus)
	_, err := uc.Execute(t.Context(), validInput())

	require.ErrorIs(t, err, entity.ErrCommandBusOverloaded)
}

// A failed transaction sends nothing and gives the slot back.
func TestUseCase_Execute_TxFailureReleasesSlot(t *testing.T) {
	t.Parallel()

	orderRepo := createorder.NewMockOrderRepository(t)
	walletRepo := createorder.NewMockWalletRepository(t)
	marketRepo := createorder.NewMockMarketRepository(t)
	txManager := createorder.NewMockTXManager(t)
	converter := createorder.NewMockConverter(t)
	commandBus := createorder.NewMockCommandBus(t)

	expectOrderDerived(marketRepo, converter)
	ticket := &fakeTicket{}
	commandBus.EXPECT().GetWriteAccess(mock.Anything).Return(ticket, nil).Once()
	txManager.EXPECT().WithTX(mock.Anything, mock.Anything).Return(entity.ErrInsufficientBalance).Once()

	uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager, converter, commandBus)
	_, err := uc.Execute(t.Context(), validInput())

	require.ErrorIs(t, err, entity.ErrInsufficientBalance)
	require.Equal(t, 0, ticket.sends)
	require.Equal(t, 1, ticket.releases)
}

// A failed release is joined to the error that made the usecase release, so neither is
// lost.
func TestUseCase_Execute_ReleaseErrorIsJoined(t *testing.T) {
	t.Parallel()

	orderRepo := createorder.NewMockOrderRepository(t)
	walletRepo := createorder.NewMockWalletRepository(t)
	marketRepo := createorder.NewMockMarketRepository(t)
	txManager := createorder.NewMockTXManager(t)
	converter := createorder.NewMockConverter(t)
	commandBus := createorder.NewMockCommandBus(t)

	releaseErr := errors.New("release failed")
	expectOrderDerived(marketRepo, converter)
	ticket := &fakeTicket{releaseErr: releaseErr}
	commandBus.EXPECT().GetWriteAccess(mock.Anything).Return(ticket, nil).Once()
	txManager.EXPECT().WithTX(mock.Anything, mock.Anything).Return(entity.ErrInsufficientBalance).Once()

	uc := createorder.NewUseCase(orderRepo, walletRepo, marketRepo, txManager, converter, commandBus)
	_, err := uc.Execute(t.Context(), validInput())

	require.ErrorIs(t, err, entity.ErrInsufficientBalance)
	require.ErrorIs(t, err, releaseErr)
}
