package deposit

import (
	"context"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

type WalletRepository interface {
	Deposit(ctx context.Context, userID uuid.UUID, symbolID int, amount int64) error
}

type MarketRepository interface {
	GetSymbol(id int) (entity.Symbol, bool)
}

type Converter interface {
	ConvertAmountToInt(symbol entity.Symbol, amount decimal.Decimal) (int64, error)
}

type UseCase struct {
	walletRepo WalletRepository
	marketRepo MarketRepository
	converter  Converter
}

func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, "input validate")
	}

	symbol, ok := u.marketRepo.GetSymbol(input.SymbolID)

	if !ok {
		return Output{}, entity.ValidationError{Err: errors.New("market repo symbol doesn't exist")}
	}

	amountInt, err := u.converter.ConvertAmountToInt(symbol, input.Amount)
	if err != nil {
		return Output{}, errors.Wrap(err, "converter ConvertAmountToInt")
	}

	if err = u.walletRepo.Deposit(ctx, input.UserID, input.SymbolID, amountInt); err != nil {
		return Output{}, errors.Wrap(err, "wallet repo deposit")
	}
	return Output{}, nil
}

func NewUseCase(walletRepo WalletRepository, marketRepo MarketRepository, converter Converter) *UseCase {
	return &UseCase{walletRepo: walletRepo, marketRepo: marketRepo, converter: converter}
}
