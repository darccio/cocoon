package streaming

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"dario.cat/cocoon/rt"
)

// These opt-in cases use the same corpora, output retention, and complete stream
// lifetime as standalone_bench_test.go. Typed command construction, the facade's
// runtime plan construction, all owned results, and End/Close remain timed.
// Libraries are warm; every stream is new. Resource homes and the stateless pool
// are independently configured. Measurements must run without builds or lint,
// with CGO disabled and PGO off, using paired frozen matching-toolchain binaries.
func streamBenchBatch(ctx context.Context, library *Library, corpus streamBenchCorpus, count int, validate bool, afterGroup func() error) (totals streamBenchTotals, err error) {
	stream, err := library.NewStream()
	if err != nil {
		return totals, err
	}
	defer func() {
		if closeErr := stream.Close(); err == nil {
			err = closeErr
		}
	}()
	commands := make([]StreamCommand, 0, count)
	limits := rt.BatchLimits{MaxCalls: uint32(count), MaxInputBytes: 16 << 20, MaxOutputBytes: 16 << 20} // #nosec G115 -- Authored counts are 16, 32, 64, or 256.
	next, ended := 0, false
	for !ended {
		commands = commands[:0]
		for next < len(corpus.chunks) && len(commands) < count {
			commands = append(commands, StreamWriteCommand(corpus.chunks[next]))
			next++
		}
		// End shares the last finite group only when that group has capacity.
		// Otherwise it occupies a final one-command group.
		if next == len(corpus.chunks) && len(commands) < count {
			commands = append(commands, StreamEndCommand())
			ended = true
		}
		results, batchErr := stream.Batch(ctx, commands, limits)
		if batchErr != nil {
			return totals, batchErr
		}
		if len(results) != len(commands) {
			return totals, fmt.Errorf("Batch returned %d results for %d commands", len(results), len(commands))
		}
		for _, result := range results {
			var output []byte
			switch value := result.(type) {
			case StreamWriteResult:
				output = value.Value
				totals.writes++
			case StreamEndResult:
				output = value.Value
			default:
				return totals, fmt.Errorf("unexpected stream result %T", result)
			}
			if validate {
				offset := int(totals.bytes) // #nosec G115 -- At most one fixed 4 MiB corpus is consumed.
				if len(output) > len(corpus.want)-offset || !bytes.Equal(output, corpus.want[offset:offset+len(output)]) {
					return totals, fmt.Errorf("Batch output differs at byte %d", offset)
				}
			}
			totals.outputs++
			totals.bytes += uint64(len(output)) // #nosec G115 -- Slice lengths are nonnegative.
		}
		if afterGroup != nil {
			if groupErr := afterGroup(); groupErr != nil {
				return totals, groupErr
			}
		}
		totals.groups++
		runtime.KeepAlive(results)
		clear(results)
		clear(commands)
	}
	if totals.writes != uint64(len(corpus.chunks)) || totals.outputs != uint64(len(corpus.chunks)+1) || totals.bytes != uint64(len(corpus.want)) { // #nosec G115 -- Fixed corpus lengths are nonnegative.
		return totals, fmt.Errorf("Batch consumed %d writes/%d outputs/%d bytes, want %d/%d/%d", totals.writes, totals.outputs, totals.bytes, len(corpus.chunks), len(corpus.chunks)+1, len(corpus.want))
	}
	return totals, nil
}

func streamBenchCheckedCase(b *testing.B, corpora []streamBenchCorpus, procs, count int, path string, mixed bool) {
	b.Helper()
	homes := procs
	if path == "BatchHomes1" {
		homes = 1
	}
	pool := 1
	if mixed {
		pool = procs
	}
	library := streamBenchOpen(b, Options{Instances: pool, ResourceInstances: homes}, homes)
	var afterGroup func() error
	if mixed {
		input := bytes.Repeat([]byte("stateless"), 8)
		afterGroup = func() error {
			output, err := library.Echo(b.Context(), input)
			if err == nil && !bytes.Equal(output, input) {
				err = fmt.Errorf("mixed stateless echo changed bytes")
			}
			return err
		}
		if err := streamBenchWarmPool(b.Context(), library, pool, input); err != nil {
			b.Fatal(err)
		}
	}
	standalone := path == "StandaloneHomesP"
	for index := range homes * len(corpora) {
		corpus := corpora[index%len(corpora)]
		var err error
		if standalone {
			_, err = streamBenchStandalone(library, corpus, count, true, afterGroup)
		} else {
			_, err = streamBenchBatch(b.Context(), library, corpus, count, true, afterGroup)
		}
		if err != nil {
			b.Fatal(err)
		}
	}
	totals := streamBenchParallel(b, corpora, func(corpus streamBenchCorpus) (streamBenchTotals, error) {
		if standalone {
			return streamBenchStandalone(library, corpus, count, false, afterGroup)
		}
		return streamBenchBatch(b.Context(), library, corpus, count, false, afterGroup)
	})
	b.ReportMetric(float64(homes), "configured-homes")
	b.ReportMetric(float64(pool), "configured-pool")
	if mixed {
		b.ReportMetric(float64(totals.groups)/b.Elapsed().Seconds(), "echo/s")
		b.ReportMetric(float64(totals.groups)/float64(totals.writes), "echo/write")
	}
}

// Hold every distinct pool slot until all actual modules have completed Echo.
// A released slot cannot satisfy a later warmup borrow, regardless of reuse order.
func streamBenchWarmPool(ctx context.Context, library *Library, count int, input []byte) error {
	type readyGuest struct {
		err   error
		guest *guest
	}
	ready, done, release := make(chan readyGuest, count), make(chan error, count), make(chan struct{})
	for range count {
		go func() {
			entered := false
			err := library.pool.Do(ctx, func(value *guest) error {
				entered = true
				warmErr := value.instance.Call("echo", func(call *rt.Call) error {
					if inputErr := call.PrepareInput(uint64(len(input))); inputErr != nil { // #nosec G115 -- The fixture input is bounded and nonnegative.
						return inputErr
					}
					pointer, putErr := call.PutBytes(input)
					if putErr != nil {
						return putErr
					}
					status := rt.Status(value.module.Xcocoon_echo(int32(pointer), int32(len(input)))) // #nosec G115 -- Preserve Wasm pointer bits; input is 72 bytes.
					output, replyErr := call.Result("echo", status)
					if replyErr == nil && !bytes.Equal(output, input) {
						return fmt.Errorf("pool warmup echo changed bytes")
					}
					return replyErr
				})
				ready <- readyGuest{err: warmErr, guest: value}
				<-release
				return warmErr
			})
			if !entered {
				ready <- readyGuest{err: err}
			}
			done <- err
		}()
	}
	seen := make(map[*guest]bool, count)
	var firstErr error
	for range count {
		result := <-ready
		if result.err != nil && firstErr == nil {
			firstErr = result.err
		}
		if result.guest != nil {
			if seen[result.guest] && firstErr == nil {
				firstErr = fmt.Errorf("pool warmup borrowed the same held slot twice")
			}
			seen[result.guest] = true
		}
	}
	close(release)
	for range count {
		if err := <-done; err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func BenchmarkStreamChecked(b *testing.B) {
	for _, holdout := range []bool{false, true} {
		name := "Fixed"
		if holdout {
			name = "Holdout"
		}
		b.Run(name, func(b *testing.B) {
			for _, procs := range []int{1, 4, 16} {
				b.Run(fmt.Sprintf("P%d", procs), func(b *testing.B) {
					previous := runtime.GOMAXPROCS(procs)
					defer runtime.GOMAXPROCS(previous)
					for _, chunk := range []int{64, 4096, 16384} {
						b.Run(fmt.Sprintf("Chunk%d", chunk), func(b *testing.B) {
							corpora := streamBenchCorpora(chunk, holdout)
							for _, count := range []int{16, 32, 64, 256} {
								b.Run(fmt.Sprintf("N%d", count), func(b *testing.B) {
									for _, path := range []string{"StandaloneHomesP", "BatchHomes1", "BatchHomesP"} {
										if procs == 1 && path == "BatchHomesP" {
											continue
										}
										b.Run(path, func(b *testing.B) { streamBenchCheckedCase(b, corpora, procs, count, path, false) })
									}
								})
							}
						})
					}
				})
			}
		})
	}
}

// One Echo after each finite output group supplies mixed resource/stateless
// traffic. It runs on the independently configured pool, and its owned output
// is checked and released immediately. MB/s counts streaming input bytes only.
func BenchmarkStreamMixed(b *testing.B) {
	for _, procs := range []int{1, 4, 16} {
		b.Run(fmt.Sprintf("P%d", procs), func(b *testing.B) {
			previous := runtime.GOMAXPROCS(procs)
			defer runtime.GOMAXPROCS(previous)
			for _, count := range []int{16, 32, 64, 256} {
				b.Run(fmt.Sprintf("N%d", count), func(b *testing.B) {
					corpora := streamBenchCorpora(4096, true)
					for _, path := range []string{"StandaloneHomesP", "BatchHomes1", "BatchHomesP"} {
						b.Run(path, func(b *testing.B) { streamBenchCheckedCase(b, corpora, procs, count, path, true) })
					}
				})
			}
		})
	}
}

type streamBenchProbeResult struct {
	err         error
	samples     []time.Duration
	bytes       uint64
	calls       uint64
	windowCalls uint64
}

// The probe keeps only the last 1024 timings from calls begun while at least
// one RunParallel worker is active, including a final slow waiter that completes
// after the window.
// It releases every owned reply. Pre/post-window starts are never sampled.
// A dedicated extra goroutine competes with GOMAXPROCS ordinary stream workers.
// These clock/scheduling diagnostics are separate from throughput acceptance.
func streamBenchProbe(stream *Stream, stop <-chan struct{}, ready chan<- struct{}, active *atomic.Int64) (result streamBenchProbeResult) {
	defer func() {
		if closeErr := stream.Close(); result.err == nil {
			result.err = closeErr
		}
	}()
	input := bytes.Repeat([]byte{'p'}, 64)
	result.samples = make([]time.Duration, 1024)
	for {
		select {
		case <-stop:
			if result.windowCalls < uint64(len(result.samples)) { // #nosec G115 -- Slice length is nonnegative.
				result.samples = result.samples[:result.windowCalls]
			}
			tail, err := stream.End()
			if err == nil && result.bytes+uint64(len(tail)) != result.calls*uint64(len(input)) { // #nosec G115 -- Slice lengths are nonnegative.
				err = fmt.Errorf("fairness probe lost owned bytes")
			}
			result.err = err
			return result
		default:
		}
		record := active.Load() > 0
		started := time.Now()
		output, err := stream.Write(input)
		elapsed := time.Since(started)
		if err != nil {
			result.err = err
			return result
		}
		if record {
			result.samples[result.windowCalls%uint64(len(result.samples))] = elapsed // #nosec G115 -- Slice length is positive.
			result.windowCalls++
		}
		result.calls++
		result.bytes += uint64(len(output)) // #nosec G115 -- Slice length is nonnegative.
		if result.calls == 1 {
			close(ready)
		}
	}
}

func BenchmarkStreamFairness(b *testing.B) {
	for _, procs := range []int{1, 4, 16} {
		b.Run(fmt.Sprintf("P%d", procs), func(b *testing.B) {
			previous := runtime.GOMAXPROCS(procs)
			defer runtime.GOMAXPROCS(previous)
			homeCounts := []int{1}
			if procs != 1 {
				homeCounts = append(homeCounts, procs)
			}
			for _, homes := range homeCounts {
				b.Run(fmt.Sprintf("Homes%d", homes), func(b *testing.B) {
					for _, count := range []int{16, 32, 64, 256} {
						b.Run(fmt.Sprintf("N%d", count), func(b *testing.B) {
							library := streamBenchOpen(b, Options{Instances: 1, ResourceInstances: homes}, homes)
							corpora := streamBenchCorpora(4096, true)
							for index := range homes * len(corpora) {
								corpus := corpora[index%len(corpora)]
								if _, err := streamBenchBatch(b.Context(), library, corpus, count, true, nil); err != nil {
									b.Fatal(err)
								}
							}
							probe, err := library.NewStream()
							if err != nil {
								b.Fatal(err)
							}
							stop, ready, done := make(chan struct{}), make(chan struct{}), make(chan streamBenchProbeResult, 1)
							var active atomic.Int64
							go func() { done <- streamBenchProbe(probe, stop, ready, &active) }()
							select {
							case <-ready:
							case result := <-done:
								b.Fatal("fairness probe could not start", result.err)
							}
							streamBenchParallelWindow(b, corpora, func(corpus streamBenchCorpus) (streamBenchTotals, error) {
								return streamBenchBatch(b.Context(), library, corpus, count, false, nil)
							}, &active)
							close(stop)
							result := <-done
							if result.err != nil {
								b.Fatal("fairness probe failed", result.err)
							}
							if len(result.samples) != 0 {
								slices.Sort(result.samples)
								b.ReportMetric(float64(result.samples[(len(result.samples)-1)/2].Nanoseconds()), "probe-p50-ns")
								b.ReportMetric(float64(result.samples[(len(result.samples)-1)*95/100].Nanoseconds()), "probe-p95-ns")
								b.ReportMetric(float64(result.samples[len(result.samples)-1].Nanoseconds()), "probe-max-ns")
							}
							b.ReportMetric(float64(len(result.samples)), "probe-samples")
							b.ReportMetric(float64(result.windowCalls), "probe-window-calls")
						})
					}
				})
			}
		})
	}
}

// Heap diagnostics intentionally include GC and construction and are not used
// as throughput samples. HeapAlloc deltas are process heap observations, not RSS
// bounds. Live resources are explicit; all output references are released before
// the snapshot. Current logical guest-memory bytes are reported separately.
func BenchmarkStreamInitializedHeap(b *testing.B) {
	for _, homes := range []int{1, 4, 16} {
		for _, live := range []int{homes, 256} {
			b.Run(fmt.Sprintf("Homes%d/Live%d", homes, live), func(b *testing.B) {
				input := bytes.Repeat([]byte{'m'}, 16384)
				var heapDelta, guestBytes, initialized float64
				b.ReportAllocs()
				for b.Loop() {
					runtime.GC()
					var before, after runtime.MemStats
					runtime.ReadMemStats(&before)
					library, err := Open(Options{Instances: 1, ResourceInstances: homes})
					if err != nil {
						b.Fatal(err)
					}
					streams := make([]*Stream, live)
					for index := range streams {
						stream, err := library.NewStream()
						if err != nil {
							b.Fatal(err)
						}
						streams[index] = stream
						output, err := stream.Write(input)
						if err != nil || len(output) != len(input)-1024 {
							b.Fatal("memory fixture did not consume its owned output", err)
						}
					}
					runtime.GC()
					runtime.ReadMemStats(&after)
					for index := range library.homes {
						if home := library.homes[index].guest; home != nil {
							initialized++
							guestBytes += float64(len(home.module.Memory()))
						}
					}
					heapDelta += float64(after.HeapAlloc) - float64(before.HeapAlloc)
					for _, stream := range streams {
						if err := stream.Close(); err != nil {
							b.Fatal(err)
						}
					}
					if err := library.Close(); err != nil {
						b.Fatal(err)
					}
					runtime.KeepAlive(streams)
				}
				b.ReportMetric(heapDelta/float64(b.N), "heap-delta-B")
				b.ReportMetric(guestBytes/float64(b.N), "logical-guest-B")
				b.ReportMetric(initialized/float64(b.N), "initialized-homes")
				b.ReportMetric(float64(live), "live-resources")
			})
		}
	}
}
