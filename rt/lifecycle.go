package rt

import "sync"

// Lifecycle provides a single terminal admission boundary and drains active calls.
// Its zero value is ready for use. It must not be copied after first use.
type Lifecycle struct {
	cond    *sync.Cond
	done    chan struct{}
	err     error
	mu      sync.Mutex
	active  int
	closing bool
}

// Enter admits a call or reports that terminal shutdown has begun.
func (l *Lifecycle) Enter() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closing {
		return ErrClosed
	}
	l.active++
	return nil
}

// Leave releases one successful Enter. Every admission must have one release.
func (l *Lifecycle) Leave() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.active == 0 {
		panic("cocoon: lifecycle release without admission")
	}
	l.active--
	if l.active == 0 && l.cond != nil {
		l.cond.Broadcast()
	}
}

// Close rejects new calls, drains admitted calls, and runs cleanup exactly once.
// Concurrent callers wait for the same result. Cleanup must not reenter Close.
func (l *Lifecycle) Close(cleanup func() error) (err error) {
	l.mu.Lock()
	if l.closing {
		done := l.done
		l.mu.Unlock()
		<-done
		return l.err
	}
	l.closing = true
	l.done = make(chan struct{})
	l.cond = sync.NewCond(&l.mu)
	for l.active != 0 {
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
