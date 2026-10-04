# Cocoon performance and PGO

Cocoon preserves terminal admission/drain, shared ownership validation, epoch
invalidation, bounded/canonical result checks, and full input validation.
Benchmarks are a gate to investigate, not a reason to remove those checks.

## Measured results

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
Current Datadog Wasm SHA-256 is
`2b0b48e84f8d38f705ce49ee52609a49a7e5112dfc59d68d0d257ce3d7becef6`.

## Implemented optimizations and next experiments

The runtime now reuses call state under its instance lock, admits lifecycle
calls atomically, avoids unused scalar input reservations, copies string inputs
directly, and consumes guest-owned string/byte inputs without a second clone.
SketchAdd's initial ~162 ns/one-allocation measurement fell to ~114 ns/zero
allocations before this pass. This pass added typed unit replies, cached the
fixed output descriptor address, and decoded borrowed replies under the guest
lock before copying retained fields. Both native Rust and generated Go retain
their validation, and public `Result` still returns an owned copy.

The post-change five-second SketchAdd CPU profile attributed about 52 percent
of samples to translated guest execution, including 34 percent to the sketch
algorithm. Reply validation accounted for about 9 percent cumulatively.
Resource ownership, instance serialization, lifecycle admission, and generated
callbacks account for much of the remainder; cumulative percentages overlap.
These observations suggest experiments, not proof that a particular rewrite
will help.

Next, isolate direct guest execution, empty runtime calls, and resource calls
with the same workload. Measure Rust hot-path inlining and cold error-path
layout before considering any synchronization changes. Collect a SQL profile
separately: reducing allocation count alone did not close its timing gap.
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
