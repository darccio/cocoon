package rt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
)

// ABIVersion identifies the synchronous buffer and generation-handle protocol.
const ABIVersion = 3

// Module exposes the current linear memory, including after memory growth.
type Module interface {
	Memory() []byte
}

// ABI is implemented by generated adapters over typed wasm2go exports.
type ABI interface {
	Module
	Version() uint32
	Schema() uint64
	Init()
	Reserve(size uint32) uint32
	// Output returns a descriptor address fixed for the initialized lifetime.
	Output() uint32
	Trim(keep uint32)
}

// Limits bounds bytes per call; MaxMemory is a per-instance linear memory cap.
type Limits struct {
	MaxInput  uint64
	MaxOutput uint64
	MaxMemory uint64
}

// Validate rejects limits that cannot be represented by the wasm32 ABI.
func (l Limits) Validate() error {
	if l.MaxInput == 0 || l.MaxOutput == 0 || l.MaxMemory == 0 ||
		l.MaxInput > math.MaxUint32 || l.MaxOutput > math.MaxUint32 ||
		l.MaxMemory > uint64(math.MaxInt) || l.MaxInput > l.MaxMemory || l.MaxOutput > l.MaxMemory {
		return fmt.Errorf("%w: invalid instance limits", ErrTooLarge)
	}
	return nil
}

// InputSize checks an aggregate input size before multiplication or allocation.
func InputSize(limit, count, width uint64, other ...uint64) (uint64, error) {
	if width != 0 && count > limit/width {
		return 0, ErrTooLarge
	}
	size := count * width
	for _, part := range other {
		if part > limit || size > limit-part {
			return 0, ErrTooLarge
		}
		size += part
	}
	if size > limit || size > math.MaxUint32 || size > uint64(math.MaxInt) {
		return 0, ErrTooLarge
	}
	return size, nil
}

// Range bounds a view by logical memory length, including zero-length views.
func Range(memory []byte, pointer, size uint32) ([]byte, error) {
	end := uint64(pointer) + uint64(size)
	if end > uint64(len(memory)) {
		return nil, &memoryRangeError{pointer: pointer, end: end, length: len(memory)}
	}
	return memory[pointer:end:end], nil
}

// Small error construction lets Range inline its checked success path.
type memoryRangeError struct {
	end     uint64
	length  int
	pointer uint32
}

func (e *memoryRangeError) Error() string {
	return fmt.Sprintf("%s: memory range [%d,%d) exceeds %d", ErrProtocol, e.pointer, e.end, e.length)
}

func (*memoryRangeError) Unwrap() error { return ErrProtocol }

type instanceState uint8

const (
	instanceOpen instanceState = iota
	instancePoisoned
	instanceClosed
)

// Instance serializes guest execution and contains all recoverable guest panics.
// Guest execution cannot be preempted. Callbacks must not reenter this instance.
type Instance struct {
	module ABI
	host   *Host
	call   Call
	limits Limits
	epoch  atomic.Uint64
	mu     sync.Mutex
	alive  atomic.Bool
	state  instanceState
	output uint32
}

// NewInstance validates identity and contains traps during initialization.
func NewInstance(module ABI, schema uint64, limits Limits, hosts ...*Host) (*Instance, error) {
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	i := &Instance{module: module, limits: limits}
	if len(hosts) != 0 {
		i.host = hosts[0]
		if i.host != nil {
			i.host.Bind(module)
		}
	}
	i.epoch.Store(1)
	i.alive.Store(true)
	err := i.Call("init", func(_ *Call) error {
		if module.Version() != ABIVersion {
			return ErrABI
		}
		if module.Schema() != schema {
			return ErrSchema
		}
		module.Init()
		if uint64(len(module.Memory())) > limits.MaxMemory {
			return fmt.Errorf("%w: initial memory", ErrTooLarge)
		}
		i.output = module.Output()
		_, outputErr := Range(module.Memory(), i.output, 8)
		return outputErr
	})
	if err != nil {
		i.Close()
		return nil, err
	}
	return i, nil
}

// Epoch binds resources to this instance's current uninterrupted lifetime.
func (i *Instance) Epoch() uint64 { return i.epoch.Load() }

// Healthy reports whether resources may still be created on this instance.
func (i *Instance) Healthy() bool { return i.alive.Load() }

// Call runs a callback while holding the instance's guest execution lock.
func (i *Instance) Call(op string, invoke func(call *Call) error) (err error) {
	i.mu.Lock()
	defer i.mu.Unlock()
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

func (i *Instance) poison() {
	i.alive.Store(false)
	i.state = instancePoisoned
	i.epoch.Add(1)
}

// Close permanently releases the module and invalidates every resource.
func (i *Instance) Close() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.state != instanceClosed {
		i.alive.Store(false)
		i.state = instanceClosed
		i.epoch.Add(1)
		i.module = nil
		if i.host != nil {
			i.host.module = nil
		}
	}
}

// Recycle trims buffers and reports whether this instance can return to a pool.
func (i *Instance) Recycle(keep uint32, maximum uint64) bool {
	return i.Call("trim", func(_ *Call) error {
		i.module.Trim(keep)
		if uint64(len(i.module.Memory())) > maximum {
			return ErrTooLarge
		}
		return nil
	}) == nil
}

// Call exposes memory only for the duration of Instance.Call.
type Call struct {
	instance *Instance
	pointer  uint32
	capacity uint32
	used     uint32
}

// PrepareInput reserves all variable inputs together so their pointers stay stable.
func (c *Call) PrepareInput(size uint64) error {
	checked, err := InputSize(c.instance.limits.MaxInput, size, 1)
	if err != nil {
		return err
	}
	c.capacity = uint32(checked) // #nosec G115 -- InputSize checks the wasm32 bound.
	c.pointer = c.instance.module.Reserve(c.capacity)
	c.used = 0
	_, err = c.Range(c.pointer, c.capacity)
	return err
}

// PutBytes writes one input into the reservation and returns its guest pointer.
func (c *Call) PutBytes(data []byte) (uint32, error) {
	pointer, view, err := c.inputRegion(len(data))
	if err != nil {
		return 0, err
	}
	copy(view, data)
	return pointer, nil
}

// PutString copies directly from a string without a temporary byte allocation.
func (c *Call) PutString(data string) (uint32, error) {
	pointer, view, err := c.inputRegion(len(data))
	if err != nil {
		return 0, err
	}
	copy(view, data)
	return pointer, nil
}

func (c *Call) inputRegion(length int) (regionPointer uint32, region []byte, regionErr error) {
	if length < 0 || uint64(length) > uint64(c.capacity-c.used) {
		return 0, nil, ErrTooLarge
	}
	pointer := uint64(c.pointer) + uint64(c.used)
	if pointer > math.MaxUint32 {
		return 0, nil, ErrProtocol
	}
	ptr := uint32(pointer) // #nosec G115 -- Pointer arithmetic is checked above.
	n := uint32(length)    // #nosec G115 -- Input is bounded by the uint32 reservation.
	view, err := c.Range(ptr, n)
	if err != nil {
		return 0, nil, err
	}
	c.used += n
	return ptr, view, nil
}

// Range re-reads memory after every possible guest allocation or growth.
func (c *Call) Range(pointer, size uint32) ([]byte, error) {
	return Range(c.instance.module.Memory(), pointer, size)
}

// Result checks and copies the guest reply before the call releases its lock.
func (c *Call) Result(op string, status Status) ([]byte, error) {
	view, err := c.ResultView(op, status)
	if err != nil {
		return nil, err
	}
	return append([]byte(nil), view...), nil
}

// ResultView validates a borrowed reply for decoding under the execution lock.
// The view must not escape the call or survive another guest invocation. Decoders
// must copy any strings, bytes, or record fields retained in their typed result.
func (c *Call) ResultView(op string, status Status) ([]byte, error) {
	memory := c.instance.module.Memory()
	descriptor, err := Range(memory, c.instance.output, 8)
	if err != nil {
		return nil, err
	}
	pointer := binary.LittleEndian.Uint32(descriptor)
	size := binary.LittleEndian.Uint32(descriptor[4:])
	if uint64(size) > c.instance.limits.MaxOutput {
		return nil, ErrTooLarge
	}
	view, err := Range(memory, pointer, size)
	if err != nil {
		return nil, err
	}
	if status != OK {
		if err := FromStatus(op, status, view); err != nil {
			return nil, err
		}
	}
	return view, nil
}
