package rt

import (
	"sync"
	"sync/atomic"
)

const lifecycleClosing uint64 = 1 << 63

// Lifecycle provides a single terminal admission boundary and drains active calls.
// Its zero value is ready for use. It must not be copied after first use.
type Lifecycle struct {
	cond  *sync.Cond
	done  chan struct{}
	err   error
	mu    sync.Mutex
	state atomic.Uint64
}

// Enter admits a call or reports that terminal shutdown has begun.
func (l *Lifecycle) Enter() error {
	for {
		state := l.state.Load()
		if state&lifecycleClosing != 0 {
			return ErrClosed
		}
		if state == lifecycleClosing-1 {
			return ErrTooLarge
		}
		if l.state.CompareAndSwap(state, state+1) {
			return nil
		}
	}
}

// Leave releases one successful Enter. Every admission must have one release.
func (l *Lifecycle) Leave() {
	for {
		state := l.state.Load()
		if state&^lifecycleClosing == 0 {
			panic("cocoon: lifecycle release without admission")
		}
		if l.state.CompareAndSwap(state, state-1) {
			if state-1 == lifecycleClosing {
				// Only the final draining call takes the shutdown lock. Close
				// observes this same atomic count while holding that lock.
				l.mu.Lock()
				l.cond.Broadcast()
				l.mu.Unlock()
			}
			return
		}
	}
}

// Close rejects new calls, drains admitted calls, and runs cleanup exactly once.
// Concurrent callers wait for the same result. Cleanup must not reenter Close.
func (l *Lifecycle) Close(cleanup func() error) (err error) {
	l.mu.Lock()
	if l.state.Load()&lifecycleClosing != 0 {
		done := l.done
		l.mu.Unlock()
		<-done
		return l.err
	}
	l.done = make(chan struct{})
	l.cond = sync.NewCond(&l.mu)
	l.state.Or(lifecycleClosing)
	for l.state.Load()&^lifecycleClosing != 0 {
		l.cond.Wait()
	}
	l.mu.Unlock()
	defer func() {
		l.mu.Lock()
		l.err = err
		close(l.done)
		l.mu.Unlock()
	}()
	if cleanup != nil {
		return cleanup()
	}
	return nil
}
