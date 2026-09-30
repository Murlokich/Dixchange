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
	balanceRouter.GET("/:pair_id", d.WrapHandler(authMiddleware), d.WrapHandler(getPairBalanceHandler))

	orderRouter := apiRouter.Group("orders")
	orderRouter.POST("", d.WrapHandler(authMiddleware), d.WrapHandler(createOrderHandler))
	orderRouter.GET("", d.WrapHandler(authMiddleware), d.WrapHandler(getOrdersHandler))
	orderRouter.POST("/:id/cancel", d.WrapHandler(authMiddleware), d.WrapHandler(cancelOrderHandler))

	tradeRouter := apiRouter.Group("trades")
	tradeRouter.GET("", d.WrapHandler(authMiddleware), d.WrapHandler(getTradesHandler))

	marketRouter := apiRouter.Group("market")
	marketRouter.GET("/pairs", d.WrapHandler(getMarketDataHandler))
}
