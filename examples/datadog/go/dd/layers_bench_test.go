package dd

import (
	"testing"

	"dario.cat/cocoon/rt"
)

func benchmarkSketch(b *testing.B) (sketch *Sketch, handle int64) {
	b.Helper()
	library := openTest(b, Options{Instances: 1})
	sketch, err := library.NewSketch()
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if closeErr := sketch.Close(); closeErr != nil {
			b.Error(closeErr)
		}
	})
	if useErr := sketch.owner.Use("benchmark", func(_ *rt.Instance, value uint64) error {
		handle = int64(value) //nolint:gosec // Wasm i64 preserves all bits of the unsigned handle.
		return nil
	}); useErr != nil {
		b.Fatal(useErr)
	}
	return sketch, handle
}

// BenchmarkSketchAddLayers intentionally bypasses public safety layers to
// measure their cumulative cost. Each layer executes the same real guest
// operation on increasing values; only Facade represents a supported call.
func BenchmarkSketchAddLayers(b *testing.B) {
	for _, layer := range []string{"Guest", "Instance", "Resource", "Facade"} {
		b.Run(layer, func(b *testing.B) {
			sketch, handle := benchmarkSketch(b)
			guest := sketch.guest
			invoke := func(call *rt.Call, value float64) error {
				status := rt.Status(guest.module.Xcocoon_sketch_add(handle, value))
				reply, err := call.ResultView("add", status)
				if err != nil {
					return err
				}
				if len(reply) != 0 {
					return rt.ErrProtocol
				}
				return nil
			}
			var add func(float64) error
			switch layer {
			case "Guest":
				add = func(value float64) error {
					if status := rt.Status(guest.module.Xcocoon_sketch_add(handle, value)); status != rt.OK {
						return rt.ErrProtocol
					}
					return nil
				}
			case "Instance":
				add = func(value float64) error {
					return guest.instance.Call("add", func(call *rt.Call) error { return invoke(call, value) })
				}
			case "Resource":
				add = func(value float64) error {
					return sketch.owner.Use("add", func(instance *rt.Instance, _ uint64) error {
						return instance.Call("add", func(call *rt.Call) error { return invoke(call, value) })
					})
				}
			case "Facade":
				add = sketch.Add
			}
			b.ReportAllocs()
			b.ResetTimer()
			value := float64(0)
			for b.Loop() {
				value++
				if err := add(value); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			if count, err := sketch.Count(); err != nil || count != float64(b.N) {
				b.Fatalf("guest count = %g, want %d: %v", count, b.N, err)
			}
		})
	}
}

// BenchmarkSketchEmptyLayers isolates admission and serialization without
// guest execution or reply validation. These are not public API operations.
func BenchmarkSketchEmptyLayers(b *testing.B) {
	for _, layer := range []string{"Instance", "Resource", "Lifecycle"} {
		b.Run(layer, func(b *testing.B) {
			sketch, _ := benchmarkSketch(b)
			var invoke func() error
			switch layer {
			case "Instance":
				invoke = func() error {
					return sketch.guest.instance.Call("empty", func(_ *rt.Call) error { return nil })
				}
			case "Resource":
				invoke = func() error {
					return sketch.owner.Use("empty", func(instance *rt.Instance, _ uint64) error {
						return instance.Call("empty", func(_ *rt.Call) error { return nil })
					})
				}
			case "Lifecycle":
				invoke = func() error {
					if err := sketch.library.life.Enter(); err != nil {
						return err
					}
					sketch.library.life.Leave()
					return nil
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if err := invoke(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
