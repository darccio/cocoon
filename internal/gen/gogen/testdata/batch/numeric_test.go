package batchfixture

import (
	"bytes"
	"errors"
	"math"
	"testing"

	"dario.cat/cocoon/rt"
)

func TestNumericCurrentReservationAndNeighbors(t *testing.T) {
	t.Parallel()
	_, stream, m, _ := fixtureStream(t, Options{})
	for index := 8; index < 16384; index++ {
		m.memory[index] = 0xcc
	}
	old := m.memory
	before := bytes.Clone(old[8:16384])
	m.reserveBase, m.reserveSwap, m.reserveGrow = 33, true, 65536
	values := []float32{math.Float32frombits(0x80000000), math.Float32frombits(0x7fc00042), math.Float32frombits(1)}
	got, err := stream.Numbers(values)
	if err != nil || len(got) != len(values) {
		t.Fatal(got, err)
	}
	want := []byte{0, 0, 0, 128, 66, 0, 192, 127, 1, 0, 0, 0}
	if !bytes.Equal(m.memory[33:45], want) || m.memory[32] != 0xcc || m.memory[45] != 0xcc || !bytes.Equal(old[8:16384], before) {
		t.Fatal("direct writer used stale memory or touched reservation neighbors")
	}
	for index, value := range values {
		if math.Float32bits(value) != math.Float32bits(got[index]) {
			t.Fatal("direct numeric bits changed")
		}
	}
}

func TestNumericReservationFitBeforeMutation(t *testing.T) {
	t.Parallel()
	_, stream, m, _ := fixtureStream(t, Options{})
	before := bytes.Clone(m.memory[32:48])
	err := stream.guest.instance.Call("undersized reservation", func(call *rt.Call) error {
		if err := call.PrepareInput(4); err != nil {
			return err
		}
		_, err := putSliceU64(call, []uint64{math.MaxUint64}, 8)
		return err
	})
	if !errors.Is(err, rt.ErrTooLarge) || !bytes.Equal(m.memory[32:48], before) || !stream.guest.instance.Healthy() || len(m.operations) != 0 {
		t.Fatal("numeric bytes changed before reservation membership validation", err)
	}
}

func TestNumericReservationFaultStopsCheckedSuffix(t *testing.T) {
	for _, kind := range []string{"logical end", "pointer overflow", "trap", "memory limit"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			_, stream, m, _ := fixtureStream(t, Options{})
			m.reserveAt = 2
			switch kind {
			case "logical end":
				m.reserveBase = 65534 // Spare capacity is not logical memory.
			case "pointer overflow":
				m.reserveBase = math.MaxUint32 - 1
			case "trap":
				m.reserveTrap = true
			case "memory limit":
				m.reserveGrow = 131072
			}
			results, err := stream.Batch(t.Context(), []StreamCommand{StreamNumbersCommand([]float32{1}), StreamNumbersCommand([]float32{2}), StreamUnitCommand()}, rt.BatchLimits{})
			// Existing memory limits are checked after the whole operation;
			// a Reserve growth alone does not preempt its guest invocation.
			attempted, operations := kind == "memory limit", 1
			if attempted {
				operations++
			}
			want := rt.ErrProtocol
			if kind == "memory limit" {
				want = rt.ErrTooLarge
			} else if kind == "trap" {
				want = nil
			}
			batchFailure(t, results, err, 1, 1, attempted, want)
			if kind == "trap" {
				var fault *rt.FaultError
				if !errors.As(err, &fault) || fault.Kind != rt.FaultUnknown || fault.Op != "numbers" || fault.Detail != "deliberate reservation trap" {
					t.Fatal("reservation trap classification changed", err)
				}
			}
			if stream.guest.instance.Healthy() || len(m.operations) != operations || m.reserveCalls != 2 {
				t.Fatal("reservation fault retained home or invoked command/suffix", err)
			}
			if got := resultAt[StreamNumbersResult](t, results, 0).Value; len(got) != 1 || got[0] != 1 {
				t.Fatal("reservation fault damaged owned prefix", got)
			}
		})
	}
}

func TestNumericLateRecordValidationPrecedesReserve(t *testing.T) {
	t.Parallel()
	_, stream, m, _ := fixtureStream(t, Options{})
	bad := Envelope{Leaf: Leaf{Label: string([]byte{0xff})}}
	if _, err := stream.Mix(0, 0, 0, 0, 0, 0, false, "", nil, nil, nil, nil, nil, []float32{1}, []float64{2}, bad); err == nil || m.reserveCalls != 0 || len(m.operations) != 0 {
		t.Fatal("late invalid record reached numeric reservation", err)
	}
	results, err := stream.Batch(t.Context(), []StreamCommand{StreamNumbersCommand([]float32{3}), minimalMix(bad), StreamUnitCommand()}, rt.BatchLimits{})
	batchFailure(t, results, err, 1, 1, false, nil)
	if m.reserveCalls != 1 || len(m.operations) != 1 || !stream.guest.instance.Healthy() {
		t.Fatal("record preparation changed numeric prefix or reservation order", err)
	}
	if _, err := stream.Numbers([]float32{4}); err != nil {
		t.Fatal("expected validation error damaged home", err)
	}
}

func TestNumericMixedConstructorCursor(t *testing.T) {
	t.Parallel()
	library := fixtureLibrary(t, Options{})
	vectors, err := library.NewVectors([]uint32{math.MaxUint32}, "x", []uint64{math.MaxUint64})
	if err != nil {
		t.Fatal(err)
	}
	seen := vectors.guest.module.constructor
	if !bytes.Equal(seen["first"], bytes.Repeat([]byte{0xff}, 4)) || string(seen["text"]) != "x" || !bytes.Equal(seen["last"], bytes.Repeat([]byte{0xff}, 8)) {
		t.Fatal("constructor numeric/string/numeric cursor overlapped", seen)
	}
	if err := vectors.Close(); err != nil {
		t.Fatal(err)
	}
	empty, err := library.NewVectors(nil, "", []uint64{})
	if err != nil {
		t.Fatal("adjacent empty constructor inputs", err)
	}
	if err := empty.Close(); err != nil {
		t.Fatal(err)
	}
}
