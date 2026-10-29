package utils_test

import (
	"testing"

	"github.com/Murlokich/Dixchange/internal/utils"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const (
	validUserUID = "019f7f2c-0caa-70ab-b1ec-414f744c3e33"

	validPositiveAmount = 10
)

func TestValidateUserID(t *testing.T) {
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
			errExpected: utils.ErrEmptyUserUID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := utils.ValidateUserID(t.Context(), tc.userID)
			if tc.errExpected == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.errExpected)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		password           string
		isValid            bool
		containsErrMessage string
	}{
		{name: "invalid: empty password", password: "", isValid: false, containsErrMessage: "blank"},
		{name: "invalid: short password", password: "aaaaaaa", isValid: false, containsErrMessage: "length"},
		{name: "invalid: only lowercase", password: "aaaaaaaa", isValid: false, containsErrMessage: "contain upppercase letter"}, //nolint:lll
		{name: "invalid: no digit", password: "Aaaaaaaa", isValid: false, containsErrMessage: "contain digit"},
		{name: "invalid: no special char", password: "A1aaaaaa", isValid: false, containsErrMessage: "contain special char"},
		{name: "valid", password: "A1!aaaaa", isValid: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := utils.ValidatePassword(t.Context(), tc.password)
			if tc.isValid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.containsErrMessage)
			}
		})
	}
}

func TestValidatePositiveDecimal(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		amount      decimal.Decimal
		errExpected error
	}{
		{
			name:        "valid positive amount",
			amount:      decimal.NewFromInt(validPositiveAmount),
			errExpected: nil,
		},
		{
			name:        "invalid negative amount",
			amount:      decimal.NewFromInt(-1),
			errExpected: utils.ErrDecimalNonPositive,
		},
		{
			name:        "invalid zero amount",
			amount:      decimal.NewFromInt(0),
			errExpected: utils.ErrDecimalNonPositive,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := utils.ValidatePositiveDecimal(t.Context(), tc.amount)
			if tc.errExpected == nil {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, tc.errExpected)
			}
		})
	}
}
