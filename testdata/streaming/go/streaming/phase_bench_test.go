package streaming

import (
	"bytes"
	"fmt"
	"runtime"
	"testing"
)

// Diagnostic phases use the same source with the original and current facade.
// Each leaf sets actual processors. Libraries are warm; all calls retain the
// normal facade checks. These timings cannot be summed into a full-stream time:
// priming, output retention, live resources, timer toggles, and GC differ.
const streamPhaseGroup = 32

func streamPhaseCheckProcs(b *testing.B) {
	b.Helper()
	if actual := runtime.GOMAXPROCS(0); actual != 1 {
		b.Fatalf("phase measured GOMAXPROCS %d, want 1", actual)
	}
}

func streamPhaseOpen(b *testing.B) *Library {
	b.Helper()
	previous := runtime.GOMAXPROCS(1)
	b.Cleanup(func() { runtime.GOMAXPROCS(previous) })
	streamPhaseCheckProcs(b)
	return streamBenchOpen(b, Options{Instances: 1}, 1)
}

func streamPhaseReport(b *testing.B, unit string, count, owned, before, after int) {
	b.Helper()
	b.StopTimer()
	streamPhaseCheckProcs(b)
	b.ReportMetric(1, "actual-procs")
	b.ReportMetric(float64(count), unit+"/op")
	b.ReportMetric(float64(count)*float64(b.N)/b.Elapsed().Seconds(), unit+"/s")
	b.ReportMetric(float64(owned), "owned-B/op")
	b.ReportMetric(float64(after-before), "guest-growth-B")
	b.ReportMetric(float64(after), "guest-memory-B")
}

type streamPhaseInput struct {
	seed, prime, tail []byte
}

func streamPhaseState(name string) streamPhaseInput {
	seed := bytes.Repeat([]byte{'x'}, 1024)
	switch name {
	case "Empty":
		return streamPhaseInput{}
	case "Short":
		return streamPhaseInput{seed: seed[:17], tail: seed[:17]}
	case "Window":
		return streamPhaseInput{seed: seed, tail: seed}
	case "Injected":
		copy(seed, "</head>")
		return streamPhaseInput{seed: seed, prime: bytes.Repeat([]byte{'x'}, 64), tail: bytes.Repeat([]byte{'x'}, 1024)}
	case "MatchTail":
		input := []byte("prefix</head>suffix")
		return streamPhaseInput{seed: input, tail: streamBenchWholeOracle(input)}
	case "PartialTail":
		copy(seed[len(seed)-5:], "</hea")
		return streamPhaseInput{seed: seed, tail: seed}
	default:
		panic("unknown phase state")
	}
}

// Preparation validates the emitted prefix against the independent whole-body
// splice oracle. End later checks the full expected tail, once per resource.
func streamPhasePrepare(b *testing.B, library *Library, input streamPhaseInput) *Stream {
	b.Helper()
	stream, err := library.NewStream()
	if err != nil {
		b.Fatal(err)
	}
	var prefix []byte
	for _, chunk := range [][]byte{input.seed, input.prime} {
		if len(chunk) == 0 {
			continue
		}
		output, writeErr := stream.Write(chunk)
		if writeErr != nil {
			b.Fatal(writeErr)
		}
		prefix = append(prefix, output...)
	}
	body := append(append([]byte(nil), input.seed...), input.prime...)
	want := streamBenchWholeOracle(body)
	if len(input.tail) > len(want) || !bytes.Equal(prefix, want[:len(want)-len(input.tail)]) || !bytes.Equal(input.tail, want[len(want)-len(input.tail):]) {
		b.Fatal("prepared stream differs from whole-input oracle")
	}
	return stream
}

func BenchmarkStreamPhases(b *testing.B) {
	b.Run("P1", func(b *testing.B) {
		b.Run("NewClose", streamPhaseNewClose)
		for _, state := range []string{"Window", "Injected"} {
			b.Run("Write"+state, func(b *testing.B) {
				for _, chunk := range []int{64, 4096, 16384} {
					b.Run(fmt.Sprintf("Chunk%d", chunk), func(b *testing.B) { streamPhaseWrites(b, state, chunk) })
				}
			})
		}
		for _, state := range []string{"Empty", "Short", "Window", "Injected", "MatchTail", "PartialTail"} {
			b.Run("End"+state, func(b *testing.B) { streamPhaseGrouped(b, state, false) })
		}
		b.Run("CloseAbandoned", func(b *testing.B) { streamPhaseGrouped(b, "Window", true) })
	})
}

func streamPhaseNewClose(b *testing.B) {
	b.Helper()
	library := streamPhaseOpen(b)
	var guest *guest
	for range streamPhaseGroup {
		stream, err := library.NewStream()
		if err != nil {
			b.Fatal(err)
		}
		guest = stream.guest
		if err := stream.Close(); err != nil {
			b.Fatal(err)
		}
	}
	before := len(guest.module.Memory())
	b.ReportAllocs()
	streamPhaseCheckProcs(b)
	b.ResetTimer()
	for range b.N {
		stream, err := library.NewStream()
		if err != nil {
			b.Fatal(err)
		}
		if err := stream.Close(); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	streamPhaseCheckProcs(b)
	streamPhaseReport(b, "new-close", 1, 0, before, len(guest.module.Memory()))
}

func streamPhaseWrites(b *testing.B, state string, chunk int) {
	b.Helper()
	library := streamPhaseOpen(b)
	stream := streamPhasePrepare(b, library, streamPhaseState(state))
	b.Cleanup(func() {
		if err := stream.Close(); err != nil {
			b.Error(err)
		}
	})
	input := bytes.Repeat([]byte{'x'}, chunk)
	var outputs [streamPhaseGroup][]byte
	// Warm input/reply capacities and prove stable full-window output.
	for index := range outputs {
		output, err := stream.Write(input)
		if err != nil || !bytes.Equal(output, input) {
			b.Fatalf("steady Write: bytes=%d, err=%v", len(output), err)
		}
		outputs[index] = output
	}
	// Force different replies through the same guest buffer before checking
	// retained slices, then restore the same stationary x window before timing.
	different := bytes.Repeat([]byte{'y'}, chunk)
	replaceWindow := func(replacement []byte) {
		var output []byte
		for range (1024+chunk-1)/chunk + 1 {
			var err error
			output, err = stream.Write(replacement)
			if err != nil || len(output) != chunk {
				b.Fatalf("replace window: bytes=%d, err=%v", len(output), err)
			}
		}
		if !bytes.Equal(output, replacement) {
			b.Fatal("replacement did not reach stationary output")
		}
	}
	replaceWindow(different)
	for _, output := range outputs[:] {
		if !bytes.Equal(output, input) {
			b.Fatal("owned steady output changed")
		}
	}
	replaceWindow(input)
	clear(outputs[:])
	before := len(stream.guest.module.Memory())
	b.ReportAllocs()
	b.SetBytes(int64(chunk))
	streamPhaseCheckProcs(b)
	b.ResetTimer()
	next := 0
	for range b.N {
		output, err := stream.Write(input)
		if err != nil || len(output) != chunk {
			b.Fatalf("steady Write: bytes=%d, err=%v", len(output), err)
		}
		outputs[next] = output
		next++
		if next == len(outputs) {
			runtime.KeepAlive(&outputs)
			next = 0
		}
	}
	b.StopTimer()
	streamPhaseCheckProcs(b)
	after := len(stream.guest.module.Memory())
	for _, output := range outputs[:] {
		if output != nil && !bytes.Equal(output, input) {
			b.Fatal("retained steady output changed")
		}
	}
	tail, err := stream.End()
	if err != nil || !bytes.Equal(tail, bytes.Repeat([]byte{'x'}, 1024)) {
		b.Fatalf("steady End: bytes=%d, err=%v", len(tail), err)
	}
	streamPhaseReport(b, "writes", 1, chunk, before, after)
}

func streamPhaseGrouped(b *testing.B, state string, closeOnly bool) {
	b.Helper()
	library := streamPhaseOpen(b)
	input := streamPhaseState(state)
	var streams [streamPhaseGroup]*Stream
	var outputs [streamPhaseGroup][]byte
	prepare := func() {
		for index := range streams {
			streams[index] = streamPhasePrepare(b, library, input)
		}
	}
	verify := func() {
		if !closeOnly && len(input.tail) != 0 {
			// Free one slot before producing a different reply. All End tails
			// must survive both destruction and reuse of guest reply storage.
			if err := streams[0].Close(); err != nil {
				b.Fatal(err)
			}
			probeInput := bytes.Repeat([]byte{'y'}, len(input.tail))
			probe := streamPhasePrepare(b, library, streamPhaseInput{seed: probeInput, tail: probeInput})
			output, err := probe.End()
			if err != nil || !bytes.Equal(output, probeInput) {
				b.Fatalf("ownership probe End: bytes=%d, err=%v", len(output), err)
			}
			if err := probe.Close(); err != nil {
				b.Fatal(err)
			}
		}
		for index, stream := range streams {
			if !closeOnly && !bytes.Equal(outputs[index], input.tail) {
				b.Fatal("owned End tail differs from whole-input oracle")
			}
			if err := stream.Close(); err != nil {
				b.Fatal(err)
			}
		}
		clear(outputs[:])
		clear(streams[:])
	}
	prepare()
	guest := streams[0].guest
	for index, stream := range streams {
		if closeOnly {
			if err := stream.Close(); err != nil {
				b.Fatal(err)
			}
			continue
		}
		output, err := stream.End()
		if err != nil {
			b.Fatal(err)
		}
		outputs[index] = output
		if !bytes.Equal(output, input.tail) {
			b.Fatal("warm End tail differs from whole-input oracle")
		}
	}
	verify()
	before := len(guest.module.Memory())
	b.ReportAllocs()
	b.StopTimer()
	b.ResetTimer()
	for range b.N {
		prepare()
		streamPhaseCheckProcs(b)
		b.StartTimer()
		if closeOnly {
			for _, stream := range streams {
				if err := stream.Close(); err != nil {
					b.Fatal(err)
				}
			}
		} else {
			for index, stream := range streams {
				output, err := stream.End()
				if err != nil {
					b.Fatal(err)
				}
				outputs[index] = output
			}
		}
		b.StopTimer()
		streamPhaseCheckProcs(b)
		verify()
	}
	b.StopTimer()
	unit, owned := "ends", len(input.tail)*streamPhaseGroup
	if closeOnly {
		unit, owned = "closes", 0
	}
	streamPhaseReport(b, unit, streamPhaseGroup, owned, before, len(guest.module.Memory()))
}
