package getpairbalance_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	getpairbalance "github.com/dixchange/Monolith/internal/domain/usecase/get_pair_balance"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	validUserID = "019f7f2c-0caa-70ab-b1ec-414f744c3e33"
	validPairID = 1
)

//nolint:gochecknoglobals // acceptable in test
var (
	validUUID = uuid.MustParse(validUserID)
	validPair = entity.Pair{
		ID:    validPairID,
		Base:  entity.Symbol{ID: 1, Code: "USD", Precision: 2},
		Quote: entity.Symbol{ID: 2, Code: "KCN", Precision: 0},
	}
)

func TestUseCase_Execute_HappyPath(t *testing.T) {
	t.Parallel()

	base := entity.Balance{SymbolID: validPair.Base.ID, Main: 1056, Reserved: 44}
	quote := entity.Balance{SymbolID: validPair.Quote.ID, Main: 7, Reserved: 3}

	walletRepo := getpairbalance.NewMockWalletRepository(t)
	marketRepo := getpairbalance.NewMockMarketRepository(t)

	marketRepo.EXPECT().GetPair(validPairID).Return(validPair, true).Once()
	// the repository gives no order guarantee, so quote is returned first on purpose
	walletRepo.EXPECT().GetPairBalances(mock.Anything, validUUID, validPair.Base.ID, validPair.Quote.ID).
		Return([]entity.Balance{quote, base}, nil).Once()

	uc := getpairbalance.NewUseCase(walletRepo, marketRepo)
	output, err := uc.Execute(t.Context(), getpairbalance.Input{UserID: validUUID, PairID: validPairID})

	require.NoError(t, err)
	require.Equal(t, getpairbalance.Output{Pair: validPair, Base: base, Quote: quote}, output)
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		userID uuid.UUID
		pairID int
	}{
		{name: "nil user id", userID: uuid.Nil, pairID: validPairID},
		{name: "negative pair id", userID: validUUID, pairID: -1},
		{name: "zero pair id", userID: validUUID, pairID: 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// no expectations set up: the mocks panic if the usecase
			// calls them, proving Validate short-circuits Execute.
			walletRepo := getpairbalance.NewMockWalletRepository(t)
			marketRepo := getpairbalance.NewMockMarketRepository(t)

			uc := getpairbalance.NewUseCase(walletRepo, marketRepo)
			_, err := uc.Execute(t.Context(), getpairbalance.Input{UserID: tc.userID, PairID: tc.pairID})

			var valErr entity.ValidationError
			require.ErrorAs(t, err, &valErr)
		})
	}
}

func TestUseCase_Execute_PairDoesNotExist(t *testing.T) {
	t.Parallel()

	walletRepo := getpairbalance.NewMockWalletRepository(t)
	marketRepo := getpairbalance.NewMockMarketRepository(t)

	marketRepo.EXPECT().GetPair(validPairID).Return(entity.Pair{}, false).Once()

	uc := getpairbalance.NewUseCase(walletRepo, marketRepo)
	_, err := uc.Execute(t.Context(), getpairbalance.Input{UserID: validUUID, PairID: validPairID})

	var valErr entity.ValidationError
	require.ErrorAs(t, err, &valErr)
	require.ErrorContains(t, err, "market repo pair doesn't exist")
}

func TestUseCase_Execute_BalanceRecordMissing(t *testing.T) {
	t.Parallel()

	base := entity.Balance{SymbolID: validPair.Base.ID}
	quote := entity.Balance{SymbolID: validPair.Quote.ID}

	testCases := []struct {
		name     string
		balances []entity.Balance
	}{
		{name: "no records", balances: []entity.Balance{}},
		{name: "base missing", balances: []entity.Balance{quote}},
		{name: "quote missing", balances: []entity.Balance{base}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			walletRepo := getpairbalance.NewMockWalletRepository(t)
			marketRepo := getpairbalance.NewMockMarketRepository(t)

			marketRepo.EXPECT().GetPair(validPairID).Return(validPair, true).Once()
			walletRepo.EXPECT().GetPairBalances(mock.Anything, validUUID, mock.Anything, mock.Anything).
				Return(tc.balances, nil).Once()

			uc := getpairbalance.NewUseCase(walletRepo, marketRepo)
			_, err := uc.Execute(t.Context(), getpairbalance.Input{UserID: validUUID, PairID: validPairID})

			require.ErrorIs(t, err, entity.ErrBalanceNotFound)
		})
	}
}

func TestUseCase_Execute_RepositoryError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("connection refused")

	walletRepo := getpairbalance.NewMockWalletRepository(t)
	marketRepo := getpairbalance.NewMockMarketRepository(t)

	marketRepo.EXPECT().GetPair(validPairID).Return(validPair, true).Once()
	walletRepo.EXPECT().GetPairBalances(mock.Anything, validUUID, mock.Anything, mock.Anything).Return(nil, repoErr).Once()

	uc := getpairbalance.NewUseCase(walletRepo, marketRepo)
	_, err := uc.Execute(t.Context(), getpairbalance.Input{UserID: validUUID, PairID: validPairID})

	require.ErrorIs(t, err, repoErr)
	require.NotErrorIs(t, err, entity.ErrBalanceNotFound)
}
