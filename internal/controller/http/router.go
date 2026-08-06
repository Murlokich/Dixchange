package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterEndpoints(router *gin.Engine, d *Dependencies) {
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})

	apiRouter := router.Group("api")

	authRouter := apiRouter.Group("auth")
	authRouter.POST("/login", d.WrapHandler(loginHandler))
	authRouter.POST("/register", d.WrapHandler(registerHandler))

	balanceRouter := apiRouter.Group("balance")
	balanceRouter.POST("/deposit", d.WrapHandler(authMiddleware), d.WrapHandler(depositHandler))

	orderRouter := apiRouter.Group("orders")
	orderRouter.POST("", d.WrapHandler(authMiddleware), d.WrapHandler(createOrderHandler))
}
