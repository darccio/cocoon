# Private bounded numerical feasibility proof

Status: specification signed as `8c8791b` before implementation, 2026-10-12;
the [completed proof and cost observations](performance.md#bounded-numerical-feasibility-certified-intervals-2026-10-12)
meet its numerical rule. Online construction and M1 remain unqualified.
This is an offline
diagnostic for one pinned mapping configuration. It defines no public API,
cache, compiler transformation or changed runtime contract. PR #8 merged at
`aeaf01b341a85024804e9ae1abe43af90d36e4d1` after all four CI checks passed;
local main reproduces the previously validated tree. This specification was
committed after guest, host and adversarial review and exact staged validation
before implementing the ignored proof; that checkpoint is retained in the evidence.

## Measured reason and question

The [exact-log ceiling](performance.md#structural-scalar-ceiling-exact-log-oracle-2026-10-12)
reduces checked fixed-prefix scalar time about 19% against untouched production
and 22% against an identical disabled-oracle ELF in both sets. Its 64 MiB table
is unsupported, outlining regresses, and broad/negative/SQL holdout losses
remain constraints. M1 remains unaccepted. No representative consuming SDK
profile is available; this proof uses the public synthetic Datadog fixture.

The next finite question is whether a bounded offline procedure can certify
useful intervals of distinct binary64 inputs as one exact logical bin. A
certificate must cover every representable input in its range, with uncertainty
returning no certificate. Construction cost is an output, not assumed cheap.
An affirmative proof would justify a separately specified cheap-construction
or lookup experiment; it would not qualify a guest cache or scalar latency.

## Pinned function and domain

Derive the operations, literal bits and bindings from current `fn176`, including
the original log block, two separately rounded affine operations, `math.Floor`
and `i32_trunc_sat_f64_s`. Pin the complete translated module SHA-256
`626550cafa323c490a6a2ec6b2b341656c40e223c7c84701e414078633dcc298`
and unchanged Wasm
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`.
Whole-source/function selectors are diagnostic identities, not generic
production eligibility. Do not substitute host ln/exp, transplant reference
payload code, simplify the polynomial, reassociate or contract operations.

The current constructor publishes these exact words; validate them against
its actual source and output before the proof:

| Field | Binary64 bits | Existing mapping offset |
| --- | --- | --- |
| Gamma | `3ff0400000000000` | +32 |
| Multiplier | `40501fead4ea6e2b` | +40 |
| Minimum indexable input | `0010400000000000` | +48 |
| Index offset | `4094ea0000000000` | +56 |

Gamma binds the configuration/encoding identity; it is not an extra existing
read on the selected numeric path. Threshold, multiplier and offset reads at
+48/+40/+56 remain distinct observable operations for any later integration.

Requests use inclusive unsigned input-bit endpoints `[L,U]`, an exact four-word
tuple and source/version identity. Require ordered positive, normal, finite
binary64 endpoints at or above the minimum. Positive finite bit order equals
numeric order. Below-minimum values take a separate zero-count/no-bin path,
not logical bin zero. Zero, signed/negative input, subnormal, NaN/Inf, reversed
or overflowing ranges and mismatched identities are outside the initial domain.
Reject any interval containing exact 1.0, crossing an original exponent boundary
or crossing the normalization carry. Initial accepted outputs are finite,
nonsaturating signed int32 bins; uncertain saturation also rejects.

## Universal certificate argument

With input high word `H`, the original normal branch forms `A=H+0x95f62`,
`K=(A>>20)-1023`, and reconstructed reduced bits
`(input & 0xffffffff) | (((A & 0xfffff)+0x3fe6a09e)<<32)`.
Prove lack of wrap, constant original exponent and carry tag across each
accepted range; the carry seam is mantissa high word `0x6a09e`. Inside that
cell, the reconstruction is an increasing integer transformation of input
bits into positive finite reduced binary64 bits. Endpoint reduction bounds
follow from that integer relation, not logarithm monotonicity. Exact-one's
shortcut and other branches remain rejected rather than merged into this proof.

Parse the selected normal-path assignments into an immutable expression graph.
Preserve every reassignment, literal's compiled binary64 value and explicit
rounding boundary. Whitelist only the current constants, temporary references,
exact reduction, unary signs and rounded `+`, `-`, `*`, `/`; an unknown binding,
node, conversion, branch, helper or source drift stops the experiment. Every
intermediate interval encloses all values of its actual rounded source node.

For addition/subtraction use the correct extremal endpoints. For multiplication
and division consider all four signed endpoint combinations; reject division
if the denominator range includes zero. Decode finite binary64 endpoints as
exact dyadic rationals, compute exact real extrema, and round outward directly
to representable binary64 bounds. Do not double-round through a finite-precision
big.Float. Include underflow and both zero signs where they are harmless to
later operations; reject any case where lost sign/branch information matters.
Reject nonfinite or unordered enclosures and unsupported overflow. Correlation
may widen bounds and produce uncertainty; it never permits narrowing them.

Induction over the graph proves containment at every rounded operation. Apply
the multiplier and offset with their two original rounding boundaries. Accept
only when the whole final enclosure lies in one interval `[k,k+1)`, with strict
upper inequality and `k` safely inside signed int32. Floor and the original
saturating conversion then produce that same logical bin for every covered
input. An upper bound equal to `k+1`, sampled endpoint agreement, an epsilon or
assumed mathematical ln monotonicity cannot certify membership.

The [Go floating-point rules](https://go.dev/ref/spec#Floating_point_operators)
make explicit conversions rounding barriers. The
[Wasm numeric contract](https://webassembly.github.io/spec/core/exec/numerics.html#floating-point-operations)
specifies nearest-even arithmetic. The proof pins Go 1.26.8's local specification
and compiler/source identities as well as these primary references; current
web documentation alone does not establish historical compiler behavior.

## Bounds fixed before implementation

| Quantity | Maximum and failure behavior |
| --- | --- |
| Normal-path rounded arithmetic nodes | 96, including affine operations; unsupported graph rejects |
| Total graph nodes/live binary64 interval slots | 256; fixed arrays, no recursion or unbounded maps |
| Immutable graph representation | 16 KiB including capacity/padding |
| Retained certificates | 64 |
| Numeric bank/configuration/version metadata | 4 KiB including array capacity, headers and padding on amd64/386 |
| Seed requests | The 64 seeds below, in fixed order |
| Candidate enclosures per seed | Initial request plus at most four refinements |
| All candidate evaluations per complete fixed-bank construction | 320; failed/rejected requests consume the ledger too |
| Exact endpoint operations per rounded node | At most four, with separate conversion counters |
| Directed bound conversions per candidate | At most 192, two per rounded node; independent verification has its own ledger |
| Explicit rounder work per conversion | At most four checked shifts, eight comparisons and one integer quotient/remainder; no search or refinement loop |
| Exact integer numerator/denominator and scaling temporaries | At most 4,096 bits, including unreduced products and rounder/comparison temporaries; conservatively check before an operation could exceed the cap |
| Logical builder workspace including one transfer trace | 64 KiB total; count capacity and reject overflow |
| Certificate output | Fixed endpoints and logical bin; no pointers, handles, store slots or retained guest views |

The exact rational reference uses allocating `math/big`. Logical arrays, limb
lengths, node/attempt counts and serialized trace bounds do not impose a hard
Go-heap/RSS limit. Record constructor allocations, transient heap observations
and retained bytes separately, including rejected work. No bounded-process or
cheap online-constructor claim follows. The independent checker and source
preparation also have their own recorded costs; neither is silently charged
to a smaller timed scope. Any later native proposer needs a separate soundness
argument or must pay the trusted verification cost.

The one trace shares that workspace budget and records node/operand IDs and
binary64 input/output bounds, not a retained history of exact rational
temporaries. The independent checker recomputes those extrema. Graph storage
has its separate 16 KiB cap; trace export and rejected attempts cannot evade
the workspace/attempt ledgers.

Derive the rounding exponent from integer bit lengths and a bounded comparison,
then round the scaled significand directly; do not search adjacent floats.
Each node stores binary64 bounds before the next primitive, so unreduced
arithmetic and shift-width bounds can be justified from finite input encodings.
Count zero/overflow rejection and conversion/comparison temporaries too.
Standard-library big-integer internals remain allocating reference machinery;
the abstract-operation ledger is not a CPU-instruction or wall-time limit.
Keep their source/toolchain identity and allocation/cost observations explicit.

Freeze these seeds before any proof outcome: first the binary64 values
`42, 43, 0.5, 2, 16, 64, 256, 4096, 1000000, 1000001, 1e-12, 1e-11,
1e12, 1e100, 1e-100, 1e300`; then for unbiased exponents
`-1000,-900,-600,-300,-30,-1,0,1,5,10,20,40,100,300,600,1000`, use fraction
bits `0x2000000000000`, `0x8000000000000`, `0xe000000000000` in that order.
Record all seed bits with the source graph before construction.

For each seed bit value `B`, initially request `[B-2^40,B+2^40]` with checked
unsigned arithmetic. On uncertainty, halve the half-width at most four times;
stop on the first certified interval or the fixed request limit. Reject
unsupported branch crossings rather than clipping them after observing results.
No search for a sampled boundary, hand-selected successful seed, capacity
sweep or post-result retuning is allowed. At most one certificate is retained
per seed. Overlapping certificates must agree in logical bin and remain bound
to the same configuration/source; contradictory overlap is a proof failure.

## Independent executable checks

Use three distinct checks. The source-derived exact rational enclosure supplies
the mathematical transfer argument. A separate integer quotient/remainder
nearest-even rounding implementation validates constants and primitive bounds,
including denormals, halfway ties, cancellation and signed zero. The unchanged
compiled source supplies direct point traces; original Wasm fresh singleton
sketches supply independent logical bins and Count/full protobuf observations.
Neither point sampling nor Wasm histories replaces the universal interval proof.

The independent checker recomputes reduction/carry tags and each node's operand
bounds, exact extrema and directed bounds from the pinned graph and `[L,U]`.
It does not trust constructor traces, share the constructor's directed-rounding
routine or treat separate nearest-even rounding alone as a containment proof.
Compare the independently reconstructed trace with every claimed bound.
Validate graph/source correspondence independently, including every signed
corner combination, from earlier verified slots. A verifier that rounds only
the constructor's chosen extrema is insufficient. Source/graph drift is a
fatal experiment failure, distinct from per-request uncertainty for an
unsupported input or exhausted request budget.

Verify every retained certificate's branch and complete transfer trace with the
independent checker. Finite tests include every normal exponent/carry side,
low-word transitions, exact one and neighbors, threshold below/equal/above,
near-largest finite inputs, negative logical bins, affine integer boundaries,
all invalid inputs/configurations and every work/storage rejection. Exhaustively
enumerate small bounded bit intervals and document their actual finite size;
sample structured and deterministic interior bits of wide certificates.
Verification drivers have separate fixed ledgers: at most 16,384 whole-graph
concrete/interval checks, at most 262,144 tiny-span concrete members and at most
2,048 original-Wasm singleton requests per native architecture. These checks
do not construct, select or retune the fixed bank. Report actual counts and
budget failures separately from its 320 candidate evaluations.

For each accepted certificate, compare endpoints, neighboring outside inputs
and deterministic interiors against the unchanged original-Wasm singleton
reader. Fresh resources prevent collapse from hiding incorrect logical bins.
The pure proof does not substitute a caller or mutate guest state; existing
scalar/batch invalid-tail, collapse, memory, ownership, poisoning and shutdown
tests remain unchanged-production controls, not cache integration proof.

Required compiled semantic mutants challenge wrong outward bounds/signed
extrema, altered reduction carry, a bin boundary made inclusive, wrong logical
bin/configuration, truncation in place of negative floor, and acceptance after
uncertainty/budget failure. Supply an actual failing numerical case for each;
compiler/startup errors do not count. Source/constant drift rejection is a
separate explicit category. Run actual native amd64 and 386 proof binaries;
cross compilation alone proves no numerical execution on another architecture.

## Decision, costs and subsequent work

The first decision is numerical and bounded-work feasibility. Require zero
incorrect accepted certificates, all accepted traces independently checked,
deterministic certificates across native amd64/386, every declared bound
enforced and every unsupported case returning uncertainty. Require at least
16 non-singleton certificates across at least four distinct original exponent
fields, including
negative and positive logical bins, and one certificate covering all 8,192
distinct inputs in the existing NearbyUniqueCycle. Record member/real widths,
integer-input coverage and every failure even if this nonvacuity rule fails.
A negative at these limits completes the experiment without expanding them.

After correctness/review, record isolated official Go 1.26.8/CGO=0/PGO-off
constructor costs for the complete fixed bank, individual accepted and rejected
requests, and the independent checker. Source extraction is author preparation;
bank construction and required verification are separately visible setup.
Report allocations and logical/transient storage with operation/attempt counts.
Use seven alternating/reversed samples where comparisons exist; no build,
lint, fuzz, profile or reviewer tools may overlap measurement. Preregister exact
timing scopes/binaries before those costs. This round promises no timing gain
and contains no production scalar promotion rule.

Replay the existing frozen scalar/batch corpora for coverage counters only:
Repeated, UniqueCycle, NearbyUniqueCycle, MostlyUniqueCycle, Phases,
NegativeBins and BroadHoldout. Report first/revisit pass hits, misses/unknown
reasons and each 1,024-input phase. Also report distinct Increasing inputs
`1..4194304`. Use the immutable prebuilt bank, disclose its setup, and label
all results offline-bank locality. No eviction, hidden per-miss certification
or unmeasured on-demand work is implied. Parameter-mismatch replay must have
zero certified hits. Coverage alone does not predict a full-call speedup.

A positive result permits a separately specified bounded fast-construction or
lookup-cost control, followed only if warranted by a separate guest integration
ceiling and source-independent production predicate. Preserve the existing
negative-bin, broad and SQL regressions as future guards. All current parameter
reads/checks, error atomicity, ordered counters/collapse, current memory, reply
validation, admission/ownership/epochs, old-handle invalidation and draining
Close remain per-call obligations. No public contract changes are authorized.
Do not claim native arm64/big-endian execution, private SDK relevance, shipping
cache safety, attainable scalar improvement or M1 acceptance from this proof.

Retain source, graphs, certificates, work/cost samples, independent checks,
mutants, all failures and reviews in durable evidence. Run required gates and
repeat all three proof/runtime builds, preserving committed artifacts/locks.
Use small darna-validated signed local commits and exact staged snapshots;
publication of this new branch still requires the operator's request.
