package rt_test

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"github.com/darccio/cocoon/rt"
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

func TestPutStringSharesBoundedInputReservation(t *testing.T) {
	t.Parallel()
	m := newModule()
	i := newInstance(t, m)
	if err := i.Call("strings", func(call *rt.Call) error {
		if err := call.PrepareInput(5); err != nil {
			return err
		}
		left, err := call.PutString("abc")
		if err != nil {
			return err
		}
		right, err := call.PutString("de")
		if err != nil {
			return err
		}
		if right != left+3 || string(m.memory[left:right+2]) != "abcde" {
			t.Fatal("incorrect aggregate string copy")
		}
		if _, err := call.PutString("f"); !errors.Is(err, rt.ErrTooLarge) {
			t.Fatal("string crossed reservation", err)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
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

type descriptorModule struct {
	*module
	calls   int
	pointer uint32
	trap    bool
}

func (m *descriptorModule) Output() uint32 {
	m.calls++
	if m.trap {
		panic("unreachable")
	}
	return m.pointer
}

func TestOutputDescriptorInitialization(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		pointer uint32
		trap    bool
	}{
		{"past logical length", 60, false},
		{"overflow", math.MaxUint32, false},
		{"trap", 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			m := &descriptorModule{module: newModule(), pointer: test.pointer, trap: test.trap}
			i, err := rt.NewInstance(m, 7, rt.Limits{MaxInput: 64, MaxOutput: 32, MaxMemory: 128})
			if i != nil {
				i.Close()
				t.Fatal("accepted invalid output descriptor")
			}
			var fault *rt.FaultError
			if test.trap && !errors.As(err, &fault) || !test.trap && !errors.Is(err, rt.ErrProtocol) {
				t.Fatalf("descriptor initialization: %v", err)
			}
		})
	}
}

func TestCachedOutputDescriptorReadsCurrentMemory(t *testing.T) {
	t.Parallel()
	m := &descriptorModule{module: newModule(), pointer: 48}
	i, err := rt.NewInstance(m, 7, rt.Limits{MaxInput: 64, MaxOutput: 32, MaxMemory: 128})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(i.Close)
	// Replace the backing store, as guest memory growth can do. Only the address,
	// not a slice into the old store, may be cached between calls.
	m.memory = make([]byte, 128)
	binary.LittleEndian.PutUint32(m.memory[48:], 80)
	binary.LittleEndian.PutUint32(m.memory[52:], 5)
	copy(m.memory[80:], "grown")
	for range 2 {
		if err := i.Call("grown", func(call *rt.Call) error {
			data, resultErr := call.Result("grown", rt.OK)
			if resultErr == nil && string(data) != "grown" {
				t.Fatal("read stale output storage", string(data))
			}
			return resultErr
		}); err != nil {
			t.Fatal(err)
		}
	}
	if m.calls != 1 {
		t.Fatalf("output export called %d times, want once", m.calls)
	}
	// An adversarial module can shrink its logical memory while retaining spare
	// capacity. The cached descriptor still requires bounds checking each time.
	m.memory = m.memory[:52]
	if err := i.Call("shrunk", func(call *rt.Call) error {
		_, resultErr := call.Result("shrunk", rt.OK)
		return resultErr
	}); !errors.Is(err, rt.ErrProtocol) || i.Healthy() {
		t.Fatal("descriptor crossed logical memory without poisoning", err)
	}
}

func TestResultViewValidationAndLifetime(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		want          error
		name          string
		size, pointer uint32
		status        rt.Status
	}{
		{nil, "success", 3, 16, rt.OK},
		{rt.ErrTooLarge, "output limit", 33, 16, rt.OK},
		{rt.ErrProtocol, "logical length", 3, 63, rt.OK},
		{rt.ErrProtocol, "reserved status", 0, 16, rt.Pending},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			m := newModule()
			i := newInstance(t, m)
			binary.LittleEndian.PutUint32(m.memory, test.pointer)
			binary.LittleEndian.PutUint32(m.memory[4:], test.size)
			copy(m.memory[16:], "abc")
			err := i.Call("view", func(call *rt.Call) error {
				view, resultErr := call.ResultView("view", test.status)
				if resultErr == nil {
					if string(view) != "abc" || cap(view) != len(view) {
						t.Fatal("unbounded or incorrect reply", view)
					}
					m.memory[16] = 'x'
					if view[0] != 'x' {
						t.Fatal("view copied instead of borrowing")
					}
				}
				return resultErr
			})
			if !errors.Is(err, test.want) || errors.Is(err, rt.ErrProtocol) && i.Healthy() {
				t.Fatal("view changed reply validation or poisoning", err)
			}
		})
	}
	m := newModule()
	i := newInstance(t, m)
	binary.LittleEndian.PutUint32(m.memory, 16)
	binary.LittleEndian.PutUint32(m.memory[4:], 3)
	copy(m.memory[16:], "bad")
	var application *rt.AppError
	if err := i.Call("error", func(call *rt.Call) error {
		_, resultErr := call.ResultView("error", rt.ErrArg)
		return resultErr
	}); !errors.As(err, &application) || !i.Healthy() {
		t.Fatal("expected error poisoned instance", err)
	}
	m.memory[16] = 'x'
	if application.Message != "bad" {
		t.Fatal("error message retained borrowed memory")
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
