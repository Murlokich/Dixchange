package api

import (
	"github.com/gin-gonic/gin"
)

func RegisterEndpoints(router *gin.RouterGroup) {
	authRouter := router.Group("auth")

	authRouter.POST("/login", loginHandler)
	authRouter.POST("/register", registerHandler)
}
