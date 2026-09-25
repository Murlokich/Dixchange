package http

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	createorder "github.com/dixchange/Monolith/internal/domain/usecase/create_order"
	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	getmarketdata "github.com/dixchange/Monolith/internal/domain/usecase/get_market_data"
	getpairbalance "github.com/dixchange/Monolith/internal/domain/usecase/get_pair_balance"
	"github.com/dixchange/Monolith/internal/domain/usecase/login"
	"github.com/dixchange/Monolith/internal/domain/usecase/register"
	"github.com/dixchange/Monolith/pkg"
)

type Handler func(c *gin.Context, deps *Dependencies)

type TokenValidator interface {
	ValidateToken(accessToken string) (*jwt.Token, error)
}

type Dependencies struct {
	Logger         pkg.Logger
	TokenValidator TokenValidator

	Register       *register.UseCase
	Login          *login.UseCase
	Deposit        *deposit.UseCase
	CreateOrder    *createorder.UseCase
	GetMarketData  *getmarketdata.UseCase
	GetPairBalance *getpairbalance.UseCase
}

func (d *Dependencies) WrapHandler(handler Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		reqDeps := *d
		if l, ok := c.Get(loggerWithUserIDCtxKey); ok {
			reqDeps.Logger, ok = l.(pkg.Logger)
			if !ok {
				reqDeps.Logger = d.Logger
			}
		}
		handler(c, &reqDeps)
	}
}
