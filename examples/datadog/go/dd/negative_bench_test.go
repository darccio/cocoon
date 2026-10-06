package dd

import "testing"

func negativeBinSketchValues() [8]float64 {
	return [8]float64{1e-12, 1.0625e-12, 1.125e-12, 1.25e-12, 1.375e-12, 1.5e-12, 1.75e-12, 2e-12}
}

func BenchmarkSketchAddNegativeBins(b *testing.B) {
	library := openTest(b, Options{Instances: 1})
	sketch, err := library.NewSketch()
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := sketch.Close(); err != nil {
			b.Error(err)
		}
	})
	values := negativeBinSketchValues()
	index := 0
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := sketch.Add(values[index]); err != nil {
			b.Fatal(err)
		}
		index = (index + 1) % len(values)
	}
	b.StopTimer()
	if count, err := sketch.Count(); err != nil || count != float64(b.N) {
		b.Fatalf("point count=%g, want=%d: %v", count, b.N, err)
	}
}

func TestSketchNegativeBinBenchmarkCorpus(t *testing.T) {
	g, reference := newDifferential(t)
	limits := g.instanceLimits()
	library := openTest(t, Options{Instances: 1})
	for _, value := range negativeBinSketchValues() {
		encoded := assertBinMathState(t, library, reference, limits, []float64{value}, false)
		state := readSingletonSketchBin(t, encoded)
		if !state.hasBin || state.index >= 0 || state.zeroCount != 0 {
			t.Fatalf("benchmark input %g does not exercise a negative bin: %+v", value, state)
		}
	}
}
