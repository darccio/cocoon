# Translation memory views: phase 0

The 2026-10-11 analysis identifies a promising private compiler optimization:
reuse a linear-memory slice header within a short region whose effects cannot
change it. Two sketch result-publication regions reproduce the broad probe's
batch benefit. This is a measured diagnostic, not an implemented hardener pass
or a new public contract. The standalone API and ABI v3 remain unchanged.

Measurements, negative results, conditions, and raw reproduction material are in
[performance.md](performance.md#translation-analysis-positive-phase-0-2026-10-11)
and [the archive](measurements/translation-phase0.txt).

## Demonstrated pattern

Go 1.26.8 inlines wasm2go's checked scalar memory helpers. Their unsafe backing
writes nevertheless invalidate the compiler's knowledge of `Module.memory`:
subsequent accesses reload its pointer and logical length. The current sketch
function, `fn176`, has two terminal result-publication regions where a local
slice view eliminates four such reloads per successful completion. Every guest
load, store, address, integer conversion, bounds check, and stack restoration
remains in place. The logarithm and bin calculation are untouched.

This preserves live backing bytes, rather than making Wasm stack values private.
The `fn84`/`fn176` result addresses cross calls; recognizable stack-pointer
arithmetic alone does not establish nonescape or disjointness. Stores must
remain observable in exported linear memory, including after a later trap.

The diagnostic requires the exact full-module hash and ordinary generated-facade
construction. It does not establish a generic alias or effect analysis. Its
function-number and source-line filters select attribution controls only; they
must not become production selection rules.

## Requirements for the production experiment

Implement recognition in the translation/hardening pipeline, with a conservative
region predicate derived from the demonstrated publication pattern. Define that
predicate before inspecting new holdout results. Regenerate all proof modules
and locks from authored sources; never retain an independently edited module.

The pass must validate the actual bound scalar helper declarations and bodies,
numeric types, receiver, and memory field. Keep existing helper-drift rejection;
leave regions with unproved bindings, effects, or address forms unchanged rather
than rejecting a previously supported module merely because it is ineligible.
Cover helper/import/builtin shadowing and reserved-name collisions explicitly.
For unsafe accesses, prove width-end additions cannot overflow; a spelling match
for `load32` or `uint64` is insufficient. Use logical length, not capacity.

Keep the capture at the first eligible memory access. Preserve operand evaluation
counts and order, integer widths, explicit float rounding, and all guest memory
effects. A local header does not authorize forwarding values, deleting stores,
coalescing accesses, moving bounds checks, or changing math.

Treat unproved direct/indirect calls, imports, bulk operations, memory growth,
header or receiver mutation, pointer/slice escape, and control-flow boundaries
as barriers. Reload after a barrier. Labels, gotos, loops, and joins must not
enter a region after its capture. Predeclare locals where Go's declaration rules
require it; assign the view at region entry rather than hoisting the snapshot.

Establish the memory-ownership precondition explicitly. Generated construction
allocates the Go Module and its byte backing separately, so ordinary guest stores
cannot mutate the Go slice header. Imports can replace the exposed memory header
and must remain barriers. The current experiment does not cover a trusted unsafe
raw-module host fabricating a backing slice that aliases the Module object, or
concurrent raw-module mutation. Do not silently generalize its proof to those
configurations.

## Production experiment specification (2026-10-11)

This private experiment is authorized by the operator's continuation of the
analysis round. It changes no API, ABI, manifest option, ownership rule, or host
contract. Commit this predicate before implementation or new holdout timing.
The following rules select regions structurally; function numbers, source lines,
payload names, and workload results are never selection inputs.

Select a short publication tail inside one lexical Go block:

1. The tail ends in exactly two adjacent validated `store32` calls, followed by
   an assignment restoring a declared receiver `int32` field from a local i32
   plus `i32(nonnegative constant)`. The two stores address the same local i32
   base at offsets zero and four, in either original order.
2. The restoration immediately precedes a goto, a return without effects, or
   the end of the function's outer body. Nested-block fallthrough alone does
   not qualify. This is a structural anchor, not a proof that stack bytes are
   private or that result pointers do not alias.
3. Immediately preceding typed local i32 copies and at most one validated
   `int32(load32(...))` assignment may join the tail, up to four statements.
   No other expression or statement joins it. Require only live-byte accesses;
   do not cache loaded values, combine stores, or infer pointer disjointness.
4. Labels and all other control-flow nodes end the prefix. A label's attached
   statement stays unchanged. Capture after it, immediately before the first
   eligible memory access; never above a label or at function entry. No label
   can target the middle of the resulting straight-line region. A later goto
   successor receives a fresh capture if its own tail qualifies.

Validate binding objects as well as syntax. The bound scalar `load32`, `store32`,
and identity `i32` declarations must match the pinned translator's complete
signatures and bodies. Check their standard `encoding/binary`, `math/bits`,
`unsafe`, and `runtime` imports and architecture constants, plus unshadowed
builtin types and conversions. Resolve receiver/field/local types to the actual
declarations; accept builtin i32 locals, not spelling-equivalent named types.
Scalar drift or unsupported bindings leave the module/region unchanged; existing
bulk-helper drift errors remain errors.

Every eligible load/store's first argument must be exactly the validated memory
field selector on the current bound receiver object. Other buffers, slices,
aliases, fields, receivers, and shadowed receiver names are ineligible. Returns
may be bare or contain only bound local i32 values or integer literals; calls,
field/memory reads, indexing, division/modulo, and other trapping expressions
are ineligible. The restoration follows the view's final use: it is the allowed
tail anchor, not permission to reuse a header after a receiver-field write.
Validate helper compiler directives too: known `go:nosplit` is acceptable;
unexpected binding-affecting directives, including `go:linkname`, are ineligible.

Addresses are exactly `uint32(local_i32)` or
`uint64(uint32(local_i32)) + nonnegative u32 constant`. Preserve the expression
and width as written. The address is at most twice `MaxUint32`, so the helper's
end addition by three cannot overflow uint64. Do not normalize widened offsets
into wrapping i32 arithmetic. Payloads are builtin `uint32(local_i32)` or
`uint32(i32(constant))`. Calls with side-effecting arguments are ineligible.

The Module memory field must be builtin `[]byte`; validate the constructor's
separate `new(Module)` and `make([]byte, ...)` allocations and reject imported
or otherwise unproved memory construction from eligibility. The checked facade
must receive that separately allocated object: merely finding allocation calls
is insufficient. Validate the returned receiver and every generated header
construction path; receiver rebinding or a later unproved unsafe backing alias
makes the module ineligible. Normal owned growth and trusted import-side header
replacement remain allowed only under the following precondition.
The checked facade serializes guest execution. Trusted native hosts must obey
the existing memory ownership precondition: they may replace the header at an
import barrier, but may not fabricate backing storage inside the Go Module
object or race mutation. This pass does not claim to support such fabricated
unsafe aliasing.

Predeclare fresh builtin `[]byte` locals at method entry, assigning each only
at its region entry. Existing name/type collisions leave that method unchanged.
Replace only the original calls' first memory argument; keep all operands,
statement order, guest loads/stores, bounds/traps, and restoration assignments.
Views have method-local lifetime and cannot cross any unproved call, import,
growth, header/receiver write, pointer escape, bulk operation, or control-flow
boundary. Bound validated scalar helper calls are eligible region operations.
No runtime admission, reset, memory, reply, poison, or drain check changes.

Qualification uses freshly built matching before/after harnesses, a freshly
rebuilt independent spike, all broad phase-0 workloads and holdouts, and the
unchanged streaming controls. Run seven interleaved 500 ms samples and seven
reversed one-second samples, isolated with Go 1.26.8, CGO disabled, PGO off.
Preserve every sample; p-values are exploratory and unadjusted. Require repeatable
end-to-end batch benefit without a repeated material guard regression (3% or
more); review smaller detected negatives too. No scalar improvement or M1
acceptance follows from batch throughput. Reject the production pass if proof,
code size, maintenance, or cross-workload effects outweigh the measured benefit;
retain useful independent proofs and complete rejection evidence separately.
The 3% hard rejection threshold does not waive smaller repeated standalone
scalar regressions; the existing standalone guard remains applicable.

## Executable proof and qualification

Add a translated WAT fixture with independent wazero comparisons of returned
values, complete relevant memory, trap categories, and import events. Cover:

- overlapping pointers and partial-width stores, including subsequent reads;
- end-of-memory accesses and wrapping addresses;
- a trap after successful writes, preserving their bytes and observation order;
- calls/imports changing backing bytes;
- growth with spare capacity and growth that reallocates backing storage;
- loops, conditional entry, labels/gotos, and fresh captures after barriers;
- drift and shadowing rejection, plus deliberately broken transforms that the
  executable proofs detect.

Use separate generated Go tests for imports replacing the exposed native slice
header and for logical length below capacity. Arbitrary native header replacement
has no direct Wasm equivalent; WAT/wazero supplies the byte-effect and growth
oracle, while these host-specific cases require direct Go assertions.

Existing generated differential fuzzing is useful but does not cover these
effect histories by itself. Keep per-operation runtime memory checks, reset,
canonical reply validation, immediate poisoning, ownership, and draining Close.
This optimization does not amortize or bypass them.

Freeze freshly generated before/after binaries with identical harnesses and a
fresh independent reference. Repeat all four original workloads, both varied
workloads, negative bins, SQL/large-batch holdouts, and streaming controls. Report
frames, spills, code size, allocations, actual processors, and uncertainty as
well as throughput. Qualify available native architectures and every required
gate; cross compilation alone is not native performance evidence.

The phase 0 gain is limited to the measured original and repeated-value batches.
The broad batch holdout has no detected gain. The broader screen establishes no
scalar speedup; the focused control's small scalar improvement is insufficient
for a standalone latency claim. A production result needs its own qualification.
M1 performance remains unaccepted.
