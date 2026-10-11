package dd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync/atomic"
	"testing"

	"dario.cat/cocoon/cocoontest"
	"dario.cat/cocoon/rt"
)

// This private prototype preserves admission, ownership, checked execution,
// canonical replies and reservation accounting. PutBytes of the reservation
// itself advances the checked cursor without a second buffer or data transfer.
// It is a measurement control, not a supported API or generator implementation.
func benchmarkDirectAddMany(sketch *Sketch, values []float64) error {
	if sketch == nil || sketch.library == nil || sketch.owner == nil {
		return &rt.HandleError{Op: "add_many", Reason: "nil resource"}
	}
	library := sketch.library
	if err := library.life.Enter(); err != nil {
		return err
	}
	defer library.life.Leave()
	size, err := sizeSliceF64(values, library.options.MaxInput)
	if err != nil {
		return err
	}
	size, err = rt.InputSize(library.options.MaxInput, 1, 0, size)
	if err != nil {
		return err
	}
	return sketch.owner.Use("add_many", func(instance *rt.Instance, handle uint64) error {
		return instance.Call("add_many", func(call *rt.Call) error {
			pointer, inputErr := benchmarkDirectInput(call, values, size)
			if inputErr != nil {
				return inputErr
			}
			status := rt.Status(sketch.guest.module.Xcocoon_sketch_add_many(int64(handle), int32(pointer), int32(size))) //nolint:gosec // Validated wasm32 lengths; the handle preserves its i64 bits.
			reply, resultErr := call.ResultView("add_many", status)
			if resultErr != nil {
				return resultErr
			}
			if len(reply) != 0 {
				return rt.ErrProtocol
			}
			return nil
		})
	})
}

func benchmarkDirectInput(call *rt.Call, values []float64, size uint64) (uint32, error) {
	if err := call.PrepareInput(size); err != nil {
		return 0, err
	}
	pointer, err := call.PutBytes(nil)
	if err != nil {
		return 0, err
	}
	data, err := call.Range(pointer, uint32(size)) //nolint:gosec // PrepareInput checked the uint32 and host-int bounds.
	if err != nil {
		return 0, err
	}
	for index, value := range values {
		binary.LittleEndian.PutUint64(data[index*8:], math.Float64bits(value))
	}
	return call.PutBytes(data)
}

func batchingValues(corpus string, size int) []float64 {
	values := make([]float64, size)
	negative := negativeBinSketchValues()
	for index := range values {
		switch corpus {
		case "Repeated":
			values[index] = 42
		case "Nearby":
			values[index] = 1 + float64(index)/float64(max(size, 1))
		case "Unique":
			values[index] = float64(index + 1)
		case "NegativeBins":
			values[index] = negative[index%len(negative)]
		case "Holdout":
			// Deterministic, nonmonotonic, broad exponents and varied mantissas.
			values[index] = math.Ldexp(1+float64((index*7919+104729)%65521)/65521, index%601-300)
		}
	}
	return values
}

func batchingAdd(sketch *Sketch, mode string) func([]float64) error {
	switch mode {
	case "Scalar":
		return func(values []float64) error {
			for _, value := range values {
				if err := sketch.Add(value); err != nil {
					return err
				}
			}
			return nil
		}
	case "Prototype":
		return func(values []float64) error { return benchmarkDirectAddMany(sketch, values) }
	default:
		return sketch.AddMany
	}
}

func batchingProcessors(b *testing.B, processors int) {
	b.Helper()
	previous := runtime.GOMAXPROCS(processors)
	b.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	if actual := runtime.GOMAXPROCS(0); actual != processors {
		b.Fatalf("processors = %d, want %d", actual, processors)
	}
	// Repeat after ResetTimer: custom metrics are cleared by ResetTimer.
}

func batchingMetrics(b *testing.B, points, processors, before, after int) {
	b.Helper()
	if actual := runtime.GOMAXPROCS(0); actual != processors {
		b.Fatalf("processors = %d, want %d", actual, processors)
	}
	b.ReportMetric(float64(processors), "actual-procs")
	b.ReportMetric(float64(points), "points/op")
	b.ReportMetric(float64(points)*float64(b.N)/b.Elapsed().Seconds(), "points/s")
	b.ReportMetric(float64(after), "guest-B")
	b.ReportMetric(float64(after-before), "guest-growth-B")
}

// Each timed operation ingests the same ordered group. Warming one group makes
// bin coverage and reservation size stationary; scalar and batch see identical
// points. This measures steady ingestion, not a finite growth/collapse lifetime.
func BenchmarkSketchMatchedBatch(b *testing.B) {
	for _, corpus := range []string{"Repeated", "Nearby", "Unique", "NegativeBins", "Holdout"} {
		for _, size := range []int{1, 32, 1024, 8192} {
			for _, mode := range []string{"Scalar", "Facade", "Prototype"} {
				b.Run(fmt.Sprintf("%s/N%d/%s", corpus, size, mode), func(b *testing.B) {
					batchingProcessors(b, 1)
					sketch, _ := benchmarkSketch(b)
					values := batchingValues(corpus, size)
					add := batchingAdd(sketch, mode)
					if err := add(values); err != nil {
						b.Fatal(err)
					}
					before := len(sketch.guest.module.Memory())
					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						if err := add(values); err != nil {
							b.Fatal(err)
						}
					}
					b.StopTimer()
					batchingMetrics(b, size, 1, before, len(sketch.guest.module.Memory()))
					if count, err := sketch.Count(); err != nil || count != float64(size)*float64(b.N+1) {
						b.Fatalf("count = %g, want %g: %v", count, float64(size)*float64(b.N+1), err)
					}
				})
			}
		}
	}
}

// These independently timed controls are not additive components. Guest bypasses
// ownership and checked execution; only Facade is supported production behavior.
func BenchmarkSketchBatchLayers(b *testing.B) {
	for _, size := range []int{1, 32, 1024, 8192} {
		for _, layer := range []string{"Encode", "Copy", "Direct", "Guest"} {
			b.Run(fmt.Sprintf("N%d/%s", size, layer), func(b *testing.B) {
				batchingProcessors(b, 1)
				sketch, handle := benchmarkSketch(b)
				values := batchingValues("Nearby", size)
				encoded, err := encodeSliceF64(values, sketch.library.options.MaxInput)
				if err != nil {
					b.Fatal(err)
				}
				var pointer uint32
				copyInput := func(call *rt.Call) error {
					if inputErr := call.PrepareInput(uint64(len(encoded))); inputErr != nil {
						return inputErr
					}
					var putErr error
					pointer, putErr = call.PutBytes(encoded)
					return putErr
				}
				if err := sketch.AddMany(values); err != nil {
					b.Fatal(err)
				}
				if err := sketch.guest.instance.Call("prepare", copyInput); err != nil {
					b.Fatal(err)
				}
				var invoke func() error
				switch layer {
				case "Encode":
					invoke = func() error {
						data, encodeErr := encodeSliceF64(values, sketch.library.options.MaxInput)
						// Keep the allocation observable without a cross-iteration sink.
						runtime.KeepAlive(data)
						return encodeErr
					}
				case "Copy":
					invoke = func() error { return sketch.guest.instance.Call("prepare", copyInput) }
				case "Direct":
					invoke = func() error {
						return sketch.guest.instance.Call("prepare", func(call *rt.Call) error {
							_, inputErr := benchmarkDirectInput(call, values, uint64(len(encoded)))
							return inputErr
						})
					}
				case "Guest":
					invoke = func() error {
						status := rt.Status(sketch.guest.module.Xcocoon_sketch_add_many(handle, int32(pointer), int32(len(encoded)))) //nolint:gosec // Checked reservation and size are frozen before timing.
						if status != rt.OK {
							return rt.ErrProtocol
						}
						return nil
					}
				}
				before := len(sketch.guest.module.Memory())
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					if err := invoke(); err != nil {
						b.Fatal(err)
					}
				}
				b.StopTimer()
				batchingMetrics(b, size, 1, before, len(sketch.guest.module.Memory()))
				want := float64(size)
				if layer == "Guest" {
					want *= float64(b.N + 1)
				}
				if count, err := sketch.Count(); err != nil || count != want {
					b.Fatalf("count = %g, want %g: %v", count, want, err)
				}
			})
		}
	}
}

// Each worker owns one resource. Shared homes serialize guest execution and
// prototype encoding; Facade encodes before the lock. Sharded homes separate
// that lock. No resource is created in the timed region.
func BenchmarkSketchBatchContention(b *testing.B) {
	for _, processors := range []int{1, 16} {
		homeCounts := []int{1}
		if processors > 1 {
			homeCounts = append(homeCounts, processors)
		}
		for _, homes := range homeCounts {
			for _, size := range []int{32, 1024} {
				for _, mode := range []string{"Facade", "Prototype"} {
					b.Run(fmt.Sprintf("P%d/H%d/N%d/%s", processors, homes, size, mode), func(b *testing.B) {
						batchingProcessors(b, processors)
						library := openTest(b, Options{Instances: 1, ResourceInstances: homes})
						values := batchingValues("Nearby", size)
						sketches := make([]*Sketch, processors)
						for index := range sketches {
							var err error
							sketches[index], err = library.NewSketch()
							if err != nil {
								b.Fatal(err)
							}
							if err := sketches[index].AddMany(values); err != nil {
								b.Fatal(err)
							}
						}
						var next atomic.Uint32
						before := batchingHomeBytes(library)
						b.ReportAllocs()
						b.ResetTimer()
						b.RunParallel(func(pb *testing.PB) {
							index := next.Add(1) - 1
							add := batchingAdd(sketches[index], mode)
							for pb.Next() {
								if err := add(values); err != nil {
									b.Error(err)
									return
								}
							}
						})
						b.StopTimer()
						after := batchingHomeBytes(library)
						var total float64
						for _, sketch := range sketches {
							count, err := sketch.Count()
							if err != nil {
								b.Fatal(err)
							}
							total += count
							if err := sketch.Close(); err != nil {
								b.Fatal(err)
							}
						}
						if want := float64(size) * float64(b.N+processors); total != want {
							b.Fatalf("count = %g, want %g", total, want)
						}
						batchingMetrics(b, size, processors, before, after)
					})
				}
			}
		}
	}
}

func batchingHomeBytes(library *Library) int {
	var size int
	for index := range library.homes {
		if guest := library.homes[index].guest; guest != nil {
			size += len(guest.module.Memory())
		}
	}
	return size
}

func TestMatchedBatchPrototype(t *testing.T) {
	for _, corpus := range []string{"Repeated", "Nearby", "Unique", "NegativeBins", "Holdout"} {
		for _, size := range []int{0, 1, 32, 1024, 8192} {
			t.Run(fmt.Sprintf("%s/N%d", corpus, size), func(t *testing.T) {
				library := openTest(t, Options{Instances: 1})
				_, reference := newDifferential(t)
				status, reply, err := reference.Operation(t.Context(), "cocoon_sketch_new", nil, library.options.MaxInput, library.options.MaxOutput)
				if err != nil || status != rt.OK || len(reply) != 8 {
					t.Fatal(status, reply, err)
				}
				handle := binary.LittleEndian.Uint64(reply)
				values := batchingValues(corpus, size)
				for range 3 {
					for _, value := range values {
						status, _, err = reference.Operation(t.Context(), "cocoon_sketch_add", []cocoontest.Arg{{Value: handle}, {Value: math.Float64bits(value)}}, library.options.MaxInput, library.options.MaxOutput)
						if err != nil || status != rt.OK {
							t.Fatal(status, err)
						}
					}
				}
				status, want, err := reference.Operation(t.Context(), "cocoon_sketch_encode", []cocoontest.Arg{{Value: handle}}, library.options.MaxInput, library.options.MaxOutput)
				if err != nil || status != rt.OK {
					t.Fatal(status, err)
				}
				for _, mode := range []string{"Scalar", "Facade", "Prototype"} {
					sketch, newErr := library.NewSketch()
					if newErr != nil {
						t.Fatal(newErr)
					}
					for range 3 {
						if err := batchingAdd(sketch, mode)(values); err != nil {
							t.Fatal(err)
						}
					}
					count, actual := pointSketchState(t, sketch)
					if count != float64(size*3) || !bytes.Equal(actual, want) {
						t.Fatalf("%s count/complete protobuf differs from ordered Wasm scalars", mode)
					}
					if err := sketch.Close(); err != nil {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestMatchedHoldoutCollapses(t *testing.T) {
	_, reference := newDifferential(t)
	limits := rt.Limits{MaxInput: generatedInputLimit, MaxOutput: generatedOutputLimit, MaxMemory: maximumMemory}
	values := batchingValues("Holdout", 8192)
	minimum, maximum := values[0], values[0]
	for _, value := range values {
		minimum = min(minimum, value)
		maximum = max(maximum, value)
	}
	_, low := binOracleState(t, reference, limits, []float64{minimum})
	_, high := binOracleState(t, reference, limits, []float64{maximum})
	_, all := binOracleState(t, reference, limits, values)
	lowBin := readSingletonSketchBin(t, low)
	highBin := readSingletonSketchBin(t, high)
	first := readSingletonSketchBin(t, all)
	if !lowBin.hasBin || !highBin.hasBin || highBin.index-lowBin.index <= 2048 || first.index <= lowBin.index {
		t.Fatal("holdout did not force low-bin collapse in original Wasm", lowBin, highBin, first)
	}
}

func TestMatchedBatchPrototypeInvalidTail(t *testing.T) {
	library := openTest(t, Options{Instances: 1})
	sketch, err := library.NewSketch()
	if err != nil {
		t.Fatal(err)
	}
	if err := sketch.Add(1); err != nil {
		t.Fatal(err)
	}
	beforeCount, before := pointSketchState(t, sketch)
	for _, edge := range pointValidationCases() {
		if edge.valid {
			continue
		}
		checkPointArgument(t, "add_many", benchmarkDirectAddMany(sketch, []float64{2, 3, math.Float64frombits(edge.bits)}))
		count, after := pointSketchState(t, sketch)
		if count != beforeCount || !bytes.Equal(before, after) {
			t.Fatalf("prototype mutated state for invalid tail %s", edge.name)
		}
	}
	if err := benchmarkDirectAddMany(sketch, []float64{2, 3}); err != nil {
		t.Fatal("valid prototype failed after rejected tail", err)
	}
	if err := sketch.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMatchedBatchPrototypeLimitsAndClosure(t *testing.T) {
	for _, mode := range []string{"Facade", "Prototype"} {
		library := openTest(t, Options{Instances: 1, MaxInput: 16})
		sketch, err := library.NewSketch()
		if err != nil {
			t.Fatal(err)
		}
		add := batchingAdd(sketch, mode)
		for _, values := range [][]float64{nil, {}, {1}, {2, 3}} {
			if err := add(values); err != nil {
				t.Fatal(mode, "valid limited input", err)
			}
		}
		if err := add([]float64{1, 2, 3}); !errors.Is(err, rt.ErrTooLarge) {
			t.Fatal(mode, "oversized input", err)
		}
		if count, err := sketch.Count(); err != nil || count != 3 {
			t.Fatal(mode, "limit rejection changed Count", count, err)
		}
		if err := sketch.Close(); err != nil {
			t.Fatal(err)
		}
		var handle *rt.HandleError
		if err := add([]float64{1}); !errors.As(err, &handle) {
			t.Fatal(mode, "closed handle", err)
		}
		if err := add([]float64{1, 2, 3}); !errors.Is(err, rt.ErrTooLarge) {
			t.Fatal(mode, "size preflight lost precedence over closed handle", err)
		}
		if err := library.Close(); err != nil {
			t.Fatal(err)
		}
		if err := add([]float64{1}); !errors.Is(err, rt.ErrClosed) {
			t.Fatal(mode, "closed library", err)
		}
	}
}
