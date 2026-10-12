# Same-Wasm backend control

This diagnostic follows the publication-memory qualification merged in
`e56bfa9`. Its purpose is to locate the remaining standalone scalar and
guest-heavy costs before proposing another optimization. It introduces no
runtime backend option, public API, ABI, manifest policy, or default PGO profile.
The no-engine production promise and checked ABI 3 remain unchanged.

Commit this scope before qualification timing. Retain every sample and negative
result. Backend speed is a hypothesis; this experiment cannot accept a runtime
switch, approximate math, a standalone source improvement, or M1 performance.

## Matching inputs and state

Both engines execute the exact production Datadog Wasm with SHA-256
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`.
Its translated/hardened Go has SHA-256
`f92c5853385d9c4b3cfc3d164507def81beff36b9f7a8564c92f247c9a3ab948`.
Check both against the committed lock and record compiler/tool/dependency pins.
Neither module, facade, runtime, nor generated adapter changes in this round.

Reuse authored deterministic corpora and their operation order. Warm the same
ordered group before timing, start resources from corresponding states, retain
bounded outputs, and verify Count and current memory/growth outside timing.
Scalar cases cover repeated, increasing, negative-bin, and broad held-out
values. Matched batches cover repeated 1024 and broad held-out 8192 points.
Increasing scalar state can evolve beyond finite warmup; report any observed
growth and retained-state effects instead of assuming it remains stationary.
Different calibrated iteration counts traverse different increasing prefixes;
this case is an evolving-state control, not a stationary backend ratio.
SQL covers the original query and the retained holdout corpus; traces use the
original 1000-span payload. SQL/trace output oracles compare complete bytes.
Original workload controls remain separate from matched groups.

## Test-only adapters and safety work

Use explicit `wazero.NewRuntimeConfigCompiler` from pinned wazero 1.12.0;
disable `WithCloseOnContextDone` to match synchronous nonpreemptible execution.
Record every nondefault runtime/module configuration and compilation-cache
identity. Bind the same capabilities through `rt.Host`, with matching memory
limits, ABI/schema identity, and initialized output-descriptor caching.
Reacquire current linear memory after guest calls; never retain a stale view.

Cache exported `api.Function` values outside timing. The primary wazero adapter
uses `CallWithStack` and a per-instance reusable carrier. Refill every argument
slot before calling because results overwrite parameters; normalize i32 result
bits, check errors before reading results, and serialize the carrier under the
instance execution lock. A small raw cached-`Call` control measures API allocation
effects for repeated scalar and repeated batch only. Lookup or signature
inspection must not run inside timed calls.

Both matched checked roles use a shared private wrapper through actual
`rt.Lifecycle`, `rt.Resource.Use`, and `rt.Instance.Call`, with the generated
sizing, numeric/string/byte input encoding, complete canonical reply validation,
typed decoding/owned output, call-state reset, memory-limit checks, immediate
protocol/fault poisoning, epoch invalidation, and draining shutdown. Scalar
Add/New/Count/Encode/Close omit `PrepareInput`, matching the current generated
facade; variable-input calls reserve and account for aggregate bytes, including
empty variable inputs. Stateless SQL/trace use the same lazy single-slot
`rt.Pool` and `Recycle(65536, maximumMemory)` policy on both engines.

Label these roles **CheckedControl**, not generated-facade or supported backend
performance. The shared test adapter adds dispatch/indirection and can affect
inlining/code placement. Record actual generated facade and Guest/Instance/
Resource/full-facade layer controls separately; differences between separately
timed layers are rough ceilings, not additive component shares. Raw guest cases
omit host checks and have no public-operation latency claim.
Their scope includes export dispatch and workload adapters: scalar/batch check
status, while SQL/trace also reserve, copy inputs, and decode owned outputs.
Stateless CheckedControl retains an unused home as well as its measured pool
guest; `guest-B` describes the measured module, not total retained memory.
The private wrapper does not replace a poisoned resource home. Old handles
invalidate immediately, but it models fault-free warm work and containment,
not the generated facade's recovery path. Production replacement is unchanged.

The `rt.ABI` interface cannot return engine errors. The private wazero adapter
must propagate an actual engine error into the existing `rt.Instance` recovery
boundary, preserving the cause while poisoning the instance. Engine trap text
and representation can differ from translated Go. Verify fault containment,
epoch/ownership behavior, and successful/expected-error status bytes without
claiming identical backend-specific fault messages or kinds.

Before timing, prove exact statuses and replies across ordered Count/protobuf,
SQL, trace, empty-input, and invalid-tail histories. Verify expected-error
recovery, malformed reply poisoning, real engine/host fault containment, stale
handle rejection, wrapper-alias Close, growth/current-memory access, retained
output ownership, canceled admission, and draining shutdown. Unsupported native
compiler platforms must be reported rather than silently using the interpreter
for a compiled-backend performance claim.

## Warm and cold measurements

Hot execution excludes runtime creation, compilation, module instantiation,
identity/init, warmup, first-use capacity setup, and teardown on both engines.
Record Go allocations, current linear-memory bytes, timed growth, throughput,
and actual GOMAXPROCS. Setup amortization and retained-state differences remain
limitations; zero timed growth is not a process memory bound.

Each compiled guest has its own runtime and capability host, with independent
memory/globals and a harness-owned shared immutable-code compilation cache.
The cache avoids a timed cross-module capability dispatcher. Record its lifetime
and identity; creating a fresh runtime with that cache is not cold compilation.

Separate cold leaves record uncached fresh-runtime compilation/close,
cached compilation plus runtime/import setup, instantiation, identity/init/close,
and translated Go construction plus identity/init/close. Name the shared-cache
leaf `CachedCompileInstantiateInitClose`, because cached decoding/validation
and lookup are included. Seed that cache with the same Wasm outside timing;
keep the seed/code and cache alive for the measured leaf, then release them
afterward. Each benchmark process owns a fresh cache; no disk cache is used.
The uncached leaf uses neither a shared nor a disk cache and closes compiled
code/runtime within its timed scope. Cold leaves run seven fresh-process
single-operation samples in each of two reversed-order rounds (14 per leaf),
using `-test.benchtime=1x`, independently of the warm-duration samples below.
Cache identity and timed boundaries are explicit.
These independently measured scopes are not additive startup accounting.
Go heap and linear memory exclude native executable mappings; report that
limitation unless a separate-process footprint control measures them directly.
The private guest constructor reads `Output` once more after `rt.NewInstance`
has cached the descriptor privately. This is outside warm timing and included
in both construction controls; they do not represent production startup.
Both construction controls include private dispatch-adapter setup; wazero also
caches export functions and signatures. Closing a guest runtime preserves
code in its factory-owned cache, which is released after all guests close.

Use Go 1.26.8, CGO disabled, `GOAMD64=v1`, explicit `-pgo=off`, default GC/memory,
and recorded CPU affinity/governor. Freeze matching binaries and all harness,
runtime, generated-module, Wasm, overlay, and runner inputs before timing.
Each actual measured new warm or cold leaf sets P1 before resetting the timer,
asserts it immediately, asserts it again after StopTimer, and reports
`actual-procs`. Original controls preserve their existing processor policy and
disclose missing in-window processor metrics.
Seven alternating 500 ms samples precede seven one-second samples with leaf
and role order reversed. Preserve all raw outputs, failures, custom metrics,
hashes, commands, timestamps, intervals, and exploratory unadjusted p-values.
No compilation, tests, lint, fuzzing, profiling, or reviewer tools overlap any
timing window. Collect any profiles separately.

Rebuild the independent spike fresh for M1 reference controls. Its ABI 2 facade,
context, topology, and differing original batch/trace inputs remain limitations;
same-Wasm engine comparisons use the checked ABI 3 fixture instead. Fresh
scalar/SQL reference medians do not replace the original absolute limits.
Repeat original scalar/SQL/batch/trace, varied workloads, negative bins, matched
holdouts, and bounded streaming controls. No code gain can be inferred from a
newly linked harness or historical measurements on different binaries.

## Retention and reporting

Retain a useful honest harness and durable evidence even if wazero loses. Report
which workloads are guest- versus boundary-sensitive, with no unsupported causal
subtraction or general backend claim. M1 remains unaccepted until the final
production scalar/SQL satisfy both original limits and fresh within-10-percent
reference comparisons, with the existing safety and cross-workload guards.

Use guest/host/adversarial review, darna, and exact staged validation before
signed local commits. Complete all required quality gates and integration, plus
two proof/runtime reproductions; no generated-only changes are permitted.
Archive complete evidence in a compressed file with byte hashes and a readable
summary. Publication remains the operator's decision; do not push this round
without a new publication request.

## Qualification notes (2026-10-11)

The control was implemented in `9ad3434` after the initial scope commit
`5dee157`. All 1,064 planned samples succeed; all 18 primary warm engine pairs
lose in both rounds. Retain the diagnostic harness and
[complete negative evidence](performance.md#same-wasm-backend-control-diagnostic-no-go-2026-10-11).
No production/backend change or M1 acceptance follows.

Engine exports cache `context.WithoutCancel` contexts. A separate allocation
profile on the frozen binary attributes the 16 B/export allocation to that
context's Value receiver/interface conversion, reached by wazero's value
lookup. This is an adapter contribution, not an inherent compiled-engine
allocation claim. No alternative context was timed, and no subtraction or
counterfactual speed estimate is qualified.

Cold construction's zero-growth metric is a placeholder generated from final
memory bytes twice; it does not measure construction-time growth. Warm growth
is measured and zero in all 532 matched backend warm samples. Executable
mappings/RSS remain unmeasured. Translated cold construction has wide sample
dispersion, and all
three cold scopes include different work. These limitations leave the private
scope intact and require separate evidence for broader conclusions.
