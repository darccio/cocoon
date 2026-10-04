package compute

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log/slog"
	"math"
	"reflect"
	"strings"
	"testing"

	"cocoon.dev/cocoon/cocoontest"
	"cocoon.dev/cocoon/rt"
)

func TestAllValueTypesAndAggregateLimits(t *testing.T) {
	t.Parallel()
	l, openErr := Open(Options{Instances: 1})
	if openErr != nil {
		t.Fatal(openErr)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Error(err)
		}
	})
	numbers := Numbers{Signed32: -42, Unsigned32: math.MaxUint32, Signed64: math.MinInt64, Unsigned64: math.MaxUint64, Float32: -1.25, Float64: 2.5, Enabled: true}
	got, scalarErr := l.ScalarValues(t.Context(), numbers.Signed32, numbers.Unsigned32, numbers.Signed64, numbers.Unsigned64, numbers.Float32, numbers.Float64, numbers.Enabled)
	if scalarErr != nil || got != numbers {
		t.Fatalf("scalar round trip: %+v %v", got, scalarErr)
	}
	payload := Payload{Label: "hello\x00世界\nkey=value", Data: []byte{0, 255}, Numbers: numbers, Signed32: []int32{-1, math.MinInt32}, Unsigned32: []uint32{0, math.MaxUint32}, Signed64: []int64{-1, math.MinInt64}, Unsigned64: []uint64{0, math.MaxUint64}, Float32: []float32{-1.5, 2.5}, Float64: []float64{math.Inf(1), math.SmallestNonzeroFloat64}}
	result, bounceErr := l.Bounce(t.Context(), payload)
	if bounceErr != nil || !reflect.DeepEqual(result, payload) {
		t.Fatalf("record round trip: %+v %v", result, bounceErr)
	}
	payload.Data[0] = 77
	if result.Data[0] != 0 {
		t.Fatal("output aliases input")
	}
	joined, joinErr := l.Join(t.Context(), "left", "right")
	if joinErr != nil || joined != "leftright" {
		t.Fatal(joined, joinErr)
	}
	small, smallErr := Open(Options{MaxInput: 3, MaxOutput: 2, Instances: 1})
	if smallErr != nil {
		t.Fatal(smallErr)
	}
	t.Cleanup(func() {
		if err := small.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := small.Join(t.Context(), "ab", "cd"); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal(err)
	}
	if _, err := small.Echo(t.Context(), "abc"); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal(err)
	}
}

func TestRealRustPanicPoisonsHomeAndReplacesPool(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	l, openErr := Open(Options{Instances: 1, Logger: slog.New(slog.NewTextHandler(&logs, nil))})
	if openErr != nil {
		t.Fatal(openErr)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Error(err)
		}
	})
	var fault *rt.FaultError
	if err := l.TestTrap(t.Context(), 1); !errors.As(err, &fault) || fault.Kind != rt.FaultUnreachable || !strings.Contains(fault.GuestMsg, "deliberate safe Rust trap") {
		t.Fatalf("panic hook: %v", err)
	}
	if !strings.Contains(logs.String(), "deliberate safe Rust trap") {
		t.Fatal("missing panic log")
	}
	if value, err := l.Echo(t.Context(), "after trap"); err != nil || value != "after trap" {
		t.Fatal(value, err)
	}
	counter, counterErr := l.NewCounter(10)
	if counterErr != nil {
		t.Fatal(counterErr)
	}
	faultErr := counter.guest.instance.Call("guest panic", func(*rt.Call) error { counter.guest.module.Xcocoon_test_trap(1); return nil })
	if !errors.As(faultErr, &fault) {
		t.Fatal(faultErr)
	}
	var handle *rt.HandleError
	if _, err := counter.Count(); !errors.As(err, &handle) {
		t.Fatal(err)
	}
	replacement, replacementErr := l.NewCounter(20)
	if replacementErr != nil {
		t.Fatal(replacementErr)
	}
	if value, err := replacement.Add(2); err != nil || value != 22 {
		t.Fatal(value, err)
	}
	if err := replacement.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPositiveDifferentialSequences(t *testing.T) {
	t.Parallel()
	g, reference := newDifferential(t)
	payload := Payload{Label: "nested", Data: []byte{255, 0}, Numbers: Numbers{Enabled: true}, Float64: []float64{math.NaN(), math.Inf(-1), math.Copysign(0, -1)}}
	encoded, err := encodePayload(payload, generatedInputLimit)
	if err != nil {
		t.Fatal(err)
	}
	compareOperation(t, g, reference, "cocoon_bounce", []cocoontest.Arg{{Buffer: true, Bytes: encoded}})
	compareOperation(t, g, reference, "cocoon_join", []cocoontest.Arg{{Buffer: true, Bytes: []byte("a")}, {Buffer: true, Bytes: []byte("b")}})
	_, handle := compareOperation(t, g, reference, "cocoon_counter_new", []cocoontest.Arg{{Value: 7}})
	h := binary.LittleEndian.Uint64(handle)
	compareOperation(t, g, reference, "cocoon_counter_add", []cocoontest.Arg{{Value: h}, {Value: 5}})
	compareOperation(t, g, reference, "cocoon_counter_count", []cocoontest.Arg{{Value: h}})
	compareOperation(t, g, reference, "cocoon_counter_close", []cocoontest.Arg{{Value: h}})
	compareOperation(t, g, reference, "cocoon_counter_count", []cocoontest.Arg{{Value: h}})
}
