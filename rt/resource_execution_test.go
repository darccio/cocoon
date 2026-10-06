package rt_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/darccio/cocoon/rt"
)

func newCallResource(t *testing.T, instance *rt.Instance, handle uint64) *rt.Resource[int] {
	t.Helper()
	resource, err := rt.NewResource[int](instance, handle, func(uint64) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := resource.Close(); err != nil {
			t.Error(err)
		}
	})
	return resource
}

func callOwned(resource *rt.Resource[int], op string, invoke func(*rt.Call) error) error {
	return resource.Use(op, func(instance *rt.Instance, _ uint64) error {
		return instance.Call(op, invoke)
	})
}

func receiveResourceCall[T any](ctx context.Context, t *testing.T, result <-chan T) T {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-ctx.Done():
		t.Fatal("resource use did not complete:", ctx.Err())
		var zero T
		return zero
	}
}

func TestResourceUseFullWidthInterleavedOwnership(t *testing.T) {
	t.Parallel()
	instance := newInstance(t, newModule())
	const firstHandle uint64 = 0xfedcba98_89abcdef
	const secondHandle uint64 = 0x12345678_76543210
	first := newCallResource(t, instance, firstHandle)
	second := newCallResource(t, instance, secondHandle)
	for _, test := range []struct {
		resource *rt.Resource[int]
		handle   uint64
	}{
		{first, firstHandle},
		{second, secondHandle},
		{first, firstHandle},
		{second, secondHandle},
	} {
		if err := test.resource.Use("token", func(owner *rt.Instance, handle uint64) error {
			if owner != instance || handle != test.handle {
				t.Fatalf("interleaved ownership changed: handle=%x, want=%x", handle, test.handle)
			}
			return owner.Call("token", func(*rt.Call) error { return nil })
		}); err != nil {
			t.Fatal(err)
		}
		if err := instance.Call("plain", func(*rt.Call) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResourceUseNestedGuestFailuresReleaseOwnership(t *testing.T) {
	t.Parallel()
	application := &rt.AppError{Op: "expected", Status: rt.ErrApp, Message: "application"}
	for _, test := range []struct {
		invoke func(*rt.Call) error
		want   error
		name   string
		poison bool
	}{
		{func(*rt.Call) error { return application }, application, "application", false},
		{func(*rt.Call) error { return rt.ErrProtocol }, rt.ErrProtocol, "protocol", true},
		{func(*rt.Call) error { panic("unreachable") }, nil, "panic", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			instance := newInstance(t, newModule())
			resource := newCallResource(t, instance, 7)
			err := callOwned(resource, "nested", test.invoke)
			if test.want == nil {
				var fault *rt.FaultError
				if !errors.As(err, &fault) || fault.Op != "nested" || fault.Kind != rt.FaultUnreachable {
					t.Fatal("nested callback lost its contained guest fault", err)
				}
			} else if !errors.Is(err, test.want) {
				t.Fatal("resource use lost the guest error", err)
			}
			if instance.Healthy() == test.poison {
				t.Fatal("nested callback changed poison semantics")
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				done <- resource.Use("reuse", func(*rt.Instance, uint64) error {
					if test.poison {
						t.Error("invalidated ownership invoked callback")
					}
					return nil
				})
			}()
			err = receiveResourceCall(ctx, t, done)
			if test.poison {
				var handle *rt.HandleError
				if !errors.As(err, &handle) {
					t.Fatal("nested fault did not invalidate ownership", err)
				}
			} else if err != nil {
				t.Fatal("expected guest error prevented reuse", err)
			}
		})
	}
}

func TestResourceUseNestedCallAndEventualCleanup(t *testing.T) {
	t.Parallel()
	instance := newInstance(t, newModule())
	var destroyed atomic.Int32
	done := make(chan struct{}, 1)
	resource, err := rt.NewResource[int](instance, 7, func(uint64) error {
		destroyed.Add(1)
		done <- struct{}{}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := callOwned(resource, "gc", func(*rt.Call) error {
		resource = nil
		for range 3 {
			runtime.GC()
			runtime.Gosched()
			if destroyed.Load() != 0 {
				t.Error("cleanup destroyed ownership during nested guest execution")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		runtime.GC()
		runtime.Gosched()
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	receiveResourceCall(ctx, t, done)
	if destroyed.Load() != 1 {
		t.Fatal("unreachable ownership was not cleaned up exactly once")
	}
}

func TestResourceUseAliasCloseWaitsAndSharesResult(t *testing.T) {
	t.Parallel()
	instance := newInstance(t, newModule())
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	started, release := make(chan struct{}), make(chan struct{})
	releaseUse := sync.OnceFunc(func() { close(release) })
	defer releaseUse()
	var active atomic.Bool
	var destroyed atomic.Int32
	want := errors.New("destructor result")
	resource, err := rt.NewResource[int](instance, 7, func(handle uint64) error {
		if handle != 7 || active.Load() {
			t.Error("destructor overlapped active use or changed its handle")
		}
		destroyed.Add(1)
		return want
	})
	if err != nil {
		t.Fatal(err)
	}
	alias := resource
	used := make(chan error, 1)
	go func() {
		used <- callOwned(resource, "held", func(*rt.Call) error {
			active.Store(true)
			defer active.Store(false)
			close(started)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	receiveResourceCall(ctx, t, started)
	closing := make(chan struct{}, 8)
	closed := make(chan error, 8)
	for range 8 {
		go func() {
			closing <- struct{}{}
			closed <- alias.Close()
		}()
	}
	for range 8 {
		receiveResourceCall(ctx, t, closing)
	}
	select {
	case err := <-closed:
		t.Fatal("Close completed before active use was released", err)
	default:
	}
	if destroyed.Load() != 0 {
		t.Fatal("Close destroyed an active resource")
	}
	releaseUse()
	if err := receiveResourceCall(ctx, t, used); err != nil {
		t.Fatal(err)
	}
	for range 8 {
		if err := receiveResourceCall(ctx, t, closed); err != want { //nolint:errorlint // Every alias must receive the exact once-only destructor result.
			t.Fatal("alias lost the shared destructor result", err)
		}
	}
	if destroyed.Load() != 1 {
		t.Fatal("aliases invoked the destructor more than once")
	}
	var handle *rt.HandleError
	if err := resource.Use("closed", nil); !errors.As(err, &handle) {
		t.Fatal("closed alias remained usable", err)
	}
}

func TestResourceUseSerializesSharedInstance(t *testing.T) {
	t.Parallel()
	instance := newInstance(t, newModule())
	first := newCallResource(t, instance, 7)
	second := newCallResource(t, instance, 9)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	started, release := make(chan struct{}), make(chan struct{})
	releaseUse := sync.OnceFunc(func() { close(release) })
	defer releaseUse()
	var active, calls atomic.Int32
	done := make(chan error, 13)
	go func() {
		done <- callOwned(first, "held", func(*rt.Call) error {
			active.Add(1)
			defer active.Add(-1)
			calls.Add(1)
			close(started)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	receiveResourceCall(ctx, t, started)
	ready := make(chan struct{}, 12)
	for index := range 12 {
		go func() {
			invoke := instance.Call
			if index%3 != 2 {
				resource := first
				if index%3 == 1 {
					resource = second
				}
				invoke = func(op string, callback func(*rt.Call) error) error {
					return callOwned(resource, op, callback)
				}
			}
			ready <- struct{}{}
			done <- invoke("shared", func(*rt.Call) error {
				if active.Add(1) != 1 {
					t.Error("guest execution callbacks overlapped")
				}
				defer active.Add(-1)
				calls.Add(1)
				runtime.Gosched()
				return nil
			})
		}()
	}
	for range 12 {
		receiveResourceCall(ctx, t, ready)
	}
	if calls.Load() != 1 {
		t.Fatal("another callback entered a locked instance")
	}
	releaseUse()
	for range 13 {
		if err := receiveResourceCall(ctx, t, done); err != nil {
			t.Fatal(err)
		}
	}
	if calls.Load() != 13 || active.Load() != 0 {
		t.Fatal("serialized calls were lost or remained active")
	}
}
