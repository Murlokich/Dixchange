package main

import (
	"net/http"

	authAPI "github.com/dixchange/Monolith/internal/auth/api"
	"github.com/dixchange/Monolith/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	var cfg config.Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		//log.Errorf("failed to process config: %v", err)
		return
	}

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})

	apiRouter := router.Group("api")

	authAPI.RegisterEndpoints(apiRouter)

	router.Run(cfg.HTTPConfig.Addr)
}
