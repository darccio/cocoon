package batchfixture

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"runtime"
	"slices"
	"sync"
	"testing"
	"time"

	"dario.cat/cocoon/rt"
)

func fixtureStream(t *testing.T, options Options) (*Library, *Stream, *module, uint64) {
	t.Helper()
	library, err := Open(options)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := library.Close(); err != nil {
			t.Error(err)
		}
	})
	stream, err := library.NewStream()
	if err != nil {
		t.Fatal(err)
	}
	var handle uint64
	if err := stream.owner.Use("fixture", func(_ *rt.Instance, value uint64) error { handle = value; return nil }); err != nil {
		t.Fatal(err)
	}
	return library, stream, stream.guest.module, handle
}

func resultAt[T any](t *testing.T, results []StreamResult, index int) T {
	t.Helper()
	if index >= len(results) {
		t.Fatalf("missing result %d in %d results", index, len(results))
	}
	value, ok := results[index].(T)
	if !ok {
		t.Fatalf("result %d has dynamic type %T", index, results[index])
	}
	return value
}

func batchFailure(t *testing.T, results []StreamResult, err error, completed, index int, attempted bool, want error) *rt.BatchError {
	t.Helper()
	var failure *rt.BatchError
	if len(results) != completed || cap(results) != len(results) || !errors.As(err, &failure) || failure.Index != index || failure.Attempted != attempted || want != nil && !errors.Is(err, want) {
		t.Fatalf("prefix len/cap=%d/%d, failure=%v; want completed=%d index=%d attempted=%v cause=%v", len(results), cap(results), err, completed, index, attempted, want)
	}
	return failure
}

func canonicalEnvelope(t *testing.T, value Envelope) []byte {
	t.Helper()
	flag := byte(0)
	if value.Leaf.Flag {
		flag = 1
	}
	leaf, err := rt.EncodeRecord([]rt.Field{{ID: 1, Data: []byte(value.Leaf.Label)}, {ID: 2, Data: value.Leaf.Data}, {ID: 3, Data: []byte{flag}}}, 4096)
	if err != nil {
		t.Fatal(err)
	}
	var values []byte
	for _, v := range value.Values {
		values = binary.LittleEndian.AppendUint64(values, v)
	}
	data, err := rt.EncodeRecord([]rt.Field{{ID: 1, Data: leaf}, {ID: 2, Data: values}, {ID: 3, Data: binary.LittleEndian.AppendUint64(nil, math.Float64bits(value.Score))}}, 4096)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func minimalMix(value Envelope) StreamCommand {
	return StreamMixCommand(0, 0, 0, 0, 0, 0, false, "", nil, nil, nil, nil, nil, nil, nil, value)
}

func TestGeneratedBatchMixedInputsAndTaggedResults(t *testing.T) {
	t.Parallel()
	_, stream, m, handle := fixtureStream(t, Options{})
	data := []byte("old")
	envelope := Envelope{Leaf: Leaf{Label: "世界", Data: data, Flag: true}, Values: []uint64{math.MaxUint64, 7}, Score: math.Float64frombits(0x7ff8000000000042)}
	single := math.Float32frombits(0x80000000)
	double := math.Float64frombits(0x8000000000000000)
	numbers := []float32{math.Float32frombits(0x7fc00042), single, 1.25}
	commands := []StreamCommand{
		StreamEchoCommand(data), StreamTextCommand("héllo"), StreamCountCommand(math.MaxUint64), StreamFlagCommand(true), StreamFloatCommand(double), StreamNumbersCommand(numbers), StreamUnitCommand(),
		StreamMixCommand(math.MinInt32, math.MaxUint32, math.MinInt64, math.MaxUint64, single, double, true, "Σ", data, []int32{-1, math.MinInt32}, []uint32{math.MaxUint32, 3}, []int64{math.MinInt64, -2}, []uint64{math.MaxUint64, 4}, numbers, []float64{double, 2.5}, envelope),
	}
	copy(data, "new") // Constructors retain payloads; they are read only at Batch time.
	envelope.Values[1] = 9
	results, err := stream.Batch(t.Context(), commands, rt.BatchLimits{})
	if err != nil || len(results) != len(commands) || cap(results) != len(results) {
		t.Fatal(results, err)
	}
	if string(resultAt[StreamEchoResult](t, results, 0).Value) != "new" || resultAt[StreamTextResult](t, results, 1).Value != "héllo" || resultAt[StreamCountResult](t, results, 2).Value != math.MaxUint64 || !resultAt[StreamFlagResult](t, results, 3).Value || math.Float64bits(resultAt[StreamFloatResult](t, results, 4).Value) != math.Float64bits(double) {
		t.Fatal("scalar or variable tagged reply changed")
	}
	gotNumbers := resultAt[StreamNumbersResult](t, results, 5).Value
	if len(gotNumbers) != len(numbers) {
		t.Fatal(gotNumbers)
	}
	for index, value := range numbers {
		if math.Float32bits(gotNumbers[index]) != math.Float32bits(value) {
			t.Fatal("float array bits changed")
		}
	}
	_ = resultAt[StreamUnitResult](t, results, 6)
	if _, hasValue := reflect.TypeFor[StreamUnitResult]().FieldByName("Value"); hasValue {
		t.Fatal("unit result exposes a fabricated Value")
	}
	gotEnvelope := resultAt[StreamMixResult](t, results, 7).Value
	if !bytes.Equal(canonicalEnvelope(t, gotEnvelope), canonicalEnvelope(t, envelope)) {
		t.Fatal("nested result wire value changed")
	}
	if handle != 0xfedcba98_00000001 || !slices.Equal(m.handles, []uint64{handle, handle, handle, handle, handle, handle, handle, handle}) {
		t.Fatal("adapter lost bound uint64 handle bits", m.handles)
	}
	seen := m.mixed
	if seen.signed32 != math.MinInt32 || seen.unsigned32 != math.MaxUint32 || seen.signed64 != math.MinInt64 || seen.unsigned64 != math.MaxUint64 || math.Float32bits(seen.single) != math.Float32bits(single) || math.Float64bits(seen.double) != math.Float64bits(double) || seen.flag != 1 {
		t.Fatal("scalar export arguments changed", seen)
	}
	want := map[string][]byte{
		"text": []byte("Σ"), "data": []byte("new"),
		"signed32s":   binary.LittleEndian.AppendUint32(binary.LittleEndian.AppendUint32(nil, math.MaxUint32), 0x80000000),
		"unsigned32s": binary.LittleEndian.AppendUint32(binary.LittleEndian.AppendUint32(nil, math.MaxUint32), 3),
		"signed64s":   binary.LittleEndian.AppendUint64(binary.LittleEndian.AppendUint64(nil, 0x8000000000000000), math.MaxUint64-1),
		"unsigned64s": binary.LittleEndian.AppendUint64(binary.LittleEndian.AppendUint64(nil, math.MaxUint64), 4),
		"envelope":    canonicalEnvelope(t, envelope),
	}
	for _, value := range numbers {
		want["singles"] = binary.LittleEndian.AppendUint32(want["singles"], math.Float32bits(value))
	}
	for _, value := range []float64{double, 2.5} {
		want["doubles"] = binary.LittleEndian.AppendUint64(want["doubles"], math.Float64bits(value))
	}
	if !reflect.DeepEqual(seen.buffers, want) {
		t.Fatal("independent input-buffer oracle mismatch", seen.buffers, want)
	}
	// Four commands have buffers; scalar and unit commands must not reserve input.
	if m.reserveCalls != 4 {
		t.Fatal("scalar command reserved guest input", m.reserveCalls)
	}
}

func TestGeneratedBatchPreflightHasNoGuestEffects(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		commands []StreamCommand
		limits   rt.BatchLimits
		index    int
		want     error
	}{
		{"individual size", []StreamCommand{StreamUnitCommand(), StreamEchoCommand(make([]byte, 4097))}, rt.BatchLimits{}, 1, rt.ErrTooLarge},
		{"aggregate size", []StreamCommand{StreamEchoCommand([]byte("abc")), StreamEchoCommand([]byte("def"))}, rt.BatchLimits{MaxInputBytes: 5}, 1, rt.ErrTooLarge},
		{"scalar-array width", []StreamCommand{StreamUnitCommand(), StreamNumbersCommand(make([]float32, 1025))}, rt.BatchLimits{}, 1, rt.ErrTooLarge},
		{"nested-record size", []StreamCommand{StreamUnitCommand(), minimalMix(Envelope{Leaf: Leaf{Data: make([]byte, 4096)}})}, rt.BatchLimits{}, 1, rt.ErrTooLarge},
		{"nil later command", []StreamCommand{StreamUnitCommand(), nil}, rt.BatchLimits{}, 1, nil},
		{"call hard maximum", make([]StreamCommand, 257), rt.BatchLimits{}, -1, rt.ErrTooLarge},
		{"limit hard maximum", nil, rt.BatchLimits{MaxCalls: 257}, -1, rt.ErrTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, stream, m, _ := fixtureStream(t, Options{})
			results, err := stream.Batch(t.Context(), test.commands, test.limits)
			failure := batchFailure(t, results, err, 0, test.index, false, test.want)
			if test.want == nil {
				var argument *rt.AppError
				if !errors.As(failure, &argument) || argument.Status != rt.ErrArg {
					t.Fatal("invalid command type changed classification", failure)
				}
			}
			if len(m.operations) != 0 || m.reserveCalls != 0 || m.newCalls != 1 || !stream.guest.instance.Healthy() {
				t.Fatal("preflight reached guest or poisoned home")
			}
		})
	}
}

func TestGeneratedBatchMiddleEncodingErrorReturnsOwnedPrefix(t *testing.T) {
	t.Parallel()
	for _, record := range []bool{false, true} {
		t.Run(map[bool]string{false: "string", true: "nested record"}[record], func(t *testing.T) {
			t.Parallel()
			_, stream, m, _ := fixtureStream(t, Options{})
			bad := StreamTextCommand(string([]byte{0xff}))
			if record {
				bad = minimalMix(Envelope{Leaf: Leaf{Label: string([]byte{0xff})}})
			}
			results, err := stream.Batch(t.Context(), []StreamCommand{StreamEchoCommand([]byte("prefix")), bad, StreamCountCommand(7)}, rt.BatchLimits{})
			failure := batchFailure(t, results, err, 1, 1, false, nil)
			var argument *rt.AppError
			if !errors.As(failure, &argument) || argument.Op != "encode" || argument.Status != rt.ErrArg || !stream.guest.instance.Healthy() || !slices.Equal(m.operations, []string{"echo"}) || string(resultAt[StreamEchoResult](t, results, 0).Value) != "prefix" {
				t.Fatal("encoding failure lost prefix/effects", err, m.operations)
			}
			reused, reuseErr := stream.Batch(t.Context(), []StreamCommand{StreamTextCommand("healthy")}, rt.BatchLimits{})
			if reuseErr != nil || resultAt[StreamTextResult](t, reused, 0).Value != "healthy" {
				t.Fatal("healthy home could not be reused", reuseErr)
			}
			if string(resultAt[StreamEchoResult](t, results, 0).Value) != "prefix" {
				t.Fatal("later reply mutated retained prefix")
			}
		})
	}
}

func TestGeneratedBatchTypedFaultWinsExhaustedBudget(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		command StreamCommand
		payload []byte
	}{
		{"bool", StreamFlagCommand(true), []byte{2}},
		{"float width", StreamFloatCommand(1), make([]byte, 7)},
		{"array width", StreamNumbersCommand(nil), []byte{1, 2, 3}},
		{"UTF-8", StreamTextCommand("x"), []byte{0xff}},
		{"nested record", minimalMix(Envelope{}), []byte{0, 0, 0, 0}},
		{"unit", StreamUnitCommand(), []byte{1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, stream, m, _ := fixtureStream(t, Options{})
			m.replies = []fixtureReply{{}, {override: true, data: test.payload}}
			results, err := stream.Batch(t.Context(), []StreamCommand{StreamEchoCommand([]byte{1}), test.command, StreamUnitCommand()}, rt.BatchLimits{MaxOutputBytes: 1})
			batchFailure(t, results, err, 1, 1, true, rt.ErrProtocol)
			if stream.guest.instance.Healthy() || len(m.operations) != 2 || !bytes.Equal(resultAt[StreamEchoResult](t, results, 0).Value, []byte{1}) {
				t.Fatal("fault executed suffix or published malformed result")
			}
		})
	}
}

func TestGeneratedBatchExpectedErrorBudgetAndOperationLimit(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		budget uint64
		want   error
	}{
		{"expected error retained", 4, nil}, {"expected error exceeds remainder", 1, rt.ErrTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, stream, m, _ := fixtureStream(t, Options{})
			m.replies = []fixtureReply{{}, {override: true, data: []byte("bad"), status: rt.ErrApp}}
			results, err := stream.Batch(t.Context(), []StreamCommand{StreamEchoCommand([]byte{1}), StreamUnitCommand(), StreamUnitCommand()}, rt.BatchLimits{MaxOutputBytes: test.budget})
			failure := batchFailure(t, results, err, 1, 1, true, test.want)
			var app *rt.AppError
			if test.want == nil {
				if !errors.As(failure, &app) || app.Message != "bad" {
					t.Fatal("expected error lost message", failure)
				}
			} else if errors.As(failure, &app) {
				t.Fatal("oversized error retained application payload")
			}
			if !stream.guest.instance.Healthy() || len(m.operations) != 2 {
				t.Fatal("healthy error faulted home or ran suffix")
			}
			clear(m.memory[16384:])
			if app != nil && app.Message != "bad" {
				t.Fatal("expected error borrowed guest memory")
			}
			if results, err := stream.Batch(t.Context(), []StreamCommand{StreamUnitCommand()}, rt.BatchLimits{}); err != nil || len(results) != 1 {
				t.Fatal("expected-error home not reusable", err)
			}
		})
	}
	// The constructor itself needs an eight-byte handle reply.
	_, stream, m, _ := fixtureStream(t, Options{MaxOutput: 8})
	m.replies = []fixtureReply{{override: true, data: bytes.Repeat([]byte{0xff}, 9)}}
	results, err := stream.Batch(t.Context(), []StreamCommand{StreamTextCommand("x")}, rt.BatchLimits{MaxOutputBytes: 1})
	batchFailure(t, results, err, 0, 0, true, rt.ErrTooLarge)
	if !stream.guest.instance.Healthy() {
		t.Fatal("existing MaxOutput precedence changed")
	}
}

func TestGeneratedBatchPostMemoryFailureClearsStaging(t *testing.T) {
	t.Parallel()
	_, stream, m, _ := fixtureStream(t, Options{})
	m.replies = []fixtureReply{{}, {afterSize: 196608}}
	results, err := stream.Batch(t.Context(), []StreamCommand{StreamEchoCommand([]byte("prefix")), StreamEchoCommand(bytes.Repeat([]byte("failed"), 100)), StreamUnitCommand()}, rt.BatchLimits{})
	batchFailure(t, results, err, 1, 1, true, rt.ErrTooLarge)
	if stream.guest.instance.Healthy() || len(m.operations) != 2 || string(resultAt[StreamEchoResult](t, results, 0).Value) != "prefix" {
		t.Fatal("post-memory failure published staging or ran suffix")
	}
	// cap==len above ensures callers cannot reslice into the failed result slot.
	defer func() {
		if recover() == nil {
			t.Error("failed staged result accessible through returned capacity")
		}
	}()
	_ = results[:2]
}

func TestGeneratedBatchRepliesSurviveMemoryGrowth(t *testing.T) {
	t.Parallel()
	_, stream, m, _ := fixtureStream(t, Options{})
	m.replies = []fixtureReply{{}, {grow: true}}
	results, err := stream.Batch(t.Context(), []StreamCommand{StreamEchoCommand([]byte("retained")), StreamEchoCommand([]byte("second"))}, rt.BatchLimits{})
	if err != nil || len(results) != 2 || cap(results) != 2 || len(m.memory) != 131072 {
		t.Fatal(results, err)
	}
	clear(m.memory[16384:])
	if string(resultAt[StreamEchoResult](t, results, 0).Value) != "retained" || string(resultAt[StreamEchoResult](t, results, 1).Value) != "second" {
		t.Fatal("typed results retained borrowed views")
	}
}

func TestGeneratedBatchCancellationPreservesCompletedCommand(t *testing.T) {
	t.Parallel()
	for _, last := range []bool{false, true} {
		t.Run(map[bool]string{false: "before next", true: "during last"}[last], func(t *testing.T) {
			t.Parallel()
			_, stream, m, _ := fixtureStream(t, Options{})
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			m.cancelAt = 1
			m.cancel = cancel
			commands := []StreamCommand{StreamEchoCommand([]byte("done"))}
			if !last {
				commands = append(commands, StreamUnitCommand())
			}
			results, err := stream.Batch(ctx, commands, rt.BatchLimits{})
			if last {
				if err != nil || len(results) != 1 {
					t.Fatal("final cancellation replaced success", err)
				}
			} else {
				batchFailure(t, results, err, 1, 1, false, context.Canceled)
			}
			if len(m.operations) != 1 || string(resultAt[StreamEchoResult](t, results, 0).Value) != "done" {
				t.Fatal("cancellation lost current result or started next command")
			}
		})
	}
}

func awaitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case <-signal:
	case <-timer.C:
		t.Fatal("fixture did not signal")
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
}
func awaitError(t *testing.T, result <-chan error) error {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case err := <-result:
		return err
	case <-timer.C:
		t.Fatal("fixture did not complete")
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	return nil
}
func awaitClosing(t *testing.T, library *Library) {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	for {
		err := library.life.Enter()
		if errors.Is(err, rt.ErrClosed) {
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		library.life.Leave()
		select {
		case <-timer.C:
			t.Fatal("library admission not terminal")
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		default:
			runtime.Gosched()
		}
	}
}
func blockedBatch(t *testing.T, stream *Stream, m *module) (func(), <-chan error) {
	t.Helper()
	started, release := make(chan struct{}), make(chan struct{})
	m.started, m.release, m.blockAt = started, release, 1
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	done := make(chan error, 1)
	go func() {
		results, err := stream.Batch(t.Context(), []StreamCommand{StreamEchoCommand([]byte("a")), StreamEchoCommand([]byte("b"))}, rt.BatchLimits{})
		if err == nil && len(results) != 2 {
			err = errors.New("admitted batch lost results")
		}
		done <- err
	}()
	awaitSignal(t, started)
	return unblock, done
}

func TestGeneratedBatchLibraryCloseDrainsWholeBuffer(t *testing.T) {
	t.Parallel()
	library, stream, m, _ := fixtureStream(t, Options{})
	guest := stream.guest
	unblock, done := blockedBatch(t, stream, m)
	closing := make(chan error, 1)
	go func() { closing <- library.Close() }()
	awaitClosing(t, library)
	results, err := stream.Batch(t.Context(), nil, rt.BatchLimits{})
	batchFailure(t, results, err, 0, -1, false, rt.ErrClosed)
	select {
	case err := <-closing:
		t.Fatal("Close released admitted buffer", err)
	default:
	}
	unblock()
	if err := awaitError(t, done); err != nil {
		t.Fatal(err)
	}
	if err := awaitError(t, closing); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(m.operations, []string{"echo", "echo"}) || guest.instance.Healthy() || guest.module != nil || library.homes[0].guest != nil {
		t.Fatal("Close failed to drain and release exact home")
	}
}

func TestGeneratedBatchAliasCloseWaitsForWholeBuffer(t *testing.T) {
	t.Parallel()
	_, stream, m, _ := fixtureStream(t, Options{})
	copyValue := reflect.New(reflect.TypeOf(stream).Elem())
	copyValue.Elem().Set(reflect.ValueOf(stream).Elem())
	alias, ok := copyValue.Interface().(*Stream)
	if !ok {
		t.Fatal("alias changed type")
	}
	unblock, done := blockedBatch(t, stream, m)
	closing := make(chan error, 1)
	go func() { closing <- alias.Close() }()
	select {
	case err := <-closing:
		t.Fatal("alias Close passed active ownership", err)
	default:
	}
	unblock()
	if err := awaitError(t, done); err != nil {
		t.Fatal(err)
	}
	closeErr := awaitError(t, closing)
	if closeErr != nil || stream.Close() != closeErr || m.closeCalls != 1 || len(m.operations) != 2 {
		t.Fatal("alias destructor did not execute exactly once after batch", closeErr)
	}
	results, err := stream.Batch(t.Context(), []StreamCommand{StreamUnitCommand()}, rt.BatchLimits{})
	failure := batchFailure(t, results, err, 0, -1, false, nil)
	var invalid *rt.HandleError
	if !errors.As(failure, &invalid) || len(m.operations) != 2 {
		t.Fatal("closed ownership reached guest", failure)
	}
}

// With no admissible concrete commands, empty buffers still validate ownership,
// cancellation and terminal admission; generation must emit no unused locals.
func TestGeneratedIdleBatchLifecycle(t *testing.T) {
	t.Parallel()
	library, _, m, _ := fixtureStream(t, Options{})
	idle, err := library.NewIdle()
	if err != nil {
		t.Fatal(err)
	}
	results, err := idle.Batch(t.Context(), nil, rt.BatchLimits{})
	if err != nil || len(results) != 0 || cap(results) != 0 || len(m.operations) != 0 || m.reserveCalls != 0 {
		t.Fatal(results, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	results, err = idle.Batch(ctx, nil, rt.BatchLimits{})
	var failure *rt.BatchError
	if !errors.Is(err, context.Canceled) || !errors.As(err, &failure) || failure.Index != -1 || failure.Attempted || len(results) != 0 {
		t.Fatal("empty idle buffer ignored cancellation", err)
	}
	aliasValue := reflect.New(reflect.TypeOf(idle).Elem())
	aliasValue.Elem().Set(reflect.ValueOf(idle).Elem())
	alias, ok := aliasValue.Interface().(*Idle)
	if !ok {
		t.Fatal("idle alias changed type")
	}
	if err := idle.Close(); err != nil || alias.Close() != err || m.closeCalls != 1 {
		t.Fatal("idle Close did not share once-only ownership", err)
	}
	results, err = alias.Batch(t.Context(), nil, rt.BatchLimits{})
	var invalid *rt.HandleError
	if !errors.As(err, &failure) || failure.Index != -1 || failure.Attempted || !errors.As(err, &invalid) || len(results) != 0 {
		t.Fatal("empty idle buffer bypassed closed ownership", err)
	}
	fresh, err := library.NewIdle()
	if err != nil {
		t.Fatal(err)
	}
	if err := library.Close(); err != nil {
		t.Fatal(err)
	}
	results, err = fresh.Batch(t.Context(), nil, rt.BatchLimits{})
	if !errors.Is(err, rt.ErrClosed) || !errors.As(err, &failure) || failure.Index != -1 || failure.Attempted || len(results) != 0 {
		t.Fatal("empty idle buffer bypassed terminal admission", err)
	}
}

func TestGeneratedScalarOnlyBatch(t *testing.T) {
	t.Parallel()
	library, _, m, _ := fixtureStream(t, Options{})
	scalar, err := library.NewScalar()
	if err != nil {
		t.Fatal(err)
	}
	var handle uint64
	if err := scalar.owner.Use("fixture", func(_ *rt.Instance, value uint64) error { handle = value; return nil }); err != nil {
		t.Fatal(err)
	}
	results, err := scalar.Batch(t.Context(), []ScalarCommand{ScalarCountCommand(math.MaxUint64), ScalarUnitCommand(), ScalarCountCommand(1)}, rt.BatchLimits{MaxInputBytes: 1})
	if err != nil || len(results) != 3 || cap(results) != 3 {
		t.Fatal(results, err)
	}
	first, ok := results[0].(ScalarCountResult)
	if !ok || first.Value != math.MaxUint64 {
		t.Fatal("scalar-only batch lost u64 bits", results[0])
	}
	if _, ok := results[1].(ScalarUnitResult); !ok {
		t.Fatal("scalar-only unit tag changed", results[1])
	}
	last, ok := results[2].(ScalarCountResult)
	if !ok || last.Value != 1 || m.reserveCalls != 0 || !slices.Equal(m.handles, []uint64{handle, handle, handle}) || !slices.Equal(m.operations, []string{"scalar_count", "scalar_unit", "scalar_count"}) {
		t.Fatal("scalar-only batch reserved input or lost ordering", m.operations)
	}
}
