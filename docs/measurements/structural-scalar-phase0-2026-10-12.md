# Structural scalar ceiling: exact log oracle, 2026-10-12

The recorded ceiling rule passes: original checked fixed-prefix scalar improves
87.49→70.26 ns (−19.69%) in screening and 87.74→70.74 ns (−19.38%) in
confirmation versus untouched production. Against the byte-identical disabled
oracle, improvements are 22.30%/21.72%. All four exploratory unadjusted p-values
are .001. This justifies a separately specified bounded feasibility proof only.
The premeasurement rule requires at least10%median time reduction and p<.05
against both untouched and identical disabled controls in both sets.
No production implementation, generic interval certificate, new public
contract or M1 performance acceptance follows.

Entry revision is clean `8a7246e994dd3e1935fb3775e8f6c32aa2c8681e`, two
documentation-only commits above production `6be3980`. Main and origin/main
compare equal at entry. No representative consuming SDK is available; this
is public synthetic Datadog work, not a measurement of the operator's private
SDK. The original inject-bench workspace is absent here. Previous separately
isolated profiles are reused only because the ordinary baseline ELF reproduces
their exact `c73daeca…` identity, not relabelled as fresh or removable shares.

The ignored authored hardener postpass extracts only the exact pure logarithm
block from current fn176, with p17 its only live-out value. Original parameter
reads, checks, rounding, counters, memory writes, error and trap order remain.
All three modules regenerate from committed Wasm; untouched regeneration
matches production, and compute/streaming are identical across variants.
Baseline, pure outline, disabled oracle and enabled oracle run the same
unchanged benchmark harness. False/true modes share one exact ELF and table;
only the logged runtime flag differs. Production runtime/facades, admission,
ownership, reset, cancellation, memory checks, replies, fault poisoning,
epochs, old handles and draining Close remain unchanged.

The oracle is an unsupported 64 MiB offline table of 8,388,608 exact copied
log-bit entries for integer inputs 0..8388607. All other inputs execute the
exact fallback. Table SHA-256 is
`48f45a3568b4ad0862e748c11372f46cc46fe3fcf80236965e11f7990d15314a`.
Measured processes load existing bytes before timing; they do not run the
millions-of-log generation loop in setup. Baseline/outline lack this allocation;
GC/cache/startup/power and code-layout confounds remain against them. Single
one-iteration external-time maxRSS is 13,328/13,876/81,496/81,612 KiB for
baseline/outline/false/true, not qualified RSS. The caller frame grows 128→136 B
and keeps all 78 bounds sites; smaller local text does not predict latency.

Official Go 1.26.8; CGO=0, amd64-v1, PGO off; Ryzen7 5800HS, Linux7.2.8,
powersave/boost, unrestricted CPUs0–15; GOMAXPROCS16 framework. All 520
frozen hashes match through 1,232 raw samples in seven alternating screening
and seven reversed confirmation passes. Most leaves use 500ms/1s; original
scalar, raw guest, matched Increasing and broad scalar use fixed N4194304/
8388608. Original scalar inputs are 1..N; final long input falls back once.
Increasing warms8192, then uses8193..N+8192, with8193 high fallbacks in long
confirmation. Fixed prefixes are diagnostic, not adaptive M1 acceptance.
Actual-procs metrics confirm462P1/224P16 rows;546original/raw rows use framework
P16. No build, test, lint, fuzz, profile or reviewer tool overlaps timing.
All samples, including scalar oracle/reference confirmation ±14% intervals,
are retained. No selected retiming or layout rescue is used.

Pure outlining loses original scalar2.46%/1.63%, raw guest3.09%/2.52%, negative
bins2.07%/2.16%, varied SQL1.05%/1.05%, broad batch3.12%/2.75%; it is rejected.
Enabled fallback-heavy negatives versus untouched repeat for negative bins
2.87%/3.45%, broad scalar3.40%/2.55%, broad batch10.15%/9.51%, and SQL holdout
1.71%/1.49%, all detected. Disabled comparisons on these misses are mostly
undetected, not equivalence or free misses. Trace's confirmation gain versus
untouched has no enabled/disabled gain. Complete19time rows, all comparator
p-values, allocation and custom metrics remain bundled.

Streaming uses one byte-identical ELF across all roles, so its differences
are run variation. Oracle comparisons versus baseline/disabled are all
undetected, including adverse fixedP16one-home confirmation+.56%(.053).
Untouched confirmation writes/s: P1fixed1.606M/holdout1.738M; P16one-home
1.353M/1.514M,16homes10.050M/10.466M. These full-lifetime observations are
not strict ceilings, causal guest gains or scalar/session equivalence.
The numerator counts Write calls excluding End; timed scope is the complete
16KiB New/Write/End/Close lifetime, with Open/warmup/output oracles untimed.

Native amd64 and actual native386 proofs pass for four roles×two packages.
All8Mtable/wrapper checks+1764edge-bit cases compare an exact-source
reconstruction, not an independent ln algorithm or exported Wasm intermediate.
Independent original-Wasm runs compare six complete finite Count/protobuf
histories including16Kprefixes/high4M/8Mtails, errors, recovery and lifetimes;
there is no full4M/8MWasmreplay. RawGuest checks timed fullNCount; bare scalar
harness is unchanged. Four mutants compile and fail named semantic assertions
on both architectures. Native proof and all actual-mode fixture/runtime
contracts/differential/bulk tests pass, but no arbitrary interval certificate
is supplied. Initial test expectation/helper-path/tool-cache failures and
their corrected commands are preserved as preparation negatives.

All seven gates (check/race/rust/cross/smoke/fuzz/integration) on unchanged
production and two repeated builds of all three proofs/runtime pass,
reproducing61fingerprints. Diagnostic overlays have the separate actual-mode
fixture/runtime tests and native proofs above, not production qualification
from these gates. Check uses
Go1.26.8/pinnedgolangci-lint2.13.1 with49strictlinters; other gates Go1.27.2,
Rust1.97/Binaryen133. Cross compilation is not native execution; no native
arm64 performance is claimed. Staged snapshot tests, darna and signed local
commit verification occur after packaging and are not self-embedded.

Next commit a private bounded feasibility specification before implementation:
prove pinned-floating-point membership and fallback (endpoints/assumed
mathematical monotonicity are insufficient), measure synthetic locality and
bounded lookup/certification/miss costs, preserve original checks/state/lifetime
barriers, then decide whether a generic binding/effect/escape predicate is
worth production proof. Park it if costs erase the ceiling. Neither older
exact-value cache is promoted. Any public contract requires explicit approval.

Unsupported confirmation oracle70.74ns still misses absolute55.44ns and
diagnostic fresh-reference+10%60.698ns scalar limits. Fixed-N untouched87.74ns
is not a replacement adaptive M1 sample. Adaptive SQL2126ns narrowly meets
absolute2215.4ns/fresh2130.7ns in this run, not overall acceptance.

Binary executables, emitter executables, the64MiB `table.bin` and redundant
reference generated module are omitted from the bundle. Their exact hashes,
source/overlays/build commands, offline table generator and official toolchain
provenance are retained for reproduction. The table generator can regenerate
the exact little-endian bytes and hash outside timing. All diagnostic module
variants, proof source and semantic-failure logs are preserved. Reference
authored sources are copied with before/after hashes and locked upstream
revision; the reused profile provenance explicitly identifies its prior round.

To reproduce the omitted table, restore the frozen checkout and diagnostic
sources, adapting recorded absolute overlay paths if relocating. Use the
recorded commands in `table-generator-records.json`, but write new output
names so the archived logs remain intact (the original runner opens them with
exclusive creation):

```sh
GOTOOLCHAIN=local CGO_ENABLED=0 GOAMD64=v1 GOFLAGS=-buildvcs=false \
  GOCACHE=/tmp/cocoon-phase0-table-repro-cache \
  PATH="$PWD/.cache/toolchains/go1.26.8/go/bin:$PATH" \
  .cache/toolchains/go1.26.8/go/bin/go test -pgo=off -c \
  -overlay=.cache/structural-phase0/table-generator-overlay.json \
  -o /tmp/cocoon-phase0-table-repro.test ./examples/datadog/go/dd/internal/wasm
COCOON_PHASE0_CREATE_LOG_ORACLE_OUTPUT=/tmp/cocoon-phase0-table-repro.bin \
  /tmp/cocoon-phase0-table-repro.test -test.run='^TestPhase0WriteLogOracle$' -test.v
wc -c /tmp/cocoon-phase0-table-repro.bin
sha256sum /tmp/cocoon-phase0-table-repro.bin
```

Expected size is67,108,864bytes and SHA is48f45a3568b4ad0862e748c11372f46cc46fe3fcf80236965e11f7990d15314a.

Exact-byte compressed evidence. Raw benchmark names, every sample/custom metric, failure logs, derived comparison aliases, scripts and frozen/build/reference provenance are preserved in the bundle. Binary executables and unchanged committed generated modules are identified by hashes and build commands, not embedded. No whitespace normalization changes bundled bytes.

{
  "inventory": [
    {
      "directory": ".cache/structural-phase0/paired",
      "mode": "warm",
      "rows": 1232,
      "groups": 176,
      "completed": "2026-10-12T17:22:54.070131+00:00"
    }
  ],
  "payload_artifacts": 2535,
  "tar_members_including_manifest": 2536,
  "compressed_bytes": 5724951,
  "compressed_sha256": "0cae16262901a2978983aea55d7e66e6157114dd86348979b2f36165d786407e"
}

Audit without extraction or execution: verify the compressed SHA256, open gzip/tar in memory, then verify each MANIFEST.json byte length and SHA256. All member paths are relative. Reproduce the unchanged production inputs from frozen revision and exact fingerprints.
