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

## Performance work in progress

1. Rebuild and benchmark the spike independently on the same machine, without
   importing its implementation into Cocoon. Its recorded final results came
   from a different machine, so retain both the original acceptance thresholds
   and a clearly labeled same-machine comparison.
2. Profile host call overhead and generated guest work separately. The current
   SketchAdd profile spends about half its time in translated guest execution;
   ownership/lifecycle checks and reply validation account for much of the rest.
3. Optimize measured hot paths without removing bounds checks, canonical reply
   validation, fault containment, shared ownership, or terminal draining Close.
   Add regression tests and regenerate affected proof artifacts with each change.
4. Record repeatable before/after samples, allocations, tool versions, and
   comparison limits in [performance notes](performance.md). SQL was about
   2.32 microseconds and SketchAdd 114 nanoseconds before this pass. The original
   within-10-percent thresholds are 2.2154 microseconds and 55.44 nanoseconds.

## Remaining M1 qualification and quality work

- Run the configured GitHub workflow on a fresh checkout, including native
  arm64 execution, pinned dependency fetching, and reproducible artifact builds.
  A remote destination/publication decision is still needed; none is configured.
- Run native 386 tests outside the former sandbox, or under qemu if necessary.
  Cross-compilation already passes; prior execution was blocked by the sandbox.
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
