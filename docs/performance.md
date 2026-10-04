# Performance and PGO

The safety contracts add costs absent from the spike: terminal admission/drain,
shared ownership validation, epoch invalidation, bounded/canonical result checks,
and full input validation. Benchmarks are a gate to investigate, not a reason to
remove those checks.

On the local AMD Ryzen 7 5800HS (Go 1.26.8, CGO disabled, three 300 ms runs), the
SQL benchmark using the spike's query measured **2.32–2.34 µs/op**, two Go
allocations. SketchAdd using increasing values measured **113.8–114.4 ns/op**,
zero allocations. The spike's recorded final medians were **2.014 µs** and
**50.4 ns**. The proposed within-10%
target is **not met**. These are recorded-baseline comparisons, not a fresh
same-machine spike run: that checkout lacks its embedded reference Wasm fixture.
Keep this gap visible until measured changes close it.

The runtime now reuses call state under its instance lock, admits lifecycle
calls atomically, avoids unused scalar input reservations, copies string inputs
directly, and consumes guest-owned string/byte inputs without a second clone.
SketchAdd's initial ~162 ns/one-allocation measurement fell to ~114 ns/zero
allocations. Both native Rust and generated Go retain their validation.

Run repeatable samples with `make bench`. For a profile:

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
