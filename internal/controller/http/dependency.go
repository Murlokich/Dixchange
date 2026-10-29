package http

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	cancelorder "github.com/Murlokich/Dixchange/internal/domain/usecase/cancel_order"
	createorder "github.com/Murlokich/Dixchange/internal/domain/usecase/create_order"
	"github.com/Murlokich/Dixchange/internal/domain/usecase/deposit"
	getmarketdata "github.com/Murlokich/Dixchange/internal/domain/usecase/get_market_data"
	getorders "github.com/Murlokich/Dixchange/internal/domain/usecase/get_orders"
	getpairbalance "github.com/Murlokich/Dixchange/internal/domain/usecase/get_pair_balance"
	gettrades "github.com/Murlokich/Dixchange/internal/domain/usecase/get_trades"
	"github.com/Murlokich/Dixchange/internal/domain/usecase/login"
	"github.com/Murlokich/Dixchange/internal/domain/usecase/register"
	"github.com/Murlokich/Dixchange/pkg"
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
	CancelOrder    *cancelorder.UseCase
	GetMarketData  *getmarketdata.UseCase
	GetPairBalance *getpairbalance.UseCase
	GetOrders      *getorders.UseCase
	GetTrades      *gettrades.UseCase
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
