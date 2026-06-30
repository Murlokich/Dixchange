package http

import (
	"github.com/gin-gonic/gin"
)

func RegisterEndpoints(router *gin.RouterGroup, d *AuthDependencies) {
	authRouter := router.Group("auth")

	authRouter.POST("/login", d.WrapHandler(loginHandler))
	authRouter.POST("/register", d.WrapHandler(registerHandler))
}
