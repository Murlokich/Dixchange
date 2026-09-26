package getorders

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type OrderRepository interface {
	GetUserOrdersByStatus(ctx context.Context, userID uuid.UUID, status string) ([]entity.Order, error)
}

type MarketRepository interface {
	GetPair(id int) (entity.Pair, bool)
}

type UseCase struct {
	orderRepo  OrderRepository
	marketRepo MarketRepository
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, "input validate")
	}

	orders, err := u.orderRepo.GetUserOrdersByStatus(ctx, input.UserID, input.Status)
	if err != nil {
		return Output{}, errors.Wrap(err, "order repo get user orders by status")
	}

	// the repository returns only the pair ID, the rest of the pair comes from the market
	for i := range orders {
		pair, ok := u.marketRepo.GetPair(orders[i].MarketPair.ID)
		if !ok {
			return Output{}, errors.Errorf("market repo pair %d of order %s doesn't exist",
				orders[i].MarketPair.ID, orders[i].ID)
		}
		orders[i].MarketPair = pair
	}

	return Output{Orders: orders}, nil
}

func NewUseCase(orderRepo OrderRepository, marketRepo MarketRepository) *UseCase {
	return &UseCase{orderRepo: orderRepo, marketRepo: marketRepo}
}
