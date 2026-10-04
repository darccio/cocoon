package rt

import (
	"runtime"
	"sync"
)

// NoCopy lets go vet diagnose accidental copies of generated resource wrappers.
type NoCopy struct{}

// Lock is a vet marker, not a runtime lock.
func (*NoCopy) Lock() {}

// Unlock is a vet marker, not a runtime lock.
func (*NoCopy) Unlock() {}

type destructor struct {
	drop   func(handle uint64) error
	err    error
	handle uint64
	once   sync.Once
}

func (d *destructor) close() {
	d.once.Do(func() { d.err = d.drop(d.handle) })
}

type ownership struct {
	instance *Instance
	token    *destructor
	cleanup  runtime.Cleanup
	epoch    uint64
	mu       sync.Mutex
	closed   bool
}

// Resource shares one ownership cell across every copy of a generated wrapper.
// Explicit Close is the normal path. The shared cell has a best-effort GC cleanup.
type Resource[T any] struct {
	_    NoCopy
	cell *ownership
}

// NewResource binds a nonzero generation-tagged handle to one instance epoch.
// The destructor must not retain the returned Resource or its ownership cell.
func NewResource[T any](instance *Instance, handle uint64, drop func(handle uint64) error) (*Resource[T], error) {
	if instance == nil || handle == 0 || drop == nil || !instance.alive.Load() {
		return nil, &HandleError{Op: "new", Reason: "invalid ownership"}
	}
	token := &destructor{drop: drop, handle: handle}
	cell := &ownership{instance: instance, token: token, epoch: instance.Epoch()}
	cell.cleanup = runtime.AddCleanup(cell, func(d *destructor) {
		// A user destructor panic must not terminate a GC cleanup goroutine.
		defer func() { recover() }() //nolint:errcheck // Cleanup cannot propagate a destructor panic.
		d.close()
	}, token)
	return &Resource[T]{cell: cell}, nil
}

// Use serializes resource use with Close and validates instance identity.
func (r *Resource[T]) Use(op string, invoke func(instance *Instance, handle uint64) error) error {
	if r == nil || r.cell == nil {
		return &HandleError{Op: op, Reason: "nil resource"}
	}
	cell := r.cell
	cell.mu.Lock()
	defer cell.mu.Unlock()
	defer runtime.KeepAlive(cell)
	if cell.closed || !cell.instance.alive.Load() || cell.epoch != cell.instance.Epoch() {
		return &HandleError{Op: op, Reason: "closed or invalidated instance"}
	}
	return invoke(cell.instance, cell.token.handle)
}

// Close invalidates every alias and runs the destructor once, outside the cell lock.
func (r *Resource[T]) Close() error {
	if r == nil || r.cell == nil {
		return nil
	}
	cell := r.cell
	cell.mu.Lock()
	cell.closed = true
	cell.mu.Unlock()
	cell.cleanup.Stop()
	cell.token.close()
	runtime.KeepAlive(cell)
	return cell.token.err
}
