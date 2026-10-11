package streaming

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"testing"

	"dario.cat/cocoon/cocoontest"
	"dario.cat/cocoon/rt"
)

const streamWindow = 1024

var (
	streamPattern = []byte("</head>")
	streamSnippet = []byte("<cocoon>")
)

// Whole-input splice is independent of the guest's tail and emit algorithm.
func streamOracle(input []byte) []byte {
	index := bytes.Index(input, streamPattern)
	if index < 0 {
		return bytes.Clone(input)
	}
	output := make([]byte, 0, len(input)+len(streamSnippet))
	output = append(output, input[:index]...)
	output = append(output, streamSnippet...)
	return append(output, input[index:]...)
}

func openStreaming(tb testing.TB, options Options) *Library {
	tb.Helper()
	library, err := Open(options)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if closeErr := library.Close(); closeErr != nil {
			tb.Error(closeErr)
		}
	})
	return library
}

func newStream(tb testing.TB, library *Library) *Stream {
	tb.Helper()
	stream, err := library.NewStream()
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if closeErr := stream.Close(); closeErr != nil && !errors.Is(closeErr, rt.ErrClosed) && !errors.Is(closeErr, rt.ErrPoisoned) {
			tb.Error(closeErr)
		}
	})
	return stream
}

func streamResultBytes(tb testing.TB, result StreamResult) []byte {
	tb.Helper()
	switch value := result.(type) {
	case StreamWriteResult:
		return value.Value
	case StreamEndResult:
		return value.Value
	default:
		tb.Fatalf("unexpected streaming result %T", result)
		return nil
	}
}

func assertStreamFailure(tb testing.TB, results []StreamResult, err error, count, index int, attempted bool) *rt.BatchError {
	tb.Helper()
	var failure *rt.BatchError
	if len(results) != count || !errors.As(err, &failure) || failure.Index != index || failure.Attempted != attempted {
		tb.Fatalf("results=%d error=%v; want count=%d index=%d attempted=%t", len(results), err, count, index, attempted)
	}
	return failure
}

func TestStreamingPatternPartitionsAndOwnedResults(t *testing.T) {
	t.Parallel()
	library := openStreaming(t, Options{Instances: 1})
	var cases [][][]byte
	for split := 0; split <= len(streamPattern); split++ {
		cases = append(cases, [][]byte{append(bytes.Repeat([]byte("a"), 31), streamPattern[:split]...), {}, append(bytes.Clone(streamPattern[split:]), bytes.Repeat([]byte("b"), 2*streamWindow)...)})
	}
	for cutoff := 0; cutoff <= len(streamPattern); cutoff++ {
		input := append(bytes.Repeat([]byte("a"), 31), streamPattern...)
		input = append(input, bytes.Repeat([]byte("b"), streamWindow-len(streamPattern)+cutoff)...)
		cases = append(cases, [][]byte{input})
	}
	for _, input := range [][]byte{nil, []byte("ordinary"), []byte("</hea"), []byte("</head></head>"), []byte("<</head></hea</head>"), {0, 255, 128, '<', '/', 'h', 'e', 'a', 'd', '>', 255, 0}, bytes.Repeat([]byte("x"), streamWindow), bytes.Repeat([]byte("x"), 65536)} {
		cases = append(cases, [][]byte{input})
	}
	byteInput := []byte("prefix</head>suffix</hea")
	var singleBytes [][]byte
	for index := range byteInput {
		singleBytes = append(singleBytes, byteInput[index:index+1])
	}
	cases = append(cases, singleBytes)
	for index, chunks := range cases {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			t.Parallel()
			input := bytes.Join(chunks, nil)
			want := streamOracle(input)
			standalone := newStream(t, library)
			var standaloneResults [][]byte
			for _, chunk := range chunks {
				output, err := standalone.Write(chunk)
				if err != nil {
					t.Fatal(err)
				}
				standaloneResults = append(standaloneResults, output)
			}
			end, err := standalone.End()
			if err != nil {
				t.Fatal(err)
			}
			standaloneResults = append(standaloneResults, end)
			batched := newStream(t, library)
			commands := make([]StreamCommand, 0, len(chunks)+1)
			for _, chunk := range chunks {
				commands = append(commands, StreamWriteCommand(chunk))
			}
			commands = append(commands, StreamEndCommand())
			results, err := batched.Batch(t.Context(), commands, rt.BatchLimits{})
			if err != nil || len(results) != len(commands) {
				t.Fatal(len(results), err)
			}
			var batchOutput []byte
			for _, result := range results {
				batchOutput = append(batchOutput, streamResultBytes(t, result)...)
			}
			// These writes reused the same home memory after every standalone result.
			if !bytes.Equal(bytes.Join(standaloneResults, nil), want) || !bytes.Equal(batchOutput, want) {
				t.Fatalf("stream mismatch: standalone=%x batch=%x want=%x", bytes.Join(standaloneResults, nil), batchOutput, want)
			}
			var argument *rt.AppError
			if _, endErr := batched.End(); !errors.As(endErr, &argument) || argument.Status != rt.ErrArg {
				t.Fatal("End after End", endErr)
			}
			if _, writeErr := batched.Write(nil); !errors.As(writeErr, &argument) || argument.Status != rt.ErrArg || !batched.guest.instance.Healthy() {
				t.Fatal("Write after End", writeErr)
			}
			batchOutput = nil
			for _, result := range results {
				batchOutput = append(batchOutput, streamResultBytes(t, result)...)
			}
			if !bytes.Equal(batchOutput, want) {
				t.Fatal("retained batch results changed after guest error replies")
			}
		})
	}
}

func TestStreamingCommandsReadPayloadAtBatchTime(t *testing.T) {
	t.Parallel()
	stream := newStream(t, openStreaming(t, Options{Instances: 1}))
	chunk := []byte("before</head>")
	command := StreamWriteCommand(chunk)
	copy(chunk, "after!")
	results, err := stream.Batch(t.Context(), []StreamCommand{command, StreamEndCommand()}, rt.BatchLimits{})
	if err != nil || len(results) != 2 || !bytes.Equal(streamResultBytes(t, results[1]), streamOracle(chunk)) {
		t.Fatal(results, err)
	}
}

func TestStreamingBatchExpectedFailurePreservesPrefix(t *testing.T) {
	t.Parallel()
	stream := newStream(t, openStreaming(t, Options{Instances: 1}))
	chunk := bytes.Repeat([]byte("a"), 2*streamWindow)
	results, err := stream.Batch(t.Context(), []StreamCommand{StreamWriteCommand(chunk), StreamFaultCommand(1), StreamWriteCommand([]byte("must not run"))}, rt.BatchLimits{})
	failure := assertStreamFailure(t, results, err, 1, 1, true)
	var application *rt.AppError
	if !errors.As(failure, &application) || application.Status != rt.ErrApp || application.Message != "deliberate streaming error" || !stream.guest.instance.Healthy() {
		t.Fatal(failure)
	}
	tail, err := stream.End()
	if err != nil || !bytes.Equal(append(bytes.Clone(streamResultBytes(t, results[0])), tail...), chunk) {
		t.Fatal("expected error mutated or continued stream", err)
	}
	if faultErr := stream.Fault(0); faultErr != nil {
		t.Fatal("healthy expected-error home could not be reused", faultErr)
	}
}

// Err observes this test's sole guest only at operation boundaries. Canceling
// when the first 1KiB reply appears avoids relying on the number of context polls.
type cancelAfterStreamWrite struct {
	context.Context //nolint:containedctx // Test-only observer cancels between completed guest operations.
	stream          *Stream
	cancel          context.CancelFunc
	output          uint32
}

func TestStreamingBatchLimitsAndPartialEffects(t *testing.T) {
	t.Parallel()
	stream := newStream(t, openStreaming(t, Options{Instances: 1}))
	results, err := stream.Batch(t.Context(), []StreamCommand{StreamWriteCommand(bytes.Repeat([]byte("a"), streamWindow)), StreamWriteCommand(bytes.Repeat([]byte("b"), streamWindow)), StreamEndCommand()}, rt.BatchLimits{MaxOutputBytes: 1})
	if !errors.Is(assertStreamFailure(t, results, err, 1, 1, true), rt.ErrTooLarge) || len(streamResultBytes(t, results[0])) != 0 || !stream.guest.instance.Healthy() {
		t.Fatal("output-budget failure", err)
	}
	// The caller cannot expose failed-command staging by reslicing the prefix.
	// The generated adapter separately clears the backing array's unused slots.
	if cap(results) != len(results) {
		t.Fatal("successful prefix exposes failed-command result capacity")
	}
	tail, err := stream.End()
	if err != nil || !bytes.Equal(tail, bytes.Repeat([]byte("b"), streamWindow)) {
		t.Fatal("failing export effect was lost or End ran after failure", err)
	}
	preflight := newStream(t, stream.library)
	results, err = preflight.Batch(t.Context(), []StreamCommand{StreamWriteCommand(bytes.Repeat([]byte("a"), 2*streamWindow)), StreamWriteCommand(make([]byte, 65537))}, rt.BatchLimits{})
	if !errors.Is(assertStreamFailure(t, results, err, 0, 1, false), rt.ErrTooLarge) {
		t.Fatal(err)
	}
	if output, endErr := preflight.End(); endErr != nil || len(output) != 0 {
		t.Fatal("later preflight failure ran earlier guest command", endErr)
	}
}

func TestStreamingCloseDiscardsAbandonedTail(t *testing.T) {
	t.Parallel()
	library := openStreaming(t, Options{Instances: 1})
	abandoned := newStream(t, library)
	if output, err := abandoned.Write([]byte("unfinished</hea")); err != nil || len(output) != 0 {
		t.Fatal(output, err)
	}
	if err := abandoned.Close(); err != nil {
		t.Fatal(err)
	}
	reused := newStream(t, library)
	if output, err := reused.End(); err != nil || len(output) != 0 {
		t.Fatal("abandoned tail migrated to a reused slot", output, err)
	}
	var handle *rt.HandleError
	if _, err := abandoned.End(); !errors.As(err, &handle) {
		t.Fatal("abandoned handle remained live", err)
	}
}

func (c cancelAfterStreamWrite) Err() error {
	descriptor, err := rt.Range(c.stream.guest.module.Memory(), c.output, 8)
	if err != nil {
		return err
	}
	if binary.LittleEndian.Uint32(descriptor[4:]) == streamWindow {
		c.cancel()
	}
	return c.Context.Err()
}

func TestStreamingBatchCancellationRetainsCompletedWrite(t *testing.T) {
	t.Parallel()
	stream := newStream(t, openStreaming(t, Options{Instances: 1}))
	base, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := cancelAfterStreamWrite{Context: base, stream: stream, cancel: cancel}
	if observeErr := stream.guest.instance.Call("test output observation", func(*rt.Call) error {
		ctx.output = stream.guest.module.Output()
		return nil
	}); observeErr != nil {
		t.Fatal(observeErr)
	}
	chunk := bytes.Repeat([]byte("a"), 2*streamWindow)
	results, err := stream.Batch(ctx, []StreamCommand{StreamWriteCommand(chunk), StreamWriteCommand([]byte("must not run")), StreamEndCommand()}, rt.BatchLimits{})
	if !errors.Is(assertStreamFailure(t, results, err, 1, 1, false), context.Canceled) {
		t.Fatal(err)
	}
	tail, err := stream.End()
	if err != nil || !bytes.Equal(append(bytes.Clone(streamResultBytes(t, results[0])), tail...), chunk) {
		t.Fatal("cancellation discarded or continued stream", err)
	}
	results, err = stream.Batch(base, nil, rt.BatchLimits{})
	if !errors.Is(assertStreamFailure(t, results, err, 0, -1, false), context.Canceled) {
		t.Fatal(err)
	}
}

func TestStreamingTrapPoisonsOnlyBoundHomeAndNeverMigrates(t *testing.T) {
	t.Parallel()
	library := openStreaming(t, Options{Instances: 1, ResourceInstances: 2})
	faulted := newStream(t, library)
	otherHome := newStream(t, library)
	sameHome := newStream(t, library)
	if faulted.guest != sameHome.guest || faulted.guest == otherHome.guest {
		t.Fatal("round-robin homes did not bind exact instances")
	}
	results, err := faulted.Batch(t.Context(), []StreamCommand{StreamWriteCommand(bytes.Repeat([]byte("a"), 2*streamWindow)), StreamFaultCommand(2), StreamEndCommand()}, rt.BatchLimits{})
	failure := assertStreamFailure(t, results, err, 1, 1, true)
	var fault *rt.FaultError
	if !errors.As(failure, &fault) || fault.Kind != rt.FaultUnreachable || !strings.Contains(fault.GuestMsg, "deliberate streaming trap") {
		t.Fatal(failure)
	}
	var handle *rt.HandleError
	if _, writeErr := sameHome.Write(nil); !errors.As(writeErr, &handle) {
		t.Fatal("same-home resource survived trap", writeErr)
	}
	if output, endErr := otherHome.End(); endErr != nil || len(output) != 0 {
		t.Fatal("other home was poisoned", endErr)
	}
	_ = newStream(t, library) // Advance past the healthy slot to the poisoned slot.
	replacement := newStream(t, library)
	if replacement.guest == faulted.guest || !replacement.guest.instance.Healthy() {
		t.Fatal("poisoned slot did not get a new instance")
	}
	if _, endErr := faulted.End(); !errors.As(endErr, &handle) {
		t.Fatal("old handle migrated to replacement", endErr)
	}
	if echo, echoErr := library.Echo(t.Context(), []byte("pool after home trap")); echoErr != nil || string(echo) != "pool after home trap" {
		t.Fatal("home trap affected stateless pool", echoErr)
	}
	if !bytes.Equal(streamResultBytes(t, results[0]), bytes.Repeat([]byte("a"), streamWindow)) {
		t.Fatal("completed prefix changed after poisoning/replacement")
	}
}

func streamingDifferential(tb testing.TB, input, partition []byte) {
	tb.Helper()
	g, reference := newDifferential(tb)
	status, encodedHandle := compareOperation(tb, g, reference, "cocoon_stream_new", nil)
	if status != rt.OK || len(encodedHandle) != 8 {
		tb.Fatal("constructor reply", status, encodedHandle)
	}
	handle := binary.LittleEndian.Uint64(encodedHandle)
	if handle == 0 || handle>>32 == 0 {
		tb.Fatal("constructor returned an invalid generation handle", handle)
	}
	var output []byte
	var retained [][]byte
	for offset, index := 0, 0; offset < len(input) || index == 0; index++ {
		width := 257
		if len(partition) != 0 {
			width = int(partition[index%len(partition)]) + 1
		}
		end := min(len(input), offset+width)
		args := []cocoontest.Arg{{Value: handle}, {Buffer: true, Bytes: input[offset:end]}}
		writeStatus, reply := compareOperation(tb, g, reference, "cocoon_stream_write", args)
		if writeStatus != rt.OK {
			tb.Fatal("valid stateful write failed", writeStatus)
		}
		retained = append(retained, reply)
		output = append(output, reply...)
		if len(output) < end-streamWindow || len(output) > end+len(streamSnippet) {
			tb.Fatal("wire output is inconsistent with the bounded holdback", end, len(output))
		}
		offset = end
		if offset == len(input) {
			break
		}
	}
	for _, kind := range []uint64{0, 1, 3} {
		got, _ := compareOperation(tb, g, reference, "cocoon_stream_fault", []cocoontest.Arg{{Value: handle}, {Value: kind}})
		want := []rt.Status{rt.OK, rt.ErrApp, rt.ErrArg}[min(kind, 2)]
		if got != want {
			tb.Fatal("expected stateful status", kind, got, want)
		}
	}
	endStatus, tail := compareOperation(tb, g, reference, "cocoon_stream_end", []cocoontest.Arg{{Value: handle}})
	if endStatus != rt.OK || !bytes.Equal(append(output, tail...), streamOracle(input)) || !bytes.Equal(bytes.Join(retained, nil), output) {
		tb.Fatal("stateful differential oracle mismatch", endStatus)
	}
	if got, _ := compareOperation(tb, g, reference, "cocoon_stream_write", []cocoontest.Arg{{Value: handle}, {Buffer: true}}); got != rt.ErrArg {
		tb.Fatal("write after end status", got)
	}
	if got, _ := compareOperation(tb, g, reference, "cocoon_stream_end", []cocoontest.Arg{{Value: handle}}); got != rt.ErrArg {
		tb.Fatal("end after end status", got)
	}
	if got, _ := compareOperation(tb, g, reference, "cocoon_stream_close", []cocoontest.Arg{{Value: handle}}); got != rt.OK {
		tb.Fatal("close status", got)
	}
	if got, _ := compareOperation(tb, g, reference, "cocoon_stream_write", []cocoontest.Arg{{Value: handle}, {Buffer: true}}); got != rt.ErrHandle {
		tb.Fatal("stale handle status", got)
	}
	_, reused := compareOperation(tb, g, reference, "cocoon_stream_new", nil)
	if len(reused) != 8 || binary.LittleEndian.Uint64(reused) == handle || uint32(binary.LittleEndian.Uint64(reused)) != uint32(handle) { // #nosec G115 -- Deliberately compare the low 32-bit slab index.
		tb.Fatal("closed slab slot was not reused with a new generation", handle, reused)
	}
	if got, _ := compareOperation(tb, g, reference, "cocoon_stream_end", []cocoontest.Arg{{Value: handle}}); got != rt.ErrHandle {
		tb.Fatal("old handle resolved after slot reuse", got)
	}
	if got, _ := compareOperation(tb, g, reference, "cocoon_stream_close", []cocoontest.Arg{{Value: binary.LittleEndian.Uint64(reused)}}); got != rt.OK {
		tb.Fatal("reused handle close", got)
	}
}

func TestStreamingStatefulDifferential(t *testing.T) {
	t.Parallel()
	for _, input := range [][]byte{nil, []byte("prefix</head>suffix</head>"), {0, 255, '<', '/', 'h', 'e', 'a', 'd', '>', 0}, bytes.Repeat([]byte("</hea"), 511), bytes.Repeat([]byte("x"), 65536)} {
		streamingDifferential(t, input, []byte{0, 6, 255, 3, 128})
	}
}

func FuzzStreamingDifferential(f *testing.F) {
	f.Add([]byte{}, []byte{})
	f.Add([]byte("prefix</head>suffix"), []byte{0, 1, 6, 255})
	f.Add([]byte{0, 255, '<', '/', 'h', 'e', 'a', 'd', '>', 0}, []byte{3, 0})
	f.Fuzz(func(t *testing.T, input, partition []byte) {
		if len(input) > 4096 || len(partition) > 64 {
			t.Skip()
		}
		streamingDifferential(t, input, partition)
	})
}
