# Low-copy numeric inputs

This specifies the private host encoding stage of roadmap track 1. Phase 0
[measurements](performance.md#low-copy-batching-measurements-before-implementation-2026-10-11)
support an implementation experiment, not performance acceptance. Guest, host,
and adversarial reviews precede implementation. No public API, manifest option,
ABI version, or guest lifetime contract is added.

## Scope

Optimize top-level `[]i32`, `[]u32`, `[]i64`, `[]u64`, `[]f32`, and `[]f64`
parameters for stateless functions, resource constructors/methods, and checked
resource commands. Preserve existing owned slice encoders for record fields and
raw contract/differential references. Strings, bytes, and records retain their
existing paths. Rust still copies input bytes, decodes an owned vector, validates
the entire batch, then performs mutations in original order.

Scalar-only, string/bytes-only, and record-only operation bodies stay unchanged.
The implementation changes the generator and regenerates every proof and lock;
it never optimizes generated modules independently.

## Checked reservation protocol

Standalone calls retain admission and all argument/aggregate size checks, then
ordered UTF-8 and record encoding validation before ownership resolution or
guest reservation. Numeric slices need no host value-domain validation. Checked
Batch retains whole-plan sizing and per-command encoding validation during
Prepare; UTF-8 failures do not move to whole-plan preflight.

Each call still makes one `PrepareInput(total)` under its existing execution
lock. Numeric parameters use a private generated helper receiving the exact
checked byte size for that slice. In declaration order, the helper:

1. Obtains the current checked reservation cursor with `PutBytes(nil)`.
2. Borrows a logical-memory-bounded view using `Range(pointer, byteSize)`.
3. Calls `PutBytes(view)` and checks its result before writing. This enforces
   remaining reservation capacity and advances the actual runtime cursor.
4. Overwrites every byte with little-endian integer representations or IEEE
   float bits, using fixed-width stores, and returns the checked pointer.

The same-address copy is semantically harmless. The measured Go 1.26.8 amd64
binary checks pointer equality and skips `memmove`; performance on another
compiler/target must be measured. The helper's checks remain in the full call.

All size products and sums pass `InputSize` before uint32/int narrowing. Existing
runtime cursor arithmetic checks a base-plus-offset overflow, including an empty
argument after the 4 GiB boundary. Each helper receives an individual part size;
the reservation is their prechecked aggregate. Normal `PutString`/`PutBytes`
calls can follow numeric helpers because the actual cursor has advanced. Batch
still requires total used bytes to equal the command's declared input size.

Borrowed views are local to the helper and never stored in command callbacks or
results. Between obtaining the view and completing writes, no reservation,
guest export, memory growth, host re-entry, or fallible encoding validation runs.
Only pointer/length values reach guest invocation. Inputs remain stable during
the existing synchronous operation.

## Errors, ownership, and shutdown

Existing per-call reset, memory checks, reply validation, typed owned decoding,
and immediate fault poisoning remain unchanged. Reserve traps, invalid ranges,
and memory-limit faults pass through the existing checked boundary. No later
command runs after a fault. Expected argument/size errors retain healthy reuse
and the existing ordering relative to stale handles or reservation failures.

No session policy changes: admission, bounded call/byte budgets, context checks,
successful prefixes and `Attempted`, nonpreemptible guest execution, re-entry
restrictions, home affinity, handle epochs, explicit abandoned-resource Close,
and draining library Close retain the approved checked-sessions contract.

## Retention proof

Compiled generator fixtures cover all six types and mixed argument order,
unaligned starts, empty slices, integer extrema and exact float payload bits;
stateless/resource/constructor and checked-command paths; independent wire
goldens; exact/excess aggregate bounds; and validation before reservation.
Forced backing-store replacement, logical length versus spare capacity,
invalid/trapped reservations, memory growth faults, and undersized reservations
verify current memory, no out-of-reservation mutation, poisoning and prefix stop.

The real Wasm proof exercises top-level numeric slices in addition to retained
record codecs. Compare the typed facade with independently encoded original
Wasm inputs and retained outputs across growth/reuse. Sketch tests retain exact
Count/full-protobuf agreement, ordered collapse, malformed tails, late-invalid
atomicity, and recovery. Both original proofs and streaming are rebuilt and
reproduced, with their locks.

Require isolated paired full-facade gains, repeated/reversed confirmation,
matched sizes/corpora and contention, scalar/SQL/trace/streaming guards, fresh
independent spike samples, and all roadmap quality gates. Report absent gains
and regressions; reject material scalar or SQL regressions. Lower Go allocations
do not establish zero guest allocation or RSS bounds. M1 performance remains
unaccepted.
