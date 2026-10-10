package unitfixture

import (
	"context"
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"testing"
	"time"

	"dario.cat/cocoon/rt"
)

// Factory controls belong to one library's logger, so parallel fixtures never
// share mutable constructor hooks or guest state.
type homeFactory struct {
	mu      sync.Mutex
	modules []*module
	create  func(index int, value *module)
	active  int
	peak    int
}

func (*homeFactory) Enabled(context.Context, slog.Level) bool  { return true }
func (*homeFactory) Handle(context.Context, slog.Record) error { return nil }
func (f *homeFactory) WithAttrs([]slog.Attr) slog.Handler      { return f }
func (f *homeFactory) WithGroup(string) slog.Handler           { return f }

func (f *homeFactory) fixtureModule() *module {
	f.mu.Lock()
	value := freshFixtureModule()
	index := len(f.modules)
	f.modules = append(f.modules, value)
	f.active++
	if f.active > f.peak {
		f.peak = f.active
	}
	create := f.create
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.active--
		f.mu.Unlock()
	}()
	if create != nil {
		create(index, value)
	}
	return value
}

func (f *homeFactory) counts() (created, active, peak int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.modules), f.active, f.peak
}

func homeLibrary(t *testing.T, homes int, factory *homeFactory) *Library {
	t.Helper()
	options := Options{Instances: 1, ResourceInstances: homes}
	if factory != nil {
		options.Logger = slog.New(factory)
	}
	library, err := Open(options)
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

func awaitHomeCursor(t *testing.T, library *Library, cursor int) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		library.homeMu.Lock()
		current := library.nextHome
		library.homeMu.Unlock()
		if current == cursor {
			return
		}
		select {
		case <-timer.C:
			t.Fatal("constructor did not select its home")
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		default:
			runtime.Gosched()
		}
	}
}

func TestResourceHomesDefaultsBoundsAndLaziness(t *testing.T) {
	t.Parallel()
	for _, homes := range []int{-1, 257} {
		if library, err := Open(Options{ResourceInstances: homes}); library != nil || !errors.Is(err, rt.ErrTooLarge) {
			t.Fatal("invalid home bound accepted", homes, library, err)
		}
	}
	for _, homes := range []int{0, 1, 2, 256} {
		factory := &homeFactory{}
		library := homeLibrary(t, homes, factory)
		want := homes
		if want == 0 {
			want = 1
		}
		if len(library.homes) != want || library.options.ResourceInstances != want || library.options.Instances != 1 {
			t.Fatal("resource homes changed independent pool configuration")
		}
		if created, _, _ := factory.counts(); created != 0 {
			t.Fatal("Open eagerly initialized a home")
		}
		first, _ := fixtureItem(t, library)
		second, _ := fixtureItem(t, library)
		if (first.guest == second.guest) != (want == 1) {
			t.Fatal("default or explicit placement changed")
		}
		if created, _, _ := factory.counts(); created != min(want, 2) {
			t.Fatal("constructor initialized unused homes", created)
		}
		if err := library.Close(); err != nil {
			t.Fatal(err)
		}
		for index := range library.homes {
			if library.homes[index].guest != nil {
				t.Fatal("Close retained a home")
			}
		}
	}
}

func TestResourceHomesRoundRobinAndIndependentPool(t *testing.T) {
	t.Parallel()
	factory := &homeFactory{}
	library := homeLibrary(t, 2, factory)
	a, aHandle := fixtureItem(t, library)
	b, bHandle := fixtureItem(t, library)
	c, cHandle := fixtureItem(t, library)
	if a.guest == b.guest || a.guest != c.guest || aHandle != bHandle || aHandle == cHandle {
		t.Fatal("round-robin homes lost exact identity or per-home handles")
	}
	named, err := library.NewNamed("fourth")
	if err != nil || named.guest != b.guest {
		t.Fatal("resource types do not share the placement sequence", err)
	}
	var pooled *guest
	if err := library.pool.Do(t.Context(), func(value *guest) error { pooled = value; return nil }); err != nil {
		t.Fatal(err)
	}
	if pooled == a.guest || pooled == b.guest {
		t.Fatal("resource placement borrowed the stateless pool")
	}
	if created, _, _ := factory.counts(); created != 3 {
		t.Fatal("unexpected instance count", created)
	}
}

func TestResourceHomesSerializeOnlyTheirLane(t *testing.T) {
	t.Parallel()
	library := homeLibrary(t, 2, nil)
	a, _ := fixtureItem(t, library)
	b, _ := fixtureItem(t, library)
	c, _ := fixtureItem(t, library)
	started, release := make(chan struct{}), make(chan struct{})
	a.guest.module.addStarted, a.guest.module.addRelease = started, release
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	aDone, bDone, cDone := make(chan error, 1), make(chan error, 1), make(chan error, 1)
	go func() { aDone <- a.Add() }()
	awaitFixtureSignal(t, started)
	cEntered := make(chan struct{})
	go func() { close(cEntered); cDone <- c.Add() }()
	awaitFixtureSignal(t, cEntered)
	go func() { bDone <- b.Add() }()
	if err := awaitFixtureResult(t, bDone); err != nil {
		t.Fatal("a blocked home serialized another home", err)
	}
	select {
	case err := <-cDone:
		t.Fatal("same-home call bypassed the execution lock", err)
	default:
	}
	if a.guest.module.addCalls != 1 || b.guest.module.addCalls != 1 {
		t.Fatal("blocked home executed its next export")
	}
	unblock()
	for _, done := range []<-chan error{aDone, cDone} {
		if err := awaitFixtureResult(t, done); err != nil {
			t.Fatal(err)
		}
	}
	if a.guest.module.addCalls != 2 {
		t.Fatal("same-home waiter did not resume")
	}
}

func TestResourceHomePoisonReplacementKeepsExactIdentity(t *testing.T) {
	t.Parallel()
	library := homeLibrary(t, 2, nil)
	a, handle := fixtureItem(t, library)
	alias := fixtureAlias(t, a)
	b, _ := fixtureItem(t, library)
	c, _ := fixtureItem(t, library)
	oldGuest, oldModule := a.guest, a.guest.module
	epoch := oldGuest.instance.Epoch()
	oldModule.addReply = fixtureReply{pointer: 16, size: 1, message: "x"}
	if err := a.Add(); !errors.Is(err, rt.ErrProtocol) || oldGuest.instance.Healthy() || oldGuest.instance.Epoch() != epoch+1 {
		t.Fatal("fault did not poison exactly the resource home", err)
	}
	for _, item := range []*Item{a, alias, c} {
		var invalid *rt.HandleError
		if err := item.Add(); !errors.As(err, &invalid) {
			t.Fatal("poisoned alias reached the guest", err)
		}
	}
	if oldModule.addCalls != 1 || b.Add() != nil {
		t.Fatal("poison escaped its home or replayed an old handle")
	}
	d, _ := fixtureItem(t, library) // The next constructor selects healthy B.
	if d.guest != b.guest || oldGuest.module == nil {
		t.Fatal("a sibling constructor replaced the wrong slot")
	}
	fresh, freshHandle := fixtureItem(t, library)
	if fresh.guest == oldGuest || fresh.guest == b.guest || freshHandle != handle || oldGuest.module != nil || oldGuest.instance.Epoch() != epoch+2 {
		t.Fatal("replacement migrated handles or failed to retire the old home")
	}
	for _, item := range []*Item{a, alias, c} {
		var invalid *rt.HandleError
		if err := item.Add(); !errors.As(err, &invalid) {
			t.Fatal("old wrapper resolved against a numerically identical new handle", err)
		}
	}
	if oldModule.addCalls != 1 || fresh.guest.module.addCalls != 0 || len(oldModule.closeCalls) != 0 {
		t.Fatal("old handle reached a retired or replacement module")
	}
	if b.Add() != nil || fresh.Add() != nil || library.Unit(t.Context()) != nil {
		t.Fatal("home replacement damaged sibling or stateless execution")
	}
	closedEpoch := oldGuest.instance.Epoch()
	if err := library.Close(); err != nil || oldGuest.instance.Epoch() != closedEpoch {
		t.Fatal("library close disposed a retired home again", err)
	}
}

func TestResourceHomeFailedEncodingStillAdvancesPlacement(t *testing.T) {
	t.Parallel()
	factory := &homeFactory{}
	library := homeLibrary(t, 2, factory)
	if item, err := library.NewNamed(string([]byte{0xff})); item != nil || err == nil {
		t.Fatal("invalid UTF-8 constructor accepted", err)
	}
	if library.nextHome != 1 || library.homes[0].guest != nil {
		t.Fatal("encoding failure did not consume its lazy placement")
	}
	if created, _, _ := factory.counts(); created != 0 {
		t.Fatal("failed encoding initialized a home")
	}
	b, _ := fixtureItem(t, library)
	if library.homes[1].guest != b.guest || library.homes[0].guest != nil {
		t.Fatal("constructor retried the failed attempt's slot")
	}
	if _, err := library.NewNamed(string(make([]byte, 33))); !errors.Is(err, rt.ErrTooLarge) || library.nextHome != 1 {
		t.Fatal("size preflight did not advance placement", err)
	}
}

func TestResourceHomeInitFailureRetriesWithoutBlockingSibling(t *testing.T) {
	t.Parallel()
	initStarted, release := make(chan struct{}), make(chan struct{})
	factory := &homeFactory{create: func(index int, value *module) {
		if index == 0 {
			value.initHook = func() { close(initStarted); <-release; panic("init failed") }
		}
	}}
	library := homeLibrary(t, 2, factory)
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	failed := make(chan error, 1)
	go func() { _, err := library.NewItem(); failed <- err }()
	awaitFixtureSignal(t, initStarted)
	b, _ := fixtureItem(t, library)
	if library.homes[1].guest != b.guest {
		t.Fatal("blocked initialization held the placement lock")
	}
	retry := make(chan error, 1)
	go func() { _, err := library.NewItem(); retry <- err }()
	awaitHomeCursor(t, library, 1)
	if created, _, _ := factory.counts(); created != 2 {
		t.Fatal("same slot started a concurrent initialization", created)
	}
	unblock()
	if err := awaitFixtureResult(t, failed); err == nil {
		t.Fatal("initialization trap accepted")
	}
	if err := awaitFixtureResult(t, retry); err != nil {
		t.Fatal("failed slot could not be retried", err)
	}
	if library.homes[0].guest == nil || library.homes[0].guest == b.guest || b.Add() != nil {
		t.Fatal("retry changed sibling identity")
	}
}

func TestResourceHomeFactoryCapacityAndCloseDrain(t *testing.T) {
	t.Parallel()
	started, release := make(chan struct{}, 2), make(chan struct{})
	factory := &homeFactory{create: func(_ int, _ *module) { started <- struct{}{}; <-release }}
	library := homeLibrary(t, 2, factory)
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	first, second, third := make(chan error, 1), make(chan error, 1), make(chan error, 1)
	go func() { _, err := library.NewItem(); first <- err }()
	awaitFixtureSignal(t, started)
	go func() { _, err := library.NewItem(); second <- err }()
	awaitFixtureSignal(t, started)
	go func() { _, err := library.NewItem(); third <- err }()
	awaitHomeCursor(t, library, 1)
	if created, active, peak := factory.counts(); created != 2 || active != 2 || peak != 2 {
		t.Fatal("home creation exceeded capacity", created, active, peak)
	}
	closing := make(chan error, 1)
	go func() { closing <- library.Close() }()
	awaitFixtureClosing(t, library)
	if _, err := library.NewItem(); !errors.Is(err, rt.ErrClosed) {
		t.Fatal("Close admitted another constructor", err)
	}
	select {
	case err := <-closing:
		t.Fatal("Close did not drain in-flight initialization", err)
	default:
	}
	unblock()
	for _, done := range []<-chan error{first, second, third, closing} {
		if err := awaitFixtureResult(t, done); err != nil {
			t.Fatal("Close interrupted admitted construction", err)
		}
	}
	if created, active, peak := factory.counts(); created != 2 || active != 0 || peak != 2 {
		t.Fatal("waiting constructor created an extra home", created, active, peak)
	}
	for index := range library.homes {
		if library.homes[index].guest != nil {
			t.Fatal("Close retained an initialized slot")
		}
	}
}

func TestResourceHomeExpectedConstructorErrorPreservesSlot(t *testing.T) {
	t.Parallel()
	factory := &homeFactory{create: func(index int, value *module) {
		if index == 0 {
			value.newReply = fixtureReply{pointer: 16, size: 3, status: rt.ErrApp, message: "bad"}
		}
	}}
	library := homeLibrary(t, 2, factory)
	if item, err := library.NewItem(); item != nil || err == nil {
		t.Fatal("expected constructor error accepted", err)
	}
	a := library.homes[0].guest
	if a == nil || !a.instance.Healthy() || library.nextHome != 1 {
		t.Fatal("expected constructor error retired or lost its home")
	}
	b, _ := fixtureItem(t, library)
	a.module.newReply = fixtureReply{}
	retried, _ := fixtureItem(t, library)
	if retried.guest != a || b.guest == a {
		t.Fatal("healthy error home was replaced or did not advance placement")
	}
}

func TestResourceHomeFactoryPanicLeavesRetryableSlot(t *testing.T) {
	t.Parallel()
	factory := &homeFactory{create: func(index int, _ *module) {
		if index == 0 {
			panic("factory failed")
		}
	}}
	library := homeLibrary(t, 2, factory)
	if item, err := library.NewItem(); item != nil || err == nil {
		t.Fatal("factory panic escaped or was accepted", err)
	}
	if library.homes[0].guest != nil || library.nextHome != 1 {
		t.Fatal("failed factory occupied or failed to advance its slot")
	}
	b, _ := fixtureItem(t, library)
	a, _ := fixtureItem(t, library)
	if library.homes[0].guest != a.guest || library.homes[1].guest != b.guest || a.guest == b.guest {
		t.Fatal("factory failure left a slot unusable")
	}
	if created, active, peak := factory.counts(); created != 3 || active != 0 || peak != 1 {
		t.Fatal("factory failure leaked creation capacity", created, active, peak)
	}
}
