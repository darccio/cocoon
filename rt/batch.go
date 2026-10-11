package rt

import (
	"context"
	"errors"
	"fmt"
)

// BatchLimits bounds one ordered resource buffer. Zero fields select 32 calls
// and 1 MiB of encoded input/output; hard maxima are 256 calls and 16 MiB each.
// Output bounds retained wire bytes, not total host or guest memory.
type BatchLimits struct {
	MaxCalls       uint32
	MaxInputBytes  uint64
	MaxOutputBytes uint64
}

func (l BatchLimits) normalized() (BatchLimits, error) {
	if l.MaxCalls == 0 {
		l.MaxCalls = 32
	}
	if l.MaxInputBytes == 0 {
		l.MaxInputBytes = 1 << 20
	}
	if l.MaxOutputBytes == 0 {
		l.MaxOutputBytes = 1 << 20
	}
	if l.MaxCalls > 256 || l.MaxInputBytes > 16<<20 || l.MaxOutputBytes > 16<<20 {
		return l, ErrTooLarge
	}
	return l, nil
}

// BatchError identifies the failing or next command, or -1 for whole-buffer
// rejection. Attempted means that command's operation export was entered;
// successful earlier commands and possible failing-export effects remain.
type BatchError struct { //nolint:govet // Preserve the approved public positional field order.
	Index     int
	Attempted bool
	Cause     error
}

func (e *BatchError) Error() string {
	return fmt.Sprintf("batch command %d (attempted=%t): %v", e.Index, e.Attempted, e.Cause)
}

// Unwrap exposes the underlying context, ownership, guest, or limit error.
func (e *BatchError) Unwrap() error { return e.Cause }

// BatchCommand is a trusted generated adapter for one resource operation.
// Prepare must reserve and fill exactly InputBytes; it may be nil only for zero
// bytes. Invoke enters one export with the bound full-width handle. Decode
// validates and stages an owned result. Call/reply views must not escape, and
// callbacks must not re-enter the library, perform host I/O, or call other exports.
type BatchCommand struct { //nolint:govet // Preserve the approved public positional field order.
	Op         string
	InputBytes uint64
	Prepare    func(*Call) error
	Invoke     func(handle uint64) Status
	Decode     func(reply []byte) error
}

func batchArgument(message string) error {
	return &AppError{Op: "batch", Status: ErrArg, Message: message}
}

func batchFailure(index int, attempted bool, cause error) error {
	return &BatchError{Index: index, Attempted: attempted, Cause: cause}
}

// Batch admits once on the resource's actual library lifecycle and holds its
// ownership and execution locks for the ordered buffer. Every command retains
// the standalone checked boundary. Cancellation is checked between commands;
// synchronous execution and mutex waits cannot be interrupted. completed counts
// only commands whose entire boundary succeeded, including the memory check.
func (r *Resource[T]) Batch(ctx context.Context, life *Lifecycle, limits BatchLimits, commands []BatchCommand) (completed uint32, err error) {
	if ctx == nil || life == nil {
		return 0, batchFailure(-1, false, batchArgument("nil context or lifecycle"))
	}
	limits, err = limits.normalized()
	if err != nil {
		return 0, batchFailure(-1, false, err)
	}
	if uint64(len(commands)) > uint64(limits.MaxCalls) {
		return 0, batchFailure(-1, false, ErrTooLarge)
	}
	var input uint64
	for index, command := range commands {
		if command.Invoke == nil || command.Decode == nil || command.Prepare == nil && command.InputBytes != 0 {
			return 0, batchFailure(index, false, batchArgument("missing adapter callback"))
		}
		if command.InputBytes > limits.MaxInputBytes-input {
			return 0, batchFailure(index, false, ErrTooLarge)
		}
		input += command.InputBytes
	}
	if r == nil || r.cell == nil {
		return 0, batchFailure(-1, false, &HandleError{Op: "batch", Reason: "nil resource"})
	}
	for index, command := range commands {
		if command.InputBytes > r.cell.instance.limits.MaxInput {
			return 0, batchFailure(index, false, ErrTooLarge)
		}
	}
	if contextErr := ctx.Err(); contextErr != nil {
		return 0, batchFailure(-1, false, contextErr)
	}
	if admissionErr := life.Enter(); admissionErr != nil {
		return 0, batchFailure(-1, false, admissionErr)
	}
	defer life.Leave()
	index, attempted := -1, false
	err = r.Use("batch", func(i *Instance, handle uint64) error {
		i.mu.Lock()
		defer i.mu.Unlock()
		switch i.state {
		case instanceClosed:
			return ErrClosed
		case instancePoisoned:
			return ErrPoisoned
		case instanceOpen:
		}
		if len(commands) != 0 {
			index = 0
		}
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
		remaining := limits.MaxOutputBytes
		for k, command := range commands {
			index, attempted = k, false
			if contextErr := ctx.Err(); contextErr != nil {
				return contextErr
			}
			if checkedErr := i.checked(command.Op, func(call *Call) error {
				if command.Prepare != nil {
					if prepareErr := command.Prepare(call); prepareErr != nil {
						return prepareErr
					}
				}
				if uint64(call.capacity) != command.InputBytes || uint64(call.used) != command.InputBytes {
					return batchArgument("reservation does not match declared input")
				}
				attempted = true
				status := command.Invoke(handle)
				// Reuse structural validation before interpreting the actual status.
				// Keep batch wire accounting out of standalone ResultView entirely.
				view, replyErr := call.ResultView(command.Op, OK)
				if replyErr != nil {
					return replyErr
				}
				wireBytes := uint64(len(view))
				if status != OK {
					replyErr = FromStatus(command.Op, status, view)
				}
				if replyErr != nil && errors.Is(replyErr, ErrProtocol) {
					return replyErr
				}
				if replyErr == nil {
					if decodeErr := command.Decode(view); decodeErr != nil {
						return decodeErr
					}
				}
				if wireBytes > remaining {
					return ErrTooLarge
				}
				remaining -= wireBytes
				return replyErr
			}); checkedErr != nil {
				return checkedErr
			}
			completed++
		}
		return nil
	})
	if err != nil {
		return completed, batchFailure(index, attempted, err)
	}
	return completed, nil
}
