package rt

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
)

// This synthetic ABI measures runtime overhead, not translated Rust speed.
// Each stream holds back at most 1 KiB. Memory and stream slots are preallocated.
type sessionBenchModule struct {
	failure string
	memory  []byte
	streams []sessionBenchStream
	writes  uint64
	input   uint32
	failAt  uint64
}

type sessionBenchStream struct {
	pending [1024]byte
	used    int
}

func (m *sessionBenchModule) Memory() []byte { return m.memory }
func (*sessionBenchModule) Version() uint32  { return ABIVersion }
func (*sessionBenchModule) Schema() uint64   { return 7 }
func (*sessionBenchModule) Init()            {}
func (*sessionBenchModule) Output() uint32   { return 0 }
func (*sessionBenchModule) Trim(uint32)      {}
func (m *sessionBenchModule) Reserve(n uint32) uint32 {
	m.input = n
	return 16
}

func (m *sessionBenchModule) resolve(handle uint64) *sessionBenchStream {
	index := uint32(handle & 0xffffffff) // #nosec G115 -- Mask extracts the wasm32 slot.
	if handle>>32 != 1 || index == 0 || uint64(index) > uint64(len(m.streams)) {
		panic("invalid synthetic handle")
	}
	return &m.streams[index-1]
}

func (m *sessionBenchModule) write(handle uint64, n int) Status {
	s := m.resolve(handle)
	if n < 0 || uint64(n) > uint64(m.input) {
		panic("invalid synthetic reservation")
	}
	m.writes++
	if m.writes == m.failAt {
		switch m.failure {
		case "Trap":
			panic("unreachable")
		case "Memory":
			m.memory = make([]byte, 65537)
		case "Reply":
			binary.LittleEndian.PutUint32(m.memory, 65537)
			return OK
		case "Status":
			return Pending
		}
	}
	data := m.memory[16 : 16+n]
	emit := max(0, s.used+n-len(s.pending))
	old := min(emit, s.used)
	out := m.memory[32768 : 32768+emit]
	copy(out, s.pending[:old])
	copy(out[old:], data[:emit-old])
	copy(s.pending[:], s.pending[old:s.used])
	remaining := s.used - old
	copy(s.pending[remaining:], data[emit-old:])
	s.used = remaining + n - (emit - old)
	binary.LittleEndian.PutUint32(m.memory, 32768)
	binary.LittleEndian.PutUint32(m.memory[4:], uint32(emit)) // #nosec G115 -- emit is bounded by the input size.
	return OK
}

func sessionBenchInstance(tb testing.TB, streams int) (*Instance, *sessionBenchModule) {
	tb.Helper()
	m := &sessionBenchModule{memory: make([]byte, 65536), streams: make([]sessionBenchStream, streams)}
	i, err := NewInstance(m, 7, Limits{MaxInput: 16384, MaxOutput: 16384, MaxMemory: 65536}, NewHost(nil))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(i.Close)
	return i, m
}

func sessionBenchResource(tb testing.TB, i *Instance, handle uint64) *Resource[sessionBenchStream] {
	tb.Helper()
	r, err := NewResource[sessionBenchStream](i, handle, func(uint64) error { return nil })
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if closeErr := r.Close(); closeErr != nil {
			tb.Error(closeErr)
		}
	})
	return r
}

func sessionBenchInvoke(m *sessionBenchModule, handle uint64, data []byte, call *Call) error {
	if err := call.PrepareInput(uint64(len(data))); err != nil {
		return err
	}
	if _, err := call.PutBytes(data); err != nil {
		return err
	}
	status := m.write(handle, len(data))
	// Copy each reply, including inside a sequence: no borrowed view survives.
	_, err := call.Result("write", status)
	return err
}

// sessionBenchChecked is a test-only copy of Instance.Call's checked boundary,
// with its mutex held by the caller. Keep every per-operation check in sync.
// This is a ceiling experiment, not a proposed or supported runtime API.
func sessionBenchChecked(i *Instance, op string, invoke func(*Call) error) (err error) {
	switch i.state {
	case instanceClosed:
		return ErrClosed
	case instancePoisoned:
		return ErrPoisoned
	case instanceOpen:
	}
	defer func() {
		if value := recover(); value != nil {
			i.poison()
			var message string
			if i.host != nil {
				message = i.host.takePanic()
			}
			err = Classify(op, value, message)
		}
	}()
	if i.host != nil {
		i.host.takePanic()
	}
	i.call = Call{instance: i}
	err = invoke(&i.call)
	if uint64(len(i.module.Memory())) > i.limits.MaxMemory {
		i.poison()
		return fmt.Errorf("%s: %w: linear memory exceeded maximum", op, ErrTooLarge)
	}
	if err != nil && errors.Is(err, ErrProtocol) {
		i.poison()
	}
	return err
}

func sessionBenchSequence(life *Lifecycle, r *Resource[sessionBenchStream], m *sessionBenchModule, data []byte, count int, locked bool) error {
	if err := life.Enter(); err != nil {
		return err
	}
	defer life.Leave()
	return r.Use("sequence", func(i *Instance, handle uint64) error {
		if locked {
			i.mu.Lock()
			defer i.mu.Unlock()
		}
		invoke := func(c *Call) error { return sessionBenchInvoke(m, handle, data, c) }
		for range count {
			var err error
			if locked {
				err = sessionBenchChecked(i, "write", invoke)
			} else {
				err = i.Call("write", invoke)
			}
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// BenchmarkSessionLayers measures separate mechanisms, not additive costs.
func BenchmarkSessionLayers(b *testing.B) {
	for _, layer := range []string{"Admission", "Ownership", "GuestHandle", "Mutex", "CheckedEmpty", "ReplyUnit", "Guest"} {
		b.Run(layer, func(b *testing.B) {
			i, m := sessionBenchInstance(b, 1)
			r := sessionBenchResource(b, i, 1<<32|1)
			var life Lifecycle
			call := &Call{instance: i}
			var invoke func() error
			switch layer {
			case "Admission":
				invoke = func() error {
					if err := life.Enter(); err != nil {
						return err
					}
					life.Leave()
					return nil
				}
			case "Ownership":
				invoke = func() error { return r.Use("resolve", func(*Instance, uint64) error { return nil }) }
			case "GuestHandle":
				invoke = func() error { runtime.KeepAlive(m.resolve(1<<32 | 1)); return nil }
			case "Mutex":
				invoke = func() error { i.mu.Lock(); i.mu.Unlock(); return nil } //nolint:gocritic,staticcheck // Isolate the bare lock/unlock cost without defer overhead or guest work.
			case "CheckedEmpty":
				invoke = func() error { return i.Call("empty", func(*Call) error { return nil }) }
			case "ReplyUnit":
				invoke = func() error { return call.ResultUnit("unit", OK) }
			case "Guest":
				invoke = func() error { m.write(1<<32|1, 0); return nil }
			}
			b.ReportAllocs()
			for b.Loop() {
				if err := invoke(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSessionThroughput uses one resource per worker. Home paths share one
// instance; Private paths give each worker its own instance as a diagnostic.
// Pool paths include both admissions and the checked recycler used by facades.
// An op is a sequence; ns/call and calls/s normalize sequences to commands.
func BenchmarkSessionThroughput(b *testing.B) {
	for _, size := range []int{0, 64, 4096, 16384} {
		b.Run(fmt.Sprintf("Bytes%d", size), func(b *testing.B) {
			paths := []string{"Pool1", "PoolP", "Home", "Admit32", "Locked32", "Private", "Private32"}
			if size == 64 {
				paths = append(paths, "Locked4", "Locked16", "Locked64")
			}
			for _, path := range paths {
				b.Run(path, func(b *testing.B) { benchmarkSessionThroughput(b, size, path) })
			}
		})
	}
}

func benchmarkSessionThroughput(b *testing.B, size int, path string) {
	b.Helper()
	workers := runtime.GOMAXPROCS(0)
	data := make([]byte, size)
	for n := range data {
		data[n] = byte(n % 251) // #nosec G115 -- modulo bounds the byte.
	}
	var m *sessionBenchModule
	resources := make([]*Resource[sessionBenchStream], workers)
	var modules []*sessionBenchModule
	private := path == "Private" || path == "Private32"
	if private {
		for n := range workers {
			instance, module := sessionBenchInstance(b, 1)
			resources[n] = sessionBenchResource(b, instance, 1<<32|1)
			modules = append(modules, module)
		}
	} else if path != "Pool1" && path != "PoolP" {
		var i *Instance
		i, m = sessionBenchInstance(b, workers)
		modules = append(modules, m)
		for n := range workers {
			resources[n] = sessionBenchResource(b, i, 1<<32|uint64(n+1)) // #nosec G115 -- n is a positive worker index.
		}
	}
	count := 1
	locked := path == "Locked32" || path == "Private32" || path == "Locked4" || path == "Locked16" || path == "Locked64"
	switch path {
	case "Locked32", "Private32", "Admit32":
		count = 32
	case "Locked4":
		count = 4
	case "Locked16":
		count = 16
	case "Locked64":
		count = 64
	}
	var life Lifecycle
	var pool *Pool[Instance]
	if path == "Pool1" || path == "PoolP" {
		capacity := 1
		if path == "PoolP" {
			capacity = workers
		}
		modules = nil
		var err error
		pool, err = NewPool(capacity, func() (*Instance, error) {
			instance, module := sessionBenchInstance(b, 1)
			modules = append(modules, module)
			return instance, nil
		}, func(instance *Instance) bool { return instance.Recycle(4096, 65536) }, (*Instance).Close)
		if err != nil {
			b.Fatal(err)
		}
		b.Cleanup(func() {
			if closeErr := pool.Close(); closeErr != nil {
				b.Error(closeErr)
			}
		})
		// Populate every lazy slot sequentially before timing or parallel use.
		for range capacity {
			if err := pool.Do(b.Context(), func(*Instance) error { return nil }); err != nil {
				b.Fatal(err)
			}
		}
	}
	var next atomic.Uint64
	ctx := b.Context()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		worker := int(next.Add(1) - 1) // #nosec G115 -- RunParallel starts exactly GOMAXPROCS workers.
		module := m
		if private {
			module = modules[worker]
		}
		for pb.Next() {
			var err error
			if pool != nil {
				err = sessionBenchPoolCall(ctx, &life, pool, data)
			} else {
				err = sessionBenchSequence(&life, resources[worker], module, data, count, locked)
			}
			if err != nil {
				b.Error(err)
				return
			}
		}
	})
	b.StopTimer()
	var writes uint64
	for _, module := range modules {
		writes += module.writes
		for n := range module.streams {
			stream := &module.streams[n]
			if stream.used > len(stream.pending) {
				b.Fatal("unbounded pending window")
			}
		}
	}
	want := uint64(b.N) * uint64(count) // #nosec G115 -- benchmark iteration and command counts are nonnegative.
	if writes != want {
		b.Fatalf("writes = %d, want %d", writes, want)
	}
	seconds := b.Elapsed().Seconds()
	b.ReportMetric(float64(writes)/seconds, "calls/s")
	b.ReportMetric(seconds*1e9/float64(writes), "ns/call")
	if size != 0 {
		b.ReportMetric(float64(writes)*float64(size)/seconds/(1<<20), "MiB/s")
	}
}

func sessionBenchPoolCall(ctx context.Context, life *Lifecycle, pool *Pool[Instance], data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := life.Enter(); err != nil {
		return err
	}
	defer life.Leave()
	return pool.Do(ctx, func(i *Instance) error {
		m, ok := i.module.(*sessionBenchModule)
		if !ok {
			return ErrProtocol
		}
		return i.Call("write", func(c *Call) error { return sessionBenchInvoke(m, 1<<32|1, data, c) })
	})
}

func TestSessionBenchHoldback(t *testing.T) {
	i, m := sessionBenchInstance(t, 1)
	var input, output, saved []byte
	for _, size := range []int{0, 1, 1023, 0, 64, 4096, 1, 16384, 1024, 7} {
		chunk := make([]byte, size)
		for n := range chunk {
			chunk[n] = byte((len(input) + n) % 251) // #nosec G115 -- modulo bounds the byte.
		}
		input = append(input, chunk...)
		var reply []byte
		if err := i.Call("write", func(c *Call) error {
			if err := c.PrepareInput(uint64(len(chunk))); err != nil {
				return err
			}
			if _, err := c.PutBytes(chunk); err != nil {
				return err
			}
			var err error
			reply, err = c.Result("write", m.write(1<<32|1, size))
			return err
		}); err != nil {
			t.Fatal(err)
		}
		if saved != nil && !bytes.Equal(saved, input[:len(saved)]) {
			t.Fatal("owned result changed after another write")
		}
		if saved == nil && len(reply) != 0 {
			saved = reply
		}
		output = append(output, reply...)
		if !bytes.Equal(output, input[:max(0, len(input)-1024)]) {
			t.Fatal("emitted bytes differ from the holdback oracle")
		}
	}
	stream := m.resolve(1<<32 | 1)
	output = append(output, stream.pending[:stream.used]...)
	if !bytes.Equal(output, input) {
		t.Fatal("pending tail lost or reordered bytes")
	}
}

func TestSessionBenchSequenceFault(t *testing.T) {
	for _, failure := range []string{"Trap", "Memory", "Reply", "Status"} {
		for _, locked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/Locked%t", failure, locked), func(t *testing.T) {
				i, m := sessionBenchInstance(t, 1)
				r := sessionBenchResource(t, i, 1<<32|1)
				m.failAt, m.failure = 2, failure
				var life Lifecycle
				err := sessionBenchSequence(&life, r, m, []byte{1}, 32, locked)
				switch failure {
				case "Trap":
					var fault *FaultError
					if !errors.As(err, &fault) || fault.Op != "write" || fault.Kind != FaultUnreachable {
						t.Fatalf("fault classification: %v", err)
					}
				case "Memory":
					if !errors.Is(err, ErrTooLarge) {
						t.Fatalf("memory error: %v", err)
					}
				case "Reply", "Status":
					if !errors.Is(err, ErrProtocol) {
						t.Fatalf("reply error: %v", err)
					}
				}
				if m.writes != 2 || i.Healthy() || i.Epoch() != 2 {
					t.Fatalf("writes=%d healthy=%v epoch=%d", m.writes, i.Healthy(), i.Epoch())
				}
				if err := life.Close(nil); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

// TestSessionBenchBoundary checks the prototype before trusting its timings.
func TestSessionBenchBoundary(t *testing.T) {
	for _, failure := range []string{"Protocol", "Trap", "Memory"} {
		t.Run(failure, func(t *testing.T) {
			i, m := sessionBenchInstance(t, 1)
			i.mu.Lock()
			defer i.mu.Unlock()
			calls := 0
			for range 3 {
				err := sessionBenchChecked(i, "fault", func(c *Call) error {
					calls++
					if calls == 1 {
						return c.PrepareInput(4)
					}
					if _, err := c.PutBytes([]byte{1}); !errors.Is(err, ErrTooLarge) {
						t.Fatal("call reservation was not reset")
					}
					switch failure {
					case "Protocol":
						return fmt.Errorf("wrapped: %w", ErrProtocol)
					case "Trap":
						panic("unreachable")
					case "Memory":
						m.memory = make([]byte, 65537)
					}
					return nil
				})
				if err != nil {
					break
				}
			}
			if calls != 2 || i.Healthy() || i.Epoch() != 2 {
				t.Fatalf("calls=%d, healthy=%v, epoch=%d", calls, i.Healthy(), i.Epoch())
			}
			if err := sessionBenchChecked(i, "later", func(*Call) error { t.Fatal("executed after fault"); return nil }); !errors.Is(err, ErrPoisoned) {
				t.Fatalf("after fault: %v", err)
			}
		})
	}
}
