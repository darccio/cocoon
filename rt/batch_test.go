package rt

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func batchFixture(t *testing.T) (*Resource[sessionBenchStream], *Instance, *sessionBenchModule, *Lifecycle) {
	t.Helper()
	i, m := sessionBenchInstance(t, 1)
	return sessionBenchResource(t, i, 0xabcdef98_76543210), i, m, &Lifecycle{}
}

func batchReply(m *sessionBenchModule, data []byte, status Status) BatchCommand {
	return BatchCommand{Op: "write", Invoke: func(uint64) Status {
		m.writes++
		copy(m.memory[32768:], data)
		binary.LittleEndian.PutUint32(m.memory, 32768)
		binary.LittleEndian.PutUint32(m.memory[4:], uint32(len(data))) // #nosec G115 -- Test payloads fit the fixture's memory.
		return status
	}, Decode: func([]byte) error { return nil }}
}

func assertBatchFailure(t *testing.T, completed uint32, err error, prefix uint32, index int, attempted bool) *BatchError {
	t.Helper()
	var failure *BatchError
	if completed != prefix || !errors.As(err, &failure) || failure.Index != index || failure.Attempted != attempted {
		t.Fatalf("batch completed=%d, error=%v; want prefix=%d index=%d attempted=%t", completed, err, prefix, index, attempted)
	}
	if failure.Error() == "" || failure.Unwrap() == nil {
		t.Fatal("missing batch diagnostic/cause")
	}
	return failure
}

func TestBatchPreflightRejectsWholePlan(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		edit     func([]BatchCommand)
		name     string
		limits   BatchLimits
		count    int
		index    int
		argument bool
	}{
		{name: "missing invoke", count: 2, index: 1, argument: true, edit: func(c []BatchCommand) { c[1].Invoke = nil }},
		{name: "missing decode", count: 2, index: 1, argument: true, edit: func(c []BatchCommand) { c[1].Decode = nil }},
		{name: "missing prepare", count: 2, index: 1, argument: true, edit: func(c []BatchCommand) { c[1].InputBytes = 1 }},
		{name: "aggregate input", count: 2, index: 1, limits: BatchLimits{MaxInputBytes: 1}, edit: func(c []BatchCommand) {
			for k := range c {
				c[k].InputBytes = 1
				c[k].Prepare = func(*Call) error { return nil }
			}
		}},
		{name: "input overflow", count: 2, index: 1, edit: func(c []BatchCommand) {
			c[1].InputBytes = math.MaxUint64
			c[1].Prepare = func(*Call) error { return nil }
		}},
		{name: "individual input", count: 2, index: 1, edit: func(c []BatchCommand) { c[1].InputBytes = 16385; c[1].Prepare = func(*Call) error { return nil } }},
		{name: "default count", count: 33, index: -1},
		{name: "explicit count", count: 2, index: -1, limits: BatchLimits{MaxCalls: 1}},
		{name: "hard calls", count: 0, index: -1, limits: BatchLimits{MaxCalls: 257}},
		{name: "hard input", count: 0, index: -1, limits: BatchLimits{MaxInputBytes: 16<<20 + 1}},
		{name: "hard output", count: 0, index: -1, limits: BatchLimits{MaxOutputBytes: 16<<20 + 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r, i, m, life := batchFixture(t)
			commands := make([]BatchCommand, test.count)
			for k := range commands {
				commands[k] = batchReply(m, nil, OK)
			}
			if test.edit != nil {
				test.edit(commands)
			}
			n, err := r.Batch(t.Context(), life, test.limits, commands)
			failure := assertBatchFailure(t, n, err, 0, test.index, false)
			var app *AppError
			if test.argument {
				if !errors.As(failure, &app) || app.Status != ErrArg || app.Op != "batch" {
					t.Fatal(failure)
				}
			} else if !errors.Is(failure, ErrTooLarge) {
				t.Fatal(failure)
			}
			if m.writes != 0 || life.state.Load() != 0 || !i.Healthy() {
				t.Fatal("preflight had effects or leaked admission")
			}
		})
	}
}

func TestBatchEmptyAndAdmission(t *testing.T) {
	t.Parallel()
	r, i, m, life := batchFixture(t)
	for _, ctx := range []context.Context{nil, t.Context()} {
		supplied := life
		if ctx != nil {
			supplied = nil
		}
		n, err := r.Batch(ctx, supplied, BatchLimits{}, nil)
		failure := assertBatchFailure(t, n, err, 0, -1, false)
		var app *AppError
		if !errors.As(failure, &app) || app.Status != ErrArg {
			t.Fatal(failure)
		}
	}
	n, err := r.Batch(t.Context(), life, BatchLimits{MaxCalls: 256, MaxInputBytes: 16 << 20, MaxOutputBytes: 16 << 20}, nil)
	if n != 0 || err != nil || m.writes != 0 || life.state.Load() != 0 {
		t.Fatal(n, err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	n, err = r.Batch(ctx, life, BatchLimits{}, nil)
	if !errors.Is(assertBatchFailure(t, n, err, 0, -1, false), context.Canceled) {
		t.Fatal(err)
	}
	var missing *Resource[sessionBenchStream]
	n, err = missing.Batch(t.Context(), life, BatchLimits{}, nil)
	var handle *HandleError
	if !errors.As(assertBatchFailure(t, n, err, 0, -1, false), &handle) {
		t.Fatal(err)
	}
	if closeErr := life.Close(nil); closeErr != nil {
		t.Fatal(closeErr)
	}
	n, err = r.Batch(t.Context(), life, BatchLimits{}, nil)
	if !errors.Is(assertBatchFailure(t, n, err, 0, -1, false), ErrClosed) {
		t.Fatal(err)
	}
	if closeErr := r.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	n, err = r.Batch(t.Context(), &Lifecycle{}, BatchLimits{}, nil)
	if !errors.As(assertBatchFailure(t, n, err, 0, -1, false), &handle) {
		t.Fatal(err)
	}
	if !i.Healthy() {
		t.Fatal("argument/ownership errors poisoned home")
	}
}

func TestBatchReservationMatchesMetadata(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		data     string
		declared uint64
		capacity uint64
		skip     bool
	}{
		{"missing reservation", "", 1, 0, true},
		{"underfilled", "x", 2, 2, false},
		{"overdeclared", "xy", 1, 2, false},
		{"unused zero reservation", "", 0, 1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r, i, m, life := batchFixture(t)
			command := batchReply(m, nil, OK)
			command.InputBytes = test.declared
			command.Prepare = func(call *Call) error {
				if test.skip {
					return nil
				}
				if err := call.PrepareInput(test.capacity); err != nil {
					return err
				}
				_, err := call.PutString(test.data)
				return err
			}
			n, err := r.Batch(t.Context(), life, BatchLimits{}, []BatchCommand{batchReply(m, nil, OK), command, batchReply(m, nil, OK)})
			failure := assertBatchFailure(t, n, err, 1, 1, false)
			var app *AppError
			if !errors.As(failure, &app) || app.Status != ErrArg || m.writes != 1 || !i.Healthy() || life.state.Load() != 0 {
				t.Fatal(failure)
			}
			n, err = r.Batch(t.Context(), life, BatchLimits{}, []BatchCommand{batchReply(m, nil, OK)})
			if n != 1 || err != nil {
				t.Fatal("healthy reservation error prevented reuse", n, err)
			}
		})
	}
}

func TestBatchMiddleCheckedFaults(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		want      error
		edit      func(*BatchCommand, *Instance, *sessionBenchModule)
		name      string
		attempted bool
		fault     bool
		poison    bool
	}{
		{name: "descriptor", want: ErrProtocol, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) {
			c.Invoke = func(uint64) Status { m.writes++; binary.LittleEndian.PutUint32(m.memory, 65537); return OK }
		}},
		{name: "empty pointer", want: ErrProtocol, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) {
			c.Invoke = func(uint64) Status {
				m.writes++
				binary.LittleEndian.PutUint32(m.memory, 65537)
				binary.LittleEndian.PutUint32(m.memory[4:], 0)
				return OK
			}
		}},
		{name: "invalid argument-error pointer", want: ErrProtocol, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) {
			c.Invoke = func(uint64) Status {
				m.writes++
				binary.LittleEndian.PutUint32(m.memory, 65537)
				binary.LittleEndian.PutUint32(m.memory[4:], 0)
				return ErrArg
			}
		}},
		{name: "invalid limit-error pointer", want: ErrProtocol, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) {
			c.Invoke = func(uint64) Status {
				m.writes++
				binary.LittleEndian.PutUint32(m.memory, 65537)
				binary.LittleEndian.PutUint32(m.memory[4:], 0)
				return ErrLimit
			}
		}},
		{name: "reserved status", want: ErrProtocol, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) { *c = batchReply(m, nil, Pending) }},
		{name: "unknown status", want: ErrProtocol, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) { *c = batchReply(m, nil, Status(-1)) }},
		{name: "typed decode", want: ErrProtocol, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, _ *sessionBenchModule) {
			c.Decode = func([]byte) error { return fmt.Errorf("typed: %w", ErrProtocol) }
		}},
		{name: "joined decode", want: ErrProtocol, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, _ *sessionBenchModule) {
			c.Decode = func([]byte) error { return errors.Join(errors.New("decode"), ErrProtocol) }
		}},
		{name: "unit decode", want: ErrProtocol, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) {
			*c = batchReply(m, []byte{1}, OK)
			c.Decode = func(reply []byte) error {
				if len(reply) != 0 {
					return ErrProtocol
				}
				return nil
			}
		}},
		{name: "prepare trap", fault: true, poison: true, edit: func(c *BatchCommand, _ *Instance, _ *sessionBenchModule) {
			c.Prepare = func(*Call) error { panic("unreachable") }
		}},
		{name: "export trap", attempted: true, fault: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) {
			c.Invoke = func(uint64) Status { m.writes++; panic("unreachable") }
		}},
		{name: "decode trap", attempted: true, fault: true, poison: true, edit: func(c *BatchCommand, _ *Instance, _ *sessionBenchModule) {
			c.Decode = func([]byte) error { panic("unreachable") }
		}},
		{name: "memory after decode", want: ErrTooLarge, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) {
			c.Decode = func([]byte) error { m.memory = make([]byte, 65537); return nil }
		}},
		{name: "memory after error", want: ErrTooLarge, attempted: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) {
			c.Decode = func([]byte) error { m.memory = make([]byte, 65537); return &AppError{Status: ErrApp} }
		}},
		{name: "memory after prepare", want: ErrTooLarge, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) {
			c.Prepare = func(*Call) error { m.memory = make([]byte, 65537); return &AppError{Status: ErrArg} }
		}},
		{name: "host panic", attempted: true, fault: true, poison: true, edit: func(c *BatchCommand, _ *Instance, m *sessionBenchModule) {
			c.Invoke = func(uint64) Status { m.writes++; panic(hostPanic{value: "host value"}) }
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r, i, m, life := batchFixture(t)
			second := batchReply(m, nil, OK)
			test.edit(&second, i, m)
			n, err := r.Batch(t.Context(), life, BatchLimits{}, []BatchCommand{batchReply(m, nil, OK), second, {Op: "third", Invoke: func(uint64) Status { t.Error("ran after failure"); return OK }, Decode: func([]byte) error { return nil }}})
			failure := assertBatchFailure(t, n, err, 1, 1, test.attempted)
			if test.fault {
				var guest *FaultError
				var host *HostError
				if !errors.As(failure, &guest) && !errors.As(failure, &host) {
					t.Fatal(failure)
				}
				if test.name == "host panic" && (host == nil || host.Op != "write" || host.Value != "host value") {
					t.Fatal("host panic lost its exact classification", failure)
				}
				if guest != nil && (guest.Op != "write" || guest.Kind != FaultUnreachable) {
					t.Fatal(guest)
				}
			} else if !errors.Is(failure, test.want) {
				t.Fatal(failure)
			}
			if i.Healthy() == test.poison || i.Epoch() != 2 || life.state.Load() != 0 {
				t.Fatal("fault did not poison/release immediately", i.Epoch(), failure)
			}
			n, err = r.Batch(t.Context(), life, BatchLimits{}, nil)
			var handle *HandleError
			if !errors.As(assertBatchFailure(t, n, err, 0, -1, false), &handle) {
				t.Fatal("old resource survived", err)
			}
		})
	}
}

func TestBatchReplyBudgetAndValidationPrecedence(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		want      error
		name      string
		status    Status
		malformed bool
		perCall   bool
		poison    bool
		badRange  bool
	}{
		{name: "success exceeds remaining", status: OK, want: ErrTooLarge},
		{name: "application exceeds remaining", status: ErrApp, want: ErrTooLarge},
		{name: "argument exceeds remaining", status: ErrArg, want: ErrTooLarge},
		{name: "handle exceeds remaining", status: ErrHandle, want: ErrTooLarge},
		{name: "guest limit exceeds remaining", status: ErrLimit, want: ErrTooLarge},
		{name: "malformed typed exceeds remaining", status: OK, malformed: true, want: ErrProtocol, poison: true},
		{name: "reserved exceeds remaining", status: Pending, want: ErrProtocol, poison: true},
		{name: "unknown exceeds remaining", status: Status(99), want: ErrProtocol, poison: true},
		{name: "negative status exceeds remaining", status: Status(-1), want: ErrProtocol, poison: true},
		{name: "invalid expected error pointer precedes status", status: ErrArg, badRange: true, want: ErrProtocol, poison: true},
		{name: "per operation max precedes status and range", status: Pending, perCall: true, want: ErrTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			r, i, m, life := batchFixture(t)
			var decoded int
			second := batchReply(m, []byte("larger"), test.status)
			second.Decode = func([]byte) error {
				decoded++
				if test.malformed {
					return ErrProtocol
				}
				return nil
			}
			if test.perCall {
				second.Invoke = func(uint64) Status {
					m.writes++
					binary.LittleEndian.PutUint32(m.memory, math.MaxUint32)
					binary.LittleEndian.PutUint32(m.memory[4:], 16385)
					return test.status
				}
			} else if test.badRange {
				second.Invoke = func(uint64) Status {
					m.writes++
					binary.LittleEndian.PutUint32(m.memory, math.MaxUint32)
					binary.LittleEndian.PutUint32(m.memory[4:], 6)
					return test.status
				}
			}
			n, err := r.Batch(t.Context(), life, BatchLimits{MaxOutputBytes: 1}, []BatchCommand{batchReply(m, []byte{1}, OK), second, batchReply(m, nil, OK)})
			failure := assertBatchFailure(t, n, err, 1, 1, true)
			if !errors.Is(failure, test.want) || i.Healthy() == test.poison || m.writes != 2 {
				t.Fatal(failure)
			}
			if (decoded == 1) != (test.status == OK) {
				t.Fatal("typed validation ordering", decoded)
			}
			var app *AppError
			var handle *HandleError
			if errors.As(failure, &app) || errors.As(failure, &handle) {
				t.Fatal("retained over-budget guest message", failure)
			}
		})
	}
	r, i, m, life := batchFixture(t)
	for _, status := range []Status{ErrApp, ErrArg, ErrHandle, ErrLimit} {
		n, err := r.Batch(t.Context(), life, BatchLimits{MaxOutputBytes: 3}, []BatchCommand{batchReply(m, []byte("bad"), status)})
		failure := assertBatchFailure(t, n, err, 0, 0, true)
		var app *AppError
		var handle *HandleError
		if status == ErrApp || status == ErrArg {
			if !errors.As(failure, &app) || app.Message != "bad" || app.Status != status {
				t.Fatal(failure)
			}
		}
		if status == ErrHandle {
			if !errors.As(failure, &handle) || handle.Reason != "bad" {
				t.Fatal(failure)
			}
		}
		if status == ErrLimit && !errors.Is(failure, ErrTooLarge) {
			t.Fatal(failure)
		}
		copy(m.memory[32768:], "xyz")
		if app != nil && app.Message != "bad" || handle != nil && handle.Reason != "bad" || !i.Healthy() {
			t.Fatal("expected reply not owned/healthy")
		}
	}
}

func TestBatchAccountsCurrentErrorAfterStandalone(t *testing.T) {
	t.Parallel()
	r, i, m, life := batchFixture(t)
	standalone := batchReply(m, []byte("standalone error"), ErrApp)
	err := i.Call("standalone", func(c *Call) error {
		_, replyErr := c.ResultView("standalone", standalone.Invoke(0))
		return replyErr
	})
	var app *AppError
	if !errors.As(err, &app) || app.Message != "standalone error" {
		t.Fatal("missing standalone error", err)
	}
	command := batchReply(m, nil, ErrApp)
	n, err := r.Batch(t.Context(), life, BatchLimits{MaxOutputBytes: 1}, []BatchCommand{command})
	failure := assertBatchFailure(t, n, err, 0, 0, true)
	if !errors.As(failure, &app) || app.Status != ErrApp || app.Message != "" || !i.Healthy() {
		t.Fatal("stale bytes masked empty expected error", failure)
	}
}

func TestBatchResetGrowthAndOwnedReplies(t *testing.T) {
	t.Parallel()
	r, i, m, life := batchFixture(t)
	var outputs [][]byte
	first := batchReply(m, []byte("first"), OK)
	first.InputBytes = 3
	first.Prepare = func(c *Call) error {
		if err := c.PrepareInput(3); err != nil {
			return err
		}
		_, err := c.PutString("abc")
		return err
	}
	first.Decode = func(view []byte) error {
		outputs = append(outputs, bytes.Clone(view))
		i.host.lastPanic = "stale"
		return nil
	}
	second := batchReply(m, []byte("second"), OK)
	second.Prepare = func(c *Call) error {
		if c.pointer != 0 || c.used != 0 || c.capacity != 0 || i.host.lastPanic != "" {
			t.Error("per-command state leaked")
		}
		if err := c.PrepareInput(0); err != nil {
			return err
		}
		return nil
	}
	invoke := second.Invoke
	second.Invoke = func(handle uint64) Status {
		if handle != 0xabcdef98_76543210 {
			t.Error("lost handle width")
		}
		m.memory = bytes.Clone(m.memory)
		return invoke(handle)
	}
	second.Decode = func(view []byte) error { outputs = append(outputs, bytes.Clone(view)); return nil }
	n, err := r.Batch(t.Context(), life, BatchLimits{}, []BatchCommand{first, second})
	if n != 2 || err != nil || len(outputs) != 2 || string(outputs[0]) != "first" || string(outputs[1]) != "second" {
		t.Fatal(n, err, outputs)
	}
	third := batchReply(m, nil, OK)
	third.Invoke = func(uint64) Status { panic("unreachable") }
	n, err = r.Batch(t.Context(), life, BatchLimits{}, []BatchCommand{third})
	failure := assertBatchFailure(t, n, err, 0, 0, true)
	var fault *FaultError
	if !errors.As(failure, &fault) || fault.GuestMsg != "" {
		t.Fatal("stale panic message crossed buffers", err)
	}
}

func TestBatchCancellationBetweenCommands(t *testing.T) {
	t.Parallel()
	for _, last := range []bool{false, true} {
		t.Run(fmt.Sprint(last), func(t *testing.T) {
			t.Parallel()
			r, _, m, life := batchFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			first := batchReply(m, []byte{1}, OK)
			invoke := first.Invoke
			first.Invoke = func(h uint64) Status { cancel(); return invoke(h) }
			commands := []BatchCommand{first}
			if !last {
				commands = append(commands, batchReply(m, nil, OK))
			}
			n, err := r.Batch(ctx, life, BatchLimits{}, commands)
			if last {
				if n != 1 || err != nil {
					t.Fatal("final cancellation replaced success", n, err)
				}
			} else if !errors.Is(assertBatchFailure(t, n, err, 1, 1, false), context.Canceled) {
				t.Fatal(err)
			}
			if m.writes != 1 || life.state.Load() != 0 {
				t.Fatal("cancelled command ran/leaked admission")
			}
		})
	}
	for _, status := range []Status{ErrArg, Pending} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			t.Parallel()
			r, i, m, life := batchFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			command := batchReply(m, nil, status)
			invoke := command.Invoke
			command.Invoke = func(h uint64) Status { cancel(); return invoke(h) }
			n, err := r.Batch(ctx, life, BatchLimits{}, []BatchCommand{command})
			failure := assertBatchFailure(t, n, err, 0, 0, true)
			if errors.Is(failure, context.Canceled) || status == Pending && !errors.Is(failure, ErrProtocol) || i.Healthy() != (status == ErrArg) {
				t.Fatal("cancel replaced produced error", err)
			}
		})
	}
}

func batchAwait[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-t.Context().Done():
		t.Fatal("test canceled")
	case <-time.After(5 * time.Second):
		t.Fatal("batch synchronization timed out")
	}
	var zero T
	return zero
}

func batchWaitState(t *testing.T, life *Lifecycle, want uint64) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for life.state.Load() != want {
		if time.Now().After(deadline) {
			t.Fatalf("admission state %d, want %d", life.state.Load(), want)
		}
		runtime.Gosched()
	}
}

func TestBatchCancelledMutexWaiterDrains(t *testing.T) {
	t.Parallel()
	for _, mode := range []struct{ ownership, empty bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		ownershipLock := mode.ownership
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			t.Parallel()
			r, i, m, life := batchFixture(t)
			if ownershipLock {
				r.cell.mu.Lock()
			} else {
				i.mu.Lock()
			}
			release := sync.OnceFunc(func() {
				if ownershipLock {
					r.cell.mu.Unlock()
				} else {
					i.mu.Unlock()
				}
			})
			defer release()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			commands := []BatchCommand{batchReply(m, nil, OK)}
			expectedIndex := 0
			if mode.empty {
				commands = nil
				expectedIndex = -1
			}
			done := make(chan error, 1)
			go func() {
				n, err := r.Batch(ctx, life, BatchLimits{}, commands)
				if n != 0 {
					t.Error(n)
				}
				done <- err
			}()
			batchWaitState(t, life, 1)
			cancel()
			select {
			case err := <-done:
				t.Fatal("mutex wait became interruptible", err)
			default:
			}
			closed := make(chan error, 1)
			go func() { closed <- life.Close(nil) }()
			batchWaitState(t, life, lifecycleClosing|1)
			release()
			failure := assertBatchFailure(t, 0, batchAwait(t, done), 0, expectedIndex, false)
			if !errors.Is(failure, context.Canceled) || m.writes != 0 {
				t.Fatal(failure)
			}
			if err := batchAwait(t, closed); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBatchDrainAndAliasClose(t *testing.T) {
	t.Parallel()
	i, m := sessionBenchInstance(t, 1)
	life := &Lifecycle{}
	var active atomic.Bool
	var destroyed atomic.Int32
	r, resourceErr := NewResource[sessionBenchStream](i, 7, func(uint64) error {
		if active.Load() {
			t.Error("destructor overlapped buffer execution")
		}
		destroyed.Add(1)
		return nil
	})
	if resourceErr != nil {
		t.Fatal(resourceErr)
	}
	t.Cleanup(func() {
		if closeErr := r.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	entered, release := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	defer unblock()
	first := batchReply(m, nil, OK)
	invoke := first.Invoke
	first.Invoke = func(h uint64) Status {
		active.Store(true)
		close(entered)
		batchAwait(t, release)
		return invoke(h)
	}
	last := batchReply(m, nil, OK)
	last.Decode = func([]byte) error { active.Store(false); return nil }
	done := make(chan error, 1)
	go func() {
		n, err := r.Batch(t.Context(), life, BatchLimits{}, []BatchCommand{first, batchReply(m, nil, OK), last})
		if n != 3 {
			t.Error("drain truncated admitted buffer", n)
		}
		done <- err
	}()
	batchAwait(t, entered)
	var cleanups atomic.Int32
	closed := make(chan error, 1)
	go func() { closed <- life.Close(func() error { cleanups.Add(1); return nil }) }()
	batchWaitState(t, life, lifecycleClosing|1)
	n, err := r.Batch(t.Context(), life, BatchLimits{}, nil)
	if !errors.Is(assertBatchFailure(t, n, err, 0, -1, false), ErrClosed) {
		t.Fatal(err)
	}
	aliasClosed := make(chan error, 1)
	go func() { aliasClosed <- r.Close() }()
	select {
	case closeErr := <-aliasClosed:
		t.Fatal("alias Close overlapped active buffer", closeErr)
	default:
	}
	unblock()
	if waitErr := batchAwait(t, done); waitErr != nil {
		t.Fatal(waitErr)
	}
	if waitErr := batchAwait(t, closed); waitErr != nil {
		t.Fatal(waitErr)
	}
	if waitErr := batchAwait(t, aliasClosed); waitErr != nil {
		t.Fatal(waitErr)
	}
	if m.writes != 3 || cleanups.Load() != 1 || destroyed.Load() != 1 {
		t.Fatal("lost admitted commands or repeated cleanup")
	}
	n, err = r.Batch(t.Context(), &Lifecycle{}, BatchLimits{}, nil)
	var handle *HandleError
	if !errors.As(assertBatchFailure(t, n, err, 0, -1, false), &handle) {
		t.Fatal("closed alias survived", err)
	}
}

func TestBatchEncodingFailurePreservesHealthyPrefix(t *testing.T) {
	t.Parallel()
	r, i, m, life := batchFixture(t)
	command := batchReply(m, nil, OK)
	command.Prepare = func(*Call) error { return &AppError{Op: "encode", Status: ErrArg, Message: "invalid UTF-8"} }
	n, err := r.Batch(t.Context(), life, BatchLimits{}, []BatchCommand{batchReply(m, nil, OK), command, batchReply(m, nil, OK)})
	failure := assertBatchFailure(t, n, err, 1, 1, false)
	var app *AppError
	if !errors.As(failure, &app) || app.Op != "encode" || app.Status != ErrArg || !i.Healthy() || m.writes != 1 {
		t.Fatal(failure)
	}
	n, err = r.Batch(t.Context(), life, BatchLimits{MaxCalls: 256, MaxOutputBytes: 1}, []BatchCommand{batchReply(m, []byte{1}, OK), batchReply(m, nil, OK), batchReply(m, nil, OK)})
	if n != 3 || err != nil {
		t.Fatal("exact budget prevented empty replies/reuse", n, err)
	}
	commands := make([]BatchCommand, 256)
	for k := range commands {
		commands[k] = batchReply(m, nil, OK)
	}
	n, err = r.Batch(t.Context(), life, BatchLimits{MaxCalls: 256}, commands)
	if n != 256 || err != nil {
		t.Fatal("hard call bound unavailable", n, err)
	}
}

type batchReserveTrap struct{ *sessionBenchModule }

func (*batchReserveTrap) Reserve(uint32) uint32 { panic("unreachable") }

func TestBatchReservationTrapBeforeOperation(t *testing.T) {
	t.Parallel()
	r, i, m, life := batchFixture(t)
	i.module = &batchReserveTrap{m}
	command := batchReply(m, nil, OK)
	command.Prepare = func(c *Call) error { return c.PrepareInput(0) }
	n, err := r.Batch(t.Context(), life, BatchLimits{}, []BatchCommand{batchReply(m, nil, OK), command, batchReply(m, nil, OK)})
	failure := assertBatchFailure(t, n, err, 1, 1, false)
	var fault *FaultError
	if !errors.As(failure, &fault) || fault.Op != "write" || fault.Kind != FaultUnreachable || i.Healthy() || i.Epoch() != 2 || m.writes != 1 {
		t.Fatal(failure)
	}
}

func TestBatchHomeFaultDuringExecutionLockWait(t *testing.T) {
	t.Parallel()
	r, i, m, life := batchFixture(t)
	entered, release := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	defer unblock()
	trap := make(chan error, 1)
	go func() {
		trap <- i.Call("other resource", func(*Call) error { close(entered); batchAwait(t, release); panic("unreachable") })
	}()
	batchAwait(t, entered)
	done := make(chan error, 1)
	go func() {
		n, err := r.Batch(t.Context(), life, BatchLimits{}, []BatchCommand{batchReply(m, nil, OK)})
		if n != 0 {
			t.Error(n)
		}
		done <- err
	}()
	batchWaitState(t, life, 1)
	// Wait until ownership is held, then fault the shared home while its execution
	// lock is still held. Either ownership validation or the instance state gate
	// may observe the fault; neither may enter the queued operation.
	deadline := time.Now().Add(5 * time.Second)
	for r.cell.mu.TryLock() {
		r.cell.mu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("batch did not acquire ownership")
		}
		runtime.Gosched()
	}
	unblock()
	var fault *FaultError
	if !errors.As(batchAwait(t, trap), &fault) {
		t.Fatal("missing original home fault")
	}
	failure := assertBatchFailure(t, 0, batchAwait(t, done), 0, -1, false)
	var handle *HandleError
	if !errors.Is(failure, ErrPoisoned) && !errors.As(failure, &handle) {
		t.Fatal(failure)
	}
	if m.writes != 0 || life.state.Load() != 0 || i.Epoch() != 2 {
		t.Fatal("queued batch executed on failed home")
	}
}

func TestBatchOwnershipSurvivesAbandonment(t *testing.T) {
	t.Parallel()
	i, m := sessionBenchInstance(t, 1)
	var destroyed atomic.Int32
	dropped := make(chan struct{}, 1)
	resource, err := NewResource[sessionBenchStream](i, 7, func(uint64) error { destroyed.Add(1); dropped <- struct{}{}; return nil })
	if err != nil {
		t.Fatal(err)
	}
	first := batchReply(m, nil, OK)
	invoke := first.Invoke
	first.Invoke = func(h uint64) Status {
		resource = nil
		for range 3 {
			runtime.GC()
			runtime.Gosched()
			if destroyed.Load() != 0 {
				t.Error("cleanup overlapped active buffer")
			}
		}
		return invoke(h)
	}
	second := batchReply(m, nil, OK)
	second.Decode = func([]byte) error {
		if destroyed.Load() != 0 {
			t.Error("cleanup ran between admitted commands")
		}
		return nil
	}
	n, batchErr := resource.Batch(t.Context(), &Lifecycle{}, BatchLimits{}, []BatchCommand{first, second})
	if n != 2 || batchErr != nil {
		t.Fatal(n, batchErr)
	}
	for range 3 {
		runtime.GC()
		runtime.Gosched()
	}
	batchAwait(t, dropped)
	if destroyed.Load() != 1 || m.writes != 2 {
		t.Fatal("abandoned resource not destroyed exactly once")
	}
}
