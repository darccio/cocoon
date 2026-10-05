package rt_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/darccio/cocoon/rt"
)

func TestPoolPanicDoesNotLoseSlot(t *testing.T) {
	t.Parallel()
	var created, destroyed atomic.Int32
	p, err := rt.NewPool(1, func() (*int, error) { created.Add(1); return new(int), nil }, func(_ *int) bool { return true }, func(_ *int) { destroyed.Add(1) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Error(err)
		}
	})
	func() {
		defer func() {
			if recover() != "caller" {
				t.Error("caller panic not preserved")
			}
		}()
		if err := p.Do(t.Context(), func(_ *int) error { panic("caller") }); err != nil {
			t.Error(err)
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := p.Do(ctx, func(_ *int) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if created.Load() != 2 || destroyed.Load() != 1 {
		t.Fatal("panicked instance was reused")
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if destroyed.Load() != 2 {
		t.Fatal("Close did not destroy retained instance")
	}
	if !errors.Is(p.Do(ctx, nil), rt.ErrClosed) {
		t.Fatal("pool Close is not terminal")
	}
}

func TestPoolCancellationAndFactoryFailure(t *testing.T) {
	t.Parallel()
	want := errors.New("factory failed")
	p, err := rt.NewPool[int](1, func() (*int, error) { return nil, want }, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if !errors.Is(p.Do(ctx, nil), context.Canceled) {
		t.Fatal("canceled admission accepted")
	}
	for range 2 {
		if !errors.Is(p.Do(t.Context(), nil), want) {
			t.Fatal("factory error lost slot")
		}
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := rt.NewPool[int](0, nil, nil, nil); err == nil {
		t.Fatal("invalid pool accepted")
	}
}

func TestPoolRecyclerPanic(t *testing.T) {
	t.Parallel()
	panics := true
	p, err := rt.NewPool[int](1, func() (*int, error) { return new(int), nil }, func(_ *int) bool {
		if panics {
			panic("recycler")
		}
		return false
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recover() != "recycler" {
				t.Error("recycler panic missing")
			}
		}()
		if err := p.Do(t.Context(), func(_ *int) error { return nil }); err != nil {
			t.Error(err)
		}
	}()
	panics = false
	if err := p.Do(t.Context(), func(_ *int) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
}

type cancelAfterBorrowContext struct {
	context.Context //nolint:containedctx // Test-only wrapper injects cancellation at the post-borrow check.
	cancel          context.CancelFunc
	checks          int
}

func (c *cancelAfterBorrowContext) Err() error {
	c.checks++
	if c.checks == 2 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestPoolCancellationAfterReadyBorrow(t *testing.T) {
	t.Parallel()
	var created int
	pool, err := rt.NewPool(1, func() (*int, error) { created++; return new(int), nil }, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	for range 2 {
		ctx, cancel := context.WithCancel(t.Context())
		if err := pool.Do(&cancelAfterBorrowContext{Context: ctx, cancel: cancel}, func(_ *int) error {
			t.Error("canceled borrow executed callback")
			return nil
		}); !errors.Is(err, context.Canceled) {
			t.Error("cancellation after borrow ignored", err)
		}
		cancel()
		if err := pool.Do(t.Context(), func(_ *int) error { return nil }); err != nil {
			t.Fatal("canceled borrow lost capacity", err)
		}
	}
	if created != 1 {
		t.Fatalf("canceled borrow created or discarded a guest: %d factories", created)
	}
}

type waitingPoolContext struct {
	context.Context //nolint:containedctx // Test-only wrapper signals entry into the blocking admission path.
	waiting         chan struct{}
	once            sync.Once
}

func (c *waitingPoolContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

func TestPoolUnavailableSlotWaitsForCancellation(t *testing.T) {
	t.Parallel()
	pool, err := rt.NewPool(1, func() (*int, error) { return new(int), nil }, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := pool.Close(); err != nil {
			t.Error(err)
		}
	})
	started, release := make(chan struct{}), make(chan struct{})
	releaseHeld := sync.OnceFunc(func() { close(release) })
	defer releaseHeld()
	held := make(chan error, 1)
	go func() {
		held <- pool.Do(t.Context(), func(_ *int) error {
			close(started)
			<-release
			return nil
		})
	}()
	<-started
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	waiter := &waitingPoolContext{Context: ctx, waiting: make(chan struct{})}
	result := make(chan error, 1)
	go func() {
		result <- pool.Do(waiter, func(_ *int) error {
			t.Error("waiting canceled callback executed")
			return nil
		})
	}()
	<-waiter.waiting
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal("blocked admission ignored cancellation", err)
	}
	releaseHeld()
	if err := <-held; err != nil {
		t.Fatal(err)
	}
	if err := pool.Do(t.Context(), func(_ *int) error { return nil }); err != nil {
		t.Fatal("waiting cancellation lost capacity", err)
	}
}
