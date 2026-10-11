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
