package dd

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"testing"

	"dario.cat/cocoon/cocoontest"
	"dario.cat/cocoon/rt"
)

func backendProcessors(b *testing.B) {
	b.Helper()
	previous := runtime.GOMAXPROCS(1)
	b.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	backendAssertProcessors(b)
}

func backendAssertProcessors(b *testing.B) {
	b.Helper()
	if actual := runtime.GOMAXPROCS(0); actual != 1 {
		b.Fatalf("actual processors = %d, want 1", actual)
	}
}

func backendMetrics(b *testing.B, points, before, after int) {
	b.Helper()
	backendAssertProcessors(b)
	b.ReportMetric(1, "actual-procs")
	b.ReportMetric(float64(after), "guest-B")
	b.ReportMetric(float64(after-before), "guest-growth-B")
	if points != 0 {
		b.ReportMetric(float64(points), "points/op")
		b.ReportMetric(float64(points)*float64(b.N)/b.Elapsed().Seconds(), "points/s")
	}
}

func BenchmarkBackendWarm(b *testing.B) {
	workloads := []string{"ScalarRepeated", "ScalarIncreasing", "ScalarNegativeBins", "ScalarBroadHoldout", "BatchRepeated1024", "BatchBroad8192", "SQLOriginal", "SQLHoldout", "Trace1k"}
	for _, engine := range []string{"TranslatedGo", "CompiledWazero"} {
		b.Run(engine, func(b *testing.B) {
			for _, layer := range []string{"RawGuest", "CheckedControl"} {
				b.Run(layer, func(b *testing.B) {
					for _, workload := range workloads {
						b.Run(workload, func(b *testing.B) { backendWarm(b, engine, layer, workload) })
					}
				})
			}
			if engine == "CompiledWazero" {
				b.Run("RawCachedCall", func(b *testing.B) {
					for _, workload := range []string{"ScalarRepeated", "BatchRepeated1024"} {
						b.Run(workload, func(b *testing.B) { backendWarm(b, engine, "RawCachedCall", workload) })
					}
				})
			}
		})
	}
}

func backendWarm(b *testing.B, engine, layer, workload string) {
	b.Helper()
	backendProcessors(b)
	var g *backendGuest
	var control *backendControl
	if layer == "CheckedControl" {
		control = newBackendControl(b, engine)
		g = control.home
	} else {
		factory := newBackendFactory(b, engine, layer == "RawCachedCall")
		var err error
		g, err = factory.guest(b.Context())
		backendRequire(b, err)
		b.Cleanup(func() { backendRequire(b, g.close()) })
	}
	switch workload {
	case "SQLOriginal", "SQLHoldout", "Trace1k":
		backendWarmStateless(b, g, control, workload)
	default:
		backendWarmSketch(b, g, control, workload)
	}
}

func backendWarmSketch(b *testing.B, g *backendGuest, control *backendControl, workload string) {
	b.Helper()
	var handle uint64
	var add func(float64) error
	var batch func([]float64) error
	var count func() (float64, error)
	if control != nil {
		sketch, err := control.sketch()
		backendRequire(b, err)
		b.Cleanup(func() { backendRequire(b, sketch.owner.Close()) })
		add, batch, count = sketch.add, sketch.batch, sketch.count
	} else {
		var err error
		handle, err = g.rawSketch()
		backendRequire(b, err)
		b.Cleanup(func() { backendRequire(b, backendStatus(g.drop(handle))) })
		add = func(value float64) error { return backendStatus(g.add(handle, value)) }
		batch = func(values []float64) error { return backendStatus(g.rawBatch(handle, values)) }
		count = func() (float64, error) { return g.rawCount(handle) }
	}
	corpus, points, isBatch := "Repeated", 1, false
	switch workload {
	case "ScalarIncreasing":
		corpus = "UniqueCycle"
	case "ScalarNegativeBins":
		corpus = "NegativeBins"
	case "ScalarBroadHoldout":
		corpus = "BroadHoldout"
	case "BatchRepeated1024":
		points, isBatch = 1024, true
	case "BatchBroad8192":
		points, isBatch = 8192, true
	}
	values := binReuseValues(corpus)
	if isBatch {
		batchCorpus := "Repeated"
		if workload == "BatchBroad8192" {
			batchCorpus = "Holdout"
		}
		values = batchingValues(batchCorpus, points)
		backendRequire(b, batch(values))
	} else {
		for _, value := range values {
			backendRequire(b, add(value))
		}
	}
	before := len(g.abi.Memory())
	b.ReportAllocs()
	backendAssertProcessors(b)
	b.ResetTimer()
	for index := range b.N {
		var err error
		if isBatch {
			err = batch(values)
		} else {
			value := values[index%len(values)]
			if workload == "ScalarIncreasing" {
				value = float64(len(values)) + float64(index) + 1
			}
			err = add(value)
		}
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	backendAssertProcessors(b)
	after := len(g.abi.Memory())
	gotCount, err := count()
	backendRequire(b, err)
	wantCount := float64(len(values)) + float64(b.N)*float64(points)
	if gotCount != wantCount {
		b.Fatalf("count = %g, want %g", gotCount, wantCount)
	}
	backendMetrics(b, points, before, after)
}

func backendWarmStateless(b *testing.B, g *backendGuest, control *backendControl, workload string) {
	b.Helper()
	queries := []string{"SELECT * FROM users WHERE id = 42 AND name = 'bob'"}
	if workload == "SQLHoldout" {
		queries = translationSQLQueries()
	}
	input := tracePayload(1000)
	sql := g.rawSQL
	trace := g.rawTrace
	if control != nil {
		sql = func(query string) (string, error) { return control.sql(b.Context(), query) }
		trace = func(data []byte) ([]byte, error) { return control.trace(b.Context(), data) }
		// Inspect the actual stateless pool guest, rather than the unused home.
		backendRequire(b, control.pool.Do(b.Context(), func(poolGuest *backendGuest) error { g = poolGuest; return nil }))
	}
	oracle := backendOracle(b)
	var lastSQL string
	var lastTrace []byte
	var expectedTrace []byte
	expectedSQL := make([]string, len(queries))
	if workload == "Trace1k" {
		status, data, err := oracle.Operation(b.Context(), "cocoon_obfuscate_traces", []cocoontest.Arg{{Buffer: true, Bytes: input}}, backendInputLimit, backendInputLimit)
		backendRequire(b, err)
		backendRequire(b, rt.FromStatus("obfuscate_traces", status, data))
		expectedTrace = data
		lastTrace, err = trace(input)
		backendRequire(b, err)
		if !bytes.Equal(lastTrace, expectedTrace) {
			b.Fatal("warm trace oracle mismatch")
		}
	} else {
		for index, query := range queries {
			status, data, err := oracle.Operation(b.Context(), "cocoon_obfuscate_sql", []cocoontest.Arg{{Buffer: true, Bytes: []byte(query)}}, backendInputLimit, backendInputLimit)
			backendRequire(b, err)
			backendRequire(b, rt.FromStatus("obfuscate_sql", status, data))
			expectedSQL[index] = string(data)
			lastSQL, err = sql(query)
			backendRequire(b, err)
			if lastSQL != expectedSQL[index] {
				b.Fatal("warm SQL oracle mismatch")
			}
		}
	}
	before := len(g.abi.Memory())
	b.ReportAllocs()
	backendAssertProcessors(b)
	b.ResetTimer()
	for index := range b.N {
		var err error
		if workload == "Trace1k" {
			lastTrace, err = trace(input)
		} else {
			lastSQL, err = sql(queries[index%len(queries)])
		}
		if err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	backendAssertProcessors(b)
	after := len(g.abi.Memory())
	if workload == "Trace1k" {
		if !bytes.Equal(lastTrace, expectedTrace) {
			b.Fatal("timed trace oracle mismatch")
		}
		b.ReportMetric(1000, "spans/op")
	} else if b.N != 0 && lastSQL != expectedSQL[(b.N-1)%len(queries)] {
		b.Fatal("timed SQL oracle mismatch")
	}
	backendMetrics(b, 0, before, after)
}

func BenchmarkBackendCold(b *testing.B) {
	b.Run("TranslatedGo/ConstructInitClose", func(b *testing.B) {
		backendColdGuest(b, "TranslatedGo")
	})
	b.Run("CompiledWazero/CachedCompileInstantiateInitClose", func(b *testing.B) {
		backendColdGuest(b, "CompiledWazero")
	})
	b.Run("CompiledWazero/FreshRuntimeCompileClose", func(b *testing.B) {
		backendCompilerSupported(b, "CompiledWazero")
		backendProcessors(b)
		b.ReportAllocs()
		backendAssertProcessors(b)
		b.ResetTimer()
		for range b.N {
			engineRuntime := backendRuntime(b.Context(), nil)
			compiled, err := engineRuntime.CompileModule(b.Context(), generatedWasm)
			if err != nil {
				b.Fatal(err)
			}
			backendRequire(b, compiled.Close(context.WithoutCancel(b.Context())))
			backendRequire(b, engineRuntime.Close(context.WithoutCancel(b.Context())))
		}
		b.StopTimer()
		backendAssertProcessors(b)
		b.ReportMetric(1, "actual-procs")
	})
}

func backendColdGuest(b *testing.B, engine string) {
	b.Helper()
	backendProcessors(b)
	factory := newBackendFactory(b, engine, false)
	if engine == "CompiledWazero" {
		seed, err := factory.guest(b.Context())
		backendRequire(b, err)
		// The seed runtime/code and harness-local cache outlive the timed leaf.
		b.Cleanup(func() { backendRequire(b, seed.close()) })
	}
	var guestBytes int
	b.ReportAllocs()
	backendAssertProcessors(b)
	b.ResetTimer()
	for range b.N {
		guest, err := factory.guest(b.Context())
		if err != nil {
			b.Fatal(err)
		}
		guestBytes = len(guest.abi.Memory())
		backendRequire(b, guest.close())
	}
	b.StopTimer()
	backendMetrics(b, 0, guestBytes, guestBytes)
}

func TestBackendRawControls(t *testing.T) {
	for _, engine := range []string{"TranslatedGo", "CompiledWazero"} {
		for _, useCall := range []bool{false, true} {
			if engine == "TranslatedGo" && useCall {
				continue
			}
			t.Run(fmt.Sprintf("%s/Call=%t", engine, useCall), func(t *testing.T) {
				factory := newBackendFactory(t, engine, useCall)
				guest, err := factory.guest(t.Context())
				backendRequire(t, err)
				t.Cleanup(func() { backendRequire(t, guest.close()) })
				handle, err := guest.rawSketch()
				backendRequire(t, err)
				values := batchingValues("Holdout", 8192)
				wantCount, wantData := backendOracleSketch(t, values)
				backendRequire(t, backendStatus(guest.rawBatch(handle, values)))
				count, err := guest.rawCount(handle)
				backendRequire(t, err)
				data, err := guest.rawReply("encode", guest.encode(handle))
				backendRequire(t, err)
				if count != wantCount || !bytes.Equal(data, wantData) {
					t.Fatal("raw carrier state differs from ordered original Wasm")
				}
				backendRequire(t, backendStatus(guest.drop(handle)))
			})
		}
	}
}
