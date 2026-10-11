package dd

import (
	"math"
	"runtime"
	"testing"
)

const binReuseCorpusSize = 8192

var binReuseCorpora = []string{"Repeated", "UniqueCycle", "NearbyUniqueCycle", "MostlyUniqueCycle", "Phases", "BroadHoldout", "NegativeBins"}

// Freeze these inputs before selecting a cache. All finite corpora repeat after
// 8,192 points in timed loops; the separate Increasing control never repeats.
func binReuseValues(corpus string) []float64 {
	values := make([]float64, binReuseCorpusSize)
	negative := negativeBinSketchValues()
	for index := range values {
		switch corpus {
		case "Repeated":
			values[index] = 42
		case "UniqueCycle":
			values[index] = float64(index + 1)
		case "NearbyUniqueCycle":
			values[index] = math.Float64frombits(math.Float64bits(42) + uint64(index)) // #nosec G115 -- The index is bounded by the corpus size.
		case "MostlyUniqueCycle":
			values[index] = float64(1_000_000 + index)
			if index%16 == 0 {
				values[index] = 42
			}
		case "Phases":
			values[index] = float64(1_000_000 + index)
			if index/1024%2 == 0 {
				values[index] = float64(40 + index%16)
			}
		case "BroadHoldout":
			// An odd multiplier permutes the 52-bit mantissa; unique indices
			// cannot repeat a value even when their exponents coincide.
			mantissa := uint64(index+1) * 0x9e3779b97f4a7c15 & ((1 << 52) - 1) // #nosec G115 -- The index is positive and bounded.
			exponent := uint64(723 + index*37%601)                             // #nosec G115 -- Exponents are in [723, 1323].
			values[index] = math.Float64frombits(exponent<<52 | mantissa)
		case "NegativeBins":
			values[index] = negative[index%len(negative)]
		default:
			panic("unknown bin-reuse corpus")
		}
	}
	return values
}

func BenchmarkSketchBinReuse(b *testing.B) {
	for _, corpus := range append([]string{"Increasing"}, binReuseCorpora...) {
		b.Run(corpus, func(b *testing.B) {
			batchingProcessors(b, 1)
			sketch, _ := benchmarkSketch(b)
			values := binReuseValues("UniqueCycle")
			if corpus != "Increasing" {
				values = binReuseValues(corpus)
			}
			// Warm the store with one corpus, including collapse for the
			// broad holdout. UniqueCycle stresses the cache capacity.
			for _, value := range values {
				if err := sketch.Add(value); err != nil {
					b.Fatal(err)
				}
			}
			index := 0
			before := len(sketch.guest.module.Memory())
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				value := values[index&(binReuseCorpusSize-1)]
				if corpus == "Increasing" {
					value = float64(binReuseCorpusSize + index + 1)
				}
				if err := sketch.Add(value); err != nil {
					b.Fatal(err)
				}
				index++
			}
			b.StopTimer()
			batchingMetrics(b, 1, 1, before, len(sketch.guest.module.Memory()))
			if count, err := sketch.Count(); err != nil || count != float64(b.N+len(values)) {
				b.Fatalf("count=%g, want=%d: %v", count, b.N+len(values), err)
			}
		})
	}
}

// This interface is implemented only by the hash-checked diagnostic overlay.
// The committed generated module and its public facade remain unchanged.
type binReuseProbe interface {
	BinReuseOriginalMapping(float64, [4]uint64) int32
	BinReuseCachedMapping(float64, [4]uint64) int32
	BinReuseConfigure(int)
	BinReuseContains(float64, [4]uint64) bool
	BinReuseOccupied(float64) bool
}

func binReuseParameters() [4]uint64 {
	// Exact default constructor stores in the pinned generated module; the
	// diagnostic driver checks that constructor and the entire module hash.
	return [4]uint64{0x3ff0400000000000, 4634239110019182123, 0x0010400000000000, 0x4094ea0000000000}
}

func binReuseReplay(t *testing.T, probe binReuseProbe, values []float64, observed map[uint64]int32, capacity int) {
	t.Helper()
	probe.BinReuseConfigure(capacity)
	parameters := binReuseParameters()
	for replay := range 2 {
		var hits, misses, evictions int
		for index, value := range values {
			if probe.BinReuseContains(value, parameters) {
				hits++
			} else {
				misses++
				if probe.BinReuseOccupied(value) {
					evictions++
				}
			}
			if probe.BinReuseCachedMapping(value, parameters) != observed[math.Float64bits(value)] {
				t.Fatalf("cached mapping differs: capacity=%d replay=%d", capacity, replay)
			}
			if (index+1)%1024 == 0 {
				t.Logf("cache capacity=%d replay=%d phase=%d hits=%d misses=%d collisions/evictions=%d", capacity, replay, index/1024, hits, misses, evictions)
				hits, misses, evictions = 0, 0, 0
			}
		}
	}
}

func TestSketchBinReuseIdentity(t *testing.T) {
	g, _ := newDifferential(t)
	probe, ok := any(g.module).(binReuseProbe)
	if !ok {
		t.Skip("requires the diagnostic overlay")
	}
	parameters := binReuseParameters()
	other, _ := newDifferential(t)
	otherProbe, ok := any(other.module).(binReuseProbe)
	if !ok {
		t.Fatal("inconsistent diagnostic module")
	}
	for _, capacity := range []int{1, 16, 64, 256} {
		probe.BinReuseConfigure(capacity)
		otherProbe.BinReuseConfigure(capacity)
		_ = probe.BinReuseCachedMapping(42, parameters)
		if otherProbe.BinReuseContains(42, parameters) {
			t.Fatal("fresh module shares another module's entries")
		}
		if !probe.BinReuseContains(42, parameters) || probe.BinReuseContains(math.Nextafter(42, 43), parameters) {
			t.Fatal("cache does not compare full value bits")
		}
		for field := range parameters {
			changed := parameters
			changed[field]++
			if probe.BinReuseContains(42, changed) {
				t.Fatalf("mapping field %d absent from cache identity", field)
			}
			if got, want := probe.BinReuseCachedMapping(42, changed), probe.BinReuseOriginalMapping(42, changed); got != want {
				t.Fatalf("mapping parameter change: got=%d want=%d", got, want)
			}
			_ = probe.BinReuseCachedMapping(42, parameters)
		}
		probe.BinReuseConfigure(capacity)
		if probe.BinReuseContains(42, parameters) {
			t.Fatal("reset retained a stale entry")
		}
	}
}

func BenchmarkSketchBinMapping(b *testing.B) {
	for _, mode := range []string{"Original", "OriginalIncreasing", "Hit1", "Miss1", "Hit64", "Miss64"} {
		b.Run(mode, func(b *testing.B) {
			batchingProcessors(b, 1)
			sketch, _ := benchmarkSketch(b)
			probe, ok := any(sketch.guest.module).(binReuseProbe)
			if !ok {
				b.Skip("requires the diagnostic overlay")
			}
			capacity := 1
			if mode == "Hit64" || mode == "Miss64" {
				capacity = 64
			}
			probe.BinReuseConfigure(capacity)
			parameters := binReuseParameters()
			mapValue := probe.BinReuseCachedMapping
			if mode == "Original" || mode == "OriginalIncreasing" {
				mapValue = probe.BinReuseOriginalMapping
			}
			_ = mapValue(42, parameters)
			increasing := mode == "OriginalIncreasing" || mode == "Miss1" || mode == "Miss64"
			index := 0
			var result int32
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				value := float64(42)
				if increasing {
					value = float64(index + 1)
				}
				result = mapValue(value, parameters)
				index++
			}
			b.StopTimer()
			if runtime.GOMAXPROCS(0) != 1 {
				b.Fatal("actual processor mismatch")
			}
			b.ReportMetric(1, "actual-procs")
			b.ReportMetric(float64(result), "last-bin")
		})
	}
}

func TestSketchBinReuseCorpora(t *testing.T) {
	g, reference := newDifferential(t)
	limits := g.instanceLimits()
	parameters := binReuseParameters()
	probe, hasProbe := any(g.module).(binReuseProbe)
	for _, corpus := range binReuseCorpora {
		t.Run(corpus, func(t *testing.T) {
			values := binReuseValues(corpus)
			bins := make(map[int64]bool)
			observed := make(map[uint64]int32)
			var previous int64
			adjacent := 0
			for index, value := range values {
				bits := math.Float64bits(value)
				bin, seen := observed[bits]
				if !seen {
					count, encoded := binOracleState(t, reference, limits, []float64{value})
					state := readSingletonSketchBin(t, encoded)
					if count != math.Float64bits(1) || !state.hasBin || state.zeroCount != 0 {
						t.Fatalf("unexpected singleton state for %016x: %+v", bits, state)
					}
					bin = int32(state.index) // #nosec G115 -- The protobuf helper checks the int32 range.
					observed[bits] = bin
				}
				bins[int64(bin)] = true
				if index != 0 && previous == int64(bin) {
					adjacent++
				}
				previous = int64(bin)
				if hasProbe && probe.BinReuseOriginalMapping(value, parameters) != bin {
					t.Fatalf("extracted mapping differs for %016x", bits)
				}
			}
			if corpus == "UniqueCycle" || corpus == "NearbyUniqueCycle" || corpus == "BroadHoldout" {
				if len(observed) != len(values) {
					t.Fatal("unique corpus contains duplicate value bits")
				}
			}
			if corpus == "BroadHoldout" && len(bins) <= 2048 {
				t.Fatal("holdout does not exceed the pinned store capacity")
			}
			t.Logf("locality points=%d distinct-values=%d distinct-bins=%d adjacent-same-bin=%d", len(values), len(observed), len(bins), adjacent)
			if hasProbe {
				for _, capacity := range []int{1, 16, 64, 256} {
					binReuseReplay(t, probe, values, observed, capacity)
				}
			}
			// Ordered full-facade state comparison crosses collapse and then
			// revisits low bins, with both scalar and prevalidated batch calls.
			points := append(append(append([]float64(nil), values...), values...), values[:32]...)
			library := openTest(t, Options{Instances: 1})
			assertBinMathState(t, library, reference, limits, points, false)
			assertBinMathState(t, library, reference, limits, points, true)
		})
	}
}
