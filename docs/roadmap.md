# Cocoon roadmap

This is the resume point for future development conversations. Updated
2026-10-06. The synchronous M1 framework and Datadog proof are released privately
as v0.1.0. The quality pass and five performance passes are complete, including
the nine-round multi-agent loop, the PGO/resource-call follow-up, and exact
numeric-lowering investigation.
Performance acceptance remains unmet.
Async and HTTP are a separate milestone, not partially implemented M1 features.

## Current working state

The repository now uses ordinary `.git` metadata. The sandbox workaround
`.cocoon-git` has been migrated, with all history preserved. No special Git
environment variables are required.

Go tests, vet, race checks, 49 strict linters, native Rust tests and Clippy,
Wasm verification, differential fuzzing, and cross-platform test compilation
have passed locally. Rebuilding both proof packages reproduces their committed
artifacts and lock hashes. The private repository and first release are
[github.com/darccio/cocoon v0.1.0](https://github.com/darccio/cocoon/releases/tag/v0.1.0).
All three [release qualification CI jobs](https://github.com/darccio/cocoon/actions/runs/37235658925)
passed: native amd64 and arm64 Go quality gates and the fresh pinned end-to-end
build. The release tag points to that qualified code commit. No license has
been selected.

Native linux/386 tests now pass outside the former sandbox. Current runtime
coverage is 97.8 percent; the Rust and Go generators are 94.6 and 96.7 percent.

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

The six tracks below are planned experiments, not implemented features or
measured improvements. Distinguish standalone `SketchAdd` latency from ingestion
throughput: batching and sessions can improve throughput without making an
individual `Add` faster.

Start with a matched-input baseline matrix: scalar versus batch calls, repeated
values, distinct nearby values, mostly unique values, and negative-bin values.
Measure encoding, copies, allocation, admission, reply validation, and guest
execution separately as well as end to end. Existing scalar and 1,000-point
batch benchmarks use different inputs; their per-point timings are not a
controlled speedup comparison. Record cache hit rates and estimated ceilings
before committing to larger designs.

Execution priority is low-copy batching first, with the bin-reuse viability
study as the next scalar investigation. Sessions and typed returns need a
compatibility decision before implementation. Translation analysis and the
alternative-backend comparison are diagnostic tracks; use their results to
decide whether larger compiler or runtime work is justified.

#### 1. Low-copy batching

- First generate encoding directly into a checked guest-memory reservation,
  eliminating the intermediate Go byte buffer and copy. Keep reservation and
  memory access inside the execution lock; do not let borrowed views escape or
  survive memory growth. Benchmark sizes from one point through large batches,
  including allocation and memory-use measurements.
- If guest decoding remains material, investigate a call-scoped borrowed or
  lazy-decoded input path instead of collecting an owned Rust `Vec`. This needs
  a generator/guest API design and lifetime/aliasing review, not an isolated
  generated-file edit. Validate the entire batch before any sketch mutation and
  retain input order, including collapse behavior.
- Gate both stages on malformed input, invalid values late in a batch,
  allocation/growth failures, and exact Count/full-protobuf equivalence. Retain
  each stage only if it gives repeatable end-to-end gains without scalar or SQL
  regressions. The generator change should remain useful beyond sketches.

#### 2. Exact bin reuse

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

- Measure the reply-validation share before designing a new ABI. Explore a
  payload-free successful unit return and direct scalar returns, with explicit
  status/error handling. The measured share limits the possible scalar gain.
- Produce a versioned schema, compatibility/migration plan, and generator/runtime
  design for review before implementation. Keep ABI 3's canonical descriptor
  validation unchanged; this is not permission to skip existing checks.
- Qualify unknown statuses, malformed errors, traps, memory limits, poisoning,
  and resource ownership, plus old/new ABI compatibility. Retain only if the
  end-to-end gain warrants the additional contract and maintenance cost.

#### 5. Translation optimizations

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

The new point-boundary differential tests explicitly assert the reference status
and canonical unit reply. The generic differential comparison still compares
error messages without requiring equal non-OK status codes. Strengthen that
general oracle in a focused quality change; do not infer status equivalence from
matching messages.

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
