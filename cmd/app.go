package main

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/dixchange/Monolith/internal/auth/adapter/crypto/argon2"
	"github.com/dixchange/Monolith/internal/auth/adapter/jwt"
	authSQL "github.com/dixchange/Monolith/internal/auth/adapter/sql"
	"github.com/dixchange/Monolith/internal/config"
	api "github.com/dixchange/Monolith/internal/controller/http"
	createorder "github.com/dixchange/Monolith/internal/domain/usecase/create_order"
	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	"github.com/dixchange/Monolith/internal/domain/usecase/login"
	"github.com/dixchange/Monolith/internal/domain/usecase/register"
	marketSQL "github.com/dixchange/Monolith/internal/market/adapter/sql"
	"github.com/dixchange/Monolith/internal/market/service"
	bookwriter "github.com/dixchange/Monolith/internal/matching_engine/book_writer"
	"github.com/dixchange/Monolith/internal/matching_engine/engine"
	orderbook "github.com/dixchange/Monolith/internal/matching_engine/order_book"
	orderSQL "github.com/dixchange/Monolith/internal/order/adapter/sql"
	tradeSQL "github.com/dixchange/Monolith/internal/trade/adapter/sql"
	tradeservice "github.com/dixchange/Monolith/internal/trade/service"
	tradewriter "github.com/dixchange/Monolith/internal/trade/trade_writer"
	walletSQL "github.com/dixchange/Monolith/internal/wallet/adapter/sql"
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
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.Wrap(err, "pgxpool ParseConfig")
	}
	// Registers shopspringDecimal adapter to convert Numeric(15, 6) into decimal.Decimal
	// used in domain/entity. Runtime parser for pgx
	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		shopspringDecimal.Register(conn.TypeMap())
		return nil
	}
	pgxPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "pgxpool create")
	}
	return pgxPool, nil
}

//nolint:funlen // for main length is accepted; mainly consists of initializations
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logrusLogger := logrus.New()
	logrusLogger.SetFormatter(&logrus.JSONFormatter{})
	log := pkg.NewLogrusAdapter(logrusLogger)

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

	userRepo := authSQL.NewRepository(postgresPool, pkg.GetDB)
	balanceRepo := walletSQL.NewRepository(postgresPool, pkg.GetDB)
	marketRepo := marketSQL.NewRepository(postgresPool, pkg.GetDB)
	orderRepo := orderSQL.NewRepository(postgresPool, pkg.GetDB)
	tradeRepo := tradeSQL.NewRepository(postgresPool, pkg.GetDB)
	marketCache, err := service.NewDBCache(ctx, marketRepo)
	if err != nil {
		log.Errorf("create market db cache: %v", err)
		return
	}
	pwdHasher := argon2.NewArgon2(&cfg.Argon2Config)
	txManager := pkg.NewTXManager(postgresPool)
	jwtManager, err := jwt.NewManager(&cfg.JWTConfig)
	if err != nil {
		log.Errorf("create jwt manager: %v", err)
		return
	}
	converter := service.NewConverter()
	orderBook := orderbook.NewBook(&cfg.OrderBookConfig)
	bookWriter := bookwriter.NewBookWriter(&cfg.BookWriterConfig)
	tradeWriter := tradewriter.NewTradeWriter(&cfg.TradeWriterConfig)

	matchingEngine := engine.NewMatchingEngine(orderBook, bookWriter, tradeWriter, log)
	tradeService := tradeservice.NewTradeService(tradeWriter, tradeRepo, txManager, log)

	// Start matching engine and trade service before any traffic comes in
	go func() {
		matchingEngine.Run(ctx)
	}()
	go func() {
		tradeService.Run(ctx)
	}()

	regUseCase := register.NewUseCase(userRepo, pwdHasher, balanceRepo, marketCache, txManager)
	logUseCase := login.NewUseCase(userRepo, pwdHasher, jwtManager)
	depositUseCase := deposit.NewUseCase(balanceRepo, marketCache, converter)
	createOrderUseCase := createorder.NewUseCase(orderRepo, balanceRepo, marketCache, txManager, converter, bookWriter)

	deps := api.Dependencies{
		Logger:         log,
		TokenValidator: jwtManager,
		Register:       regUseCase,
		Login:          logUseCase,
		Deposit:        depositUseCase,
		CreateOrder:    createOrderUseCase,
	}

	router := gin.Default()
	api.SetCORSMiddleware(router, cfg.HTTPConfig.FrontendHost)

	api.RegisterEndpoints(router, &deps)

	srv := &http.Server{
		Addr:              cfg.HTTPConfig.Addr,
		ReadHeaderTimeout: time.Duration(cfg.HTTPConfig.ReadHeaderTimeoutSeconds) * time.Second,
		Handler:           router,
	}

	go func() {
		if err = srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server listen: %s\n", err)
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
