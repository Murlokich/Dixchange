package http

import (
	"github.com/gin-gonic/gin"

	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	"github.com/dixchange/Monolith/internal/domain/usecase/login"
	"github.com/dixchange/Monolith/internal/domain/usecase/register"
)

type Handler func(c *gin.Context, deps *Dependencies)

type Logger interface {
	Infof(string, ...any)
	Errorf(string, ...any)
}

type Dependencies struct {
	Logger Logger

	Register *register.UseCase
	Login    *login.UseCase
	Deposit  *deposit.UseCase
}

func (d *Dependencies) WrapHandler(handler Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c, d)
	}
}
