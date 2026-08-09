package login_test

import (
	"testing"

	"github.com/dixchange/Monolith/internal/domain/entity"
	"github.com/dixchange/Monolith/internal/domain/usecase/login"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	validEmail    = "user@example.com"
	validPassword = "Passw0rd!"
)

func TestUseCase_Execute_HappyPath(t *testing.T) {
	t.Parallel()

	repo := login.NewMockRepository(t)
	pwdHasher := login.NewMockPasswordHasher(t)
	jwtIssuer := login.NewMockJWTIssuer(t)

	user := entity.User{ID: uuid.New(), Email: validEmail, PasswordHash: "hashed-password"}
	const wantToken = "access-token"

	repo.EXPECT().GetUserByEmail(mock.Anything, validEmail).Return(user, nil).Once()
	pwdHasher.EXPECT().ComparePasswords(validPassword, user.PasswordHash).Return(true, nil).Once()
	jwtIssuer.EXPECT().IssueToken(user.ID).Return(wantToken, nil).Once()

	uc := login.NewUseCase(repo, pwdHasher, jwtIssuer)
	output, err := uc.Execute(t.Context(), login.Input{Email: validEmail, Password: validPassword})

	require.NoError(t, err)
	require.Equal(t, wantToken, output.AccessToken)
}

func TestUseCase_Execute_InvalidInput(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		email string
		pwd   string
	}{
		{name: "invalid email", email: "not-an-email", pwd: validPassword},
		{name: "invalid password", email: validEmail, pwd: "weak"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// no expectations set up: the mocks panic if the usecase
			// calls them, proving Validate short-circuits Execute.
			repo := login.NewMockRepository(t)
			pwdHasher := login.NewMockPasswordHasher(t)
			jwtIssuer := login.NewMockJWTIssuer(t)

			uc := login.NewUseCase(repo, pwdHasher, jwtIssuer)
			_, err := uc.Execute(t.Context(), login.Input{Email: tc.email, Password: tc.pwd})

			var valErr entity.ValidationError
			require.ErrorAs(t, err, &valErr)
		})
	}
}

func TestUseCase_Execute_UserNotFound(t *testing.T) {
	t.Parallel()

	repo := login.NewMockRepository(t)
	pwdHasher := login.NewMockPasswordHasher(t)
	jwtIssuer := login.NewMockJWTIssuer(t)

	repo.EXPECT().
		GetUserByEmail(mock.Anything, validEmail).
		Return(entity.User{}, errors.Wrap(entity.ErrNoUserFound, "user repo get by email")).
		Once()

	uc := login.NewUseCase(repo, pwdHasher, jwtIssuer)
	_, err := uc.Execute(t.Context(), login.Input{Email: validEmail, Password: validPassword})

	require.ErrorIs(t, err, login.ErrInvalidCredentials)
}

func TestUseCase_Execute_PasswordMismatch(t *testing.T) {
	t.Parallel()

	repo := login.NewMockRepository(t)
	pwdHasher := login.NewMockPasswordHasher(t)
	jwtIssuer := login.NewMockJWTIssuer(t)

	user := entity.User{ID: uuid.New(), Email: validEmail, PasswordHash: "hashed-password"}

	repo.EXPECT().GetUserByEmail(mock.Anything, validEmail).Return(user, nil).Once()
	pwdHasher.EXPECT().ComparePasswords(validPassword, user.PasswordHash).Return(false, nil).Once()

	uc := login.NewUseCase(repo, pwdHasher, jwtIssuer)
	_, err := uc.Execute(t.Context(), login.Input{Email: validEmail, Password: validPassword})

	require.ErrorIs(t, err, login.ErrInvalidCredentials)
}
