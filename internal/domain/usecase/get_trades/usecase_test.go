package gettrades_test

import (
	"testing"
	"time"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	gettrades "github.com/Murlokich/Dixchange/internal/domain/usecase/get_trades"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	validUserID = "019f7f2c-0caa-70ab-b1ec-414f744c3e33"
	pageSize    = 2
)

//nolint:gochecknoglobals // acceptable in test
var (
	validUUID = uuid.MustParse(validUserID)
	otherUser = uuid.MustParse("019f7f2c-0caa-70ab-b1ec-414f744c3e44")
	// USD/KCN: 1.00 USD lot at a price of 1 KCN
	validPair = entity.Pair{
		ID:        1,
		Base:      entity.Symbol{ID: 1, Code: "USD", Precision: 2},
		Quote:     entity.Symbol{ID: 2, Code: "KCN", Precision: 0},
		LotSize:   100,
		PriceTick: 1,
	}
	baseTime = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
)

// storedTrade builds a trade as the repository returns it: the pair carries only its ID.
// The user buys in even trades and sells in odd ones.
func storedTrade(i int) entity.Trade {
	me := entity.TradeParty{UserID: validUUID, OrderID: uuid.New()}
	other := entity.TradeParty{UserID: otherUser, OrderID: uuid.New()}
	trade := entity.Trade{
		ID:         uuid.New(),
		MarketPair: entity.Pair{ID: validPair.ID},
		Bid:        me,
		Ask:        other,
		Price:      3,
		Quantity:   200,
		CreatedAt:  baseTime.Add(-time.Duration(i) * time.Minute),
	}
	if i%2 == 1 {
		trade.Bid, trade.Ask = other, me
	}
	return trade
}

func TestUseCase_Execute_PageWithMore(t *testing.T) {
	t.Parallel()

	stored := []entity.Trade{storedTrade(0), storedTrade(1), storedTrade(2)}

	tradeRepo := gettrades.NewMockTradeRepository(t)
	marketRepo := gettrades.NewMockMarketRepository(t)

	tradeRepo.EXPECT().GetUserTrades(mock.Anything, validUUID, (*uuid.UUID)(nil), pageSize+1).
		Return(stored, nil).Once()
	marketRepo.EXPECT().GetPair(validPair.ID).Return(validPair, true).Times(pageSize)

	uc := gettrades.NewUseCase(tradeRepo, marketRepo, pageSize)
	output, err := uc.Execute(t.Context(), gettrades.Input{UserID: validUUID})

	require.NoError(t, err)
	require.Len(t, output.Trades, pageSize, "the extra trade only signals another page")

	buy, sell := output.Trades[0], output.Trades[1]
	require.Equal(t, entity.UserTradeParty{Side: entity.BuySide, OrderID: stored[0].Bid.OrderID}, buy.Party)
	require.Equal(t, entity.UserTradeParty{Side: entity.SellSide, OrderID: stored[1].Ask.OrderID}, sell.Party)
	require.Equal(t, validPair, buy.Trade.MarketPair)
	require.Equal(t, int64(6), buy.Total, "2.00 USD at 3 KCN costs 6 KCN")

	require.Equal(t, &stored[1].ID, output.NextCursor,
		"the cursor points at the last returned trade, not the extra one")
}

func TestUseCase_Execute_LastPage(t *testing.T) {
	t.Parallel()

	cursorID := uuid.New()
	cursor := &cursorID
	stored := []entity.Trade{storedTrade(1), storedTrade(2)}

	tradeRepo := gettrades.NewMockTradeRepository(t)
	marketRepo := gettrades.NewMockMarketRepository(t)

	tradeRepo.EXPECT().GetUserTrades(mock.Anything, validUUID, cursor, pageSize+1).Return(stored, nil).Once()
	marketRepo.EXPECT().GetPair(validPair.ID).Return(validPair, true).Times(len(stored))

	uc := gettrades.NewUseCase(tradeRepo, marketRepo, pageSize)
	output, err := uc.Execute(t.Context(), gettrades.Input{UserID: validUUID, Cursor: cursor})

	require.NoError(t, err)
	require.Len(t, output.Trades, len(stored))
	require.Nil(t, output.NextCursor)
}

func TestUseCase_Execute_NoTrades(t *testing.T) {
	t.Parallel()

	tradeRepo := gettrades.NewMockTradeRepository(t)
	marketRepo := gettrades.NewMockMarketRepository(t)

	tradeRepo.EXPECT().GetUserTrades(mock.Anything, validUUID, mock.Anything, pageSize+1).
		Return([]entity.Trade{}, nil).Once()

	uc := gettrades.NewUseCase(tradeRepo, marketRepo, pageSize)
	output, err := uc.Execute(t.Context(), gettrades.Input{UserID: validUUID})

	require.NoError(t, err)
	require.Empty(t, output.Trades)
	require.Nil(t, output.NextCursor)
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	// no expectations set up: the mocks panic if the usecase
	// calls them, proving Validate short-circuits Execute.
	tradeRepo := gettrades.NewMockTradeRepository(t)
	marketRepo := gettrades.NewMockMarketRepository(t)

	uc := gettrades.NewUseCase(tradeRepo, marketRepo, pageSize)
	_, err := uc.Execute(t.Context(), gettrades.Input{UserID: uuid.Nil})

	var valErr entity.ValidationError
	require.ErrorAs(t, err, &valErr)
}

func TestUseCase_Execute_UserIsNotParticipant(t *testing.T) {
	t.Parallel()

	trade := storedTrade(0)
	trade.Bid.UserID = uuid.New()

	tradeRepo := gettrades.NewMockTradeRepository(t)
	marketRepo := gettrades.NewMockMarketRepository(t)

	tradeRepo.EXPECT().GetUserTrades(mock.Anything, validUUID, mock.Anything, pageSize+1).
		Return([]entity.Trade{trade}, nil).Once()
	marketRepo.EXPECT().GetPair(validPair.ID).Return(validPair, true).Once()

	uc := gettrades.NewUseCase(tradeRepo, marketRepo, pageSize)
	_, err := uc.Execute(t.Context(), gettrades.Input{UserID: validUUID})

	require.ErrorIs(t, err, entity.ErrNotTradeParticipant)
}

func TestUseCase_Execute_PairDoesNotExist(t *testing.T) {
	t.Parallel()

	tradeRepo := gettrades.NewMockTradeRepository(t)
	marketRepo := gettrades.NewMockMarketRepository(t)

	tradeRepo.EXPECT().GetUserTrades(mock.Anything, validUUID, mock.Anything, pageSize+1).
		Return([]entity.Trade{storedTrade(0)}, nil).Once()
	marketRepo.EXPECT().GetPair(validPair.ID).Return(entity.Pair{}, false).Once()

	uc := gettrades.NewUseCase(tradeRepo, marketRepo, pageSize)
	_, err := uc.Execute(t.Context(), gettrades.Input{UserID: validUUID})

	require.ErrorContains(t, err, "market repo pair 1 doesn't exist")
}

func TestUseCase_Execute_RepositoryError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("connection refused")

	tradeRepo := gettrades.NewMockTradeRepository(t)
	marketRepo := gettrades.NewMockMarketRepository(t)

	tradeRepo.EXPECT().GetUserTrades(mock.Anything, validUUID, mock.Anything, pageSize+1).Return(nil, repoErr).Once()

	uc := gettrades.NewUseCase(tradeRepo, marketRepo, pageSize)
	_, err := uc.Execute(t.Context(), gettrades.Input{UserID: validUUID})

	require.ErrorIs(t, err, repoErr)
}
