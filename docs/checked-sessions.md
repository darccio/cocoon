# Bounded checked resource commands — approved contract

Status: explicitly approved by the operator, 2026-10-10 ("Approved and
unlocked"). Phase 0 is complete. Approval covers the public Go and manifest
contracts and the complete approval boundary below; Phase 2 may proceed after
the signed review commits. The runtime bridge and lazy homes are implemented
and reviewed, with opt-in typed facade commands and compiled consumer proofs.
The translated streaming proof is implemented with native, contract, wazero,
bulk-memory, and stateful fuzz coverage. Final performance qualification remains pending.
Runtime evidence and rejected shapes are recorded in the
performance notes.

The evidence, raw samples, provenance, and negative results are in
[performance notes](performance.md#bounded-checked-sessions-phase-0-2026-10-10)
and the [complete sample log](measurements/checked-sessions-phase0.txt).
The test-only prototype is not an implementation of this contract.

## Decision and scope

Propose both finite resource command buffers and optional bounded resource
home sharding. At GOMAXPROCS 16, synthetic 64-byte writes achieve 5.22 Mcalls/s
on the current shared-home shape, 8.99 with 32 checked calls sharing admission,
ownership, and execution locking, and 38.83 with independent worker instances
and the same sequence. Independent instances with standalone calls achieve
only 5.45 Mcalls/s. This is consistent with shared per-call admission contention,
but sequencing also changes ownership/execution locking and loop overhead;
the contended admission share is not isolated.
For 4 KiB writes the corresponding sequence ceilings are much lower, 0.974
and 1.667 Mcalls/s. These are attainable runtime hypotheses, not translated
streaming performance or an accepted implementation gain.

A buffer executes a finite, already available list of commands on one existing
resource. It admits against its library and uses that resource's exact home.
It never borrows a stateless pool slot or receives its own instance. Its
admission, ownership lock, and execution lock cover that buffer only. A host
can process a stream with repeated bounded buffers, releasing all locks between
them. Network reads and caller-supplied orchestration callbacks are not part of
the generated Batch API.

Home sharding supplies concurrency between resources. A resource is assigned
one of a bounded number of library-owned homes at construction and never moves.
Commands inside one resource remain ordered and serialized. Dedicated instances
per resource are rejected for this iteration: the live resource count could
make instance memory unbounded. Pinning resources in the stateless pool is
also rejected: long-lived streams could consume all stateless slots.

The standalone constructors, methods, resource Close, and library Close retain
their signatures and default behavior. This is a throughput feature. No
standalone scalar latency improvement is promised. ABI v3, guest exports,
generation handles, record encoding, capabilities, and synchronous execution
remain unchanged. Async, Tokio, HTTP capabilities, and the external Datadog
demo and injector changes are out of scope.

## Public surface requiring approval

The proposed additions are:

- Generated `Options.ResourceInstances int`. Zero means one resource home,
  independently of `Options.Instances` and GOMAXPROCS. Values 1–256 select that
  many lazy homes; negative or larger values fail Open with `rt.ErrTooLarge`.
  Stateless pool configuration is unchanged. The opt-in changes which resources
  share guest instance-global state and multiplies possible instance memory.
- Per-resource manifest directive `checked_batches = true`, default false.
  It enables typed commands and `Batch` only for that resource. It changes
  generated host API/build provenance, not ABI/schema identity: the same guest
  methods and wire encodings are composed. Record the directive in the full
  manifest/build lock. Existing manifests remain valid without it.
- Runtime `rt.BatchLimits`, `rt.BatchError`, `rt.BatchCommand`, the low-level
  `rt.Resource[T].Batch` bridge, and generated typed command/result
  types and constructors, plus the resource's `Batch` method below. These are
  additions to the public API. Adding an Options field can affect external
  unkeyed literals; this source-compatibility limit must be documented.

For a fixture declaring Stream.Write(bytes) -> bytes and Stream.End() -> bytes,
the generated API shape is:

```go
type StreamCommand interface { /* unexported resource-specific marker */ }
func StreamWriteCommand(chunk []byte) StreamCommand
func StreamEndCommand() StreamCommand

type StreamResult interface { /* unexported resource-specific marker */ }
type StreamWriteResult struct { Value []byte }
type StreamEndResult struct { Value []byte }

func (s *Stream) Batch(
    ctx context.Context,
    commands []StreamCommand,
    limits rt.BatchLimits,
) ([]StreamResult, error)

type BatchLimits struct {
    MaxCalls       uint32
    MaxInputBytes  uint64
    MaxOutputBytes uint64
}

type BatchError struct {
    Index     int
    Attempted bool
    Cause     error
}
```

Other resources get their own command/result types, with each result's Value
having the method's declared Go type. Unit methods get a tagged result without
a Value. The generated resource API introduces no `any` result payload or
untyped caller-supplied guest callback. `BatchError.Unwrap` exposes Cause for
errors.Is/errors.As. Exported
identifiers shown here are the proposed spelling, not yet shipped symbols.

New and Close are excluded from commands. A buffer targets one resource;
commands cannot carry another resource's handle. Nil commands and dynamic
types other than the generated command implementations are rejected during
preflight, including types that satisfy the marker through embedding.
Generator validation must reject all enabled Batch/command/result identifier
collisions before publication; opting out leaves existing method names usable.

Command constructors retain references to caller-owned payloads without copying
large values eagerly. Payloads are read at Batch time: changing/reusing backing
storage before execution changes queued inputs. Caller-owned command values keep
those references until the caller releases them. Callers must keep the commands
slice, payload slices, and record contents stable while Batch reads them, as
for standalone calls. The runtime retains
neither inputs, commands, borrowed guest views, nor a callable session after
Batch returns. Returned typed results and errors own their retained data.

Generated consumer packages cannot access runtime-private locks. Their exact
public execution bridge is therefore also part of this approval:

```go
// Package rt; callbacks are trusted host adapter code.
type BatchCommand struct {
    Op         string
    InputBytes uint64
    Prepare    func(*Call) error
    Invoke     func(handle uint64) Status
    Decode     func(reply []byte) error
}

func (r *Resource[T]) Batch(
    ctx context.Context,
    life *Lifecycle,
    limits BatchLimits,
    commands []BatchCommand,
) (completed uint32, err error)
```

The generated facade supplies its library's Lifecycle and a bounded generated
plan. The bridge performs preflight metadata validation, one admission,
ownership/epoch validation, locking, contexts, per-command checks, reply/status
validation, budgets, and partial-completion/error accounting. The caller must
pass the resource's actual library lifecycle; a standalone runtime user may
provide its own shared lifecycle. Neither may substitute an unrelated lifecycle
to evade shutdown. Nil lifecycle/Invoke/Decode are host argument errors.

Prepare encodes only that command's inputs through the fresh Call. It may be
nil only when InputBytes is zero; actual reservation capacity/used must match
the declared size before Invoke. A metadata/reservation mismatch is a host
argument error (ErrArg), Index=current, Attempted=false, without poisoning
unless preparation independently produced an existing fault. Invoke enters
exactly one operation export using the supplied full-width bound handle and
returns its status. The bridge
sets Attempted immediately before Invoke, so preparation failures and export
failures are distinguishable even when they panic. Decode runs only after an
OK reply passes descriptor/status checks; it performs complete typed validation
and stages owned results. Unit decoders reject nonempty success replies.

Prepare/Invoke/Decode must not block on host I/O, re-enter the library, call other
exports, retain Call/reply views, or use them from another goroutine. These are
trusted adapter callbacks, like existing Instance.Call/Resource.Use callbacks,
not an untrusted-host sandbox. Generated user-facing Batch exposes only typed
commands and owned results. Low-level callbacks cannot escape the synchronous
bridge lifetime; every command still receives a separate checked boundary.
The facade publishes/truncates its staged results to completed and clears any
failed-command staging so hidden array capacity cannot retain failed results.

## Bounds and admission

Zero fields in BatchLimits select defaults: 32 calls, 1 MiB encoded input,
and 1 MiB retained encoded output. Hard maxima are 256 calls and 16 MiB for
each byte budget. Nonzero values may lower or raise defaults within those
maxima. Validate limits and command count before allocating a command plan.
The measured N=16/32 region is a useful starting point; N=64 did not improve
the shared-home 64-byte ceiling at GOMAXPROCS 16. Limits bound call/byte work,
not elapsed time or arbitrary guest CPU cost.

Preflight sizes the complete buffer with overflow checks, including every
variable argument, scalar array width, and record header. Aggregate encoded
input must fit MaxInputBytes, and every individual command must also fit the
existing library/manifest MaxInput. Standalone scalar arguments and the implicit
resource handle consume call budget but no ABI input-buffer byte budget.
Preflight does not execute guest code, reserve guest memory, or claim to
validate payload-specific semantic arguments. An invalid later preflight
command rejects the entire buffer before any effect.

Preflight validates command types and encoded sizes only. UTF-8 and other host
wire-format/record encoding checks remain in each command's Prepare, as in
standalone generation; an encoding failure in command k returns the earlier
successful prefix and Attempted=false. It does not reject already completed
commands retroactively. Preparation errors retain their existing underlying
classification; this is distinct from guest argument errors after Invoke.

After preflight and a context check, Batch enters the library Lifecycle exactly
once. It then acquires shared resource ownership, verifies its exact instance
and epoch, acquires that instance's execution lock, and checks context again.
The interval from successful Enter through deferred Leave is one admitted
buffer, including lock waits. The ownership and execution locks each cover
the ordered command loop. All releases and KeepAlive paths are defer-safe.
There is no additional stateless-pool admission or recycling.

Nil context is a host argument error, not a panic. Invalid/nil command types,
missing required bridge callbacks, and nil lifecycle likewise use an
`rt.AppError{Op: "batch", Status: rt.ErrArg}` as Cause; they do not poison an
instance. For nil context/lifecycle use Index=-1; invalid command metadata/type
uses its command index. All set Attempted=false. Resource ownership failures
(nil/closed/foreign epoch) use the existing HandleError with Index=-1 and
Attempted=false. An unavailable instance before the loop uses its existing
ErrClosed/ErrPoisoned with Index=-1, Attempted=false, including empty buffers.

Empty buffers still validate options, context, ownership, and library admission,
then return an empty result without invoking the guest. They cannot bypass
terminal Close. Cancellation after acquiring locks for an empty buffer uses
Index=-1, Attempted=false. Before admission, count/input/limit failures return
no results and a BatchError with Attempted=false; Index identifies the offending command
when one exists and is -1 for whole-buffer limits. Pre-admission cancellation
or library rejection likewise uses Index=-1, Attempted=false.

Every reply is checked against the current descriptor, existing MaxOutput,
current logical memory, status, and declared result encoding before publishing
a result. Each command's validated wire reply bytes, including error payloads,
are charged against the remaining output budget. Do not copy a retained reply
that would exceed that budget; stop with ErrTooLarge and Attempted=true.
When validation requires a temporary owned decode/error, it remains limited
by that operation's MaxOutput and is discarded on budget failure. Protocol
faults take precedence over the additional batch-budget error when the normal
per-operation checks classify them as faults. Existing per-operation validation
precedence, including MaxOutput errors, must be preserved.
For an otherwise valid expected guest-error payload that exceeds the remaining
budget, return the batch ErrTooLarge rather than retaining that oversized
application/handle error message. The failing export's effects still remain.

MaxOutputBytes bounds retained encoded reply/error bytes, not guest memory or
total process RSS. A failed command can generate a within-MaxOutput guest
reply larger than the remaining budget. The implementation must not hide that
effect: its failing result is not published, Attempted=true, and the buffer
stops. Peak host staging consists of bounded command metadata, inputs read
from the caller, retained results within the budget, one operation's bounded
input-encoding staging, and at most one operation's bounded temporary decode.
Current record encoding can allocate nested field encodings plus the aggregate;
account for this using MaxInput and the manifest's bounded record shape/depth,
without assuming a new low-copy encoder. Go object overhead and guest linear
memory are separate. This bounded overshoot for validation needs explicit
approval with the rest of the contract; it is not a promise to preempt guest allocation.

The encoded-output budget does not cover host/context/ownership diagnostics or
panic metadata outside a wire reply. Existing Rust panic-hook GuestMsg and
HostError.Value behavior remains; a host panic value can retain arbitrary
caller-owned host data. Account for these separately under existing capability/
instance behavior, without asserting an RSS bound or inventing new log limits.

## Per-command checked boundary

Under the shared lock, every command must still perform the current checks:

1. Verify instance open state; reset Call reservation/pointer/capacity/used and
   clear previous host panic metadata. No command can inherit a prior call's
   input reservation or panic message.
2. Check the individual input size; reserve, encode, and re-read current guest
   memory after any allocation/growth. Guest reservation and handle/type/
   generation checks execute for each export as today.
3. Invoke exactly that guest export, validate the reply completely, enforce
   output budget, and decode/copy owned output before the next invocation.
   No borrowed view remains usable across commands, even under the same lock.
4. Check per-instance MaxMemory after every command, including error returns.
   Contain and classify recoverable guest/host panics for that command.
5. Immediately poison and increment the epoch on protocol errors, traps, or
   excess linear memory according to existing policy. Stop the buffer on any
   error. Never run a following command after a fault.

Stage the decoded result until the entire checked core returns success,
including the post-command MaxMemory check. A valid reply followed by excess
memory produces no successful result for that command; discard its staging.

The per-command core must be shared with standalone Instance.Call rather than
leaving a second production copy of the benchmark's experimental helper.
Its standalone path retains the current lock and behavior. No safety check is
moved to a single end-of-buffer validation. The additional input/output budgets
are host errors and do not themselves poison a healthy instance.

The implementation shares this core at generation time: one authored runtime
template emits the standalone locked adapter and the private held-lock adapter.
Only lock acquisition/release differs. Normal tests reproduce the committed
generated region in authored instance.go and compare every checked statement
after removing the lock prefix. Update the template and run go generate ./rt;
the template, tool, and output move together. The runtime remains subject to
all strict linters. This preserves one
safety policy without adding a helper frame to standalone execution.

## Cancellation, partial completion, and re-entry

After rejecting nil context, check cancellation before admission, after acquiring ownership
and execution, and before each command boundary. Cancellation prevents a new
command from starting; it cannot interrupt an executing synchronous guest or
the existing mutex waits. A canceled lock waiter returns after it acquires and
releases the locks without executing. No cancellable-mutex or forced-abort
promise is added. Deadline expiration cannot bound infinite guest execution.

After command k succeeds, publish its owned typed result into the successful
prefix. If cancellation arrives during k, finish its full checked boundary and
preserve that result; stop before k+1. If every command has finished, return
success even if cancellation arrives after the last command. For a guest error
or fault already produced by k, that error wins over simultaneous cancellation.

On failure return the successful prefix and *BatchError. Index is the zero-based
failing/next command. Attempted=true means that command's guest operation export
was entered; it may have mutated state even without a successful result.
Failures during input reservation/encoding or cancellation between commands
set Attempted=false. A reservation trap can nevertheless poison the instance.
Completed count is len(results); no failed command receives a fabricated result.
Preflight rejects the whole plan, so it returns an empty prefix even for an
invalid later command. Success-only fixture checks do not prove these rules.

Buffers are ordered, not transactional. Earlier effects remain after expected
application/argument/handle errors or output-budget failure; the failing export's
effects are payload-dependent. Faults invalidate all handles on that home.
Healthy expected-error paths may be used in a later buffer. Neither the buffer
nor an instance reset rolls back guest state.

The generated resource API has no caller callback exposing rt.Call, raw memory,
unlocked execution, or an escapable session object. Its adapter uses the explicitly
approved low-level runtime bridge above. Host capability callbacks invoked during a buffer
must not call methods, constructors, Batch, resource Close, or library Close
on that same library; these can deadlock or create lock cycles across homes.
Ordinary concurrent callers are allowed and serialize on the relevant locks.
No command can perform host I/O outside its declared synchronous capability
as a way of extending the command plan.

## Home placement, poisoning, and shutdown

ResourceInstances configures a separate lazy set of home slots. Constructor
attempts select successive slots round-robin under a short metadata lock,
advancing the cursor even on failed attempts. Initialization and guest execution
must not hold the library-wide placement lock. Each home slot serializes lazy
creation/replacement; the number of concurrent creations and current homes
is bounded by ResourceInstances. No live-resource balancing or sticky host key
is promised. Uneven resource lifetimes can still cause uneven lane load.

Each successful constructor binds its wrapper/ownership cell/destructor to the
exact selected home and epoch. A home fault immediately invalidates all its
resource types and aliases. Other homes and the stateless pool are unaffected.
A later constructor assigned that slot replaces its poisoned home using the
existing retirement/Close policy. Old handles remain bound to the old instance;
they never migrate or resolve successfully against its replacement, even if
the guest reuses the same numeric handle bits. Initialization failure must
restore the slot's ability to retry without exceeding creation capacity.

Aliased wrappers share ownership and destruction as today. Resource Close
waits for its active buffer's ownership lock, invalidates all aliases, and runs
the destructor once outside the cell lock. Explicit Close remains preferred.
A host that abandons a stream cancels its current buffer if applicable and
closes the resource. If it drops all wrappers, existing best-effort cleanup
remains the fallback. There is no suspended session, independently retained
lease, automatic End, rollback, or new guaranteed GC drain. End and Close are
separate: the fixture's End emits the tail; Close can discard abandoned state.

Library Close sets the same terminal admission boundary immediately. It rejects
new buffers, constructors, ordinary standalone methods, and pool calls; it
drains every already admitted buffer through its entire bounded list or earlier error/
cancellation. In-flight buffers do not re-enter admission for each command,
so shutdown does not reject the remainder of already admitted healthy work.
After drain, release every current home and pool instance once and invalidate
resources. Retired-home cleanup must not lose or duplicate instance disposal.
Concurrent Close callers share the terminal result as today. Synchronous guest
execution and mutex waits still have no forced-abort deadline.
Resource.Close retains its existing destructor cleanup path without a library
Lifecycle admission. It remains usable for cleanup during/after terminal
shutdown and serializes with instance disposal as today; no new admission rule
is imposed on that path.

Host pauses between successive buffers hold no admission or guest lock. No
fairness or maximum scheduling delay is promised by the mutex; qualification
must measure competing-stream latency before choosing recommended buffer sizes.
Resource-home memory is bounded per instance, not process-wide; possible current
instance memory is roughly (stateless pool size + ResourceInstances) times
MaxMemory, with resource/slab/host allocations and temporarily retiring instances
accounted separately. Opt-in sharding is not an unlimited-instance escape hatch.

## Implementation and acceptance after approval

Start each small implementation round with guest-side, host-side, and
adversarial review. Retain changes only after darna --committable --dependants,
staged validation, exact staged-snapshot tests, and strict lint. First factor
the checked per-operation core without changing standalone behavior; then add
bounded commands and home placement in separately reviewable commits.

Add an authored streaming fixture under testdata: owned state, bounded 1 KiB
holdback, a partial-match pattern spanning chunk boundaries, writes returning
owned emittable bytes, End flushing the tail, and Close discarding state.
Generate its contracts, bulk-memory tests, and wazero adapters. Generated
single-operation adapters currently use invalid method handles; supplement
them with explicit valid constructor -> mixed writes -> End -> Close sequences
and a whole-input oracle. Verify empty/exact-window chunks, every partial-match
split, growth, retained outputs, limits, errors/traps in the middle, stale/
foreign handles, and differential status equality, not only equal messages.

Add host regressions for command count/byte overflow and malformed plans,
per-call reset and panic-message reset, malformed replies and decoder failures,
immediate poisoning, cancellation while waiting and between commands, alias
Close, abandonment/GC, terminal drain races, round-robin/initialization failure,
poisoned-slot replacement, sibling-home survival, and total instance capacity.
No test may retain a valid Call or borrowed view after its command.
Include malformed typed output with exhausted remaining output budget, an
expected guest error exceeding that budget, a valid decode followed by excess
MaxMemory, and a middle host encoding failure. These must verify error
precedence, Attempted, and the exact published prefix.

Regenerate both existing proof packages and their locks from authored/runtime/
generator sources, plus the new fixture. Never edit translated modules alone.
Complete make check, make race, make rust, make cross, make smoke, make fuzz,
and make integration, then repeat proof builds and compare artifacts/locks.
Record native versus cross-compiled coverage and subprocess race limitations.

Before/after measurements use frozen matching binaries, Go 1.26.8, CGO disabled,
PGO off, isolated execution, and repeated rotated pairs for scalar, SQL, batch,
trace, varied SQL/scalar, negative bins, streaming chunk sizes, and independent
streaming holdouts. Include GOMAXPROCS contention, command-length sensitivity,
construction/write/End/Close, live-instance memory, mixed stateless traffic,
and competing-stream latency. Extra context/budget work in this proposed API
was not present in the Phase 0 prototype and must be measured.

Retain each stage only for repeatable end-to-end throughput gains without
material standalone scalar, SQL, batch, or holdout regressions. Record rejected
changes and all samples; do not add screening gains or mark M1 performance
accepted. Sharding/buffer trade-offs may require narrower opt-in guidance if
large chunks, fairness, or constructor costs erase the ceiling advantage.

## Approved boundary

The operator's approval covers the new Options field and its 1–256 bound, manifest opt-in,
typed command/result API, the low-level runtime bridge/callback restrictions,
error semantics, default/hard budgets including
one-operation validation staging, cancellation/mutex-wait behavior, and
resource-home placement/lifecycle rules. This approval authorizes the reviewed
implementation and qualification plan; it does not accept unmeasured gains or
mark M1 performance accepted.
