package dd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"dario.cat/cocoon/cocoontest"
	"dario.cat/cocoon/rt"
)

const backendInputLimit uint64 = 67108864

func TestBackendPinnedWasm(t *testing.T) {
	if got := fmt.Sprintf("%x", sha256.Sum256(generatedWasm)); got != "ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333" {
		t.Fatalf("backend-control payload drifted: %s", got)
	}
}

// These controls are test-only. They share the real checked runtime boundaries,
// but their function dispatch is not the generated production facade.
type backendGuest struct {
	abi      rt.ABI
	instance *rt.Instance
	new      func() rt.Status
	add      func(uint64, float64) rt.Status
	batch    func(uint64, uint32, uint32) rt.Status
	count    func(uint64) rt.Status
	encode   func(uint64) rt.Status
	drop     func(uint64) rt.Status
	sql      func(uint32, uint32) rt.Status
	trace    func(uint32, uint32) rt.Status
	dispose  func() error
	output   uint32
}

func (g *backendGuest) close() error {
	g.instance.Close()
	return g.dispose()
}

type backendExport struct {
	function api.Function
	ctx      context.Context //nolint:containedctx // Serialized test adapter retains its noncancelable API call context, prepared outside warm timing.
	stack    [3]uint64
	params   int
	size     int
	useCall  bool
}

func (f *backendExport) invoke(a, b, c uint64) uint64 {
	f.stack[0], f.stack[1], f.stack[2] = a, b, c
	if f.useCall {
		result, err := f.function.Call(f.ctx, f.stack[:f.params]...)
		if err != nil {
			panic(err)
		}
		if len(result) == 0 {
			return 0
		}
		return result[0]
	}
	if err := f.function.CallWithStack(f.ctx, f.stack[:f.size]); err != nil {
		// rt.ABI cannot return an engine error. The actual error reaches the
		// existing rt.Instance recovery boundary, which poisons the instance.
		panic(err)
	}
	return f.stack[0]
}

type backendWasm struct {
	module  api.Module
	version backendExport
	schema  backendExport
	init    backendExport
	reserve backendExport
	output  backendExport
	trim    backendExport
}

func (m *backendWasm) Memory() []byte {
	memory := m.module.Memory()
	view, ok := memory.Read(0, memory.Size())
	if !ok {
		panic("current guest memory unavailable")
	}
	return view
}

func (m *backendWasm) Version() uint32 { return api.DecodeU32(m.version.invoke(0, 0, 0)) }
func (m *backendWasm) Schema() uint64  { return m.schema.invoke(0, 0, 0) }
func (m *backendWasm) Init()           { m.init.invoke(0, 0, 0) }
func (m *backendWasm) Reserve(size uint32) uint32 {
	return api.DecodeU32(m.reserve.invoke(uint64(size), 0, 0))
}
func (m *backendWasm) Output() uint32   { return api.DecodeU32(m.output.invoke(0, 0, 0)) }
func (m *backendWasm) Trim(keep uint32) { m.trim.invoke(uint64(keep), 0, 0) }

// Each factory owns its cache; immutable compiled instructions are shared,
// while each guest owns a runtime, capability host, memory and globals.
type backendFactory struct {
	cache   wazero.CompilationCache
	logger  *slog.Logger
	engine  string
	useCall bool
}

var backendRuntimeConstruction sync.Mutex

func newBackendFactory(tb testing.TB, engine string, useCall bool) *backendFactory {
	tb.Helper()
	backendCompilerSupported(tb, engine)
	factory := &backendFactory{engine: engine, useCall: useCall, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if engine == "CompiledWazero" {
		factory.cache = wazero.NewCompilationCache()
		tb.Cleanup(func() {
			if err := factory.cache.Close(context.WithoutCancel(tb.Context())); err != nil {
				tb.Error(err)
			}
		})
	}
	return factory
}

func backendCompilerSupported(tb testing.TB, engine string) {
	tb.Helper()
	if engine != "CompiledWazero" {
		return
	}
	supportedOS := false
	switch runtime.GOOS {
	case "linux", "darwin", "freebsd", "netbsd", "windows":
		supportedOS = runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64"
	case "dragonfly", "solaris", "illumos":
		supportedOS = runtime.GOARCH == "amd64"
	}
	if !supportedOS {
		tb.Skipf("pinned wazero compiled backend unavailable on %s/%s; interpreter substitution is excluded", runtime.GOOS, runtime.GOARCH)
	}
	// CPU-feature or executable-mmap failure remains a reported constructor
	// error, rather than silently falling back to the interpreter.
}

func backendRuntime(ctx context.Context, cache wazero.CompilationCache) wazero.Runtime {
	// The pinned engine initializes a process-wide version string lazily.
	// Construction is outside warm timing; execution remains independently locked.
	backendRuntimeConstruction.Lock()
	defer backendRuntimeConstruction.Unlock()
	config := wazero.NewRuntimeConfigCompiler().WithCloseOnContextDone(false)
	if cache != nil {
		config = config.WithCompilationCache(cache)
	}
	return wazero.NewRuntimeWithConfig(ctx, config)
}

func (f *backendFactory) guest(ctx context.Context) (_ *backendGuest, err error) {
	return f.guestLimits(ctx, rt.Limits{MaxInput: backendInputLimit, MaxOutput: backendInputLimit, MaxMemory: maximumMemory})
}

func (f *backendFactory) guestLimits(ctx context.Context, limits rt.Limits) (_ *backendGuest, err error) {
	var dispose func() error
	defer func() {
		if value := recover(); value != nil {
			err = rt.Classify("construct", value, "")
		}
		if err != nil && dispose != nil {
			err = errors.Join(err, dispose())
		}
	}()
	g := &backendGuest{}
	var host *rt.Host
	if f.engine == "TranslatedGo" {
		m, capability := newModule(Options{Logger: f.logger})
		host = capability
		g.abi = m
		g.new = func() rt.Status { return rt.Status(m.Xcocoon_sketch_new()) }
		g.add = func(h uint64, v float64) rt.Status { return rt.Status(m.Xcocoon_sketch_add(int64(h), v)) } //nolint:gosec // Preserve raw i64 handle bits.
		g.batch = func(h uint64, p, n uint32) rt.Status {
			return rt.Status(m.Xcocoon_sketch_add_many(int64(h), int32(p), int32(n))) //nolint:gosec // Preserve validated Wasm scalar bits.
		}
		g.count = func(h uint64) rt.Status { return rt.Status(m.Xcocoon_sketch_count(int64(h))) }                  //nolint:gosec // Raw i64 bits.
		g.encode = func(h uint64) rt.Status { return rt.Status(m.Xcocoon_sketch_encode(int64(h))) }                //nolint:gosec // Raw i64 bits.
		g.drop = func(h uint64) rt.Status { return rt.Status(m.Xcocoon_sketch_close(int64(h))) }                   //nolint:gosec // Raw i64 bits.
		g.sql = func(p, n uint32) rt.Status { return rt.Status(m.Xcocoon_obfuscate_sql(int32(p), int32(n))) }      //nolint:gosec // Wasm i32 bits.
		g.trace = func(p, n uint32) rt.Status { return rt.Status(m.Xcocoon_obfuscate_traces(int32(p), int32(n))) } //nolint:gosec // Wasm i32 bits.
		dispose = func() error { return nil }
	} else {
		engineRuntime := backendRuntime(ctx, f.cache)
		dispose = func() error { return engineRuntime.Close(context.WithoutCancel(ctx)) }
		host = rt.NewHost(f.logger)
		builder := engineRuntime.NewHostModuleBuilder("cocoon")
		builder.NewFunctionBuilder().WithFunc(host.Log).Export("log")
		builder.NewFunctionBuilder().WithFunc(host.RandomGet).Export("random_get")
		builder.NewFunctionBuilder().WithFunc(host.ClockNanos).Export("clock_nanos")
		if _, err = builder.Instantiate(ctx); err != nil {
			return nil, err
		}
		compiled, compileErr := engineRuntime.CompileModule(ctx, generatedWasm)
		if compileErr != nil {
			return nil, compileErr
		}
		m, instantiateErr := engineRuntime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithStartFunctions())
		if instantiateErr != nil {
			return nil, instantiateErr
		}
		export := func(name string) backendExport {
			function := m.ExportedFunction(name)
			if function == nil {
				panic("missing backend export: " + name)
			}
			definition := function.Definition()
			params := len(definition.ParamTypes())
			size := max(params, len(definition.ResultTypes()))
			if size > 3 {
				panic("backend export exceeds reusable carrier")
			}
			return backendExport{function: function, ctx: context.WithoutCancel(ctx), params: params, size: size, useCall: f.useCall}
		}
		abi := &backendWasm{module: m, version: export("cocoon_abi_version"), schema: export("cocoon_schema_hash"), init: export("cocoon_init"), reserve: export("cocoon_in_reserve"), output: export("cocoon_out"), trim: export("cocoon_trim")}
		g.abi = abi
		host.Bind(abi)
		newFn, addFn, batchFn := export("cocoon_sketch_new"), export("cocoon_sketch_add"), export("cocoon_sketch_add_many")
		countFn, encodeFn, dropFn := export("cocoon_sketch_count"), export("cocoon_sketch_encode"), export("cocoon_sketch_close")
		sqlFn, traceFn := export("cocoon_obfuscate_sql"), export("cocoon_obfuscate_traces")
		status := func(bits uint64) rt.Status { return rt.Status(int32(api.DecodeU32(bits))) } //nolint:gosec // Interpret raw Wasm i32 status bits.
		g.new = func() rt.Status { return status(newFn.invoke(0, 0, 0)) }
		g.add = func(h uint64, v float64) rt.Status { return status(addFn.invoke(h, math.Float64bits(v), 0)) }
		g.batch = func(h uint64, p, n uint32) rt.Status { return status(batchFn.invoke(h, uint64(p), uint64(n))) }
		g.count = func(h uint64) rt.Status { return status(countFn.invoke(h, 0, 0)) }
		g.encode = func(h uint64) rt.Status { return status(encodeFn.invoke(h, 0, 0)) }
		g.drop = func(h uint64) rt.Status { return status(dropFn.invoke(h, 0, 0)) }
		g.sql = func(p, n uint32) rt.Status { return status(sqlFn.invoke(uint64(p), uint64(n), 0)) }
		g.trace = func(p, n uint32) rt.Status { return status(traceFn.invoke(uint64(p), uint64(n), 0)) }
	}
	g.dispose = dispose
	g.instance, err = rt.NewInstance(g.abi, schemaHash, limits, host)
	if err != nil {
		return nil, err
	}
	// Instance's descriptor cache is private. Raw controls cache it with one
	// extra export here, outside warm timing and inside both cold factory scopes.
	g.output = g.abi.Output()
	return g, nil
}

type backendControl struct {
	factory *backendFactory
	home    *backendGuest
	pool    *rt.Pool[backendGuest]
	life    rt.Lifecycle
}

func newBackendControl(tb testing.TB, engine string) *backendControl {
	tb.Helper()
	factory := newBackendFactory(tb, engine, false)
	home, err := factory.guest(tb.Context())
	if err != nil {
		tb.Fatal(err)
	}
	c := &backendControl{factory: factory, home: home}
	c.pool, err = rt.NewPool(1, func() (*backendGuest, error) { return factory.guest(context.WithoutCancel(tb.Context())) }, func(g *backendGuest) bool { return g.instance.Recycle(65536, maximumMemory) }, func(g *backendGuest) {
		if closeErr := g.close(); closeErr != nil {
			tb.Error(closeErr)
		}
	})
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if closeErr := c.close(); closeErr != nil {
			tb.Error(closeErr)
		}
	})
	return c
}

func (c *backendControl) close() error {
	return c.life.Close(func() error { return errors.Join(c.pool.Close(), c.home.close()) })
}

type backendSketch struct {
	control *backendControl
	owner   *rt.Resource[backendSketch]
}

func (s *backendSketch) close() error {
	if s == nil || s.control == nil || s.owner == nil {
		return &rt.HandleError{Op: "close", Reason: "nil resource"}
	}
	return s.owner.Close()
}

func (c *backendControl) sketch() (*backendSketch, error) {
	if err := c.life.Enter(); err != nil {
		return nil, err
	}
	defer c.life.Leave()
	g := c.home
	var handle uint64
	err := g.instance.Call("new", func(call *rt.Call) error {
		view, resultErr := call.ResultView("new", g.new())
		if resultErr != nil {
			return resultErr
		}
		handle, resultErr = decodeU64(view, backendInputLimit)
		return resultErr
	})
	if err != nil {
		return nil, err
	}
	owner, err := rt.NewResource[backendSketch](g.instance, handle, func(h uint64) error {
		return g.instance.Call("close", func(call *rt.Call) error { return call.ResultUnit("close", g.drop(h)) })
	})
	if err != nil {
		return nil, err
	}
	return &backendSketch{control: c, owner: owner}, nil
}

func (s *backendSketch) use(op string, invoke func(*rt.Call, uint64) error) error {
	if err := s.control.life.Enter(); err != nil {
		return err
	}
	defer s.control.life.Leave()
	return s.owner.Use(op, func(instance *rt.Instance, handle uint64) error {
		return instance.Call(op, func(call *rt.Call) error { return invoke(call, handle) })
	})
}

func (s *backendSketch) add(value float64) error {
	return s.use("add", func(call *rt.Call, handle uint64) error {
		return call.ResultUnit("add", s.control.home.add(handle, value))
	})
}

func (s *backendSketch) batch(values []float64) error {
	// Generated methods admit before sizing; retain that order here.
	if err := s.control.life.Enter(); err != nil {
		return err
	}
	defer s.control.life.Leave()
	size, err := sizeSliceF64(values, backendInputLimit)
	if err != nil {
		return err
	}
	size, err = rt.InputSize(backendInputLimit, 1, 0, size)
	if err != nil {
		return err
	}
	return s.owner.Use("add_many", func(instance *rt.Instance, handle uint64) error {
		return instance.Call("add_many", func(call *rt.Call) error {
			if inputErr := call.PrepareInput(size); inputErr != nil {
				return inputErr
			}
			pointer, putErr := putSliceF64(call, values, size)
			if putErr != nil {
				return putErr
			}
			return call.ResultUnit("add_many", s.control.home.batch(handle, pointer, uint32(size))) //nolint:gosec // InputSize checked wasm32 bounds.
		})
	})
}

func (s *backendSketch) count() (float64, error) {
	var count float64
	err := s.use("count", func(call *rt.Call, handle uint64) error {
		view, resultErr := call.ResultView("count", s.control.home.count(handle))
		if resultErr != nil {
			return resultErr
		}
		count, resultErr = decodeF64(view, backendInputLimit)
		return resultErr
	})
	return count, err
}

func (s *backendSketch) encode() ([]byte, error) {
	var data []byte
	err := s.use("encode", func(call *rt.Call, handle uint64) error {
		view, resultErr := call.ResultView("encode", s.control.home.encode(handle))
		if resultErr != nil {
			return resultErr
		}
		data, resultErr = decodeBytes(view, backendInputLimit)
		return resultErr
	})
	return data, err
}

func (c *backendControl) sql(ctx context.Context, query string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := c.life.Enter(); err != nil {
		return "", err
	}
	defer c.life.Leave()
	size, err := sizeString(query, backendInputLimit)
	if err != nil {
		return "", err
	}
	size, err = rt.InputSize(backendInputLimit, 1, 0, size)
	if err != nil {
		return "", err
	}
	if !utf8.ValidString(query) {
		return "", &rt.AppError{Op: "encode", Status: rt.ErrArg, Message: "invalid UTF-8"}
	}
	var output string
	err = c.pool.Do(ctx, func(g *backendGuest) error {
		return g.instance.Call("obfuscate_sql", func(call *rt.Call) error {
			if inputErr := call.PrepareInput(size); inputErr != nil {
				return inputErr
			}
			pointer, putErr := call.PutString(query)
			if putErr != nil {
				return putErr
			}
			view, resultErr := call.ResultView("obfuscate_sql", g.sql(pointer, uint32(size))) //nolint:gosec // InputSize checked wasm32 bounds.
			if resultErr != nil {
				return resultErr
			}
			output, resultErr = decodeString(view, backendInputLimit)
			return resultErr
		})
	})
	return output, err
}

func (c *backendControl) trace(ctx context.Context, input []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := c.life.Enter(); err != nil {
		return nil, err
	}
	defer c.life.Leave()
	size, err := sizeBytes(input, backendInputLimit)
	if err != nil {
		return nil, err
	}
	size, err = rt.InputSize(backendInputLimit, 1, 0, size)
	if err != nil {
		return nil, err
	}
	data, err := encodeBytes(input, backendInputLimit)
	if err != nil {
		return nil, err
	}
	var output []byte
	err = c.pool.Do(ctx, func(g *backendGuest) error {
		return g.instance.Call("obfuscate_traces", func(call *rt.Call) error {
			if inputErr := call.PrepareInput(size); inputErr != nil {
				return inputErr
			}
			pointer, putErr := call.PutBytes(data)
			if putErr != nil {
				return putErr
			}
			view, resultErr := call.ResultView("obfuscate_traces", g.trace(pointer, uint32(size))) //nolint:gosec // InputSize checked wasm32 bounds.
			if resultErr != nil {
				return resultErr
			}
			output, resultErr = decodeBytes(view, backendInputLimit)
			return resultErr
		})
	})
	return output, err
}

// Raw replies use ordinary Go bounds checks and generated typed decoding. They
// omit admission/ownership, rt.Call state and limits, poisoning and recycling.
func (g *backendGuest) rawReply(op string, status rt.Status) ([]byte, error) {
	memory := g.abi.Memory()
	descriptor := memory[g.output : uint64(g.output)+8]
	pointer := binary.LittleEndian.Uint32(descriptor)
	size := binary.LittleEndian.Uint32(descriptor[4:])
	view := memory[pointer : uint64(pointer)+uint64(size)]
	return view, rt.FromStatus(op, status, view)
}

func (g *backendGuest) rawSketch() (uint64, error) {
	view, err := g.rawReply("new", g.new())
	if err != nil {
		return 0, err
	}
	return decodeU64(view, backendInputLimit)
}

func (g *backendGuest) rawBatch(handle uint64, values []float64) rt.Status {
	size := uint32(len(values) * 8) //nolint:gosec // Authored fixed corpora are bounded below 64 MiB.
	pointer := g.abi.Reserve(size)
	memory := g.abi.Memory()[pointer : uint64(pointer)+uint64(size)]
	for index, value := range values {
		binary.LittleEndian.PutUint64(memory[index*8:], math.Float64bits(value))
	}
	return g.batch(handle, pointer, size)
}

func (g *backendGuest) rawSQL(query string) (string, error) {
	size := uint32(len(query)) //nolint:gosec // Authored fixed corpus is bounded below 64 MiB.
	pointer := g.abi.Reserve(size)
	copy(g.abi.Memory()[pointer:uint64(pointer)+uint64(size)], query)
	view, err := g.rawReply("obfuscate_sql", g.sql(pointer, size))
	if err != nil {
		return "", err
	}
	return decodeString(view, backendInputLimit)
}

func (g *backendGuest) rawTrace(input []byte) ([]byte, error) {
	size := uint32(len(input)) //nolint:gosec // Authored fixed payload is bounded below 64 MiB.
	pointer := g.abi.Reserve(size)
	copy(g.abi.Memory()[pointer:uint64(pointer)+uint64(size)], input)
	view, err := g.rawReply("obfuscate_traces", g.trace(pointer, size))
	if err != nil {
		return nil, err
	}
	return decodeBytes(view, backendInputLimit)
}

func backendRequire(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Fatal(err)
	}
}

func backendOracle(tb testing.TB) *cocoontest.Guest {
	tb.Helper()
	guest := cocoontest.New(tb, generatedWasm)
	for _, identity := range []struct {
		name string
		want uint64
	}{{"cocoon_abi_version", rt.ABIVersion}, {"cocoon_schema_hash", schemaHash}} {
		got, err := guest.Call(tb.Context(), identity.name)
		backendRequire(tb, err)
		if len(got) != 1 || got[0] != identity.want {
			tb.Fatalf("original identity %s = %v, want %d", identity.name, got, identity.want)
		}
	}
	_, err := guest.Call(tb.Context(), "cocoon_init")
	backendRequire(tb, err)
	return guest
}

func backendOracleSketch(tb testing.TB, values []float64) (count float64, data []byte) {
	tb.Helper()
	guest := backendOracle(tb)
	status, data, err := guest.Operation(tb.Context(), "cocoon_sketch_new", nil, backendInputLimit, backendInputLimit)
	backendRequire(tb, err)
	backendRequire(tb, rt.FromStatus("new", status, data))
	handle, err := decodeU64(data, backendInputLimit)
	backendRequire(tb, err)
	for _, value := range values {
		status, data, err = guest.Operation(tb.Context(), "cocoon_sketch_add", []cocoontest.Arg{{Value: handle}, {Value: math.Float64bits(value)}}, backendInputLimit, backendInputLimit)
		backendRequire(tb, err)
		backendRequire(tb, rt.FromStatus("add", status, data))
		if len(data) != 0 {
			tb.Fatal("original scalar reply is not unit")
		}
	}
	status, data, err = guest.Operation(tb.Context(), "cocoon_sketch_count", []cocoontest.Arg{{Value: handle}}, backendInputLimit, backendInputLimit)
	backendRequire(tb, err)
	backendRequire(tb, rt.FromStatus("count", status, data))
	count, err = decodeF64(data, backendInputLimit)
	backendRequire(tb, err)
	status, data, err = guest.Operation(tb.Context(), "cocoon_sketch_encode", []cocoontest.Arg{{Value: handle}}, backendInputLimit, backendInputLimit)
	backendRequire(tb, err)
	backendRequire(tb, rt.FromStatus("encode", status, data))
	return count, data
}

func TestBackendControlOrderedState(t *testing.T) {
	for _, corpus := range []string{"Repeated", "NegativeBins", "Holdout", "ScalarBroad", "Increasing"} {
		t.Run(corpus, func(t *testing.T) {
			values := batchingValues(corpus, 8192)
			if corpus == "Repeated" {
				values = batchingValues(corpus, 1024)
			}
			if corpus == "ScalarBroad" {
				values = binReuseValues("BroadHoldout")
			}
			if corpus == "Increasing" {
				values = binReuseValues("UniqueCycle")
			}
			revisit := values[:32]
			if corpus == "Increasing" {
				revisit = make([]float64, 32)
				for index := range revisit {
					revisit[index] = float64(len(values) + index + 1)
				}
			}
			ordered := append(append([]float64(nil), values...), revisit...)
			wantCount, wantData := backendOracleSketch(t, ordered)
			for _, engine := range []string{"TranslatedGo", "CompiledWazero"} {
				t.Run(engine, func(t *testing.T) {
					control := newBackendControl(t, engine)
					sketch, err := control.sketch()
					backendRequire(t, err)
					backendRequire(t, sketch.batch(values))
					for _, value := range revisit {
						backendRequire(t, sketch.add(value))
					}
					count, err := sketch.count()
					backendRequire(t, err)
					data, err := sketch.encode()
					backendRequire(t, err)
					if count != wantCount || !bytes.Equal(data, wantData) {
						t.Fatalf("ordered state mismatch: count %g/%g, protobuf %d/%d bytes", count, wantCount, len(data), len(wantData))
					}
					before := bytes.Clone(data)
					for _, bad := range []float64{-1, math.NaN(), math.Inf(1)} {
						checkPointArgument(t, "add", sketch.add(bad))
					}
					for _, bad := range []float64{-1, math.NaN(), math.Inf(1)} {
						checkPointArgument(t, "add_many", sketch.batch([]float64{42, 0, bad}))
						got, encodeErr := sketch.encode()
						backendRequire(t, encodeErr)
						if !bytes.Equal(got, before) {
							t.Fatal("invalid tail partially mutated state")
						}
					}
					backendRequire(t, sketch.batch(nil))
					backendRequire(t, sketch.add(43))
					if !bytes.Equal(data, before) {
						t.Fatal("owned output changed after later guest writes")
					}
					alias := &backendSketch{control: control, owner: sketch.owner}
					backendRequire(t, alias.close())
					var handleError *rt.HandleError
					if !errors.As(sketch.add(1), &handleError) {
						t.Fatal("closed alias remained usable")
					}
					fresh, freshErr := control.sketch()
					backendRequire(t, freshErr)
					backendRequire(t, fresh.add(1))
					if !errors.As(sketch.add(1), &handleError) {
						t.Fatal("new resource revived old closed alias")
					}
					backendRequire(t, fresh.close())
				})
			}
		})
	}
}

type backendPanicHandler struct{ calls *atomic.Int32 }

func (h backendPanicHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h backendPanicHandler) Handle(context.Context, slog.Record) error {
	h.calls.Add(1)
	panic("deliberate backend host logger panic")
}
func (h backendPanicHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h backendPanicHandler) WithGroup(string) slog.Handler      { return h }

func TestBackendControlHostFault(t *testing.T) {
	for _, engine := range []string{"TranslatedGo", "CompiledWazero"} {
		t.Run(engine, func(t *testing.T) {
			factory := newBackendFactory(t, engine, false)
			var calls atomic.Int32
			factory.logger = slog.New(backendPanicHandler{calls: &calls})
			guest, err := factory.guest(t.Context())
			backendRequire(t, err)
			t.Cleanup(func() { backendRequire(t, guest.close()) })
			handle, err := guest.rawSketch()
			backendRequire(t, err)
			owner, err := rt.NewResource[backendSketch](guest.instance, handle, func(h uint64) error {
				return guest.instance.Call("close", func(call *rt.Call) error { return call.ResultUnit("close", guest.drop(h)) })
			})
			backendRequire(t, err)
			err = guest.instance.Call("host-fault", func(*rt.Call) error {
				guest.abi.Reserve(uint32(backendInputLimit + 1))
				return nil
			})
			var hostError *rt.HostError
			var faultError *rt.FaultError
			if (!errors.As(err, &hostError) && !errors.As(err, &faultError)) || calls.Load() != 1 || guest.instance.Healthy() {
				t.Fatalf("guest log import did not contain host panic: %v, imports=%d", err, calls.Load())
			}
			var diagnostic string
			if hostError != nil {
				diagnostic = fmt.Sprint(hostError.Value)
			} else {
				// FaultError.Error may prefer the guest's recorded panic message;
				// Detail retains the actual engine error and host panic cause.
				diagnostic = faultError.Detail
			}
			if !strings.Contains(diagnostic, "deliberate backend host logger panic") {
				t.Fatalf("host panic cause was lost: %s", diagnostic)
			}
			var handleError *rt.HandleError
			err = owner.Use("stale-host", func(*rt.Instance, uint64) error {
				t.Fatal("host fault left resource usable")
				return nil
			})
			if !errors.As(err, &handleError) {
				t.Fatalf("host fault epoch: %v", err)
			}
		})
	}
}

func TestBackendControlLimitsAndReset(t *testing.T) {
	for _, engine := range []string{"TranslatedGo", "CompiledWazero"} {
		t.Run(engine, func(t *testing.T) {
			factory := newBackendFactory(t, engine, false)
			guest, err := factory.guestLimits(t.Context(), rt.Limits{MaxInput: 8, MaxOutput: 8, MaxMemory: maximumMemory})
			backendRequire(t, err)
			t.Cleanup(func() { backendRequire(t, guest.close()) })
			err = guest.instance.Call("input-limit", func(call *rt.Call) error { return call.PrepareInput(9) })
			if !errors.Is(err, rt.ErrTooLarge) || !guest.instance.Healthy() {
				t.Fatalf("input limit should remain healthy: %v", err)
			}
			handle, err := guest.rawSketch()
			backendRequire(t, err)
			err = guest.instance.Call("output-limit", func(call *rt.Call) error {
				_, resultErr := call.ResultView("output-limit", guest.encode(handle))
				return resultErr
			})
			if !errors.Is(err, rt.ErrTooLarge) || !guest.instance.Healthy() {
				t.Fatalf("output limit should remain healthy: %v", err)
			}
			err = guest.instance.Call("first-reservation", func(call *rt.Call) error {
				if prepareErr := call.PrepareInput(8); prepareErr != nil {
					return prepareErr
				}
				_, putErr := call.PutBytes([]byte{1})
				return putErr
			})
			backendRequire(t, err)
			err = guest.instance.Call("reset-reservation", func(call *rt.Call) error {
				_, putErr := call.PutBytes([]byte{1})
				return putErr
			})
			if !errors.Is(err, rt.ErrTooLarge) || !guest.instance.Healthy() {
				t.Fatalf("reservation state was not reset: %v", err)
			}

			probe, err := factory.guest(t.Context())
			backendRequire(t, err)
			initial := uint64(len(probe.abi.Memory()))
			backendRequire(t, probe.close())
			limited, err := factory.guestLimits(t.Context(), rt.Limits{MaxInput: initial, MaxOutput: initial, MaxMemory: initial + 65536})
			backendRequire(t, err)
			t.Cleanup(func() { backendRequire(t, limited.close()) })
			limitedHandle, err := limited.rawSketch()
			backendRequire(t, err)
			limitedOwner, err := rt.NewResource[backendSketch](limited.instance, limitedHandle, func(h uint64) error {
				return limited.instance.Call("close", func(call *rt.Call) error { return call.ResultUnit("close", limited.drop(h)) })
			})
			backendRequire(t, err)
			err = limited.instance.Call("memory-limit", func(call *rt.Call) error { return call.PrepareInput(1024 * 1024) })
			if !errors.Is(err, rt.ErrTooLarge) || limited.instance.Healthy() {
				t.Fatalf("post-operation memory growth did not poison: %v", err)
			}
			var handleError *rt.HandleError
			err = limitedOwner.Use("stale-memory", func(*rt.Instance, uint64) error {
				t.Fatal("memory failure left resource usable")
				return nil
			})
			if !errors.As(err, &handleError) {
				t.Fatalf("memory fault epoch: %v", err)
			}
			err = limited.instance.Call("after-memory", func(*rt.Call) error {
				t.Fatal("memory failure admitted subsequent callback")
				return nil
			})
			if !errors.Is(err, rt.ErrPoisoned) {
				t.Fatalf("subsequent memory-limited call: %v", err)
			}
		})
	}
}

func TestBackendControlCacheIsolation(t *testing.T) {
	for _, engine := range []string{"TranslatedGo", "CompiledWazero"} {
		t.Run(engine, func(t *testing.T) {
			factory := newBackendFactory(t, engine, false)
			first, err := factory.guest(t.Context())
			backendRequire(t, err)
			t.Cleanup(func() { backendRequire(t, first.close()) })
			second, err := factory.guest(t.Context())
			backendRequire(t, err)
			t.Cleanup(func() { backendRequire(t, second.close()) })
			firstHandle, err := first.rawSketch()
			backendRequire(t, err)
			secondHandle, err := second.rawSketch()
			backendRequire(t, err)
			backendRequire(t, backendStatus(first.add(firstHandle, 42)))
			count, err := second.rawCount(secondHandle)
			backendRequire(t, err)
			if count != 0 {
				t.Fatal("shared compilation cache shared guest sketch state")
			}
			firstPointer := first.abi.Reserve(16)
			secondPointer := second.abi.Reserve(16)
			firstView, err := rt.Range(first.abi.Memory(), firstPointer, 16)
			backendRequire(t, err)
			secondView, err := rt.Range(second.abi.Memory(), secondPointer, 16)
			backendRequire(t, err)
			for index := range firstView {
				firstView[index] = 0xa5
				secondView[index] = 0x5a
			}
			if !bytes.Equal(firstView, bytes.Repeat([]byte{0xa5}, 16)) {
				t.Fatal("shared compilation cache aliased guest memory")
			}
			backendRequire(t, backendStatus(first.drop(firstHandle)))
			backendRequire(t, backendStatus(second.drop(secondHandle)))
		})
	}
}

func TestBackendControlStateless(t *testing.T) {
	oracle := backendOracle(t)
	queries := append([]string{"SELECT * FROM users WHERE id = 42 AND name = 'bob'", ""}, translationSQLQueries()...)
	for _, engine := range []string{"TranslatedGo", "CompiledWazero"} {
		t.Run(engine, func(t *testing.T) {
			control := newBackendControl(t, engine)
			for _, query := range queries {
				status, data, err := oracle.Operation(t.Context(), "cocoon_obfuscate_sql", []cocoontest.Arg{{Buffer: true, Bytes: []byte(query)}}, backendInputLimit, backendInputLimit)
				backendRequire(t, err)
				got, gotErr := control.sql(t.Context(), query)
				if status == rt.OK {
					backendRequire(t, gotErr)
					if got != string(data) {
						t.Fatal("SQL complete output mismatch")
					}
				} else {
					var app *rt.AppError
					if !errors.As(gotErr, &app) || app.Status != status || app.Message != string(data) {
						t.Fatalf("SQL error status/payload mismatch: %d %q / %v", status, data, gotErr)
					}
				}
			}
			for _, input := range [][]byte{tracePayload(1000), {0x90}, nil, {0xc1}, {0x90, 0}} {
				status, data, err := oracle.Operation(t.Context(), "cocoon_obfuscate_traces", []cocoontest.Arg{{Buffer: true, Bytes: input}}, backendInputLimit, backendInputLimit)
				backendRequire(t, err)
				got, gotErr := control.trace(t.Context(), input)
				if status == rt.OK {
					backendRequire(t, gotErr)
					if !bytes.Equal(got, data) {
						t.Fatal("trace output mismatch")
					}
				} else {
					var app *rt.AppError
					if !errors.As(gotErr, &app) || app.Status != status || app.Message != string(data) {
						t.Fatalf("error status/payload mismatch: %d %q / %v", status, data, gotErr)
					}
				}
			}
			retained, err := control.trace(t.Context(), tracePayload(1000))
			backendRequire(t, err)
			copyBefore := bytes.Clone(retained)
			_, err = control.sql(t.Context(), queries[0])
			backendRequire(t, err)
			if !bytes.Equal(retained, copyBefore) {
				t.Fatal("pool recycle or subsequent call changed owned trace output")
			}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			if _, err = control.sql(ctx, queries[0]); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled admission: %v", err)
			}
			_, err = control.sql(t.Context(), "\xff")
			var app *rt.AppError
			if !errors.As(err, &app) || app.Status != rt.ErrArg {
				t.Fatalf("invalid UTF-8: %v", err)
			}
		})
	}
}

func TestBackendControlFaultAndGrowth(t *testing.T) {
	for _, engine := range []string{"TranslatedGo", "CompiledWazero"} {
		t.Run(engine, func(t *testing.T) {
			control := newBackendControl(t, engine)
			sketch, err := control.sketch()
			backendRequire(t, err)
			before := len(control.home.abi.Memory())
			values := batchingValues("Repeated", 262144)
			backendRequire(t, sketch.batch(values))
			if after := len(control.home.abi.Memory()); after <= before {
				t.Fatalf("growth probe did not grow: %d -> %d", before, after)
			}
			count, err := sketch.count()
			backendRequire(t, err)
			if count != float64(len(values)) {
				t.Fatalf("growth lost count: %g", count)
			}
			err = sketch.use("invalid-pointer", func(call *rt.Call, handle uint64) error {
				// Guest-side input validation rejects this without a trap.
				return call.ResultUnit("invalid-pointer", control.home.batch(handle, 0xfffffffc, 16))
			})
			var app *rt.AppError
			if !errors.As(err, &app) || app.Status != rt.ErrArg || !control.home.instance.Healthy() {
				t.Fatalf("invalid guest input was not a healthy argument error: %v", err)
			}
			err = sketch.use("guest-panic", func(_ *rt.Call, _ uint64) error {
				// Bypass host input checks deliberately. The guest panics before
				// allocating, invokes its log import and executes unreachable.
				control.home.abi.Reserve(uint32(backendInputLimit + 1))
				return nil
			})
			var fault *rt.FaultError
			if !errors.As(err, &fault) || control.home.instance.Healthy() {
				t.Fatalf("real trap did not poison: %v", err)
			}
			var handleError *rt.HandleError
			if !errors.As(sketch.add(1), &handleError) {
				t.Fatal("faulted epoch still admits old resource")
			}
			if err = control.home.instance.Call("after-fault", func(*rt.Call) error {
				t.Fatal("poisoned callback executed")
				return nil
			}); !errors.Is(err, rt.ErrPoisoned) {
				t.Fatalf("subsequent call: %v", err)
			}
		})
	}
}

func TestBackendControlDrain(t *testing.T) {
	for _, engine := range []string{"TranslatedGo", "CompiledWazero"} {
		t.Run(engine, func(t *testing.T) {
			control := newBackendControl(t, engine)
			sketch, err := control.sketch()
			backendRequire(t, err)
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
			go func() {
				done <- sketch.use("drain", func(call *rt.Call, handle uint64) error {
					close(entered)
					<-release
					return call.ResultUnit("drain", control.home.add(handle, 42))
				})
			}()
			<-entered
			closed := make(chan error, 1)
			go func() { closed <- control.close() }()
			deadline := time.Now().Add(5 * time.Second)
			for {
				admissionErr := control.life.Enter()
				if errors.Is(admissionErr, rt.ErrClosed) {
					break
				}
				backendRequire(t, admissionErr)
				control.life.Leave()
				if time.Now().After(deadline) {
					close(release)
					t.Fatal("close never reached terminal admission")
				}
				runtime.Gosched()
			}
			select {
			case err = <-closed:
				t.Fatalf("close did not drain active call: %v", err)
			default:
			}
			close(release)
			backendRequire(t, <-done)
			backendRequire(t, <-closed)
			if !errors.Is(sketch.add(1), rt.ErrClosed) {
				t.Fatal("closed lifecycle admitted resource call")
			}
		})
	}
}

func TestBackendControlProtocolPoison(t *testing.T) {
	for _, engine := range []string{"TranslatedGo", "CompiledWazero"} {
		t.Run(engine, func(t *testing.T) {
			control := newBackendControl(t, engine)
			sketch, newErr := control.sketch()
			backendRequire(t, newErr)
			err := control.home.instance.Call("corrupt-reply", func(call *rt.Call) error {
				status := control.home.new()
				memory := control.home.abi.Memory()
				binary.LittleEndian.PutUint32(memory[control.home.output:], 0xfffffffc)
				binary.LittleEndian.PutUint32(memory[control.home.output+4:], 8)
				_, resultErr := call.ResultView("corrupt-reply", status)
				return resultErr
			})
			if !errors.Is(err, rt.ErrProtocol) || control.home.instance.Healthy() {
				t.Fatalf("malformed reply did not poison: %v", err)
			}
			var handleError *rt.HandleError
			if !errors.As(sketch.add(1), &handleError) {
				t.Fatal("protocol fault did not invalidate prior resource epoch")
			}
			err = control.home.instance.Call("after-protocol", func(*rt.Call) error {
				t.Fatal("callback ran after protocol fault")
				return nil
			})
			if !errors.Is(err, rt.ErrPoisoned) {
				t.Fatalf("after protocol fault: %v", err)
			}
		})
	}
}

func (g *backendGuest) rawCount(handle uint64) (float64, error) {
	view, err := g.rawReply("count", g.count(handle))
	if err != nil {
		return 0, err
	}
	return decodeF64(view, backendInputLimit)
}

func backendStatus(status rt.Status) error {
	if status != rt.OK {
		return fmt.Errorf("unexpected raw guest status %d", status)
	}
	return nil
}
