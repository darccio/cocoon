// Package cocoontest provides a reference WebAssembly engine for tests only.
// Generated production packages depend on rt, never this package or wazero.
package cocoontest

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"cocoon.dev/cocoon/rt"
)

// Arg carries either scalar Wasm bits or a variable input buffer region.
type Arg struct {
	Bytes  []byte
	Value  uint64
	Buffer bool
}

// Guest serializes reference execution and owns its test-only wazero runtime.
type Guest struct {
	runtime wazero.Runtime
	module  api.Module
	host    *rt.Host
	mu      sync.Mutex
}

// New instantiates the exact bytes translated into Go and registers cleanup.
func New(tb testing.TB, wasm []byte) *Guest {
	tb.Helper()
	guest, err := Instantiate(tb.Context(), wasm)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if closeErr := guest.Close(context.WithoutCancel(tb.Context())); closeErr != nil {
			tb.Error(closeErr)
		}
	})
	return guest
}

// Instantiate creates a reference guest with shielded log, random, and clock imports.
func Instantiate(ctx context.Context, wasm []byte) (*Guest, error) {
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
	host := rt.NewHost(slog.New(slog.NewTextHandler(io.Discard, nil)))
	guest := &Guest{runtime: runtime, host: host}
	builder := runtime.NewHostModuleBuilder("cocoon")
	builder.NewFunctionBuilder().WithFunc(host.Log).Export("log")
	builder.NewFunctionBuilder().WithFunc(host.RandomGet).Export("random_get")
	builder.NewFunctionBuilder().WithFunc(host.ClockNanos).Export("clock_nanos")
	if _, err := builder.Instantiate(ctx); err != nil {
		return nil, errors.Join(err, runtime.Close(context.WithoutCancel(ctx)))
	}
	module, err := runtime.InstantiateWithConfig(ctx, wasm, wazero.NewModuleConfig().WithStartFunctions())
	if err != nil {
		return nil, errors.Join(err, runtime.Close(context.WithoutCancel(ctx)))
	}
	guest.module = module
	host.Bind(guest)
	return guest, nil
}

// Memory exposes current reference memory to its host capabilities.
// As with rt.Module, only call this while the guest execution lock is held.
func (g *Guest) Memory() []byte {
	if g.module == nil || len(g.module.ExportedMemoryDefinitions()) == 0 {
		return nil
	}
	memory := g.module.Memory()
	view, ok := memory.Read(0, memory.Size())
	if !ok {
		return nil
	}
	return view
}

// Call invokes a typed Wasm export using raw Wasm scalar bits.
func (g *Guest) Call(ctx context.Context, name string, args ...uint64) ([]uint64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.call(ctx, name, args...)
}

func (g *Guest) call(ctx context.Context, name string, args ...uint64) ([]uint64, error) {
	if g.module == nil {
		return nil, rt.ErrClosed
	}
	function := g.module.ExportedFunction(name)
	if function == nil {
		return nil, fmt.Errorf("%w: missing reference export %s", rt.ErrProtocol, name)
	}
	return function.Call(ctx, args...)
}

// Operation reserves aggregate input, invokes a status export, and copies its
// bounded reply. App/argument/handle statuses remain available for comparison.
func (g *Guest) Operation(ctx context.Context, name string, args []Arg, inputLimit, outputLimit uint64) (status rt.Status, output []byte, err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var size uint64
	for _, arg := range args {
		if arg.Buffer {
			size, err = rt.InputSize(inputLimit, 1, size, uint64(len(arg.Bytes)))
			if err != nil {
				return 0, nil, err
			}
		}
	}
	reserved, err := g.call(ctx, "cocoon_in_reserve", size)
	if err != nil {
		return 0, nil, err
	}
	if len(reserved) != 1 || reserved[0] > 0xffffffff {
		return 0, nil, rt.ErrProtocol
	}
	pointer := uint32(reserved[0])                            // #nosec G115 -- Explicitly checked wasm32 pointer above.
	input, err := rt.Range(g.Memory(), pointer, uint32(size)) // #nosec G115 -- InputSize enforces wasm32 size.
	if err != nil {
		return 0, nil, err
	}
	var parameters []uint64
	var offset uint32
	for _, arg := range args {
		if arg.Buffer {
			copy(input[offset:], arg.Bytes)
			parameters = append(parameters, uint64(pointer)+uint64(offset), uint64(len(arg.Bytes)))
			offset += uint32(len(arg.Bytes)) // #nosec G115 -- Every region is part of the checked aggregate input.
		} else {
			parameters = append(parameters, arg.Value)
		}
	}
	values, err := g.call(ctx, name, parameters...)
	if err != nil {
		return 0, nil, err
	}
	if len(values) != 1 || values[0] > 0xffffffff {
		return 0, nil, rt.ErrProtocol
	}
	status = rt.Status(int32(values[0])) // #nosec G115 -- Interpret the checked raw i32 bits as an ABI status.
	descriptor, err := g.call(ctx, "cocoon_out")
	if err != nil {
		return 0, nil, err
	}
	if len(descriptor) != 1 || descriptor[0] > 0xffffffff {
		return 0, nil, rt.ErrProtocol
	}
	view, err := rt.Range(g.Memory(), uint32(descriptor[0]), 8) // #nosec G115 -- Checked wasm32 pointer above.
	if err != nil {
		return 0, nil, err
	}
	length := binary.LittleEndian.Uint32(view[4:])
	if uint64(length) > outputLimit {
		return 0, nil, rt.ErrTooLarge
	}
	result, err := rt.Range(g.Memory(), binary.LittleEndian.Uint32(view), length)
	if err != nil {
		return 0, nil, err
	}
	return status, append([]byte(nil), result...), nil
}

// Close is idempotent and releases the reference runtime and its linear memory.
func (g *Guest) Close(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.module == nil {
		return nil
	}
	err := g.runtime.Close(ctx)
	g.module = nil
	g.host.Bind(nil)
	return err
}
