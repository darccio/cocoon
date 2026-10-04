package rt_test

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"cocoon.dev/cocoon/rt"
)

func TestHostCapabilities(t *testing.T) {
	t.Parallel()
	m := newModule()
	h := rt.NewHost(slog.New(slog.NewTextHandler(io.Discard, nil)))
	h.Bind(m)
	h.Random = bytes.NewReader([]byte("seed"))
	if h.RandomGet(16, 4) != 0 || string(m.memory[16:20]) != "seed" {
		t.Fatal("random capability failed")
	}
	if h.RandomGet(16, 4) != 1 || h.RandomGet(64, 1) != 1 {
		t.Fatal("random failure ignored")
	}
	for level := range int32(5) {
		h.Log(level, 16, 4)
	}
	h.Log(4, 64, 1)
	copyData, err := h.Copy(16, 4)
	if err != nil {
		t.Fatal(err)
	}
	m.memory[16] = 'x'
	if string(copyData) != "seed" {
		t.Fatal("host retained guest memory")
	}
	h.Now = func() time.Time { return time.Unix(12, 34) }
	if h.ClockNanos() != 12000000034 {
		t.Fatal("incorrect clock")
	}
}

func TestHostPanicAndGuestMessage(t *testing.T) {
	t.Parallel()
	for _, hostFault := range []bool{false, true} {
		m := newModule()
		copy(m.memory[16:], "panic message")
		h := rt.NewHost(slog.New(slog.NewTextHandler(io.Discard, nil)))
		i, err := rt.NewInstance(m, 7, rt.Limits{MaxInput: 64, MaxOutput: 32, MaxMemory: 128}, h)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(i.Close)
		err = i.Call("panic", func(_ *rt.Call) error {
			if hostFault {
				h.Now = func() time.Time { panic("clock bug") }
				h.ClockNanos()
			}
			h.Log(4, 16, 13)
			panic("unreachable")
		})
		if hostFault {
			var host *rt.HostError
			if !errors.As(err, &host) || host.Value != "clock bug" || host.Error() == "" {
				t.Fatalf("host fault: %v", err)
			}
		} else {
			var fault *rt.FaultError
			if !errors.As(err, &fault) || fault.GuestMsg != "panic message" {
				t.Fatalf("guest fault: %v", err)
			}
		}
	}
}
