package dd

import (
	"runtime"
	"strings"
	"testing"

	"dario.cat/cocoon/cocoontest"
	"dario.cat/cocoon/rt"
)

// These queries are held out from the original and varied SQL workloads. They
// cover longer scans, comments, escaped literals, Unicode, and quoted names.
func translationSQLQueries() []string {
	return []string{
		"/* route: warehouse */ SELECT \"item label\", amount FROM ledger WHERE note = 'it''s ready' AND amount >= -1.25e-9",
		"SELECT '世界', 'café', 9223372036854775807 FROM \"quoted table\" WHERE \"quoted column\" = 'mañana'",
		"WITH recent AS (SELECT event_id FROM event_log WHERE created_at > '2026-09-15') SELECT event_id FROM recent LIMIT 87",
		"SELECT object_id FROM archive WHERE object_id IN (" + strings.Repeat("17,23,41,59,", 128) + "83)",
		"UPDATE notes SET body = '" + strings.Repeat("quoted '' text 世界 ", 128) + "' WHERE note_id = 6701",
		"SELECT /* inline comment */ metric, value FROM readings -- trailing comment\nWHERE value BETWEEN 0.00017 AND 98765.4321",
	}
}

func BenchmarkObfuscateSQLTranslationHoldout(b *testing.B) {
	batchingProcessors(b, 1)
	library := openTest(b, Options{Instances: 1})
	queries := translationSQLQueries()
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
	b.StopTimer()
	if runtime.GOMAXPROCS(0) != 1 {
		b.Fatal("actual processor mismatch")
	}
	b.ReportMetric(1, "actual-procs")
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "queries/s")
}

func TestTranslationSQLHoldout(t *testing.T) {
	t.Parallel()
	g, reference := newDifferential(t)
	library := openTest(t, Options{Instances: 1})
	for range 2 {
		for index, query := range translationSQLQueries() {
			status, expected := compareOperation(t, g, reference, "cocoon_obfuscate_sql", []cocoontest.Arg{{Buffer: true, Bytes: []byte(query)}})
			output, err := library.ObfuscateSQL(t.Context(), query)
			if status != rt.OK || len(expected) == 0 || err != nil || output != string(expected) {
				t.Fatalf("query %d: status=%d output=%q expected=%q: %v", index, status, output, expected, err)
			}
		}
	}
}
