package getorders_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	getorders "github.com/dixchange/Monolith/internal/domain/usecase/get_orders"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const validUserID = "019f7f2c-0caa-70ab-b1ec-414f744c3e33"

//nolint:gochecknoglobals // acceptable in test
var (
	validUUID = uuid.MustParse(validUserID)
	validPair = entity.Pair{
		ID:        1,
		Base:      entity.Symbol{ID: 1, Code: "USD", Precision: 2},
		Quote:     entity.Symbol{ID: 2, Code: "KCN", Precision: 0},
		LotSize:   100,
		PriceTick: 1,
	}
)

func TestUseCase_Execute_ResolvesMarketPair(t *testing.T) {
	t.Parallel()

	// the repository returns orders carrying only the pair ID
	stored := []entity.Order{
		{ID: uuid.New(), UserID: validUUID, MarketPair: entity.Pair{ID: validPair.ID}, Status: entity.StatusOpen},
		{ID: uuid.New(), UserID: validUUID, MarketPair: entity.Pair{ID: validPair.ID}, Status: entity.StatusOpen},
	}

	orderRepo := getorders.NewMockOrderRepository(t)
	marketRepo := getorders.NewMockMarketRepository(t)

	orderRepo.EXPECT().GetUserOrdersByStatus(mock.Anything, validUUID, entity.StatusOpen).Return(stored, nil).Once()
	marketRepo.EXPECT().GetPair(validPair.ID).Return(validPair, true).Times(len(stored))

	uc := getorders.NewUseCase(orderRepo, marketRepo)
	output, err := uc.Execute(t.Context(), getorders.Input{UserID: validUUID, Status: entity.StatusOpen})

	require.NoError(t, err)
	require.Len(t, output.Orders, len(stored))
	for i, order := range output.Orders {
		require.Equal(t, stored[i].ID, order.ID)
		require.Equal(t, validPair, order.MarketPair)
	}
}

func TestUseCase_Execute_NoOrders(t *testing.T) {
	t.Parallel()

	orderRepo := getorders.NewMockOrderRepository(t)
	marketRepo := getorders.NewMockMarketRepository(t)

	orderRepo.EXPECT().GetUserOrdersByStatus(mock.Anything, validUUID, entity.StatusOpen).
		Return([]entity.Order{}, nil).Once()

	uc := getorders.NewUseCase(orderRepo, marketRepo)
	output, err := uc.Execute(t.Context(), getorders.Input{UserID: validUUID, Status: entity.StatusOpen})

	require.NoError(t, err)
	require.Empty(t, output.Orders)
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		userID uuid.UUID
		status string
	}{
		{name: "nil user id", userID: uuid.Nil, status: entity.StatusOpen},
		{name: "empty status", userID: validUUID, status: ""},
		{name: "filled status is not supported yet", userID: validUUID, status: entity.StatusFilled},
		{name: "cancelled status is not supported yet", userID: validUUID, status: entity.StatusCancelled},
		{name: "unknown status", userID: validUUID, status: "Open"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// no expectations set up: the mocks panic if the usecase
			// calls them, proving Validate short-circuits Execute.
			orderRepo := getorders.NewMockOrderRepository(t)
			marketRepo := getorders.NewMockMarketRepository(t)

			uc := getorders.NewUseCase(orderRepo, marketRepo)
			_, err := uc.Execute(t.Context(), getorders.Input{UserID: tc.userID, Status: tc.status})

			var valErr entity.ValidationError
			require.ErrorAs(t, err, &valErr)
		})
	}
}

func TestUseCase_Execute_PairDoesNotExist(t *testing.T) {
	t.Parallel()

	stored := []entity.Order{{ID: uuid.New(), MarketPair: entity.Pair{ID: 42}}}

	orderRepo := getorders.NewMockOrderRepository(t)
	marketRepo := getorders.NewMockMarketRepository(t)

	orderRepo.EXPECT().GetUserOrdersByStatus(mock.Anything, validUUID, entity.StatusOpen).Return(stored, nil).Once()
	marketRepo.EXPECT().GetPair(42).Return(entity.Pair{}, false).Once()

	uc := getorders.NewUseCase(orderRepo, marketRepo)
	_, err := uc.Execute(t.Context(), getorders.Input{UserID: validUUID, Status: entity.StatusOpen})

	require.ErrorContains(t, err, "market repo pair 42")
	var valErr entity.ValidationError
	require.NotErrorAs(t, err, &valErr, "a stored order with an unknown pair is an internal error")
}

func TestUseCase_Execute_RepositoryError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("connection refused")

	orderRepo := getorders.NewMockOrderRepository(t)
	marketRepo := getorders.NewMockMarketRepository(t)

	orderRepo.EXPECT().GetUserOrdersByStatus(mock.Anything, validUUID, entity.StatusOpen).Return(nil, repoErr).Once()

	uc := getorders.NewUseCase(orderRepo, marketRepo)
	_, err := uc.Execute(t.Context(), getorders.Input{UserID: validUUID, Status: entity.StatusOpen})

	require.ErrorIs(t, err, repoErr)
}
