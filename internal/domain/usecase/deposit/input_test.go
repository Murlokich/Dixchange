package deposit_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const (
	validUSDAmount = 10
	validUserUID   = "019f7f2c-0caa-70ab-b1ec-414f744c3e33"
)

func TestInput_Validate_UserID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		userID      uuid.UUID
		errExpected error
	}{
		{
			name:        "valid uid",
			userID:      uuid.MustParse(validUserUID),
			errExpected: nil,
		},
		{
			name:        "invalid nil uid",
			userID:      uuid.Nil,
			errExpected: deposit.ErrEmptyUserUID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := deposit.Input{UserID: tc.userID, Amount: decimal.NewFromInt(validUSDAmount)}
			err := input.Validate(t.Context())
			if tc.errExpected == nil {
				require.NoError(t, err)
			} else {
				var valError entity.ValidationError
				require.ErrorAs(t, err, &valError)
				require.ErrorIs(t, valError.Err, tc.errExpected)
			}
		})
	}
}

func TestInput_Validate_USDAmount(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		usdAmount   decimal.Decimal
		errExpected error
	}{
		{
			name:        "valid usd amount",
			usdAmount:   decimal.NewFromInt(validUSDAmount),
			errExpected: nil,
		},
		{
			name:        "invalid negative usd amount",
			usdAmount:   decimal.NewFromInt(-1),
			errExpected: deposit.ErrUSDAmountNonPositive,
		},
		{
			name:        "invalid zero usd amount",
			usdAmount:   decimal.NewFromInt(0),
			errExpected: deposit.ErrUSDAmountNonPositive,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := deposit.Input{UserID: uuid.MustParse(validUserUID), Amount: tc.usdAmount}
			err := input.Validate(t.Context())
			if tc.errExpected == nil {
				require.NoError(t, err)
			} else {
				var valError entity.ValidationError
				require.ErrorAs(t, err, &valError)
				require.ErrorIs(t, valError.Err, tc.errExpected)
			}
		})
	}
}
