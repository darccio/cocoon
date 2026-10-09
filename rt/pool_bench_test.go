package rt_test

import (
	"testing"

	"dario.cat/cocoon/rt"
)

// BenchmarkPoolAvailable measures admission and reuse with a ready slot.
func BenchmarkPoolAvailable(b *testing.B) {
	pool, err := rt.NewPool(1, func() (*int, error) { return new(int), nil }, nil, nil)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := pool.Close(); err != nil {
			b.Error(err)
		}
	})
	invoke := func(_ *int) error { return nil }
	if err := pool.Do(b.Context(), invoke); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := pool.Do(b.Context(), invoke); err != nil {
			b.Fatal(err)
		}
	}
}
