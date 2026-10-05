package rt

import (
	"context"
	"fmt"
)

// Pool bounds the number of simultaneously borrowed instances and lazy factories.
// A callback panic discards its instance and always restores the admission slot.
type Pool[T any] struct {
	factory func() (*T, error)
	recycle func(instance *T) bool
	destroy func(instance *T)
	slots   chan *T
	life    Lifecycle
}

// NewPool constructs a bounded lazy pool with optional recycling and disposal.
func NewPool[T any](size int, factory func() (*T, error), recycle func(instance *T) bool, destroy func(instance *T)) (*Pool[T], error) {
	if size <= 0 || factory == nil {
		return nil, fmt.Errorf("%w: pool requires a positive capacity and factory", ErrProtocol)
	}
	p := &Pool[T]{factory: factory, recycle: recycle, destroy: destroy, slots: make(chan *T, size)}
	for range size {
		p.slots <- nil
	}
	return p, nil
}

// Do borrows an instance. Context bounds admission, not synchronous execution.
func (p *Pool[T]) Do(ctx context.Context, invoke func(instance *T) error) error {
	if err := p.life.Enter(); err != nil {
		return err
	}
	defer p.life.Leave()
	if err := ctx.Err(); err != nil {
		return err
	}
	var instance *T
	select {
	case instance = <-p.slots:
	default:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case instance = <-p.slots:
		}
	}
	// Install the release before factory, callback, or recycler can panic.
	completed := false
	defer func() {
		defer func() { p.slots <- instance }()
		if !completed {
			discard := instance
			instance = nil
			if discard != nil && p.destroy != nil {
				p.destroy(discard)
			}
		}
	}()
	if err := ctx.Err(); err != nil {
		completed = true
		return err
	}
	if instance == nil {
		var err error
		instance, err = p.factory()
		if err != nil {
			return err
		}
		if instance == nil {
			return fmt.Errorf("%w: pool factory returned nil", ErrProtocol)
		}
	}
	err := invoke(instance)
	if p.recycle != nil && !p.recycle(instance) {
		return err
	}
	completed = true
	return err
}

// Close drains all admitted calls and destroys every retained instance once.
func (p *Pool[T]) Close() error {
	return p.life.Close(func() error {
		for range cap(p.slots) {
			instance := <-p.slots
			if instance != nil && p.destroy != nil {
				p.destroy(instance)
			}
		}
		return nil
	})
}
