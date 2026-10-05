# Cocoon roadmap

This is the resume point for future development conversations. Updated
2026-10-05. The synchronous M1 framework and Datadog proof are released privately
as v0.1.0. Performance acceptance is deferred by the user and remains unmet.
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
coverage is 95.6 percent; the Rust and Go generators are 94.6 and 96.7 percent.

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

## Deferred performance work

The user chose to defer further optimization until after the first complete
MVP. Performance acceptance remains unmet; it is not a gate for that release.

The same-machine comparison is complete and recorded in
[performance notes](performance.md), including all five samples and build
provenance. Three tested, darna-validated commits specialize unit guest replies,
cache the fixed output descriptor address, and decode typed replies without an
intermediate copy. SketchAdd improved from 114.4 to 103.2 nanoseconds; SQL remains
about 2.30 microseconds while reducing Go allocations from two to one. Both
original acceptance limits remain unmet: 2.2154 microseconds and 55.44 nanoseconds.
Those measurements precede the final source-path and compiler-metadata
reproducibility fixes; remeasure the release artifacts when performance work
resumes.

1. Add isolated measurements of direct guest execution, empty instance calls,
   and resource calls using identical workloads. The post-change SketchAdd
   profile attributes about 52 percent to translated guest execution and
   9 percent cumulatively to reply validation; remaining host costs need
   separate measurements before changing synchronization.
2. Experiment with Rust hot-path inlining and cold error-path layout, measuring
   each change independently. Profile SQL separately; its reduced allocation
   count did not produce a meaningful timing improvement.
3. Optimize measured hot paths without removing bounds checks, canonical reply
   validation, fault containment, shared ownership, or terminal draining Close.
   Add regression tests and regenerate affected proof artifacts with each change.
4. Repeat isolated before/after and independent spike samples. Retain the
   original absolute limits as well as the same-machine comparison. Neither
   benchmark currently meets either interpretation of the 10-percent target;
   do not mark M1 performance accepted.

## Follow up quality work

The follow-up quality pass is complete as of 2026-10-05. Performance remains
deferred; no async implementation was added in this pass.

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
