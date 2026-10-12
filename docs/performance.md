# Cocoon performance and PGO

Cocoon preserves terminal admission/drain, shared ownership validation, epoch
invalidation, bounded/canonical result checks, and full input validation.
Benchmarks are a gate to investigate, not a reason to remove those checks.

## Contained load bounds: production qualification (2026-10-12)

Retain the conservative authored hardener pass and its executable proof. Varied
SQL and the SQL holdout repeat detected end-to-end gains against both the original
binary and the copied-check control. All three measured batches repeat smaller
benefits against the original binary. The original SQL confirmation difference
is undetected; standalone scalar gains are not established. Streaming supplies
no demonstrated improvement and its upward one-home medians remain negatives.
**M1 performance remains unaccepted.**

The private [predicate](translation-contained-loads.md) and phase-0 evidence were
committed in `be8f6e0` before implementation. The reviewed production candidate
is `22cdc0a`; its exact staged tree
`e412ecf494dbab217d8585fe1fdb9a92d448135d` passes tests on Go 1.27.1 and 1.26.8,
vet, formatter and pinned strict lint before the signed local commit. No new
API, ABI, manifest option, ownership rule or host contract is introduced.
The publication-view predicate and every runtime admission/memory/reset/reply/
poison/drain check remain unchanged. Guest, host and adversarial reviews approve
retention under this measured scope, with the negative results below.

The [readable record](measurements/contained-loads-production-2026-10-12.md),
[actual compressed archive](measurements/contained-loads-production-2026-10-12.tar.gz)
and [SHA-256](measurements/contained-loads-production-2026-10-12.tar.gz.sha256)
preserve all 826 raw rows, all custom metrics, exact commands/timestamps,
source/build/reference identities, SSA/assembly, proof/mutant/gate/reproduction
logs, reviews and procedural negatives. Every member has an exact byte hash.
No sample or outlier was discarded. Production results are independently paired;
phase-0 and earlier publication-tail gains are not added to them.

### Production predicate and proof

Use declaration identity together with the bound name, explicit builtin i32
bases, proved builtin load destinations and scalar local assignments, and fresh
facts for each straight-line block. Keep the first and every new-maximum check
at its original position. Only a contained later read switches to a validated
copy of `load32` without its unsafe-branch endpoint check. Every guest read,
operand, endian conversion, checked unaligned fallback and nosplit directive
remains. Unknown calls/imports, writes, growth, control/labels, declarations and
unproved expressions end facts; assignment invalidates its destination after
its RHS. Address-taking or closure capture excludes the base throughout the
method; `go` excludes the method. Full helper/import/architecture/receiver and
separately allocated construction guards remain mandatory. Unsupported shapes
and namespace collisions skip the optional pass without new build errors.

Fresh authored generation selects **318 Datadog, 99 compute and 20 streaming**
reads, with all three modules and locks regenerated. It deliberately omits one
of phase 0's 319 Datadog reads: `fn177` encounters an inferred arithmetic
`:=` temporary whose type the narrow recognizer does not prove. No function
number, source line, payload identity, hash or benchmark result selects production
regions. The original Wasm, Rust/shim, facade and adapter bytes remain unchanged;
only the three translated-Go lock hashes change.

Authored AST tests cover grouped bindings, inferred/explicit typing, pure locals,
i64 identity drift, shadowing, constructor/helper/import/namespace drift,
first/new-maximum/read order, RHS-before-kill, earlier escapes, barriers, labels
and composition with the existing publication pass. A weighted WAT fixture
executes **520** independent wazero interpreter cases across ten exports,
26 bases and two flags. It checks results, complete memory including growth,
traps, import events and observable prefixes. Exact nonvacuous selection asserts
only the intended covered reads in two fixture exports; grouped two-base reads
retain separate guards. Native tests add logical length below capacity,
unaligned/end/high-bit cases, header shrink/replacement, live changed backing,
shadowed callback imports, labels, globals and first-trap prefixes.

Original, production, copied-check and forced-checked-fallback fixture roles pass
native Go 1.26.8 amd64 and 386. Three deliberately broken transforms must fail
runtime assertions: removing the first check, crossing an import barrier, and
using stale range facts after an impure locally named identity callback. They
run with readable spare backing capacity so failure is a semantic mismatch,
not a hardware crash. Full generated consumers/runtime also pass native 386.
The smoke child is CGO-disabled and is not race-instrumented by its parent.
Cross compilation supplies no native arm64 execution or performance evidence.

### Conditions, identities and comparison limits

Timing freezes clean `22cdc0a` and **167** input/binary fingerprints; all match
through both rounds and final audits before evidence documentation. Baseline
module overlays exactly restore `be8f6e0` while keeping the same authored
harness/runtime/facade sources; its Datadog binary reproduces the preceding
baseline byte for byte. The production role is named `probe` in preserved raw
records; it is the actual authored candidate, not the phase-0 overlay. `recheck`
differs only by restoring one endpoint-check line in that private helper, with
identical selected call inventory. Streaming now pairs original and regenerated
modules rather than merely refreshing an unchanged binary.

Measure Go 1.26.8, Linux/amd64, CGO disabled, `GOAMD64=v1`, explicit `-pgo=off`,
default GC/memory settings, Ryzen 7 5800HS (8 physical/16 logical cores), unpinned
CPUs 0–15, powersave governor and boost enabled. Policy stays unchanged. Rust
1.97.0, Binaryen 133, wasm2go 0.4.16 and wazero 1.12.0 are pinned. Quality uses
Go 1.27.1 and golangci-lint 2.13.1. Every exact argv, environment/build information
and boundary timestamp remains in the archive.

Seventeen groups run 59 fresh processes per pass: twelve original/corpus leaves
with four roles, one raw guest leaf with three, and four streams with two.
Seven interleaved 500 ms passes followed by seven reversed one-second passes
yield 826 successful rows, seven samples per case/round. Matched leaves verify
actual P1 (280 rows), both stream roles actual P16 (112). Original/layer leaves
retain initial framework P16 without an in-window metric (434); the framework
suffix does not prove timed processors. Compilation, tests, lint, fuzzing,
profiling and reviewer tools stay stopped during timing. Process checks cannot
exclude unrelated system activity. No cold/startup measurements are run.

The independent reference is freshly rebuilt from its own locked Rust/Go source,
separate absent target, std/panic_abort, own hardener and original native/wazero
oracles; all ten journaled commands succeed. All 56 selected source hashes match
before/after/current snapshots and are preserved as bytes. Its three artifact
hashes reproduce phase 0. Reference ABI 2 borrowed inputs/background contexts/
four instances differ from ABI 3 owned checked inputs/test contexts/one home;
original batch/trace inputs also differ. Reference ratios are contextual, not a
pure facade cost. Increasing scalar uses different calibrated prefixes of one
ordered corpus and is evolving-state work, not equal final states.

| Identity | SHA-256 |
| --- | --- |
| Unchanged Datadog Wasm | `ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333` |
| Production translated Go | `626550cafa323c490a6a2ec6b2b341656c40e223c7c84701e414078633dcc298` |
| Baseline executable | `59a3d4159f82a3f91dea9507b4cd24474f43efc19ceceb867400588e9de539cb` |
| Production executable | `74af29033f97ef277b9dd2708e7eaa7d0556c5486715866f5dfb8cfc715b3685` |
| Recheck executable | `e1e70b2aa1a748e150bc5daa0179d3f1683bcca738998aa55a035227c7abde34` |
| Fresh reference executable | `f8846dd93bd22b1adcf3b7762f685c8e96199a9d77a62f7d8ad9f1639a48bbea` |

### Paired outcomes

Entries are exact medians, screening → confirmation, with seven samples per
role/round. Time change is production/baseline minus one. P-values below are
exploratory and unadjusted; complete intervals and secondary comparisons are
archived. Nondetection does not prove equality or absence of a smaller loss.

| Workload | Baseline, screen → confirm | Production, screen → confirm | Time change, screen → confirm | p, screen / confirm |
| --- | ---: | ---: | ---: | --- |
| Original SQL | 2.216 → 2.217 µs | 2.136 → 2.134 µs | -3.61% → -3.74% | 0.001 / 0.073 |
| Original scalar | 87.91 → 87.80 ns | 87.87 → 88.11 ns | -0.05% → +0.35% | 0.710 / 0.620 |
| Original batch 1000 | 38.775 → 38.809 µs | 38.201 → 38.287 µs | -1.48% → -1.35% | 0.026 / 0.038 |
| Original trace 1000 | 2.663214 → 2.683518 ms | 2.554478 → 2.580011 ms | -4.08% → -3.86% | 0.001 / 0.001 |
| Varied scalar | 89.00 → 88.96 ns | 88.36 → 88.91 ns | -0.72% → -0.06% | 0.383 / 1.000 |
| Varied SQL | 2.857 → 2.827 µs | 2.664 → 2.698 µs | -6.76% → -4.56% | 0.001 / 0.001 |
| Negative bins | 88.32 → 88.54 ns | 88.51 → 88.74 ns | +0.22% → +0.23% | 0.832 / 0.535 |
| SQL holdout, P1 | 5.670 → 5.691 µs | 5.447 → 5.465 µs | -3.93% → -3.97% | 0.011 / 0.001 |
| Increasing scalar, P1 | 89.35 → 89.69 ns | 89.34 → 89.73 ns | -0.01% → +0.04% | 1.000 / 0.620 |
| Broad scalar, P1 | 88.85 → 89.42 ns | 88.70 → 88.73 ns | -0.17% → -0.77% | 1.000 / 0.097 |
| Repeated batch 1024, P1 | 39.466 → 39.484 µs | 39.074 → 39.150 µs | -0.99% → -0.85% | 0.015 / 0.026 |
| Broad batch 8192, P1 | 318.490 → 320.441 µs | 312.380 → 314.529 µs | -1.92% → -1.84% | 0.001 / 0.001 |
| Raw guest scalar | 53.67 → 53.89 ns | 53.01 → 53.22 ns | -1.23% → -1.24% | 0.017 / 0.004 |

Varied SQL and SQL holdout repeat detected gains versus both baseline and
recheck, satisfying the specified end-to-end SQL criterion. Against recheck,
varied SQL has −5.46%/−4.93% (p=.001 both), holdout −3.68%/−3.75% (p=.007/.001).
Original SQL confirmation is undetected versus baseline (.073) **and recheck
(.053)**; its favorable median is not a repeated detected original-SQL benefit.
Original batch repeats a gain versus baseline (.026/.038), but its confirmation
versus recheck is undetected (.053). Repeated and broad batches repeat detected
gains against both controls. Confirmation SQL holdout throughput rises
175,711→182,976 queries/s; repeated/broad batch throughput rises
25.935→26.156 and 25.565→26.045 million points/s.

Trace improves −4.08%/−3.86% versus baseline (p=.001 both) and −2.66%/−2.35%
versus recheck (.002/.011). Recheck itself improves −1.46%/−1.55% against baseline
(.026/.017): restored checks and matching text sizes do not eliminate role or
placement effects. No pure per-check causal attribution follows. No detected
standalone scalar gain or loss occurs in either primary round; negative bins
have upward +0.22%/+0.23% medians (.832/.535), original scalar +0.35% confirmation
(.620), increasing +0.04% confirmation (.620). Broad scalar's confirmation gain
versus recheck (.043) does not repeat and is undetected versus baseline.
The raw guest's −1.23%/−1.24% is separate scope, not facade latency acceptance. Original scalar versus recheck has +0.37%/+0.44% medians
(.734/.403); increasing has +0.49%/+0.89% (.073/.209). Neither establishes a
repeated detected loss, but their upward directions remain recorded.

Scalar/batch allocate zero. Original SQL reports 57→52 B and one allocation in
all three translated roles. Holdout reports baseline 96→84 B, production 95→83 B,
all one allocation. Trace baseline/production reports 254,395/252,471 B screen
and 233,892/232,988 B confirmation, all one allocation; calibration/allocator
amortization changes these bytes, so no memory-footprint gain is claimed.
Reference trace reports two/one allocations across rounds. Matched batches retain
1,179,648/1,441,792 current guest bytes, zero growth; this is neither process RSS
nor peak/resident memory. Streaming does not report timed guest-memory metrics.

### Streaming guard and ceilings

All four streams pair the same checked facade and authored harness with original
and regenerated guest modules, actual P16. Timing covers each complete 16 KiB lifetime: New, Write, End and
Close. Fixed input uses nominal 64-byte chunks and retains output in groups of
32; holdout varies chunk boundaries. Library Open, warm-up and full byte-oracle
validation are excluded. Both roles retain 16,392/16,390 owned
output bytes and 250/268 allocations for fixed/holdout. Sixteen-home controls
retain configured pool one. Times and every writes/s/streams/s/byte metric remain
in raw records; these rates establish no resource-topology change by this pass.

| P16 workload | Baseline time, screen → confirm | Production time, screen → confirm | Time change | p, screen / confirm |
| --- | ---: | ---: | ---: | --- |
| One home, fixed | 188.461 → 189.475 µs | 192.002 → 192.664 µs | +1.88% → +1.68% | 0.165 / 0.053 |
| 16 homes, fixed | 25.744 → 27.076 µs | 24.811 → 27.668 µs | -3.62% → +2.19% | 0.259 / 1.000 |
| One home, holdout | 228.056 → 231.835 µs | 229.621 → 232.709 µs | +0.69% → +0.38% | 0.038 / 0.535 |
| 16 homes, holdout | 33.048 → 35.394 µs | 32.869 → 34.148 µs | -0.54% → -3.52% | 0.456 / 0.710 |

One-home fixed time rises +1.88%/+1.68% (p=.165/.053); retain the repeated upward
direction, not a claim of unchanged performance. One-home holdout screening's
+0.69% loss is detected (.038), but confirmation's +0.38% is undetected (.535).
Sixteen-home fixed flips from −3.62% to +2.19%, both undetected; holdout's
−0.54%/−3.52% is undetected. There is no demonstrated streaming gain. None trips
the specified repeated 3% guard threshold; the strict smaller standalone scalar
guard also has no repeated detected loss. These limits do not establish
streaming equivalence or authorize broader optimization regions.

| Observed confirmation P16 throughput | Baseline streams/s → production | Baseline writes/s → production |
| --- | ---: | ---: |
| One home, fixed | 5,278 → 5,190 | 1,351,101 → 1,328,738 |
| 16 homes, fixed | 36,933 → 36,142 | 9,454,836 → 9,252,453 |
| One home, holdout | 4,313 → 4,297 | 1,481,660 → 1,476,090 |
| 16 homes, holdout | 28,254 → 29,284 | 9,705,140 → 10,059,079 |

Production fixed throughput scales 6.96× from one home to sixteen; holdout scales
6.81×. These are workload-specific observed rates, not a universal throughput bound. These lower-than-phase-0 rates are different-run conditions, not additional
before/after losses or gains. Only the same-round pairs above qualify this pass.

### Machine size, gates and acceptance

Ordinary timing-binary assembly and separate Go 1.26 opt/regalloc/genssa dumps
show actual code changes; no diagnostic SSA binary is timed. Sketch native symbol
text shrinks 5,834→5,462 B and frame 144→128 B; static panic sites fall 88→78.
SQL text shrinks 111,763→110,236 B, frame remains 856 B, panic sites 1,501→1,452.
Scalar export remains 3,004 B/112 B/57 sites. Static SSA StoreReg/LoadReg counts
are sketch 40/160→37/137, SQL 365/4,652→360/4,599, export unchanged 18/49.
These are static compiler operations, not dynamic spill counts or isolated
latency savings. Recheck restores the original three symbols' sizes/frames/sites.

Summed translated guest native symbol text falls 831,228→822,216 B (−9,012 B);
this is not a Wasm byte reduction. Full native `.text` falls 9,056 B. The test
executable grows 14,435,431→14,524,743 B (+89,312 B, +0.62%), dominated by
`.debug_rnglists` growth of 99,147 B. `.debug_loclists` falls 1,848 B and
`.gopclntab` 2,644 B. The checked control is 14,442,607 B. Source/debug/code
placement and register allocation therefore remain attribution limits.
Generation-time analysis cost is not independently benchmarked in this round.

All six gates plus integration pass on final production: `make check` (tests,
vet and all pinned strict linters), `race`, `rust`, `cross`, `smoke`, `fuzz`,
and `integration`. Two complete compute/Datadog/streaming proof builds plus
runtime regeneration reproduce all 61 artifact/harness fingerprints and locks.
Available native amd64/386 proofs and generated consumers pass; cross covers
arm64, 386, Darwin, Windows and js/wasm. Exact staged tests/vet/fmt/lint pass
before `22cdc0a`. No default PGO profile or new backend is added; no branch/tag
is pushed in this round.

Preserve implementation negatives: the first generic scanner incorrectly keyed
grouped variables by their shared declaration node; tests/review force distinct
binding keys. Earlier address/callback escapes needed a whole-method guard, and
a standalone helper printer dropped nosplit metadata until explicit copying.
All were corrected before generated artifacts or qualification timing. The first
`make check` read AST tests before their final t.Parallel edits and failed lint;
the final sequential gate sequence passes. Native386's sandbox runner fails
before execution (bad system call); authorized execution passes. The spec
checkpoint's overlapping staged validations caused a linter lock failure;
sequential final validation passes. Failure logs and corrections are retained,
not silently converted into successful samples.

M1 original absolute limits remain scalar **55.44 ns** and SQL **2.2154 µs**.
Fresh reference plus 10% gives scalar **60.808/61.193 ns**, SQL
**2.1560/2.1692 µs** (screen/confirmation). Production SQL medians
2.136/2.134 µs meet both SQL limits, but scalar **87.87/88.11 ns** misses both
scalar limits. Confirmation contextual production/reference ratios are scalar
1.584×, SQL 1.082×, batch 1.070×, trace 1.269×. Useful throughput and SQL gains
do not satisfy the full acceptance gate: **M1 remains unaccepted**.

Retain `be8f6e0` for measurement/specification and `22cdc0a` for the tested generic
pass/proofs/regenerated locks; the final evidence commit records qualification.
Reject the incorrect declaration-only/escape/directive drafts before retention;
do not promote the phase-0 module overlays. Next inspect matching scalar
profiles and actual compiler code for another demonstrated exact pattern or
measure the typed-return share before specifying a new ABI. A typed-return
contract still needs operator approval. Keep the streaming negatives as guards,
never broaden this predicate from favorable workload results, and do not add
historical screening percentages together.

## Contained load bounds: phase 0 (2026-10-12)

Matching profiles and compiler output identify redundant `load32` endpoint
checks within short straight-line regions. The private probe repeats original,
varied, and holdout SQL gains against both the untouched binary and a copied
helper retaining the checks. Original batch and repeated-value batch also repeat
smaller gains. This warrants a narrowly specified production experiment, not
retention of edited generated modules. **Production remains unchanged at this
checkpoint; M1 performance remains unaccepted.** Standalone scalar and trace
improvements are not established.

The [private production specification](translation-contained-loads.md) must be
committed before implementation. Guest, host, and adversarial review cover the
probe, proof and interpretation; the phase-0 archive retains their findings and
pre-timing corrections. The [readable record](measurements/contained-loads-phase0-2026-10-12.md),
[actual compressed archive](measurements/contained-loads-phase0-2026-10-12.tar.gz),
and [SHA-256](measurements/contained-loads-phase0-2026-10-12.tar.gz.sha256) preserve
all 770 raw process rows, all custom metrics, profiles, SSA/assembly, fixture and
mutant sources/logs, scripts, provenance, intervals and negatives. No sample or
outlier was discarded. Member bytes have a SHA-256 manifest.

### Conditions and provenance

Begin after PR #6 merges: freshly fetched `main` and `origin/main` both identify
`776b897e74b7faef44375b2c31bd3c17ac88f876`; the branch starts clean. Timing freezes
that unchanged production tree and 189 input/binary fingerprints, all matching
through both rounds and the final audit. Production Wasm remains
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`;
translated Go remains
`f92c5853385d9c4b3cfc3d164507def81beff36b9f7a8564c92f247c9a3ab948`.
Baseline executable SHA-256 is
`59a3d4159f82a3f91dea9507b4cd24474f43efc19ceceb867400588e9de539cb`;
probe is `93d81048dafa25305833a5486eba019733d11e1b44e9c81f820f04882c94e68c`;
recheck is `cf57003adc967ae24cba3958e5e6a6bb6c6043ca64e4783b8e878058be04b38b`.
All three use identical authored harnesses and original-Wasm oracles. Neither
an overlay nor a generated-module-only edit is retained in production.

Measure Go 1.26.8, Linux/amd64, CGO disabled, `GOAMD64=v1`, explicit `-pgo=off`,
default GC/memory settings, Ryzen 7 5800HS (8 physical/16 logical cores), unpinned
CPUs 0–15, powersave governor and boost enabled. Rust 1.97.0, Binaryen 133,
wasm2go 0.4.16 and wazero 1.12.0 are pinned. CPU policy stays unchanged. Exact
commands, UTC timestamps, executable build info and environment are archived.

Seven alternating 500 ms passes followed by seven reversed one-second passes
run 55 fresh process invocations each: twelve original/corpus leaves with four
roles, one raw guest leaf with three, and four unchanged streaming leaves. This
produces 770 rows, with seven samples per case/round. Matched leaves verify P1
in-window (280 rows); streams verify P16 (56). The 434 original/layer rows retain
framework CPU 16 and lack an in-window processor metric; the suffix does not
prove timed processors. Separate ten-second P1 CPU profiles precede timing and
are excluded from acceptance samples. No compilation, tests, lint, fuzzing,
profiling or reviewer tools overlap qualification timing. The process audit
cannot exclude unrelated system activity. No cold/startup comparison was run.

The independent unversioned spike is freshly rebuilt from its own locked Rust
inputs and clean libdatadog `7f3b16fe1b4bfc2a016ef869459e915ee02d6d6a` dependency, with ten journaled commands;
its original native/wazero tests and 48 matched one-iteration oracles pass.
Reference Wasm/Go hashes reproduce the preceding round; the executable is
`f8846dd93bd22b1adcf3b7762f685c8e96199a9d77a62f7d8ad9f1639a48bbea`.
The archive preserves 56 selected reference source files and their hashes.
Reference ABI 2 borrowed inputs, background context and four-instance topology
remain different from ABI 3 checked owned inputs, test contexts and one home.
Original batch/trace inputs also differ; reference ratios are contextual.
Increasing-scalar roles consume different calibrated prefixes of the same
ordered corpus, so they describe evolving state rather than equal final states.

### Pattern, machine code, and executable proof

Five separate sequential CPU profiles locate translated guest work: `fn176`
flat/cumulative samples are 32.39%/42.00% of checked scalar, 54.33%/70.50% of
raw scalar, and 57.18%/78.46% of original batch. `fn96` is 46.99%/77.84% of SQL
and 36.63%/64.66% of trace. Inlined/cumulative shares overlap and are not additive
attainable gains. An existing Go 1.24 pprof viewer reads the Go 1.26 profile;
ordinary Go 1.26 opt/regalloc/genssa dumps and matching assembly are retained.

The sketch region reads offsets four, eight, then zero on one unchanged local
base. Keep the first endpoint check (+7) and new maximum (+11) at their original
positions; only the final +3 check is contained. The diagnostic selects 319
later checks across 69 methods by bound local identity and straight-line range
facts. Every guest read stays live; no stores/loads are forwarded, no header is
cached, and no bounds check moves. Unknown calls, imports, writes, growth,
rebinding, declarations, labels and other control boundaries clear facts.
The helper's unaligned fallback/endian semantics remain intact. The whole-source
hash pins helpers, construction, bindings and architecture assumptions for this
diagnostic; it is not a generic production proof.

| Ordinary symbol | Baseline/recheck text, frame, static panic sites | Probe text, frame, static panic sites |
| --- | ---: | ---: |
| Sketch body `fn176` | 5,834 B; 144 B; 88 | 5,462 B; 128 B; 78 |
| SQL body `fn96` | 111,763 B; 856 B; 1,501 | 110,236 B; 856 B; 1,452 |
| Scalar export | 3,004 B; 112 B; 57 | 3,004 B; 112 B; 57 |

These are static sites, not dynamic counts. The sketch frame changes, and the
helper/call rename can affect placement and register allocation. The recheck
control restores baseline text sizes/frames/site counts but is not a perfect
causal control. No claim of fewer spills or a pure per-check cost follows.

The authored WAT fixture compares nine exports against wazero's interpreter on
16 bases and two flag values each: results, traps, import counts, and full memory
including growth. Only the intended two contained fixture reads are selected;
rising ranges, stores, imports, rebinding, wrapping and conditional entry retain
checks. Native proofs add logical lengths 0–20 with spare capacity, exact/short
endpoints, unaligned and high-bit addresses, import-side shrink/backing replacement,
first-trap prefixes, labels, global bases and an impure local `i32` callback.
Original, probe and recheck pass native Go 1.26 amd64 and 386. An explicit checked
unaligned fallback also passes. There is no native arm64 execution evidence.

Three semantic mutants fail the executable oracle: removing the first check,
carrying a fact across an import that shrinks the header, and trusting a local
impure identity callback by name. The callback is declared before the first
checked read so a declaration barrier cannot mask the name-only purity bug.
All three whole Datadog packages pass their original contract, differential,
backend and corpus tests before timing. The checked facade still performs every
runtime memory/reset/ownership/reply/poison/drain check.

### Paired outcomes and negative results

Entries are exact sample medians, screening → confirmation. Units are retained
per row. Change is probe/baseline minus one. All raw samples, every allocation
and throughput metric, intervals, reference/recheck comparisons, and exploratory
unadjusted p-values remain in the archive; nondetection is not equivalence.

| Workload | Baseline, screen → confirm | Probe, screen → confirm | Time change, screen → confirm |
| --- | ---: | ---: | ---: |
| Original SQL | 2.205 → 2.197 µs | 2.116 → 2.116 µs | -4.04% → -3.69% |
| Original scalar | 88.02 → 87.62 ns | 87.75 → 87.49 ns | -0.31% → -0.15% |
| Original batch 1000 | 38.716 → 38.551 µs | 38.047 → 37.904 µs | -1.73% → -1.68% |
| Original trace 1000 | 2.622961 → 2.600098 ms | 2.614163 → 2.597131 ms | -0.34% → -0.11% |
| Varied scalar | 88.39 → 88.75 ns | 87.71 → 88.24 ns | -0.77% → -0.57% |
| Varied SQL | 2.812 → 2.800 µs | 2.675 → 2.655 µs | -4.87% → -5.18% |
| Negative bins | 89.28 → 88.28 ns | 88.02 → 87.65 ns | -1.41% → -0.71% |
| SQL holdout, P1 | 5.671 → 5.665 µs | 5.446 → 5.471 µs | -3.97% → -3.42% |
| Increasing scalar, P1 | 89.18 → 89.15 ns | 88.65 → 88.91 ns | -0.59% → -0.27% |
| Broad scalar, P1 | 89.42 → 89.14 ns | 89.04 → 88.81 ns | -0.42% → -0.37% |
| Repeated batch 1024, P1 | 39.402 → 39.389 µs | 39.116 → 39.066 µs | -0.73% → -0.82% |
| Broad batch 8192, P1 | 319.311 → 319.300 µs | 313.224 → 312.823 µs | -1.91% → -2.03% |
| Raw guest scalar | 53.92 → 53.76 ns | 52.55 → 52.66 ns | -2.54% → -2.05% |

Original and varied SQL each detect the probe benefit versus both baseline and
recheck in both rounds (p=.001 throughout). SQL holdout versus baseline has
p=.001/.017, versus recheck .001/.011. Original batch repeats a −1.73%/−1.68%
benefit (p=.001 both); repeated batch repeats −0.73%/−0.82% (p=.001/.010).
Broad batch's −1.91%/−2.03% baseline differences have p=.128/.001: only the
confirmation difference is detected. Against recheck it has p=.026/.001.
This does not establish a repeated detected broad-batch baseline gain.

Original scalar's −0.31%/−0.15% is undetected (p=1.000/.833); the raw guest
improves −2.54%/−2.05% (p=.023/.001), which does not establish facade latency
improvement. Varied scalar versus baseline is undetected/detected (.165/.002)
but versus recheck remains undetected (.128/.089). Negative bins versus baseline
has p=.005/.026; versus recheck .128/.004. Increasing and broad scalar differences
remain undetected in both rounds. No repeated probe scalar loss appears, but the
rename/control effects prohibit a general standalone speedup claim.

Trace's −0.34%/−0.11% is undetected (p=.620/1.000). The recheck trace rises
+0.34%/+1.07% against baseline (p=.318/.011); the confirmation control slowdown
is retained as a negative. Removing checks therefore does not supply a universal
workload gain. Probe/reference confirmation ratios remain scalar **1.583×**,
SQL **1.090×**, batch **1.065×**, trace **1.299×**; these include the reference
contract/input differences described above. M1's original absolute limits remain
55.44 ns scalar and 2.2154 µs SQL. Fresh reference plus 10% gives 60.786/60.808 ns
scalar and 2.1472/2.1362 µs SQL (screen/confirmation). The diagnostic SQL is below
both sets of SQL limits, but scalar misses both by a wide margin. A private probe
cannot qualify production M1, and SQL alone cannot satisfy the full acceptance gate.

Scalar/batch allocate zero in all three translated roles. Original SQL reports
57 → 52 B/one allocation (screen → confirmation); holdout metrics are preserved. Trace medians vary slightly
with allocator calibration while retaining one allocation; this is not a
memory-footprint optimization. Matched corpus/batch leaves retain timed current linear-memory and growth
metrics; these are neither process RSS nor peak/resident memory. Streaming leaves
do not report timed linear-memory metrics.

### Streaming ceiling refresh and gates

The streaming binary and source are unchanged: these are P16 refresh controls,
not before/after qualification of a generic transform. Fixed 64-byte streams have
256 writes and 16,392 owned output bytes; holdout metrics retain their varying
chunk history. All streaming contract, wazero and bulk-memory tests still pass.

| P16 control | Time, screen → confirm | Streams/s, screen → confirm | Writes/s, screen → confirm |
| --- | ---: | ---: | ---: |
| One home, fixed | 181.692 → 181.957 µs | 5,504 → 5,496 | 1,408,979 → 1,406,929 |
| 16 homes, fixed | 23.459 → 23.970 µs | 42,628 → 41,719 | 10,912,669 → 10,680,060 |
| One home, holdout | 221.516 → 218.258 µs | 4,514 → 4,582 | 1,550,678 → 1,573,824 |
| 16 homes, holdout | 31.340 → 31.472 µs | 31,908 → 31,774 | 10,960,348 → 10,914,305 |

Fixed confirmation throughput scales 7.59× from one home to sixteen homes;
holdout scales 6.93×. These describe the existing checked resource topology,
not a new streaming improvement. Owned output bytes remain 16,392/16,390 and
allocations 250/268 for fixed/holdout.

All required gates pass sequentially before isolated timing: `make check`
(Go 1.27.1, pinned golangci-lint 2.13.1), `race`, `rust`, `cross`, `smoke`,
`fuzz`, plus `integration`. Two complete compute/Datadog/streaming proof builds
and runtime regeneration reproduce all 61 tracked artifact fingerprints and
locks. Native 386 needs an escalated runner because of the sandbox; its initial
approval timed out before execution, and the permitted retry passes. Cross
compilation covers arm64, 386, Darwin, Windows and js/wasm, not native arm64 timing.
Staged-snapshot validation is required before committing this documentation.
These gates qualify the unchanged production tree and diagnostic proof, not a
future generic pass; implementation must regenerate and repeat its own gates.

Procedural negatives remain archived: the first fixture invocation treated Go
payload filenames as additional `go run` sources and failed before tests; adding
`--` fixes it. Initial helper AST positions caused excess formatter line breaks,
corrected before any overlay build/timing. Review rejects global address bases,
reserved-name collisions and name-only identity purity before freeze. The older
182-file freeze is preserved; no measurement used it. Initial frame parsing
missed `ADDQ $-0x80`; corrected machine evidence records the true 128-byte frame.
The independent reference's unused-import warning is retained. None is a discarded
benchmark sample. The next step is implementing the committed generic predicate
and independently qualifying it; phase-0 results must not be added to earlier gains.

## Same-Wasm backend control: diagnostic no-go (2026-10-11)

Retain the private benchmark controls and their negative evidence. Compiled
wazero loses every matched warm workload against translated Go in both rounds,
including batch, SQL, and trace work. This configuration supplies no reason to
change the production backend. Production source and generated artifacts remain
unchanged. **M1 performance remains unaccepted:** refreshed scalar and SQL both
exceed the original absolute limits and the fresh reference plus 10% limits.

The [private scope](backend-control.md) was committed in `5dee157` before the
controls in `9ad3434` and before qualification timing. The controls introduce
no public contract, backend option, ABI change, or default PGO profile. Guest,
host, and adversarial reviewers approve retention as diagnostic evidence.
The [readable record](measurements/backend-control-2026-10-11.md) accompanies an
[actual compressed archive](measurements/backend-control-2026-10-11.tar.gz) and
[compressed SHA-256](measurements/backend-control-2026-10-11.tar.gz.sha256).
The archive preserves all 1,064 raw process samples, custom metrics, exact
commands/timestamps, summaries, benchstat intervals, fingerprints, build and
reference provenance, gate logs, reviews, and procedural negatives. Member
bytes have a SHA-256 manifest. No sample or outlier was discarded; the archive
does not expand the evidence into a large text diff.

### Conditions, identity, and comparable work

Start from merged `e56bfa9`, with `main` and freshly fetched `origin/main` equal.
Timing freezes the clean `9ad3434` tree. All 117 frozen input/binary fingerprints
match through both rounds and the final audit before evidence documentation.
Production Wasm is
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`;
translated Go is
`f92c5853385d9c4b3cfc3d164507def81beff36b9f7a8564c92f247c9a3ab948`.
The freshly built timing binary is
`59a3d4159f82a3f91dea9507b4cd24474f43efc19ceceb867400588e9de539cb`.
These are the same production payload bytes used in the preceding round.

Use Go 1.26.8, Linux/amd64, CGO disabled, `GOAMD64=v1`, explicit `-pgo=off`,
default GC/memory settings, and unpinned CPUs 0–15 on Ryzen 7 5800HS (8 physical,
16 logical cores). The recorded governor is powersave with boost enabled;
neither policy nor affinity was changed. Rust 1.97.0, Binaryen 133,
wasm2go 0.4.16, and wazero 1.12.0 are pinned. Exact environment, dynamic CPU
frequencies, executable hashes, and build information are archived.

Warm timing has 73 fresh process invocations per pass: seven alternating 500 ms
passes, then seven one-second passes with leaf and role order reversed, yielding
1,022 rows. Cold timing adds 42 fresh-process single-operation observations:
seven per scope in each of two reversed-order rounds. Every process succeeds.
New matched warm leaves report actual P1 (672 rows); streaming reports actual
P16 (56 rows); all 42 cold leaves report actual P1. The 294 original/layer
controls retain their existing processor policy and lack an in-window custom
processor metric. Their initial P16 and `-16` framework suffix do not prove the
timed count. Compilation, tests, lint, fuzzing, profiling, and reviewer tools
do not overlap qualification timing. Process checks and the sequential driver
cannot prove absence of unrelated system activity.

The compiled control explicitly selects wazero's compiler, caches export
functions and signatures, and reuses a `CallWithStack` carrier, refilling every
argument and checking errors before reading results. Each guest owns its
runtime, capability host, memory, and globals; the factory shares immutable
compiled code. Engine calls use cached `context.WithoutCancel` contexts and
`WithCloseOnContextDone(false)`. Stateless admission checks the caller's context;
Lifecycle and standalone resource methods are context-free.
There is no timed interpreter fallback or cross-module capability dispatcher.

`CheckedControl` uses the real `rt.Lifecycle`, `Resource.Use`, `Instance.Call`,
one-slot lazy pool, and recycling policy. Both engines preserve per-call memory
checks/reset, ownership/epochs, canonical reply checks, immediate poisoning,
and draining Close. Scalar calls do not reserve input; variable-input calls do.
Functional oracles cover complete ordered Count/protobuf, SQL, and trace bytes,
invalid late batch values without mutation, expected errors, retained outputs,
growth/current views, alias Close, malformed replies, actual guest/host faults,
canceled admission, and drain. Backend-specific fault kinds/messages differ;
containment and causes are checked. This private adapter adds dispatch and
omits poisoned-home replacement. It is not a generated facade or supported
backend. Old handles never migrate, and no recovery event enters warm timing.

`RawGuest` still includes export dispatch and workload adapters. SQL/trace
reserve and copy inputs, check status/bounds, and decode owned output; they are
not pure guest-body timings. Stateless checked setup retains an unused home
alongside the measured pool guest. Increasing-scalar roles consume different
calibrated prefixes of the same ordered corpus: Count is verified, but final
states and prefix lengths differ. Those rows describe evolving state rather
than stationary equal-state work.

### Same-Wasm warm outcomes

Entries are exact sample medians, screening → confirmation, with seven samples
per role/round. Time change is compiled/translated minus one. All 18 primary
engine pairs have exploratory, unadjusted **p=.001 in both rounds**. An
undetected difference elsewhere does not establish equivalence. Complete
intervals and every metric remain in the archive; these comparisons do not
identify one intrinsic engine or guest-body cost.

| RawGuest workload | Translated Go, screen → confirm | Compiled stack, screen → confirm | Time change, screen → confirm |
| --- | ---: | ---: | ---: |
| Repeated scalar | 56.32 → 56.06 ns | 141.6 → 143.3 ns | +151.42% → +155.62% |
| Increasing scalar | 56.60 → 56.20 ns | 143.0 → 146.3 ns | +152.65% → +160.32% |
| Negative bins | 56.26 → 56.07 ns | 140.6 → 144.5 ns | +149.91% → +157.71% |
| Broad scalar holdout | 54.17 → 54.55 ns | 145.0 → 147.5 ns | +167.68% → +170.39% |
| Repeated batch 1024 | 39.419 → 39.525 µs | 53.906 → 53.470 µs | +36.75% → +35.28% |
| Broad batch 8192 | 319.876 → 320.114 µs | 401.430 → 401.739 µs | +25.50% → +25.50% |
| Original SQL | 1.860 → 1.882 µs | 3.531 → 3.516 µs | +89.84% → +86.82% |
| SQL holdout | 4.886 → 4.922 µs | 9.757 → 9.876 µs | +99.69% → +100.65% |
| Trace 1k | 2.696649 → 2.679231 ms | 4.292910 → 4.291769 ms | +59.19% → +60.19% |

| CheckedControl workload | Translated Go, screen → confirm | Compiled stack, screen → confirm | Time change, screen → confirm |
| --- | ---: | ---: | ---: |
| Repeated scalar | 96.64 → 97.03 ns | 198.6 → 203.1 ns | +105.50% → +109.32% |
| Increasing scalar | 97.34 → 97.92 ns | 201.7 → 200.2 ns | +107.21% → +104.45% |
| Negative bins | 96.32 → 96.34 ns | 197.1 → 200.7 ns | +104.63% → +108.32% |
| Broad scalar holdout | 97.68 → 98.43 ns | 202.3 → 203.2 ns | +107.10% → +106.44% |
| Repeated batch 1024 | 39.774 → 39.596 µs | 53.961 → 53.541 µs | +35.67% → +35.22% |
| Broad batch 8192 | 319.179 → 321.049 µs | 404.893 → 403.439 µs | +26.85% → +25.66% |
| Original SQL | 2.186 → 2.236 µs | 3.969 → 4.018 µs | +81.56% → +79.70% |
| SQL holdout | 5.676 → 5.641 µs | 10.972 → 10.639 µs | +93.31% → +88.60% |
| Trace 1k | 2.688565 → 2.675899 ms | 4.287272 → 4.280850 ms | +59.46% → +59.98% |

Confirmation checked repeated-batch throughput is 25.861 versus 19.125 million
points/s; broad batch is 25.516 versus 20.305 million points/s. Translated
scalar/batch controls allocate zero. Compiled stack scalar allocates 16 B/one
allocation; batch allocates 32 B/two allocations (reserve plus operation).
Raw SQL uses 48/80 B and one/three allocations (Go/compiled); checked SQL
48/96 B and one/four. Holdout SQL uses 72/104 B raw and 72/120 B checked.
Trace uses 212,992/213,024 B raw and 212,992/213,040 B checked, with one/three
and one/four allocations. These heap metrics include the adapter/context.

The cached API `Call` control takes 178.3 → 176.9 ns on raw repeated scalar,
32 B/two allocations, versus stack's 141.6 → 143.3 ns and 16 B/one allocation.
Its penalty is +25.92%/+23.45%, p=.001 in both rounds. Repeated batch `Call`
takes 53.685 → 53.348 µs, 64 B/four allocations, versus stack's
53.906 → 53.470 µs and 32 B/two. Its −0.41%/−0.23% differences are undetected
(p=.477/.209); reduced allocation does not establish a batch latency gain.

A separate 100,000-operation allocation diagnostic on the frozen binary uses
`memprofilerate=1`; its timings are excluded from the 1,064 samples. The profile
attributes allocations to `context.withoutCancelCtx.Value`, Go 1.26 line 609
(`return value(c, key)`), called by wazero's `callWithStack` value lookup.
This matches the observed 16 B/export and is a context-adapter contribution,
not evidence of unavoidable compiled-engine allocation. An existing Go 1.24
pprof viewer reads the Go 1.26 profile after the 1.26/1.27 installations report
no pprof tool. Commands, viewer hash, profile, list, and caller chain are
archived. No alternative context was timed; subtracting allocation bytes or
predicting its speed would not requalify these comparisons.

Warm measured logical memory matches engines: 1,179,648 B for scalar/repeated
batch/SQL, 1,441,792 B for broad batch, and 2,359,296 B for trace. All 532 matched
backend warm samples report zero timed growth. Those observations exclude the
unused home, cached code, test framework, and native executable mappings.
No RSS or retained
process-memory bound is established.

### Current production layers and fresh independent reference

Existing scalar layers are refreshed in the same current binary:

| Layer | Screen → confirm ns/op |
| --- | ---: |
| Guest | 54.25 → 54.07 |
| Checked Instance | 72.11 → 72.52 |
| Resource ownership + Instance | 82.08 → 82.35 |
| Generated facade | 90.06 → 89.84 |
| Empty checked Instance callback | 15.73 → 15.81 |
| Empty Resource callback | 24.87 → 25.07 |
| Empty lifecycle admission | 7.377 → 7.272 |

All allocate zero. Independently timed differences are rough attainable
ceilings, not additive admission/ownership/reply/execution shares. In particular,
the 35.77 ns confirmation facade-minus-guest gap does not prove a saving from
removing one component. Private checked adapter timings are distinct from the
supported facade and cannot replace this measurement.

Rebuild the independent unversioned spike with its locked source, Rust 1.97
std/panic_abort, Binaryen 133 metaDCE/O3, wasm2go 0.4.16 unsafe, and its own
hardener. All ten journaled build/test commands and 48 one-iteration
matched/corpus oracles succeed before timing. Wasm, translated module, and
binary reproduce previous hashes:
`8bda5ff7acff11fd64fd47a0a48165a71d610c28413665e4cc4d4034150d0651`,
`5ff253143943f199c58a89849b478a1c6c46b75ee744ad69b5b3d66cbe1602ab`,
and `f8846dd93bd22b1adcf3b7762f685c8e96199a9d77a62f7d8ad9f1639a48bbea`.
Its ABI 2 borrowed-input facade, background context, four-instance topology,
and differing original batch/trace inputs remain limits. Those original rows
are reference context, not a safety-overhead decomposition. Matched corpora
align inputs and Count checks. Reference source and build scripts are archived.

| Workload | Current, screen → confirm | Fresh reference, screen → confirm |
| --- | ---: | ---: |
| Original scalar | 88.80 → 88.14 ns | 55.67 → 55.64 ns |
| Original SQL | 2.260 → 2.237 µs | 1.976 → 1.966 µs |
| Original batch 1k | 39.069 → 38.592 µs | 35.929 → 35.612 µs |
| Original trace 1k | 2.736681 → 2.693541 ms | 2.117528 → 2.071231 ms |
| Varied scalar | 88.80 → 89.36 ns | 56.72 → 55.74 ns |
| Varied SQL | 2.903 → 2.826 µs | 2.524 → 2.532 µs |
| Negative bins | 90.01 → 88.96 ns | 55.58 → 55.41 ns |
| P1 SQL holdout | 5.727 → 5.695 µs | 5.252 → 5.058 µs |
| P1 increasing scalar | 89.38 → 89.40 ns | 57.63 → 56.47 ns |
| P1 broad scalar | 89.74 → 89.90 ns | 58.45 → 57.96 ns |
| P1 repeated batch 1024 | 39.962 → 39.627 µs | 37.098 → 36.493 µs |
| P1 broad batch 8192 | 322.857 → 320.430 µs | 299.922 → 298.509 µs |

M1 original absolute limits remain scalar **55.44 ns** and SQL **2.2154 µs**.
Fresh reference plus 10% gives 61.237 ns/2.1736 µs in screening and
61.204 ns/2.1626 µs in confirmation. Both production workloads exceed both
applicable limits in both rounds (original reference pairs p=.001 throughout).
Confirmation current/reference overhead is +58.41% scalar and +13.78% SQL.
The SQL holdout's margin moves from +9.04% to +12.59%; it is not an acceptance
substitute. Scalar and matched batch retain zero allocations. Original SQL
has one allocation on each side, current 57→52 B versus reference 79→64 B.
Original trace heap bytes and allocations vary with calibration and differing
inputs; every value is preserved, not treated as a new source change.

These single-current-role refresh controls provide no production before/after
candidate. Historical measurements and changed binary linking cannot establish
source gains, regressions, or guard equivalence.

### Cold compound scopes

Each cell has seven fresh-process `1x` samples per round. Cache seeding is
outside timing and remains alive; no disk cache is used. Uncached compilation
uses neither shared nor disk cache and closes code/runtime inside timing.

| Scope | Screen → confirm median | Go B/op | Allocations/op |
| --- | ---: | ---: | ---: |
| Translated construction + identity/init/close | 71.005 → 59.723 µs | 1,120,368 | 117 |
| Fresh runtime + uncached compile + code/runtime close | 221.801677 → 221.121024 ms | 42,943,368 | 79,218 |
| Cached compile + runtime/imports + instantiate + identity/init/close | 4.202519 → 4.302899 ms | 2,095,128 | 9,138 |

Translated construction is dispersed: screening interval ±245%, range
50.596–245.257 µs; confirmation ±72%, range 35.858–102.766 µs. Single observations
do not provide a precise startup estimate. Uncached compilation does not
instantiate a module. Both construction controls include private adapter setup
and an extra Output export after `rt.NewInstance`; wazero also caches functions
and signatures. Cached guest close preserves factory code until later cache
disposal. These compound scopes cannot be added/subtracted as startup shares
or reported as production construction cost.

Final constructed logical memory is 1,114,112 B on both sides. The cold
`guest-growth-B=0` metric passes final bytes as both before and after and is a
placeholder, not measured construction-time growth. Go heap totals exclude
native executable mappings; executable footprint/RSS remain unmeasured.

### Streaming refresh and qualification

Four existing P16 leaves process complete 16 KiB streams through standalone
New/Write/End/Close, retaining owned outputs in groups of 32. Times include the
complete resource lifetime, outputs, retention, and accounting; Open/warmup and
full byte validation are outside timing. Sixteen-home leaves exercise the
existing explicit home setting. N32 labels mean retention groups, not typed
Batch execution. These are current observed throughput ceilings for the
workload, not strict capacity bounds or a new checked-session speedup.

| Corpus / homes | Screen → confirm µs/stream | Screen → confirm streams/s | Screen → confirm writes/s |
| --- | ---: | ---: | ---: |
| Fixed / one | 189.982 → 190.091 | 5,264 → 5,261 | 1,347,495 → 1,346,721 |
| Fixed / sixteen | 25.972 → 28.161 | 38,504 → 35,510 | 9,856,901 → 9,090,607 |
| Holdout / one | 226.624 → 230.585 | 4,413 → 4,337 | 1,515,727 → 1,489,689 |
| Holdout / sixteen | 33.534 → 34.670 | 29,820 → 28,844 | 10,243,313 → 9,907,774 |

Confirmation write-rate ratios are 6.750× fixed and 6.651× holdout. Fixed uses
256 writes and 257 outputs/stream; holdout varies chunk widths, including empty
writes, averaging 343.5 writes and 344.5 outputs. More writes/s therefore does
not mean faster streams. Fixed owned bytes are 16,392/stream, holdout median
16,390; Go allocations remain 250/268. Median one-home B/op is 17,515→17,508
fixed and 19,622→19,618 holdout, with setup amortization retained. Metrics are
independently rounded, so products/reciprocals of displayed medians can differ.
No previous streaming gain or default-stream regression is newly accepted.

All required gates pass: `make check` (tests, vet, pinned golangci-lint 2.13.1
and 49 strict linters), `make race`, `make rust`, `make cross`, `make smoke`,
`make fuzz`, and `make integration`. Quality uses Go 1.27.1. Final changed-proof
checks also pass Go 1.26.8 amd64 and native 386; unsupported compiled-wazero
platforms explicitly skip its private proofs, with no interpreter fallback.
Native arm64 hardware is unavailable. Two final complete builds of compute,
Datadog, streaming, and `go generate ./rt` reproduce all 61 recorded
proof/runtime/harness files. Generated modules, locks, and facades remain
unchanged. Darna and exact staged-snapshot tests, vet, formatting, and strict
lint pass before retaining each commit.

Procedural failures remain visible. Early proof-construction errors are
recorded as notes where original tool stdout was not saved; they are not
invented raw logs. A final staged lint run found one wasted initialization;
changing it to a declaration corrected the lint. The initial reproduction
guard then correctly caught that single authored test edit between builds;
generated files were unchanged. Two whole reproduction passes ran again after
the final source freeze and matched. A wasm2go version query initially used a
read-only default cache and passed on retry with workspace GOCACHE. The fresh
independent Rust shim retains its upstream unused-import warning. Functional
one-step runs and allocation-profile timings are excluded from qualification.

Park a backend switch for this measured configuration. The retained controls
can test a separately specified context/call-carrier experiment, but do not
predict its outcome. Resume track 5 with matching profiles and Go SSA/machine
code to find another demonstrated, exact translation pattern before proposing
an implementation. Scalar needs substantial progress; SQL's small original
margin does not satisfy its stricter fresh limit. A typed-return or guest API
proposal still needs explicit public-contract approval before implementation.

## Publication memory views: production qualification (2026-10-11)

Retain the conservative hardener pass for batch throughput. Original batch time
falls 11.39% in screening and 11.58% in confirmation; the matched repeated-value
batch falls 11.91% and 11.90%. Confirmation throughput for the matched batch rises
from 22.99 to 26.10 million points/s. The broad batch holdout, SQL, and streaming
have no detected gain. Standalone scalar guards have no repeated slowdown, but
this round makes no scalar latency claim. **M1 performance remains unaccepted.**
The pass introduces no public API, ABI, manifest option, or lock format change.
The implementation is retained in `cd90392`; guest, host, and adversarial
reviewers approve its proofs and the complete qualification evidence.

The [private specification](translation-memory-views.md) was committed in
`865be46` before implementation. `f18a6ec` clarified capture placement inside
labelled blocks before restoring that rule or running new timings. Production
selection uses bound declarations, helper bodies, ownership and address proofs,
and short publication-tail syntax; function numbers, lines, payload names, and
holdout results are never selection inputs. The measured output has 21 Datadog
captures across 20 methods, including both sketch publication tails, and four
captures each in compute and streaming. Every original access, operand order,
logical-length check, trap, and guest byte effect remains. Runtime admission,
reset, ownership, reply validation, poisoning, and drain checks are unchanged.

[The complete archive](measurements/translation-memory-views.txt) retains all
826 raw outputs and metrics, exact process commands and timestamps, frozen
source/binary fingerprints, build and reference provenance, controls, matching
SSA/disassembly, proof/gate logs, and procedural failures. Its compressed bundle
preserves exact artifact bytes with a per-member SHA-256 manifest; readable
sections normalize whitespace only. No sample or outlier was discarded.

### Conditions and provenance

Start from `f18a6ec`, with `main` and freshly fetched `origin/main` both at
`1510d69`. Measurements use Go 1.26.8, CGO disabled, `GOAMD64=v1`, explicit
`-pgo=off`, default GC/memory limits, and unpinned CPUs 0–15 on the Ryzen 7 5800HS
(8 physical / 16 logical cores). The recorded governor is powersave, with boost
enabled; neither policy nor affinity was changed. All six before/after/reload/
reference/streaming binaries were built fresh before timing. All authored
benchmark files byte-match `865be46`; overlays change only generated modules or
adapt the independent reference. All 166 frozen files remain identical through
both rounds and the final hash audit before documentation edits.

Seventeen leaves produce 59 role invocations per pass. Seven alternating 500 ms
passes precede seven one-second passes with role and leaf order reversed. No
compilation, tests, lint, fuzzing, profiling, or reviewer tools overlap timing.
The runner cannot itself prove absence of unrelated external work. Original
seven and raw-guest leaves use initial GOMAXPROCS=16, `-test.cpu=16`, and frozen
source without a processor override; their `-16` suffix is additional evidence,
not an in-window metric. New SQL/corpus/matched leaves verify actual P1, and all
four streaming leaves verify actual P16. Their initial `-16` suffix alone does
not describe the timed processor count. Every custom metric is archived.

The baseline Datadog Go module hashes to
`f50d20e7fe091569353ac1ced7865a53cd23bc9add668c3aac0f5f3a0ac8eed4`;
the production module hashes to
`f92c5853385d9c4b3cfc3d164507def81beff36b9f7a8564c92f247c9a3ab948`.
Original Wasm remains
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`.
Both original proof packages and streaming were regenerated from authored
inputs. Only translated Go bytes and corresponding `go_sha256` lock fields
change; Wasm, facade, schema, and generated proof inputs remain identical.

### Paired outcomes

Entries are medians; each role has seven samples per round. Time change is
after/before minus one. P-values are exploratory and unadjusted across the
matrix. An undetected difference does not establish equivalence. Complete
benchstat intervals and every allocation/custom metric remain in the archive.

| Workload | Before, screen → confirm | After, screen → confirm | Time change, screen → confirm | p, screen / confirm |
| --- | ---: | ---: | ---: | ---: |
| Scalar | 90.51 → 90.17 ns | 88.44 → 89.34 ns | −2.29% → −0.92% | .001 / .002 |
| SQL | 2.167 → 2.184 µs | 2.184 → 2.204 µs | +0.78% → +0.92% | .135 / .090 |
| Batch 1k | 43.248 → 43.287 µs | 38.323 → 38.274 µs | −11.39% → −11.58% | .001 / .001 |
| Trace 1k | 2.553 → 2.589 ms | 2.592 → 2.615 ms | +1.56% → +1.01% | .053 / .456 |
| Varied scalar | 91.39 → 90.55 ns | 89.52 → 89.29 ns | −2.05% → −1.39% | .002 / .001 |
| Varied SQL | 2.763 → 2.761 µs | 2.775 → 2.778 µs | +0.43% → +0.62% | .684 / .332 |
| Negative bins | 90.70 → 90.62 ns | 89.18 → 88.93 ns | −1.68% → −1.86% | .001 / .001 |
| SQL holdout, P1 | 5.443 → 5.427 µs | 5.446 → 5.456 µs | +0.06% → +0.53% | .902 / .334 |
| Increasing scalar, P1 | 91.16 → 91.06 ns | 90.00 → 89.92 ns | −1.27% → −1.25% | .001 / .002 |
| Broad scalar holdout, P1 | 90.63 → 89.73 ns | 90.54 → 90.26 ns | −0.10% → +0.59% | .558 / .777 |
| Repeated batch 1024, P1 | 44.577 → 44.542 µs | 39.267 → 39.240 µs | −11.91% → −11.90% | .001 / .001 |
| Broad batch 8192, P1 | 323.542 → 321.896 µs | 324.239 → 322.280 µs | +0.22% → +0.12% | .805 / .902 |
| Raw scalar guest | 54.39 → 54.52 ns | 53.75 → 53.74 ns | −1.18% → −1.43% | .004 / .001 |
| Stream, one home, fixed, P16 | 188.144 → 188.597 µs | 189.231 → 189.853 µs | +0.58% → +0.67% | .535 / .209 |
| Stream, one home, holdout, P16 | 228.473 → 229.689 µs | 229.427 → 230.506 µs | +0.42% → +0.36% | .456 / .535 |
| Stream, 16 homes, fixed, P16 | 28.003 → 27.870 µs | 27.812 → 28.398 µs | −0.68% → +1.89% | .805 / .209 |
| Stream, 16 homes, holdout, P16 | 35.724 → 35.839 µs | 36.106 → 36.894 µs | +1.07% → +2.94% | .209 / .456 |

The SQL and trace upward medians are negatives, even though neither has a
detected difference. Confirmation trace intervals are ±7% before and ±26%
after; matched repeated batch after is ±11%, and broad batch after is ±12%.
The 16-home streaming holdout's +2.94% confirmation is close to the predeclared
3% repeated-loss rejection threshold, with ±20% before and ±3% after. Its screen
loss is +1.07%; neither round detects a difference. Retention accepts this
remaining uncertainty, not a claim of streaming equivalence. There is no
repeated loss of 3% or more and no smaller repeated standalone scalar loss.
The broad scalar confirmation's +0.59% remains visible.

Scalar, bin, matched-batch, and raw-guest leaves stay at zero Go allocations.
SQL/varied/holdout and traces report one allocation/op in both production roles.
Streaming stays at 250 allocations/stream fixed and 268 holdout. Allocation
bytes vary for setup-amortizing benchmarks: confirmation varied SQL is 77→78
B/op, trace 233,074→233,206 B/op, and one-home holdout 19,618→19,617 B/stream.
These values are retained rather than rounded into a claim of identical bytes.
Reported corpus/matched guest sizes and zero timed growth match across roles.

### Reload control, reference, and machine code

The reload control derives from the actual production-selected inventory,
preserving declarations and all 54 memory-containing statements (42 stores,
12 loads), but refreshing the header between accesses. It repeats batch
penalties: +1.80%/+1.62% for original batch and +1.24%/+1.27% for matched repeat.
The broad holdout is +0.89%/+0.82%, detected only in confirmation (p=.026).
Trace control is +1.40%/+0.87%, detected only in screening (p=.038/1.000).
Scalar control also improves, −1.68%/−1.47% (p=.038/.004). Thus source placement,
liveness, bounds code, and code shape remain attribution limits; this control
restores inspected header reloads, not identical machine code. Reject it as a
production alternative and make no standalone latency or sole-mechanism claim.

Rebuild the independent spike from its locked unversioned source snapshot with
Rust 1.97 std/panic_abort, Binaryen 133 metaDCE/O3, wasm2go 0.4.16 unsafe, and its
own hardener. Fresh native/wazero tests and 48 one-iteration matched/corpus
oracles pass before timing. The rebuilt Wasm and Go binary reproduce their
previous hashes, respectively
`8bda5ff7acff11fd64fd47a0a48165a71d610c28413665e4cc4d4034150d0651` and
`f8846dd93bd22b1adcf3b7762f685c8e96199a9d77a62f7d8ad9f1639a48bbea`.
Its ABI 2 borrowed-input facade, background context, and four instances differ
from checked ABI 3. Original batch/trace inputs also differ; their times are
context only, not facade overhead. Matched corpora and Count checks align.
Confirmation reference scalar is 55.85 ns versus production 89.34 ns; SQL is
1.993 versus 2.204 µs. Matched repeated batch is 36.562 versus 39.240 µs, and
broad batch is 299.072 versus 322.280 µs. These gaps remain; M1 is not accepted.

Matching Go 1.26.8 SSA and machine code confirm four eliminated header reloads
in the two sketch tails. Go 1.27.1 `nm`/`objdump` tools inspect those binaries;
diagnostic SSA binaries are not timing binaries. Sketch `fn176` is 5,866→5,834
bytes, frame 144 bytes in all roles. SSA StoreReg/LoadReg counts remain 40/160,
and stack-operand instruction counts remain 209; stack operands are not all
spills. All 20 selected-method frames remain unchanged. One stack-operand count in
`fn150` falls 1,438→1,437; the other 19 remain unchanged. This is not a proven
spill reduction.
Total generated TEXT shrinks 991,637→991,552 bytes (85 bytes); reload is 991,660.
Offset-pattern counts 102→98→102 corroborate the inspected reloads, not a
general alias proof. No fewer-spills claim follows.

### Streaming rates and qualification

These are fresh before/after generated binaries for four P16 workloads,
using standalone per-write methods. They do not requalify the earlier P1/64
default-stream cost or claim a command-buffer speedup. Fixed streams use 256
64-byte writes, retain outputs in groups of 32, and produce 257 outputs. Rates include complete stream lifetimes, bounded output
retention, and standalone Write/End/Close calls; they are observed throughput,
not strict ceilings. Holdouts vary
chunks/matches, with 344.5 outputs/stream on average. Sixteen-home rows use the
explicit resource-home setting with pool size one. Confirmation metrics are:

| Workload | Writes/s, before → after | Streams/s, before → after | MB/s, before → after |
| --- | ---: | ---: | ---: |
| One home, fixed | 1,357,395 → 1,348,412 | 5,302 → 5,267 | 86.87 → 86.30 |
| One home, holdout | 1,495,502 → 1,490,201 | 4,354 → 4,338 | 71.33 → 71.08 |
| 16 homes, fixed | 9,185,652 → 9,014,624 | 35,881 → 35,213 | 587.88 → 576.94 |
| 16 homes, holdout | 9,584,554 → 9,310,540 | 27,903 → 27,105 | 457.16 → 444.09 |

All seven required commands pass: `make check` (tests, vet, pinned golangci-lint 2.13.1 and
49 strict linters), `make race`, `make rust`, `make cross`, `make smoke`,
`make fuzz`, plus `make integration`. Quality uses Go 1.27.1; all fixtures use
Rust 1.97.0 and Binaryen 133. Two further builds of all three proof packages
and `go generate ./rt` reproduce all 59 recorded files/locks. Exact staged-tree
tests under Go 1.27 and 1.26, vet, formatter, strict lint, and darna pass.

The new fixture runs WAT through the actual pinned translator and production
hardener, then compares 14 complete per-operation histories with original Wasm
in wazero. It covers overlapping writes, later reads, late traps, widened versus
wrapping addresses, imports, growth, and labelled loop re-entry. Native Go
assertions cover header replacement and logical length below capacity. Three
post-validation semantic mutants must run and fail the memory oracle; a compile
failure cannot satisfy that check. Final strengthened fixture and full generated
consumer oracles pass natively on Go 1.26.8 amd64 and 386. Timings are amd64 only;
there is no native arm64 hardware. Cross checks compile all required targets,
and the CGO-disabled smoke child is not race-instrumented by its parent.
Trusted native hosts must obey the documented ownership precondition excluding
fabricated backing aliases inside Module and concurrent raw mutation.

Pre-timing negative runs are preserved: competing lint runs hit its process
lock; an interim blanket labelled-block exclusion disabled the loop proof;
native386 first hit sandbox seccomp; a first SSA selector produced no method
dumps; the independent shim retained its upstream unused-import warning.
Final retries, the pre-timing private spec clarification, and every required
gate pass. Retain the generic pass and executable proofs because the repeated
batch benefit survives those checks with negligible generated code-size cost.
Keep further translation changes separate, with new profiles and their own
qualification; this result supplies no broad scalar/SQL/streaming acceptance.

## Translation analysis: positive phase 0 (2026-10-11)

The memory-view diagnostic finds a promising batch optimization. Reusing the
linear-memory slice header in just two sketch result-publication regions lowers
the original batch's measured time by 11.21% in the narrow screen and 11.86% in
confirmation. The matched repeated-value batch improves by 11.55% and 12.80%.
The broad batch holdout has no detected gain. Retain the SQL holdout harness,
[raw evidence](measurements/translation-phase0.txt), and
[production experiment requirements](translation-memory-views.md). At this
phase-0 checkpoint, no compiler pass, generated-module edit, public API, lock
change, or default profile is retained. M1 performance remains unaccepted.

### Matching profiles and machine code

Start from `5a5c8df` on a clean tree; `main` and freshly fetched `origin/main`
both equal `1510d69`. The profile binary has SHA-256
`cd533f99b56712793f8456c4188a874399711b1fe7052101be87750ebb4eb82e`,
identical to the preceding frozen untouched binary. The unchanged translated
Datadog module hashes to
`f50d20e7fe091569353ac1ced7865a53cd23bc9add668c3aac0f5f3a0ac8eed4`;
its original Wasm hashes to
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`.
Five sequential ten-second CPU-profile benchmarks precede timing. Profiles are
diagnostics, not timing samples. Compiler SSA opt/regalloc/genssa dumps and
disassembly use matching Go 1.26.8 production source and ordinary optimization
flags; dump flags add no optimization. The timing binaries include the new
holdout harness and therefore have different linking from the profile binary.

| Profile | Guest hotspot | Flat | Cumulative |
| --- | --- | ---: | ---: |
| Checked scalar | Sketch export | 7.33% | 51.54% |
| Checked scalar | `fn176` | 29.81% | 37.39% |
| Raw scalar guest | `fn176` | 47.71% | 63.47% |
| SQL | `fn96` | 45.44% | 75.61% |
| Batch 1k | `fn176` | 60.32% | 80.62% |
| Trace 1k | `fn96` | 37.35% | 63.06% |

Checked scalar reply validation accounts for 11.99% cumulatively. Inlined
helpers and containing functions overlap; these percentages are attribution,
not additive attainable gains. The earlier conclusion still holds: the dense
log polynomial is register-resident, explicit rounding is meaningful, and no
new exact math simplification is justified.

The demonstrated pattern is surviving memory-header reloads after inlined
unsafe stores. A SQL initializer reloads the pointer and logical length after
each store, and its two later stack-slot loads also survive. Broad stack-value
promotion is rejected as a starting point: addresses cross calls, generic
pointers can alias, and wrapping i32 addition differs from widened addition.
Keeping every original access while reusing only the slice header has a smaller
proof obligation.

### Diagnostic scope and controls

The ignored AST utility requires the exact full-module SHA. It recognizes
simple straight-line scalar regions, predeclares slice locals to preserve Go
goto rules, and assigns a view at the original first memory-containing statement.
Unproved calls, imports, growth, header mutation, labels, branches, gotos, returns,
nested blocks, declarations, and unsupported expressions are barriers. All loads,
stores, casts, helper checks, and stack restorations retain their original order.
It does not forward values or change logarithms, validation, reply publication,
or checked host work. The ownership precondition is ordinary generated-facade
construction, with separate Module and backing-byte allocations; fabricated
unsafe raw-host aliases and concurrent raw-module mutation are outside this
diagnostic's proof.

The broad probe covers 1,426 regions and 5,656 memory references. An AST roundtrip
byte-matches the untouched module. A control keeps the same region/local
inventory but reloads the header before each memory-containing statement.
Disassembly confirms that this restores inspected reloads; it does not restore
identical machine code. Neither helper bindings nor ownership/address proofs
are generic: the full hash pins them. Existing full Datadog tests and focused
original-Wasm/count/bin/SQL oracles pass before timing. Those checks qualify
screening, not a reusable production transform.

| Function | Before bytes / frame | Broad view bytes / frame | Reload control bytes / frame |
| --- | ---: | ---: | ---: |
| SQL `fn96` | 111,763 / 856 | 109,518 / 872 | 111,891 / 856 |
| Sketch `fn176` | 5,866 / 144 | 5,834 / 144 | 5,797 / 144 |
| Handle `fn84` | 1,029 / 32 | 966 / 32 | 1,061 / 32 |
| Scalar export | 3,004 / 112 | 2,696 / 112 | 3,016 / 112 |

These frames exclude saved-frame/return bookkeeping. The broad math function
has four more instructions despite slightly fewer bytes. Code size, register
allocation, frame size, and placement all change; fewer header loads alone do
not establish the timing mechanism.

### Broad paired measurements

Use Go 1.26.8, CGO disabled, `GOAMD64=v1`, explicit `-pgo=off`, default GC/memory,
and unpinned CPUs 0–15 on the Ryzen 7 5800HS (8 physical / 16 logical cores).
Freeze every binary first. Seven interleaved 500 ms samples per leaf are followed
by seven one-second samples with role and leaf order reversed. No compilation,
lint, tests, fuzzing, profiling, or reviewer tools overlap timing. Preserve every
sample. Original/varied/negative/raw-guest leaves start with GOMAXPROCS=16;
new SQL, bin-corpus, and matched-batch leaves verify actual P1. The `-16` suffix
on those leaves reports initial processors, not their timed processor count.
Translation p-values are exploratory and unadjusted across the comparison matrix.

Rebuild the independent spike from its locked Rust 1.97 shim with std/panic_abort,
Binaryen 133 metaDCE/O3, wasm2go 0.4.16 unsafe, and its own hardener. Native Go and
wazero tests pass; only harness/corpora are shared. Its unversioned local source
archive and generated artifacts are hashed. ABI 2 borrowed input, background
context, and four instances differ from Cocoon's checked ABI 3. Original batch
and trace inputs also differ, so their reference times are controls, not facade
overhead comparisons. Matched batches use identical corpora and Count checks.

| Workload | Before, screen → confirm | Broad view, screen → confirm | Time change, screen → confirm |
| --- | ---: | ---: | ---: |
| Scalar | 99.14 → 91.24 ns | 100.8 → 90.36 ns | +1.67% → −0.96%, undetected |
| SQL | 2.415 → 2.263 µs | 2.424 → 2.227 µs | +0.37% → −1.59%, undetected |
| Batch 1k | 46.080 → 44.101 µs | 41.132 → 39.731 µs | −10.74% → −9.91%, p=.017/.001 |
| Trace 1k | 3.006 → 2.654 ms | 2.880 → 2.646 ms | −4.21% → −0.30%, undetected |
| Varied scalar | 97.01 → 92.68 ns | 95.34 → 91.56 ns | −1.72% → −1.21%, undetected |
| Varied SQL | 3.202 → 2.816 µs | 3.150 → 2.864 µs | −1.62% → +1.70%, undetected |
| Negative bins | 108.0 → 92.81 ns | 102.8 → 90.91 ns | −4.81% → −2.05%, undetected |
| New SQL holdout, P1 | 6.133 → 5.572 µs | 6.662 → 5.629 µs | +8.63% → +1.02%, p=.535/.435 |
| Repeated batch 1024, P1 | 47.942 → 44.970 µs | 43.999 → 40.341 µs | −8.22% → −10.29%, p=.002/.026 |
| Broad batch 8192, P1 | 351.116 → 326.724 µs | 351.522 → 329.074 µs | +0.12% → +0.72%, undetected |

Increasing and broad scalar bin corpora also show no detected change. Checked
scalar and batch workloads remain zero-allocation; SQL/trace allocations are
preserved in raw rows. The SQL holdout cycles six long/comment/escaped/Unicode/
quoted-name queries and compares complete outputs against original Wasm in two
untimed correctness passes. Its benchmark includes initial lazy guest setup,
not an untimed full-corpus warm pass.

Screening dispersion is substantial: original SQL's interval reaches ±66%.
The first SQL holdout slowdown does not repeat and is not statistically detected;
it establishes neither a material regression nor equivalence. The reload control
lacks the repeated-batch gain, but its confirmation regresses varied SQL by 4.44%
and trace by 3.75% (both p=.038), demonstrating code-shape sensitivity. Retain
these negatives rather than claiming all effect derives from header reuse.

Fresh confirmation reference scalar is 57.10 ns and SQL is 2.054 µs. The checked
baseline is 59.79% and 10.18% slower respectively. Baseline scalar 91.24 ns remains
above the original 55.44 ns limit; SQL 2.263 µs is above 2.2154 µs. The measured
candidate does not satisfy those limits either. No M1 acceptance follows.

### Narrow attribution experiment

The successful AddMany per-point loop retains the same point-load → `fn176` →
status-load sequence and 208-byte frame. Its ten rewritten regions mostly cover
one-time setup/finalization or errors. The two `fn176` terminal regions at baseline
lines 36198 and 36206 execute per point and remove four memory-header reloads.
The 144-byte frame and original guest accesses remain. Three further hash-pinned
controls select the nine sketch-function regions, ten batch-export regions, or
only those two publication regions. These function/line filters are diagnostic
controls, not acceptable production rules.

Repeat seven 500 ms samples and seven reversed one-second samples with the same
frozen before/broad/narrow binaries. This focused experiment covers original
scalar/batch, raw guest, and matched repeated/broad batches; it does not qualify
the narrowed variants across all workloads or native architectures.

| Workload | Before, confirm | Two regions, confirm | Time change, screen → confirm |
| --- | ---: | ---: | ---: |
| Batch 1k | 43.365 µs | 38.220 µs | −11.21% → −11.86%, p=.001/.001 |
| Repeated batch 1024, P1 | 44.792 µs | 39.060 µs | −11.55% → −12.80%, p=.001/.001 |
| Broad batch 8192, P1 | 321.449 µs | 321.337 µs | +0.44% → −0.03%, undetected |
| Checked scalar | 90.02 ns | 88.71 ns | −1.82% → −1.46%, p=.155/.002 |

The two-region confirmation gives about 13.46% more original-batch points/s and
14.68% more repeated-batch points/s. The whole sketch-function control reproduces
the batch gain too. The batch-export-only control instead regresses original
batch by 2.48% and repeated batch by 2.17% in confirmation (both p=.001).
This supports targeting the publication pattern. It does not eliminate placement/
register effects or establish a general per-byte or standalone-scalar benefit.
Narrow scalar improvement is detected only in its own confirmation, not its
screening round. Broad scalar comparisons have no detected gain; the narrowed
candidate has not run the full guard matrix. Absolute latency remains far above M1.

### Streaming controls, review, and next step

At actual P16 with 64-byte fixed writes, the unchanged single-home stream gives
1.242M writes/s in screening and 1.301M in confirmation; sixteen resource homes
give 6.181M and 7.453M. Held-out chunking gives 1.404M/1.391M on one home and
6.811M/7.933M on sixteen homes. These are same-binary synthetic workload controls,
not before/after gains from this Datadog-only probe. Variation from previous
rounds is preserved; it is not evidence of a streaming code regression.

Guest, host, and adversarial reviewers approve diagnostic timing and the retained
harness/evidence scope. Broad retention is deferred because it lacks generic
binding/effect/ownership proofs and executable WAT adversaries. The narrowed
publication pattern is a positive candidate for that implementation round;
see [the production requirements](translation-memory-views.md). No NEW public
contract is proposed or approved in this analysis.

The retained harness/evidence pass `make check` with pinned golangci-lint 2.13.1,
`make race`, `make rust`, `make cross`, `make smoke`, `make fuzz`, and
`make integration`. Two further builds reproduce all three proof packages,
locks, and runtime adapters unchanged. Exact staged-snapshot tests/strict lint
and darna precede the signed local commit. Production qualification of a future
hardener pass remains pending.

## Exact bin reuse: phase 0 no-go (2026-10-11)

Reject both measured default exact-value cache candidates. Retain only the
measurement harness and [raw evidence](measurements/bin-reuse-phase0.txt).
Neither capacity passes the preregistered requirement: the same capacity must
improve complete checked scalar calls by at least 5% on constant inputs **and**
a frozen mixed/phase corpus, versus untouched execution, in both rounds
(p < 0.05). Confirmed median regressions above 1% on miss-heavy, held-out or
original guards also reject a candidate. Smaller detected costs and uncertain
negatives remain reportable. This is a decision about these implementations
as defaults, not a claim that every exact cache or repeated batch is useless.
No production code, generated artifact, lock, public contract or manifest
option changes. No implementation spec is approved or needed for the retained
diagnostic work. M1 performance remains unaccepted.

### Frozen experiment and limits

Baseline is merged `main=origin/main=1510d69`; local work starts on
`task/exact-bin-reuse`. The private diagnostic driver checks the SHA-256 of the
entire pinned translated module, uniquely extracts its existing ln/affine/
floor/saturated-conversion block, and builds temporary Go overlays. It uses
neither host `math.Log` nor copied upstream/reference mapping code. The four
shapes are untouched, extracted/no-cache, extracted/cache1 and
extracted/cache64. These are not retained compiler specializations.

The cache stores only logical i32 indices, keyed by all five u64 words: value
bits plus current gamma, multiplier, minimum and offset bits. Parameter loads
remain inside execution, after unchanged validity/zero-threshold checks.
Every original counter/store/collapse update still executes in order. Tables
are private to a module; replacement creates empty storage. Direct mapping
hashes value bits with two xor-right-33 steps around multiplication by
`0xff51afd7ed558ccd`, then masks by capacity minus one; full-key equality and
an explicit valid flag decide hits. A valid collision replaces the entry.
On amd64 entries occupy 48 B: 48/3,072 B for capacities 1/64, plus a 24 B module
slice field and allocator overhead. Capacity 256 is 12,288 B of entry data.
This host memory is outside guest logical lengths; zero warmed B/op does not
count construction. All extracted variants add the slice field/helper code,
so untouched-to-extracted includes layout, allocation size and placement
effects; it does not isolate a function-call cost.

Go 1.26.8, CGO disabled, GOAMD64=v1, PGO off, default GC/memory, unpinned Ryzen
7 5800HS CPUs 0–15 (8 physical/16 logical). Seven 300 ms samples interleave
roles per leaf, reversing role order each sample. Separate seven 500 ms samples
reverse starting roles, families and leaves on the same frozen binaries.
Hashes remain identical after both rounds. No compilation, lint, fuzz or
reviewer tools overlap timing. New scalar/mapping/matched leaves set, assert
and report actual P1; original serial guards use initial P16. Printed `-16`
suffixes alone are not processor evidence. No samples are filtered. Dispersion
is substantial in several screening cells; p-values are exploratory and
unadjusted across the matrix. An undetected difference is not equivalence.

### Value and bin locality

The 8,192-point corpora are frozen before cache selection/timing. Timed arrays
repeat after a full pass, so uniqueness below is within each finite cycle.
The additional Increasing control and original `SketchAdd` instead continue
through nonrepeating integers; measured N stays below binary64's integer
precision limit. A complete untimed pass warms each new scalar store. Exact
logical bins come from fresh singleton sketches in the unchanged Wasm under
wazero, not collapsed store offsets or approximate host logarithms. Zero
count and logical index zero are distinguished separately by boundary tests.

| Corpus | Distinct values | Logical bins | Adjacent same-bin pairs / 8,191 | Second-pass hits: cap1 / cap64 |
| --- | ---: | ---: | ---: | ---: |
| Repeated (42) | 1 | 1 | 8,191 | 8,192 / 8,192 |
| UniqueCycle (1…8,192) | 8,192 | 377 | 7,815 | 0 / 0 |
| NearbyUniqueCycle (adjacent bits above 42) | 8,192 | 1 | 8,191 | 0 / 0 |
| MostlyUniqueCycle (one hot point per 16) | 7,681 | 3 | 7,167 | 0 / 413 |
| Phases (16 hot integers / unique cold phases) | 4,112 | 18 | 4,092 | 0 / 0 |
| BroadHoldout (unique mantissa/exponent permutation) | 8,192 | 5,373 | 0 | 0 / 0 |
| NegativeBins (existing eight-value cycle) | 8 | 8 | 0 | 0 / 8,192 |

Actual cache probes run outside timing, with first-pass and continuous second-
pass hits, misses, collisions/evictions per 1,024-point phase in the archive.
Capacities 1/16/64/256 are checked. Capacity 64's mostly-unique hit rate is
5.04%, below the nominal 6.25% hot frequency because collisions evict 42.
The 16 hot integers map to only four slots at capacity 64 and thrash, giving
zero hits in hot phases too. Keep this negative; the hash is not rerolled
against the observed corpora. The broad/unique/nearby cycles have zero replay
hits even at capacity 256. These are properties of the frozen policy, not
universal hit-rate bounds for another hash, associativity or capacity.

High adjacent-bin locality supports investigating certified intervals. It
does not certify membership: oracle samples, monotone bit search or approximate
exponential thresholds cannot substitute for a proof of the pinned mapping's
floating-point behavior. An interval design must fall back around boundaries
and uncertified values and retain ordered collapse. None is implemented here.

### Mapping controls and complete checked calls

Raw mapping controls include common indirect method invocation, bypass
ownership/locks/validation/current guest parameter loads, and are not additive
parts of facade latency. Original and hits use 42; OriginalIncreasing and
miss-dominated controls use identical increasing integers. The latter prime
42 once and can have at most one surviving priming hit. The increasing flag
is precomputed outside timing. Screening → confirmation medians (ns/op):

| Control | Screening | Confirmation |
| --- | ---: | ---: |
| Original mapping, 42 | 17.61 | 16.83 |
| Original mapping, increasing | 18.76 | 17.74 |
| Cache1 hit | 12.84 | 11.42 |
| Cache1 miss/fill | 40.32 | 39.30 |
| Cache64 hit | 13.92 | 12.66 |
| Cache64 miss/fill | 40.33 | 39.35 |

The attainable measured raw saving is only about 4–5 ns on 42, while a miss
adds roughly 21–22 ns. A weighted estimate using these matched input classes
needs about 80%/84% hits for capacities 1/64 just to break even. It is not a
full-call ceiling or prediction: instruction layout, direct versus indirect
calls, memory loads and stores interact. The increasing standalone workload
has no exact reuse regardless of hash quality.

Confirmation complete P1 scalar medians, ns/point; all cache miss-heavy losses
listed detect a difference (p ≤ 0.015). Constant cap1's modest gain is p=0.011;
screening detects no constant gain for either capacity.

| Corpus | Untouched | Extracted | Cache1 | Cache64 | Cache64 change: screening / confirmation |
| --- | ---: | ---: | ---: | ---: | --- |
| Increasing | 91.86 | 92.85 | 115.4 | 116.0 | +25.53% / +26.28% |
| Repeated | 91.21 | 92.45 | 88.53 | 91.36 | −3.35% / +0.16% |
| UniqueCycle | 92.39 | 94.02 | 110.8 | 114.7 | +20.02% / +24.15% |
| NearbyUniqueCycle | 91.69 | 94.93 | 112.3 | 115.3 | +27.66% / +25.75% |
| MostlyUniqueCycle | 90.88 | 92.35 | 111.7 | 114.9 | +24.97% / +26.43% |
| Phases | 92.54 | 92.60 | 110.4 | 114.5 | +27.44% / +23.73% |
| BroadHoldout | 91.43 | 94.26 | 116.9 | 120.4 | +29.54% / +31.69% |
| NegativeBins | 90.58 | 95.82 | 111.0 | 91.69 | +5.70% / +1.23% |

The cap64 NegativeBins comparison detects no gain or loss in either round.
Cap1 constant improvement is only 3.60%/2.94%, below the 5% criterion and
unconfirmed in screening. Neither mixed/phase candidate improves. Extraction
alone increases BroadHoldout 3.10% in confirmation (p=0.007), while most other
scalar extraction comparisons detect no difference. Keep the full-shape cost
separate from lookup/fill overhead.
Extraction's Increasing screening cost is 11.46% (p=0.038), without
confirmation (+1.08%, p=0.209). NegativeBins extraction rises 5.78% in
confirmation with benchstat's rounded p=0.050; retain that threshold ambiguity
rather than treating the printed value as proof of p < 0.05.

Original guards and matched batch qualification reject retention too:

| Workload | Confirmation untouched → cache64 | Cache64 change: screening / confirmation | Confirmation p |
| --- | --- | --- | ---: |
| Original increasing scalar | 91.22 → 109.9 ns | +16.42% / +20.48% | 0.001 |
| Original AddMany1k | 43.41 → 57.90 µs | +37.00% / +33.36% | 0.001 |
| Matched repeated N1,024 | 45.18 → 28.96 µs | −32.27% / −35.90% | 0.001 |
| Matched holdout N8,192 | 325.9 → 498.8 µs | +50.08% / +53.05% | 0.001 |
| SQL | 2.175 → 2.244 µs | +0.21% / +3.17% | 1.000 |
| Varied SQL | 2.785 → 2.841 µs | +1.75% / +2.01% | 0.620 |
| Varied scalar | 91.02 → 89.56 ns | +0.84% / −1.60% | 0.318 |
| Negative-bin scalar | 93.01 → 85.43 ns | −10.41% / −8.15% | 0.128 |
| Traces | 2.612 → 2.619 ms | −1.62% / +0.27% | 0.620 |

SQL/varied/trace differences and cap64's confirmation negative-bin gain are
not detected; their point estimates still remain. Cap1 original scalar and
AddMany1k regress 23.22%/34.41% in confirmation, both p=0.001. Its matched
repeated batch gains 35.80%/37.26%, while the matched holdout loses 48.20%/50.21%.
Extraction alone loses 4.94%/7.56% on matched holdout (p=0.002/0.017), and its
repeated-batch screening cost is 2.61% (p=0.038), without confirmation
(p=0.053). Strong repeated-batch gains do not qualify a default scalar cache.

The unchanged streaming binary is shared by all DD-only overlay shapes;
there is no before/after stream implementation difference to test. Fresh P16
64 B fixed/held-out default streams reach 1.275/1.417 million writes/s in
screening, 1.331/1.480 in confirmation. Sixteen-home standalone streams reach
7.555/7.933 then 8.877/10.009 million writes/s. These single-binary controls
show dispersion; they are not cache gains or a resolution of the older default
streaming negative. Existing checked Batch32 remains unchanged.

### Reference, correctness, gates and next work

Rebuild the independent local spike from its hash-recorded unversioned sources:
Rust 1.97.0 `--locked` std/panic_abort, Binaryen 133 meta-DCE/O3, wasm2go 0.4.16
unsafe and its own hardener. Its Wasm again reproduces
`8bda5ff7acff11fd64fd47a0a48165a71d610c28413665e4cc4d4034150d0651`.
Native Go/wazero tests and matched one-iteration Count checks pass. Only corpus
and benchmark code is shared, with no Cocoon implementation transplant or
external source edit. Raw labels/provenance/commands remain in the archive;
only comparison files normalize the reference package label. As in track 1,
the original batch/trace shapes and ABI2 borrowed/background/four-instance
contract differ; new P1 scalar corpora and matched groups use identical inputs.
Fresh original scalar is 57.48 ns reference versus 91.22 ns untouched Cocoon
(+58.70%); SQL is 1.982 versus 2.175 µs (+9.74%). Historical runs with different
harness/binary placement are not paired evidence of an intervening regression.
The real injector benchmark remains unavailable at its original machine path.

Before freeze all three reviewers approve the guest, host and adversarial
controls. Every overlay passes the entire DD test package, including point
validation/errors, late-invalid batch atomicity/recovery, boundary/zero tests
and growth/collapse. New tests check full value/parameter identity, collision
replacement, reset and fresh module isolation. Every corpus compares Count and
full protobuf after two complete ordered passes plus low-bin revisits, through
both standalone scalar and prevalidated batch calls, against original Wasm.
The 5,373-bin holdout necessarily crosses the 2,048-bin store capacity.

`make check`, `race`, `rust`, `cross`, `smoke`, `fuzz` and `integration` pass
using Go 1.27.1, official golangci-lint 2.13.1, Rust 1.97.0 and Binaryen 133.
These gates, including race/fuzz, use unchanged generated modules. The private
overlays receive the full DD test package, not separate race/fuzz qualification.
Cross checks compile rather than execute non-native targets. Race includes
generated consumer subprocesses; fuzz covers parsers, hardening and all
generated wazero adapters plus stateful streaming. Two further builds of each
of the three proofs and locks plus runtime adapter generation reproduce all
57 existing fixture/runtime files and the new harness unchanged. Exact staged
snapshot tests/vet/strict lint and Darna are required before the local signed
measurement commit. Gate/proof logs and source/binary hashes are archived.

Retain the phase-0 harness, decision and evidence in one local measurement
commit. Reject cache1/cache64 and extraction-only as production changes; no
rejected generated module is committed. Next choose whether to specify a
certified interval experiment or pursue translation analysis. Interval work
needs a credible floating-point proof and upstream API or checked compiler
specialization before implementation, original fallback, new holdouts and all
existing correctness/performance guards. Exact-value cache gains on repeated
batches alone do not solve standalone M1. No publication follows from this
round.

## Low-copy numeric inputs: qualified host encoding (2026-10-11)

Retain the private generator stage of track 1. Top-level numeric slices now
encode directly into their checked guest-memory reservation in stateless calls,
resource constructors/methods, and checked commands. The six integer/float types
share the existing checked cursor and bounded view APIs. Argument/aggregate size
checks, UTF-8/record validation order, admission, ownership, per-operation reset
and memory checks, reply validation, and immediate poisoning remain intact.
Rust still decodes an owned vector and validates the entire sketch batch before
mutation. Record codecs remain owned. No public API, ABI, manifest option, or
borrowed guest lifetime contract is added; the [spec](low-copy-batching.md)
precedes implementation in phase-0 commit `b04e04e`.

### Rejected placement and retained placement

[Round 1](measurements/low-copy-round1.txt) interleaved the new helpers with
existing codecs. Its batch gains repeated, but SQL rose 1.07% in confirmation
(2.143 to 2.166 µs, p=0.038), with the same direction in screening. The streaming
holdout P16/64 B rose 1.34% in confirmation (p=0.026). Fixed P16/4 KiB rose 0.54%
in screening (p=0.017), without a detected difference in confirmation. Reject
that candidate before committing it: batch gains and passing safety gates do
not waive the latency guards. All samples, source hashes, and negatives remain.

Round 2 appends the identical private helper definitions after operations and
resources, preserving existing codec/operation relative order. Go 1.26.8 amd64
inspection shows the four SQL function addresses restored to baseline; round 1
had moved them by 352 bytes. Equal bodies and changed placement motivated the
experiment but do not establish cache alignment as the cause. The new placement
qualifies on its own [paired/reversed evidence](measurements/low-copy-round2.txt).
All three reviewers approved the bounded implementation and the ordering delta
before retention. No padding, compiler flags, runtime shortcuts, or isolated
edits to generated code were used.

### Conditions and batch results

Baseline is `main=origin/main=47ae180` with the identical matched harness from
`b04e04e`. Frozen before/after and independently rebuilt spike binaries use Go
1.26.8, CGO disabled, GOAMD64=v1, PGO off, default GC/memory, and unpinned Ryzen
7 5800HS CPUs 0–15 (8 physical/16 logical). Seven 500 ms samples interleave
implementations per leaf, reversing role order on alternate samples. A separate
seven 1 s confirmation reverses the starting role, family and leaf order, without
rebuilding binaries. No compilation, lint, fuzz, or reviewer tools overlap timing.
Actual P1/P16 is set/asserted/reported in new leaves; the initial `-16` suffix
is not evidence of a P16 serial measurement. Original serial guards use initial
P16. P-values are exploratory, unadjusted across the matrix; no samples are
filtered. “No detected regression” does not prove timing equivalence.

The screening matrix covers all five corpora at 1/32/1,024/8,192 points, P1/P16,
one/sixteen homes, the seven original/varied/negative guards, and default/checked
streaming. Confirmation repeats the original guards, repeated/holdout matched
cells, P16 contention, and selected streaming guards including both 64 B holdouts
and fixed P16/4 KiB. Each group repeats after an untimed warmup; distinct values
are unique within a group, not an indefinitely new stream. The collapse oracle
crosses capacity during initial ingestion, not necessarily during timed groups.
Count and full protobuf agreement, invalid-tail atomicity, and recovery are
checked against the original Wasm before timing.

Selected P1 medians (µs/group; all listed batch comparisons p=0.001):

| Corpus / points | Before | After | Screening change | Confirmation change |
| --- | ---: | ---: | ---: | ---: |
| Repeated / 1 | 0.2181 | 0.1893 | −13.20% | −13.34% |
| Repeated / 32 | 1.737 | 1.673 | −3.68% | −4.02% |
| Repeated / 1,024 | 45.99 | 44.29 | −3.70% | −3.24% |
| Repeated / 8,192 | 361.7 | 351.8 | −2.71% | −2.73% |
| Holdout / 1 | 0.2181 | 0.1894 | −13.16% | −13.14% |
| Holdout / 32 | 1.740 | 1.681 | −3.39% | −3.56% |
| Holdout / 1,024 | 41.82 | 40.08 | −4.15% | −4.45% |
| Holdout / 8,192 | 331.6 | 319.6 | −3.61% | −4.75% |

The original 1,000-point batch improves 45.85 to 43.09 µs (−6.00%), then 46.27
to 43.28 µs (−6.46%) in confirmation. One-point `AddMany` is still much slower
than one scalar `Add` (~90 ns); these are batch throughput gains, not a scalar
latency claim. In every matched cell the temporary 8×N-byte Go buffer and its
allocation disappear (one allocation to zero). Rust allocation/copying remains.
The frozen candidate's `PutBytes` assembly checks source/destination equality
and bypasses `memmove` for the same guest view; this is a measured compiler/target
fact, not a portable performance guarantee. All cursor/range checks still run.

At P16 with one home, N32 reaches 18.017 million points/s and N1,024 22.289
million points/s; screening time drops 6.67%/4.78%, confirmed 6.20%/3.57%.
Sixteen homes reach 181.344/240.285 million points/s. The N32 sharded gain is
not detected in confirmation (p=0.053); N1,024 improves 7.61% then 10.28%, with
notable dispersion, so no fixed scaling guarantee follows. Logical guest memory
is unchanged: 1,179,648 B for one warmed home, 18,874,368 B for sixteen; serial
N8,192 uses 1,441,792 B, smaller groups 1,179,648 B. Measured post-warmup growth
is zero. These are logical lengths, not RSS or zero-guest-allocation proofs.

### Standalone and streaming guards

| Workload | Screening before → after | Screening p | Confirmation before → after | Confirmation p |
| --- | --- | ---: | --- | ---: |
| SQL | 2.148 → 2.132 µs | 1.000 | 2.154 → 2.156 µs | 0.620 |
| Scalar Add | 89.10 → 89.65 ns | 0.559 | 89.61 → 89.94 ns | 0.535 |
| Traces | 2.532 → 2.523 ms | 0.620 | 2.565 → 2.557 ms | 0.259 |
| Varied SQL | 2.725 → 2.720 µs | 0.648 | 2.737 → 2.732 µs | 0.438 |
| Varied Add | 89.75 → 89.79 ns | 0.878 | 90.51 → 90.15 ns | 0.902 |
| Negative-bin Add | 90.02 → 90.25 ns | 0.436 | 90.09 → 90.19 ns | 0.735 |

No standalone scalar, SQL, trace, or streaming guard detects a regression in
either retained-candidate run. Streaming has no numeric input and shows no
established benefit from this stage. The fixed P16/4 KiB confirmation still has
a +0.48% point estimate (p=0.053). Fixed P16 BatchHomesP has a +1.8% screening
estimate (p=0.053), not repeated in confirmation (p=0.902); other absent gains
and wide sharded streaming dispersion remain in the archive. This does not
resolve the older default-stream
negative against the pre-session implementation.

For synthetic 64 B streams at P16, the default one home reaches 1.369 million
writes/s (fixed) / 1.544 million (holdout); sixteen homes with standalone calls
reach 10.546/10.587 million, and Batch32 with sixteen homes 9.618/10.463 million.
Confirmation reaches 1.370/1.518 million default, 9.564/9.873 million standalone
sharded, and 8.795/9.979 million Batch sharded. Batch with one home remains
1.184/1.301 million in screening: sessions do not remove serialization. These
are warmed aggregate throughput, not per-request latency. Fixed sharded Batch
still allocates about 49.8 KB/801 allocations per stream versus 17.5 KB/250 for
standalone. The resource-home and Batch tradeoffs from track 3 remain.

### Independent reference, proofs, and next work

The original spike is a hash-recorded local source archive without `.git`.
Fresh Rust 1.97.0 `--locked` std/panic_abort Wasm builds, Binaryen 133 meta-DCE/O3,
wasm2go 0.4.16 unsafe, and its own hardener reproduce Wasm hash
`8bda5ff7acff11fd64fd47a0a48165a71d610c28413665e4cc4d4034150d0651`.
Actual commands, source hashes, native-Go/wazero oracles, and build metadata are
archived. Raw package labels stay intact; only comparison input labels are
normalized for benchstat. New matched P1 harnesses use identical ordered groups.
Original batch values (`i*1.5` versus `i`) and trace shapes (20×50 versus 1×1,000)
differ, and the original ABI2 borrowed decode/mutate and four-instance/background
context contracts differ from Cocoon's owned/prevalidated ABI3 calls. Compare
complete implementations; do not subtract their gap as safety overhead.

Fresh scalar is 55.51 ns reference versus 89.65 ns candidate (+61.50%), SQL
1.945 versus 2.132 µs (+9.61%). Matched reference repeated N1,024/8,192 is
36.28/290.5 µs versus 44.29/351.8 µs (+22.10%/+21.13%); holdout N8,192 is
297.4 versus 319.6 µs (+7.48%). The real injector reference at the original
`/home/bits/inject-bench` path is unavailable on this machine; no fresh
native/translated/wazero real-payload claim follows. **M1 performance remains
unaccepted**, and no default PGO profile is added.

Qualification uses Go 1.27.1, Rust 1.97.0 and strict lint 2.13.1. `make check`,
`make race`, `make rust`, `make cross`, `make smoke`, `make fuzz`, and
`make integration` pass. Integration rebuilds and checks all three
proofs and every shim's native tests/all-feature Clippy. `make fuzz` covers the
parsers/hardener and all three generated wazero adapters plus streaming's own
adapter. Cross checks compile all five targets; they are not native runs.
Compiled generated-consumer fixtures exercise the six types, mixed/unaligned
inputs, exact float bits, constructors/methods and checked commands. Independent
wire goldens and original-Wasm checks cover bounds, malformed tails, retained
owned output, current backing memory, undersized reservations before mutation,
reserve traps/overflow/logical bounds, memory faults, typed errors, and prefix
stop. Two repeated builds reproduce all 57 proof/runtime files and locks byte
for byte. Datadog and streaming Wasm remain byte-identical to baseline; compute
Wasm changes only to add the synthetic numeric proof operations. Retained
commits additionally validate their exact staged snapshots with darna.

Track 1's host stage is qualified. The subsequent exact-bin-reuse measurements
above reject both default cache prototypes; certified intervals or translation
analysis require a separate decision. Guest borrowed/lazy input decoding remains
a separate design and
lifetime/aliasing review requiring operator approval for any new public contract
before implementation. These throughput results do not accept scalar latency
or the remaining guest-decoding cost.

## Low-copy batching: measurements before implementation (2026-10-11)

After all three PRs merged, `main` and `origin/main` matched at `47ae180`.
Track 1 starts with matched inputs and a private checked prototype, before any
generator, runtime, or guest change. [Raw samples, driver, provenance, oracle
checks, and disassembly](measurements/low-copy-phase0.txt) retain every sample.

The serial matrix ingests identical ordered groups through public scalar calls,
public `AddMany`, and the prototype, at 1, 32, 1,024, and 8,192 points. Corpora
cover repeated values, nearby distinct values, integers distinct within each
group, verified nonzero negative bins, and an independent nonmonotonic broad
exponent/mantissa holdout. All groups repeat after one untimed warmup; this is
steady ingestion, not an indefinitely unique stream or a finite growth lifetime.
The Wasm-derived collapse assertion verifies the large holdout crosses the pinned
2,048-bin capacity. Complete protobuf and Count comparisons against ordered
original-Wasm scalar calls cover three repetitions and empty inputs. Prototype
proofs also cover invalid tails, exact errors, healthy recovery, input limits,
closed resources/libraries, and size-error precedence over a closed handle.

Seven isolated 500 ms samples use Go 1.26.8, CGO disabled, GOAMD64=v1, PGO off,
default GC/memory, and unpinned AMD Ryzen 7 5800HS CPUs 0–15. Each measured leaf
sets, asserts, and reports actual GOMAXPROCS, serial P1 or contention P1/P16.
The printed `-16` suffix reflects the initial setting, not a P1 measurement.
Scalar/facade/prototype group order reverses on alternating samples. No builds,
linting, fuzzing, or reviewer tool work overlapped timing. Go 1.26.8 tests and
one-iteration benchmark oracles passed beforehand, as did strict lint 2.13.1.

Selected medians (microseconds per group; no scalar-latency improvement claim):

| Corpus / points | Ordered scalars | Public AddMany | Checked prototype | Prototype vs AddMany |
| --- | ---: | ---: | ---: | --- |
| Repeated / 1 | 0.09297 | 0.2226 | 0.2243 | no detected change, p=0.831 |
| Repeated / 32 | 2.886 | 1.786 | 1.694 | −5.15%, p=0.001 |
| Repeated / 1,024 | 92.241 | 47.089 | 44.421 | −5.67%, p=0.001 |
| Repeated / 8,192 | 737.442 | 372.421 | 353.798 | −5.00%, p=0.001 |
| Nearby / 1,024 | 92.733 | 46.901 | 44.785 | −4.51%, p=0.026 |
| Distinct group / 8,192 | 742.070 | 369.963 | 356.068 | no detected change, p=0.073 |
| Negative bins / 1 | 0.09267 | 0.2251 | 0.2425 | no detected change, p=0.209 |
| Negative bins / 1,024 | 92.714 | 47.550 | 44.961 | −5.44%, p=0.026 |
| Holdout / 1,024 | 90.695 | 43.404 | 40.393 | −6.94%, p=0.001 |
| Holdout / 8,192 | 730.207 | 342.795 | 322.779 | −5.84%, p=0.001 |

Several prototype samples have 20–50% dispersion. Small batches and multiple
corpus/size cells show no detected gain; raw results include them all. These are
exploratory, unadjusted comparisons. Removing one temporary Go allocation saves
8×N B/call in every measured prototype cell; Go allocations become zero. Rust
still clones/decodes owned vectors, so guest execution remains allocating.

Independently timed controls (N=1/32/1,024/8,192) measure encode-only medians
20.93 ns / 80.66 ns / 1.796 µs / 11.55 µs; checked reserve/copy of preencoded
input 37.94 ns / 42.13 ns / 129.0 ns / 1.479 µs; checked direct preparation
51.83 ns / 76.94 ns / 987.9 ns / 7.304 µs; and raw prepared guest execution
94.22 ns / 1.510 µs / 43.42 µs / 347.4 µs. Their preparations, checks, locking,
and state differ: do not add or subtract them into an end-to-end cost model.
Preparation throughput is not ingestion throughput; raw guest calls bypass
supported ownership/execution checks.

The prototype preserves admission, owner resolution, execution locking,
per-operation state/memory checks, canonical reply validation, and poisoning.
It obtains a checked reservation cursor with `PutBytes(nil)`, borrows its current
bounded memory using `Range`, writes little-endian floats, and advances actual
reservation usage with `PutBytes` of that same view. The frozen Go 1.26.8 amd64
`PutBytes` assembly compares source/destination and skips `runtime.memmove` when
equal. This verifies the measured copy bypass, not a cross-toolchain performance
guarantee; the extra range/accounting work remains in the prototype. Inspection
used Go 1.24.1 objdump because the installed Go 1.26 package lacks that tool.

Contention uses one warmed resource per worker, with encoding outside the
instance lock in the facade and inside it in the prototype. At P16, one home
limits N32 to 16.255 vs 17.174 million points/s and N1,024 to 20.868 vs 21.816
million points/s (−5.38% and −4.34% time/op, p=0.001/0.002). Sixteen homes reach
139.462 vs 149.061 million points/s at N32 and 166.589 vs 184.086 at N1,024;
neither sharded timing difference is detected (p=0.073/0.165). Logical guest
memory is 1,179,648 B for one home and 18,874,368 B for sixteen, with measured
zero growth after warmup. Serial N8,192 uses 1,441,792 B; smaller groups use
1,179,648 B, also with zero warmed growth. These are logical lengths, not RSS
or proofs of zero guest allocation.

The measured ceiling justifies a small host-only generator experiment for all
six top-level numeric slice types using existing checked reservation APIs.
Keep Rust owned decoding, record codecs, standalone scalar/SQL bodies, and the
public contract unchanged. Retention still requires fresh paired full-facade,
scalar/SQL/trace/streaming holdouts and independent spike comparisons, fault and
growth proofs, all quality gates, and repeat builds. No production gain or M1
acceptance is established by this phase.

## Default streaming cost investigation (2026-10-11)

After PR #1 merged at `0a5969e`, the first follow-up investigates the repeated
default fixed P1/64 B slowdown. It retains diagnostic benchmarks and evidence;
there is **no production optimization or new public contract**. Default
standalone streams with ResourceInstances=1 bear the repeated fixed-stream cost;
home-sharding throughput gains require opt-in. **M1 performance is not accepted**.

The [raw archive](measurements/default-streaming-phases.txt) contains all samples,
drivers, commands, source/binary hashes, build metadata, and summaries. Conditions
match streaming qualification: Go 1.26.8, CGO disabled, PGO off, GOAMD64=v1,
Ryzen 5800HS, CPUs 0–15 without affinity, and default GC/memory settings. Each
leaf sets and asserts actual P1 despite the process's initial `-16` suffix.
Compilation, checks, fuzzing, profiles, and reviewer tools do not overlap timing.
The baseline remains Phase 0 `36a8e6f`, with production unchanged from `9623228`.
Both sides use the same standalone benchmark, Wasm, and translated guest.

### Frozen full-stream reproduction

Seven alternating 700 ms pairs screen fixed/holdout inputs at 64 B and 4 KiB;
seven reversed-order 1 s pairs confirm 64 B. After diagnostic measurements,
another seven 1 s pairs repeat the original frozen binaries.

| Workload | Frozen screen | Reversed confirmation | Frozen repeat after diagnostics |
|---|---:|---:|---:|
| Fixed 64 B | 157.2 → 160.4 µs, +2.01%, p=0.007 | 156.5 → 161.2 µs, +3.01%, p=0.001 | 157.7 → 161.6 µs, +2.50%, p=0.004 |
| Holdout 64 B | 197.0 → 196.3 µs, p=0.710 | 197.9 → 198.1 µs, p=0.902 | 197.9 → 197.3 µs, p=0.209 |
| Fixed 4 KiB | 5.137 → 5.139 ms, p=1.000 | — | — |
| Holdout 4 KiB | 5.468 → 5.423 ms, p=0.209 | — | — |

The fixed 64 B cost repeats; these sets detect no holdout or 4 KiB regression.
No detected difference is not equivalence. Allocation counts are unchanged.

### Bounded phase controls

`BenchmarkStreamPhases` uses identical source with both facades and the ordinary
checked APIs. NewClose measures a warm complete resource lifetime. Write controls
prime a full 1 KiB window, either before or after injection, then retain only
the trailing 32 owned replies. End and abandoned Close measure groups of 32
fresh resources. Preparation and verification are untimed; cleanup follows
timed End, while abandoned Close times disposal directly. End states cover
empty/short/full windows, prior injection, an entire match, and a partial match
at the tail. Independent whole-body oracles validate preparation and End; untimed
different replies overwrite guest storage to check retained-result ownership.
Live resources never exceed 32, independent of b.N. Close warm-up abandons
pending data without End.

Both binaries pass a 1x oracle run before seven alternating 250 ms pairs.
The following End/Close times divide the raw group ns/op by 32; NewClose and
Writes already measure one call/lifetime. These controls are **not additive**:
priming, output retention, live resources, timer transitions, and GC differ
from a complete stream.

| Control | Baseline | Merged | p-value |
|---|---:|---:|---:|
| NewClose | 551.5 ns | 564.0 ns | 0.128 |
| Write, uninjected, 64 B | 698.9 ns | 692.9 ns | 0.318 |
| Write, uninjected, 4 KiB | 23.60 µs | 23.35 µs | 0.259 |
| Write, uninjected, 16 KiB | 92.98 µs | 92.33 µs | 0.318 |
| Write, injected, 64 B | 637.3 ns | 624.2 ns | 0.012 |
| Write, injected, 4 KiB | 20.72 µs | 19.87 µs | 0.001 |
| Write, injected, 16 KiB | 79.84 µs | 78.18 µs | 0.004 |
| End, empty | 127.4 ns | 124.0 ns | 0.053 |
| End, short | 255.5 ns | 250.5 ns | 0.364 |
| End, uninjected full window | 5.922 µs | 5.881 µs | 0.805 |
| End, injected full window | 5.047 µs | 4.972 µs | 0.165 |
| End, whole match at tail | 239.4 ns | 232.8 ns | 0.053 |
| End, partial match at tail | 5.872 µs | 5.919 µs | 0.535 |
| Close, abandoned full window | 98.56 ns | 98.13 ns | 0.596 |

No phase detects a slowdown in this newly linked comparison. Some injected
Write controls favor the merged version; they do not establish a production
gain. Host Go allocations match: NewClose uses 200 bytes/8 allocations per
lifetime, each steady Write copies its chunk into one owned allocation,
nonempty End copies its tail into one allocation, and empty End/Close allocate
nothing. This does not count Rust/Wasm allocator activity.
Every control reports zero post-warm guest growth. Warm linear memory is
1,179,648 bytes except the 16 KiB Write controls, which use 1,245,184 bytes.
Write growth is sampled before untimed End. Grouped growth covers the
whole diagnostic, including untimed preparation, ownership probes and disposal;
it is neither an End-only measure nor a resident-memory bound.

### Adding benchmark source changes the comparison's sign

Seven reversed-order 700 ms full-stream pairs in the diagnostic binaries give
fixed 64 B 162.2 → 159.2 µs (−1.85%, p=0.038), fixed 4 KiB 5.215 → 5.114 ms
(−1.94%, p=0.017), holdout 64 B 203.1 → 198.5 µs (−2.24%, p=0.026), and holdout
4 KiB 5.438 → 5.401 ms (p=0.383). The holdout candidate has 21% dispersion.
This is one exploratory set, without independent confirmation of a gain.
P-values are unadjusted across exploratory comparisons.
Removing only the new benchmark source and rebuilding reproduces **both frozen
binaries byte for byte**. The post-diagnostic frozen repeat still detects the
fixed slowdown. Consequently, adding test source can change the measured sign
without a production-source change. Link placement and allocation/GC effects
are hypotheses, not an established cause or a fix.

Source review finds unchanged standalone Write/End/Close bodies, Resource and
Lifecycle paths, and original Instance Call/ResultView bodies and footprint.
Normalized frozen instructions agree for outer End, Close, execution/reply
validation, resource use, and guest writes; Write and End's callback differ
only in option-field offsets. Normalization masks unnamed data relocations;
it does not prove equal placement, referenced data, metadata, or performance.
Resource construction has the new home-selection machinery; selection is not
repeated on each write. These observations narrow the search but do not
attribute the complete-stream delta to one check or constructor cost.

Retain the diagnostic controls; retain no speculative production rewrite from
this round. The default fixed-stream negative remains unresolved and visible.
Future candidates need frozen end-to-end guardrails and holdouts even if their
phase controls improve. Resume low-copy batching with matched scalar/batch input
corpora; refresh the independently rebuilt spike before any M1 acceptance claim.

### Qualification

Guest, host, and adversarial reviewers checked the harness, raw evidence, and
conclusions before retention. `make check` (tests, vet, 49 strict linters with
official golangci-lint 2.13.1), `make race`, `make rust`, `make cross`, `make
smoke`, `make fuzz`, and `make integration` pass with Go 1.27.1/Rust 1.97.0/
Binaryen 133. Smoke initially failed because the sandbox denied Go module
verification network access; the authorized network retry passed. Two further
builds of all three proofs plus runtime generation reproduce all 54 existing
tracked proof/runtime files byte for byte, including their locks. A Go 1.27.1
race-instrumented 1x run exercises every new benchmark's oracle and ownership
checks; its timings are validation only, excluded from performance results.
The retained change adds only diagnostics and documentation, with no generated
artifact or lock delta. Darna and exact staged-snapshot tests gate its commit.

## Streaming qualification harness correction (2026-10-11)

The first end-to-end streaming run and the stopped private-layout candidate
used parent-only GOMAXPROCS setters. Go 1.26's `testing.processBench` resets
GOMAXPROCS to its CPU-list value before each measured leaf: these binaries
started with 16 processors, so P1/P4 labels did not describe actual timing.
All streaming qualification conclusions from these runs are excluded, without
relabeling lower-P samples. Heap diagnostics have no P-labelled mismatch but
are superseded too. The corrected harness sets/restores GOMAXPROCS inside each
measured leaf, checks it before and after RunParallel, and reports actual-procs.
The measurement runner must reject a missing or mismatched metric. Identical
corrected standalone source is used on both sides of the new comparison.

The [historical raw archive](measurements/checked-sessions-streaming-before-layout.txt)
preserves the original harness, hashes, samples, and stopped candidate (exit
130). Its independent DD scalar/SQL/batch/trace pairs remain valid: they never
used parent processor setters. The pre-layout negative-bin slowdown repeated
at 0.64% (p=0.036) and 0.97% (p=0.017), so that shape failed standalone
qualification. Phase 0 and R8 runtime measurements remain valid: they set
GOMAXPROCS externally per process, without P-labelled parent setters.

An earlier attempt also overlapped the final race check. It was stopped after
about 26 seconds and all its samples were discarded before restarting in
isolation. The same archive preserves that excluded attempt separately. No
overlapping sample contributes to an accepted measurement.

## Bounded resource streaming qualification (2026-10-11)

The operator-approved [contract](checked-sessions.md) is implemented: finite
opt-in typed command buffers share admission, ownership, and execution locking;
resources can use a separate bounded set of lazy homes. Each command retains
reset, memory checks, reply validation, owned publication, and immediate fault
poisoning. Handles remain attached to their exact home/epoch after poisoned-slot
replacement. Qualification supports **opt-in concurrent streaming**, with the
negative default-path and Batch results below. **M1 performance is not accepted**.
This is a synthetic exact-byte matching payload; performance of the real
Injector and its dd-trace embedding remains separate follow-up work.

### Conditions and comparisons

Measurements use Go 1.26.8, CGO disabled, PGO off, GOAMD64=v1, Ryzen 5800HS,
CPUs 0–15 without affinity, and default GC/memory settings. System Go 1.27.1
and official golangci-lint 2.13.1 qualify code. Builds, checks, fuzzing, and
reviewer tools stop before each measurement. Every stream row reports verified
actual-procs; Go's initial `-16` name suffix is not the leaf's processor count.
The corrected screen takes 155.4 seconds; the option/diagnostic sweep takes
497.2 seconds, with a separate reversed-order standalone confirmation.

The baseline is the Phase 0 snapshot (`36a8e6f`, production unchanged from
`9623228`). Its original CLI builds the same authored streaming shim with the
host-only checked_batches flag omitted. Baseline and candidate Wasm SHA-256:
`9141023ca801ef1cb0825ea155dd42b4d83bcee9ce6c9a6ad453b40badb49955`.
The corrected standalone benchmark is copied unchanged to the baseline:
`4e98e93d5658fae33558b9af70d208031da25bd8cf0d9c59af13e2234ee86b59`.
This compares complete facades around identical guest code.

Seven alternating 500 ms DD pairs cover scalar, SQL, batch, trace, varied
scalar/SQL, and negative bins; a second seven-pair set reverses the first order.
Two seven-pair 250 ms stream sets cover fixed and independent holdout corpora
at P=1/4/16, 64 B/4 KiB/16 KiB chunks. Seven option sets compare N32, then
N=16/64/256 at P16. Options run in fixed within-process order. Three diagnostic
rounds cover mixed stateless traffic (250 ms), competition (500 ms), and
initialized memory (3 iterations). Short windows, GC, frequency effects,
shared-machine conditions, fixed ordering, and repeated exploratory comparisons
limit inference; p-values are unadjusted. Absence of a detected difference is
not an equivalence proof.

A timed operation is a complete stream: New, all writes, retained owned outputs,
End, and Close. Open, lazy initialization, inputs/oracles, and full byte checks
occur outside timing. Fixed input has 256 equal chunks. Holdouts rotate four
corpora with no match, crossing/repeated matches, partial tails, binary bytes,
and uneven/empty partitions; actual write counts vary. Paths retain one finite
output group without history proportional to b.N. Input reaches 4 MiB at the
largest size. Batch includes command construction, plans, decoding/boxing,
and owned results, with MaxCalls=N and 16 MiB input/output budgets. Parallel
ns/op is aggregate stream throughput, not individual caller latency.

The [complete qualification log](measurements/checked-sessions-streaming.txt)
contains every sample, source/binary hashes, exact drivers/commands, independent
repeats, and summaries. The [historical archive](measurements/checked-sessions-streaming-before-layout.txt)
retains invalid processor-labelled sets, earlier scalar negatives, and excluded
procedural attempts. Those stream sets do not support qualification.

### Standalone guardrails and rejected shapes

The final R12 shape restores the original Call and ResultView bodies and
Instance footprint. One authored template generates locked/held-lock adapters
in the original Call region, with reproduction and statement-equivalence tests;
strict lint still covers the authored runtime file. Batch snapshots validated
wire length before status interpretation/typed decoding, keeping its accounting
out of standalone ResultView. The private Library restores the original
lifecycle offset and places new homes/cursor after it. Options offsets change;
these are implementation choices, not proof of cache placement or the cause of
earlier slowdowns.

| Workload | First baseline → R12 | Independent baseline → R12 |
|---|---:|---:|
| SQL | 2169 → 2190 ns, p=0.066 | 2169 → 2180 ns, p=0.090 |
| Scalar | 88.64 → 88.18 ns, p=0.002 | 88.77 → 88.70 ns, p=0.902 |
| Batch 1k | 44.78 → 44.59 µs, p=0.011 | 44.75 → 44.65 µs, p=0.165 |
| Trace 1k | 2.617 → 2.625 ms, p=0.902 | 2.623 → 2.607 ms, p=0.902 |
| Negative bins | 89.31 → 89.02 ns, p=0.128 | 89.57 → 89.16 ns, p=0.535 |
| Varied scalar | 89.28 → 89.36 ns, p=0.435 | 89.62 → 89.00 ns, p=0.128 |
| Varied SQL | 2809 → 2786 ns, p=0.015 | 2806 → 2795 ns, p=0.247 |

Neither set detects a scalar/SQL/batch regression. Allocation counts are
unchanged. SQL medians remain roughly 0.5–1% higher, and the confirmation
candidate includes a 21% SQL dispersion: a small cost is not excluded.
No standalone scalar improvement or blanket no-regressions claim is made.

Default P1/64 B fixed streams regress 155.7 → 160.4 µs (+3.00%, p=0.001),
confirmed at 155.7 → 160.5 µs (+3.05%, p=0.002), with unchanged allocations.
This negative remains part of the retained opt-in throughput trade-off.
P1/16 KiB holdout is +0.67% (p=0.004) in the first set, but not repeated
(21.00 → 20.95 ms, p=0.383). The option sweep's P1/64 B sharded-standalone
holdout is +2.20% (p=0.011); it has no independent confirmation. Other held-out
negative point estimates likewise are not independently confirmed. All point
estimates and samples are retained. Sharding does not promise a
single-stream/default latency improvement.

Additional uncommitted shapes were rejected:

| Shape | Reason | Evidence |
|---|---|---|
| Layout alone | P1/64 B holdout +1.56%, repeated +2.75%; fixed +2.23% in confirmation | [raw samples/patch](measurements/checked-sessions-streaming-layout-rejected.txt) |
| R9, batch-local wire accounting with helper core | Multiple standalone stream regressions of 1–3%; not qualified | [raw samples/patch](measurements/checked-sessions-r9-rejected.txt) |
| R10, cold state predicate and lifecycle prefix | Fixed P1/64 B +3.75%, repeated +4.72%; holdout +0.86% in second set | [raw samples/patch](measurements/checked-sessions-r10-rejected.txt) |
| R11, shared template in separate runtime file | Negative-bin scalar +2.74%; fixed P1/64 B +2.21% | [raw samples/patch](measurements/checked-sessions-r11-rejected.txt) |

R1–R7 and earlier R8 foundation/harness evidence remain below. Rejected designs
are not retained commits or cumulative speedups.

### Streaming ceilings

Medians in **million writes/s**, N32. Old/default and new/default use one home.
Sharded standalone and sharded Batch use P homes; the separate stateless pool
is configured to one and stays unused. P1's final Batch column reuses BatchHomes1.

| Corpus | P | Chunk | Old/default | New/default | Sharded standalone | Batch, one home | Batch, P homes |
|---|---:|---:|---:|---:|---:|---:|---:|
| Fixed | 1 | 64 B | 1.64420 | 1.59633 | 1.59950 | 1.34171 | 1.34171 |
| Fixed | 1 | 4096 B | 0.05054 | 0.05061 | 0.05075 | 0.04977 | 0.04977 |
| Fixed | 1 | 16384 B | 0.01295 | 0.01290 | 0.01278 | 0.01266 | 0.01266 |
| Fixed | 4 | 64 B | 1.44998 | 1.46013 | 4.94357 | 1.23452 | 4.33661 |
| Fixed | 4 | 4096 B | 0.04893 | 0.04916 | 0.19740 | 0.05012 | 0.19342 |
| Fixed | 4 | 16384 B | 0.01253 | 0.01250 | 0.04905 | 0.01268 | 0.04879 |
| Fixed | 16 | 64 B | 1.42854 | 1.44544 | 11.16153 | 1.22629 | 10.36181 |
| Fixed | 16 | 4096 B | 0.04723 | 0.04784 | 0.46550 | 0.04995 | 0.46699 |
| Fixed | 16 | 16384 B | 0.01249 | 0.01239 | 0.11478 | 0.01267 | 0.11729 |
| Holdout | 1 | 64 B | 1.77818 | 1.77923 | 1.73991 | 1.43140 | 1.43140 |
| Holdout | 1 | 4096 B | 0.06745 | 0.06787 | 0.06696 | 0.06572 | 0.06572 |
| Holdout | 1 | 16384 B | 0.01759 | 0.01747 | 0.01753 | 0.01713 | 0.01713 |
| Holdout | 4 | 64 B | 1.61222 | 1.60694 | 5.41892 | 1.38019 | 4.93383 |
| Holdout | 4 | 4096 B | 0.06573 | 0.06511 | 0.25852 | 0.06667 | 0.25834 |
| Holdout | 4 | 16384 B | 0.01713 | 0.01711 | 0.06492 | 0.01743 | 0.06327 |
| Holdout | 16 | 64 B | 1.58146 | 1.60226 | 11.81841 | 1.34872 | 11.60406 |
| Holdout | 16 | 4096 B | 0.06486 | 0.06466 | 0.59915 | 0.06696 | 0.59580 |
| Holdout | 16 | 16384 B | 0.01713 | 0.01707 | 0.15436 | 0.01729 | 0.15277 |

At P16/64 B, sharding raises 1.429 to 11.162 Mwrites/s (7.81×); sharded
Batch32 reaches 10.362 (7.25×). The independent holdout reaches 11.818
standalone and 11.604 batched versus 1.581 old/default. P16/4 KiB fixed
sharded standalone/Batch reach 0.466/0.467 versus 0.0472 old/default.
The gain is primarily placement: translated guest calls run concurrently.

Typed Batch alone loses roughly 15–20% on 64-byte cases. Fixed 64 B/N32
allocates 49.84 KB and 801 allocations per stream versus 17.50 KB and 250
standalone. Reusing three adapter closures per buffer avoids three per command,
but typed command/result boxing and plan creation remain real costs. Runtime
admission ceilings do not establish a small-write facade gain. Prefer sharded
standalone writes for this synthetic small-chunk workload.

P1 batching also loses on larger chunks. Some larger shared-home P4/16 cases
have only small point-estimate gains; P4 sharded Batch generally trails sharded
standalone, including the 16 KiB holdout. At P16 fixed 4/16 KiB Batch is slightly
higher, while holdouts are slightly lower. These are workload-specific choices,
not a general Batch recommendation or per-byte translation improvement.
The external inject-bench checkout is absent here; its supplied 1–2 µs/call
facade cost and ~11× translation overhead are sanity references, not reproduced
or directly comparable measurements. Phase 0 admission primitives in tens of
nanoseconds are a small share of those payload costs.

### Length, contention, and memory negatives

N16/64/256 does not establish monotonic sharded gains. Full length samples are
in the log; larger buffers cannot be chosen from the runtime ceiling alone.
Mixed 4 KiB holdout traffic performs one owned stateless Echo per output group
through a separately warmed P-slot pool. At P16/N32, sharded standalone reaches
0.608 Mwrites/s and 19,925 Echo/s; sharded Batch reaches 0.589 and 19,294,
versus one-home Batch 0.0681 and 2,231. These are coupled rates, not independent
stateless latency. Echo/write is 0.03276 at N32 and 0.005461 at N256; raw
rates must not hide the changed mix.

A dedicated 64-byte standalone probe competes with buffered streams on 4 KiB
holdouts. Timings include scheduling and the clock. Calls begun while workers
are active enter a trailing 1024-sample ring, including a final slow waiter.
Quantiles below are medians of three diagnostic runs, not whole-window
latency distributions or guarantees; maxima are medians of each run's
retained-sample maximum.

| P16 homes | N | Probe p95 | Median of retained maxima | Retained samples |
|---|---:|---:|---:|---:|
| 1 | 16 | 3.99 ms | 4.74 ms | 540 |
| 1 | 32 | 7.82 ms | 9.11 ms | 132 |
| 1 | 64 | 14.72 ms | 15.82 ms | 163 |
| 1 | 256 | 36.95 ms | 47.63 ms | 59 |
| 16 | 16 | 1.49 µs | 4557.50 µs | 1024 |
| 16 | 32 | 1.09 µs | 3.18 µs | 1024 |
| 16 | 64 | 0.75 µs | 4.80 µs | 1024 |
| 16 | 256 | 1.45 µs | 10.53 µs | 1024 |

Larger shared buffers materially increase competing-call latency. Sharded tails
are nonmonotonic, and a limited ring cannot prove fairness. P1 also has scheduler
stalls measured in milliseconds. N32 bounds work, not latency; responsiveness
requires smaller-buffer experiments and workload-specific measurement.

Initialized-memory medians after GC, outputs released, live wrappers kept alive.
The configured one-slot pool stays lazy/uninitialized. MB is decimal; heap
observations are not process RSS or hard bounds.

| Initialized homes | Live streams | Go heap delta | Logical guest memory |
|---|---:|---:|---:|
| 1 | 1 | 1.403 MB | 1.245 MB |
| 1 | 256 | 1.810 MB | 1.704 MB |
| 4 | 4 | 5.610 MB | 4.981 MB |
| 4 | 256 | 5.663 MB | 5.243 MB |
| 16 | 16 | 22.439 MB | 19.923 MB |
| 16 | 256 | 22.489 MB | 19.923 MB |

Sharding trades memory for concurrency. Bounded lazy homes are not one instance
per stream; uneven lifetimes can overload a lane. Per-instance MaxMemory,
temporarily retiring homes, inputs, typed metadata, and host decode overhead
remain separate from wire-output budgets.

### Proof and qualification

The synthetic fixture retains at most 1 KiB between writes, injects once before
the first exact `</head>`, flushes End, and exercises expected errors/traps.
Write working buffers and linear memory exceed that retained window. Native
whole-input oracles, partial-match splits, translated contract/typed tests, bulk-memory proofs, and
valid-handle wazero sequences/fuzzing cover the stateful case. Middle failures
prove Attempted and exact publication, including post-decode memory faults,
exhausted budgets, cancellation, abandonment, aliases, replacement, sibling
survival, and draining Close. Generated differential comparisons check status
as well as reply bytes.

Final gates and retained commits are recorded in the roadmap. Cross checks
compile arm64/386/Darwin/Windows/js targets; they do not claim native execution.
Race instruments actual compiled generated consumers. Both existing proofs and
the streaming fixture rebuild from authored sources/locks; translated modules
are never optimized alone. All requested gates pass on the final R12 tree;
the approved contract is retained under the narrow throughput guidance above.

## Checked resource bridge qualification (2026-10-11)

This section records the earlier runtime-foundation rounds, superseded by the
complete translated-stream qualification above. The [approved contract](checked-sessions.md)
followed Phase 0. Qualification compares frozen binaries against the original standalone API;
the throughput feature must preserve its latency. These exploratory runtime
ceilings do not accept M1 performance.

Two checked-core shapes were rejected before retaining implementation. R1
factored `Instance.Call` into an inline wrapper and a shared boundary taking an
optional-lock flag, and factored `ResultView` into a wrapper over a reply helper.
Seven alternating baseline/candidate pairs show scalar 88.69 → 90.25 ns
(+1.76%, p=0.006), SQL 2167 → 2212 ns (+2.08%, p=0.017), negative bins
89.29 → 91.02 ns (+1.94%, p=0.012), varied scalar 89.33 → 91.71 ns
(+2.66%, p=0.001), and varied SQL 2802 → 2832 ns (+1.07%, p=0.017).
Batch and trace have no detected difference. No allocation count changed.
The [complete R1 samples](measurements/checked-sessions-r1-rejected.txt) include
its bridge throughput measurements and frozen binary hashes.

R2 restored the original successful `ResultView` path, tracking only validated
non-OK wire bytes in reset call state. It still regressed scalar 88.70 →
90.26 ns (+1.76%, p=0.001), negative bins 88.81 → 90.41 ns (+1.80%, p=0.001),
and varied scalar 89.40 → 90.62 ns (+1.36%, p=0.001). SQL, batch, traces,
and varied SQL have no detected difference; allocation counts are unchanged.
The [complete R2 samples](measurements/checked-sessions-r2-rejected.txt) preserve
this negative result. R1/R2 are uncommitted candidate shapes against `bfc77f4`;
neither is an accepted implementation or speedup.

R3 removed the conditional locking branch by putting unconditional lock/unlock
in standalone Call and invoking the shared full boundary in another frame.
It regressed scalar 88.65 → 92.13 ns (+3.93%, p=0.001), negative bins
89.16 → 93.22 ns (+4.55%, p=0.017), and varied scalar 89.47 → 92.39 ns
(+3.26%, p=0.001), despite decreases in batch (-0.39%) and varied SQL (-1.35%).
SQL and trace have no detected difference. This shape is also rejected; the
[complete R3 samples](measurements/checked-sessions-r3-rejected.txt) retain all
seven alternating pairs and binary provenance.

R4 shared inlineable state/reset/memory/protocol helpers, with the recovery
method directly deferred by the two small ordering adapters. It regressed
scalar 88.72 → 90.74 ns (+2.28%, p=0.001), negative bins 88.96 → 91.44 ns
(+2.79%, p=0.001), and varied scalar 89.56 → 91.56 ns (+2.23%, p=0.001).
SQL (-0.92%), traces (-2.08%), and varied SQL (-2.39%) decreased; batch has no
detected difference. This shape is rejected too. The
[complete R4 samples](measurements/checked-sessions-r4-rejected.txt) retain the
negative scalar outcome alongside those decreases.

R5 restored the original direct recovery closure, sharing the cold fault path
instead of calling a deferred method on success. No difference is detected for
scalar (89.99 → 88.88 ns, p=0.274), SQL, batch, traces, or negative bins. Varied
scalar has a small negative result (89.18 → 89.49 ns, +0.35%, p=0.023), while
varied SQL decreases 1.14%. The [complete R5 samples](measurements/checked-sessions-r5.txt)
include the baseline scalar's wider dispersion. This round was refined further
before committing; it supports no standalone latency improvement claim.

R6 made every reset field explicit and skipped state-error construction when
open. Scalar 88.68 → 88.95 ns (+0.30%, p=0.041), negative bins 88.96 →
89.80 ns (+0.94%, p=0.002), and varied scalar 89.42 → 90.04 ns
(+0.69%, p=0.011) have small negative differences. SQL, traces, and varied SQL
decreased; batch has no detected difference. The [complete R6 samples](measurements/checked-sessions-r6.txt)
preserve these negatives; no standalone latency benefit is claimed.

R7 changed the state predicate so healthy calls fall through and restored the
original scalar-before-pointer reset order. Scalar (88.87 → 89.00 ns,
p=0.302) and varied scalar have no detected difference, but negative bins
89.29 → 90.62 ns (+1.49%, p=0.026) regress. SQL, batch, and varied SQL decrease;
traces have no detected difference. The [complete R7 samples](measurements/checked-sessions-r7-rejected.txt)
retain this rejected shape and its negative holdout.

### Retained runtime foundation (R8)

The original R8 commit preserves Call layout and reset. Batch-only validated error
wire bytes live privately on Instance and reset before each command, without
adding standalone successful-call stores. Shared inlineable helpers own state,
reset, memory, and protocol policy; the small locked/unlocked ordering adapters
recover directly in deferred closures and share the cold fault classifier.
Every command retains its memory check, reset, reply validation, and immediate
poisoning. This is a runtime bridge; typed facade commands, optional lazy homes,
and the translated streaming fixture were not yet implemented at that stage.
R12 replaces the helper/Instance-ledger implementation with the shared authored
template and batch-local accounting described above. R8 timings remain evidence
for that historical foundation, not the final typed streaming implementation.

Seven alternating baseline/R8 pairs detect no standalone scalar regression
(88.64 → 88.86 ns, p=0.330), SQL (2180 → 2176 ns, p=0.689), batch
(45.02 → 44.79 µs, p=0.383), traces (2.612 → 2.596 ms, p=0.053), negative
bins (88.85 → 89.07 ns, p=0.165), or varied scalar (89.21 → 89.23 ns,
p=0.929). Varied SQL decreases 0.99% (p=0.038), with a 22% baseline
dispersion. Allocation counts are unchanged. These are standalone guardrails,
not a scalar latency improvement claim or equivalence proof. All comparisons
are exploratory, unadjusted for multiple candidate rounds; final end-to-end
qualification needs independent paired samples.

Runtime medians in Mcalls/s, with plans reused and full bridge checks:

| GOMAXPROCS | Chunk | One home, standalone | One home, Batch32 | Worker-private, standalone | Worker-private, Batch32 |
|---|---:|---:|---:|---:|---:|
| 1 | 64 B | 7.217 | 7.971 | 7.167 | 7.952 |
| 4 | 64 B | 5.694 | 7.817 | 14.202 | 21.760 |
| 16 | 64 B | 4.858 | 7.404 | 5.704 | 34.813 |
| 1 | 4 KiB | 1.527 | 1.484 | 1.533 | 1.510 |
| 4 | 4 KiB | 0.960 | 0.957 | 1.391 | 1.259 |
| 16 | 4 KiB | 0.746 | 0.926 | 1.605 | 1.682 |
| 1 | 16 KiB | 0.426 | 0.399 | 0.427 | 0.395 |
| 4 | 16 KiB | 0.322 | 0.299 | 0.451 | 0.522 |
| 16 | 16 KiB | 0.271 | 0.324 | 0.452 | 0.561 |

At 16 processors, the shared-home 64-byte bridge gains 52% throughput, while
private instances with batching reach 7.17 times that standalone shared ceiling.
The private-instance path remains a diagnostic placement hypothesis, not the
bounded lazy-home implementation. At 4 KiB the respective rates are 0.746,
0.926, and 1.682 Mcalls/s, with far less admission-amortization headroom.
Negative results matter: P1 4/16 KiB batching and P4 shared 16 KiB batching are
slower than standalone; P4 private 4 KiB batching also loses throughput.
Retaining one buffer of results changes live allocation/GC behavior versus
the prototype. Generated typed plans and complete stream lifetimes must be
measured before recommending either option.

The [complete R8 sample log](measurements/checked-sessions-runtime.txt) records
binary/source hashes, every sample, exact commands, and limitations. It includes
zero-byte controls and the checked prototype comparators. Actual command
limits, contexts, byte budgets, owned staging, prefix accounting, and per-call
checks are included; typed command construction and New/End/Close are excluded.
Integer allocation metrics can show 0/call or 31/32-command buffer because the
first holdback replies are empty; these paths are not allocation-free.
No measurements overlap checks. R8 passed strict check/race and both proof
rebuilds retain the existing artifacts/locks. M1 performance remains unaccepted.

Both rounds use Go 1.26.8, CGO disabled, PGO off, GOAMD64=v1, the Phase 0
Ryzen machine without affinity, GOMAXPROCS=16, seven 500 ms samples per workload,
alternating pair order. All builds, checks, and reviewers completed before
timing. Raw files include every sample. Short windows, frequency variation,
GC, and shared-machine conditions still limit inference.

R1's actual runtime bridge, including context and byte budgets, reaches
7.37 Mcalls/s at GOMAXPROCS 16 for 64-byte writes on one home versus 5.25 for
standalone calls; worker-private instances reach 35.15. For 4 KiB these are
0.940, 0.775, and 1.804 Mcalls/s. However, the reusable plans exclude generated
typed-plan construction and complete stream lifetimes. The bridge also retains
one buffer of owned outputs per worker (up to 512 KiB per worker at 16 KiB
chunks), whereas the Phase 0 loop discards copies immediately. P1 4 KiB bridge
is slower than standalone (663 versus 651 ns), and P4 16 KiB shared bridge is
slower (3350 versus 3128 ns). Gains do not justify the standalone regressions;
these measurements explain the rejected candidate, not a retained feature.

## Bounded checked sessions: Phase 0 (2026-10-10)

Measurement precedes the proposed contract in [the review spec](checked-sessions.md).
This phase added only test/benchmark code; runtime, generators, standalone APIs,
Wasm, translated modules, and proof locks remained unchanged for its samples.
Phase 0 implemented or accepted no throughput feature. M1 performance
acceptance remains open.

### Conditions and provenance

The clean starting checkout was `main` at `9623228`, equal to the local
`origin/main` ref. This machine's checkout is `/home/dario/Code/cocoon`, not the
supplied `/home/bits/cocoon`. Measurements use the already installed
`/usr/lib64/go/1.26/bin/go` (Go 1.26.8), linux/amd64, `GOAMD64=v1`, CGO disabled,
and explicit `-pgo=off`. The system Go 1.27.1 is used for quality checks.
The CPU is an AMD Ryzen 7 5800HS with 8 physical/16 logical cores, frequency
boost enabled, CPUs 0–15 allowed, and no affinity. The complete measurement
runner took 288.0 seconds. Compilation, linting, tests, fuzzing, and reviewer
work had finished before the recorded samples; none ran concurrently.

Each of seven 250 ms throughput sets runs GOMAXPROCS 1, 4, and 16, rotating
processor-setting order between sets. Subbenchmark order within a process is
fixed. Primitive layers use five 300 ms samples at GOMAXPROCS 1. The seven
existing Datadog workloads use seven 500 ms samples at GOMAXPROCS 16, followed
by five 300 ms real-guest layer samples at GOMAXPROCS 1. No samples are removed.
These are exploratory medians, without significance or production-speedup
claims; short windows, fixed within-process ordering, GC, and shared-machine
frequency effects limit interpretation.

Build the frozen binaries before timing:

```sh
CGO_ENABLED=0 GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go \
  /usr/lib64/go/1.26/bin/go test -pgo=off -c \
  -o .cache/sessions-phase0/rt.test ./rt
CGO_ENABLED=0 GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go \
  /usr/lib64/go/1.26/bin/go test -pgo=off -c \
  -o .cache/sessions-phase0/dd-baseline.test ./examples/datadog/go/dd
GOMAXPROCS=16 CGO_ENABLED=0 .cache/sessions-phase0/rt.test \
  -test.run '^$' -test.bench '^BenchmarkSessionThroughput$' \
  -test.benchtime=250ms -test.count=1 -test.benchmem
```

With trailing whitespace normalized, full commands, iteration counts, ns/sequence, ns/call, calls/s, input MiB/s,
bytes/sequence, and allocations/sequence are preserved in the committed
[complete raw log](measurements/checked-sessions-phase0.txt). Its SHA-256 is
`517833c82c0cab692a376e6f525aafd04c6dd0693e8ffc4c4dc32d963d32c774`. The ignored individual logs are under `.cache/sessions-phase0`;
the committed log is the durable copy. Binary SHA-256 hashes are:

- Runtime benchmark: `030f8cef86ad3d88ee1100f02763fe8091bf79ea823856dc58df06c4e80258b7`.
- Unchanged Datadog baseline: `aec965fbf86f67ceaf5a916a0df83dd306b4d27e580d8dd3d5b484f43bbb1c32`.
- Authored benchmark source: `9a79138a9e7048299bc223b12f940b0160b24e7df5836b42fd9dd87024e30996`.

The packaged Go toolchain lacks `go tool buildid`; binary hashes provide the
available provenance. Quality uses the official golangci-lint 2.13.1 binary
(revision `6d2288e0`, built with Go 1.27.0), verified against its published
archive checksum. The pre-existing cached linter reports `(devel)` and was
not used to qualify this addition.

### What is measured

`rt/session_bench_test.go` supplies a native-Go synthetic ABI with a 1 KiB
hold-back window per stream, fixed generation-one handles, preallocated memory,
and bounded input/output copying. Empty writes still reserve and encode their
zero-byte variable input. This is a host-runtime ceiling, not translated-Rust
or injector throughput. It omits guest input ownership allocations, real Rust
slab reuse, charset/partial-match parsing, construction, end, and disposal.
Nonempty replies are copied into owned Go bytes in every checked path.

Each parallel worker has its own resource ownership cell. `Home` is a
count-one synthetic checked resource path sharing one instance, including
library admission, ownership, execution lock, reset, host panic metadata,
input checks, owned reply validation, post-call memory checks, and poisoning.
`Admit32` shares one admission and ownership lock but uses 32 existing checked
`Instance.Call`s. `Locked32` also shares the execution lock; its test-only
checked boundary repeats every current per-operation check and stops on the
first error. `Locked4/16/64` are sequence-length controls at 64-byte inputs.
No prototype API is exported.

`Private` and `Private32` give each worker a fixed independent instance while
retaining shared library admission. They are hypothetical placement controls,
not implemented resource APIs. They use 64 KiB linear memory plus a 1 KiB
window per worker, versus one 64 KiB instance and one window per resource in
`Home`. `Pool1` and `PoolP` warm every lazy slot before timing and include both
admissions, context checks, a checked call, and a checked recycler. Synthetic
trim has no guest work. Pool capacity is one or GOMAXPROCS respectively; these
are stateless-envelope controls and cannot maintain a stream across arbitrary
pool borrows.

A benchmark operation is a whole sequence. `ns/call` and `calls/s` normalize
by its command count; parallel ns/op is aggregate throughput, not caller
latency. MiB/s counts input bytes, not output bytes; the initial 1 KiB holdback
is not emitted. B/op and allocs/op remain per sequence. Go reports allocs/op
as a truncated integer: the first holdback writes have no output, so a row
showing zero can still allocate on almost every steady-state nonempty write.
No allocation removal is claimed. Tests independently check mixed-size byte
order, exact/empty window boundaries, tail preservation, owned-output stability,
call-state reset, and actual sequence termination at command two for traps,
excess memory, malformed replies, and reserved status.

### Admission and serialization findings

Primitive medians (and all raw ns/op samples) are:

| Mechanism | Median ns | Raw samples in ns |
| --- | ---: | --- |
| Admission | 6.668 | 6.708, 6.669, 6.668, 6.664, 6.664 |
| Ownership | 11.23 | 11.45, 11.23, 11.2, 11.39, 11.15 |
| GuestHandle | 2.533 | 2.633, 2.533, 2.789, 2.533, 2.532 |
| Mutex | 5.391 | 5.398, 5.39, 5.391, 5.387, 5.391 |
| CheckedEmpty | 15.53 | 15.93, 15.53, 16.46, 15.52, 15.51 |
| ReplyUnit | 9.181 | 9.822, 9.208, 9.181, 9.18, 9.181 |
| Guest | 16.16 | 16.16, 16.18, 16.21, 16.15, 16.16 |

Admission is 6.668 ns without contention; host ownership is 11.23 ns, the bare
mutex is 5.391 ns, and canonical unit reply validation is 9.181 ns. Empty
checked calls include state/reset, host panic metadata, recovery, and memory
checks as well as locking (15.53 ns). Guest handle resolution and the synthetic
kernel remain per command. These independently compiled mechanisms are
not additive cost accounting. They do not justify skipping checks.

Real translated sketch layer medians are 53.54 ns guest, 72.64 ns checked
instance, 82.79 ns resource, and 89.31 ns facade. Guest work therefore remains
material even for a scalar sketch. Layer differences provide rough ceilings,
not exact component shares or a claim about standalone latency improvements.

| Input per write | GOMAXPROCS | Home Mcalls/s | Locked32 Mcalls/s | Private32 Mcalls/s | Home / Locked32 ns per call |
| --- | ---: | ---: | ---: | ---: | --- |
| 64 B | 1 | 7.355 | 9.220 | 9.225 | 136 / 108.5 |
| 64 B | 4 | 6.141 | 9.245 | 25.449 | 162.8 / 108.2 |
| 64 B | 16 | 5.215 | 8.986 | 38.832 | 191.7 / 111.3 |
| 4096 B | 1 | 1.583 | 1.668 | 1.671 | 631.6 / 599.4 |
| 4096 B | 4 | 0.990 | 1.119 | 1.416 | 1010 / 893.4 |
| 4096 B | 16 | 0.770 | 0.974 | 1.667 | 1298 / 1027 |
| 16384 B | 1 | 0.433 | 0.445 | 0.445 | 2310 / 2249 |
| 16384 B | 4 | 0.324 | 0.385 | 0.454 | 3086 / 2599 |
| 16384 B | 16 | 0.278 | 0.345 | 0.467 | 3591 / 2901 |

At GOMAXPROCS 16, the 64-byte shared-home ceiling is 5.22 Mcalls/s, rising to
8.99 Mcalls/s with 32 checked calls per sequence; independent worker instances
with the same sequence reach 38.83 Mcalls/s. Shared-lock sequencing improves
this synthetic path 1.72×, but leaves the shared-home ceiling intact. The
independent-instance sequence is 4.32× the shared-home sequence, with more
instance memory. This supports reviewing both amortization and placement.

For 4 KiB writes at GOMAXPROCS 16, the corresponding ceilings are 0.770,
0.974, and 1.667 Mcalls/s (about 3009, 3803, and 6511 input MiB/s). For 16 KiB,
they are 0.278, 0.345, and 0.467 Mcalls/s. Copy/allocation and serialized guest
work remain, so large-chunk gains are much smaller than the tiny-call ceiling.
These rates are not expectations for translated streaming payloads.

### Negative results and scope limits

- Independent instances alone do not solve tiny-call contention here:
  64-byte `Private` reaches only 5.45 Mcalls/s at GOMAXPROCS 16 versus Home's
  5.22. This is consistent with shared per-call admission contention, but also
  changes ownership/execution locking and loop overhead; the attribution is
  not isolated. At GOMAXPROCS 4 it reaches
  13.94 Mcalls/s. Scaling is not monotonic; more processors are not a guaranteed
  gain for cheap work.
- Sharing only admission/ownership is weaker than also sharing the execution
  lock: 64-byte `Admit32` is 139.3 ns/call at GOMAXPROCS 16 versus Locked32's
  111.3. For 4 KiB, Admit32 is 1212 versus Home's 1298 and Locked32's 1027.
- On one processor, extending shared execution from Admit32 to Locked32 adds
  little for large chunks: 4 KiB is 598.7 versus 599.4 ns/call, and 16 KiB is
  2262 versus 2249. These medians do not establish an isolated benefit.
- At GOMAXPROCS 16, 16 KiB Private32 is 2143 ns/call versus Private's 2121;
  sequencing is slightly slower by the medians. No significance claim is made.
- Longer sequences are not automatically better. At GOMAXPROCS 16, 64-byte
  Locked4/16/32/64 medians are 122.9/112.6/111.3/112.0 ns/call; N=64 gives no
  further measured benefit. Competing-stream tail latency and fairness were
  not measured and must be qualified after approval.
- A lock held while waiting for host network data would invalidate this
  CPU-only ceiling and extend Close indefinitely. Success throughput and the
  prototype fault checks do not establish a production session lifecycle.

The supplied cross-ecosystem reference `/home/bits/inject-bench` is absent on
this machine; an equivalent local directory was not found. Its reported
~1–2 µs/call facade cost and ~11× per-byte translation overhead are operator
context, not independently reproduced measurements. Uncontended admission
here is only about 0.3–0.7 percent of that reported fixed cost. Removing tens
of nanoseconds of host work cannot address an 11× guest translation cost;
contention amortization and resource placement are the relevant hypotheses.
The kernels, wrappers, inputs, and machines are not controlled equivalents.

### Fresh real-workload baseline

All seven existing workloads remain unchanged. These are fresh baseline
samples for a future paired Phase 2 run, not a before/after acceptance result.
Differences from historical sessions are not source improvements/regressions.

| Workload | Median ns/op | Raw samples in ns/op |
| --- | ---: | --- |
| ObfuscateSQL | 2156 | 2186, 2156, 2146, 2173, 2145, 2152, 2157 |
| SketchAdd | 88.36 | 88.69, 88.14, 88.13, 88.12, 89.35, 88.36, 88.47 |
| SketchAddMany1k | 44612 | 44612, 44865, 44547, 44746, 44573, 44343, 44799 |
| ObfuscateTraces1k | 2598198 | 2608015, 2597670, 2571825, 2601919, 2564812, 2634479, 2598198 |
| SketchAddNegativeBins | 88.91 | 88.95, 90.91, 88.91, 88.37, 88.57, 88.46, 88.91 |
| SketchAddVaried | 88.96 | 88.79, 90.06, 89.65, 88.89, 88.96, 88.87, 90.59 |
| ObfuscateSQLVaried | 2779 | 2786, 2770, 2782, 2771, 2773, 2785, 2779 |

SQL remains below its original 2215.4 ns limit in this run; scalar SketchAdd
at 88.36 ns remains above 55.44 ns. No new same-machine reference is available,
and M1 performance is not accepted. At this stage Phase 2 needed matched before/after scalar,
SQL, batch, trace, varied, negative-bin, streaming, and independent holdout
samples, including fairness and live-instance memory under contention.

### Review and qualification status

Guest-side, host-side, and adversarial reviewers inspected the harness before
measurement and retention. Review fixes add current Host/reset work, empty
input reservation, outer pool context checks, correct topology setup, byte
oracles, and middle-of-sequence faults. The single-call-around-a-loop shortcut
was rejected because it postpones memory checks/reset/poisoning. Synthetic
ceilings are retained as diagnostics; no production optimization is retained.

The Phase 0 addition passes `make check` with official golangci-lint 2.13.1,
`make race`, Go 1.26.8 runtime tests, darna, and exact staged-snapshot tests.
The operator explicitly approved the specification on 2026-10-10. Phase 2 was
pending at this point; its final translated-stream evidence and quality gates
are recorded above and in the roadmap.

## Current results

The fifth performance pass on 2026-10-06 investigated exact guest math lowering
and tested a guarded floor/saturated-conversion fast path with separate guest,
host, and adversarial review agents. Neither the unrestricted screen nor the
longer pinned confirmation established a timing improvement. The candidate
was removed; production code, both Wasm proofs, translated modules, facades,
and locks are unchanged. Performance acceptance remains open.

Two independent test commits remain: exact sketch-bin transitions and a verified
negative-bin benchmark (`1672671`), and executable numeric lowering contracts
included in `make smoke` (`b03d81b`). These are correctness/measurement additions,
not a claimed speedup. No profile or compiler-default change was retained.

### Exact guest math and conversion trial

The pinned libdatadog mapping at `libdd-ddsketch/src/lib.rs:295` computes a bin
from `floor(value.ln() * multiplier + index_offset)`. Its Wasm logarithm comes
from Rust 1.97's compiler-builtins/libm implementation, not Go's `math.Log`.
The translated hot path is `fn176`; the baseline compound floor/conversion is
at `examples/datadog/go/dd/internal/wasm/module.go:35812`.

Assembly inspection found that the dense log polynomial already stays in XMM
registers without floating-point spills or helper calls. The explicit float64
conversions do not add standalone instructions there; they preserve rounding
boundaries and prevent multiply/add fusion or reassociation. `math.Floor` is
already a guarded SSE4.1 `ROUNDSD`, and saturated conversion is inlined. The
58.29-percent guest, 43.58-percent math, and 10.16-percent result-validation
profile categories overlap. The roughly 14-percent flat polynomial and
2.14-percent flat conversion samples are attribution, not available speedups.
No exact, beneficial polynomial rewrite was identified.

The trial rewrote only a direct, correctly bound
`i32_trunc_sat_f64_s(math.Floor(expr))` into a helper with this guarded body:

```go
if f >= 0 && f < 0x1p31 {
    return int32(f)
}
return i32_trunc_sat_f64_s(math.Floor(f))
```

In that representable interval, truncation toward zero equals floor; all
negative, NaN, infinite, and out-of-range values use the original fallback.
The operand AST, including its explicit float conversions, was preserved and
evaluated once. See the [Go numeric-conversion rules](https://go.dev/ref/spec#Conversions_between_numeric_types),
[rounding rules](https://go.dev/ref/spec#Floating-point_operators), and
[Wasm saturated-conversion definition](https://webassembly.github.io/spec/core/exec/numerics.html#op-trunc-sat).
This is an integer-result equivalence argument, not a floating-point-environment
or NaN-payload equivalence claim.

The experimental AST pass required the exact unique helper signature/body,
real unshadowed math import, bound callee, one argument, and no ellipsis or
reserved-name collision. Review found that text alone also permits shadowed
built-in numeric types: `type int32 = float64` changes the fast path's result.
The trial therefore rejected bound non-predeclared numeric types, with a
regression test. Conflicting explicit math aliases were also rejected. The
fallback was appended after original-site rewriting to prevent recursion.
Datadog had one match; compute had none. These guards were tested but are not
shipped because the entire optimization was withdrawn.

Values below one can still populate positive-index bins. All existing varied
values do; they do not qualify the negative-index fallback. The new eight-value
corpus spans `1e-12` through `2e-12`, and the original Wasm independently proves
that each value populates a negative bin.

### Fifth-pass measurements

All binaries use Go 1.26.8, CGO disabled, `GOAMD64=v1`, and explicit `-pgo=off`
on the AMD Ryzen 7 5800HS. Seven rotated before/after/reference triples run
without compilation, lint, tests, fuzzing, or agent work. No samples are removed.
The independent reference is unchanged; only aligned SQL/scalar inputs support
overhead comparisons. Reference batch/trace inputs and pool size differ, and
the new negative-bin workload has no matched reference benchmark.

The first screen uses 16 Go processors, no affinity, and 500 ms per benchmark:

| Workload | Original | Candidate | Result |
| --- | --- | --- | --- |
| SQL | 1.905 µs | 1.913 µs | No detected change, p=0.927 |
| SketchAdd | 76.23 ns | 75.62 ns | No detected change, p=0.512 |
| SketchAddMany1k | 37.09 µs | 36.82 µs | No detected change, p=0.710 |
| ObfuscateTraces1k | 2.285 ms | 2.332 ms | No detected change, p=0.209 |
| SketchAddVaried | 76.58 ns | 75.27 ns | No detected change, p=0.209 |
| ObfuscateSQLVaried | 2.450 µs | 2.430 µs | No detected change, p=0.383 |
| SketchAddNegativeBins | 76.09 ns | 79.47 ns | No detected change, p=0.097 |

The longer confirmation uses one Go processor, CPU 14, and one-second samples:

| Workload | Original | Candidate | Result |
| --- | --- | --- | --- |
| SQL | 1.821 µs | 1.820 µs | No detected change, p=0.710 |
| SketchAdd | 74.37 ns | 73.28 ns | No detected change, p=0.165 |
| SketchAddMany1k | 35.22 µs | 35.89 µs | No detected change, p=0.097 |
| ObfuscateTraces1k | 2.157 ms | 2.196 ms | No detected change, p=0.209 |
| SketchAddVaried | 74.74 ns | 74.08 ns | No detected change, p=0.644 |
| ObfuscateSQLVaried | 2.339 µs | 2.352 µs | No detected change, p=0.833 |
| SketchAddNegativeBins | 73.97 ns | 75.69 ns | No detected change, p=0.165 |

The apparent scalar shifts are −0.80 and −1.47 percent; negative-bin shifts
are +4.44 and +2.33 percent. None establishes a gain or regression. Variability
remains substantial even with affinity, including 24-percent scalar and
75-percent trace confidence-interval bounds in the confirmation. The rejection
is for insufficient evidence, not proof of equivalence or impossibility.
All Go allocation counts remain unchanged; trace bytes vary slightly with
amortized setup and collection. P-values are exploratory and unadjusted.

Original/reference SQL and scalar medians are 1.905/1.669 µs and 76.23/48.40 ns
in the screen: 14.1 and 57.5 percent overhead. Pinned medians are
1.821/1.621 µs and 74.37/46.05 ns: 12.3 and 61.5 percent overhead. Neither
fresh same-run 10-percent target is met. SQL remains below its original
2.2154 µs limit; scalar remains above 55.44 ns. Differences from the preceding
pass are session variation, not a retained source improvement.

### Retained contracts and qualification

Nine adjacent binary64 boundaries were located by bit-pattern search against
the unchanged Wasm interpreter, not a host logarithm. They span negative bins,
the zero/positive-bin transitions, values around one, very small and large
exponents, and the zero-counter threshold. Fresh singletons prevent the
2048-bin collapse policy from hiding a wrong bin. Scalar and batch paths check
exact Count bits and complete protobuf bytes; triples and zero/tiny/negative-bin
mixes are also compared. The authored protobuf reader includes leading zero
counts when locating the first populated bin, with its own regression.

The independent WAT fixture runs pinned assembly, optimization, actual
wasm2go translation, and actual hardening before compiling the resulting Go.
Both Go and wazero check explicit integer/bit goldens: integer-limit neighbors,
signed zero, subnormals, finite extremes, infinities, signed signaling/quiet
NaNs, separate f32/f64 multiply/add rounding, addition order, once-only side
effects, bounds traps, and post-trap continuation. A deliberately broken helper
must fail with a semantic mismatch, not merely a compilation error. The fixture
also passes against the original hardener, independently of the candidate.

Candidate qualification passed strict check, both real proof builds and repeat
builds, Rust checks, race tests, five differential/structural fuzz targets,
external/relocation/Go-only smoke, and all five cross-compilation targets.
Numeric contracts ran natively on linux/386 and amd64-v3 as well as amd64-v1.
Arm64 was cross-compiled, not natively executed in this session. The smoke
fixture's CGO-disabled child is not race-instrumented by a parent race run.
Candidate hardening coverage was 94.6 percent, with the new pass at 95.5 percent;
restored hardening remains 94.0 percent. No coverage gain is claimed from code
that was removed. Restored production passes the retained contracts and proof
rebuilds, and its benchmark binary is byte-identical to the frozen baseline.
Each retained commit passes darna and exact staged-snapshot tests.

### Fifth-pass provenance

The before/after binaries include identical authored Datadog tests from
`1672671`. Their SHA-256 hashes are
`4ac489cc3c8e993570706aa4259cd64ed7e1da27f47a2dc66d901e47fa3a0ffa` and
`81bb7a7248e8758c540429ecb6e3d95bc970c0591af30d2a7fb8bad30aa99237`.
Their build IDs are `22fd5ce8f45a38453adc4f12c11106e2ea46992e` and
`9b7b88349f8771e52228a4fbc3420faf06e7f9fb`. The independent reference binary
hash is `bf3f15d3241684b35b0b0b448799373e8d66951f955827cce38fd7e365237dd8`.
Among generated proof artifacts, the candidate changed only translated Go
and its lock digest, to
`0e16305e4792902b3b6bd307bcd47ddc3e0783b4c8db918978233dfab8afb631`;
the retained Go hash is again
`f50d20e7fe091569353ac1ced7865a53cd23bc9add668c3aac0f5f3a0ac8eed4`.
Datadog Wasm remains
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`.
Compute and all other proof artifacts/lock fields are unchanged.

Complete raw logs and summaries are in ignored `.cache` under
`round12-floor-screen` and `round12-floor-pinned`; the rejected source patch
is saved as `.cache/round12-floor-candidate.patch`. It is experimental evidence,
not maintained production code. The retained commits are local and unpushed.

## Fourth performance pass

The fourth performance pass on 2026-10-06 tested profile-guided compilation
(PGO) and a fused checked resource-call path, with separate host, guest, and
adversarial review agents. No production source change or default profile
was retained.
The useful additions are held-out varied workloads (`2d95490`), ownership and
nested execution regressions (`f2d0158`), and a compiled generated-facade shutdown
regression (`85b55d0`). Runtime coverage increased from 97.0 to 97.8 percent.
The Wasm, translated modules, generated facades, and proof locks are unchanged.
The longer unrestricted comparison confirms a 5.36-percent scalar PGO gain,
including a 4.98-percent held-out gain, but batches regress 1.26 percent. PGO
remains an opt-in trade-off, and performance acceptance remains open.

### Profile guided compilation

Both Cocoon and the independent reference trained their own CPU profile, with
PGO explicitly disabled. Only the original SQL, increasing-value scalar,
1,000-value batch, and 1,000-trace benchmarks were selected. Each class ran for
two seconds with 16 Go processors and no affinity. This is equal nominal time
per class, not a production request mix. Cocoon's new varied-value and varied-SQL
benchmarks were held out from training. The reference retains the earlier
independently built/hardened overlay; its batch and trace inputs and pool size
differ from Cocoon's, so reference overhead comparisons use aligned SQL and
scalar workloads only.

The AMD Ryzen 7 5800HS comparisons use Go 1.26.8, CGO disabled, and explicit
`-pgo=off` or an absolute path to that implementation's own profile. Seven
sample sets rotate the four variants: Cocoon off/on and reference off/on.
Compilation, tests, fuzzing, lint, and agent work stop before timing. Profiles
are captured separately from unprofiled measurements; outliers are not removed.
All p-values are exploratory and unadjusted for repeated comparisons.

The initial unrestricted 500 ms screen did not establish a scalar improvement
(75.99 to 74.41 ns, p=0.383) or a varied-scalar improvement
(76.46 to 74.36 ns, p=0.318). Varied SQL improved 4.56 percent (p=0.038);
the other original workloads had no detected timing change. A longer pinned
confirmation used one Go processor, CPU 14, and one-second samples:

| Workload | PGO off | PGO on | Pinned result |
| --- | --- | --- | --- |
| SQL | 1.833 µs | 1.733 µs | −5.46%, p=0.010 |
| SketchAdd | 74.34 ns | 70.14 ns | −5.65%, p=0.026 |
| SketchAddMany1k | 35.27 µs | 35.29 µs | No detected change, p=0.902 |
| ObfuscateTraces1k | 2.164 ms | 2.083 ms | No detected change, p=0.097 |
| SketchAddVaried | 73.53 ns | 69.99 ns | −4.81%, p=0.001 |
| ObfuscateSQLVaried | 2.320 µs | 2.245 µs | −3.23%, p=0.011 |

The matched PGO reference scalar median is 45.36 ns, leaving 54.6 percent
overhead in this pinned run. Its SQL improves from 1.587 to 1.517 µs
(−4.41%, p=0.002); scalar, batch, and traces have no detected reference gain.
PGO does not close either the original 55.44 ns scalar limit or the pinned
same-machine 49.896 ns limit. Allocation counts are unchanged for all six
Cocoon workloads. Amortized construction bytes are not per-call allocations.

The final unrestricted confirmation returned to 16 Go processors and no
affinity, with seven rotated one-second sample sets and the same frozen
profiles and binaries. It establishes a synthetic scalar benefit beyond the
pinned run, but also exposes a batch regression:

| Workload | PGO off | PGO on | Unrestricted result |
| --- | --- | --- | --- |
| SQL | 1.843 µs | 1.765 µs | −4.23%, p=0.001 |
| SketchAdd | 77.37 ns | 73.22 ns | −5.36%, p=0.004 |
| SketchAddMany1k | 35.945 µs | 36.399 µs | +1.26%, p=0.004 |
| ObfuscateTraces1k | 2.164 ms | 2.104 ms | −2.78%, p=0.001 |
| SketchAddVaried | 77.15 ns | 73.31 ns | −4.98%, p=0.003 |
| ObfuscateSQLVaried | 2.376 µs | 2.253 µs | −5.18%, p=0.001 |

Percentages use unrounded medians. The reference improves SQL from 1.655 to
1.572 µs (−5.02%, p=0.001); its scalar is 47.52 versus 47.23 ns (p=0.710),
with no detected batch or trace change. No-PGO-to-no-PGO overhead is 11.4
percent for SQL and 62.8 percent for scalar. PGO-to-own-PGO overhead is 12.3
and 55.0 percent. Both PGO same-run 10-percent limits remain unmet:
1.7292 µs for SQL and 51.953 ns for scalar. SQL is below the original absolute
2.2154 µs limit; scalar still exceeds 55.44 ns. Allocation counts remain
unchanged. The slower batch is a measured trade-off, not an accepted universal
optimization. The earlier screen's inconclusive scalar result remains recorded.

No benchmark profile is installed as `default.pgo`. Synthetic gains do not
establish a production speedup, and profiles belong to the consuming application.
The [profile recipe](#benchmarking-and-profiles) preserves explicit controls and
excludes the held-out workloads. Training weights were not tuned against these
held-out results; further tuning needs independent traffic and a new holdout.

### Rejected fused resource calls

The candidate replaced generated `Resource.Use` → `Instance.Call` adapters
with one callback and a shared checked execution core. Both locks, ownership
and epoch checks, full call reset, panic containment, memory limits, reply
validation, and lifecycle admission/drain remained intact. It added
`Resource.Call` and callback-scoped `Call.Handle` only for this experiment;
neither API remains in the branch.

Seven rotated unrestricted 500 ms triples found no established gain in any
of the six workloads. Scalar medians were 74.81 versus 72.07 ns (p=0.259),
and varied scalar was 78.03 versus 74.71 ns (p=0.535). The longer CPU 14,
one-processor, one-second confirmation also did not improve scalar calls:

| Workload | Original path | Fused candidate | Pinned result |
| --- | --- | --- | --- |
| SQL | 1.851 µs | 1.808 µs | −2.32%, p=0.043 |
| SketchAdd | 73.06 ns | 73.43 ns | No detected change, p=0.318 |
| SketchAddMany1k | 35.04 µs | 35.17 µs | No detected change, p=0.805 |
| ObfuscateTraces1k | 2.140 ms | 2.147 ms | No detected change, p=0.535 |
| SketchAddVaried | 74.34 ns | 73.14 ns | No detected change, p=0.318 |
| ObfuscateSQLVaried | 2.329 µs | 2.340 µs | No detected change, p=0.833 |

The candidate was removed because the intended scalar benefit was not
established. These results do not prove equivalence. The observed SQL change
does not isolate causality: its generated operation and guest parser were
unchanged. All Go allocation counts were unchanged.

The retained tests exercise full-width interleaved handles, contained nested
guest failures, active ownership through GC followed by eventual cleanup,
alias Close and shared destructor results, and shared-instance serialization.
The compiled facade fixture blocks an admitted guest call, observes actual
terminal admission, rejects new resource/constructor/stateless calls, then
checks drain and once-only home release. Its subprocess is CGO-disabled; the
parent race invocation does not instrument that child. Runtime and real proof
tests separately pass under the race detector.

### Profiles and qualification

A working Go-built `pprof` frontend was recovered from
`.cache/go/00/00f01208b59cc8aadfd1faa6e2680739540033a3f6e8fedd79f8e89539b0530e-d/pprof`.
The distribution toolchains still omit that frontend; profile inspection is
no longer blocked. The matching retained pre-holdout scalar profile attributes
58.29 percent cumulatively to the guest export, 43.58 percent to sketch math,
and 10.16 percent to `ResultView`. A separate PGO scalar profile attributes
53.17, 41.80, and 8.99 percent respectively. These cumulative categories
overlap and cannot be added or treated as independent costs. Their difference
does not establish which compiler optimization caused a timing change.

The matching retained SQL profile puts 85.15 percent cumulatively in the
guest export and 77.87 percent in translated function `fn96` (49.02 percent
flat). Both retained profiles match build ID
`5005a40a835d17d4d76c9237ae6169b3c810febe` and the pre-holdout binary hash
`7657ad19a6fdb499d9650bede60c2f25df31e5ff9cfe2eed06dbac805e965fe2`.

In the retained profile, the log polynomial's dense source line accounts for
about 14 percent flat and saturated float conversion for about 2.14 percent
flat. Those are sampled attribution, not promised optimization gains. The fifth
pass above investigates exact lowering; preserve rounding and sketch bin boundaries.
Never substitute approximate math or edit generated modules alone.

Restored production is byte-identical to the frozen pre-experiment benchmark
binaries, both with and without PGO. Full check, integration, cross-platform
compilation, external/relocation/Go-only smoke, all five fuzz targets, Go 1.26.8
PGO-enabled tests and race checks, and native linux/386 targeted tests pass.
Both proofs rebuild to the original artifacts and locks. An independently
authored external consumer also builds and runs with PGO off/on, exercising
numeric boundaries, atomic batch errors, SQL decoding, and terminal admission.
Each retained test commit passes darna and exact staged-snapshot Go/proof tests.
The pass's commits are local and unpushed.

The Cocoon profile SHA-256 is
`f75e852793200b22ce61668326860e9047cc7150404cced26cad489bafbc3af2`;
the reference profile is
`1334598cd2ffbd0fd8e3684c9c22ea3909e233ae7bfcc94603958ad8aa0792c7`.
Cocoon off/on binary hashes are
`d5fdf9913af526e3e93779283186146557af9ca3ba4bb6c7328c3bf23b471a80` and
`d9c949317052141c92ee7371cc734a34f66179f077f6ac6101ec8f58bb0a20dc`;
reference off/on hashes are
`bf3f15d3241684b35b0b0b448799373e8d66951f955827cce38fd7e365237dd8` and
`f8f51b711bd9571a09609937e15860e8fdc00fcac927bdde45fdfe777471d06f`.
The rejected fused binary is
`7a8706ad61ca772448bb517b7614459f4b3dce26eed06a9453837ac95206d1fb`.
The off training and PGO scalar profile build IDs are respectively
`83928cd627234b621c12e8f5ac6d8179b7d3fc44` and
`e3b76825c536eb9e92f0fe3e9f17df5058fd90d5`.

Ignored raw logs use `.cache/round10-pgo-{off,on,reference-off,reference-on}-{1..7}.txt`,
`.cache/round10-pgo-confirm-*`, `.cache/round11-fused-{before,after,reference}-{1..7}.txt`,
`.cache/round10-pgo-final-*`, and `.cache/round11-fused-confirm-*`. Training profiles are
`.cache/round10-{cocoon,reference}-mixed.prof`; separate PGO scalar attribution
uses `.cache/round10-pgo-sketch.prof`. Restored qualification logs use
`.cache/round11-restored-*`; staged checks use `.cache/round11-staged-*`.
Matching retained profile summaries are `.cache/round11-retained-*-top.txt`
and `.cache/round11-retained-sql-cumulative.txt`.

## Third performance loop results

The 2026-10-05 and 2026-10-06 multi-agent loop completed nine rounds. No
performance optimization survived the full workload comparison. The branch
retains a destructor correctness fix and stronger callback, handle, numeric,
and optimizer regressions. The range predicate initially retained in `0dd1701`
was withdrawn in `db580ca`; its tests remain. Binaryen remains at 133 with
ordinary `-O3`, and the original Datadog Wasm and translated module are restored.

The retained production binary is byte-identical to the fix-only binary used
in the longer three-way comparison. Seven rotated one-second sample sets used
Go 1.26.8, CGO disabled, no PGO, the AMD Ryzen 7 5800HS, 16 Go processors, and
no CPU affinity. Loop-start, destructor-fix-only, and range-candidate order
rotated; the unchanged independent reference followed each set. All compilation
and quality jobs had finished before measurement. Both Cocoon builds contain
the same final Datadog Go tests, including the newly added boundary tests.

| Version | SQL median | SketchAdd median |
| --- | --- | --- |
| Matched loop-start production (`969dfca`) | 1.782 µs | 74.63 ns |
| Retained production (`db580ca`) | 1.767 µs | 74.33 ns |
| Independent reference | 1.617 µs | 46.65 ns |

No retained timing gain or regression is established: SQL p=0.927, scalar
p=0.710, batch p=1.000, and traces p=0.383, each with n=7. Batch medians are
35.492 versus 35.558 µs; trace medians are 2.1380 versus 2.1595 ms. Go allocation
counts remain one, zero, one, and one respectively. Construction contributes
amortized bytes to stateless benchmarks; bytes/op changes do not imply an
eliminated per-call allocation. These exploratory comparisons have not been
adjusted for testing multiple candidates.

Remaining same-machine reference overhead is 9.3 percent for SQL and
59.3 percent for SketchAdd. SQL's median is below both the original 2.2154 µs
limit and this run's 1.7787 µs limit, but that narrow margin is not a guarantee
across sessions. SketchAdd exceeds both 55.44 ns and this run's 51.315 ns limit.
Overall performance acceptance remains open. The change from the earlier
11.3/66.8-percent overhead figures is not a code improvement: the paired
loop-start comparison does not establish one.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Loop start | 1755, 1767, 1834, 1796, 1800, 1782, 1741 | 75.38, 73.68, 71.93, 74.84, 74.63, 75.22, 71.98 |
| Retained | 1738, 1823, 1846, 1767, 1763, 1766, 1809 | 71.32, 73.17, 74.33, 75.80, 78.12, 76.28, 72.41 |
| Reference | 1609, 1597, 1675, 1617, 1560, 1647, 1640 | 45.23, 47.91, 47.09, 46.65, 45.34, 47.50, 45.97 |

The final binary SHA-256 is
`7657ad19a6fdb499d9650bede60c2f25df31e5ff9cfe2eed06dbac805e965fe2`;
the matched loop-start binary is
`4315e367c165904a824280a531420205e8ba665c16d51984860735933047d5e4`;
the unchanged reference binary is
`bf3f15d3241684b35b0b0b448799373e8d66951f955827cce38fd7e365237dd8`.
The retained binary also matches the earlier `round3-point-base.test`, despite
later test-only commits and regenerated source locks. The Datadog Wasm hash is
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`;
translated Go is
`f50d20e7fe091569353ac1ced7865a53cd23bc9add668c3aac0f5f3a0ac8eed4`.

Full check, integration, race, cross-platform compilation, external CLI and
Go-only smoke, and all five differential/parser/hardening fuzz targets pass
after restoring the candidates. Go 1.26.8 and native linux/386 targeted tests
pass. Both proof packages reproduce their artifacts, generated tests, and
locks byte-for-byte. Runtime coverage remains 97.0 percent; Go and Rust
generator coverage remains 96.7 and 94.6 percent. Each code/test commit passed
darna and exact staged-snapshot Go tests. The loop's commits are local and have
not been pushed; the preceding pass's CI completed successfully.

Raw acceptance samples are `.cache/round8-attribution-{before,fix,reference}-{1..7}.txt`.
The superseded 500 ms final screen is `.cache/loop-final-{before,final,reference}-*`.
Profiles were captured separately from acceptance samples in
`.cache/loop-retained-{sketch,sql}.prof`. A cached frontend recovered during the
fourth pass makes these available for attribution; the retained scalar and SQL
findings are recorded above. Existing earlier attributed profiles remain below.

### Multi agent experiment results

Rounds 1 through 7 used seven alternating 500 ms pairs, one Go processor and
CPU 14 affinity for screening. Longer confirmations used one-second pairs.
Rounds 8 and 9 used the unrestricted 16-processor configuration above.
No benchmark ran concurrently with compilation, lint, tests, fuzzing, or agent
work. Outliers remain in the samples; inconclusive results are not proof of
equivalence. Screening gains are not additive and do not override the final
workload comparison.

| Round | Experiment | Evidence and disposition |
| --- | --- | --- |
| 1 | Conditional reused Call reset | No detected change in any workload; removed. Callback replacement/reservation regressions retained in `c1c5f52`. |
| 2 | Specialized successful unit decoding | Confirmation: scalar −2.20% (p=0.024), traces +1.64% (p=0.004); removed. |
| 3 | Ordered finite range predicate | Scalar screen −1.93% (p=0.008), no detected SQL/trace change; initially retained, later withdrawn after rounds 8 and final screening. Numeric regressions retained. |
| 4 | Single checked mutable slab lookup | Confirmation: scalar −5.35%, SQL +1.93%, batch +1.54%, traces +2.71%; removed. Handle retirement/reuse tests retained in `be98f8a`. |
| 5 | Binaryen O4 | Confirmation: batch −0.46% (p=0.026), no established SQL/scalar gain; kept O3. Exact policy and semantic contract tests retained. |
| 6 | O3 without StackIR | No established gain; noisy baseline outliers retained. Removed. Eager-select and operand-order tests retained. |
| 7 | O3 to convergence | No established gain, with noisy candidate outliers; removed. Convergence semantic variant retained. |
| 8 | Three-way attribution | Baseline→range build SQL +2.13% (p=0.011), no detected scalar change. No detected baseline→fix-only change; fix-only→range SQL inconclusive (p=0.128). Withdrew the range change without claiming its isolated causality. |
| 9 | Unsigned float-bit classifier | Versus fix-only: batch −2.85% (p=0.007), SQL +2.55% (p=0.035), no detected scalar change; removed. |

The initial unrestricted range screen also found SQL +3.03 percent (p=0.038)
without establishing a scalar gain. Static inspection found the SQL export
and parser source unchanged by either the destructor fix or predicate changes.
Code layout, cache, or compiler effects are hypotheses, not demonstrated causes.
Do not manually pad or edit generated modules to chase those timings.

The destructor fix (`3fb2ffc`) closes a pre-existing canonical-unit gap. Its
`ResultUnit` helper delegates to fully checked borrowed `ResultView` and rejects
nonempty successful replies. Generated destructors use it; ordinary unit
methods retain their existing borrowed-view/empty checks. It does not add work
to either timed SQL or scalar benchmark path. A compiled mock-facade regression
demonstrably fails against the old destructor generator. Mock fixture subprocess
tests use CGO-disabled Go; the parent race invocation does not instrument that
subprocess. Runtime and real proof tests separately run under the race detector.

Binaryen candidates kept all existing features, validation, meta-DCE, memory
limits, and AST hardening. An independent WAT/wazero fixture asserts expected
numeric, signed-zero, NaN transport, rounding, trap ordering, memory effects,
eager-select, and operand-order contracts for O3, O4, no-StackIR, and convergence.
This supplements Go-versus-the-same-Wasm differential tests, which cannot detect
a shared optimizer error. `make smoke` includes the fixture in CI.

The O4 diagnostic optimization run took 1.36 seconds/127,916 KiB peak RSS
versus O3's 1.01 seconds/118,328 KiB; these are single observations, not stable
build-cost estimates. Three alternating convergence runs took 2.19–2.21 seconds
versus O3's 1.05 seconds. Convergence keeps its final non-improving iteration;
smaller output is not guaranteed by that policy.

| Datadog optimizer | Wasm bytes | Translated Go bytes |
| --- | --- | --- |
| Range-based O3 experiment baseline | 293861 | 2240411 |
| O4 candidate | 294285 | 2235047 |
| No StackIR candidate | 294759 | 2242288 |
| Convergence candidate | 293312 | 2238105 |

These sizes compare the same range-predicate source, not the restored final
source. Candidate Wasm hashes are O4
`4084e0a45346fe69eba53cd9493211bc0ac5339133da33e426f8a1bb83a7d7d8`,
no StackIR `5108d1e3eccca33c6f59d9e4fcdeb6cdff650749caa98b2af97e76fe8fa8702d`,
convergence `b71afc1c54900a191020da9e83dd7d99f519fbd9712bb00d911cc111e6e7c310`,
and the float-bit classifier
`626db603bf535387ab4e3c91eb7c6cb3eafb820fb70c4bd64cc11ebce4c6ceeb`.
Ignored raw logs use `.cache/round1-call-*`, `.cache/round2-unit-*`,
`.cache/round3-point-*`, `.cache/round4-slab-*`, `.cache/round5-o4-*`,
`.cache/round6-no-stack-ir-*`, `.cache/round7-converge-*`,
`.cache/round8-attribution-*`, and `.cache/round9-bits-*`.

## Second performance pass

The second 2026-10-05 pass retained four host-side optimizations, with no change
to the Wasm artifacts or translator pin. Seven interleaved 500 ms sample sets
used Go 1.26.8, CGO disabled, no PGO, the AMD Ryzen 7 5800HS, 16 Go processors,
no CPU affinity, and the same query and increasing-value workloads. Pre-pass
and current order alternated; the independent reference followed each pair.
All compilation, linting, tests, and fuzzing had finished before measurement.

| Version | SQL median | SketchAdd median |
| --- | --- | --- |
| Before second pass (`6082ad7`) | 1.880 µs | 80.00 ns |
| Current (`0c653b3`) | 1.839 µs | 77.31 ns |
| Independent reference | 1.652 µs | 46.34 ns |

SketchAdd improves by 3.36 percent (p=0.001, n=7); SQL has no detected change
(p=0.128). Batch sketch medians are 36.402 versus 36.468 µs (p=0.902);
trace medians are 2.1968 versus 2.1637 ms (p=0.209). None establishes a timing
regression. The pre-pass trace samples include a 2.976 ms outlier, retained
rather than filtered. Allocation counts are unchanged: one for SQL, zero for
scalar adds, one for batch adds, and one for traces. Stateless bytes/op include
amortized lazy guest construction; changes in that metric are not evidence
of an eliminated per-call allocation.

Remaining same-machine reference overhead is 11.3 percent for SQL and
66.8 percent for SketchAdd. Both exceed this run's 10-percent limits of
1.8172 µs and 50.974 ns. SQL is below the original absolute 2.2154 µs limit
in this run; SketchAdd remains above 55.44 ns. Overall acceptance remains open.
The unchanged pre-pass binary also runs faster here than in the first pass,
so the historical 96.45 to current 77.31 ns difference is not a code improvement.
Use the paired 80.00 to 77.31 ns comparison.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Before | 1904, 1880, 1923, 1856, 1825, 1950, 1815 | 80.28, 84.27, 80.00, 79.24, 79.53, 79.25, 82.58 |
| Current | 1798, 1864, 1873, 1836, 1839, 1845, 1756 | 73.86, 78.76, 77.31, 76.32, 78.99, 77.65, 75.21 |
| Reference | 1638, 1678, 1652, 1670, 1660, 1620, 1652 | 45.87, 46.34, 49.23, 52.08, 47.02, 44.88, 45.25 |

Five separate 300 ms layer samples give medians of 46.07 ns for the guest,
65.90 ns for the checked instance, 72.94 ns for resource use, and 80.00 ns
for the diagnostic facade. Empty instance/resource callbacks take 14.23 and
22.06 ns; lifecycle enter/leave takes 6.198 ns. Five separate 500 ms samples
put the ready pool at 73.75 ns. These callbacks differ from the public benchmark
and are not exact additive cost accounting.

Separate three-second profiles attribute 55.6 percent of SketchAdd samples
cumulatively to the translated export, 39.2 percent to the sketch algorithm,
and 11.8 percent to reply validation. SQL's translated export accounts for
85.9 percent cumulatively; its parser accounts for 79.6 percent. These percentages
overlap, describe sampled execution rather than independent costs, and do not
establish the next optimization's benefit. The final profile build ID is
`c09b4d52203a98b361e0e66e1a3f7a18f8e66559`.

`make check`, `make race`, `make cross`, `make integration`, `make smoke`, and
`make fuzz` pass. Both proofs reproduce on repeat builds. Targeted tests also
pass with Go 1.26.8 and native linux/386. Runtime coverage is now 97.0 percent,
up from 95.6 percent before this pass. Each code or test commit passed darna
and exact staged-snapshot Go tests. Bounds, reply checks, panic containment,
shared ownership, cancellation, and terminal draining Close remain intact.
Raw final samples use `.cache/pass2-final-{before,final,reference}-{1..7}.txt`.
The review also found a pre-existing destructor unit-reply strictness gap;
it was subsequently fixed in the multi-agent loop above.

## First performance pass

The first 2026-10-05 pass retained three small optimizations: inline unit success
publication (`e099514`), a checked successful-status fast path (`14e056c`), and
cold unit-error publication (`078c9c9`). The final seven interleaved 500 ms sample
sets used Go 1.26.8, CGO disabled, no PGO, the AMD Ryzen 7 5800HS, and the same
query and increasing-value workloads. Release and current order alternated
between sets; each set also measured the independent reference. All compilation
and quality checks had finished before timing.

| Version | SQL median | SketchAdd median |
| --- | --- | --- |
| Release artifact | 2.269 µs | 114.4 ns |
| Current | 2.272 µs | 96.45 ns |
| Independent reference | 1.959 µs | 55.48 ns |

SketchAdd improves by 15.69 percent (p=0.001, n=7); SQL has no detected change
(p=0.710). Go allocation counts remain one for SQL and zero for SketchAdd.
Current overhead relative to the same-machine reference is 16.0 percent for
SQL and 73.8 percent for SketchAdd. Neither the original absolute limits
(2.2154 µs and 55.44 ns) nor the current same-machine 10-percent limits
(2.1549 µs and 61.028 ns) are met. M1 performance acceptance remains open.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Release | 2320, 2250, 2269, 2281, 2285, 2256, 2261 | 116.2, 114.4, 113.5, 113.2, 114.0, 115.9, 115.1 |
| Current | 2267, 2279, 2240, 2259, 2274, 2273, 2272 | 95.42, 96.51, 95.87, 96.85, 96.85, 96.25, 96.45 |
| Reference | 1946, 1948, 1974, 1970, 1948, 1975, 1959 | 55.18, 55.97, 55.45, 54.88, 55.48, 57.10, 55.63 |

Final five-sample layer medians are 54.86 ns for the direct guest, 78.08 ns for
the checked instance, 89.81 ns for the resource layer, and 97.83 ns for the
complete facade. These diagnostic callbacks differ from the public benchmark
and are not exact additive accounting. Separate three-second profiles attribute
53.7 percent of SketchAdd cumulatively to its translated export, 38.6 percent
to the sketch algorithm, and 7.4 percent to reply validation. SQL attributes
80.5 percent to its translated export. Cumulative percentages overlap.

`make check`, `make race`, `make cross`, `make integration`, and `make smoke`
pass for the final code. Both proofs reproduce across repeat builds; their
generated differential fuzz tests pass. Targeted tests also pass with Go
1.26.8 and native linux/386. Runtime statement coverage remains 95.6 percent.
Every code commit passed darna and the exact staged Go snapshot tests. No safety
checks or shutdown guarantees were removed, and no translator pin was changed.

## Release baseline

On 2026-10-05, the released Datadog artifact was remeasured after the source-path
and compiler-metadata reproducibility fixes. Five interleaved, unprofiled samples
used the same machine, Go 1.26.8, CGO disabled, no PGO, and 500 ms per benchmark.
Compilation, linting, tests, and fuzzing were finished before measurement.

| Version | SQL median | SQL Go allocations | SketchAdd median | SketchAdd Go allocations |
| --- | --- | --- | --- | --- |
| Cocoon release artifact | 2.225 µs | 1 | 113.5 ns | 0 |
| Independently rebuilt spike | 1.954 µs | 1 | 55.41 ns | 0 |

This is 13.9 percent SQL overhead and 104.8 percent SketchAdd overhead relative
to the same-machine reference. Both original absolute targets and the fresh
same-machine 10-percent targets remain unmet. Historical pre-release numbers
below must not be treated as the release's current performance.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Release | 2267, 2245, 2218, 2225, 2221 | 114.7, 115.6, 113.0, 113.5, 112.9 |
| Spike | 1943, 1954, 1960, 1942, 1994 | 56.26, 56.08, 55.41, 55.25, 55.12 |

### Call layer measurements

`BenchmarkSketchAddLayers` executes the same increasing-value workload through
the guest alone, instance serialization and reply validation, resource ownership,
and the complete facade. Each subbenchmark checks the resulting guest count
outside the timer. Only the facade is a supported public call; the other layers
deliberately bypass protections for measurement, not as alternative APIs.

Five 500 ms samples produced these medians, all with zero Go allocations:

| Layer | Time |
| --- | --- |
| Direct guest | 58.68 ns |
| Instance and checked reply | 83.56 ns |
| Resource and checked instance | 97.70 ns |
| Complete facade | 117.9 ns |
| Empty instance callback | 18.58 ns |
| Empty resource and instance callback | 27.51 ns |
| Lifecycle enter and leave | 7.208 ns |

These diagnostics include callback dispatch and have different compiler layouts
from the public benchmark. Differences between rows are approximate cumulative
costs, not independent additive guarantees. The release SketchAdd profile puts
53.9 percent cumulatively in the translated export and 6.4 percent in reply
validation. A separate SQL profile puts 79.2 percent in its translated export.
Profiles used three-second runs and are not acceptance timings.

### Unit reply inlining

Forcing Rust to inline `reply_unit` was measured independently before any other
change. Both proof artifacts and locks were regenerated. The paired medians
were 113.8 to 98.53 ns for SketchAdd, a 13.4 percent reduction with no Go
allocations. SQL moved from 2.243 to 2.262 µs, within about one percent; this
experiment does not establish a SQL improvement.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Before | 2228, 2281, 2232, 2243, 2244 | 113.8, 113.7, 114.3, 114.2, 113.0 |
| Inline unit reply | 2262, 2261, 2292, 2258, 2269 | 103.2, 98.53, 98.56, 98.04, 98.27 |

The descriptor, error encoding, output limit, and canonical empty success
contract are unchanged. Native unit-reply tests and an additional translated
versus wazero test cover data-to-unit and error-to-unit transitions.

### Successful reply status fast path

`ResultView` now skips general status dispatch for `OK` only after validating
the descriptor, output limit, and complete logical memory range. Error messages
still become owned Go data, and reserved or unknown statuses still poison the
instance. New regressions cover negative and unknown codes and a zero-length
success reply beyond logical memory.

A five-pair 500 ms screening run was followed by seven interleaved one-second
samples with execution order alternated between pairs. The longer run's medians
were 97.61 to 95.90 ns for SketchAdd and 2.298 to 2.296 µs for SQL. `benchstat`
reported a 1.75 percent SketchAdd reduction (p=0.026, n=7) and no detected SQL
change (p=0.594). Allocation counts remain zero and one respectively. This is
a modest improvement, not evidence that the remaining overhead is solved.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Before | 2270, 2301, 2291, 2298, 2309, 2288, 2301 | 98.10, 97.36, 97.73, 97.61, 97.46, 97.60, 100.1 |
| Status fast path | 2279, 2299, 2276, 2296, 2310, 2282, 2297 | 95.62, 100.8, 95.76, 95.73, 95.90, 96.08, 96.26 |

### Cold unit error publication

A broader seven-pair comparison caught a tradeoff missed by the first screening:
unit inlining plus the host fast path reduced SketchAdd by 15.05 percent but
increased SQL from 2.234 to 2.293 µs (2.64 percent, p=0.001). A weaker `#[inline]`
hint produced byte-identical Datadog Wasm, so it could not address that slowdown.

Moving only unit-error publication into a private `#[cold]` helper retained the
inline success path. Seven further 500 ms pairs alternated release and candidate
order, followed each pair with the previous implementation, and included batch
sketches and trace obfuscation. The release-to-candidate medians were:

| Benchmark | Release | Cold unit helper | Detected change |
| --- | --- | --- | --- |
| SQL | 2.230 µs | 2.241 µs | None, p=0.251 |
| SketchAdd | 113.2 ns | 95.89 ns | −15.29 percent, p=0.001 |
| SketchAddMany1k | 46.39 µs | 45.90 µs | None, p=0.165 |
| ObfuscateTraces1k | 2.561 ms | 2.574 ms | None, p=0.620 |

All allocation counts are unchanged. The trace samples include a 3.091 ms
candidate outlier; do not interpret “none detected” as proof of identical
performance under every workload. The SQL recovery is measured code-generation
evidence, not a claim that a unit-error helper directly speeds up the SQL parser.
Native tests cover all four error statuses, embedded NUL and Unicode messages,
stable descriptor addresses, output limits, and error-to-unit transitions.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Release | 2234, 2230, 2221, 2426, 2217, 2234, 2217 | 113.3, 113.2, 113.3, 113.1, 113.8, 113.2, 112.8 |
| Previous | 2330, 2314, 2290, 2286, 2284, 2295, 2293 | 97.12, 97.85, 96.63, 96.31, 95.72, 96.02, 101.7 |
| Cold unit helper | 2270, 2229, 2224, 2241, 2241, 2232, 2255 | 95.89, 95.17, 95.17, 96.02, 99.55, 96.45, 95.54 |

### Discarded guest layout experiments

Adding `#[cold]` to the argument, application, limit, and handle error constructors
did not establish a repeatable public-call improvement. SQL medians were
2.301 versus 2.278 µs; paired differences varied in sign. SketchAdd medians were
98.65 versus 99.49 ns. The annotations were removed rather than retaining a
code-layout change without convincing evidence.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Inline unit baseline | 2259, 2318, 2301, 2309, 2273 | 98.95, 100.5, 98.65, 98.40, 98.12 |
| Cold errors | 2288, 2278, 2239, 2229, 2290 | 97.77, 99.49, 98.46, 99.85, 101.6 |

Forcing both `Slab::index` and `Slab::get_mut` to inline also failed to establish
a benefit. SketchAdd medians were 99.12 versus 99.41 ns; SQL was 2.291 versus
2.273 µs, again with mixed-sign paired differences. Those annotations were
removed independently of the cold error experiment.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Inline unit baseline | 2276, 2263, 2291, 2326, 2300 | 99.02, 99.12, 98.29, 102.8, 102.3 |
| Inline resource lookup | 2266, 2296, 2273, 2250, 2299 | 99.41, 99.00, 99.53, 98.62, 101.4 |

## Second performance pass

The second 2026-10-05 pass retained four host-side changes. All screening
comparisons used Go 1.26.8, CGO disabled, no PGO, seven alternating 500 ms
sample pairs, CPU 14 affinity, and one Go processor. CPU affinity made the
small scalar differences easier to distinguish. These screening values must
not be compared directly with the earlier unrestricted 16-processor results.

| Change | Measured operation | Before | After | Detected change |
| --- | --- | --- | --- | --- |
| Skip error matching for successful callbacks (`c7e148e`) | SketchAdd | 79.80 ns | 78.24 ns | −1.95 percent, p=0.001 |
| Inline checked memory ranges (`469e865`) | SketchAdd | 79.27 ns | 77.83 ns | −1.82 percent, p=0.026 |
| Direct lifecycle admission for resource calls (`aec4d6d`) | SketchAdd | 78.61 ns | 75.78 ns | −3.60 percent, p=0.001 |
| Receive ready pool slots without a blocking select (`b30d62e`) | PoolAvailable | 109.60 ns | 72.37 ns | −33.97 percent, p=0.001 |

These percentages describe separate comparisons and must not be added. No
screening comparison established a significant SQL improvement. The pool
experiment moved SQL from 1.861 to 1.802 µs, but p=0.079 does not establish a
full-facade gain. The isolated pool benchmark is a diagnostic, not a substitute
for SQL acceptance.

Successful callbacks still undergo the post-call memory check; wrapped and
joined protocol errors still poison their instance. Range errors now construct
a private error value and format their unchanged diagnostic on demand. The
smaller success path inlines with Go 1.26.8 while preserving logical-length,
overflow, spare-capacity, and zero-length checks and
`errors.Is(err, rt.ErrProtocol)`.

Resource methods have no caller context, so checking `context.Background().Err()`
was redundant. They now enter the same lifecycle directly, retaining nil-wrapper,
nil-library, ownership, epoch, terminal admission, and draining Close behavior.
Stateless operations still check their caller context. Both facades and locks
were regenerated; both Wasm artifacts and translated guest sources are unchanged.

The pool retains context checks before and after borrowing. A ready slot uses
a nonblocking receive; an unavailable slot still waits on slot availability or
context cancellation. The existing panic-safe release path is unchanged. New
tests deterministically cancel after both lazy and initialized ready borrows,
cancel while waiting, and verify capacity and instance reuse.

| Comparison | Before samples in ns | After samples in ns |
| --- | --- | --- |
| Successful callback | 79.53, 80.37, 79.50, 80.21, 79.80, 81.09, 79.79 | 78.04, 79.00, 78.24, 78.07, 78.41, 78.20, 79.11 |
| Checked ranges | 79.72, 79.27, 79.86, 80.07, 78.81, 78.59, 78.56 | 77.22, 77.06, 77.83, 77.20, 78.65, 78.74, 78.96 |
| Resource admission | 79.28, 77.77, 77.18, 79.48, 80.20, 78.61, 77.45 | 76.87, 76.41, 75.55, 74.96, 75.78, 76.77, 74.69 |
| Ready pool | 109.0, 109.6, 109.1, 112.3, 110.2, 111.4, 109.5 | 71.14, 74.13, 71.01, 73.20, 72.83, 71.05, 72.37 |

### Rejected second pass experiments

Publishing an encoded reply by taking its vector instead of copying into the
cached output buffer passed native tests but regressed SQL from 1.946 to
2.055 µs (5.60 percent, p=0.007) and traces from 2.205 to 2.278 ms
(3.27 percent, p=0.001). Allocation counts were unchanged. Changing ownership
also changes buffer lifetimes; the comparison does not isolate the regression's
cause. The candidate and its regenerated artifacts were removed. Its Datadog
Wasm hash was `d1eaa8b1dd37afd1fcf715f719df04a7de18447a10b925276b348959c4575d5a`.

Combining resource unlock and KeepAlive defers moved SketchAdd from 75.82 to
74.71 ns but did not establish a significant gain (p=0.259). The production
change was removed. Its panic-unlock and GC-during-use regressions passed against
the original implementation and were retained independently (`0c653b3`).

An unrestricted initial range-inlining screen was inconclusive. Forcing 16 Go
processors onto one CPU produced heavy scheduling variance and is not retained
as performance evidence. The table above uses the cleaner one-processor
confirmation, not either noisy screen. Raw logs remain in ignored `.cache`
under `pass2-pinned`, `pass2-bounds-confirm`, `pass2-admission`, `pass2-pool`,
`pass2-owned`, and `pass2-resource-exit` prefixes.

## Historical pre release results

On 2026-10-04, five interleaved samples of the pre-change binary, independently
rebuilt spike, and current binary ran on the local AMD Ryzen 7 5800HS. Each
sample used Go 1.26.8, CGO disabled, no PGO, and 500 ms per benchmark. Compilation,
linting, tests, and fuzzing were finished before measurement. SQL used the same
query; SketchAdd used increasing values. Medians are:

| Version | SQL time | SQL Go allocations | SketchAdd time | SketchAdd Go allocations |
| --- | --- | --- | --- | --- |
| Cocoon before this pass (`e9052fd`) | 2.299 µs | 2 | 114.4 ns | 0 |
| Independently rebuilt spike | 1.950 µs | 1 | 55.18 ns | 0 |
| Cocoon after this pass (`ec3398d`) | 2.298 µs | 1 | 103.2 ns | 0 |

SketchAdd improved by about 9.8 percent. SQL eliminated one allocation and
reduced allocated bytes from 105–106 to 57–58 per operation, but its runtime
did not improve beyond sample variation. A scalar Count reply now also decodes
without Go allocation, protected by an allocation regression test.

The original spike medians, recorded on a different cloud VM, were 2.014 µs
and 50.4 ns. The original within-10-percent limits remain **2.2154 µs and
55.44 ns**. Neither was met in that session. Those results also exceeded 110 percent of the
fresh same-machine spike medians; do not mark M1 performance accepted.

For reproducible accounting, the five unprofiled samples in nanoseconds were:

| Version | SQL samples | SketchAdd samples |
| --- | --- | --- |
| Before | 2313, 2296, 2303, 2298, 2299 | 114.7, 121.1, 113.2, 113.8, 114.4 |
| Spike | 1936, 1944, 1950, 1961, 1983 | 54.87, 55.53, 55.18, 55.23, 55.18 |
| After | 2315, 2292, 2298, 2306, 2298 | 103.6, 103.0, 103.2, 103.2, 102.9 |

## Comparison provenance

The spike was built from its own unchanged Rust sources and Cargo lock using
Rust 1.97.0, rebuilt `std,panic_abort`, Binaryen 133 meta-DCE and `-O3`, the
268435456-byte memory cap, and wasm2go v0.4.16 with `-unsafe`. An independently
written AST preparation step applied its bulk-helper length bound. Go overlays
supplied the generated module and embedded Wasm without modifying reference
sources or copying its implementation into Cocoon. Its Go tests and wazero
differential tests passed before the final comparison.

Both builds use libdatadog revision
`7f3b16fe1b4bfc2a016ef869459e915ee02d6d6a`, but their Cargo dependency versions
and guest contracts differ. The spike uses four library instances and a
background context; Cocoon uses one instance and the benchmark context. The
benchmarks are serial. This is a comparison of the complete implementations,
not a controlled estimate of safety-check costs alone.

The reference Cargo lock SHA-256 is
`c39203ded9bc15f520da886bf3d1bcab8e0b6addd583d91edbdde3cfe5519c77`;
its rebuilt Wasm SHA-256 is
`8bda5ff7acff11fd64fd47a0a48165a71d610c28413665e4cc4d4034150d0651`.
The historical pre-release Datadog Wasm SHA-256 was
`2b0b48e84f8d38f705ce49ee52609a49a7e5112dfc59d68d0d257ce3d7becef6`.
The release baseline Wasm SHA-256 is
`9af0f6a8dcf934b3eb977dfdabe11ef5224bc1cded91353ee5f6687b86fd9f95`.
The retained unit-inlining artifact SHA-256 is
`d9b0eee8b99d2a728dc231ed10e4579fba22d4602c9f2860c5c0ed93e93ffd7e`.
The host status fast path does not change the guest artifact.
The final cold unit-error helper artifact SHA-256 is
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`.

## Implemented optimizations and next experiments

The runtime now reuses call state under its instance lock, admits lifecycle
calls atomically, avoids unused scalar input reservations, copies string inputs
directly, and consumes guest-owned string/byte inputs without a second clone.
SketchAdd's initial ~162 ns/one-allocation measurement fell to ~114 ns/zero
allocations before the earlier pass. That pass added typed unit replies, cached
the fixed output descriptor address, and decoded borrowed replies under the guest
lock before copying retained fields. Both native Rust and generated Go retain
their validation, and public `Result` still returns an owned copy.

The earlier post-change five-second SketchAdd CPU profile attributed about
52 percent of samples to translated guest execution, including 34 percent to the sketch
algorithm. Reply validation accounted for about 9 percent cumulatively.
Resource ownership, instance serialization, lifecycle admission, and generated
callbacks account for much of the remainder; cumulative percentages overlap.
These observations suggest experiments, not proof that a particular rewrite
will help.

The first 2026-10-05 pass added the isolated benchmarks and separate SQL profile above,
retained unit-reply inlining, successful-reply status dispatch, and a focused
cold unit-error helper. It rejected general cold error and resource lookup
annotations that did not establish an improvement. The second pass optimized
successful callback bookkeeping, checked range inlining, resource admission,
and ready pool borrowing. It rejected owned reply publication and combined
resource exit defers. The third loop retained correctness and optimizer-contract
tests, not a timing gain. The fourth pass found a synthetic PGO scalar benefit
with a batch regression and rejected fused resource calls. The fifth found no
justified log-polynomial rewrite and no established guarded-conversion gain.
Next, prioritize measured batch-only or guest SQL experiments and representative
application profiles; adding screening gains is not valid accounting.
Retain bounds checks, canonical replies, trap containment, shared ownership,
and draining Close, with staged-snapshot tests for each atomic change. See the
[roadmap](roadmap.md) for other pending acceptance work.

## Benchmarking and profiles

Run repeatable samples with `make bench`. To repeat the measured configuration,
use the pinned comparison compiler and finish compilation before measurement:

```sh
CGO_ENABLED=0 /usr/lib64/go/1.26/bin/go test -pgo=off -c \
  -o .cache/dd-bench.test ./examples/datadog/go/dd
.cache/dd-bench.test -test.run '^$' \
  -test.bench '^(BenchmarkObfuscateSQL|BenchmarkSketchAdd)$' \
  -test.benchtime=500ms -test.count=5 -test.benchmem
```

Collect profiles separately from unprofiled acceptance samples. For the
synthetic four-class experiment, explicitly exclude the varied held-out and
call-layer diagnostic benchmarks:

```sh
CGO_ENABLED=0 /usr/lib64/go/1.26/bin/go test -pgo=off -c \
  -o .cache/dd-profile.test ./examples/datadog/go/dd
GOMAXPROCS=16 .cache/dd-profile.test -test.run '^$' \
  -test.bench '^(BenchmarkObfuscateSQL|BenchmarkSketchAdd|BenchmarkSketchAddMany1k|BenchmarkObfuscateTraces1k)$' \
  -test.benchtime=2s -test.cpuprofile=.cache/dd-mixed.prof
go tool pprof -top .cache/dd-profile.test .cache/dd-mixed.prof
```

This gives each class equal nominal time, not a representative request mix.
For deployment, collect profiles from the consuming application's workload.
Merge compatible profiles with `go tool pprof -proto a.prof b.prof > default.pgo`,
place `default.pgo` beside that application's main package, or pass
`go build -pgo=/absolute/path/to/profile ./cmd/app`. A library does not enable
PGO globally for its consumers; Cocoon does not ship a default profile.
See the [Go PGO guide](https://go.dev/doc/pgo) for consuming-main placement and
profile selection.

Go PGO changes host compilation, not the pinned Rust/Wasm translation or
schema. Compare with explicit `-pgo=off`, profile each implementation separately,
and rerun tests and race checks after changing compiler options. Benchmark
profiles remain experimental until representative of deployed traffic.
Profiles can contain application symbols and paths; review them before sharing.
