// Package rt implements Cocoon's synchronous ABI, isolation, and ownership contracts.
package rt

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// Status is an ABI v3 operation result.
type Status int32

// ABI result codes are stable across generated packages.
const (
	OK        Status = 0
	ErrApp    Status = 1
	ErrArg    Status = 2
	ErrHandle Status = 3
	Pending   Status = 4
	ErrLimit  Status = 5
)

// Sentinel errors support errors.Is across facade and runtime boundaries.
var (
	ErrClosed   = errors.New("cocoon: closed")
	ErrPoisoned = errors.New("cocoon: poisoned instance")
	ErrTooLarge = errors.New("cocoon: size limit exceeded")
	ErrABI      = errors.New("cocoon: incompatible ABI")
	ErrSchema   = errors.New("cocoon: incompatible schema")
	ErrProtocol = errors.New("cocoon: invalid guest reply")
)

// AppError is an expected guest failure or invalid argument.
type AppError struct {
	Op      string
	Message string
	Status  Status
}

func (e *AppError) Error() string { return fmt.Sprintf("%s: %s", e.Op, e.Message) }

// HandleError describes a closed, stale, or foreign resource handle.
type HandleError struct {
	Op     string
	Reason string
}

func (e *HandleError) Error() string { return fmt.Sprintf("%s: invalid handle: %s", e.Op, e.Reason) }

// FaultKind classifies a recoverable guest trap.
type FaultKind uint8

// Fault categories are independent of the guest panic's Go representation.
const (
	FaultUnknown FaultKind = iota
	FaultUnreachable
	FaultMemory
	FaultArithmetic
	FaultIndirectCall
	FaultTable
)

func (k FaultKind) String() string {
	switch k {
	case FaultUnknown:
		return "unknown"
	case FaultUnreachable:
		return "unreachable"
	case FaultMemory:
		return "memory"
	case FaultArithmetic:
		return "arithmetic"
	case FaultIndirectCall:
		return "indirect call"
	case FaultTable:
		return "table"
	default:
		return "unknown"
	}
}

// FaultError poisons its instance and invalidates that instance's resources.
type FaultError struct {
	Op       string
	Detail   string
	GuestMsg string
	Kind     FaultKind
}

func (e *FaultError) Error() string {
	detail := e.Detail
	if e.GuestMsg != "" {
		detail = e.GuestMsg
	}
	return fmt.Sprintf("%s: guest %s fault: %s", e.Op, e.Kind, detail)
}

// HostError distinguishes a panic in a host import from a guest trap.
type HostError struct {
	Value any
	Op    string
}

func (e *HostError) Error() string { return fmt.Sprintf("%s: host import panic: %v", e.Op, e.Value) }

type hostPanic struct{ value any }

// Classify contains every panic value; unknown panics remain terminal faults.
func Classify(op string, value any, guestMessage string) error {
	if host, ok := value.(hostPanic); ok {
		return &HostError{Op: op, Value: host.value}
	}
	kind := FaultUnknown
	if message, ok := value.(string); ok {
		switch message {
		case "unreachable":
			kind = FaultUnreachable
		case "integer overflow", "invalid conversion to integer":
			kind = FaultArithmetic
		case "out of bounds memory access":
			kind = FaultMemory
		case "uninitialized element", "indirect call type mismatch":
			kind = FaultIndirectCall
		case "out of bounds table access":
			kind = FaultTable
		}
	}
	if runtimeErr, ok := value.(runtime.Error); ok {
		message := runtimeErr.Error()
		switch {
		case strings.HasPrefix(message, "runtime error: integer divide by zero"):
			kind = FaultArithmetic
		case strings.HasPrefix(message, "runtime error: index out of range"),
			strings.HasPrefix(message, "runtime error: slice bounds out of range"):
			kind = FaultMemory
		case strings.HasPrefix(message, "interface conversion:"):
			kind = FaultIndirectCall
		}
	}
	return &FaultError{Op: op, Kind: kind, Detail: fmt.Sprint(value), GuestMsg: guestMessage}
}

// FromStatus interprets an operation status without accepting reserved codes.
func FromStatus(op string, status Status, message []byte) error {
	switch status {
	case OK:
		return nil
	case ErrApp, ErrArg:
		return &AppError{Op: op, Status: status, Message: string(message)}
	case ErrHandle:
		return &HandleError{Op: op, Reason: string(message)}
	case ErrLimit:
		return fmt.Errorf("%s: %w: %s", op, ErrTooLarge, message)
	case Pending:
		return fmt.Errorf("%s: %w: async status in synchronous operation", op, ErrProtocol)
	default:
		return fmt.Errorf("%s: %w: unknown status %d", op, ErrProtocol, status)
	}
}
