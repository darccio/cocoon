package rt_test

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"dario.cat/cocoon/rt"
)

func TestResultUnitValidation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		want          error
		name          string
		pointer, size uint32
		status        rt.Status
		poison        bool
	}{
		{nil, "empty zero", 0, 0, rt.OK, false},
		{nil, "empty logical end", 64, 0, rt.OK, false},
		{rt.ErrProtocol, "nonempty success", 16, 1, rt.OK, true},
		{rt.ErrProtocol, "empty spare capacity", 65, 0, rt.OK, true},
		{rt.ErrProtocol, "empty maximum pointer", math.MaxUint32, 0, rt.OK, true},
		{rt.ErrProtocol, "overflowing payload", math.MaxUint32, 2, rt.OK, true},
		{rt.ErrProtocol, "truncated payload", 63, 2, rt.OK, true},
		{rt.ErrTooLarge, "output limit", 16, 33, rt.OK, false},
		{rt.ErrTooLarge, "limit before payload bounds", math.MaxUint32, 33, rt.OK, false},
		{rt.ErrTooLarge, "limit before application status", math.MaxUint32, 33, rt.ErrArg, false},
		{rt.ErrTooLarge, "limit before reserved status", math.MaxUint32, 33, rt.Pending, false},
		{rt.ErrProtocol, "reserved status", 16, 0, rt.Pending, true},
		{rt.ErrProtocol, "unknown status", 16, 0, rt.Status(6), true},
		{rt.ErrProtocol, "negative status", 16, 0, rt.Status(-1), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			m := newModule()
			i := newInstance(t, m)
			epoch := i.Epoch()
			binary.LittleEndian.PutUint32(m.memory, test.pointer)
			binary.LittleEndian.PutUint32(m.memory[4:], test.size)
			err := i.Call("unit", func(call *rt.Call) error {
				return call.ResultUnit("unit", test.status)
			})
			if !errors.Is(err, test.want) || i.Healthy() == test.poison || (i.Epoch() != epoch) != test.poison {
				t.Fatalf("unit result = %v, healthy = %v, epoch = %d", err, i.Healthy(), i.Epoch())
			}
		})
	}
}

func TestResultUnitOwnsExpectedErrors(t *testing.T) {
	t.Parallel()
	const message = "bad ☃\x00"
	for _, status := range []rt.Status{rt.ErrApp, rt.ErrArg, rt.ErrHandle, rt.ErrLimit} {
		t.Run(statusName(status), func(t *testing.T) {
			t.Parallel()
			m := newModule()
			i := newInstance(t, m)
			epoch := i.Epoch()
			binary.LittleEndian.PutUint32(m.memory, 16)
			binary.LittleEndian.PutUint32(m.memory[4:], uint32(len(message)))
			copy(m.memory[16:], message)
			err := i.Call("unit", func(call *rt.Call) error {
				return call.ResultUnit("unit", status)
			})
			if err == nil || !i.Healthy() || i.Epoch() != epoch {
				t.Fatalf("expected error = %v, healthy = %v, epoch = %d", err, i.Healthy(), i.Epoch())
			}
			before := err.Error()
			clear(m.memory[16 : 16+len(message)])
			if err.Error() != before {
				t.Fatal("unit error retained borrowed message storage")
			}
			switch status {
			case rt.ErrApp, rt.ErrArg:
				var application *rt.AppError
				if !errors.As(err, &application) || application.Op != "unit" || application.Status != status || application.Message != message {
					t.Fatalf("application error = %v", err)
				}
			case rt.ErrHandle:
				var handle *rt.HandleError
				if !errors.As(err, &handle) || handle.Op != "unit" || handle.Reason != message {
					t.Fatalf("handle error = %v", err)
				}
			case rt.ErrLimit:
				if !errors.Is(err, rt.ErrTooLarge) || err.Error() != "unit: "+rt.ErrTooLarge.Error()+": "+message {
					t.Fatalf("limit error = %v", err)
				}
			case rt.OK, rt.Pending:
				t.Fatal("unexpected successful or reserved error status", status)
			}
		})
	}
}

func statusName(status rt.Status) string {
	if status == rt.ErrApp {
		return "application"
	}
	if status == rt.ErrArg {
		return "argument"
	}
	if status == rt.ErrHandle {
		return "handle"
	}
	return "limit"
}

func TestResultUnitReadsCurrentMemory(t *testing.T) {
	t.Parallel()
	m := &descriptorModule{module: newModule(), pointer: 48}
	i, err := rt.NewInstance(m, 7, rt.Limits{MaxInput: 64, MaxOutput: 32, MaxMemory: 128})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(i.Close)
	// A stale backing store would report a nonempty success after replacement.
	binary.LittleEndian.PutUint32(m.memory[48:], 16)
	binary.LittleEndian.PutUint32(m.memory[52:], 1)
	m.memory = make([]byte, 128)
	binary.LittleEndian.PutUint32(m.memory[48:], 128)
	for range 2 {
		if err := i.Call("unit", func(call *rt.Call) error {
			return call.ResultUnit("unit", rt.OK)
		}); err != nil {
			t.Fatal("unit result read stale memory", err)
		}
	}
	if m.calls != 1 {
		t.Fatalf("output export called %d times, want once", m.calls)
	}
	// Keep spare capacity, but make the cached descriptor cross logical length.
	m.memory = m.memory[:52]
	if err := i.Call("unit", func(call *rt.Call) error {
		return call.ResultUnit("unit", rt.OK)
	}); !errors.Is(err, rt.ErrProtocol) || i.Healthy() {
		t.Fatal("unit descriptor bypassed logical memory bounds", err)
	}
}

func TestResultUnitSuccessDoesNotAllocate(t *testing.T) {
	i := newInstance(t, newModule())
	allocations := testing.AllocsPerRun(100, func() {
		if err := i.Call("unit", func(call *rt.Call) error {
			return call.ResultUnit("unit", rt.OK)
		}); err != nil {
			t.Fatal(err)
		}
	})
	if allocations != 0 {
		t.Fatalf("successful unit result allocated %g times", allocations)
	}
}
