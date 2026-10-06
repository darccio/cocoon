# Cocoon performance and PGO

Cocoon preserves terminal admission/drain, shared ownership validation, epoch
invalidation, bounded/canonical result checks, and full input validation.
Benchmarks are a gate to investigate, not a reason to remove those checks.

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
