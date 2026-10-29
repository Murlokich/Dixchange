package deposit_test

import (
	"testing"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
	"github.com/Murlokich/Dixchange/internal/domain/usecase/deposit"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const (
	validUserID   = "019f7f2c-0caa-70ab-b1ec-414f744c3e33"
	validSymbolID = 1
	validAmount   = 10
)

var (
	validUUID          = uuid.MustParse(validUserID)     //nolint:gochecknoglobals // acceptable in test
	validAmountDecimal = decimal.NewFromInt(validAmount) //nolint:gochecknoglobals // acceptable in test
)

func TestInput_Validate_SymbolID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		symbolID           int
		isValid            bool
		containsErrMessage string
	}{
		{name: "valid: positive", symbolID: validSymbolID, isValid: true},
		{name: "invalid: negative", symbolID: -1, isValid: false, containsErrMessage: "no less than"},
		{name: "invalid: zero", symbolID: 0, isValid: false, containsErrMessage: "blank"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := deposit.Input{
				UserID:   validUUID,
				SymbolID: tc.symbolID,
				Amount:   validAmountDecimal,
			}
			err := input.Validate(t.Context())
			if tc.isValid {
				require.NoError(t, err)
			} else {
				var valError entity.ValidationError
				require.ErrorAs(t, err, &valError)
				require.ErrorContains(t, valError, tc.containsErrMessage)
			}
		})
	}
}
