package rt_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"cocoon.dev/cocoon/rt"
)

func TestLifecycleDrainsAndClosesOnce(t *testing.T) {
	t.Parallel()
	var life rt.Lifecycle
	if err := life.Enter(); err != nil {
		t.Fatal(err)
	}
	var cleaned atomic.Int32
	var group sync.WaitGroup
	want := errors.New("cleanup")
	started := make(chan struct{})
	group.Go(func() {
		close(started)
		if err := life.Close(func() error { cleaned.Add(1); return want }); !errors.Is(err, want) {
			t.Errorf("Close: %v", err)
		}
	})
	<-started
	// Admission can race the first Close; release all calls that win that race.
	for {
		if err := life.Enter(); errors.Is(err, rt.ErrClosed) {
			break
		}
		life.Leave()
	}
	if cleaned.Load() != 0 {
		t.Fatal("closed before draining")
	}
	for range 8 {
		group.Go(func() {
			if err := life.Close(nil); !errors.Is(err, want) {
				t.Errorf("concurrent Close: %v", err)
			}
		})
	}
	life.Leave()
	group.Wait()
	if cleaned.Load() != 1 {
		t.Fatal("cleanup did not run exactly once")
	}
	if !errors.Is(life.Enter(), rt.ErrClosed) {
		t.Fatal("Close was not terminal")
	}
}

func TestLifecycleZeroAndMisuse(t *testing.T) {
	t.Parallel()
	var life rt.Lifecycle
	if err := life.Close(nil); err != nil {
		t.Fatal(err)
	}
	func() {
		defer func() {
			if recover() == nil {
				t.Error("unbalanced Leave did not panic")
			}
		}()
		life.Leave()
	}()
}

func TestLifecycleAdmissionShutdownRace(t *testing.T) {
	t.Parallel()
	for range 100 {
		var life rt.Lifecycle
		var active atomic.Int32
		start := make(chan struct{})
		var workers sync.WaitGroup
		for range 8 {
			workers.Go(func() {
				<-start
				for range 50 {
					if err := life.Enter(); err != nil {
						return
					}
					active.Add(1)
					active.Add(-1)
					life.Leave()
				}
			})
		}
		close(start)
		if err := life.Close(func() error {
			if active.Load() != 0 {
				t.Error("cleanup raced an admitted call")
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		workers.Wait()
		if !errors.Is(life.Enter(), rt.ErrClosed) {
			t.Fatal("readmission after close")
		}
	}
}
