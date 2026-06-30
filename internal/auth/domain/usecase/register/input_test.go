package register_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dixchange/Monolith/internal/auth/domain/usecase/register"
)

const (
	EmailErrMessage      = "email"
	MatchRegexErrMessage = "valid format"
	BlankErrMessage      = "blank"

	ValidEmail    = "example@gmail.com"
	ValidPassword = "Aa1@bbbbbbbbbbb"
)

func TestInput_Validate_Email(t *testing.T) {
	testCases := []struct {
		Name               string
		Email              string
		IsValid            bool
		ContainsErrMessage string
	}{
		{
			Name:               "Invalid: Empty email",
			Email:              "",
			IsValid:            false,
			ContainsErrMessage: BlankErrMessage,
		},
		{
			Name:               "Invalid: Single word email",
			Email:              "word",
			IsValid:            false,
			ContainsErrMessage: EmailErrMessage,
		},
		{
			Name:               "Invalid: only domain",
			Email:              "@gmail.com",
			IsValid:            false,
			ContainsErrMessage: EmailErrMessage,
		},
		{
			Name:               "Invalid: only .com",
			Email:              ".com",
			IsValid:            false,
			ContainsErrMessage: EmailErrMessage,
		},
		{
			Name:               "Valid: gmail.com",
			Email:              "example@gmail.com",
			IsValid:            true,
			ContainsErrMessage: EmailErrMessage,
		},
		{
			Name:               "Valid: yahoo.com",
			Email:              "example@yahoo.com",
			IsValid:            true,
			ContainsErrMessage: EmailErrMessage,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			input := register.Input{Email: tc.Email, Password: ValidPassword}
			err := input.Validate(t.Context())
			if tc.IsValid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.ContainsErrMessage)
			}
		})
	}
}

func TestInput_Validate_Password(t *testing.T) {
	testCases := []struct {
		Name               string
		Password           string
		IsValid            bool
		ContainsErrMessage string
	}{
		{
			Name:               "Invalid: Empty password",
			Password:           "",
			IsValid:            false,
			ContainsErrMessage: BlankErrMessage,
		},
		{
			Name:               "Invalid: Short password",
			Password:           "aaaaaaa",
			IsValid:            false,
			ContainsErrMessage: "length",
		},
		{
			Name:               "Invalid: Only lowercase",
			Password:           "aaaaaaaa",
			IsValid:            false,
			ContainsErrMessage: MatchRegexErrMessage,
		},
		{
			Name:               "Invalid: No digit",
			Password:           "Aaaaaaaa",
			IsValid:            false,
			ContainsErrMessage: MatchRegexErrMessage,
		},
		{
			Name:               "Invalid: No special char",
			Password:           "A1aaaaaa",
			IsValid:            false,
			ContainsErrMessage: MatchRegexErrMessage,
		},
		{
			Name:               "Valid",
			Password:           "A1!aaaaa",
			IsValid:            true,
			ContainsErrMessage: MatchRegexErrMessage,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			input := register.Input{Email: ValidEmail, Password: tc.Password}
			err := input.Validate(t.Context())
			if tc.IsValid {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.ContainsErrMessage)
			}
		})
	}
}
