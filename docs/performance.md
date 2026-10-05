# Cocoon performance and PGO

Cocoon preserves terminal admission/drain, shared ownership validation, epoch
invalidation, bounded/canonical result checks, and full input validation.
Benchmarks are a gate to investigate, not a reason to remove those checks.

## Current results

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
the focused follow-up is saved in the [roadmap](roadmap.md#additional-abi-follow-up).

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
55.44 ns**. Neither is met. Current results also exceed 110 percent of the
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
resource exit defers. Next, use fresh profiles to choose larger guest-side or
callback-layout experiments; adding the screening gains is not valid accounting.
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

Collect profiles separately from unprofiled acceptance samples:

```sh
CGO_ENABLED=0 go test ./examples/datadog/go/dd -run '^$' \
  -bench . -benchtime=5s -cpuprofile=.cache/dd.prof -o .cache/dd.test
go tool pprof -top .cache/dd.prof
```

Collect representative workload profiles rather than only a tiny scalar loop.
Merge compatible CPU profiles with `go tool pprof -proto a.prof b.prof > default.pgo`
in a networked/user shell. Place `default.pgo` beside the consuming main package,
or pass `go build -pgo=/path/to/default.pgo`. Go PGO changes host compilation,
not the pinned Rust/Wasm translation or schema. Compare with `-pgo=off`, retain
profiles only when they represent deployed traffic, and rerun tests/race checks
after changing compiler options. Profiles can contain application symbols and
paths; review them before sharing.
