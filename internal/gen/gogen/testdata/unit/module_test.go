package unitfixture

import (
	"errors"
	"math"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/darccio/cocoon/rt"
)

func fixtureLibrary(t *testing.T) *Library {
	t.Helper()
	library, err := Open(Options{Instances: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := library.Close(); err != nil {
			t.Error(err)
		}
	})
	return library
}

func fixtureItem(t *testing.T, library *Library) (*Item, uint64) {
	t.Helper()
	item, err := library.NewItem()
	if err != nil {
		t.Fatal(err)
	}
	var handle uint64
	if err := item.owner.Use("fixture", func(_ *rt.Instance, value uint64) error {
		handle = value
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return item, handle
}

func fixtureAlias(t *testing.T, item *Item) *Item {
	t.Helper()
	// Reflection models a copy without triggering the deliberate NoCopy vet diagnostic.
	value := reflect.New(reflect.TypeOf(item).Elem())
	value.Elem().Set(reflect.ValueOf(item).Elem())
	alias, ok := value.Interface().(*Item)
	if !ok {
		t.Fatal("copied wrapper changed type")
	}
	return alias
}

func awaitFixtureSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-signal:
	case <-timer.C:
		t.Fatal("fixture operation did not start")
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
}

func awaitFixtureResult(t *testing.T, result <-chan error) error {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case err := <-result:
		return err
	case <-timer.C:
		t.Fatal("fixture operation did not finish")
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	return nil
}

func awaitFixtureClosing(t *testing.T, library *Library) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		err := library.life.Enter()
		if errors.Is(err, rt.ErrClosed) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		library.life.Leave()
		select {
		case <-timer.C:
			t.Fatal("library did not enter terminal admission")
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		default:
			runtime.Gosched()
		}
	}
}

func TestGeneratedLibraryCloseDrainsResourceCall(t *testing.T) {
	t.Parallel()
	library := fixtureLibrary(t)
	item, handle := fixtureItem(t, library)
	alias := fixtureAlias(t, item)
	guest := item.guest
	module := guest.module
	started, release := make(chan struct{}), make(chan struct{})
	module.addStarted, module.addRelease = started, release
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	// Also release on a failing assertion, before the library cleanup drains.
	t.Cleanup(unblock)
	addDone := make(chan error, 1)
	go func() { addDone <- item.Add() }()
	awaitFixtureSignal(t, started)

	closeDone := make(chan error, 1)
	go func() { closeDone <- library.Close() }()
	awaitFixtureClosing(t, library)
	concurrentCloseDone := make(chan error, 1)
	go func() { concurrentCloseDone <- library.Close() }()
	for _, done := range []<-chan error{closeDone, concurrentCloseDone} {
		select {
		case err := <-done:
			t.Fatal("library closed while an admitted resource call was active", err)
		default:
		}
	}
	for _, invoke := range []func() error{
		item.Add,
		alias.Add,
		func() error { _, err := library.NewItem(); return err },
		func() error { return library.Unit(t.Context()) },
	} {
		result := make(chan error, 1)
		go func() { result <- invoke() }()
		if err := awaitFixtureResult(t, result); !errors.Is(err, rt.ErrClosed) {
			t.Fatal("terminal admission accepted a generated call", err)
		}
	}

	unblock()
	if err := awaitFixtureResult(t, addDone); err != nil {
		t.Fatal("shutdown interrupted an admitted resource call", err)
	}
	for _, done := range []<-chan error{closeDone, concurrentCloseDone} {
		if err := awaitFixtureResult(t, done); err != nil {
			t.Fatal(err)
		}
	}
	if module.addCalls != 1 || module.lastAddHandle != handle || guest.instance.Healthy() || guest.module != nil || library.home != nil {
		t.Fatal("shutdown did not drain exactly one call and release its home")
	}
	epoch := guest.instance.Epoch()
	if err := library.Close(); err != nil || guest.instance.Epoch() != epoch {
		t.Fatal("repeated library close released the home again", err)
	}
	closeErr := item.Close()
	if !errors.Is(closeErr, rt.ErrClosed) || alias.Close() != closeErr || module.closeCalls[handle] != 0 {
		t.Fatal("copied resource destructor did not retain its once-only closed-home result", closeErr)
	}
}

func TestGeneratedDestructorCanonicalAndExpectedErrors(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		reply fixtureReply
	}{
		{"empty", fixtureReply{pointer: 16}},
		{"empty at memory end", fixtureReply{pointer: 64}},
		{"application", fixtureReply{pointer: 16, size: 3, status: rt.ErrApp, message: "bad"}},
		{"argument", fixtureReply{pointer: 16, size: 3, status: rt.ErrArg, message: "bad"}},
		{"handle", fixtureReply{pointer: 16, size: 3, status: rt.ErrHandle, message: "bad"}},
		{"limit", fixtureReply{pointer: 16, size: 3, status: rt.ErrLimit, message: "bad"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			library := fixtureLibrary(t)
			item, handle := fixtureItem(t, library)
			sibling, siblingHandle := fixtureItem(t, library)
			if handle == siblingHandle {
				t.Fatal("mock constructor reused a live handle")
			}
			guest := item.guest
			epoch := guest.instance.Epoch()
			guest.module.closeReply = test.reply
			closeErr := item.Close()
			assertExpectedCloseError(t, closeErr, test.reply.status)
			if !guest.instance.Healthy() || guest.instance.Epoch() != epoch {
				t.Fatal("expected destructor reply poisoned its home")
			}
			if repeated := item.Close(); repeated != closeErr || guest.module.closeCalls[handle] != 1 {
				t.Fatal("destructor did not retain its once-only result", repeated)
			}
			clear(guest.module.memory[16:])
			assertExpectedCloseError(t, closeErr, test.reply.status)
			if err := sibling.Add(); err != nil || guest.module.addCalls != 1 {
				t.Fatal("expected close error invalidated another resource", err)
			}
			fresh, _ := fixtureItem(t, library)
			if fresh.guest != guest {
				t.Fatal("healthy home was replaced")
			}
			guest.module.closeReply = fixtureReply{pointer: 16}
			for _, resource := range []*Item{sibling, fresh} {
				if err := resource.Close(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func assertExpectedCloseError(t *testing.T, err error, status rt.Status) {
	t.Helper()
	switch status {
	case rt.OK:
		if err != nil {
			t.Fatal(err)
		}
	case rt.ErrApp, rt.ErrArg:
		var application *rt.AppError
		if !errors.As(err, &application) || application.Op != "close" || application.Status != status || application.Message != "bad" {
			t.Fatal("destructor lost its owned application error", err)
		}
	case rt.ErrHandle:
		var handle *rt.HandleError
		if !errors.As(err, &handle) || handle.Op != "close" || handle.Reason != "bad" {
			t.Fatal("destructor lost its owned handle error", err)
		}
	case rt.ErrLimit:
		if !errors.Is(err, rt.ErrTooLarge) || err.Error() != "close: cocoon: size limit exceeded: bad" {
			t.Fatal("destructor lost its owned limit error", err)
		}
	case rt.Pending:
		t.Fatal("reserved status is not an expected close error")
	default:
		t.Fatal("unsupported fixture status", status)
	}
}

func TestGeneratedDestructorMalformedReplies(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		reply fixtureReply
		want  error
	}{
		{"nonempty success", fixtureReply{pointer: 16, size: 1, message: "x"}, rt.ErrProtocol},
		{"empty beyond memory", fixtureReply{pointer: 65}, rt.ErrProtocol},
		{"payload beyond memory", fixtureReply{pointer: 63, size: 2}, rt.ErrProtocol},
		{"payload overflow", fixtureReply{pointer: math.MaxUint32, size: 2}, rt.ErrProtocol},
		{"descriptor beyond logical memory", fixtureReply{pointer: 16, shrinkDescriptor: true}, rt.ErrProtocol},
		{"reserved status", fixtureReply{pointer: 16, status: rt.Pending}, rt.ErrProtocol},
		{"unknown status", fixtureReply{pointer: 16, status: rt.Status(6)}, rt.ErrProtocol},
		{"negative status", fixtureReply{pointer: 16, status: rt.Status(-1)}, rt.ErrProtocol},
		{"expected status with invalid payload", fixtureReply{pointer: 65, status: rt.ErrApp}, rt.ErrProtocol},
		{"output limit before bounds and status", fixtureReply{pointer: math.MaxUint32, size: 33, status: rt.Pending}, rt.ErrTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			library := fixtureLibrary(t)
			item, handle := fixtureItem(t, library)
			sibling, _ := fixtureItem(t, library)
			guest := item.guest
			module := guest.module
			epoch := guest.instance.Epoch()
			module.closeReply = test.reply
			closeErr := item.Close()
			if !errors.Is(closeErr, test.want) {
				t.Fatal("malformed destructor reply accepted", closeErr)
			}
			if repeated := item.Close(); repeated != closeErr || module.closeCalls[handle] != 1 {
				t.Fatal("malformed destructor executed more than once", repeated)
			}
			poisoned := errors.Is(test.want, rt.ErrProtocol)
			if guest.instance.Healthy() == poisoned || (guest.instance.Epoch() != epoch) != poisoned {
				t.Fatal("destructor changed protocol or limit poisoning")
			}
			addErr := sibling.Add()
			if poisoned {
				var invalid *rt.HandleError
				if !errors.As(addErr, &invalid) || module.addCalls != 0 {
					t.Fatal("poisoned sibling reached the guest", addErr)
				}
			} else if addErr != nil || module.addCalls != 1 {
				t.Fatal("output-limit error poisoned a sibling", addErr)
			}
			fresh, _ := fixtureItem(t, library)
			if (fresh.guest != guest) != poisoned || !fresh.guest.instance.Healthy() {
				t.Fatal("home replacement did not match protocol poisoning")
			}
			if poisoned {
				if err := sibling.Close(); !errors.Is(err, rt.ErrClosed) {
					t.Fatal("old home was not released on replacement", err)
				}
				if module.closeCalls[handle] != 1 || len(module.closeCalls) != 1 {
					t.Fatal("invalidated destructor reached the old module")
				}
			} else {
				module.closeReply = fixtureReply{pointer: 16}
				if err := sibling.Close(); err != nil {
					t.Fatal(err)
				}
			}
			if err := fresh.Close(); err != nil {
				t.Fatal("replacement destructor failed", err)
			}
		})
	}
}

func TestGeneratedOrdinaryUnitReplies(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name  string
		reply fixtureReply
		want  error
	}{
		{"empty at end", fixtureReply{pointer: 64}, nil},
		{"nonempty success", fixtureReply{pointer: 16, size: 1, message: "x"}, rt.ErrProtocol},
		{"empty beyond memory", fixtureReply{pointer: 65}, rt.ErrProtocol},
		{"limit before empty check", fixtureReply{pointer: 16, size: 33}, rt.ErrTooLarge},
	} {
		for _, stateless := range []bool{false, true} {
			name := "resource/" + test.name
			if stateless {
				name = "stateless/" + test.name
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				library := fixtureLibrary(t)
				item, _ := fixtureItem(t, library)
				var borrowed *guest
				var callErr error
				if stateless {
					if err := library.pool.Do(t.Context(), func(value *guest) error {
						borrowed = value
						value.module.unitReply = test.reply
						return nil
					}); err != nil {
						t.Fatal(err)
					}
					callErr = library.Unit(t.Context())
				} else {
					borrowed = item.guest
					borrowed.module.addReply = test.reply
					callErr = item.Add()
				}
				if !errors.Is(callErr, test.want) || borrowed.instance.Healthy() == errors.Is(test.want, rt.ErrProtocol) {
					t.Fatal("ordinary unit reply changed validation or poisoning", callErr)
				}
				if stateless {
					if err := library.Unit(t.Context()); err != nil {
						// A protocol fault discards the pooled module; healthy
						// limit errors retain its deliberately oversized reply.
						if !errors.Is(test.want, rt.ErrTooLarge) || !errors.Is(err, rt.ErrTooLarge) {
							t.Fatal("poisoned pool capacity was not reusable", err)
						}
					}
				}
				if err := item.Close(); err != nil && !errors.Is(err, rt.ErrPoisoned) {
					t.Fatal(err)
				}
			})
		}
	}
}
