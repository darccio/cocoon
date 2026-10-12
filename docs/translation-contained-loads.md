# Contained load bounds

The 2026-10-12 phase-0 probe removes a later `load32` bounds check only when a
successful earlier read in the same straight-line region already proves its
entire byte range. SQL repeats a 3–5% improvement; batch benefits are smaller.
Standalone scalar and trace gains are not established. The probe is private,
hash-pinned diagnostic code, not a production transform. Its complete evidence
is linked from [performance.md](performance.md#contained-load-bounds-phase-0-2026-10-12).
M1 performance remains unaccepted.

## Scope and demonstrated pattern

The original sketch body reads one local i32 base at offsets four, eight, then
zero. Keep the first read's endpoint check at `base + 7` and the larger second
read's check at `base + 11`, at their original evaluation positions. Only the
last read's `base + 3` check is redundant. A larger check must never be moved
ahead of an earlier access: that would change the first trap and observable
prefix. SQL contains analogous reads at offsets forty followed by thirty-two.

The diagnostic selects 319 contained reads across 69 methods of the exact
current translated module. It compares the original helper, a separately named
copy with its unsafe-branch check removed, and a separately named copy retaining
that check. The recheck control has the same selected calls. Function numbers,
line numbers, payload identities, source hashes, and benchmark outcomes are
diagnostic provenance only; they must not select production candidates.

This specification precedes production implementation. It adds no API, ABI,
manifest option, ownership rule, or host contract. Per-call admission, memory
limits/reset, canonical reply validation, immediate poisoning, ownership/epochs,
and draining Close all remain unchanged. The existing publication-view predicate
is not broadened by this experiment.

## Production predicate

Recognize bound declarations and types, rather than spellings. Require validated
scalar helpers, imports, architecture constants, Module memory field, receiver,
and separately allocated construction; reuse the hardener's existing checks
for these mandatory guards.
Validate the complete `load32` signature/body and compiler directives. Preserve
existing bulk-helper drift errors. Unsupported scalar shapes, bindings,
construction paths, or effects leave the module or region unchanged.

Inspect only ordinary methods on the actual bound `*Module`. Skip nested function
literals. Each lexical block begins with no facts; control-flow joins, nested
blocks, loops, branches, labels, gotos, returns, and declarations end the current
region. A proved single scalar assignment using `:=` is the exception to the
declaration barrier described below. No fact flows into a nested block or across
labelled entry. Conservative
barriers may forgo opportunities; no general control-flow analysis is intended.

An eligible read is the sole RHS of a single local scalar assignment, optionally
wrapped in the unshadowed builtin `int32` conversion. Its bound helper's first
argument is exactly the current receiver's validated `memory` field. Other slices,
aliases, receivers, fields, or captured publication views do not qualify.
The destination and address base must be actual function-local bindings or
parameters. Require an explicitly declared builtin `int32` address base. Prove
the destination's builtin `int32`/`uint32` type from its declaration, allowing a
short declaration whose type follows directly from the validated load or builtin
conversion. Globals and spelling-equivalent named types are ineligible. Exclude
blank destinations and unproved declarations.

Accept only these addresses, preserving their original expressions:

- builtin `uint32(local_i32)`;
- builtin `uint64(uint32(local_i32)) + nonnegative_u32_literal`.

The base is zero-extended, not an arbitrary wrapping i32 expression. The largest
possible inclusive endpoint is `2 * MaxUint32 + 3`, which fits uint64. Do not
normalize the widened addition into wrapping arithmetic. Track facts by the
bound local's declaration identity together with its bound name and maximum
successful offset endpoint. Grouped parameters or variables share a declaration
node but are distinct bindings; they must never share facts or invalidation.
For a later eligible read on the same unchanged base, remove its check only
when `later_offset + 3 <= previous_maximum_endpoint`. Keep the first check and
every new maximum, at their original positions. A successful endpoint check
against logical slice length proves the lower nonnegative range as well.

Only proved scalar local assignments may preserve facts between reads. Validate
all numeric types and builtin conversions, local bindings, and any identity
wrapper's complete bound declaration and directives. An impure or shadowed
`i32`/`i64` is a barrier. Reject pointers, indexing, slices, selectors, closure
creation, address-taking, and unproved calls. After evaluating an assignment's
RHS, invalidate facts for its destination: reads observe the old base before
the assignment changes it. Multi-assignments, compound assignments, inc/dec,
receiver/header writes, stores, imports, growth, bulk operations, defer/go, and
unproved statements clear all facts. Do not forward loaded values, remove guest
reads, cache the header, or infer disjointness of memory addresses.

Prove base nonescape for the entire method before creating facts. Exclude bases
whose address is taken or which a function literal captures, even before the
first checked read. A callback could retain that binding and change it later;
clearing facts at the earlier escape is insufficient. Methods containing a
`go` statement are ineligible. This is a conservative eligibility restriction,
not concurrency support or a change to the trusted host ownership precondition.

Use a collision-free private helper only for proved later reads. Its reserved
name anywhere in the input, including local scopes, leaves this pass inactive
without a new build error. Derive its body
from the validated helper, deleting only the unsafe branch's endpoint check.
Keep the original endian conversion, unsafe read, address widths, nosplit
directive, and checked unaligned fallback. Append no helper when no call qualifies.
Preserve original operands, evaluation counts/order, and guest memory reads.
The private helper never escapes or becomes part of the public interface.

The checked facade serializes execution and owns separately allocated backing
bytes and Module header. Trusted native hosts may replace the exposed header at
an import barrier, but cannot race mutation or fabricate backing storage inside
the Module object. This is the existing ownership precondition, not a new host
permission. A native host's slice length can be shorter than capacity: earlier
checks and all renewed checks must use logical length. Mutation through imports
must end facts even when the new allocation or spare capacity keeps bytes readable.

## Proof and qualification

Add an authored WAT fixture translated with the pinned toolchain, using wazero
as the independent oracle. Compare values, full relevant memory, traps, and
ordered import effects before/after. Cover contained and rising ranges,
unaligned addresses, short memory and exact endpoints, high-bit and wrapping
addresses, overwritten bases, stores, imports, growth, nested/conditional entry,
labels/gotos, local/global shadowing, drift, and reserved-name collisions.

Native tests must additionally exercise length below capacity, import-side
header shrinking/replacement, and changed backing bytes. Keep a checked fallback
control. Deliberately broken first-check deletion, crossing an import barrier,
and name-only purity must fail executable semantic tests. A static selection
assertion alone is insufficient. Test available native amd64 and 386; arm64
cross compilation does not establish execution or performance on arm64.

Implement only in authored hardener source, with tests in the same atomic commit.
Regenerate compute, Datadog, and streaming proof modules and locks together.
Review guest, host, and adversarial concerns before retaining each change; test
the staged snapshot and validate with `darna --committable --dependants`.

Freeze freshly built production before/after binaries with matching harnesses and
a freshly rebuilt independent reference. Repeat all four original workloads,
varied scalar/SQL, negative bins, SQL and broad scalar/batch holdouts, raw guest
control, and streaming controls. Use isolated Go 1.26.8, CGO disabled, PGO off,
seven interleaved 500 ms samples followed by seven reversed one-second samples.
Preserve every raw sample, custom metric, and negative result. P-values are
exploratory and unadjusted; nondetection does not establish equivalence.

Require a repeatable end-to-end SQL benefit without repeated material guard
regressions of 3% or more; review smaller detected negatives too. This threshold
does not waive a smaller repeated standalone scalar regression. Record code
size, frames, register-allocation/placement effects, allocations, actual processor
counts, and uncertainty. The diagnostic's sketch frame changes from 144 to
128 bytes, so timing is not a clean per-check causal estimate. Reject the pass
if proof, maintenance, size, or workload effects outweigh the measured benefit.

All required gates (`check`, `race`, `rust`, `cross`, `smoke`, `fuzz`), integration,
and two identical proof/runtime rebuilds must pass after the final implementation.
Production qualification is separate from phase 0. Any throughput improvement
does not establish standalone scalar speedup or M1 acceptance.
