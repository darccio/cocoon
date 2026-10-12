# Fast certified interval cost evidence, 2026-10-12

**Numerical criteria pass; economic criteria fail. Park the fixed one-slot
candidate unchanged. M1 performance remains unaccepted.** No production cache,
compiler pass, guest/runtime implementation, public contract or private SDK
workload is added. See the complete
[performance results](../performance.md#fast-interval-construction-and-mapping-costs-2026-10-12)
and unchanged historical [private specification](../fast-certified-interval-cost.md).

Specification commit `45d50ac74bf3d2e790f7ba29c95608221b05403f` was signed and
its exact staged tree validated before any new implementation. Its SHA-256 is
`d557897390ddac943f0452d9a640608151ea250632d8dbab3f62b71c7b5ef0f8`.
The branch enters at `7c8afee8ebe7167257a97d31c88e481081e532d3`, following the
prior rational feasibility proof, with local main equal to origin/main at
merged `aeaf01b341a85024804e9ae1abe43af90d36e4d1`. No branch was published.

The [compressed evidence](fast-certified-interval-cost-2026-10-12.tar.gz)
and [SHA-256](fast-certified-interval-cost-2026-10-12.tar.gz.sha256) contain
an exact per-member SHA-256/byte manifest. Members use safe relative names
under `.cache/fast-interval-phase0/`, plus `MANIFEST.json`. ELF executables,
Go/Cargo caches and extraction snapshots are omitted; source, generated output,
build commands, ELF hashes and compiler identities are retained. The original
public fixture source/Wasm and all 61 production fingerprints identify the
unchanged repository inputs. The old four reference files are copied byte
exactly, with the new containment adapter beside them.

The archive is 12,755,380 bytes, with 832 payload files plus the manifest.
Its SHA-256 is
`6db732dbed0dd80d0a267dc673b80b707c0893275bfc673c50f43bc3f462c1c4`.
Packaging verifies every member's type, safe name, size and digest, including
all 546 raw measurements and all 18 intended numerical fault logs.

## Contents and provenance

| Evidence | Location inside the archived ignored directory |
| --- | --- |
| Entry, original production and old proof identities | `entry.json`, `proof-before.json`, `proof-after.json`, `reference-copy.json` |
| Signed specification checkpoint and staged validation | `signed-spec.json`, `spec/`, `frozen-spec.md` |
| Guest/host/adversarial entry and applied reviews | `*-entry.md`, `*-spec-review.md`, `*-implementation-review.md`, source/command journals |
| Native constructor/model/generator and source-ordered output | `proof/native.go`, `proof/native_generated.go`, `proof/internal/generator/`, `proof/cmd/generate-native/` |
| Unchanged rational reference and independent claimed-prior adapter | `proof/reference/`, `proof/adversarial_test.go` |
| One-slot policy, numerical corpus copies, costs and full replay | `proof/cache.go`, `proof/replay_test.go`, `proof/cost_test.go` |
| Admitted constructor/current-memory probe, native/Wasm and corpus checks | `consumer_test.go`, `consumer.mod`, `consumer.sum`, `consumer-overlay.json` |
| Final builds, actual native executions and finite counts | `proof-run.py`, `proof-command-records.jsonl`, `proof-*.log`, `proof-consumer-*.jsonl`, `finite-proof-ledger.json` |
| Compiled expected numerical assertion failures | `mutant-run.py`, `mutant-records-*.json`, `mutant-*.log` |
| Exact full-N lockstep histories/counters/digests | `functional-replay-*.jsonl`, `functional-replay-summary.json` |
| Frozen commands/iterations/source/toolchain/ELF identities | `cost-plan.json`, `cost-frozen.json`, `cost-run.py` |
| Every raw measured row and subprocess command | `costs/`, `cost-rows.jsonl`, `cost-command-records.jsonl`, `cost-complete.json` |
| Component summaries and all exploratory comparisons | `cost-summary.json`, `replay-statistics.json`, `statistics/`, `summarize-replays.py` |
| Preparation/heap/retention and nonadditive break-even | `costs/memory-observations.log`, `allocation-and-break-even.json` |
| Ordinary assembly/escape observations and byteexact regeneration | `compiler-*.txt`, `compiler-observations.json`, `compiler-observations.py`, `proof-logs/guest-escape.log`, `native-regenerated.go.txt` |
| Historical Go rules/math source identities | `primary-contracts.json`, `cost-frozen.json` |
| Machine/CPU/load and PID/PPID/command-only conditions | `measurement-before.json`, `measurement-after.json` |
| Seven production gates and two complete artifact reproductions | `qualify.py`, `qualification-command-records.json`, `gate-*.log`, `repeat*-*.log` |
| Procedural failures, corrections and earlier author checks | `proof-logs/`, `containment-*-tests.log`, focused host/adversarial logs and command journals |

## Result and limits

Both actual native architectures produce the identical 57-certificate bank
from 92 requests, preserving all fixed seeds/widths. The new generated numeric
program ID is
`96a5212f2ba3a8450193fabb491447353b4cd96a0087ca5c9e77cdcfeb54e220`;
the source is `626550ca…dcc298`, Wasm `ae0119ca…8c6333` and bank
`7cbaa190…78686`. Models are 4,208/4,204 bytes, bank plus encoding
2,814/2,554 bytes and active builder workspace plus observer export capacity
56,240/53,152 bytes. Later preparation/summary evidence serialization allocates
outside that bounded numeric scope; these figures do not cap every JSON record.
Logical representations do not bound heap, stack or RSS. The amd64 single
slot is 104 bytes; model storage has its separate 16 KiB cap.

Per architecture, final finite proof sums guest 611 + adversarial 12,431 +
reference 16 + consumer 2,553 + host allocation 260 + host policy 29 +
selected numerical faults 8 = **15,908/16,384 whole evaluations**.
Tiny 114 members also count toward whole; fresh original-Wasm 521 requests
stay below 2,048. Every 4,121 accepted adversarial singleton trace verifies
independently from claimed prior bounds. Every unique guest accepted source,
ownership/allocation witness and all 57 fixed-bank traces are independently
checked; allocation repetitions use the same witness outside the allocation
region. Raw dyadic decoding of 1,075 bits is accounted separately from the previous
235-bit arithmetic observation. Native checker scratch remains guarded by
4,096 bits, not a hard process bound. All 18 compiled fault selections fail
at their intended numerical assertion; preparation errors are not fault credit.

The complete 4M/8M checks have separate declared full-prefix ledgers:
eight corpora × (4,194,304 + 8,388,608) = **100,663,296 inputs per architecture**.
Every enabled and disabled output agrees with full compiled OriginalBin,
and both architectures' counters/digests match. Every input consumes at most
five proposals; each complete replay resets an empty slot. These large pure
mapping comparisons do not pretend to fit the finite mathematical driver cap
or to execute millions of original-Wasm resource histories. Domain fallback
does not label below-threshold inputs as logical bin zero.

The 546 retained samples comprise (15 component scopes + 24 complete replay
roles) × 7 fresh-process samples × 2 sets. Measurement uses official Go 1.26.8,
CGO=0, GOAMD64=v1, PGO off, actual P1, Ryzen 5800HS/Linux 7.2.8, powersave/boost
and no imposed affinity, from 20:39:30 to 20:48:45 UTC. Every source/compiler/
linker/spec/ELF/input identity is frozen before measurements. Each benchmark
slash component is separately anchored. No build/lint/fuzz/profile/memory
or review tool overlaps timing; all failures and rows remain visible.

Confirmation native accepted construction is 927.1 ns, versus 204.660 µs rational;
the full native bank is 61.966 µs versus 12.920 ms. The bank still allocates
one 4,864-byte observer trace; every timed warmed individual candidate, lookup
and complete replay allocates zero. Full independent verification remains
1.081 ms and allocating. Preparation's separate 546-row median is 144.183 ms
and 41,237,552 bytes. A component reuse estimate has negative `O-H`: the direct
Repeated loop is 14.05/14.41 ns, but slot lookup alone 30.42/31.09 ns. No finite
favorable reuse count follows; different argument/control/loop scopes also
prevent additive full-call predictions.

Increasing hits 98.860%/99.310%, but complete enabled 36.96/33.51 ns exceeds
direct 14.39/14.32 ns. Screening's gain versus disabled is 9.88%, below the
frozen 10% threshold. Broad 841.6/840.6 ns and NegativeBins 796.1/798.1 ns have
zero hits and breach both controls' repeated 2% guards. Every other corpus
breaches the direct guard, including almost perfect Repeated/Nearby reuse.
The disabled wrapper is itself slower than direct. All 48 unadjusted benchstat
comparisons are archived (all p=.001 except screening Increasing enabled versus
disabled p=.004); these exploratory tests have no multiplicity correction or
representative-traffic acceptance claim. No sample/policy/width/capacity/retry
was selected or rescued after results.

## Reproduction and qualification

Restore the archived ignored directory beside the unchanged checkout, inspect
the manifest, and use the identified official Go 1.26.8 toolchain. Absolute
paths in the historical temporary consumer modfile/overlay/freeze should be
resolved to that checkout when reproducing elsewhere. Preserve the original
reference bytes and signed specification; do not modify generated modules
alone. `proof-run.py` builds six executables with `-pgo=off` and executes
native tests, and `mutant-run.py` requires assertion failures rather than
startup errors. Use fresh evidence output paths for a new run.

The retained exact driver order is: `proof-run.py build final`, actual
`proof-run.py amd64` and `386`, all nine `mutant-run.py` modes per architecture,
`compiler-observations.py`, and four `proof-run.py replay ARCH N` executions.
Only after applied reviews, complete proof/replays and idle build/review tools
does `cost-run.py freeze` precede `cost-run.py run`. Seven fresh-process samples
per cell follow the committed rotated/reversed 39-cell plan. `cost-run.py memory`
runs separately after measurement; `cost-run.py summarize` and
`summarize-replays.py` retain every component/comparison. Historical numeric
prefixes are 4,194,304/8,388,608 and exact component iterations are in
`cost-plan.json`; reducing them is a different run, not reproduction.

All required production gates pass: `make check`, `race`, `rust`, `cross`,
`smoke`, `fuzz`, `integration`. Check uses official golangci-lint 2.13.1 with
49 strict linters and Go 1.26.8; other gates use system Go 1.27.2 and Rust 1.97.0.
Two repeats of compute, Datadog, streaming and generated runtime builds
reproduce all 61 before fingerprints and committed locks. Private proof receives
its separate actual native checks and Go vet. Darna and an extracted exact
staged snapshot with both Go versions, vet, formatting and the pinned 49 linters
precede each signed local commit. Commit identity is approved Dario Castañé
<d@rio.hn> with the approved local SSH key. No push, PR or merge is performed.

The retained changes are specification and evidence; this fixed policy is
rejected for production/guest integration. Earlier streaming rates remain
historical one/sixteen-home P16 observations (1.353M/10.050M writes/s,
holdouts 1.514M/10.466M), not new numeric results or hard capacity ceilings.
Standalone scalar, SQL, batch and streaming implementations remain unchanged.
Any materially different policy or structural experiment needs a separately
reviewed specification and complete controls. Checked ownership, epochs,
per-operation memory/call-state/reply checks, immediate poisoning and draining
Close remain future per-call duties. No public contract or M1 acceptance is
approved by this numerical diagnostic.
