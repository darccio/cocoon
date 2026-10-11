package streaming

import (
	"bytes"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
)

// This file can be copied unchanged beside the pre-feature facade: it uses only
// Open, NewStream, Write, End, and Close. An operation is a complete stream with
// 256 nominal chunks, including construction, owned outputs, End, and disposal.
// Open, lazy-home warmup, immutable input/oracle creation, and one full byte
// validation are outside timing. No PGO profile is supplied by this harness.
// Each measured leaf sets GOMAXPROCS; testing resets it before timing leaves.
// Retained outputs cover one finite group, matching a Batch's result lifetime.
// Corpus and group retention are fixed independently of b.N: at Chunk16384 the
// largest input is 4 MiB and a Retain256 group retains at most that plus 8 bytes.
const streamBenchChunks = 256

type streamBenchCorpus struct {
	chunks [][]byte
	want   []byte
	bytes  int
}

type streamBenchTotals struct {
	outputs uint64
	writes  uint64
	bytes   uint64
	groups  uint64
}

func streamBenchCorpora(chunk int, holdout bool) []streamBenchCorpus {
	count := 1
	if holdout {
		count = 4
	}
	corpora := make([]streamBenchCorpus, count)
	for variant := range count {
		body := make([]byte, chunk*streamBenchChunks)
		for index := range body {
			body[index] = byte('a' + index%23) // #nosec G115 -- The alphabet is bounded to ASCII.
		}
		if !holdout || variant != 0 {
			position := chunk - 3
			if holdout && variant == 1 {
				position = 1021 // A match also crosses the 1 KiB emission window.
			}
			copy(body[position:], "</head>")
			copy(body[len(body)/2+5:], "</head></head>")
		}
		if holdout {
			copy(body[len(body)-5:], "</hea")
			if variant == 3 {
				for index := 7; index < len(body); index += 257 {
					body[index] = 0xff
				}
			}
		}
		widths := []int{chunk}
		if holdout {
			widths = []int{max(1, chunk/2), chunk + 1, max(1, chunk-3), 0, chunk + 17}
		}
		var chunks [][]byte
		for offset, step := 0, variant; offset < len(body); step++ {
			width := widths[step%len(widths)]
			end := min(len(body), offset+width)
			chunks = append(chunks, body[offset:end:end])
			offset = end
		}
		corpora[variant] = streamBenchCorpus{chunks: chunks, want: streamBenchWholeOracle(body), bytes: len(body)}
	}
	return corpora
}

// Whole-body splicing does not use the guest's window or streaming emit loop.
func streamBenchWholeOracle(input []byte) []byte {
	position := bytes.Index(input, []byte("</head>"))
	if position < 0 {
		return append([]byte(nil), input...)
	}
	output := make([]byte, 0, len(input)+len("<cocoon>"))
	output = append(output, input[:position]...)
	output = append(output, "<cocoon>"...)
	return append(output, input[position:]...)
}

func streamBenchOpen(b *testing.B, options Options, homes int) *Library {
	b.Helper()
	library, err := Open(options)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := library.Close(); err != nil {
			b.Error(err)
		}
	})
	for range homes {
		stream, err := library.NewStream()
		if err != nil {
			b.Fatal(err)
		}
		if err := stream.Close(); err != nil {
			b.Fatal(err)
		}
	}
	return library
}

// afterGroup is used only by the separate mixed-traffic diagnostic. The paired
// standalone and Batch throughput cases pass nil and retain no output history.
func streamBenchStandalone(library *Library, corpus streamBenchCorpus, retain int, validate bool, afterGroup func() error) (totals streamBenchTotals, err error) {
	stream, err := library.NewStream()
	if err != nil {
		return totals, err
	}
	defer func() {
		if closeErr := stream.Close(); err == nil {
			err = closeErr
		}
	}()
	outputs := make([][]byte, 0, retain)
	flush := func() error {
		if validate {
			offset := int(totals.bytes) // #nosec G115 -- At most one fixed 4 MiB corpus is consumed.
			for _, output := range outputs {
				offset -= len(output)
			}
			for _, output := range outputs {
				if !bytes.Equal(output, corpus.want[offset:offset+len(output)]) {
					return fmt.Errorf("retained owned output changed at byte %d", offset)
				}
				offset += len(output)
			}
		}
		if afterGroup != nil {
			if groupErr := afterGroup(); groupErr != nil {
				return groupErr
			}
		}
		totals.groups++
		runtime.KeepAlive(outputs)
		clear(outputs)
		outputs = outputs[:0]
		return nil
	}
	consume := func(output []byte) error {
		if validate {
			offset := int(totals.bytes) // #nosec G115 -- At most one fixed 4 MiB corpus is consumed.
			if len(output) > len(corpus.want)-offset || !bytes.Equal(output, corpus.want[offset:offset+len(output)]) {
				return fmt.Errorf("stream output differs at byte %d", offset)
			}
		}
		totals.outputs++
		totals.bytes += uint64(len(output)) // #nosec G115 -- Slice lengths are nonnegative.
		outputs = append(outputs, output)
		if len(outputs) == retain {
			return flush()
		}
		return nil
	}
	for _, chunk := range corpus.chunks {
		output, writeErr := stream.Write(chunk)
		if writeErr != nil {
			return totals, writeErr
		}
		totals.writes++
		if consumeErr := consume(output); consumeErr != nil {
			return totals, consumeErr
		}
	}
	tail, err := stream.End()
	if err != nil {
		return totals, err
	}
	if consumeErr := consume(tail); consumeErr != nil {
		return totals, consumeErr
	}
	if len(outputs) != 0 {
		if flushErr := flush(); flushErr != nil {
			return totals, flushErr
		}
	}
	if totals.outputs != uint64(len(corpus.chunks)+1) || totals.bytes != uint64(len(corpus.want)) { // #nosec G115 -- Fixed corpus lengths are nonnegative.
		return totals, fmt.Errorf("stream consumed %d outputs/%d bytes, want %d/%d", totals.outputs, totals.bytes, len(corpus.chunks)+1, len(corpus.want))
	}
	return totals, nil
}

type streamBenchRun func(streamBenchCorpus) (streamBenchTotals, error)

func streamBenchParallel(b *testing.B, corpora []streamBenchCorpus, procs int, run streamBenchRun) streamBenchTotals {
	b.Helper()
	return streamBenchParallelWindow(b, corpora, procs, run, nil)
}

func streamBenchParallelWindow(b *testing.B, corpora []streamBenchCorpus, procs int, run streamBenchRun, active *atomic.Int64) streamBenchTotals {
	b.Helper()
	workers := runtime.GOMAXPROCS(0)
	if workers != procs {
		b.Fatalf("measured leaf has GOMAXPROCS %d, want %d", workers, procs)
	}
	problems := make(chan error, workers)
	var next atomic.Uint64
	var writes, outputs, outputBytes, groups atomic.Uint64
	b.SetBytes(int64(corpora[0].bytes))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		if active != nil {
			active.Add(1)
			defer active.Add(-1)
		}
		variant := int(next.Add(1)-1) % len(corpora) // #nosec G115 -- Only GOMAXPROCS worker indices are used.
		var workerWrites, workerOutputs, workerBytes, workerGroups uint64
		for pb.Next() {
			totals, err := run(corpora[variant])
			if err != nil {
				problems <- err
				return
			}
			workerWrites += totals.writes
			workerOutputs += totals.outputs
			workerBytes += totals.bytes
			workerGroups += totals.groups
			variant = (variant + 1) % len(corpora)
		}
		writes.Add(workerWrites)
		outputs.Add(workerOutputs)
		outputBytes.Add(workerBytes)
		groups.Add(workerGroups)
	})
	b.StopTimer()
	if actual := runtime.GOMAXPROCS(0); actual != workers {
		b.Fatalf("GOMAXPROCS changed during measurement: started %d, ended %d", workers, actual)
	}
	b.ReportMetric(float64(workers), "actual-procs")
	close(problems)
	for err := range problems {
		b.Fatal(err)
	}
	b.ReportMetric(float64(writes.Load())/b.Elapsed().Seconds(), "writes/s")
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "streams/s")
	b.ReportMetric(float64(outputs.Load())/float64(b.N), "outputs/stream")
	b.ReportMetric(float64(outputBytes.Load())/float64(b.N), "owned-B/stream")
	b.ReportMetric(float64(groups.Load())/float64(b.N), "groups/stream")
	return streamBenchTotals{outputs: outputs.Load(), writes: writes.Load(), bytes: outputBytes.Load(), groups: groups.Load()}
}

func BenchmarkStreamStandalone(b *testing.B) {
	for _, holdout := range []bool{false, true} {
		corpusName := "Fixed"
		if holdout {
			corpusName = "Holdout"
		}
		b.Run(corpusName, func(b *testing.B) {
			for _, procs := range []int{1, 4, 16} {
				b.Run(fmt.Sprintf("P%d", procs), func(b *testing.B) {
					for _, chunk := range []int{64, 4096, 16384} {
						b.Run(fmt.Sprintf("Chunk%d", chunk), func(b *testing.B) {
							corpora := streamBenchCorpora(chunk, holdout)
							for _, retain := range []int{16, 32, 64, 256} {
								b.Run(fmt.Sprintf("Retain%d", retain), func(b *testing.B) {
									previous := runtime.GOMAXPROCS(procs)
									defer runtime.GOMAXPROCS(previous)
									library := streamBenchOpen(b, Options{Instances: 1}, 1)
									for _, corpus := range corpora {
										if _, err := streamBenchStandalone(library, corpus, retain, true, nil); err != nil {
											b.Fatal(err)
										}
									}
									streamBenchParallel(b, corpora, procs, func(corpus streamBenchCorpus) (streamBenchTotals, error) {
										return streamBenchStandalone(library, corpus, retain, false, nil)
									})
								})
							}
						})
					}
				})
			}
		})
	}
}
