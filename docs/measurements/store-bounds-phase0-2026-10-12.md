# Covered store32 bounds: two-site phase-zero record, 2026-10-12

The preregistered disposition is no-go. Original scalar screening is
87.81→88.03 ns (+0.25%, p=.620); confirmation is 88.02→86.71 ns (−1.49%,
p=.005). Both copied-check comparisons improve, but a repeated untouched gain
is required. Increasing scalar gains against both controls in both rounds,
and repeated batch gains against baseline. No repeated detected probe guard
loss was found. The copied-check trace control itself regresses 2.56%/4.04%
(p=.001/.001), so its comparison does not establish a production trace gain.
Keep evidence only, with no production spec, pass, public contract or artifact
change. M1 remains unaccepted: scalar misses absolute and fresh limits, and
SQL confirmation is 2,173 ns against a fresh 2,158.2 ns ceiling.

All 966 samples, 19 groups and 138 round/role cells are retained. There is no
outlier exclusion or selected retiming. Measurements use official Go 1.26.8,
CGO disabled and PGO off, with a freshly rebuilt independent locked reference
and 56 copied source inputs. Profiles run separately. The authored hardener
postpass regenerates all three module variants, preserving 318 previously
qualified contained-load sites. Ordinary assembly removes two guards but
increases export instructions from 642 to 643. Native amd64/386 contract,
wazero and bulk-memory tests, plus bounded helper and semantic-mutant proofs,
precede timing. These do not prove a generic predicate or native ARM/endian
execution, or assign causal nanoseconds to individual checks.

All stream binaries are identical and their role names are brackets only.
Untouched confirmation writes/s: P1 fixed 1.586M / holdout 1.713M; P16 one home
1.308M / 1.464M; P16 sixteen homes 9.488M / 9.328M. All comparisons are
undetected, including adverse fixed sixteen-home medians of +10.96%/+3.25%.
The numerator counts Write calls only, excluding End, and the denominator
includes New/Write/End/Close. These observations do not establish equivalence
or a capacity bound. The complete report includes all nineteen comparisons and custom metrics.

All seven gates pass: check, race, rust, cross, smoke, fuzz and integration.
Quality uses Go 1.27.2; check/lint use Go 1.26.8 and official golangci-lint
2.13.1 because of its known Go 1.27 export-data reader limitation. Two complete
rebuilds reproduce 61 committed proof/runtime files captured before gates.
All 411 frozen hashes still match after fifteen gate/rebuild commands. Staged
snapshot, darna and SSH-signing validation follow packaging; their later logs
are not claimed as embedded in this pre-stage bundle.

The initial go-run source-argument startup failure and corrected emitter
binary driver are preserved. The guest proof discloses provisional preparation
and superseded early logs; final helper/source/native evidence is pinned.
All reviewers agree with the measured no-go. No local build, test, lint, fuzz,
profile or reviewer tools overlap qualified timing; unrelated external system
work cannot be ruled out. Details and resume point are in docs/performance.md
and docs/roadmap.md. Park this exact probe without selected retiming.

Exact-byte compressed evidence. Raw benchmark names, every sample/custom metric, failure logs, derived comparison aliases, scripts and frozen/build/reference provenance are preserved in the bundle. Binary executables and unchanged committed generated modules are identified by hashes and build commands, not embedded. No whitespace normalization changes bundled bytes.

{
  "inventory": [
    {
      "directory": ".cache/next-pattern/paired",
      "mode": "warm",
      "rows": 966,
      "groups": 138,
      "completed": "2026-10-12T15:44:15.961934+00:00"
    }
  ],
  "payload_artifacts": 1914,
  "tar_members_including_manifest": 1915,
  "compressed_bytes": 4382813,
  "compressed_sha256": "12d8423163be04947490407b275d425aff9f690cf68dac47284ebcdb18839bb1"
}

Audit without extraction or execution: verify the compressed SHA256, open gzip/tar in memory, then verify each MANIFEST.json byte length and SHA256. All member paths are relative. Reproduce the unchanged production inputs from frozen revision and exact fingerprints.
