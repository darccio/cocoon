# Private fast interval construction and mapping costs

Status: specification before implementation, 2026-10-12. Entry is signed
`7c8afee`, after the [bounded numerical proof](certified-interval-feasibility.md).
The operator authorized testing fast construction and lookup/miss costs with
an explicit stopping rule. This defines no public API, production cache,
compiler pass or changed runtime contract. Review and commit this specification
with exact staged validation before implementing the ignored prototype.

## Measured question and scope

The rational reference proves useful intervals, but complete construction
costs 12.257 ms, or 69.607 ms with required verification, allocating 6.35 or
22.95 MB. Its immutable bank hits only 0.043% of Increasing and 0% of Broad.
The unsupported exact-log oracle saves about 19% of checked fixed-prefix scalar
time; it neither qualifies an attainable implementation nor meets M1's target.

Test one native constructor and one fixed one-slot numeric policy. First prove
source correspondence and universal containment, then measure construction,
lookup and complete replay costs, including every miss, rejected proposal,
exact fallback, install and empty-state reset. Individual primitive costs alone
cannot establish amortization. No full checked scalar/SQL/batch/stream gain or
M1 acceptance follows from a pure numeric experiment.

Use only Cocoon's public synthetic Datadog fixture. Keep all production sources,
generated modules, Wasm, manifests and locks unchanged. Preserve the previous
ignored proof and exact rational checker byte for byte; copy their identified
source into the new ignored reference package as necessary. Retain the new
generator together with its diagnostic generated output. No native arm64,
big-endian or representative private SDK claim is permitted.

## Function, preparation and domain

Pin the same complete translated source SHA-256
`626550cafa323c490a6a2ec6b2b341656c40e223c7c84701e414078633dcc298`
and original Wasm
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`.
Retain the original 76-node/30-rounded-operation graph, constants and bindings,
including separate multiplier and offset operations, floor and saturation.
The exact four words remain gamma `3ff0400000000000`, multiplier
`40501fead4ea6e2b`, minimum `0010400000000000`, offset `4094ea0000000000`.

Author preparation parses the pinned source, independently verifies every
graph node/literal/edge, and produces an owned private immutable numeric model.
It must copy the numeric array and configuration; no caller-mutable Graph,
Params, AST, maps, source substrings or guest memory view may be retained.
Compile the numeric graph into straight-line diagnostic Go with explicit
rounding boundaries. The generator, compiled program/node identity and owned
model must agree before construction. Retain original graph metadata in the
model even when the generated program supplies execution.

Moving AST/SHA binding out of each numeric request is allowed only because
that model is privately owned, immutable and source-bound during preparation.
Source drift, invalid/unprepared model or generated-program mismatch is fatal.
Mutating or releasing the original input graph after preparation must not
alter model results. Record preparation cost, allocations and retained bytes
separately. This is no authority to hoist a future live resource's parameter
reads, ownership/epoch checks, current-memory checks or checked ABI work.

Construction receives the exact source/version and four-word configuration
identity with inclusive positive-normal binary64 bit endpoints. Retain the
previous checked ordering, minimum, exact-one, exponent/carry, integer
reconstruction, denominator-zero, nonfinite and nonsaturating strict-bin rules.
Unsupported input/configuration, budget or uncertain containment returns no
certificate; it must never promote a partial/zero-value trace. Below-minimum
input is a separate no-bin path, not logical bin zero. Do not clip seams,
approximate ln/exp, rewrite the polynomial or change source operand order.
An ineligible query's source/version or parameter identity is uncertainty and
zero lookup hits; a changed prepared model, preparation source or generated
program is fatal. These categories must remain distinct.

## Native enclosure argument

For each rounded binary operation, start from finite ordered operand intervals.
Addition/subtraction use the appropriate two extremal endpoints; multiplication
and division evaluate all four signed corners. Division rejects any denominator
range containing zero. Compute each corner as one separately rounded binary64
operation, with explicit conversions preventing contraction or reassociation.
Reject a nonfinite corner; then take the numeric minimum and maximum.

If `r` is nearest-even rounding of a finite exact corner result, its adjacent
representable values toward negative and positive infinity enclose that real
result and its rounded source value. Therefore apply one outward `Nextafter`
to the selected minimum and maximum. Reject a nonfinite resulting endpoint.
This conservative argument includes halfway ties, cancellation, gradual
underflow and both zero signs. Zero sign is unobserved on this selected graph
after reduction; denominator ranges containing zero always reject. If that
premise changes, stop rather than erase an observable sign.

The original integer cell translation supplies reduced-input containment.
Induction over these rounded transfers supplies every node's enclosure; the
original affine boundaries and strict `[k,k+1)` test supply the logical bin.
Native bounds may be wider and reject intervals accepted by the reference.
No loss of coverage permits shrinking bounds or adding epsilon after results.

This is a derived mathematical argument using the
[Go conversion/rounding rules](https://go.dev/ref/spec#Floating_point_operators),
[Wasm numeric semantics](https://webassembly.github.io/spec/core/exec/numerics.html#floating-point-operations)
and [adjacent-value contract](https://pkg.go.dev/math#Nextafter). Pin the exact
official Go 1.26.8 local specification, math/nextafter.go and compiler identities;
current web pages alone do not establish historical implementation behavior.

## Independent verification and fixed bounds

Add a new containment adapter beside the unchanged rational checker. It must
independently bind source/model and prove the initial cell, then recompute each
node's exact rational extrema from previously verified *claimed native* bounds.
Require the native enclosure to contain the independently required directed
bounds, and carry those verified native bounds forward. Comparing only with
the old globally tighter trace or requiring endpoint equality is insufficient.
Verify final strict-bin containment separately. The checker must not call the
native constructor, share its float transfer helper or trust chosen corners.
Its full binding/arithmetic cost remains separately visible; the native
constructor's own soundness argument is required to omit online rational checks.

Retain these hard logical limits before outcomes:

| Quantity | Bound |
| --- | --- |
| Numeric graph/owned model including padding/capacity | 16 KiB |
| Total nodes/interval slots | 256 |
| Rounded operations per candidate | 96 |
| Native corner operations per rounded node | 4 |
| Outward-neighbor steps per candidate | 192, two per rounded node |
| Candidate workspace including transient trace/handoff/report | 64 KiB |
| Retained fixed bank plus encoding/configuration | 4 KiB, at most 64 certificates |
| Fixed-bank candidates | 320, including rejected work |
| Rational checker integer temporaries | 4,096 bits, guarded before possible growth |
| Whole numerical verification evaluations per native architecture | 16,384 |
| Tiny concrete members, also charged to whole ledger | 262,144 |
| Fresh original-Wasm singleton requests per architecture | 2,048 |

Use the same 64 seeds in the same order, initial `[B-2^40,B+2^40]` and at most
four halvings, first successful interval only. Record all seed bits, attempts,
failures and widths; no tuning, capacity sweep or post-result seam clipping.
Explicitly account native operations, outward steps and workspace on success
and rejection. Candidate construction and lookup must allocate zero dynamic
bytes in their warmed numeric scopes; preparation/checker/source compilation
have separate costs. Logical arrays do not establish a hard process/RSS limit.

Test primitive signed corners, ties, underflow, zero, overflow, division through
zero, integer reduction seams, all normal exponent/carry sides, low-word
transitions, threshold, exact one and neighbors, negative bins, strict affine
boundaries, identity mismatches and every bound rejection. Verify every accepted
fixed-bank native trace independently. Compare endpoints/interiors/outside
neighbors to full compiled source and fresh original-Wasm bin/Count/protobufs.
Run actual native amd64 and 386, not cross-compilation alone.

Compiled numerical mutants must expose actual wrong results for inward bounds,
missed signed corner, wrong generated operand/carry, inclusive bin upper bound,
negative truncation, wrong bin/configuration and acceptance after uncertainty
or budget exhaustion. Separate source/model ownership/drift tests are required.
Compiler/startup failures do not count as semantic fault detection.

## Fixed numeric policy and complete costs

The experiment retains one latest certificate, its source/configuration identity
and validity flag. Owned numeric state plus metadata must fit 4 KiB. Start each
complete replay empty. On each default-config input:

1. Check source/configuration and membership; a hit returns the certified bin.
2. On a miss, first check the point domain. Zero, signed/negative, subnormal,
   nonfinite, below-minimum and exact-one inputs perform zero proposals and
   take the exact reference path. This fixed domain check is timed and counted.
   Otherwise request an interval centered on that input with half-width `2^40`,
   then at most four halvings on uncertainty. Rejections consume their budgets.
3. On first success, replace the one slot and return its bin for the center.
4. If all five fail or the point is unsupported, return the unchanged full
   compiled OriginalBin result. No failed proposal installs state.
   Failed or unsupported construction retains the previous valid slot unchanged;
   only a successful replacement or the replay's initial reset changes it.

Every eligible miss retries every uncertainty category up to the same five
requests, including cell/domain/boundary rejection of the requested interval.
Fatal model/source failure stops the replay; it is not a numerical fallback.
Exact 1.0 at the start of Increasing is thus one timed domain bypass and zero
proposals. Unsupported-point coverage tests respect the separate no-bin
semantics; timed default corpora exercise the selected mapping's eligible inputs
and exact-one shortcut. No different-config fallback is claimed.

Freeze this policy before results. There is no locality detector, backoff,
corpus-specific bypass, learned width, alternative capacity, boundary search
or eviction experiment. Parameter/source mismatch must produce zero hits;
functional mismatches test lookup/constructor rather than incorrectly applying
the default reference mapping to a different configuration. Timed corpora use
the verified default tuple and finite positive inputs of the existing fixtures.

The on-demand replay has a distinct ledger from the 320-candidate fixed bank:
at most five proposals per input, at most `5*N` over a replay of `N` inputs.
`N` is exactly 4,194,304 in screening and 8,388,608 in confirmation, so maximum
total proposals are 20,971,520 and 41,943,040. Record actual hits, misses,
each refinement/rejection, installs, fallbacks, resets and digest; checked
uint64 counters must reconcile. This finite replay is no session API or global
production cache-admission bound. The separate finite proof-driver limits do
not include these declared full-N mapping replays. Before timing, run untimed
lockstep enabled/disabled/direct checks against the exact compiled reference
for every input of every full-N replay and record their own actual counts and
digests. Timed loops retain a rolling output digest and compare it after the
timer with the prevalidated expected digest; they perform no extra reference
calculation for a hit. Digests establish replay consistency, not the universal
membership proof. Finite fresh-Wasm checks supply the separate stateful oracle.

Measure native/reference individual accepted, early-cell-rejected and evaluated
boundary-rejected requests, complete fixed-bank construction, required checker
cost, one-slot hit/miss/mismatch and fixed-bank first/last-hit/miss lookup costs.
Then use three complete mapping roles: direct original, disabled wrapper,
enabled wrapper. Disabled/enabled must use the same byte-identical ELF and owned
model. Disabled performs exact fallback without proposal/install; it retains
declared setup/state. Direct original is the untouched numeric reference with
the same input loop. Preserve any disabled-wrapper overhead versus direct.

Replay Increasing from 1 through N without a prior filled state. This is an
empty-slot replay with a prepared model, not a cold-start measurement. Replay all
seven frozen 8,192-value corpora cyclically for the same N, including Broad,
Negative, MostlyUnique and alternating Phases; reset for each complete replay.
Include all lookup, proposal/refinement, exact fallback, install and reset costs
in the timer. Source/model preparation and driver input allocation are outside
and separately recorded for every role. No subtraction of miss/setup cost,
condition-selected warm-up or qualified sample dropping is permitted.

Use official Go 1.26.8, CGO=0, GOAMD64=v1, PGO off and actual P1. Freeze all
sources, binaries, toolchains, numeric policies, scope/iteration counts and
exact commands after applied reviews before timing. Seven fresh-process samples
per scope per set; rotate/reverse orders. Retain every raw metric/condition and
failure. No build, lint, fuzz, profile, memory observation or reviewer tools
overlap timing. Keep process provenance to PID/PPID/command names. Observe
preparation, transient heap and retained storage separately and label their
scope; no hidden trusted-verifier or graph-compilation work is free setup.

## Decision and stopping rule

Numerical feasibility requires zero incorrect accepted certificates, all
accepted fixed-bank and declared finite proof-driver traces independently
verified, deterministic native amd64/386 results,
all declared bounds enforced and zero allocations in warmed candidate/lookup
scopes. Retain the previous nonvacuity requirement: at least 16 non-singleton
certificates across four original exponent fields, both bin signs, and one
covering all 8,192 nearby inputs. Failure completes a negative at these limits.

Advance this fixed candidate only if complete Increasing replay reduces median
ns/input by at least 10% against both direct and same-ELF disabled controls,
with exploratory unadjusted p<.05 in both 4M/8M sets. For every other corpus,
reject a slowdown over 2% versus either control detected at p<.05 in both sets.
Broad and negative-bin guards are mandatory; repeated/nearby success cannot
override them. Report all comparisons and multiple-testing limitations; this
is a private diagnostic advancement rule, not M1 or full-call acceptance.
Report smaller or single-set losses too, even when they do not trigger the
predeclared two-set 2% rejection rule. Report measured reuse break-even from
constructor/hit/original costs as an explanatory estimate, not an additive
prediction or a replacement for the complete replay.

Any incorrect certificate is an immediate numerical no-go. Any failed numeric,
allocation or economic criterion parks this candidate unchanged: no width,
capacity, retry, bypass or corpus policy retuning in this round. Retain the
negative results and identify whether construction, lookup/miss overhead or
poor locality explains the failure. A later materially different candidate
requires separate evidence and a new reviewed specification.

A positive permits only a separately specified guest integration ceiling and
eligibility predicate independent of these whole-source diagnostic identities.
All live parameter reads, counter/collapse ordering, error atomicity, admission,
ownership, epochs/old-handle invalidation, per-operation memory/call-state/reply
checks, immediate poisoning and draining Close remain future per-call duties.
No new public contract is approved, and M1 remains unaccepted.

Retain durable proof/generator/cost/review/failure evidence. Run the seven
required gates (`check`, `race`, `rust`, `cross`, `smoke`, `fuzz`, `integration`)
on unchanged production, and repeat all
three proof/runtime builds with identical artifacts/locks. Use darna, exact
staged validation and signed local atomic commits; do not publish the branch.
