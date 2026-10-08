package createorder

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type OrderRepository interface {
	CreateOrderCommand(ctx context.Context, command entity.OrderCommand) error
}

type WalletRepository interface {
	ReserveFunds(ctx context.Context, userID uuid.UUID, symbolID int, amount int64) error
}

type MarketRepository interface {
	GetPair(id int) (entity.Pair, bool)
	GetSymbol(id int) (entity.Symbol, bool)
}

type TXManager interface {
	WithTX(ctx context.Context, fn func(ctx context.Context) error) (err error)
}

type Converter interface {
	DeriveOrderWithQuantitativeFields(
		order entity.Order,
		quantity decimal.Decimal,
		price decimal.Decimal,
	) (entity.Order, error)
}

type CommandBus interface {
	SendPlaceOrder(ctx context.Context, commandID uuid.UUID, entityOrder entity.Order) error
}

type UseCase struct {
	orderRepo  OrderRepository
	walletRepo WalletRepository
	marketRepo MarketRepository
	txManager  TXManager
	converter  Converter
	commandBus CommandBus
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

	debitAmount, err := order.GetDebitOrderedAmount()
	if err != nil {
		return Output{}, errors.Wrap(err, "get debit amount")
	}

	// The order reaches the orders only once the engine has placed it: until then it is
	// a create command in the command log, with its funds already reserved.
	commandID, err := uuid.NewV7()
	if err != nil {
		return Output{}, errors.Wrap(err, "gen command id")
	}
	command := entity.NewCreateOrderCommand(commandID, order)

	fn := func(ctx context.Context) error {
		if err = u.walletRepo.ReserveFunds(ctx, order.UserID, symbol.ID, debitAmount); err != nil {
			return errors.Wrap(err, "wallet repo reserve funds")
		}
		if err = u.orderRepo.CreateOrderCommand(ctx, command); err != nil {
			return errors.Wrap(err, "repo create order command")
		}
		return nil
	}

	if err = u.txManager.WithTX(ctx, fn); err != nil {
		return Output{}, errors.Wrap(err, "tx manager with tx")
	}

	if err = u.commandBus.SendPlaceOrder(ctx, commandID, order); err != nil {
		// TODO: introduce mechanism to reject request if engine is overloaded
		return Output{}, errors.Wrap(err, "command bus send place order")
	}

	return Output{}, nil
}

func (u *UseCase) inputToOrder(input Input) (entity.Order, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return entity.Order{}, errors.Wrap(err, "gen order id")
	}
	pair, ok := u.marketRepo.GetPair(input.MarketPairID)
	if !ok {
		return entity.Order{}, entity.ValidationError{Err: errors.New("market repo pair doesn't exist")}
	}

	orderWithoutQuantitativeFields := entity.Order{
		ID:         id,
		UserID:     input.UserID,
		MarketPair: pair,
		Side:       input.Side,
		Status:     entity.StatusOpen,
	}

	fullOrder, err := u.converter.DeriveOrderWithQuantitativeFields(
		orderWithoutQuantitativeFields,
		input.QuantityOrdered,
		input.PricePerUnit,
	)
	if err != nil {
		return entity.Order{}, errors.Wrap(err, "converter DeriveOrderWithQuantitativeFields")
	}

	return fullOrder, nil
}

func NewUseCase(
	repo OrderRepository,
	walletRepo WalletRepository,
	marketRepo MarketRepository,
	txManager TXManager,
	converter Converter,
	commandBus CommandBus,
) *UseCase {
	return &UseCase{
		orderRepo:  repo,
		walletRepo: walletRepo,
		marketRepo: marketRepo,
		txManager:  txManager,
		converter:  converter,
		commandBus: commandBus,
	}
}
