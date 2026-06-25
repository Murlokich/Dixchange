package http

import (
	"github.com/gin-gonic/gin"

	"github.com/dixchange/Monolith/internal/auth/domain/usecase/register"
)

type AuthHandler func(c *gin.Context, deps *AuthDependencies)

type AuthDependencies struct {
	reg *register.UseCase
}

func (d *AuthDependencies) WrapHandler(handler AuthHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		handler(c, d)
	}
}

func NewAuthDependencies(reg *register.UseCase) *AuthDependencies {
	return &AuthDependencies{
		reg: reg,
	}
}
