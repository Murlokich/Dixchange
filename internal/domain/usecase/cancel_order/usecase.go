package cancelorder

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type OrderRepository interface {
	GetOrderCancelInfo(ctx context.Context, orderID uuid.UUID) (entity.OrderCancelInfo, error)
	CreateOrderCommand(ctx context.Context, command entity.OrderCommand) error
}

type CommandBus interface {
	SendCancelOrder(ctx context.Context, commandID, orderID uuid.UUID) error
}

type UseCase struct {
	orderRepo  OrderRepository
	commandBus CommandBus
}

// Execute records the cancel request in the command log and sends it to the matching
// engine, which decides whether the order is still there to cancel. Success means the
// request was accepted, not that the order is cancelled.
//
// It returns entity.ErrOrderNotFound or entity.ErrOrderNotOwned for an order the user
// can't cancel, and entity.ErrOrderNotCancellable for their own order that is no longer
// active. The order status is checked only after ownership, so nothing about another
// user's order is revealed.
func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, "input validate")
	}

	info, err := u.orderRepo.GetOrderCancelInfo(ctx, input.OrderID)
	if err != nil {
		return Output{}, errors.Wrap(err, "order repo get order cancel info")
	}
	if info.UserID != input.UserID {
		return Output{}, errors.Wrapf(entity.ErrOrderNotOwned, "order %s is owned by user %s", input.OrderID, info.UserID)
	}
	if !entity.IsActiveOrderStatus(info.Status) {
		return Output{}, errors.Wrapf(entity.ErrOrderNotCancellable, "order %s status %s", input.OrderID, info.Status)
	}

	commandID, err := uuid.NewV7()
	if err != nil {
		return Output{}, errors.Wrap(err, "gen command id")
	}
	command := entity.NewCancelOrderCommand(commandID, input.OrderID)
	if err = u.orderRepo.CreateOrderCommand(ctx, command); err != nil {
		return Output{}, errors.Wrap(err, "order repo create order command")
	}

	if err = u.commandBus.SendCancelOrder(ctx, commandID, input.OrderID); err != nil {
		// TODO: introduce mechanism to reject request if engine is overloaded
		return Output{}, errors.Wrap(err, "command bus send cancel order")
	}

	return Output{}, nil
}

func NewUseCase(orderRepo OrderRepository, commandBus CommandBus) *UseCase {
	return &UseCase{orderRepo: orderRepo, commandBus: commandBus}
}
