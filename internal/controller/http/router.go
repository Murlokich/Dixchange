package http

import (
	"github.com/gin-gonic/gin"
)

func RegisterEndpoints(router *gin.RouterGroup, d *Dependencies) {
	authRouter := router.Group("auth")
	authRouter.POST("/login", d.WrapHandler(loginHandler))
	authRouter.POST("/register", d.WrapHandler(registerHandler))

	balanceRouter := router.Group("balance")
	balanceRouter.POST("/:user_id/deposit", d.WrapHandler(depositHandler))
}
