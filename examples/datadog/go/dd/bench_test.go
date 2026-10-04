package dd

import "testing"

func BenchmarkObfuscateSQL(b *testing.B) {
	library := openTest(b, Options{Instances: 1})
	query := "SELECT * FROM users WHERE id = 42 AND name = 'bob'"
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := library.ObfuscateSQL(b.Context(), query); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSketchAdd(b *testing.B) {
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
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := sketch.Add(42); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSketchAddMany1k(b *testing.B) {
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
	values := make([]float64, 1000)
	for index := range values {
		values[index] = float64(index)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if err := sketch.AddMany(values); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkObfuscateTraces1k(b *testing.B) {
	library := openTest(b, Options{Instances: 1})
	payload := tracePayload(1000)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := library.ObfuscateTraces(b.Context(), payload); err != nil {
			b.Fatal(err)
		}
	}
}
