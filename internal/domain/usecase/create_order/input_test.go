package createorder_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	createorder "github.com/dixchange/Monolith/internal/domain/usecase/create_order"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const (
	validUserID       = "019f7f2c-0caa-70ab-b1ec-414f744c3e33"
	validMarketPairID = 1

	validQuantityOrdered int64 = 5
	validPricePerUnit    int64 = 10
)

var (
	validUUID = uuid.MustParse(validUserID) //nolint:gochecknoglobals // acceptable in test

	validQuantityDecimal = decimal.NewFromInt(validQuantityOrdered) //nolint:gochecknoglobals // acceptable in test
	validPriceDecimal    = decimal.NewFromInt(validPricePerUnit)    //nolint:gochecknoglobals // acceptable in test
)

func TestInput_Validate_Action(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		action             string
		isValid            bool
		containsErrMessage string
	}{
		{name: "valid: buy", action: entity.BuyAction, isValid: true},
		{name: "valid: sell", action: entity.SellAction, isValid: true},
		{name: "invalid: empty", action: "", isValid: false, containsErrMessage: "blank"},
		{name: "invalid: gibberish", action: "asdf", isValid: false, containsErrMessage: "not valid status"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := createorder.Input{
				UserID:          validUUID,
				MarketPairID:    validMarketPairID,
				Action:          tc.action,
				QuantityOrdered: validQuantityDecimal,
				PricePerUnit:    validPriceDecimal,
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

func TestInput_Validate_MarketPairID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name               string
		marketPairID       int
		isValid            bool
		containsErrMessage string
	}{
		{name: "valid: positive", marketPairID: validMarketPairID, isValid: true},
		{name: "invalid: negative", marketPairID: -1, isValid: false, containsErrMessage: "no less than"},
		{name: "invalid: zero", marketPairID: 0, isValid: false, containsErrMessage: "blank"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			input := createorder.Input{
				UserID:          validUUID,
				MarketPairID:    tc.marketPairID,
				Action:          entity.BuyAction,
				QuantityOrdered: validQuantityDecimal,
				PricePerUnit:    validPriceDecimal,
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
