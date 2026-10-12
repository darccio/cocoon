# Cocoon roadmap

This is the resume point for future development conversations. Updated
2026-10-12. The synchronous M1 framework and Datadog proof were released privately
as v0.1.0. The quality pass and five performance passes are complete, including
the nine-round multi-agent loop, the PGO/resource-call follow-up, and exact
numeric-lowering investigation.
Performance acceptance remains unmet.
Async and HTTP are a separate milestone, not partially implemented M1 features.

## Current working state

The repository now uses ordinary `.git` metadata. The sandbox workaround
`.cocoon-git` has been migrated, with all history preserved. No special Git
environment variables are required.

[PR #7](https://github.com/darccio/cocoon/pull/7) merged at `6be3980` after
licenses, native amd64/arm64 quality and end-to-end CI passed. Local `main`
fast-forwarded to `origin/main` before the next experiment. Its conservative
contained-load pass remains qualified; its adjacent read/write scalar probe
remains a no-go. The subsequent checked unit-reply phase-0 round also retains
evidence only, as recorded under track 4. The subsequent covered store32
phase-0 experiment also retains evidence only: its scalar gain appears in
confirmation but fails the predeclared repeated untouched comparison. The
subsequent exact-log oracle establishes a larger scalar ceiling, about 19% in
both fixed-prefix sets, without establishing a bounded implementation. Pure
outlining and fallback-heavy inputs regress. The next step is a private
certified-interval feasibility proof; M1 performance remains unaccepted.

Go tests, vet, race checks, 49 strict linters, native Rust tests and Clippy,
Wasm verification, differential fuzzing, and cross-platform test compilation
have passed locally. Rebuilding both original proof packages and the streaming
fixture reproduces their committed artifacts and lock hashes. The first private
release was
[github.com/darccio/cocoon v0.1.0](https://github.com/darccio/cocoon/releases/tag/v0.1.0).
All three [release qualification CI jobs](https://github.com/darccio/cocoon/actions/runs/37235658925)
passed: native amd64 and arm64 Go quality gates and the fresh pinned end-to-end
build. The release tag points to that qualified code commit. The repository is
now licensed under Apache-2.0, with a reproducible dependency license inventory
and CI coverage checks. The original v0.1.0 tag predates those licensing changes
and is retracted.
[v0.2.0](https://github.com/darccio/cocoon/releases/tag/v0.2.0) publishes the
module as `dario.cat/cocoon`; the GitHub repository remains the source host.

Native linux/386 tests pass outside the former sandbox. Before the checked-command
work, recorded runtime coverage was 97.8 percent; the Rust and Go generators
were 94.6 and 96.7 percent.

The installed-CLI smoke test passes with Go 1.26.8 and 1.27.1. It initializes,
formats, builds, verifies, and deterministically rebuilds a separate author's
shim, then runs a Go-only consumer against stripped production sources with
module fetching disabled and Rust/Wasm tools unavailable. The consumer also
exercises the checked-in Datadog package. Authenticated installation and a
separate consumer of the published private module at `v0.1.0` also pass without
local replacements on both Go versions. The tagged Go-only consumer passes with
fetching disabled and Rust/Wasm tools unavailable. Run the checkout smoke test
with `make smoke`.
The smoke test now also relocates both the author module and guest support
crate into paths containing spaces and checks identical artifacts and locks.

CI exposed an omitted slice upper bound in hardening, embedded absolute Rust
source paths, and path-dependent Cargo crate metadata for dependencies outside
the shim workspace. All are fixed and regression-tested. CI uses the
SHA-verified official Binaryen bundle; locks fingerprint its executables and
the compiler metadata normalizer. The normalizer's source hash invalidates
Cargo's cache when its implementation changes. Both proof artifacts and locks
match across relocated checkouts and the clean CI checkout, including repeated
rebuilds.

## Performance work

The user resumed optimization on 2026-10-05 after the first complete MVP and
follow-up quality pass. Performance acceptance remains unmet; it was not a gate
for v0.1.0.

Fresh release-artifact measurements and historical results are recorded in
[performance notes](performance.md), including raw samples and build
provenance. The release baseline is 2.225 microseconds for SQL and 113.5
nanoseconds for SketchAdd. Diagnostic benchmarks now separate guest execution,
checked instance calls, resource ownership, facade calls, and empty runtime
callbacks. Separate CPU profiles attribute 53.9 percent of SketchAdd and
79.2 percent of SQL samples cumulatively to their translated exports.

The first retained experiment (`e099514`) forces Rust to inline canonical unit
replies. Paired SketchAdd medians improve from 113.8 to 98.53 nanoseconds; SQL remains
roughly unchanged. Both proof artifacts and locks were regenerated, and a new
differential regression exercises empty replies after nonempty data and errors.
Bounds, reply validation, fault containment, shared ownership, and draining
Close are unchanged. Both original acceptance limits remain unmet:
2.2154 microseconds and 55.44 nanoseconds.

The second retained experiment (`14e056c`) avoids general status dispatch only
after fully checking a successful reply. Seven alternating one-second sample
pairs improve SketchAdd by a further 1.75 percent without a detected SQL change.
New runtime regressions reject unknown and negative status codes and zero-length
success replies beyond logical memory. Cold error constructors and forced
resource lookup inlining did not establish benefits and were removed; their
measurements remain in the performance notes.

The broader comparison exposed a 2.6 percent SQL slowdown from the combined
changes. A focused cold helper for unit-error publication (`078c9c9`) recovered
that loss while preserving the scalar gain. Native tests now exercise all unit-error
statuses and transitions back to canonical empty success replies. The weaker
inline hint alone produced identical Datadog Wasm and was not retained.

The first-pass final comparison is 114.4 to 96.45 nanoseconds for SketchAdd,
a 15.69 percent reduction. SQL is statistically unchanged at 2.272 microseconds.
Remaining same-machine reference overhead is 16.0 percent for SQL and
73.8 percent for SketchAdd; original and same-machine 10-percent targets are
still unmet. Batch sketch and trace comparisons show no detected timing or
allocation regressions. All local quality, staged-snapshot, repeat-build, smoke,
and generated differential fuzz checks pass for the final code, including
Go 1.26.8 and native linux/386 tests. Runtime coverage remained 95.6 percent
at that point.

The second pass retained four host-side changes: successful callback error
matching (`c7e148e`), checked range inlining (`469e865`), direct resource lifecycle
admission (`aec4d6d`), and ready pool borrowing (`b30d62e`). CPU-pinned screening
confirmed small scalar gains and a 34 percent isolated ready-pool improvement;
those separate percentages are not additive. Existing panic and GC ownership
regressions were also expanded (`0c653b3`). The generated facades and locks were
updated without changing either Wasm artifact or translated guest source.

The fresh seven-sample unrestricted comparison is 80.00 to 77.31 nanoseconds
for SketchAdd, a 3.36 percent gain. SQL is 1.880 versus 1.839 microseconds with
no detected timing change. Remaining reference overhead is 11.3 percent for
SQL and 66.8 percent for SketchAdd. The unchanged baseline also runs faster
in this session than previously; do not present historical-to-current absolute
differences as code improvements. Both fresh same-machine 10-percent targets
remain unmet. SQL is below its original absolute limit in this run, while the
scalar benchmark still exceeds its original limit. Batch and trace comparisons
show no detected timing or allocation-count regressions.

Owned reply publication regressed SQL and traces and was discarded. Combining
resource exit defers did not establish a gain and was discarded independently;
its regressions remain against the original implementation. All full local
quality gates, differential fuzzing, repeated proof builds, staged snapshots,
Go 1.26.8 tests, and native linux/386 tests pass. Runtime coverage is 97.0 percent.
Raw samples, screening conditions, profiles, and rejected experiments are
recorded in the performance notes.

### Third performance loop

The 2026-10-05 and 2026-10-06 loop used separate guest, host, and adversarial
review agents for each round. Every retained commit passes strict lint, darna,
and exact staged-snapshot Go tests. The retained code and tests also pass full
integration, race, cross-platform compilation, external CLI and Go-only smoke,
differential fuzzing, Go 1.26.8, native linux/386, and repeat proof builds.
Runtime coverage remains 97.0 percent.

Callback-state reset did not establish a gain. Specialized successful unit
replies and single-slot guest lookup improved scalar calls but regressed other
workloads and were removed. Binaryen `-O4`, `-O3 --no-stack-ir`, and
`-O3 --converge` did not justify changing the original optimization policy.
The new callback, handle-retirement, and explicit optimizer numeric/trap/evaluation
contracts were retained independently. `make smoke` now exercises the optimizer
contracts in CI, independently of comparing Go with the same optimized Wasm.

The ordered point predicate (`0dd1701`) was withdrawn in `db580ca` after the
full screen and longer three-way comparison found a SQL slowdown without an
established scalar gain. Its native/public numeric-boundary and atomic-error-state
regressions remain. A subsequent exact unsigned-bit classifier improved batches
2.85 percent but slowed SQL 2.55 percent versus fix-only and was removed too.
No performance optimization survived the nine rounds. The destructor correctness
fix remains, with no detected timing change relative to the matched loop-start
binary. Both builds contain identical Datadog Go tests.

The longer final comparison puts retained SQL at 1.767 microseconds and
SketchAdd at 74.33 nanoseconds, versus a fresh reference at 1.617 microseconds
and 46.65 nanoseconds. Overhead is 9.3 and 59.3 percent respectively. SQL's
median falls below this run's 10-percent limit with a narrow margin; scalar
exceeds both original and fresh limits. Overall acceptance is still open.
Do not attribute differences from older sessions to code changes. Raw samples,
candidate hashes, rejection evidence, and final provenance are in
[performance notes](performance.md). This loop's commits have not been pushed.

### Fourth performance pass

The 2026-10-06 follow-up used guest, host, and adversarial review agents for
PGO and fused resource calls. Each implementation trained its own equal-time
four-class CPU profile with PGO disabled. New varied scalar and SQL benchmarks
were excluded from training (`2d95490`). The initial unrestricted screen did
not establish a scalar gain. A seven-sample, one-second CPU 14 confirmation
found PGO scalar improvement of 5.65 percent and held-out scalar improvement
of 4.81 percent. The final seven-sample, one-second unrestricted confirmation
replicates scalar gains of 5.36 percent (77.37 to 73.22 ns) and 4.98 percent
on held-out values. SQL improves 4.23 percent and traces 2.78 percent, but batches
regress 1.26 percent. PGO stays opt-in because of that trade-off and the synthetic
training mix; no `default.pgo` or automatic library-wide PGO was installed.
These measurements do not establish a production workload speedup.

Same-run own-PGO reference overhead is 12.3 percent for SQL and 55.0 percent
for scalar, versus 11.4 and 62.8 percent without PGO. Both fresh same-machine
10-percent targets remain unmet. SQL remains below its original absolute limit,
while scalar still exceeds 55.44 ns. Do not attribute differences from the
preceding loop's reference figures to a source change.

The fused checked resource-call experiment retained all synchronization and
safety checks but did not establish its intended scalar benefit in either
screen or confirmation. It was removed, including the proposed `Resource.Call`
and `Call.Handle` APIs. Production, Wasm, translated modules, facades, and locks
remain unchanged. The restored benchmark binaries are byte-identical to the
frozen pre-experiment builds, both with and without PGO.

Useful ownership/nested-execution regressions remain in `f2d0158`, and the
compiled generated-facade terminal admission/drain regression in `85b55d0`.
Runtime coverage rises from 97.0 to 97.8 percent. Restored code passes strict
check, integration, cross compilation, external/relocation/Go-only smoke,
five fuzz targets, Go 1.26.8 PGO-enabled tests and race checks, and native
linux/386 targeted tests. A real external consumer runs with PGO off/on.
Retained test commits pass darna and exact staged-snapshot tests. These and
the preceding loop's commits remain local and unpushed.

A cached Go-built `pprof` frontend was recovered; the tooling blocker is
resolved. The matching retained scalar profile attributes 58.29 percent
cumulatively to guest execution, 43.58 percent to sketch math, and 10.16 percent
to reply validation. These categories overlap and must not be added. The log
polynomial accounts for about 14 percent flat, saturated conversion about
2.14 percent flat; those are attribution, not predicted speedups. Complete
sample sets, compiler/profile/binary provenance, and experiment limits are in
[performance notes](performance.md).

### Fifth performance pass

On 2026-10-06, separate guest, host, and adversarial agents inspected the exact
log lowering and tested a guarded nonnegative floor/saturated-conversion fast
path. The polynomial already compiles without floating-point spills or helper
calls; its explicit casts preserve rounding. No safe beneficial polynomial
rewrite was identified. The conversion candidate preserved its operand and
original fallback, with strict helper/import/binding checks. Review also caught
and closed a built-in type-shadowing hole before measurement.

Neither seven-sample comparison established a gain. Unrestricted 500 ms scalar
medians were 76.23 to 75.62 ns (p=0.512); the one-second CPU-14 confirmation was
74.37 to 73.28 ns (p=0.165). The independently verified negative-bin corpus
trended slower in both runs, also inconclusively. Variability remained substantial
even with affinity; no samples were discarded. The complete candidate was
removed. Production, Wasm, translated modules, facades, and locks remain
unchanged, and the restored binary is byte-identical to the frozen baseline.
The apparent scalar shifts are not a retained improvement.

Useful tests remain in `1672671` and `b03d81b`: nine true adjacent bin boundaries
located in unchanged Wasm, full Count/protobuf scalar/batch comparisons,
zero/tiny/negative-bin mixes, a verified negative-bin benchmark, and an
independent numeric fixture that executes actual translated/hardened Go and
wazero against explicit integer/rounding/side-effect/trap goldens. A deliberately
wrong generated helper must fail semantically. `make smoke` includes the new
fixture, which also passes against the original hardener.

Qualification covers strict check, both real/repeat proof builds, Rust checks,
race, five fuzz targets, external/relocation/Go-only smoke, native linux/386 and
amd64-v3 numeric contracts, and all five cross-compilation targets. Arm64 was
cross-compiled, not natively executed in this session. The numeric fixture's
CGO-disabled child is not race-instrumented by a parent race run. Restored
hardening coverage remains 94.0 percent; removed-code coverage is not a gain.
Retained commits pass darna and exact staged-snapshot tests and remain local,
unpushed. Results, limits, hashes, and raw-log locations are in the performance
notes. No default PGO profile was added.

### Remaining performance work

Track 3 has an approved implementation and streaming qualification; track 1's
host encoding stage and track 5's publication-tail pass are also qualified.
Track 6 now has a negative private backend comparison. Further stages below
remain experiments. Distinguish standalone
`SketchAdd` latency from ingestion throughput: batching and sessions can improve
throughput without making an individual `Add` faster.

PR #1 merged at `0a5969e`. The first follow-up reproduces the default fixed
P1/64 B streaming cost at 2–3% in frozen binaries. Bounded New/Write/End/Close
controls detect no phase slowdown in their separately linked binaries, and
adding only benchmark source reverses the full-stream comparison's sign;
removing it rebuilds the frozen binaries byte for byte. This does not establish
a cause or a fix. Retain the controls and the explicit streaming negative,
with [samples and limits](performance.md#default-streaming-cost-investigation-2026-10-11).
No production change or new contract follows from that investigation. The
subsequent low-copy host encoding round qualified matched inputs and
standalone/streaming holdouts with a freshly rebuilt independent spike, as
recorded in track 1. Track 5 now also qualifies a conservative contained-load
bounds pass; standalone scalar still prevents M1 performance acceptance.

The entry baseline matrix compares scalar versus batch calls, repeated
values, distinct nearby values, mostly unique values, and negative-bin values.
Measure encoding, copies, allocation, admission, reply validation, and guest
execution separately as well as end to end. Existing scalar and 1,000-point
batch benchmarks use different inputs; their per-point timings are not a
controlled speedup comparison. Record cache hit rates and estimated ceilings
before committing to larger designs.

The low-copy host encoding stage is qualified below. Exact-value bin reuse has
a phase-0 no-go for both measured defaults. Translation analysis has produced
qualified publication-tail and contained-load passes, while compiled wazero loses the private
same-Wasm control. The latest structural ceiling below sets the next bounded
feasibility proof; do not broaden the retained translation pass without new
proof and qualification. Typed returns still need a compatibility
decision before implementation; the session decision is recorded in track 3.
Translation analysis and the alternative-backend comparison are diagnostic
tracks; use their results to decide whether larger compiler or runtime work is justified.

The next scalar phase-0 round parks adjacent numeric read/write guard reuse.
Fresh profiles and a three-site pinned prototype remove exactly the duplicate
store checks, but 826 paired rows establish only a 0.4–0.5% raw guest gain;
standalone scalar remains undetected and the broad scalar holdout has adverse
comparisons. Keep evidence only, without a production spec or broader predicate.
The subsequent existing-ABI checked unit-reply round below is also a no-go:
default scalar has no detected primary gain and SQL holdouts regress repeatedly.
See [the complete new evidence](performance.md#checked-unit-replies-existing-abi-phase-0-2026-10-12)
and the earlier [read/write outcomes](performance.md#scalar-read-and-write-bounds-phase-0-2026-10-12).
The subsequent covered store32 phase-0 round is also a no-go under its
predeclared rule. Two source-pinned descending-store checks are removed while
all writes and first/new-maximum guards remain. Native amd64/386 proofs pass;
all 411 frozen hashes match through 966 rows. Original scalar screening
87.81→88.03 ns is undetected (.620); confirmation 88.02→86.71 ns gains 1.49%
(.005). Both beat the copied-check control, but repeated untouched gain is
required. Increasing scalar gains against both controls and repeated batch
gains against baseline; these do not replace the original scalar requirement.
Keep positive and negative evidence without a private production spec or pass.

P1/P16 full-stream brackets use byte-identical binaries; all comparisons are
undetected, including adverse fixed 16-home medians. Untouched confirmation
writes/s are about 1.59M fixed / 1.71M holdout at P1; P16 one home is 1.31M /
1.46M and 16 homes is 9.49M / 9.33M. These are lifetime workload observations,
not maximum capacity bounds, causal guest gains or Batch-session equivalence.
Current original scalar misses absolute/fresh limits, and SQL also misses
confirmation's fresh limit (2,173 ns versus 2,158.2 ns). M1 stays unaccepted.
See [all outcomes and limits](performance.md#covered-store32-bounds-phase-0-2026-10-12).
The two-site probe is parked. With no representative application available,
the next separately preregistered structural diagnostic measures avoided
numeric work in the public synthetic Datadog fixture. It extracts only the
exact pure-log block and uses an unsupported 64 MiB precomputed table, with
enabled/disabled modes in one byte-identical binary. All 520 frozen hashes
match through 1,232 rows. Original checked fixed-prefix scalar improves
87.49→70.26 / 87.74→70.74 ns (19.69%/19.38%, both .001); gains against the
disabled control are 22.30%/21.72% (both .001). The predeclared 10% ceiling rule
passes all four comparisons. This is neither a production cache nor an
original adaptive M1 acceptance measurement.

Keep its negatives: pure outlining slows scalar 2.46%/1.63%; enabled negative
bins lose 2.87%/3.45%, broad scalar 3.40%/2.55%, broad batch 10.15%/9.51%, and
SQL holdout 1.71%/1.49% versus untouched, all detected in both sets. The same
ELF disabled comparison is mostly undetected on misses; misses are not free.
The caller frame grows eight bytes, and the table/setup differs from untouched.
Exact native amd64/386 proofs and complete finite original-Wasm histories pass;
they do not replay all 4M/8M timed values through Wasm or certify arbitrary
interval membership. All gates on unchanged production and two identical
61-file proof/runtime builds pass; diagnostic overlays receive separate
actual-mode fixture/runtime tests and native proofs. No production change or
public contract is retained.

Next commit a private bounded feasibility specification before implementing
any certified-interval prototype. Prove the pinned floating-point membership
and fallback conditions first, then measure hit/miss rates, lookup/certification
cost and bounded-memory ceiling; park it if costs erase the benefit. The
unsupported 70.74 ns oracle still misses scalar's absolute and diagnostic
fresh-reference limits. See
[results, provenance and limits](performance.md#structural-scalar-ceiling-exact-log-oracle-2026-10-12).
A typed-return ABI remains a separate compatibility proposal requiring explicit
operator approval before implementation.

The next private step is specified in
[bounded numerical feasibility](certified-interval-feasibility.md): an offline
exact-rational certificate proof for one pinned configuration, fixed seeds,
bounded work/logical storage and independent rounding/native/Wasm checks.
Implement that proof after the reviewed specification checkpoint. Its first
decision is sound nontrivial interval coverage and honest construction cost;
it adds no production cache, public contract or M1 acceptance.

#### 1. Low-copy batching

The 2026-10-11 entry measurements are recorded in the performance notes and
`docs/measurements/low-copy-phase0.txt`. Matched ordered scalar/batch groups and
an existing-API checked prototype cover all five corpora, four sizes, P1/P16,
one/sixteen homes, host allocations, and warmed logical guest memory. Several
large-batch prototype comparisons improve about 4–7%; small/noisy cases show
no detected gain. This supports the private host-only experiment specified in
`docs/low-copy-batching.md`. The host stage is implemented and qualified with
repeated full-facade gains, all quality gates/integration, and two identical
builds of all three proofs and locks. A first placement candidate was rejected
for a repeated SQL negative; the retained ordering passes the scalar, SQL,
trace and streaming guards with no detected regression in both timing sets.
The original 1,000-point batch improves 6.00%/6.46%, removing one Go allocation.
Matched larger groups improve about 3–5%; sharded N32's gain lacks confirmation.
The fresh spike scalar remains about 55.5 ns versus 89.7 ns, so M1 remains
unaccepted. See [retained and rejected evidence](performance.md#low-copy-numeric-inputs-qualified-host-encoding-2026-10-11).
The prototype requires no new public contract, and its measured same-address
copy bypass is verified for Go 1.26.8 amd64. Rust decoding stays owned.

- First generate encoding directly into a checked guest-memory reservation,
  eliminating the intermediate Go byte buffer and copy. Keep reservation and
  memory access inside the execution lock; do not let borrowed views escape or
  survive memory growth. Benchmark sizes from one point through large batches,
  including allocation and memory-use measurements.
  This host stage is qualified; numeric helpers are private and emitted after
  operations, with unchanged checked cursor accounting. Its gains do not accept
  standalone scalar latency, guest decoding, or default streaming performance.
- If guest decoding remains material, investigate a call-scoped borrowed or
  lazy-decoded input path instead of collecting an owned Rust `Vec`. This needs
  a generator/guest API design and lifetime/aliasing review, not an isolated
  generated-file edit. Validate the entire batch before any sketch mutation and
  retain input order, including collapse behavior.
  Any new public contract requires an explicit operator decision before
  implementation; none is introduced by the qualified host stage.
- Gate both stages on malformed input, invalid values late in a batch,
  allocation/growth failures, and exact Count/full-protobuf equivalence. Retain
  each stage only if it gives repeatable end-to-end gains without scalar or SQL
  regressions. The generator change should remain useful beyond sketches.

#### 2. Exact bin reuse

Phase 0 is complete (2026-10-11): retain the measured harness/evidence, reject
the two default exact-value cache prototypes. Frozen 8,192-point cyclic inputs
include unique, nearby, mostly-unique, hot/cold and independent broad holdouts;
Increasing and the original scalar guard are nonrepeating. Unchanged-Wasm
singleton labels show substantial nearby-bin locality but zero exact hits on
distinct nearby values. The frozen direct hash also thrashes the 16-value hot
phase; retain that negative rather than tuning it against the corpus.

Matched raw mapping controls cost about 17–18 ns original, 11–13 ns on a hit,
and 39 ns on a miss/fill in confirmation. Complete constant scalar gains fail
the 5% repeated/mixed requirement; original increasing scalar regresses
23.22%/20.48% for capacities 1/64, and original AddMany1k loses 34.41%/33.36%.
Matched repeated batches gain 37.26%/35.90%, but the held-out batch loses
50.21%/53.05%. No default cache implementation, specialization or public
contract follows. All quality gates/integration and repeated proof/lock builds
pass; the fresh spike scalar remains 57.48 ns versus 91.22 ns untouched Cocoon.
M1 performance remains unaccepted. See [conditions, raw samples and limits](performance.md#exact-bin-reuse-phase-0-no-go-2026-10-11).

The later 2026-10-12 exact-log oracle above establishes a larger avoided-work
ceiling, about 19% on checked fixed-prefix scalar versus untouched and about
22% versus its identical disabled control. It does not promote either rejected
exact-value cache or establish certified membership. Next specify a bounded
interval feasibility proof before implementing a prototype; oracle locality
alone remains insufficient. Preserve broad-input and SQL regressions in the
decision, and keep this separate from original adaptive M1 acceptance. The
requirements below remain; exact-value caching on repeated batches does not
solve standalone scalar cost.

- Measure mapping cost and value/bin locality first. An exact cache keyed by
  value bits and all mapping parameters can avoid repeated logarithms, but the
  increasing, unique-value benchmark has no exact-value hits. Include misses,
  collisions, eviction, mostly unique inputs, and a new held-out corpus; the
  existing short repeated-value cycles alone cannot justify retention.
- If nearby distinct values share bins often enough, investigate a small cache
  of certified bin intervals. Prove membership against the pinned mapping's
  floating-point rounding and bin boundaries; approximate exponential
  thresholds or oracle binary search alone are not that proof. Keep the
  original calculation for uncertified values and boundary neighborhoods.
- Cache only the logical mapping result, never a mutable store slot or pointer.
  Every original counter update and collapse operation must still execute in
  order. Mapping internals are private: choose an upstream API or a checked
  compiler specialization before implementation; do not transplant or duplicate
  reference code. Require exact boundary/oracle, Count, protobuf, growth, and
  collapse tests, with bounded cache memory and acceptable miss-path cost.

#### 3. Bounded checked sessions or command buffers

Phase 0 measurements are complete (2026-10-10), with checked runtime-only
sequences and GOMAXPROCS contention. At 16 processors, synthetic 64-byte
shared-home writes reach 5.22 Mcalls/s; 32-command shared-lock sequences reach
8.99, and independent-instance sequences reach 38.83. At 4 KiB, those ceilings
are 0.770, 0.974, and 1.667 Mcalls/s. They are runtime ceilings, not translated
streaming gains. Raw samples, current scalar/SQL/batch/trace baselines, provenance,
and negative results are in [performance notes](performance.md).

The operator approved the [Phase 1 specification](checked-sessions.md) on
2026-10-10: typed finite command buffers, a bounded runtime adapter bridge, and
optional lazy resource homes. The implemented contract preserves per-command
checks, whole-plan preflight, owned-prefix publication, immediate poisoning,
and exact home/epoch ownership. Replacement never migrates old handles; sibling
homes and the stateless pool survive a home fault. Cancellation, abandonment,
aliases, and draining Close have compiled consumer and race proofs. The owned
1 KiB hold-back streaming fixture has native, translated, contract, bulk-memory,
and valid-handle stateful wazero/fuzz proofs.

Independent paired qualification is complete on Go 1.26.8, CGO disabled. At
16 processors, fixed 64-byte streams reach 1.429 Mwrites/s on the original
single home, 11.162 with 16 homes and standalone methods (7.81×), and 10.362
with 16 homes and Batch32 (7.25×). The independent holdout reaches 11.818 and
11.604, respectively, versus 1.581 original/default. Placement supplies the
gain; typed Batch alone slows small writes by roughly 15–20%. Larger chunks,
buffer length, competing-call latency, mixed stateless traffic, and memory
show workload-specific trade-offs. One-home default P1/64-byte fixed streams
regress about 3% in both paired sets. No standalone scalar regression is
detected; SQL medians rise about 0.5–1% without a detected difference, so a
small cost is not excluded. No scalar improvement or equivalence is claimed.
See [complete qualification and negatives](performance.md#bounded-resource-streaming-qualification-2026-10-11).

Retained signed local commits, in dependency order:

| Commit | Change |
|---|---|
| `36a8e6f` | Phase 0 component/contention measurements |
| `bfc77f4` | Operator-approved specification before implementation |
| `a3ebd59` | Bounded checked runtime bridge |
| `2574186` | Separate bounded lazy resource homes |
| `5269e40` | Differential status checks, including expected errors |
| `287abf2` | Opt-in typed command buffers |
| `884c03e` | Streaming fixture and generated/native/stateful proofs |
| `26b32de` | Verified processor counts inside measured benchmark leaves |
| `a3693c2` | Shared generated safety policy and restored standalone paths |

R1–R4/R7, the pre-layout candidate, layout-only, and R9–R11 shapes were rejected
for guardrail regressions or failed qualification. R5/R6 were intermediate
screens, superseded by the retained R8/R12 implementations. Their patches,
samples, and negative results remain in the performance notes. Streaming runs
with parent-only processor setters and the stopped race-overlapping run are
excluded; the corrected harness verifies actual processors for every timed
leaf. No rejected shape is counted as a cumulative gain. No branches or tags
were pushed.

Retain this as opt-in concurrent resource throughput with the explicit default
stream cost. Prefer sharded standalone methods for synthetic small writes;
choose buffer size only after measuring payload costs, fairness, and memory.
The general M1 scalar target remains unmet; M1 performance is not accepted.

Final R12 gates pass: `make check` (tests/vet/49 strict linters, official
golangci-lint 2.13.1), `make race`, `make rust`, `make cross`, `make smoke`,
`make fuzz`, and `make integration`. Qualification uses Go 1.27.1, Rust 1.97.0,
and Binaryen 133; benchmark binaries use Go 1.26.8 with CGO disabled. Cross
checks compile linux/arm64 and 386, Darwin/arm64, Windows/amd64, and js/wasm;
they do not execute those targets. Race runs actual generated consumer
subprocesses with race instrumentation. Native shim tests and all-feature
Clippy pass for all three fixtures. Two further builds of each proof and
`go generate ./rt` reproduce all 54 committed proof/runtime files byte for
byte, including both original locks and the new streaming lock. Darna and
exact staged-snapshot tests pass before retained commits.

Track 3 is complete under this scope. Resume other tracks with the same
approval and measurement discipline; external injector ownership and demo
integration remain separate work.

The original entry requirements remain guardrails for future extensions:

- First measure the admission/ownership/execution-lock share and a prototype's
  attainable ceiling. A bounded session could admit once and execute several
  ordered calls under shared ownership; this amortizes real work and is distinct
  from the rejected callback-fusion experiment.
- Specify admission granularity, limits, cancellation, partial completion,
  re-entry restrictions, and draining `Close` before implementation. Preserve
  per-operation memory checks, call-state reset, reply validation, and immediate
  fault poisoning; subsequent commands must not run after a protocol fault.
- Keep the standalone API unchanged. Test shutdown races, errors in the middle,
  traps, and nonescaping call/memory views. Judge this as a throughput feature,
  not a claim about standalone scalar latency. Any new public contract needs
  explicit approval before implementation.

#### 4. Typed-return ABI

The 2026-10-12 existing-ABI checked unit-reply phase-0 round is a no-go.
Fresh ordinary Go 1.26.8 profiles put ResultView at 11.17% cumulative scalar
samples; the unit-only component is about 8.6 ns. These overlapping/isolated
scopes are not removable scalar shares. A source-pinned authored runtime/emitter
overlay preserves canonical descriptor/range/limit/status/empty checks, one
reply Memory query and the independent post-operation memory/poison boundary.
All three facades regenerate from the authored emitter, with unchanged guest
Wasm. Native amd64/386 query/error/lifetime and propagated generated-consumer
proofs pass; four deliberate mutants fail semantic assertions on both.

All 430 frozen hashes match through 924 paired rows. The component improves
18.72%/18.80%, but default scalar 87.41→87.03 / 88.38→88.59 ns has no detected
gain over untouched production (.456/1.000). It beats the slower call-shape
control, which itself regresses about 1.30% in both rounds. Varied scalar gains
about 1.1%, while varied SQL loses 2.28%/3.31% and SQL holdout loses 2.18%/2.53%
against baseline, with holdout losses also repeating against control. The
premeasurement promotion rule fails. Retain evidence only; no production spec,
implementation or new public contract follows.

P16 streaming uses three real linked variants because timed Close invokes
ResultUnit. All confirmation differences are undetected, including adverse
fixed 16-home medians; a one-home holdout screening loss against control also
remains recorded. Untouched confirmation writes/s are about 1.32 million with
one home/9.27 million with 16 homes for fixed chunks, and 1.48/9.45 million for
the variable/empty-chunk holdout. These are full-lifetime observations, not
strict capacity ceilings or P1/default-stream equivalence. P1 streaming was
not measured. Current scalar still misses both absolute and fresh reference
limits, so M1 remains unaccepted. See
[all comparisons, conditions and negatives](performance.md#checked-unit-replies-existing-abi-phase-0-2026-10-12).

- Measure the reply-validation share before designing a new ABI. Explore a
  payload-free successful unit return and direct scalar returns, with explicit
  status/error handling. Use profiles to choose a matched checked prototype;
  establish its attainable full-facade gain with controls and holdouts.
- Produce a versioned schema, compatibility/migration plan, and generator/runtime
  design for review before implementation. Keep ABI 3's canonical descriptor
  validation unchanged; this is not permission to skip existing checks.
- Qualify unknown statuses, malformed errors, traps, memory limits, poisoning,
  and resource ownership, plus old/new ABI compatibility. Retain only if the
  end-to-end gain warrants the additional contract and maintenance cost.

#### 5. Translation optimizations

The subsequent 2026-10-12 scalar read/write phase-0 probe is a no-go. Ordinary
Go 1.26.8 profiles and matching SSA/assembly demonstrate three surviving adjacent
load64/store64 guards. Pinned overlays preserve first checks, all accesses,
rounding and endian/fallback semantics; native amd64/386 helper proofs pass and
two semantic mutants fail runtime assertions. All 274 frozen fingerprints match
through 826 rows.
Raw guest improves about 0.4–0.5% against both controls, but default scalar
87.69→87.51 / 87.64→87.43 ns has no detected gain versus either control.
Broad scalar has a screening loss versus unchanged and a confirmation loss
versus copied-check; these are different comparators. SQL/control effects do
not rescue the declared scalar rule. Streaming uses identical binaries in both
roles: fixed P16 one-home confirmation rates are about 1.35 million writes/s,
versus 10.4 million with 16 homes, with no candidate effect claimed.
No implementation/spec/public contract follows; M1 remains unaccepted.
Preserve [the complete evidence and limits](performance.md#scalar-read-and-write-bounds-phase-0-2026-10-12)
and keep the existing production predicates narrow. The checked unit-reply
ceiling within ABI 3 is now measured under track 4 and is also a no-go; a new ABI
requires a compatibility specification and explicit operator approval before
implementation.

The 2026-10-12 matching-profile round qualifies a conservative contained-load
bounds hardener pass. The private predicate and phase-0 evidence are committed
in `be8f6e0` before the authored candidate `22cdc0a`. Full binding/construction,
type, effect and nonescape guards retain the first/new-maximum checks in place
and every guest read. Generate 318 Datadog, 99 compute and 20 streaming selected
reads, with all modules/locks regenerated; no hash/function/payload selector.

Fresh 826-row qualification repeats varied SQL −6.76%/−4.56% and SQL holdout
−3.93%/−3.97%, with all three measured batches repeating smaller gains versus
baseline. Original SQL confirmation (.073) and original batch confirmation
versus recheck (.053) are undetected. Trace improves about 4%, but the restored
check control also improves about 1.5%; no per-check causal attribution follows.
Standalone scalar gains/losses are undetected. One-home fixed streaming has
+1.88%/+1.68% medians (.165/.053), and the one-home holdout's detected screening
loss does not confirm. These negatives do not establish streaming equivalence.

All 167 frozen fingerprints match through timing; all six gates plus integration,
native amd64/386 proofs/consumers, three runtime-rejected semantic mutants, two
identical 61-artifact rebuilds and exact staged validation pass. Summed guest
native text shrinks 9,012 B but the test executable grows 89,312 B in debug metadata.
M1 remains unaccepted: SQL meets its absolute/fresh-reference limits while
scalar 87.87/88.11 ns misses both. No public contract changes.
See [paired outcomes and negatives](performance.md#contained-load-bounds-production-qualification-2026-10-12)
and [the private predicate/proof](translation-contained-loads.md). The later
exact-log structural ceiling above sets the next bounded feasibility proof.
Other translation patterns and typed returns remain separate experiments;
typed returns need a compatibility specification and operator approval.
Keep this predicate narrow and retain streaming guards. Arm64 still needs native
hardware; no earlier screening percentages are added to this qualification.

The 2026-10-11 production round retains a conservative publication-tail hardener
pass. Its private structural predicate was committed before implementation and
new timing; it validates bound helpers/types, addresses, construction/ownership,
and region effects. Function-number/line/payload filters remain diagnostic only.
All three proof modules and locks regenerate from authored inputs. Executable
WAT/wazero and native-header proofs cover aliasing, trap prefixes, growth,
imports, logical length, and labelled loop entry; three semantic mutants fail
the runtime oracle. Native Go 1.26 amd64/386 correctness, all six gates plus
integration, two identical proof/runtime rebuilds, and staged validation pass.

Fresh isolated Go 1.26.8 pairs repeat batch gains: original time −11.39%/−11.58%
and matched repeated-value time −11.91%/−11.90% (screen/confirmation). Matched
confirmation throughput rises 22.99→26.10 million points/s. Broad batch, SQL,
and streaming have no detected gain; SQL/trace upward medians and the 16-home
streaming holdout's +1.07%/+2.94% remain recorded. No smaller repeated standalone
scalar loss occurs, but code placement/control effects prevent a scalar latency
claim. M1 performance remains unaccepted. This adds no public contract.

See [conditions, negatives, rates, and complete evidence](performance.md#publication-memory-views-production-qualification-2026-10-11)
and [the private specification](translation-memory-views.md). The next matching
profile analysis is recorded above; it does not broaden this retained predicate. Track 6's
same-Wasm control is complete and negative for its measured configuration.
Treat further work as a separate measured experiment; do not broaden the
retained predicate based on this batch result or add screening gains together.
Native arm64 performance still requires hardware; cross compilation is not
execution evidence.

- Inspect matching Go compiler SSA/assembly and guest profiles for repeated
  memory accesses or Wasm stack temporaries that provably do not escape. Scope
  the first experiment to one demonstrated pattern; cleaner generated Go alone
  is not evidence of faster machine code.
- Prove alias/effect conditions before promoting memory-backed temporaries to
  Go locals or eliminating accesses. Preserve imports, memory growth, bounds
  checks, and trap/side-effect order, including adversarial aliasing cases.
- Implement in the translation/hardening pipeline with drift checks, executable
  differential tests, and regenerated proof artifacts. Use all workloads and
  available native architectures to qualify it; park the idea if analysis cost
  or code size outweighs the measured benefit.

#### 6. Alternative-backend control

The 2026-10-11 private control is complete. All 18 matched warm raw/checked
compiled-wazero comparisons lose against translated Go in both rounds, including
batch, SQL, trace, and holdouts. Retain the harness and full negative evidence;
park a production backend switch for this measured configuration. No runtime,
facade, generated module, public contract, or default PGO change is retained.
Fresh production scalar and SQL both miss their original and fresh reference
plus 10% limits, so M1 remains unaccepted.

Warm execution and three fresh-process cold compound scopes are separate.
Compiled calls use cached exports and reusable stack carriers. A separate
profile attributes their 16 B/export allocation to the private adapter's
`WithoutCancel` context boxing; no alternative context was timed. Increasing
roles have different calibrated prefixes, logical guest sizes exclude native
executable memory/RSS, and cold zero-growth is a placeholder. Layer differences
are nonadditive diagnostics; the checked adapter is not a supported facade.
All gates/integration, exact staged validation, three reviewer audits, and two
identical 61-file proof/runtime reproductions pass. Existing P16 streaming
controls refresh the observed one/sixteen-home throughput without a new source
gain or default-stream acceptance claim.

See [scope](backend-control.md) and
[paired results, provenance, limits, and compressed evidence](performance.md#same-wasm-backend-control-diagnostic-no-go-2026-10-11).
Next follow the later structural ceiling's bounded feasibility decision above.
A context/call-carrier control would be a separate private experiment; this
result cannot predict its outcome.
Any supported backend proposal still requires a public-contract decision.

- Compare the same pinned Wasm in compiled wazero against translated Go using
  identical scalar and batch inputs and equivalent facade/safety work. Separate
  warm-call cost from compilation/startup, executable-memory use, and caching.
  A faster backend is a hypothesis, not an established result.
- Use this initially as a diagnostic control for translation quality, not a
  default-runtime switch. A backend feature would require a separate decision
  about the no-engine promise, platform support/fallbacks, and startup costs,
  even if the embedding remains CGO-free.
- Keep error, trap, ownership, memory-limit, and shutdown behavior comparable.
  Identify where any difference comes from before proposing a runtime change;
  tiny export calls can have a different tradeoff from guest-heavy batches.

#### Shared qualification gates

Each implementation round starts with adversarial subagent review, then uses
small darna-validated commits with passing exact staged-snapshot tests and strict
lint. Preserve bounds checks, canonical replies, fault containment, shared
ownership, and terminal draining `Close`. Add regression tests and regenerate
affected proof artifacts; never optimize generated modules alone.

Repeat isolated before/after and independent spike samples for all four original
workloads, both varied workloads, and the negative-bin workload, plus each
track's new holdouts. Retain original absolute limits and fresh same-machine
comparisons; scalar remains above both. SQL's margin varies across sessions.
Reject unsupported gains or material cross-workload regressions, report negative
results, and do not add separate screening gains together or mark M1 performance
accepted prematurely.

#### Secondary backlog

- Consider batch-only bit classification, leaving scalar validation unchanged.
  The full predicate improved batches but regressed SQL; partial targeting is
  only a hypothesis. Retain exhaustive domain and whole-batch atomicity tests.
- Use the SQL profile/call-layer diagnostics (`8d3b074`) to select guest-side
  work. Its matching profile puts 85.15 percent cumulatively in the guest export
  and 77.87 percent in `fn96`; identify the source/assembly hot path first.
- Keep PGO opt-in at the consuming application's main package. Obtain a profile
  representative of deployed traffic; synthetic weighting and batch regressions
  limit current evidence. Use new holdouts and compare each implementation with
  its own profile under matched conditions.
- Revisit exact numeric lowering only with a new assembly-backed hypothesis,
  not the rejected guard again. Combined sign/clamp lowering or exact IEEE-bit
  guards remain untested and likely limited in upside. Preserve rounding/bin
  boundaries and checked built-in/import bindings, run executable oracles and
  both proof builds, and qualify native 386/arm64 where available. Include all
  workloads and negative bins; do not substitute approximate math.

## Follow up quality work

The follow-up quality pass is complete as of 2026-10-05. Its nine commits are
pushed, and [all three CI jobs](https://github.com/darccio/cocoon/actions/runs/37357278079)
passed. No async implementation was added in this pass.

`make check`, `make race`, `make cross`, `make integration`, and `make smoke`
passed for this pass. Both real proof builds reproduce the release artifacts
and locks without generated-source changes. The expanded CLI, publication, and
build tests also pass with Go 1.26.8 and native linux/386; the latter requires
execution outside the sandbox because its syscall filter rejects 32-bit tests.

- CLI coverage increased from 72.7 to 92.8 percent. Hermetic fixture tools cover
  build, doctor, and verification workflows, tool failures, canceled builds,
  output errors, temporary-file cleanup, initialization failures, and retries.
  They supplement rather than replace the real-tool smoke test.
- Generation/publication coverage increased from 84.8 to 92.9 percent. New tests
  exercise a real publication rename failure, restore replaced files and their
  permissions, remove newly published files, check staging cleanup, preserve
  authored files, and verify guard failure and retry behavior.
- Build orchestration coverage increased from 85.7 to 92.2 percent. New tests
  compare the entire published artifact set after tool-stage failures, reject
  missing or malformed output and helper drift, check capability and ABI policy,
  and exercise source and executable digest failures. ABI verification itself
  reaches 96.1 percent. Runtime, hardening, and both generators remain above
  94 percent.
- A regression test exposed a `doctor` component-check bug. It now checks exact
  Rust component names and accepts CRLF output and a missing final newline.
- Upstreaming the bulk-memory fix is no longer needed: wasm2go
  [PR 62](https://github.com/ncruces/wasm2go/pull/62) was merged on 2026-10-04.
  The latest tagged release checked on 2026-10-05 is still v0.4.16, predating
  that fix. Keep Cocoon's pin and checked AST hardening until a separately
  qualified upgrade; already-bounded helpers require corresponding drift-check
  updates and regenerated proof artifacts, not removal of the safety checks.

### Additional ABI follow up

The pre-existing destructor unit-reply strictness gap is fixed in `3fb2ffc`.
`Call.ResultUnit` validates a borrowed reply under the execution lock and rejects
nonempty successful unit replies with protocol poisoning. Generated destructors
use it; both facades and locks were regenerated. Compiled facade fixtures test
canonical replies, malformed/status/limit errors, once-only Close, sibling
invalidation, and healthy replacement instances. The malformed successful-reply
regression was also demonstrated to fail against the old generator.

The point-boundary differential tests explicitly assert the reference status
and canonical unit reply. The generic differential generator now also compares
status codes, including non-OK codes, before converting application errors;
all three regenerated proof adapters include this check. This follow-up is
complete. Matching messages alone do not establish status equivalence.

## Next milestone

Add async resources and a terminal event loop, real guest-future cancellation,
per-call host-operation tracking, bounded call/byte admission, budgeted shutdown,
HTTP and sleep capabilities, and response readers that reject overflow rather
than truncating. Then implement the pipeline proof using structured records.
Decide how to distribute its much larger generated source at that point.

### Tokio compatibility

Rust async support does not imply compatibility with arbitrary Tokio programs.
The proposed event loop can poll guest `Future`s, but Cocoon targets
`wasm32-unknown-unknown`, with no WASI or browser runtime. Tokio documents limited
[Wasm support](https://docs.rs/tokio/latest/tokio/#wasm-support): selected features
are available, but timers require platform support and an indefinitely idle
runtime can panic. Cocoon HTTP and sleep capabilities would not automatically
provide Tokio's networking or timer drivers.

If Tokio-based libraries are a requirement, add a separate compatibility proof
before promising support. Define the supported feature subset, bridge waking
and scheduling to the host-driven loop, and demonstrate cancellation, shutdown,
and any required timer or I/O adapters. Native multithreaded Tokio and arbitrary
socket-based crates are not assumed compatible with the current target.

## Resume and verification

Read this file, [ABI contracts](abi.md), [design review](design-review.md), and
the performance notes. Inspect `git status` before editing. Keep changes small,
use `darna --committable --dependants` and staged validation, and test the staged
snapshot before each commit. Reference directories may be inspected for evidence
but their source must not be transplanted into this implementation.

Use `make check`, `make race`, `make integration`, and `make smoke` for
correctness gates; `make cross`, `make fuzz`, and `make coverage` provide
additional qualification.
Benchmark in isolation with Go 1.26.8 and CGO disabled for comparison with the
recorded spike results. Run no compilation, linting, or fuzzing concurrently
with measurements. Local profiles and build caches live in ignored `.cache`;
durable conclusions belong in these committed docs, not only in chat or caches.
