package createorder

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order entity.Order) error
}

type WalletRepository interface {
	ReserveFunds(ctx context.Context, userID uuid.UUID, symbolID int, amount decimal.Decimal) error
}

type MarketRepository interface {
	GetPair(id int) entity.Pair
	GetSymbol(id int) entity.Symbol
	PairExists(id int) bool
}

type TXManager interface {
	WithTX(ctx context.Context, fn func(ctx context.Context) error) (err error)
}

type UseCase struct {
	orderRepo  OrderRepository
	walletRepo WalletRepository
	marketRepo MarketRepository
	txManager  TXManager
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, "input validate")
	}

	order, err := u.inputToOrder(input)
	if err != nil {
		return Output{}, errors.Wrap(err, "input to order")
	}

	symbol, err := order.GetDebitAsset()
	if err != nil {
		return Output{}, errors.Wrap(err, "get debit asset")
	}

	fn := func(ctx context.Context) error {
		if err = u.walletRepo.ReserveFunds(ctx, order.UserID, symbol.ID, order.GetDebitAmount()); err != nil {
			return errors.Wrap(err, "wallet repo reserve funds")
		}
		if err = u.orderRepo.CreateOrder(ctx, order); err != nil {
			return errors.Wrap(err, "repo create order")
		}
		return nil
	}

	if err = u.txManager.WithTX(ctx, fn); err != nil {
		return Output{}, errors.Wrap(err, "tx manager with tx")
	}
	return Output{}, nil
}

func (u *UseCase) inputToOrder(input Input) (entity.Order, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return entity.Order{}, errors.Wrap(err, "gen order id")
	}
	if !u.marketRepo.PairExists(input.MarketPairID) {
		return entity.Order{}, entity.ValidationError{Err: errors.New("market repo pair doesn't exist")}
	}

	return entity.Order{
		ID:              id,
		UserID:          input.UserID,
		MarketPair:      u.marketRepo.GetPair(input.MarketPairID),
		Action:          input.Action,
		QuantityOrdered: input.QuantityOrdered,
		PricePerUnit:    input.PricePerUnit,
		Status:          entity.StatusOpen,
	}, nil
}

func NewUseCase(
	repo OrderRepository,
	walletRepo WalletRepository,
	marketRepo MarketRepository,
	txManager TXManager,
) *UseCase {
	return &UseCase{
		orderRepo:  repo,
		walletRepo: walletRepo,
		marketRepo: marketRepo,
		txManager:  txManager,
	}
}
