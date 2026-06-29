package http

import (
	"github.com/gin-gonic/gin"

	"github.com/dixchange/Monolith/internal/auth/domain/usecase/login"
	"github.com/dixchange/Monolith/internal/auth/domain/usecase/register"
)

type AuthHandler func(c *gin.Context, deps *AuthDependencies)

type Logger interface {
	Infof(string, ...any)
	Errorf(string, ...any)
}

type AuthDependencies struct {
	logger Logger
	reg    *register.UseCase
	log    *login.UseCase
}

func (d *AuthDependencies) WrapHandler(handler AuthHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c, d)
	}
}

func NewAuthDependencies(reg *register.UseCase, log *login.UseCase, logger Logger) *AuthDependencies {
	return &AuthDependencies{
		reg:    reg,
		log:    log,
		logger: logger,
	}
}
