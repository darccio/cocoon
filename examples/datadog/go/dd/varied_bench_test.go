package dd

import (
	"strings"
	"testing"
)

// These varied inputs are held out from the PGO training benchmarks.
func variedSketchValues() [16]float64 {
	return [16]float64{
		0.01, 0.025, 0.125, 0.5,
		1.25, 3.75, 12.5, 42.25,
		100.5, 512, 1024.75, 8192,
		10000.25, 65536, 250000.5, 1000000,
	}
}

func variedSQLQueries() [8]string {
	return [8]string{
		"SELECT sku, price FROM catalog WHERE category = 'garden' AND price > 19.95",
		"SELECT shipment_id FROM shipments WHERE destination IN ('Oslo', 'Porto') AND priority = 3",
		"INSERT INTO audit_events (actor, action, sequence_no) VALUES ('operator-a', 'login', 8127)",
		"INSERT INTO stock_levels (sku, available) VALUES ('widget-42', 96)",
		"UPDATE invoices SET paid = true, memo = 'wire-transfer' WHERE invoice_no = 7351",
		"UPDATE inventory SET quantity = quantity - 7 WHERE location = 'north-bin' AND product_id = 643",
		"DELETE FROM sessions WHERE account_id = 902 AND token = 'expired-token'",
		"DELETE FROM temp_readings WHERE station = 'ridge' AND recorded_at < '2026-01-01'",
	}
}

func BenchmarkSketchAddVaried(b *testing.B) {
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
	values := variedSketchValues()
	index := 0
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if addErr := sketch.Add(values[index]); addErr != nil {
			b.Fatal(addErr)
		}
		index = (index + 1) % len(values)
	}
	b.StopTimer()
	if count, countErr := sketch.Count(); countErr != nil || count != float64(b.N) {
		b.Fatalf("point count = %g, want %d: %v", count, b.N, countErr)
	}
}

func BenchmarkObfuscateSQLVaried(b *testing.B) {
	library := openTest(b, Options{Instances: 1})
	queries := variedSQLQueries()
	index := 0
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		output, err := library.ObfuscateSQL(b.Context(), queries[index])
		if err != nil || output == "" {
			b.Fatalf("query %d produced %q: %v", index, output, err)
		}
		index = (index + 1) % len(queries)
	}
}

func TestVariedBenchmarkCorpus(t *testing.T) {
	t.Parallel()
	library := openTest(t, Options{Instances: 1})
	sketch, err := library.NewSketch()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := sketch.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	values := variedSketchValues()
	for index, value := range values {
		if addErr := sketch.Add(value); addErr != nil {
			t.Fatalf("value %d rejected: %v", index, addErr)
		}
	}
	if count, countErr := sketch.Count(); countErr != nil || count != float64(len(values)) {
		t.Fatalf("point count = %g, want %d: %v", count, len(values), countErr)
	}
	for index, query := range variedSQLQueries() {
		output, sqlErr := library.ObfuscateSQL(t.Context(), query)
		if sqlErr != nil || !strings.Contains(output, "?") {
			t.Fatalf("query %d produced %q: %v", index, output, sqlErr)
		}
	}
}
