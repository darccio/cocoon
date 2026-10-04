package rt_test

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"cocoon.dev/cocoon/rt"
)

func TestResourceAliasesCloseOnce(t *testing.T) {
	t.Parallel()
	i := newInstance(t, newModule())
	var destroyed atomic.Int32
	r, err := rt.NewResource[int](i, 1<<32|1, func(handle uint64) error {
		if handle != 1<<32|1 {
			t.Error("handle changed")
		}
		destroyed.Add(1)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	alias := r
	if err := alias.Use("count", func(owner *rt.Instance, handle uint64) error {
		if owner != i || handle != 1<<32|1 {
			t.Error("ownership changed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 10 {
		group.Go(func() {
			if err := alias.Close(); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if destroyed.Load() != 1 {
		t.Fatal("destructor called more than once")
	}
	var handleErr *rt.HandleError
	if err := r.Use("count", nil); !errors.As(err, &handleErr) {
		t.Fatal("closed alias remains usable")
	}
}

func TestResourceFaultInvalidation(t *testing.T) {
	t.Parallel()
	i := newInstance(t, newModule())
	r, err := rt.NewResource[int](i, 7, func(_ uint64) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	err = i.Call("fault", func(_ *rt.Call) error { panic("unreachable") })
	if err == nil {
		t.Fatal("fault missing")
	}
	var handleErr *rt.HandleError
	if err := r.Use("count", nil); !errors.As(err, &handleErr) {
		t.Fatal("poisoned handle accepted")
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	var nilResource *rt.Resource[int]
	if err := nilResource.Close(); err != nil {
		t.Fatal(err)
	}
	if err := nilResource.Use("count", nil); !errors.As(err, &handleErr) {
		t.Fatal("nil resource accepted")
	}
	if _, err := rt.NewResource[int](nil, 0, nil); !errors.As(err, &handleErr) {
		t.Fatal("invalid ownership accepted")
	}
}
