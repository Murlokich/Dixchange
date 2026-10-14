//go:build integration

// Package integration tests the application with a real Postgres: the use cases, the
// matching engine and settlement are wired as in cmd/app.go, but the tests control when
// the engine and settlement run and can read the database directly.
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	shopspringDecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// migrationsURL is relative to this package: go test runs with the package directory as
// the working directory.
const migrationsURL = "file://../../migrations"

// testDB is shared by every test of the package. The tests share one order book table
// and the market data, so they must not run in parallel.
var testDB *pgxpool.Pool

func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(code)
}

// run lives apart from TestMain so its deferred cleanup runs before os.Exit.
func run(m *testing.M) (int, error) {
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:18",
		postgres.WithDatabase("monolith"),
		postgres.WithUsername("developer"),
		postgres.WithPassword("password"),
		postgres.BasicWaitStrategies(),
	)
	defer func() {
		if termErr := testcontainers.TerminateContainer(container); termErr != nil {
			fmt.Fprintln(os.Stderr, "terminate postgres container:", termErr)
		}
	}()
	if err != nil {
		return 0, errors.Wrap(err, "start postgres container")
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return 0, errors.Wrap(err, "postgres connection string")
	}
	if err = migrateUp(dsn); err != nil {
		return 0, errors.Wrap(err, "migrate")
	}

	testDB, err = newPool(ctx, dsn)
	if err != nil {
		return 0, errors.Wrap(err, "connect to postgres")
	}
	defer testDB.Close()

	return m.Run(), nil
}

func migrateUp(dsn string) error {
	m, err := migrate.New(migrationsURL, dsn)
	if err != nil {
		return errors.Wrap(err, "create migration")
	}
	if err = m.Up(); err != nil {
		return errors.Wrap(err, "migrate up")
	}
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		return errors.Wrap(srcErr, "close migration source")
	}
	return errors.Wrap(dbErr, "close migration database")
}

// newPool connects the way cmd/app.go does.
func newPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, errors.Wrap(err, "parse config")
	}
	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		shopspringDecimal.Register(conn.TypeMap())
		return nil
	}
	return pgxpool.NewWithConfig(ctx, cfg)
}

// truncate removes everything the tests create. The market schema (symbols and pairs) is
// seeded by the migrations and stays.
func truncate(t *testing.T) {
	t.Helper()

	_, err := testDB.Exec(context.Background(), `TRUNCATE
		trade.trade,
		"order"."order",
		"order".command_log_usd_kcn,
		wallet.balance,
		auth.user`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}
