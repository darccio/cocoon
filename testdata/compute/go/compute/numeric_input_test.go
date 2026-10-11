package compute

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math"
	"strings"
	"testing"

	"dario.cat/cocoon/cocoontest"
	"dario.cat/cocoon/rt"
)

func numericInputLibrary(t *testing.T, limit uint64) *Library {
	t.Helper()
	library, err := Open(Options{Instances: 1, MaxInput: limit})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := library.Close(); err != nil {
			t.Error(err)
		}
	})
	return library
}

func numericInputPayload() Payload {
	return Payload{
		Label: "x", Data: []byte{0, 255, 1},
		Signed32:   []int32{-1, math.MinInt32, 0, math.MaxInt32},
		Unsigned32: []uint32{0, 1, math.MaxUint32},
		Signed64:   []int64{math.MinInt64, -1, 0, math.MaxInt64},
		Unsigned64: []uint64{0, 1, math.MaxUint64},
		Float32:    []float32{0, math.Float32frombits(0x80000000), float32(math.Inf(1)), float32(math.Inf(-1)), math.Float32frombits(0x7fc00042), math.Float32frombits(0x7f800042), math.SmallestNonzeroFloat32},
		Float64:    []float64{0, math.Float64frombits(0x8000000000000000), math.Inf(1), math.Inf(-1), math.Float64frombits(0x7ff8000000000042), math.Float64frombits(0x7ff0000000000042), math.SmallestNonzeroFloat64},
	}
}

func bounceNumericInput(t *testing.T, library *Library, payload *Payload) (Payload, error) {
	t.Helper()
	return library.BounceSlices(t.Context(), payload.Label, payload.Signed32, payload.Data, payload.Unsigned32, payload.Numbers, payload.Signed64, payload.Unsigned64, payload.Float32, payload.Float64)
}

func numericInputArguments(t *testing.T, payload *Payload) []cocoontest.Arg {
	t.Helper()
	// Independently authored byte goldens, including signaling NaN payloads.
	wires := []string{
		"ffffffff0000008000000000ffffff7f", "0000000001000000ffffffff",
		"0000000000000080ffffffffffffffff0000000000000000ffffffffffffff7f", "00000000000000000100000000000000ffffffffffffffff",
		"00000000000000800000807f000080ff4200c07f4200807f01000000",
		"00000000000000000000000000000080000000000000f07f000000000000f0ff420000000000f87f420000000000f07f0100000000000000",
	}
	parts := make([][]byte, len(wires))
	for index, wire := range wires {
		var err error
		parts[index], err = hex.DecodeString(wire)
		if err != nil {
			t.Fatal(err)
		}
	}
	numbers, err := encodeNumbers(payload.Numbers, generatedInputLimit)
	if err != nil {
		t.Fatal(err)
	}
	return []cocoontest.Arg{{Buffer: true, Bytes: []byte(payload.Label)}, {Buffer: true, Bytes: parts[0]}, {Buffer: true, Bytes: payload.Data}, {Buffer: true, Bytes: parts[1]}, {Buffer: true, Bytes: numbers}, {Buffer: true, Bytes: parts[2]}, {Buffer: true, Bytes: parts[3]}, {Buffer: true, Bytes: parts[4]}, {Buffer: true, Bytes: parts[5]}}
}

func numericPayloadWire(t *testing.T, payload *Payload) []byte {
	t.Helper()
	data, err := encodePayload(*payload, generatedOutputLimit)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDirectNumericInputsAgainstWasmGoldens(t *testing.T) {
	t.Parallel()
	library := numericInputLibrary(t, 0)
	g, reference := newDifferential(t)
	payload := numericInputPayload()
	args := numericInputArguments(t, &payload)
	_, want := compareOperation(t, g, reference, "cocoon_bounce_slices", args)
	actual, err := bounceNumericInput(t, library, &payload)
	if err != nil || !bytes.Equal(numericPayloadWire(t, &actual), want) {
		t.Fatal("typed direct input differs from independent wire goldens", err)
	}
	retained := numericPayloadWire(t, &actual)
	for _, next := range []Payload{{}, {Label: "new", Signed32: make([]int32, 8192), Numbers: Numbers{Enabled: true}}, {Label: "z", Data: []byte{9}, Float64: []float64{42}}} {
		result, err := bounceNumericInput(t, library, &next)
		if err != nil || !bytes.Equal(numericPayloadWire(t, &result), numericPayloadWire(t, &next)) {
			t.Fatal("empty/reused/large mixed reservation changed bytes", err)
		}
	}
	if !bytes.Equal(numericPayloadWire(t, &actual), retained) {
		t.Fatal("retained numeric result aliases reused guest memory")
	}
	// Every numeric buffer rejects a malformed tail in the original Wasm.
	for _, index := range []int{1, 3, 5, 6, 7, 8} {
		malformed := append([]cocoontest.Arg(nil), args...)
		malformed[index].Bytes = append(bytes.Clone(args[index].Bytes), 0)
		status, _ := compareOperation(t, g, reference, "cocoon_bounce_slices", malformed)
		if status != rt.ErrArg {
			t.Fatal("malformed numeric width accepted", index, status)
		}
	}
}

func TestDirectNumericAggregateLimitsAndLateText(t *testing.T) {
	t.Parallel()
	payload := numericInputPayload()
	var total uint64
	for _, arg := range numericInputArguments(t, &payload) {
		total += uint64(len(arg.Bytes))
	}
	exact := numericInputLibrary(t, total)
	if _, err := bounceNumericInput(t, exact, &payload); err != nil {
		t.Fatal("exact aggregate reservation rejected", err)
	}
	small := numericInputLibrary(t, total-1)
	if _, err := bounceNumericInput(t, small, &payload); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal("aggregate limit skipped", err)
	}
	payload.Unsigned64 = append(payload.Unsigned64, 7)
	if _, err := bounceNumericInput(t, exact, &payload); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal("one-element excess accepted", err)
	}
	library := numericInputLibrary(t, 0)
	payload.Label = string([]byte{0xff})
	var argument *rt.AppError
	if _, err := bounceNumericInput(t, library, &payload); !errors.As(err, &argument) || argument.Status != rt.ErrArg {
		t.Fatal("invalid mixed text accepted", err)
	}
	payload.Label = strings.Repeat("a", 32)
	if _, err := bounceNumericInput(t, library, &payload); err != nil {
		t.Fatal("invalid text damaged pool reuse", err)
	}
}

func TestDirectNumericResourceAndCheckedCommands(t *testing.T) {
	t.Parallel()
	library := numericInputLibrary(t, 0)
	counter, err := library.NewCounter(7)
	if err != nil {
		t.Fatal(err)
	}
	if count, sumErr := counter.Sum([]uint64{2, 3}); sumErr != nil || count != 12 {
		t.Fatal(count, sumErr)
	}
	results, err := counter.Batch(t.Context(), []CounterCommand{CounterSumCommand(nil), CounterSumCommand([]uint64{1, 2}), CounterCountCommand()}, rt.BatchLimits{})
	if err != nil || len(results) != 3 {
		t.Fatal(results, err)
	}
	for index, want := range []uint64{12, 15} {
		result, ok := results[index].(CounterSumResult)
		if !ok || result.Value != want {
			t.Fatal("numeric Sum result tag/value", index, results[index])
		}
	}
	result, ok := results[2].(CounterCountResult)
	if !ok || result.Value != 15 {
		t.Fatal("numeric Count result tag/value", results[2])
	}
	if count, countErr := counter.Count(); countErr != nil || count != 15 {
		t.Fatal(count, countErr)
	}
	results, err = counter.Batch(t.Context(), []CounterCommand{CounterSumCommand([]uint64{1}), CounterSumCommand([]uint64{1, math.MaxUint64}), CounterSumCommand([]uint64{9})}, rt.BatchLimits{})
	var failure *rt.BatchError
	var app *rt.AppError
	if len(results) != 1 || !errors.As(err, &failure) || failure.Index != 1 || !failure.Attempted || !errors.As(err, &app) || app.Op != "sum" || app.Status != rt.ErrApp || app.Message != "counter overflow" {
		t.Fatal("numeric failure lost checked prefix", results, err)
	}
	if count, err := counter.Count(); err != nil || count != 16 {
		t.Fatal("overflowing tail or suffix mutated counter", count, err)
	}
	if count, err := counter.Sum([]uint64{2}); err != nil || count != 18 {
		t.Fatal("expected overflow damaged resource", count, err)
	}
	if err := counter.Close(); err != nil {
		t.Fatal(err)
	}
	g, reference := newDifferential(t)
	_, data := compareOperation(t, g, reference, "cocoon_counter_new", []cocoontest.Arg{{Value: 7}})
	handle := binary.LittleEndian.Uint64(data)
	for index, encoded := range [][]byte{{2, 0, 0, 0, 0, 0, 0, 0}, {1}, {1, 0, 0, 0, 0, 0, 0, 0, 255, 255, 255, 255, 255, 255, 255, 255}, nil} {
		status, _ := compareOperation(t, g, reference, "cocoon_counter_sum", []cocoontest.Arg{{Value: handle}, {Buffer: true, Bytes: encoded}})
		if status != []rt.Status{rt.OK, rt.ErrArg, rt.ErrApp, rt.OK}[index] {
			t.Fatal("raw numeric Sum sequence status", index, status)
		}
		_, count := compareOperation(t, g, reference, "cocoon_counter_count", []cocoontest.Arg{{Value: handle}})
		if len(count) != 8 || binary.LittleEndian.Uint64(count) != 9 {
			t.Fatal("raw numeric Sum sequence changed expected counter", index, count)
		}
	}
	compareOperation(t, g, reference, "cocoon_counter_close", []cocoontest.Arg{{Value: handle}})
}
