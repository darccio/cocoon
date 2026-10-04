package rt_test

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"cocoon.dev/cocoon/rt"
)

type module struct {
	memory  []byte
	schema  uint64
	version uint32
	trap    bool
}

func (m *module) Memory() []byte  { return m.memory }
func (m *module) Version() uint32 { return m.version }
func (m *module) Schema() uint64  { return m.schema }
func (m *module) Init() {
	if m.trap {
		panic("unreachable")
	}
}

func (m *module) Reserve(n uint32) uint32 {
	if n > 16 {
		m.memory = make([]byte, 128)
	}
	return 16
}
func (*module) Output() uint32 { return 0 }
func (*module) Trim(_ uint32)  {}

func newModule() *module { return &module{memory: make([]byte, 64, 128), version: 3, schema: 7} }
func newInstance(t *testing.T, m *module) *rt.Instance {
	t.Helper()
	i, err := rt.NewInstance(m, 7, rt.Limits{MaxInput: 64, MaxOutput: 32, MaxMemory: 128})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(i.Close)
	return i
}

func TestRange(t *testing.T) {
	t.Parallel()
	memory := make([]byte, 10, 100)
	for _, test := range []struct {
		pointer, size uint32
		valid         bool
	}{
		{0, 10, true}, {10, 0, true}, {11, 0, false}, {9, 2, false}, {math.MaxUint32, 2, false},
	} {
		view, err := rt.Range(memory, test.pointer, test.size)
		if (err == nil) != test.valid {
			t.Fatalf("Range(%d,%d) = %v", test.pointer, test.size, err)
		}
		if err == nil && cap(view) != len(view) {
			t.Fatal("view can access spare capacity")
		}
	}
}

func TestCallResetsReservationWithoutAllocating(t *testing.T) {
	i := newInstance(t, newModule())
	if err := i.Call("reserve", func(call *rt.Call) error { return call.PrepareInput(4) }); err != nil {
		t.Fatal(err)
	}
	if err := i.Call("fresh", func(call *rt.Call) error { _, putErr := call.PutBytes([]byte{1}); return putErr }); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal("reservation leaked between calls", err)
	}
	allocations := testing.AllocsPerRun(100, func() {
		if err := i.Call("empty", func(*rt.Call) error { return nil }); err != nil {
			t.Fatal(err)
		}
	})
	if allocations != 0 {
		t.Fatalf("empty call allocated %g times", allocations)
	}
}

func TestInputSize(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		other        []uint64
		count, width uint64
		valid        bool
	}{
		{nil, 8, 8, true},
		{nil, 9, 8, false},
		{nil, math.MaxUint64, 8, false},
		{[]uint64{32}, 4, 8, true},
		{[]uint64{33}, 4, 8, false},
		{[]uint64{math.MaxUint64}, 0, 1, false},
	} {
		_, err := rt.InputSize(64, test.count, test.width, test.other...)
		if (err == nil) != test.valid {
			t.Fatalf("InputSize(%+v) = %v", test, err)
		}
	}
}

func TestIdentityAndInitialization(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		want    error
		schema  uint64
		version uint32
		trap    bool
	}{
		{rt.ErrABI, 7, 2, false}, {rt.ErrSchema, 8, 3, false}, {nil, 7, 3, true},
	} {
		m := newModule()
		m.version, m.schema, m.trap = test.version, test.schema, test.trap
		_, err := rt.NewInstance(m, 7, rt.Limits{MaxInput: 64, MaxOutput: 32, MaxMemory: 128})
		if test.trap {
			var fault *rt.FaultError
			if !errors.As(err, &fault) {
				t.Fatal(err)
			}
		} else if !errors.Is(err, test.want) {
			t.Fatal(err)
		}
	}
	if _, err := rt.NewInstance(newModule(), 7, rt.Limits{}); err == nil {
		t.Fatal("invalid limits accepted")
	}
}

func TestInputGrowthAndReply(t *testing.T) {
	t.Parallel()
	m := newModule()
	i := newInstance(t, m)
	var output []byte
	err := i.Call("echo", func(call *rt.Call) error {
		if err := call.PrepareInput(32); err != nil {
			return err
		}
		ptr, err := call.PutBytes([]byte("hello"))
		if err != nil {
			return err
		}
		binary.LittleEndian.PutUint32(m.memory, ptr)
		binary.LittleEndian.PutUint32(m.memory[4:], 5)
		output, err = call.Result("echo", rt.OK)
		return err
	})
	if err != nil || string(output) != "hello" {
		t.Fatalf("%q: %v", output, err)
	}
	m.memory[16] = 'x'
	if string(output) != "hello" {
		t.Fatal("reply aliases guest memory")
	}
	if !i.Recycle(0, 128) || i.Recycle(0, 64) {
		t.Fatal("incorrect recycling")
	}
}

func TestInstanceFaultAndClose(t *testing.T) {
	t.Parallel()
	i := newInstance(t, newModule())
	epoch := i.Epoch()
	if !i.Healthy() {
		t.Fatal("new instance is unhealthy")
	}
	err := i.Call("fault", func(_ *rt.Call) error { panic(42) })
	var fault *rt.FaultError
	if !errors.As(err, &fault) || fault.Kind != rt.FaultUnknown || epoch == i.Epoch() || i.Healthy() {
		t.Fatalf("fault = %v", err)
	}
	if err := i.Call("again", func(_ *rt.Call) error { t.Fatal("executed after fault"); return nil }); !errors.Is(err, rt.ErrPoisoned) {
		t.Fatal(err)
	}
	i.Close()
	i.Close()
	if err := i.Call("again", nil); !errors.Is(err, rt.ErrClosed) {
		t.Fatal(err)
	}
}

func TestReplyAndInputFailures(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		want          error
		size, pointer uint32
		status        rt.Status
	}{
		{rt.ErrTooLarge, 33, 16, rt.OK}, {rt.ErrProtocol, 20, 60, rt.OK}, {rt.ErrProtocol, 0, 16, rt.Pending},
	} {
		m := newModule()
		i := newInstance(t, m)
		binary.LittleEndian.PutUint32(m.memory, test.pointer)
		binary.LittleEndian.PutUint32(m.memory[4:], test.size)
		err := i.Call("reply", func(call *rt.Call) error { _, err := call.Result("reply", test.status); return err })
		if !errors.Is(err, test.want) {
			t.Fatalf("reply = %v", err)
		}
	}
	i := newInstance(t, newModule())
	if err := i.Call("large", func(call *rt.Call) error { return call.PrepareInput(65) }); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal(err)
	}
	if err := i.Call("overrun", func(call *rt.Call) error {
		returnErr := call.PrepareInput(1)
		if returnErr != nil {
			return returnErr
		}
		_, err := call.PutBytes([]byte("xx"))
		return err
	}); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal(err)
	}
}
