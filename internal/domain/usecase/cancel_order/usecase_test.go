package cancelorder_test

import (
	"context"
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	cancelorder "github.com/dixchange/Monolith/internal/domain/usecase/cancel_order"
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

// isCancelCommand matches the cancel command the use case logs for orderID.
func isCancelCommand(command entity.OrderCommand) bool {
	return command.ID != uuid.Nil &&
		command.Type == entity.OrderCommandCancel &&
		command.OrderID == orderID &&
		command.Place == nil
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

			// the same command ID is logged and sent, so settlement can find the command
			var loggedCommandID uuid.UUID
			orderRepo.EXPECT().CreateOrderCommand(mock.Anything, mock.MatchedBy(isCancelCommand)).
				Run(func(_ context.Context, command entity.OrderCommand) {
					loggedCommandID = command.ID
				}).
				Return(nil).Once()
			commandBus.EXPECT().SendCancelOrder(mock.Anything, mock.Anything, orderID).
				Run(func(_ context.Context, commandID, _ uuid.UUID) {
					require.Equal(t, loggedCommandID, commandID)
				}).
				Return(nil).Once()

			uc := cancelorder.NewUseCase(orderRepo, commandBus)
			_, err := uc.Execute(t.Context(), cancelorder.Input{UserID: userID, OrderID: orderID})

			require.NoError(t, err)
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
			// no expectations: the cancel must not reach the engine
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

func TestUseCase_Execute_SendFailure(t *testing.T) {
	t.Parallel()

	orderRepo := cancelorder.NewMockOrderRepository(t)
	commandBus := cancelorder.NewMockCommandBus(t)

	sendErr := errors.New("queue full")
	orderRepo.EXPECT().GetOrderCancelInfo(mock.Anything, orderID).
		Return(entity.OrderCancelInfo{UserID: userID, Status: entity.StatusOpen}, nil).Once()
	orderRepo.EXPECT().CreateOrderCommand(mock.Anything, mock.MatchedBy(isCancelCommand)).Return(nil).Once()
	commandBus.EXPECT().SendCancelOrder(mock.Anything, mock.Anything, orderID).Return(sendErr).Once()

	uc := cancelorder.NewUseCase(orderRepo, commandBus)
	_, err := uc.Execute(t.Context(), cancelorder.Input{UserID: userID, OrderID: orderID})

	require.ErrorIs(t, err, sendErr)
}

func TestUseCase_Execute_CommandLogFailure(t *testing.T) {
	t.Parallel()

	orderRepo := cancelorder.NewMockOrderRepository(t)
	// no expectations: a cancel that isn't in the command log must not reach the engine
	commandBus := cancelorder.NewMockCommandBus(t)

	logErr := errors.New("db down")
	orderRepo.EXPECT().GetOrderCancelInfo(mock.Anything, orderID).
		Return(entity.OrderCancelInfo{UserID: userID, Status: entity.StatusOpen}, nil).Once()
	orderRepo.EXPECT().CreateOrderCommand(mock.Anything, mock.MatchedBy(isCancelCommand)).Return(logErr).Once()

	uc := cancelorder.NewUseCase(orderRepo, commandBus)
	_, err := uc.Execute(t.Context(), cancelorder.Input{UserID: userID, OrderID: orderID})

	require.ErrorIs(t, err, logErr)
}
