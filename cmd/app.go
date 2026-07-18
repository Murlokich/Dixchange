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
	"github.com/dixchange/Monolith/internal/balance/adapter/sql/wallet"
	"github.com/dixchange/Monolith/internal/config"
	api "github.com/dixchange/Monolith/internal/controller/http"
	"github.com/dixchange/Monolith/internal/domain/usecase/login"
	"github.com/dixchange/Monolith/internal/domain/usecase/register"
	"github.com/dixchange/Monolith/pkg"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	shopspringDecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
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

func setupPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	const errWrapMessage = "cmd setupPostgres setupPostgres"
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.Wrap(err, errWrapMessage)
	}
	// Registers shopspringDecimal adapter to convert Numeric(15, 6) into decimal.Decimal
	// used in domain/entity. Runtime parser for pgx
	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		shopspringDecimal.Register(conn.TypeMap())
		return nil
	}
	pgxPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, errWrapMessage)
	}
	return pgxPool, nil
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

	postgresPool, err := setupPostgres(ctx, cfg.DBConfig.URL)
	if err != nil {
		log.Errorf("connect to db: %v", err)
		return
	}
	defer postgresPool.Close()

	userRepo := user.NewRepository(postgresPool, pkg.GetDB)
	balanceRepo := wallet.NewRepository(postgresPool, pkg.GetDB)
	pwdHasher := argon2.NewArgon2(&cfg.AuthConfig.Argon2Config)
	txManager := pkg.NewTXManager(postgresPool)
	jwtManager, err := jwt.NewManager(&cfg.AuthConfig.JWTConfig)
	if err != nil {
		log.Errorf("create new manager: %v", err)
		return
	}

	regUseCase := register.NewUseCase(userRepo, pwdHasher, balanceRepo, txManager)
	logUseCase := login.NewUseCase(userRepo, pwdHasher, jwtManager)

	deps := api.Dependencies{
		Logger:   log,
		Register: regUseCase,
		Login:    logUseCase,
	}

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, "pong")
	})

	apiRouter := router.Group("api")

	api.RegisterEndpoints(apiRouter, &deps)

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
