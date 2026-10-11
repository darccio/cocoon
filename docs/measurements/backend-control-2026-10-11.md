# Same-Wasm backend control — 2026-10-11

Retain the private harness and negative evidence, not a production backend
change. All 18 matched warm compiled-wazero raw/checked pairs lose against
translated Go in both rounds (exploratory unadjusted p=.001, n=7 per role).
Checked confirmation: scalar 97.03 vs 203.1 ns; repeated batch 1024 points
39.596 vs 53.541 us; SQL 2.236 vs 4.018 us; SQL holdout 5.641 vs 10.639 us;
trace 2.675899 vs 4.280850 ms (translated / compiled stack).

M1 remains unaccepted. Actual production scalar 88.14 ns and SQL 2.237 us
exceed original limits 55.44 ns / 2.2154 us and fresh reference plus 10%
limits 61.204 ns / 2.1626 us. Fresh spike reference is 55.64 ns / 1.966 us.
Source is unchanged; refresh/historical differences are not source gains.

P16 complete-stream write rates: fixed one/sixteen homes 1,346,721/9,090,607
writes/s; holdout 1,489,689/9,907,774. Existing standalone methods, complete
lifetimes and bounded owned-output retention are included. These refresh
rates are observed workload ceilings, not strict limits or a new source gain.

Private scope 5dee157; controls 9ad3434; entry main=origin/main=e56bfa9.
Measurement binary SHA256
59a3d4159f82a3f91dea9507b4cd24474f43efc19ceceb867400588e9de539cb.
Both engines use production Wasm
ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333;
translated Go
f92c5853385d9c4b3cfc3d164507def81beff36b9f7a8564c92f247c9a3ab948.

Go 1.26.8 linux/amd64, CGO disabled, GOAMD64=v1, explicit PGO off, default
GC/memory; Ryzen 7 5800HS, 8 physical/16 logical cores, unpinned CPUs 0–15,
powersave/boost on. Rust 1.97.0, Binaryen 133, wasm2go 0.4.16, wazero 1.12.0.
All 117 frozen
fingerprints match through timing and final pre-documentation audit.

1,022 warm process samples: seven alternating 500 ms passes then seven reversed
1 s passes. 42 cold samples: seven fresh-process 1x observations per leaf/order.
All pass, none discarded. New matched actual P1, streams P16; original/layer
controls retain their policy without a custom in-window processor metric.
No build/test/lint/fuzz/profile/reviewer tools overlap qualification timing.
Sequential controls cannot prove absence of unrelated system activity.

Compiled exports cache api.Function/signatures and reuse CallWithStack
carriers with cached WithoutCancel contexts. A separate memprofilerate=1,
100000x diagnostic locates 16 B/export at Go 1.26 withoutCancelCtx.Value receiver
boxing, reached by wazero's Value lookup. This is a private adapter-context
contribution, not inherent engine allocation. Profile timings are excluded;
no alternative context or counterfactual speed was qualified.

Raw roles include workload adapters; checked roles use actual runtime policies
but differ from generated facades and omit poisoned-home replacement. Evolving
Increasing prefixes differ with calibration. All 532 matched backend warm
rows have zero measured logical guest growth; cold zero-growth is a placeholder.
Logical/Go heap bytes exclude native executable mappings/RSS. Cold Go
construction ranges 35.858–102.766 us confirm and 50.596–245.257 us screen;
compound construction scopes are nonadditive.

Fresh independent ABI 2 spike rebuild reproduces previous Wasm/Go/binary hashes;
its context/topology/original batch+trace inputs differ. All ten recorded build/
test commands and 48 matched one-step oracles pass before timing. All seven
quality gates including integration pass, quality Go 1.27.1/lint 2.13.1. Final
native Go 1.26 amd64/386 proofs and two whole 61-file proof/runtime reproductions
pass. Unsupported private compiled platforms skip explicitly, not via fallback.
Guest, host and adversarial reviews approve diagnostic evidence retention.

Preserve early proof-construction notes without inventing missing raw stdout,
the wastedassign lint failure, initial reproduction guard detecting that
authored test edit, cache/version-query retry, upstream unused-import warning,
and final passing retries. Functional/profile timings are not qualification.

Detailed tables, intervals, commands, hashes, tests, reviews and scope limits:
docs/performance.md#same-wasm-backend-control-diagnostic-no-go-2026-10-11 and
docs/backend-control.md at the final evidence commit. Resume matching-profile
translation analysis; no public contract/default PGO/backend change follows.

Exact-byte compressed evidence. Raw benchmark names, every sample/custom metric, failure logs, derived comparison aliases, scripts and frozen/build/reference provenance are preserved in the bundle. Binary executables and unchanged committed generated modules are identified by hashes and build commands, not embedded. No whitespace normalization changes bundled bytes.

```json
{
  "inventory": [
    {
      "directory": ".cache/backend-control/warm-run",
      "mode": "warm",
      "rows": 1022,
      "groups": 146,
      "completed": "2026-10-11T20:31:33.634809+00:00"
    },
    {
      "directory": ".cache/backend-control/startup-run",
      "mode": "startup",
      "rows": 42,
      "groups": 6,
      "completed": "2026-10-11T20:31:40.407300+00:00"
    }
  ],
  "payload_artifacts": 1948,
  "tar_members_including_manifest": 1949,
  "compressed_bytes": 745003,
  "compressed_sha256": "7f7085f019c5439645aad1f6c817cb22dff733223f42e51134f4e802b87f296b"
}
```

Audit without extraction or execution: verify the compressed SHA256, open gzip/tar in memory, then verify each MANIFEST.json byte length and SHA256. All member paths are relative. Reproduce the committed production inputs from frozen revision and exact fingerprints.
