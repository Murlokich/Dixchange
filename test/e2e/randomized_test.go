//go:build e2e

package e2e

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"

	"github.com/Murlokich/Dixchange/internal/domain/entity"
)

// The randomized suite sends a random sequence of creates and cancels from a few users,
// without waiting for the engine or settlement in between. Every few commands it crashes
// the app (see restart) or lets everything settle and checks the invariants: the trades
// aren't known in advance, so the invariants are what is checked. A failed run writes its
// steps to a log file.
//
// Each user trades on one side only (the engine doesn't prevent self-trades), and deposits
// enough to pay for every command at the largest size, so a create can't run out of funds.
// A cancel picks one of the user's active orders as they are in the orders table at that
// moment: create returns no order id.
//
// The seed reproduces the users' sides and the commands: their users, kinds, quantities and
// prices. Which order a cancel picks also depends on what was settled by then, so it can
// differ between runs.

const (
	randomMaxQuantity = 5  // USD, from 1
	randomMinPrice    = 5  // KCN
	randomMaxPrice    = 15 // KCN
	// randomCancelShare in randomShareBase of the commands are cancels.
	randomCancelShare = 3
	randomShareBase   = 10
)

// randomSettings configures the suite; make test-e2e loads them from envs/tests/e2e.env.
type randomSettings struct {
	Users         int    `required:"true"`
	Commands      int    `required:"true"`
	ValidateEvery int    `required:"true" split_words:"true"`
	CrashEvery    int    `required:"true" split_words:"true"` // 0 disables the crashes
	LogDir        string `required:"true" split_words:"true"`
	Seed          uint64 // 0 picks one
}

// Validate requires at least one user and one command.
func (s *randomSettings) Validate() error {
	switch {
	case s.Users < 1:
		return errors.Errorf("users must be positive, got %d", s.Users)
	case s.Commands < 1:
		return errors.Errorf("commands must be positive, got %d", s.Commands)
	case s.ValidateEvery < 1:
		return errors.Errorf("validate every must be positive, got %d", s.ValidateEvery)
	case s.CrashEvery < 0:
		return errors.Errorf("crash every must not be negative, got %d", s.CrashEvery)
	}
	return nil
}

func TestRandomized_Sequence(t *testing.T) {
	var settings randomSettings
	if err := envconfig.Process("e2e_random", &settings); err != nil {
		t.Fatalf("read the settings (run the tests with make test-e2e): %v", err)
	}
	if err := settings.Validate(); err != nil {
		t.Fatalf("invalid settings: %v", err)
	}
	if settings.Seed == 0 {
		settings.Seed = uint64(time.Now().UnixNano())
	}
	t.Logf("seed %d", settings.Seed)

	r := newRandomRun(t, settings)
	for step := 1; step <= settings.Commands; step++ {
		r.command(step)
		if settings.CrashEvery > 0 && step%settings.CrashEvery == 0 {
			r.log.addf("%04d CRASH", step)
			r.h.restart(t)
		}
		if step%settings.ValidateEvery == 0 || step == settings.Commands {
			r.validate(step)
		}
	}
}

type randomUser struct {
	id   uuid.UUID
	side string
}

// randomRun is one run of the suite.
type randomRun struct {
	t        *testing.T
	h        *harness
	rng      *rand.Rand
	settings randomSettings
	users    []randomUser
	log      *randomLog
}

// newRandomRun creates the users and arranges for the log to be written if the run fails.
// Each user picks a side at random. All users can end up on one side, with no trades: rare
// with a few users, and another run picks again.
func newRandomRun(t *testing.T, settings randomSettings) *randomRun {
	t.Helper()
	r := &randomRun{
		t:        t,
		h:        newHarness(t),
		rng:      rand.New(rand.NewPCG(settings.Seed, 0)),
		settings: settings,
		log:      &randomLog{},
	}
	// Registered after newHarness, so it runs before the harness stops everything.
	t.Cleanup(func() {
		if t.Failed() {
			r.log.write(t, settings)
		}
	})

	r.log.addf("seed=%d users=%d commands=%d validate_every=%d crash_every=%d",
		settings.Seed, settings.Users, settings.Commands, settings.ValidateEvery, settings.CrashEvery)
	// Enough for every command at the largest size: sellers give base, buyers pay quote.
	baseDeposit := strconv.Itoa(settings.Commands * randomMaxQuantity)
	quoteDeposit := strconv.Itoa(settings.Commands * randomMaxQuantity * randomMaxPrice)
	for i := range settings.Users {
		side := entity.BuySide
		var id uuid.UUID
		if r.rng.IntN(2) == 0 {
			id = r.h.newUser(t, "", quoteDeposit)
		} else {
			side = entity.SellSide
			id = r.h.newUser(t, baseDeposit, "")
		}
		r.users = append(r.users, randomUser{id: id, side: side})
		r.log.addf("user %d: %s %s", i+1, id, side)
	}
	return r
}

// command sends one random command. Every call draws the same values, so the seed
// reproduces the sequence even where a cancel has nothing to cancel.
func (r *randomRun) command(step int) {
	userIndex := r.rng.IntN(len(r.users))
	cancel := r.rng.IntN(randomShareBase) < randomCancelShare
	quantity := 1 + r.rng.IntN(randomMaxQuantity)
	price := randomMinPrice + r.rng.IntN(randomMaxPrice-randomMinPrice+1)
	pick := r.rng.Float64()

	if cancel && r.cancel(step, userIndex, pick) {
		return
	}
	r.create(step, userIndex, quantity, price)
}

// create places an order; it can't fail: the user has funds for every command.
func (r *randomRun) create(step, userIndex, quantity, price int) {
	user := r.users[userIndex]
	r.log.addf("%04d CREATE   user=%d side=%s qty=%d price=%d", step, userIndex+1, user.side, quantity, price)
	err := r.h.placeOrder(r.t, user.id, user.side, strconv.Itoa(quantity), strconv.Itoa(price))
	if err != nil {
		r.t.Fatalf("step %d: create: %v", step, err)
	}
}

// cancel cancels one of the user's active orders, picked by pick in [0, 1), and reports
// whether there was one. The order may have been filled since it was read, so "not
// cancellable" is a valid outcome too.
func (r *randomRun) cancel(step, userIndex int, pick float64) bool {
	user := r.users[userIndex]
	var active []uuid.UUID
	for _, o := range r.h.orders(r.t, user.id) {
		if entity.IsActiveOrderStatus(o.Status) {
			active = append(active, o.ID)
		}
	}
	if len(active) == 0 {
		return false
	}

	// Not r.rng.IntN(len(active)): len(active) depends on what was settled by now, which
	// differs between runs, and IntN may draw a different amount of randomness for a
	// different n, which would change every command after this one. pick is drawn every
	// step, so the sequence stays the same; only the order it maps to can differ.
	orderID := active[int(pick*float64(len(active)))]
	line := fmt.Sprintf("%04d CANCEL   user=%d order=%s", step, userIndex+1, orderID)
	err := r.h.cancel(r.t, user.id, orderID)
	switch {
	case err == nil:
		r.log.addf("%s -> cancelled", line)
	case errors.Is(err, entity.ErrOrderNotCancellable):
		r.log.addf("%s -> not cancellable", line)
	default:
		r.log.addf("%s", line)
		r.t.Fatalf("step %d: cancel: %v", step, err)
	}
	return true
}

// validate lets everything settle and checks the invariants; a broken state ends the run.
func (r *randomRun) validate(step int) {
	r.log.addf("%04d VALIDATE", step)
	r.h.waitSettled(r.t)
	r.h.checkInvariants(r.t)
	if r.t.Failed() {
		r.t.FailNow()
	}
}

// randomLog keeps the steps of a run in memory; only a failed run writes them to a file.
type randomLog struct {
	lines []string
}

func (l *randomLog) addf(format string, args ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

func (l *randomLog) write(t *testing.T, settings randomSettings) {
	t.Helper()
	if err := os.MkdirAll(settings.LogDir, 0o750); err != nil {
		t.Errorf("create the log directory: %v", err)
		return
	}
	path := filepath.Join(settings.LogDir, fmt.Sprintf("randomized-%d.log", settings.Seed))
	if err := os.WriteFile(path, []byte(strings.Join(l.lines, "\n")+"\n"), 0o600); err != nil {
		t.Errorf("write the log: %v", err)
		return
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	t.Logf("log of the run: %s (replay with E2E_RANDOM_SEED=%d)", path, settings.Seed)
}
