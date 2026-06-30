package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/dixchange/Monolith/internal/auth/adapter/crypto/argon2"
	"github.com/dixchange/Monolith/internal/auth/adapter/jwt"
	"github.com/dixchange/Monolith/internal/auth/adapter/sql/user"
	authAPI "github.com/dixchange/Monolith/internal/auth/controller/http"
	"github.com/dixchange/Monolith/internal/auth/domain/usecase/login"
	"github.com/dixchange/Monolith/internal/auth/domain/usecase/register"
	"github.com/dixchange/Monolith/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	gracefulShutdownTime = 5 * time.Second
)

func runMigrations(dbConfig *config.DBConfig) error {
	m, err := migrate.New("file://../migrations", dbConfig.URL)
	if err != nil {
		return errors.Wrap(err, "create migration")
	}
	err = m.Migrate(dbConfig.MigrationVersion)
	if err != nil {
		return errors.Wrap(err, "migrate database")
	}
	errSrc, errDB := m.Close()
	if errSrc != nil {
		return errors.Wrap(errSrc, "close migration source")
	}
	if errDB != nil {
		return errors.Wrap(errDB, "close database connection")
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})

	var cfg config.Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Errorf("process config: %v", err)
		return
	}

	if err := runMigrations(&cfg.DBConfig); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Errorf("run migrations: %v", err)
		return
	}

	postgresPool, err := pgxpool.New(ctx, cfg.DBConfig.URL)
	if err != nil {
		log.Errorf("connect to db: %v", err)
		return
	}
	defer postgresPool.Close()

	userRepo := user.NewRepository(postgresPool)
	pwdHasher := argon2.NewArgon2(&cfg.AuthConfig.Argon2Config)
	jwtManager, err := jwt.NewManager(&cfg.AuthConfig.JWTConfig)
	if err != nil {
		log.Errorf("create new manager: %v", err)
		return
	}

	regUseCase := register.NewUseCase(userRepo, pwdHasher)
	logUseCase := login.NewUseCase(userRepo, pwdHasher, jwtManager)
	authDeps := authAPI.NewAuthDependencies(regUseCase, logUseCase, log)

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})

	apiRouter := router.Group("api")

	authAPI.RegisterEndpoints(apiRouter, authDeps)

	srv := &http.Server{
		Addr:              cfg.HTTPConfig.Addr,
		ReadHeaderTimeout: time.Duration(cfg.HTTPConfig.ReadHeaderTimeoutSeconds) * time.Second,
		Handler:           router,
	}

	go func() {
		if err = srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// graceful shutdown
	<-ctx.Done()

	log.Infoln("shutting down gracefully")
	// Give active connections 5 seconds to finish processing
	shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTime)
	defer cancel()

	if err = srv.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown: ", err)
	}

	postgresPool.Close()

	log.Infoln("Server exited cleanly")
}
