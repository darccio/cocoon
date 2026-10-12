# Checked unit replies: existing-ABI phase 0, 2026-10-12

Retain evidence only at merged entry 6be3980. PR #7 passed all checks and merged;
local main was synchronized before this experiment. No production specification,
implementation or public contract follows. M1 performance remains unaccepted.

All 924 samples and 430 frozen fingerprints were independently verified.
Default scalar 87.41→87.03 / 88.38→88.59 ns has no detected gain over the untouched
facade (p=.456/1.000), despite beating the slower call-shape control. Varied
scalar repeats about a 1.1% gain. Varied SQL loses 2.28%/3.31%, and SQL holdout
loses 2.18%/2.53% against baseline; the holdout also loses against control in both
rounds. These outcomes fail the premeasurement promotion rule. The isolated
reply component's 18.72%/18.80% improvement cannot be added to scalar savings.
All primary and secondary comparisons, single-round stream losses and noisy
adverse medians remain in the raw output and derived reports.

Measurements use the exact official Go 1.26.8 distribution, CGO disabled,
GOAMD64=v1, PGO off, a Ryzen 7 5800HS, unpinned CPUs 0–15, powersave plus boost
and default GC. Seven alternating 500 ms passes and seven reversed one-second
passes cover 18 groups and 66 processes per pass. Actual-procs metrics identify
280 P1 and 168 P16 streaming rows; 476 original/raw/component rows have framework
P16 without an in-window metric. Discarded Go 1.26.9 preflight, official-versus-
distro ELF differences, requested-profile duration cap and proof-tool failures
are retained. No timing was compared across toolchain distributions or retaken.

Streaming covers complete 16 KiB New/Write/End/Close lifetimes, nominal 64-byte
chunks and Retain32. Untouched confirmation rates are 1,320,480 writes/s with
one home and 9,265,249 with sixteen homes for fixed chunks; the variable/empty
holdout records 1,477,134 and 9,452,007. These are finite-lifetime observations,
not strict capacity or equivalence claims. P1 streaming is unmeasured.

Current scalar misses its 55.44 ns absolute limit and fresh reference caps
60.764/61.226 ns. Current SQL meets the fresh median caps by only 26.8/16.6 ns.
The reference uses its own fresh locked Rust/Binaryen/wasm2go/hardener build;
56 authored source files are preserved. ABI2/error/ownership/context and original
batch/trace corpus differences limit comparison.

All three runtime and propagated generated-consumer proofs pass on native
amd64 and 386; four safe mutants fail explicit semantic assertions on both.
All six required quality gates plus integration pass after timing. Check uses
Go 1.26.8 and pinned lint 2.13.1; the other Go gates use system 1.27.2. The original
Go 1.27.2 lint export-data incompatibility and successful retry are recorded.
Two full builds of all three proofs and runtime generation reproduce all 61
committed entry fingerprints. Exact staged-snapshot tests, vet, format and lint
follow archive creation before the signed commit; their later logs are outside
this immutable timing bundle.

The full narrative, all 18 time comparisons, custom metrics and stream rates
are in performance-section.md inside this bundle and docs/performance.md in
the repository. Executables and unchanged generated production modules are
identified by exact hashes and build commands rather than embedded copies.

Exact-byte compressed evidence. Raw benchmark names, every sample/custom metric, failure logs, derived comparison aliases, scripts and frozen/build/reference provenance are preserved in the bundle. Binary executables and unchanged committed generated modules are identified by hashes and build commands, not embedded. No whitespace normalization changes bundled bytes.

{
  "inventory": [
    {
      "directory": ".cache/unit-reply-phase0/warm-run",
      "mode": "warm",
      "rows": 924,
      "groups": 132,
      "completed": "2026-10-12T14:12:26.834921+00:00"
    }
  ],
  "payload_artifacts": 1863,
  "tar_members_including_manifest": 1864,
  "compressed_bytes": 2502928,
  "compressed_sha256": "731ff8f8fef8c7fd7cdc90d4f43c058d0d7ffafe7e44a1ed2c037757b8befb13"
}

Audit without extraction or execution: verify the compressed SHA256, open gzip/tar in memory, then verify each MANIFEST.json byte length and SHA256. All member paths are relative. Reproduce the unchanged production inputs from frozen revision and exact fingerprints.
