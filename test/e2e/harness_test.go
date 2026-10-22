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
	// deposits is what each user deposited, recorded by depositFunds for the invariants:
	// the database keeps only balances. It survives a restart, like the database.
	deposits map[uuid.UUID]funds

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
// When the test ends, it lets everything settle, checks the invariants and stops both.
func newHarness(t *testing.T) *harness {
	t.Helper()
	truncate(t)

	// Each harness gets its own copy, so a test can change a field for itself only.
	h := &harness{cfg: testCfg, deposits: make(map[uuid.UUID]funds)}
	h.build(t)
	h.startEngine()
	h.startSettlement()
	t.Cleanup(func() {
		// Deferred, so they stop even when a check below ends the test: they must not
		// keep running into the next one.
		defer h.stopSettlement(t)
		defer h.stopEngine(t)

		// Every test ends with the invariants checked on the settled state. A failed test
		// is skipped: its state is wrong anyway.
		if t.Failed() {
			return
		}
		if h.engineRun == nil {
			h.startEngine()
		}
		if h.settlementRun == nil {
			h.startSettlement()
		}
		h.waitSettled(t)
		h.checkInvariants(t)
	})
	return h
}

// restart simulates a crash and a restart of the app: everything in memory is lost (the
// book, the commands waiting in the command bus, the events waiting in the event bus) and
// the database stays. The components are built again on the same database and started
// the way cmd/app.go does: settlement first, then the book is loaded, then the engine.
func (h *harness) restart(t *testing.T) {
	t.Helper()
	h.stopEngine(t)
	h.stopSettlement(t)

	h.build(t)
	h.startSettlement()
	// Like gctx in the app, the load's context ends if settlement stops: the replay
	// would otherwise wait for an event bus nobody empties.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		select {
		case <-h.settlementRun.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	err := h.engine.LoadOrderBook(ctx)
	cancel()
	if err != nil {
		h.failIfStopped(t)
		t.Fatalf("load order book: %v", err)
	}
	h.startEngine()
}

// build wires new components on the shared database, as cmd/app.go does, and puts them
// in h, replacing any previous ones. Nothing is started.
func (h *harness) build(t *testing.T) {
	t.Helper()
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
	hasher := argon2.NewArgon2(&h.cfg.Argon2Config)

	commandBus := commandbus.NewCommandBus(&h.cfg.CommandBusConfig)
	eventBus := &trackedEventBus{EventBus: eventbus.NewEventBus(&h.cfg.EventBusConfig)}
	book := orderbook.NewBook(&h.cfg.OrderBookConfig, pair)

	h.pair = pair
	h.register = register.NewUseCase(userRepo, hasher, walletRepo, marketCache, txManager)
	h.deposit = deposit.NewUseCase(walletRepo, marketCache, converter)
	h.createOrder = createorder.NewUseCase(orderRepo, walletRepo, marketCache, txManager, converter, commandBus)
	h.cancelOrder = cancelorder.NewUseCase(orderRepo, commandBus)
	h.getPairBalance = getpairbalance.NewUseCase(walletRepo, marketCache)
	h.eventBus = eventBus
	h.engine = engine.NewMatchingEngine(book, commandBus, eventBus, orderRepo, logger)
	h.settlement = tradeservice.NewSettlementService(
		eventBus, tradeRepo, orderRepo, walletRepo, marketCache, txManager, logger,
	)
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

// stop cancels the component and waits for it to return. A component that had already
// returned by itself fails the test. Whatever it returns after the cancel is caused by
// it, so it is a clean stop: not always context.Canceled, since pgx reports a query cut
// short by the cancel as an i/o timeout. A nil runner (never started, or already stopped)
// does nothing.
func (r *runner) stop(t *testing.T, name string) {
	t.Helper()
	if r == nil {
		return
	}
	if err := r.stoppedWith(); err != nil {
		t.Errorf("%s stopped by itself: %v", name, err)
		return
	}
	r.cancel()
	<-r.done
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
