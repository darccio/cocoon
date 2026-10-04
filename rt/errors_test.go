package rt_test

import (
	"errors"
	"testing"

	"cocoon.dev/cocoon/rt"
)

func TestClassify(t *testing.T) {
	t.Parallel()
	tests := []struct {
		value any
		kind  rt.FaultKind
	}{
		{"unreachable", rt.FaultUnreachable},
		{"integer overflow", rt.FaultArithmetic},
		{"invalid conversion to integer", rt.FaultArithmetic},
		{"out of bounds memory access", rt.FaultMemory},
		{"uninitialized element", rt.FaultIndirectCall},
		{"indirect call type mismatch", rt.FaultIndirectCall},
		{"out of bounds table access", rt.FaultTable},
		{"unexpected", rt.FaultUnknown},
		{42, rt.FaultUnknown},
		{nil, rt.FaultUnknown},
	}
	for _, tt := range tests {
		err := rt.Classify("call", tt.value, "guest panic")
		var fault *rt.FaultError
		if !errors.As(err, &fault) || fault.Kind != tt.kind || fault.GuestMsg != "guest panic" || err.Error() == "" {
			t.Fatalf("Classify(%v) = %v", tt.value, err)
		}
		if tt.kind.String() == "" {
			t.Fatal("empty kind description")
		}
	}
	if rt.FaultKind(255).String() != "unknown" {
		t.Fatal("unknown enum value")
	}
}

func TestRuntimePanics(t *testing.T) {
	t.Parallel()
	tests := []struct {
		trap func()
		kind rt.FaultKind
	}{
		{func() { divisor := 0; _ = 1 / divisor }, rt.FaultArithmetic},
		{func() { b := []byte{}; _ = b[1] }, rt.FaultMemory},                   //nolint:gosec // Intentionally trigger a real Go bounds panic.
		{func() { b := []byte{}; _ = b[:1] }, rt.FaultMemory},                  //nolint:gosec // Intentionally trigger a real Go slice panic.
		{func() { var x any = 1; f := x.(func()); f() }, rt.FaultIndirectCall}, //nolint:errcheck,forcetypeassert // Intentionally trigger a signature panic.
	}
	for _, tt := range tests {
		func() {
			defer func() {
				err := rt.Classify("trap", recover(), "")
				var fault *rt.FaultError
				if !errors.As(err, &fault) || fault.Kind != tt.kind || err.Error() == "" {
					t.Errorf("unexpected classification: %v", err)
				}
			}()
			tt.trap()
		}()
	}
}

func TestStatuses(t *testing.T) {
	t.Parallel()
	if rt.FromStatus("op", rt.OK, nil) != nil {
		t.Fatal("OK returned error")
	}
	for _, status := range []rt.Status{rt.ErrApp, rt.ErrArg, rt.ErrHandle, rt.ErrLimit, rt.Pending, 99} {
		err := rt.FromStatus("op", status, []byte("message"))
		if err == nil || err.Error() == "" {
			t.Fatalf("status %d missing error", status)
		}
		if status == rt.ErrLimit && !errors.Is(err, rt.ErrTooLarge) {
			t.Fatal("limit error not wrapped")
		}
	}
}
