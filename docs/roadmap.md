# Cocoon roadmap

This is the resume point for future development conversations. Updated
2026-10-05. The synchronous M1 framework and Datadog proof are released privately
as v0.1.0. Two performance passes are complete after the quality pass;
performance acceptance remains unmet.
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
coverage is 97.0 percent; the Rust and Go generators are 94.6 and 96.7 percent.

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

1. Use the second-pass scalar profile to choose the next experiment. It attributes
   55.6 percent cumulatively to translated guest execution, 39.2 percent to the
   sketch algorithm, and 11.8 percent to reply validation. The isolated
   call-layer benchmarks (`8d3b074`) are available; keep measuring guest and
   host costs separately, including callback layout and checked reply decoding.
2. Use the second-pass SQL profile to choose guest-side experiments; 85.9 percent
   of samples are cumulatively in the translated export. Its reduced
   allocation count did not close the timing gap. Measure full facade calls as
   well as isolated layers before considering synchronization changes.
3. Optimize measured hot paths without removing bounds checks, canonical reply
   validation, fault containment, shared ownership, or terminal draining Close.
   Add regression tests and regenerate affected proof artifacts with each change.
4. Repeat isolated before/after and independent spike samples. Retain the
   original absolute limits as well as the same-machine comparison. Both
   benchmarks still exceed the fresh same-machine limits, and SketchAdd also
   exceeds its original limit. Do not mark M1 performance accepted.

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
