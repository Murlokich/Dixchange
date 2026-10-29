package getmarketdata_test

import (
	"testing"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	getmarketdata "github.com/Murlokich/Dixchange/internal/domain/usecase/get_market_data"
	"github.com/stretchr/testify/require"
)

func TestUseCase_Execute_ReturnsPairsSortedByID(t *testing.T) {
	t.Parallel()

	usd := entity.Symbol{ID: 1, Code: "USD", Precision: 2}
	kcn := entity.Symbol{ID: 2, Code: "KCN", Precision: 0}
	eur := entity.Symbol{ID: 3, Code: "EUR", Precision: 2}
	pair1 := entity.Pair{ID: 1, Base: usd, Quote: kcn, LotSize: 100, PriceTick: 1}
	pair2 := entity.Pair{ID: 2, Base: eur, Quote: kcn, LotSize: 100, PriceTick: 1}
	pair3 := entity.Pair{ID: 3, Base: eur, Quote: usd, LotSize: 1, PriceTick: 1, PricePrecision: 4}

	marketRepo := getmarketdata.NewMockMarketRepository(t)
	marketRepo.EXPECT().GetPairs().Return(map[int]entity.Pair{3: pair3, 1: pair1, 2: pair2}).Once()

	uc := getmarketdata.NewUseCase(marketRepo)
	output, err := uc.Execute(t.Context())

	require.NoError(t, err)
	require.Equal(t, []entity.Pair{pair1, pair2, pair3}, output.Pairs)
}

func TestUseCase_Execute_NoPairs(t *testing.T) {
	t.Parallel()

	marketRepo := getmarketdata.NewMockMarketRepository(t)
	marketRepo.EXPECT().GetPairs().Return(map[int]entity.Pair{}).Once()

	uc := getmarketdata.NewUseCase(marketRepo)
	output, err := uc.Execute(t.Context())

	require.NoError(t, err)
	require.Empty(t, output.Pairs)
}
