# Cocoon roadmap

This is the resume point for future development conversations. Updated
2026-10-04. The synchronous M1 framework and Datadog proof are implemented;
performance acceptance and independent CI execution remain open. Async and
HTTP are a separate milestone, not partially implemented M1 features.

## Current working state

The repository now uses ordinary `.git` metadata. The sandbox workaround
`.cocoon-git` has been migrated, with all history preserved. No special Git
environment variables are required.

Go tests, vet, race checks, 49 strict linters, native Rust tests and Clippy,
Wasm verification, differential fuzzing, and cross-platform test compilation
have passed locally. Rebuilding both proof packages reproduces their committed
artifacts and lock hashes. GitHub CI has not been run from this environment.
Native linux/386 tests now pass outside the former sandbox. Current runtime
coverage is 95.6 percent; the Rust and Go generators are 94.6 and 96.7 percent.

## Performance work in progress

The same-machine comparison is complete and recorded in
[performance notes](performance.md), including all five samples and build
provenance. Three tested, darna-validated commits specialize unit guest replies,
cache the fixed output descriptor address, and decode typed replies without an
intermediate copy. SketchAdd improved from 114.4 to 103.2 nanoseconds; SQL remains
about 2.30 microseconds while reducing Go allocations from two to one. Both
original acceptance limits remain unmet: 2.2154 microseconds and 55.44 nanoseconds.

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

## Remaining M1 qualification and quality work

- Run the configured GitHub workflow on a fresh checkout, including native
  arm64 execution, pinned dependency fetching, and reproducible artifact builds.
  A remote destination/publication decision is still needed; none is configured.
- Expand error-path tests where coverage remains lower: generation/publication
  64.3 percent, CLI 70.1 percent, build orchestration 78.4 percent. Runtime,
  hardening, and both generators already exceed 94 percent.
- Consider upstreaming the translator's bulk-memory length-versus-capacity fix.
  Cocoon already applies and tests its own checked AST hardening.

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

Use `make check`, `make race`, and `make integration` for correctness gates;
`make cross`, `make fuzz`, and `make coverage` provide additional qualification.
Benchmark in isolation with Go 1.26.8 and CGO disabled for comparison with the
recorded spike results. Run no compilation, linting, or fuzzing concurrently
with measurements. Local profiles and build caches live in ignored `.cache`;
durable conclusions belong in these committed docs, not only in chat or caches.
