//go:build e2e

package e2e

import (
	"context"
	"os"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/dixchange/Monolith/internal/auth/adapter/crypto/argon2"
	authSQL "github.com/dixchange/Monolith/internal/auth/adapter/sql"
	"github.com/dixchange/Monolith/internal/config"
	"github.com/dixchange/Monolith/internal/domain/entity"
	cancelorder "github.com/dixchange/Monolith/internal/domain/usecase/cancel_order"
	createorder "github.com/dixchange/Monolith/internal/domain/usecase/create_order"
	"github.com/dixchange/Monolith/internal/domain/usecase/deposit"
	getpairbalance "github.com/dixchange/Monolith/internal/domain/usecase/get_pair_balance"
	"github.com/dixchange/Monolith/internal/domain/usecase/register"
	marketSQL "github.com/dixchange/Monolith/internal/market/adapter/sql"
	"github.com/dixchange/Monolith/internal/market/service"
	commandbus "github.com/dixchange/Monolith/internal/matching_engine/command_bus"
	"github.com/dixchange/Monolith/internal/matching_engine/engine"
	eventbus "github.com/dixchange/Monolith/internal/matching_engine/event_bus"
	orderbook "github.com/dixchange/Monolith/internal/matching_engine/order_book"
	orderSQL "github.com/dixchange/Monolith/internal/order/adapter/sql"
	tradeSQL "github.com/dixchange/Monolith/internal/trade/adapter/sql"
	tradeservice "github.com/dixchange/Monolith/internal/trade/service"
	walletSQL "github.com/dixchange/Monolith/internal/wallet/adapter/sql"
	"github.com/dixchange/Monolith/pkg"
)

// pairID is the only pair the migrations seed and the only book the app runs.
const pairID = 1

// harness is the application wired as in cmd/app.go, minus HTTP and JWT: tests call the
// use cases directly. The engine and settlement run in the background and can be stopped
// and started again, so a test can hold commands or events back to set up a race.
type harness struct {
	// cfg is this harness's copy of testCfg, the config its components were built with.
	cfg  config.Config
	pair entity.Pair

	register       *register.UseCase
	deposit        *deposit.UseCase
	createOrder    *createorder.UseCase
	cancelOrder    *cancelorder.UseCase
	getPairBalance *getpairbalance.UseCase

	eventBus   *trackedEventBus
	engine     *engine.MatchingEngine
	settlement *tradeservice.SettlementService

	engineRun     *runner
	settlementRun *runner
}

// newHarness empties the database and starts the engine and settlement on an empty book.
// Both are stopped when the test ends.
func newHarness(t *testing.T) *harness {
	t.Helper()
	truncate(t)
	ctx := context.Background()

	logrusLogger := logrus.New()
	logrusLogger.SetOutput(os.Stderr)
	logrusLogger.SetLevel(logrus.WarnLevel)
	logger := pkg.NewLogrusAdapter(logrusLogger)

	userRepo := authSQL.NewRepository(testDB, pkg.GetDB)
	walletRepo := walletSQL.NewRepository(testDB, pkg.GetDB)
	orderRepo := orderSQL.NewRepository(testDB, pkg.GetDB)
	tradeRepo := tradeSQL.NewRepository(testDB, pkg.GetDB)
	marketCache, err := service.NewDBCache(ctx, marketSQL.NewRepository(testDB, pkg.GetDB))
	if err != nil {
		t.Fatalf("create market cache: %v", err)
	}
	pair, ok := marketCache.GetPair(pairID)
	if !ok {
		t.Fatalf("pair %d isn't seeded", pairID)
	}
	txManager := pkg.NewTXManager(testDB)
	converter := service.NewConverter()
	// Each harness gets its own copy, so a test can change a field for itself only.
	cfg := testCfg
	hasher := argon2.NewArgon2(&cfg.Argon2Config)

	commandBus := commandbus.NewCommandBus(&cfg.CommandBusConfig)
	eventBus := &trackedEventBus{EventBus: eventbus.NewEventBus(&cfg.EventBusConfig)}
	book := orderbook.NewBook(&cfg.OrderBookConfig, pair)

	h := &harness{
		cfg:            cfg,
		pair:           pair,
		register:       register.NewUseCase(userRepo, hasher, walletRepo, marketCache, txManager),
		deposit:        deposit.NewUseCase(walletRepo, marketCache, converter),
		createOrder:    createorder.NewUseCase(orderRepo, walletRepo, marketCache, txManager, converter, commandBus),
		cancelOrder:    cancelorder.NewUseCase(orderRepo, commandBus),
		getPairBalance: getpairbalance.NewUseCase(walletRepo, marketCache),
		eventBus:       eventBus,
		engine:         engine.NewMatchingEngine(book, commandBus, eventBus, orderRepo, logger),
		settlement: tradeservice.NewSettlementService(
			eventBus, tradeRepo, orderRepo, walletRepo, marketCache, txManager, logger,
		),
	}
	h.startEngine()
	h.startSettlement()
	t.Cleanup(func() {
		h.stopEngine(t)
		h.stopSettlement(t)
	})
	return h
}

// startEngine starts the engine; it must be stopped. Commands sent while it is stopped wait
// in the command bus.
func (h *harness) startEngine() {
	h.engineRun = startRunner(h.engine.Run)
}

// stopEngine stops the engine if it runs, failing the test if it stopped with an error.
func (h *harness) stopEngine(t *testing.T) {
	t.Helper()
	h.engineRun.stop(t, "engine")
	h.engineRun = nil
}

// startSettlement starts settlement; it must be stopped. Events published while it is
// stopped wait in the event bus.
func (h *harness) startSettlement() {
	h.settlementRun = startRunner(h.settlement.Run)
}

// stopSettlement stops settlement if it runs, failing the test if it stopped with an error.
func (h *harness) stopSettlement(t *testing.T) {
	t.Helper()
	h.settlementRun.stop(t, "settlement")
	h.settlementRun = nil
}

// failIfStopped fails the test at once if the engine or settlement has stopped on its own:
// in the app that error would crash the process.
func (h *harness) failIfStopped(t *testing.T) {
	t.Helper()
	if err := h.engineRun.stoppedWith(); err != nil {
		t.Fatalf("engine stopped: %v", err)
	}
	if err := h.settlementRun.stoppedWith(); err != nil {
		t.Fatalf("settlement stopped: %v", err)
	}
}

// runner runs one background component (the engine or settlement) until stopped.
type runner struct {
	cancel context.CancelFunc
	done   chan struct{}
	// err is what the component returned; read it only after done is closed.
	err error
}

func startRunner(run func(ctx context.Context) error) *runner {
	ctx, cancel := context.WithCancel(context.Background())
	r := &runner{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(r.done)
		r.err = run(ctx)
	}()
	return r
}

// stop cancels the component and waits for it to return. Returning because of the cancel
// is a clean stop; any other error fails the test. A nil runner (never started, or
// already stopped) does nothing.
func (r *runner) stop(t *testing.T, name string) {
	t.Helper()
	if r == nil {
		return
	}
	r.cancel()
	<-r.done
	if r.err != nil && !errors.Is(r.err, context.Canceled) {
		t.Errorf("%s stopped with error: %v", name, r.err)
	}
}

// stoppedWith returns the error the component stopped with, or nil while it still runs.
func (r *runner) stoppedWith() error {
	if r == nil {
		return nil
	}
	select {
	case <-r.done:
		if r.err == nil {
			return errors.New("returned without error")
		}
		return r.err
	default:
		return nil
	}
}

// trackedEventBus counts the events published by the engine and the events settlement has
// finished, so waitSettled can tell when settlement is idle. Settlement asks for the next
// event only after finishing the previous one, so a Consume call while holding an event
// means that event is finished.
type trackedEventBus struct {
	*eventbus.EventBus

	published atomic.Int64
	finished  atomic.Int64
	// holding is true while settlement processes an event. Only the settlement goroutine
	// touches it, and a restarted settlement starts after the previous one returned.
	holding bool
}

// Each publish is counted before the event is sent, so the count never lags behind the
// events settlement can already see.

func (b *trackedEventBus) PublishOrderPlaced(ctx context.Context, commandID uuid.UUID, trades []entity.Trade) error {
	b.published.Add(1)
	return b.uncountIfFailed(b.EventBus.PublishOrderPlaced(ctx, commandID, trades))
}

func (b *trackedEventBus) PublishOrderCancelled(ctx context.Context, commandID uuid.UUID) error {
	b.published.Add(1)
	return b.uncountIfFailed(b.EventBus.PublishOrderCancelled(ctx, commandID))
}

func (b *trackedEventBus) PublishCancelRejected(ctx context.Context, commandID uuid.UUID) error {
	b.published.Add(1)
	return b.uncountIfFailed(b.EventBus.PublishCancelRejected(ctx, commandID))
}

// uncountIfFailed takes back the count of an event that wasn't published.
func (b *trackedEventBus) uncountIfFailed(err error) error {
	if err != nil {
		b.published.Add(-1)
	}
	return err
}

func (b *trackedEventBus) Consume(ctx context.Context) (eventbus.Event, error) {
	if b.holding {
		b.finished.Add(1)
		b.holding = false
	}
	event, err := b.EventBus.Consume(ctx)
	b.holding = err == nil
	return event, err
}

// idle reports whether settlement has finished every event published so far.
func (b *trackedEventBus) idle() bool {
	return b.published.Load() == b.finished.Load()
}
