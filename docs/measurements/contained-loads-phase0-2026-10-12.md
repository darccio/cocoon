# Contained-load bounds phase-0 record, 2026-10-12


Matching profiles and compiler output identify redundant `load32` endpoint
checks within short straight-line regions. The private probe repeats original,
varied, and holdout SQL gains against both the untouched binary and a copied
helper retaining the checks. Original batch and repeated-value batch also repeat
smaller gains. This warrants a narrowly specified production experiment, not
retention of edited generated modules. **Production remains unchanged at this
checkpoint; M1 performance remains unaccepted.** Standalone scalar and trace
improvements are not established.

The [private production specification](../translation-contained-loads.md) must be
committed before implementation. Guest, host, and adversarial review cover the
probe, proof and interpretation; the phase-0 archive retains their findings and
pre-timing corrections. The [readable record](contained-loads-phase0-2026-10-12.md),
[actual compressed archive](contained-loads-phase0-2026-10-12.tar.gz),
and [SHA-256](contained-loads-phase0-2026-10-12.tar.gz.sha256) preserve
all 770 raw process rows, all custom metrics, profiles, SSA/assembly, fixture and
mutant sources/logs, scripts, provenance, intervals and negatives. No sample or
outlier was discarded. Member bytes have a SHA-256 manifest.

### Conditions and provenance

Begin after PR #6 merges: freshly fetched `main` and `origin/main` both identify
`776b897e74b7faef44375b2c31bd3c17ac88f876`; the branch starts clean. Timing freezes
that unchanged production tree and 189 input/binary fingerprints, all matching
through both rounds and the final audit. Production Wasm remains
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`;
translated Go remains
`f92c5853385d9c4b3cfc3d164507def81beff36b9f7a8564c92f247c9a3ab948`.
Baseline executable SHA-256 is
`59a3d4159f82a3f91dea9507b4cd24474f43efc19ceceb867400588e9de539cb`;
probe is `93d81048dafa25305833a5486eba019733d11e1b44e9c81f820f04882c94e68c`;
recheck is `cf57003adc967ae24cba3958e5e6a6bb6c6043ca64e4783b8e878058be04b38b`.
All three use identical authored harnesses and original-Wasm oracles. Neither
an overlay nor a generated-module-only edit is retained in production.

Measure Go 1.26.8, Linux/amd64, CGO disabled, `GOAMD64=v1`, explicit `-pgo=off`,
default GC/memory settings, Ryzen 7 5800HS (8 physical/16 logical cores), unpinned
CPUs 0–15, powersave governor and boost enabled. Rust 1.97.0, Binaryen 133,
wasm2go 0.4.16 and wazero 1.12.0 are pinned. CPU policy stays unchanged. Exact
commands, UTC timestamps, executable build info and environment are archived.

Seven alternating 500 ms passes followed by seven reversed one-second passes
run 55 fresh process invocations each: twelve original/corpus leaves with four
roles, one raw guest leaf with three, and four unchanged streaming leaves. This
produces 770 rows, with seven samples per case/round. Matched leaves verify P1
in-window (280 rows); streams verify P16 (56). The 434 original/layer rows retain
framework CPU 16 and lack an in-window processor metric; the suffix does not
prove timed processors. Separate ten-second P1 CPU profiles precede timing and
are excluded from acceptance samples. No compilation, tests, lint, fuzzing,
profiling or reviewer tools overlap qualification timing. The process audit
cannot exclude unrelated system activity. No cold/startup comparison was run.

The independent unversioned spike is freshly rebuilt from its own locked Rust
inputs and clean libdatadog `7f3b16fe1b4bfc2a016ef869459e915ee02d6d6a` dependency, with ten journaled commands;
its original native/wazero tests and 48 matched one-iteration oracles pass.
Reference Wasm/Go hashes reproduce the preceding round; the executable is
`f8846dd93bd22b1adcf3b7762f685c8e96199a9d77a62f7d8ad9f1639a48bbea`.
The archive preserves 56 selected reference source files and their hashes.
Reference ABI 2 borrowed inputs, background context and four-instance topology
remain different from ABI 3 checked owned inputs, test contexts and one home.
Original batch/trace inputs also differ; reference ratios are contextual.
Increasing-scalar roles consume different calibrated prefixes of the same
ordered corpus, so they describe evolving state rather than equal final states.

### Pattern, machine code, and executable proof

Five separate sequential CPU profiles locate translated guest work: `fn176`
flat/cumulative samples are 32.39%/42.00% of checked scalar, 54.33%/70.50% of
raw scalar, and 57.18%/78.46% of original batch. `fn96` is 46.99%/77.84% of SQL
and 36.63%/64.66% of trace. Inlined/cumulative shares overlap and are not additive
attainable gains. An existing Go 1.24 pprof viewer reads the Go 1.26 profile;
ordinary Go 1.26 opt/regalloc/genssa dumps and matching assembly are retained.

The sketch region reads offsets four, eight, then zero on one unchanged local
base. Keep the first endpoint check (+7) and new maximum (+11) at their original
positions; only the final +3 check is contained. The diagnostic selects 319
later checks across 69 methods by bound local identity and straight-line range
facts. Every guest read stays live; no stores/loads are forwarded, no header is
cached, and no bounds check moves. Unknown calls, imports, writes, growth,
rebinding, declarations, labels and other control boundaries clear facts.
The helper's unaligned fallback/endian semantics remain intact. The whole-source
hash pins helpers, construction, bindings and architecture assumptions for this
diagnostic; it is not a generic production proof.

| Ordinary symbol | Baseline/recheck text, frame, static panic sites | Probe text, frame, static panic sites |
| --- | ---: | ---: |
| Sketch body `fn176` | 5,834 B; 144 B; 88 | 5,462 B; 128 B; 78 |
| SQL body `fn96` | 111,763 B; 856 B; 1,501 | 110,236 B; 856 B; 1,452 |
| Scalar export | 3,004 B; 112 B; 57 | 3,004 B; 112 B; 57 |

These are static sites, not dynamic counts. The sketch frame changes, and the
helper/call rename can affect placement and register allocation. The recheck
control restores baseline text sizes/frames/site counts but is not a perfect
causal control. No claim of fewer spills or a pure per-check cost follows.

The authored WAT fixture compares nine exports against wazero's interpreter on
16 bases and two flag values each: results, traps, import counts, and full memory
including growth. Only the intended two contained fixture reads are selected;
rising ranges, stores, imports, rebinding, wrapping and conditional entry retain
checks. Native proofs add logical lengths 0–20 with spare capacity, exact/short
endpoints, unaligned and high-bit addresses, import-side shrink/backing replacement,
first-trap prefixes, labels, global bases and an impure local `i32` callback.
Original, probe and recheck pass native Go 1.26 amd64 and 386. An explicit checked
unaligned fallback also passes. There is no native arm64 execution evidence.

Three semantic mutants fail the executable oracle: removing the first check,
carrying a fact across an import that shrinks the header, and trusting a local
impure identity callback by name. The callback is declared before the first
checked read so a declaration barrier cannot mask the name-only purity bug.
All three whole Datadog packages pass their original contract, differential,
backend and corpus tests before timing. The checked facade still performs every
runtime memory/reset/ownership/reply/poison/drain check.

### Paired outcomes and negative results

Entries are exact sample medians, screening → confirmation. Units are retained
per row. Change is probe/baseline minus one. All raw samples, every allocation
and throughput metric, intervals, reference/recheck comparisons, and exploratory
unadjusted p-values remain in the archive; nondetection is not equivalence.

| Workload | Baseline, screen → confirm | Probe, screen → confirm | Time change, screen → confirm |
| --- | ---: | ---: | ---: |
| Original SQL | 2.205 → 2.197 µs | 2.116 → 2.116 µs | -4.04% → -3.69% |
| Original scalar | 88.02 → 87.62 ns | 87.75 → 87.49 ns | -0.31% → -0.15% |
| Original batch 1000 | 38.716 → 38.551 µs | 38.047 → 37.904 µs | -1.73% → -1.68% |
| Original trace 1000 | 2.622961 → 2.600098 ms | 2.614163 → 2.597131 ms | -0.34% → -0.11% |
| Varied scalar | 88.39 → 88.75 ns | 87.71 → 88.24 ns | -0.77% → -0.57% |
| Varied SQL | 2.812 → 2.800 µs | 2.675 → 2.655 µs | -4.87% → -5.18% |
| Negative bins | 89.28 → 88.28 ns | 88.02 → 87.65 ns | -1.41% → -0.71% |
| SQL holdout, P1 | 5.671 → 5.665 µs | 5.446 → 5.471 µs | -3.97% → -3.42% |
| Increasing scalar, P1 | 89.18 → 89.15 ns | 88.65 → 88.91 ns | -0.59% → -0.27% |
| Broad scalar, P1 | 89.42 → 89.14 ns | 89.04 → 88.81 ns | -0.42% → -0.37% |
| Repeated batch 1024, P1 | 39.402 → 39.389 µs | 39.116 → 39.066 µs | -0.73% → -0.82% |
| Broad batch 8192, P1 | 319.311 → 319.300 µs | 313.224 → 312.823 µs | -1.91% → -2.03% |
| Raw guest scalar | 53.92 → 53.76 ns | 52.55 → 52.66 ns | -2.54% → -2.05% |

Original and varied SQL each detect the probe benefit versus both baseline and
recheck in both rounds (p=.001 throughout). SQL holdout versus baseline has
p=.001/.017, versus recheck .001/.011. Original batch repeats a −1.73%/−1.68%
benefit (p=.001 both); repeated batch repeats −0.73%/−0.82% (p=.001/.010).
Broad batch's −1.91%/−2.03% baseline differences have p=.128/.001: only the
confirmation difference is detected. Against recheck it has p=.026/.001.
This does not establish a repeated detected broad-batch baseline gain.

Original scalar's −0.31%/−0.15% is undetected (p=1.000/.833); the raw guest
improves −2.54%/−2.05% (p=.023/.001), which does not establish facade latency
improvement. Varied scalar versus baseline is undetected/detected (.165/.002)
but versus recheck remains undetected (.128/.089). Negative bins versus baseline
has p=.005/.026; versus recheck .128/.004. Increasing and broad scalar differences
remain undetected in both rounds. No repeated probe scalar loss appears, but the
rename/control effects prohibit a general standalone speedup claim.

Trace's −0.34%/−0.11% is undetected (p=.620/1.000). The recheck trace rises
+0.34%/+1.07% against baseline (p=.318/.011); the confirmation control slowdown
is retained as a negative. Removing checks therefore does not supply a universal
workload gain. Probe/reference confirmation ratios remain scalar **1.583×**,
SQL **1.090×**, batch **1.065×**, trace **1.299×**; these include the reference
contract/input differences described above. M1's original absolute limits remain
55.44 ns scalar and 2.2154 µs SQL. Fresh reference plus 10% gives 60.786/60.808 ns
scalar and 2.1472/2.1362 µs SQL (screen/confirmation). The diagnostic SQL is below
both sets of SQL limits, but scalar misses both by a wide margin. A private probe
cannot qualify production M1, and SQL alone cannot satisfy the full acceptance gate.

Scalar/batch allocate zero in all three translated roles. Original SQL reports
57 → 52 B/one allocation (screen → confirmation); holdout metrics are preserved. Trace medians vary slightly
with allocator calibration while retaining one allocation; this is not a
memory-footprint optimization. Matched corpus/batch leaves retain timed current linear-memory and growth
metrics; these are neither process RSS nor peak/resident memory. Streaming leaves
do not report timed linear-memory metrics.

### Streaming ceiling refresh and gates

The streaming binary and source are unchanged: these are P16 refresh controls,
not before/after qualification of a generic transform. Fixed 64-byte streams have
256 writes and 16,392 owned output bytes; holdout metrics retain their varying
chunk history. All streaming contract, wazero and bulk-memory tests still pass.

| P16 control | Time, screen → confirm | Streams/s, screen → confirm | Writes/s, screen → confirm |
| --- | ---: | ---: | ---: |
| One home, fixed | 181.692 → 181.957 µs | 5,504 → 5,496 | 1,408,979 → 1,406,929 |
| 16 homes, fixed | 23.459 → 23.970 µs | 42,628 → 41,719 | 10,912,669 → 10,680,060 |
| One home, holdout | 221.516 → 218.258 µs | 4,514 → 4,582 | 1,550,678 → 1,573,824 |
| 16 homes, holdout | 31.340 → 31.472 µs | 31,908 → 31,774 | 10,960,348 → 10,914,305 |

Fixed confirmation throughput scales 7.59× from one home to sixteen homes;
holdout scales 6.93×. These describe the existing checked resource topology,
not a new streaming improvement. Owned output bytes remain 16,392/16,390 and
allocations 250/268 for fixed/holdout.

All required gates pass sequentially before isolated timing: `make check`
(Go 1.27.1, pinned golangci-lint 2.13.1), `race`, `rust`, `cross`, `smoke`,
`fuzz`, plus `integration`. Two complete compute/Datadog/streaming proof builds
and runtime regeneration reproduce all 61 tracked artifact fingerprints and
locks. Native 386 needs an escalated runner because of the sandbox; its initial
approval timed out before execution, and the permitted retry passes. Cross
compilation covers arm64, 386, Darwin, Windows and js/wasm, not native arm64 timing.
Staged-snapshot validation is required before committing this documentation.
These gates qualify the unchanged production tree and diagnostic proof, not a
future generic pass; implementation must regenerate and repeat its own gates.

Procedural negatives remain archived: the first fixture invocation treated Go
payload filenames as additional `go run` sources and failed before tests; adding
`--` fixes it. Initial helper AST positions caused excess formatter line breaks,
corrected before any overlay build/timing. Review rejects global address bases,
reserved-name collisions and name-only identity purity before freeze. The older
182-file freeze is preserved; no measurement used it. Initial frame parsing
missed `ADDQ $-0x80`; corrected machine evidence records the true 128-byte frame.
The independent reference's unused-import warning is retained. None is a discarded
benchmark sample. The next step is implementing the committed generic predicate
and independently qualifying it; phase-0 results must not be added to earlier gains.

Exact-byte compressed evidence. Raw benchmark names, every sample/custom metric, failure logs, derived comparison aliases, scripts and frozen/build/reference provenance are preserved in the bundle. Binary executables and unchanged committed generated modules are identified by hashes and build commands, not embedded. No whitespace normalization changes bundled bytes.

{
  "inventory": [
    {
      "directory": ".cache/translation-next/warm-run",
      "mode": "warm",
      "rows": 770,
      "groups": 110,
      "completed": "2026-10-12T08:39:49.096960+00:00"
    }
  ],
  "payload_artifacts": 1503,
  "tar_members_including_manifest": 1504,
  "compressed_bytes": 3993150,
  "compressed_sha256": "7e4c1ea67483110a46aef9f784eb405c787e1a3289ab5c902a8fd4434e9c29f2"
}

Audit without extraction or execution: verify the compressed SHA256, open gzip/tar in memory, then verify each MANIFEST.json byte length and SHA256. All member paths are relative. Reproduce the committed production inputs from frozen revision and exact fingerprints.
