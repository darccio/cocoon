# Contained-load bounds production record, 2026-10-12


Retain the conservative authored hardener pass and its executable proof. Varied
SQL and the SQL holdout repeat detected end-to-end gains against both the original
binary and the copied-check control. All three measured batches repeat smaller
benefits against the original binary. The original SQL confirmation difference
is undetected; standalone scalar gains are not established. Streaming supplies
no demonstrated improvement and its upward one-home medians remain negatives.
**M1 performance remains unaccepted.**

The private [predicate](../translation-contained-loads.md) and phase-0 evidence were
committed in `be8f6e0` before implementation. The reviewed production candidate
is `22cdc0a`; its exact staged tree
`e412ecf494dbab217d8585fe1fdb9a92d448135d` passes tests on Go 1.27.1 and 1.26.8,
vet, formatter and pinned strict lint before the signed local commit. No new
API, ABI, manifest option, ownership rule or host contract is introduced.
The publication-view predicate and every runtime admission/memory/reset/reply/
poison/drain check remain unchanged. Guest, host and adversarial reviews approve
retention under this measured scope, with the negative results below.

The [readable record](contained-loads-production-2026-10-12.md),
[actual compressed archive](contained-loads-production-2026-10-12.tar.gz)
and [SHA-256](contained-loads-production-2026-10-12.tar.gz.sha256)
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

Exact-byte compressed evidence. Raw benchmark names, every sample/custom metric, failure logs, derived comparison aliases, scripts and frozen/build/reference provenance are preserved in the bundle. Binary executables and unchanged committed generated modules are identified by hashes and build commands, not embedded. No whitespace normalization changes bundled bytes.

{
  "inventory": [
    {
      "directory": ".cache/contained-loads-production/warm-run",
      "mode": "warm",
      "rows": 826,
      "groups": 118,
      "completed": "2026-10-12T10:29:19.491029+00:00"
    }
  ],
  "payload_artifacts": 1517,
  "tar_members_including_manifest": 1518,
  "compressed_bytes": 3995716,
  "compressed_sha256": "d68c1ee610ab1c987d32f99d40025ca621395ab9e83c031643479d30a0d54c84"
}

Audit without extraction or execution: verify the compressed SHA256, open gzip/tar in memory, then verify each MANIFEST.json byte length and SHA256. All member paths are relative. Reproduce the committed production inputs from frozen revision and exact fingerprints.
