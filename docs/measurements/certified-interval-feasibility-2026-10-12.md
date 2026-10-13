# Certified-interval feasibility, 2026-10-12

Positive numerical proof only. Signed specification `8c8791b` preceded ignored
implementation on merged PR #8/main `aeaf01b`. Production source, API, modules,
proof artifacts and locks remain unchanged. M1 performance remains unaccepted.

The fixed procedure yields 57 non-singleton certificates in 92 attempts across
23 original exponent fields, both bin signs, and one interval covering all
8,192 nearby inputs. All accepted traces verify independently; executed amd64
and 386 bank bytes agree, SHA-256
`7cbaa190712acef7949eb7cb61724721f43c226499d9904d4a61cf3be7778686`.
Each architecture has 521 fresh original-Wasm bin/Count/full-protobuf checks.
Nine compiled numerical faults fail their intended assertion. The final driver
uses 13,329 whole evaluations, 626 tiny members included in that whole count,
and 521 Wasm requests within the fixed limits. Universal membership rests on
exact source reduction, rounded transfer induction and strict bin containment.

All 84 raw isolated cost rows remain: official Go 1.26.8, CGO=0, amd64v1,
actual P1, PGO off; seven samples per scope in each of two rotated/reversed
orders. Complete construction medians are 12.413/12.257 ms, or 69.354/69.607 ms
with required verification. Confirmation allocates 6.35/22.95 MB respectively.
Accepted individual requests cost 199.829 µs; early rejection 8.841 µs;
evaluated boundary rejection 200.786 µs; full checker 975.270 µs. Source
preparation, witness setup, framework 1x probe and scope warm-up are excluded
and disclosed. Raw source-preparation metrics and separate GC-disabled
MemStats remain; these are not peak RSS observations or hard heap bounds.

Actual bank plus canonical encoding is 2,814/2,554 bytes, complete logical
builder workspace 47,964/47,192 bytes on amd64/386. Allocating reference heap
is separate. The checker arithmetic/rounder 235-bit ledger excludes its raw
1,075-bit decoder denominator, which is separately guarded; 235 is not an
overall scratch maximum. Offline-bank hits are repeated/nearby 100%, Unique
0.110%, MostlyUnique 7.727%, Phases 9.375%, Negative 12.5%, Broad 0%, and
Increasing 1,806/4,194,304 (0.043%). First/revisit counts agree; all mismatched
configuration/source hits are zero. No eviction, fill or miss computation.

All seven required quality gates pass on unchanged production; two builds of
all three proof packages and runtime match 61 committed artifact/lock/harness
fingerprints. Native diagnostic overlays receive separate execution. Earlier
author checks, cache/compiler failures and pre-review builds are retained
separately from the final capped driver. Process metadata intentionally omits
unrelated shell-wrapper arguments; no sample or numerical metric is altered.

Retain the proof and cost evidence. Reject online promotion of the rational
reference and sparse offline bank. Next separately specify a fast bounded
constructor or lookup-cost control, including trusted rounding, setup/retention
and misses. Consider guest integration and eligibility only if warranted.
No scalar, SQL, batch, new streaming, native arm64 or private SDK claim follows.
See [full results and limits](../performance.md#bounded-numerical-feasibility-certified-intervals-2026-10-12)
and [the specification](../certified-interval-feasibility.md).

## Archive inventory

```json
{
  "payload_files": 218,
  "members_including_manifest": 219,
  "compressed_bytes": 728515,
  "compressed_sha256": "db4ddd2b579734fc7c5b3659c13f2e72d46147ca76c2071c5044b70122e26880",
  "cost_rows": 84,
  "native_architectures": [
    "amd64",
    "386"
  ],
  "production_files_unchanged": 61
}
```

The archive retains exact private source, graphs, certificates, traces, all
cost rows, coverage phases, full protobufs, reviews, failures, commands,
conditions, frozen provenance and quality/reproduction logs. Compiler, ELFs
and build caches are omitted; hashes and build commands identify them.

Audit the compressed SHA-256 first, then open gzip/tar in memory and verify each
MANIFEST.json byte length and hash. Members are regular files with safe relative
paths. EVIDENCE.md in the bundle gives reproduction steps. Final documentation,
staged validation and signing follow bundle creation and are recorded by the
signed Git checkpoint and local retention logs.
