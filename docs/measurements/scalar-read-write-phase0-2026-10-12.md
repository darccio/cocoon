# Scalar read and write bounds phase 0 — 2026-10-12

Reject the measured scalar hypothesis and retain evidence only. Production stays
at `20e33db`; no generic transform, production specification, public contract or
generated artifact change follows. M1 performance remains unaccepted.

Measure Go 1.26.8, Linux/amd64, CGO disabled, GOAMD64=v1, PGO off, default GC and
memory settings; Ryzen 7 5800HS with 8 physical/16 logical cores, unpinned CPUs
0–15, powersave and boost enabled. Eight isolated ten-second P1 profiles precede
the ignored three-site whole-source-pinned probe. All builds, tests, lint, fuzz,
profiles and reviewer tools stop during paired timing. Seven alternating 500 ms
passes and seven reversed one-second passes cover 17 groups/59 processes per
pass: 826 raw rows, none removed. All 274 frozen fingerprints match after timing
and gates. Matched leaves verify actual P1 in 280 rows and streams actual P16 in
112; 434 original/layer rows lack an in-window processor metric. Exploratory
p-values are unadjusted; nondetection is not equivalence.

Standalone scalar changes 87.69→87.51 ns (p=.620) and 87.64→87.43 ns (p=.901),
with copied-check p=.334/.876. Raw guest decreases about 0.4–0.5% against both
controls but does not establish facade latency gains. Broad scalar loses in
screening versus unchanged (+0.46%, p=.033) and confirmation versus copied-check
(+0.24%, p=.041); these are different comparators. The declared promotion rule
fails. SQL decreases 2.26%/1.26% versus unchanged, but only fn176 store sites
change and restored checks also improve confirmation SQL; attribution remains
unresolved. Repeated batch has a screening gain that does not confirm. Other
batch/trace comparisons are undetected, with SQL-holdout/broad-batch upward
confirmation medians preserved. Complete tables, intervals and controls remain
in the bundle and docs/performance.md.

The identical streaming executable is used in both bracket roles. Confirmation
fixed writes/s are 1,355,748/1,352,066 with one home and 10,366,603/10,412,237 with
sixteen. Holdout rates are 1,513,103/1,505,239 and 10,514,094/10,560,824. P16,
complete 16 KiB lifetimes, nominal 64-byte chunks, retained groups of 32 and the
independent stateless pool at one. Fixed/holdout streams have 250/268 allocations
and own 16,392/16,390 bytes. These are refreshed workload rates, with no candidate
streaming effect or strict universal ceiling claim.

Current scalar 87.69/87.64 ns misses the 55.44 ns absolute and fresh 110% limits
60.830/60.819 ns. SQL 2.167/2.147 µs meets absolute 2.2154 µs and fresh limits
2.1725/2.1560 µs by only 5.5/9.0 ns, without a durable margin claim. The independent
reference is freshly rebuilt from an absent Cargo target and its own locked
source/shim/hardener; all 56 source hashes, ten commands and 48 functional oracle
leaves pass, with identical prior artifact hashes. ABI 2 ownership/error/context/
topology and original batch/trace input differences remain contextual.

The probe removes exactly three static checks and seven instructions; fn176
shrinks 32 bytes, with frame and static register-store/reload counts unchanged.
Twenty-seven SSA dumps cover fn176, fn96 and the scalar export. Native amd64/386
26-scenario byte/trap/prefix proofs and four fallback/simulated-byte-swap controls
pass; two child mutants fail explicit semantic assertions. The initial 386
sandbox SIGSYS159 and a corrected canonical endian-oracle defect are preserved,
with corrected native proofs repeated before freezing. No generic selector,
actual big-endian or native arm64 execution is proved.

All six gates and integration pass after timing; race uses CGO as required,
separate from measurements. Two proof/runtime rebuilds reproduce all 61
fingerprints. Three-role reviews approve evidence-only retention. Reference
preparation version/cache probes and nested-journal audit correction are
preserved separately from successful build/oracle commands and samples.

Next measure the existing-ABI checked unit-reply specialization/call-shape
ceiling with full validation and an unchanged call-shape control. A new typed
return ABI requires a compatibility specification and explicit operator approval.

Exact-byte compressed evidence. Raw benchmark names, every sample/custom metric, failure logs, derived comparison aliases, scripts and frozen/build/reference provenance are preserved in the bundle. Binary executables and unchanged committed generated modules are identified by hashes and build commands, not embedded. No whitespace normalization changes bundled bytes.

{
  "inventory": [
    {
      "directory": ".cache/scalar-next/warm-run",
      "mode": "warm",
      "rows": 826,
      "groups": 118,
      "completed": "2026-10-12T11:48:28.037887+00:00"
    }
  ],
  "payload_artifacts": 1580,
  "tar_members_including_manifest": 1581,
  "compressed_bytes": 6134108,
  "compressed_sha256": "4080f9d433a339f8cebbf8140ee7010a2075cc665e9442b45f216ebf23d847a3"
}

Audit without extraction or execution: verify the compressed SHA256, open gzip/tar in memory, then verify each MANIFEST.json byte length and SHA256. All member paths are relative. Reproduce the unchanged production inputs from frozen revision and exact fingerprints.
