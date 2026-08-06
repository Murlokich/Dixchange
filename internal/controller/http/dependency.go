package http

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	createorder "github.com/dixchange/Monolith/internal/domain/usecase/create_order"
	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	"github.com/dixchange/Monolith/internal/domain/usecase/login"
	"github.com/dixchange/Monolith/internal/domain/usecase/register"
)

type Handler func(c *gin.Context, deps *Dependencies)

type Logger interface {
	Infof(string, ...any)
	Errorf(string, ...any)
}

type TokenValidator interface {
	ValidateToken(accessToken string) (*jwt.Token, error)
}

type Dependencies struct {
	Logger         Logger
	TokenValidator TokenValidator

	Register    *register.UseCase
	Login       *login.UseCase
	Deposit     *deposit.UseCase
	CreateOrder *createorder.UseCase
}

func (d *Dependencies) WrapHandler(handler Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c, d)
	}
}
