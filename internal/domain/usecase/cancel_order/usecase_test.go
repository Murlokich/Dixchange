package cancelorder_test

import (
	"context"
	"testing"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	cancelorder "github.com/Murlokich/Dixchange/internal/domain/usecase/cancel_order"
	"github.com/Murlokich/Dixchange/pkg"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

//nolint:gochecknoglobals // acceptable in test
var (
	userID  = uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e33")
	otherID = uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e34")
	orderID = uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e35")
)

// fakeTicket records what the usecase does with its access ticket. mockery only covers
// the usecase packages, and the ticket interface lives in pkg.
type fakeTicket struct {
	sendErr    error
	releaseErr error

	sends         int
	releases      int
	sentCommandID uuid.UUID
	sentOrderID   uuid.UUID
}

var _ pkg.AccessTicket = (*fakeTicket)(nil)

func (f *fakeTicket) SendPlaceOrder(uuid.UUID, entity.Order) error {
	return errors.New("cancel order must not send a place")
}

func (f *fakeTicket) SendCancelOrder(commandID, orderID uuid.UUID) error {
	f.sends++
	f.sentCommandID = commandID
	f.sentOrderID = orderID
	return f.sendErr
}

func (f *fakeTicket) ReleaseWriteAccess() error {
	f.releases++
	return f.releaseErr
}

// expectActiveOwnedOrder makes the order pass the ownership and status checks.
func expectActiveOwnedOrder(orderRepo *cancelorder.MockOrderRepository) {
	orderRepo.EXPECT().GetOrderCancelInfo(mock.Anything, orderID).
		Return(entity.OrderCancelInfo{UserID: userID, Status: entity.StatusOpen}, nil).Once()
}

// isCancelCommand matches the cancel command the use case logs for orderID.
func isCancelCommand(command entity.OrderCommand) bool {
	return command.ID != uuid.Nil &&
		command.Type == entity.OrderCommandCancel &&
		command.OrderID == orderID &&
		command.OrderDetails == nil
}

func TestUseCase_Execute_SendsCancelForActiveOwnedOrder(t *testing.T) {
	t.Parallel()

	for _, status := range []string{entity.StatusOpen, entity.StatusPartiallyFilled} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()

			orderRepo := cancelorder.NewMockOrderRepository(t)
			commandBus := cancelorder.NewMockCommandBus(t)

			orderRepo.EXPECT().GetOrderCancelInfo(mock.Anything, orderID).
				Return(entity.OrderCancelInfo{UserID: userID, Status: status}, nil).Once()

			ticket := &fakeTicket{}
			commandBus.EXPECT().GetWriteAccess(mock.Anything).Return(ticket, nil).Once()

			var loggedCommandID uuid.UUID
			orderRepo.EXPECT().CreateOrderCommand(mock.Anything, mock.MatchedBy(isCancelCommand)).
				Run(func(_ context.Context, command entity.OrderCommand) {
					loggedCommandID = command.ID
				}).
				Return(nil).Once()

			uc := cancelorder.NewUseCase(orderRepo, commandBus)
			_, err := uc.Execute(t.Context(), cancelorder.Input{UserID: userID, OrderID: orderID})

			require.NoError(t, err)
			// the same command ID is logged and sent, so settlement can find the command
			require.Equal(t, 1, ticket.sends)
			require.Equal(t, loggedCommandID, ticket.sentCommandID)
			require.Equal(t, orderID, ticket.sentOrderID)
			// the release is deferred unconditionally; the real ticket ignores it once sent
			require.Equal(t, 1, ticket.releases)
		})
	}
}

func TestUseCase_Execute_RejectsWithoutSending(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		info    entity.OrderCancelInfo
		repoErr error
		wantErr error
	}{
		{
			name:    "order not found",
			repoErr: errors.Wrap(entity.ErrOrderNotFound, "order"),
			wantErr: entity.ErrOrderNotFound,
		},
		{
			name:    "active order of another user",
			info:    entity.OrderCancelInfo{UserID: otherID, Status: entity.StatusOpen},
			wantErr: entity.ErrOrderNotOwned,
		},
		{
			// the status must not leak through a different error
			name:    "inactive order of another user",
			info:    entity.OrderCancelInfo{UserID: otherID, Status: entity.StatusFilled},
			wantErr: entity.ErrOrderNotOwned,
		},
		{
			name:    "own filled order",
			info:    entity.OrderCancelInfo{UserID: userID, Status: entity.StatusFilled},
			wantErr: entity.ErrOrderNotCancellable,
		},
		{
			name:    "own cancelled order",
			info:    entity.OrderCancelInfo{UserID: userID, Status: entity.StatusCancelled},
			wantErr: entity.ErrOrderNotCancellable,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderRepo := cancelorder.NewMockOrderRepository(t)
			// no expectations: a rejected cancel takes no slot and doesn't reach the engine
			commandBus := cancelorder.NewMockCommandBus(t)

			orderRepo.EXPECT().GetOrderCancelInfo(mock.Anything, orderID).Return(tc.info, tc.repoErr).Once()

			uc := cancelorder.NewUseCase(orderRepo, commandBus)
			_, err := uc.Execute(t.Context(), cancelorder.Input{UserID: userID, OrderID: orderID})

			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		input cancelorder.Input
	}{
		{name: "empty user id", input: cancelorder.Input{OrderID: orderID}},
		{name: "empty order id", input: cancelorder.Input{UserID: userID}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			uc := cancelorder.NewUseCase(
				cancelorder.NewMockOrderRepository(t),
				cancelorder.NewMockCommandBus(t),
			)
			_, err := uc.Execute(t.Context(), tc.input)

			require.ErrorAs(t, err, &entity.ValidationError{})
		})
	}
}

// An overloaded bus rejects the cancel before it is written to the command log.
func TestUseCase_Execute_NoWriteAccess(t *testing.T) {
	t.Parallel()

	// no CreateOrderCommand expectation: the mock fails the test if the cancel is logged
	orderRepo := cancelorder.NewMockOrderRepository(t)
	commandBus := cancelorder.NewMockCommandBus(t)

	expectActiveOwnedOrder(orderRepo)
	commandBus.EXPECT().GetWriteAccess(mock.Anything).Return(nil, entity.ErrCommandBusOverloaded).Once()

	uc := cancelorder.NewUseCase(orderRepo, commandBus)
	_, err := uc.Execute(t.Context(), cancelorder.Input{UserID: userID, OrderID: orderID})

	require.ErrorIs(t, err, entity.ErrCommandBusOverloaded)
}

func TestUseCase_Execute_SendFailure(t *testing.T) {
	t.Parallel()

	orderRepo := cancelorder.NewMockOrderRepository(t)
	commandBus := cancelorder.NewMockCommandBus(t)

	sendErr := errors.New("queue full")
	expectActiveOwnedOrder(orderRepo)
	ticket := &fakeTicket{sendErr: sendErr}
	commandBus.EXPECT().GetWriteAccess(mock.Anything).Return(ticket, nil).Once()
	orderRepo.EXPECT().CreateOrderCommand(mock.Anything, mock.MatchedBy(isCancelCommand)).Return(nil).Once()

	uc := cancelorder.NewUseCase(orderRepo, commandBus)
	_, err := uc.Execute(t.Context(), cancelorder.Input{UserID: userID, OrderID: orderID})

	require.ErrorIs(t, err, sendErr)
	require.Equal(t, 1, ticket.releases)
}

// A cancel that isn't in the command log must not reach the engine, and gives its slot
// back.
func TestUseCase_Execute_CommandLogFailure(t *testing.T) {
	t.Parallel()

	orderRepo := cancelorder.NewMockOrderRepository(t)
	commandBus := cancelorder.NewMockCommandBus(t)

	logErr := errors.New("db down")
	expectActiveOwnedOrder(orderRepo)
	ticket := &fakeTicket{}
	commandBus.EXPECT().GetWriteAccess(mock.Anything).Return(ticket, nil).Once()
	orderRepo.EXPECT().CreateOrderCommand(mock.Anything, mock.MatchedBy(isCancelCommand)).Return(logErr).Once()

	uc := cancelorder.NewUseCase(orderRepo, commandBus)
	_, err := uc.Execute(t.Context(), cancelorder.Input{UserID: userID, OrderID: orderID})

	require.ErrorIs(t, err, logErr)
	require.Equal(t, 0, ticket.sends)
	require.Equal(t, 1, ticket.releases)
}

// A failed release is joined to the error that made the usecase release, so neither is
// lost.
func TestUseCase_Execute_ReleaseErrorIsJoined(t *testing.T) {
	t.Parallel()

	orderRepo := cancelorder.NewMockOrderRepository(t)
	commandBus := cancelorder.NewMockCommandBus(t)

	logErr := errors.New("db down")
	releaseErr := errors.New("release failed")
	expectActiveOwnedOrder(orderRepo)
	commandBus.EXPECT().GetWriteAccess(mock.Anything).Return(&fakeTicket{releaseErr: releaseErr}, nil).Once()
	orderRepo.EXPECT().CreateOrderCommand(mock.Anything, mock.MatchedBy(isCancelCommand)).Return(logErr).Once()

	uc := cancelorder.NewUseCase(orderRepo, commandBus)
	_, err := uc.Execute(t.Context(), cancelorder.Input{UserID: userID, OrderID: orderID})

	require.ErrorIs(t, err, logErr)
	require.ErrorIs(t, err, releaseErr)
}
