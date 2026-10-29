package login

import (
	"context"

	"github.com/Murlokich/Dixchange/internal/domain/entity"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type Repository interface {
	GetUserByEmail(ctx context.Context, email string) (entity.User, error)
}

type PasswordHasher interface {
	ComparePasswords(password string, passwordHash string) (bool, error)
}

type JWTIssuer interface {
	IssueToken(userID uuid.UUID) (string, error)
}

type UseCase struct {
	repo      Repository
	pwdHasher PasswordHasher
	jwtIssuer JWTIssuer
}

// TODO: for now - assign single long term access token
// make normal authentication with refresh token, revocation, etc.
func (u *UseCase) Execute(ctx context.Context, input Input) (Output, error) {
	if err := input.Validate(ctx); err != nil {
		return Output{}, errors.Wrap(err, "input validate")
	}

	user, err := u.repo.GetUserByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, entity.ErrNoUserFound) {
			return Output{}, errors.Wrap(ErrInvalidCredentials, "repo get user by email")
		}
		return Output{}, errors.Wrap(err, "repo get user by email")
	}
	match, err := u.pwdHasher.ComparePasswords(input.Password, user.PasswordHash)
	if err != nil {
		return Output{}, errors.Wrap(err, "password hasher compare")
	}
	if !match {
		return Output{}, errors.Wrap(ErrInvalidCredentials, "password hasher compare")
	}
	token, err := u.jwtIssuer.IssueToken(user.ID)
	if err != nil {
		return Output{}, errors.Wrap(err, "jwt issuer issue token")
	}
	return Output{AccessToken: token}, nil
}

func NewUseCase(repo Repository, pwdHasher PasswordHasher, jwtIssuer JWTIssuer) *UseCase {
	return &UseCase{
		repo:      repo,
		pwdHasher: pwdHasher,
		jwtIssuer: jwtIssuer,
	}
}
