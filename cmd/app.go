package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	authAPI "github.com/dixchange/Monolith/internal/auth/api"
	"github.com/dixchange/Monolith/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})

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

	srv := &http.Server{
		Addr:    cfg.HTTPConfig.Addr,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// graceful shutdown
	<-ctx.Done()

	log.Infoln("shutting down gracefully")
	// Give active connections 5 seconds to finish processing
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Infoln("Server exited cleanly")
}
