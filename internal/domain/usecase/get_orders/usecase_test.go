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
	validUUID      = uuid.MustParse(validUserID)
	activeStatuses = []string{entity.StatusOpen, entity.StatusPartiallyFilled}
	validPair      = entity.Pair{
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

	orderRepo.EXPECT().GetUserOrdersByStatuses(mock.Anything, validUUID, activeStatuses).Return(stored, nil).Once()
	marketRepo.EXPECT().GetPair(validPair.ID).Return(validPair, true).Times(len(stored))

	uc := getorders.NewUseCase(orderRepo, marketRepo)
	output, err := uc.Execute(t.Context(), getorders.Input{UserID: validUUID, Statuses: activeStatuses})

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

	orderRepo.EXPECT().GetUserOrdersByStatuses(mock.Anything, validUUID, activeStatuses).
		Return([]entity.Order{}, nil).Once()

	uc := getorders.NewUseCase(orderRepo, marketRepo)
	output, err := uc.Execute(t.Context(), getorders.Input{UserID: validUUID, Statuses: activeStatuses})

	require.NoError(t, err)
	require.Empty(t, output.Orders)
}

func TestUseCase_Execute_ValidStatuses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		statuses []string
	}{
		{name: "open", statuses: []string{entity.StatusOpen}},
		{name: "partially filled", statuses: []string{entity.StatusPartiallyFilled}},
		{name: "both", statuses: activeStatuses},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			orderRepo := getorders.NewMockOrderRepository(t)
			marketRepo := getorders.NewMockMarketRepository(t)

			orderRepo.EXPECT().GetUserOrdersByStatuses(mock.Anything, validUUID, tc.statuses).
				Return([]entity.Order{}, nil).Once()

			uc := getorders.NewUseCase(orderRepo, marketRepo)
			_, err := uc.Execute(t.Context(), getorders.Input{UserID: validUUID, Statuses: tc.statuses})

			require.NoError(t, err)
		})
	}
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		userID   uuid.UUID
		statuses []string
	}{
		{name: "nil user id", userID: uuid.Nil, statuses: activeStatuses},
		{name: "no statuses", userID: validUUID, statuses: nil},
		{name: "empty status", userID: validUUID, statuses: []string{entity.StatusOpen, ""}},
		{name: "filled status is not supported yet", userID: validUUID, statuses: []string{entity.StatusFilled}},
		{
			name:     "cancelled status is not supported yet",
			userID:   validUUID,
			statuses: []string{entity.StatusOpen, entity.StatusCancelled},
		},
		{name: "unknown status", userID: validUUID, statuses: []string{"Open"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// no expectations set up: the mocks panic if the usecase
			// calls them, proving Validate short-circuits Execute.
			orderRepo := getorders.NewMockOrderRepository(t)
			marketRepo := getorders.NewMockMarketRepository(t)

			uc := getorders.NewUseCase(orderRepo, marketRepo)
			_, err := uc.Execute(t.Context(), getorders.Input{UserID: tc.userID, Statuses: tc.statuses})

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

	orderRepo.EXPECT().GetUserOrdersByStatuses(mock.Anything, validUUID, activeStatuses).Return(stored, nil).Once()
	marketRepo.EXPECT().GetPair(42).Return(entity.Pair{}, false).Once()

	uc := getorders.NewUseCase(orderRepo, marketRepo)
	_, err := uc.Execute(t.Context(), getorders.Input{UserID: validUUID, Statuses: activeStatuses})

	require.ErrorContains(t, err, "market repo pair 42")
	var valErr entity.ValidationError
	require.NotErrorAs(t, err, &valErr, "a stored order with an unknown pair is an internal error")
}

func TestUseCase_Execute_RepositoryError(t *testing.T) {
	t.Parallel()

	repoErr := errors.New("connection refused")

	orderRepo := getorders.NewMockOrderRepository(t)
	marketRepo := getorders.NewMockMarketRepository(t)

	orderRepo.EXPECT().GetUserOrdersByStatuses(mock.Anything, validUUID, activeStatuses).Return(nil, repoErr).Once()

	uc := getorders.NewUseCase(orderRepo, marketRepo)
	_, err := uc.Execute(t.Context(), getorders.Input{UserID: validUUID, Statuses: activeStatuses})

	require.ErrorIs(t, err, repoErr)
}
