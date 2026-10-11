# Cocoon performance and PGO

Cocoon preserves terminal admission/drain, shared ownership validation, epoch
invalidation, bounded/canonical result checks, and full input validation.
Benchmarks are a gate to investigate, not a reason to remove those checks.

## Translation analysis: positive phase 0 (2026-10-11)

The memory-view diagnostic finds a promising batch optimization. Reusing the
linear-memory slice header in just two sketch result-publication regions lowers
the original batch's measured time by 11.21% in the narrow screen and 11.86% in
confirmation. The matched repeated-value batch improves by 11.55% and 12.80%.
The broad batch holdout has no detected gain. Retain the SQL holdout harness,
[raw evidence](measurements/translation-phase0.txt), and
[production experiment requirements](translation-memory-views.md). No compiler
pass, generated-module edit, public API, lock change, or default profile is
retained. M1 performance remains unaccepted.

### Matching profiles and machine code

Start from `5a5c8df` on a clean tree; `main` and freshly fetched `origin/main`
both equal `1510d69`. The profile binary has SHA-256
`cd533f99b56712793f8456c4188a874399711b1fe7052101be87750ebb4eb82e`,
identical to the preceding frozen untouched binary. The unchanged translated
Datadog module hashes to
`f50d20e7fe091569353ac1ced7865a53cd23bc9add668c3aac0f5f3a0ac8eed4`;
its original Wasm hashes to
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`.
Five sequential ten-second CPU-profile benchmarks precede timing. Profiles are
diagnostics, not timing samples. Compiler SSA opt/regalloc/genssa dumps and
disassembly use matching Go 1.26.8 production source and ordinary optimization
flags; dump flags add no optimization. The timing binaries include the new
holdout harness and therefore have different linking from the profile binary.

| Profile | Guest hotspot | Flat | Cumulative |
| --- | --- | ---: | ---: |
| Checked scalar | Sketch export | 7.33% | 51.54% |
| Checked scalar | `fn176` | 29.81% | 37.39% |
| Raw scalar guest | `fn176` | 47.71% | 63.47% |
| SQL | `fn96` | 45.44% | 75.61% |
| Batch 1k | `fn176` | 60.32% | 80.62% |
| Trace 1k | `fn96` | 37.35% | 63.06% |

Checked scalar reply validation accounts for 11.99% cumulatively. Inlined
helpers and containing functions overlap; these percentages are attribution,
not additive attainable gains. The earlier conclusion still holds: the dense
log polynomial is register-resident, explicit rounding is meaningful, and no
new exact math simplification is justified.

The demonstrated pattern is surviving memory-header reloads after inlined
unsafe stores. A SQL initializer reloads the pointer and logical length after
each store, and its two later stack-slot loads also survive. Broad stack-value
promotion is rejected as a starting point: addresses cross calls, generic
pointers can alias, and wrapping i32 addition differs from widened addition.
Keeping every original access while reusing only the slice header has a smaller
proof obligation.

### Diagnostic scope and controls

The ignored AST utility requires the exact full-module SHA. It recognizes
simple straight-line scalar regions, predeclares slice locals to preserve Go
goto rules, and assigns a view at the original first memory-containing statement.
Unproved calls, imports, growth, header mutation, labels, branches, gotos, returns,
nested blocks, declarations, and unsupported expressions are barriers. All loads,
stores, casts, helper checks, and stack restorations retain their original order.
It does not forward values or change logarithms, validation, reply publication,
or checked host work. The ownership precondition is ordinary generated-facade
construction, with separate Module and backing-byte allocations; fabricated
unsafe raw-host aliases and concurrent raw-module mutation are outside this
diagnostic's proof.

The broad probe covers 1,426 regions and 5,656 memory references. An AST roundtrip
byte-matches the untouched module. A control keeps the same region/local
inventory but reloads the header before each memory-containing statement.
Disassembly confirms that this restores inspected reloads; it does not restore
identical machine code. Neither helper bindings nor ownership/address proofs
are generic: the full hash pins them. Existing full Datadog tests and focused
original-Wasm/count/bin/SQL oracles pass before timing. Those checks qualify
screening, not a reusable production transform.

| Function | Before bytes / frame | Broad view bytes / frame | Reload control bytes / frame |
| --- | ---: | ---: | ---: |
| SQL `fn96` | 111,763 / 856 | 109,518 / 872 | 111,891 / 856 |
| Sketch `fn176` | 5,866 / 144 | 5,834 / 144 | 5,797 / 144 |
| Handle `fn84` | 1,029 / 32 | 966 / 32 | 1,061 / 32 |
| Scalar export | 3,004 / 112 | 2,696 / 112 | 3,016 / 112 |

These frames exclude saved-frame/return bookkeeping. The broad math function
has four more instructions despite slightly fewer bytes. Code size, register
allocation, frame size, and placement all change; fewer header loads alone do
not establish the timing mechanism.

### Broad paired measurements

Use Go 1.26.8, CGO disabled, `GOAMD64=v1`, explicit `-pgo=off`, default GC/memory,
and unpinned CPUs 0–15 on the Ryzen 7 5800HS (8 physical / 16 logical cores).
Freeze every binary first. Seven interleaved 500 ms samples per leaf are followed
by seven one-second samples with role and leaf order reversed. No compilation,
lint, tests, fuzzing, profiling, or reviewer tools overlap timing. Preserve every
sample. Original/varied/negative/raw-guest leaves start with GOMAXPROCS=16;
new SQL, bin-corpus, and matched-batch leaves verify actual P1. The `-16` suffix
on those leaves reports initial processors, not their timed processor count.
Translation p-values are exploratory and unadjusted across the comparison matrix.

Rebuild the independent spike from its locked Rust 1.97 shim with std/panic_abort,
Binaryen 133 metaDCE/O3, wasm2go 0.4.16 unsafe, and its own hardener. Native Go and
wazero tests pass; only harness/corpora are shared. Its unversioned local source
archive and generated artifacts are hashed. ABI 2 borrowed input, background
context, and four instances differ from Cocoon's checked ABI 3. Original batch
and trace inputs also differ, so their reference times are controls, not facade
overhead comparisons. Matched batches use identical corpora and Count checks.

| Workload | Before, screen → confirm | Broad view, screen → confirm | Time change, screen → confirm |
| --- | ---: | ---: | ---: |
| Scalar | 99.14 → 91.24 ns | 100.8 → 90.36 ns | +1.67% → −0.96%, undetected |
| SQL | 2.415 → 2.263 µs | 2.424 → 2.227 µs | +0.37% → −1.59%, undetected |
| Batch 1k | 46.080 → 44.101 µs | 41.132 → 39.731 µs | −10.74% → −9.91%, p=.017/.001 |
| Trace 1k | 3.006 → 2.654 ms | 2.880 → 2.646 ms | −4.21% → −0.30%, undetected |
| Varied scalar | 97.01 → 92.68 ns | 95.34 → 91.56 ns | −1.72% → −1.21%, undetected |
| Varied SQL | 3.202 → 2.816 µs | 3.150 → 2.864 µs | −1.62% → +1.70%, undetected |
| Negative bins | 108.0 → 92.81 ns | 102.8 → 90.91 ns | −4.81% → −2.05%, undetected |
| New SQL holdout, P1 | 6.133 → 5.572 µs | 6.662 → 5.629 µs | +8.63% → +1.02%, p=.535/.435 |
| Repeated batch 1024, P1 | 47.942 → 44.970 µs | 43.999 → 40.341 µs | −8.22% → −10.29%, p=.002/.026 |
| Broad batch 8192, P1 | 351.116 → 326.724 µs | 351.522 → 329.074 µs | +0.12% → +0.72%, undetected |

Increasing and broad scalar bin corpora also show no detected change. Checked
scalar and batch workloads remain zero-allocation; SQL/trace allocations are
preserved in raw rows. The SQL holdout cycles six long/comment/escaped/Unicode/
quoted-name queries and compares complete outputs against original Wasm in two
untimed correctness passes. Its benchmark includes initial lazy guest setup,
not an untimed full-corpus warm pass.

Screening dispersion is substantial: original SQL's interval reaches ±66%.
The first SQL holdout slowdown does not repeat and is not statistically detected;
it establishes neither a material regression nor equivalence. The reload control
lacks the repeated-batch gain, but its confirmation regresses varied SQL by 4.44%
and trace by 3.75% (both p=.038), demonstrating code-shape sensitivity. Retain
these negatives rather than claiming all effect derives from header reuse.

Fresh confirmation reference scalar is 57.10 ns and SQL is 2.054 µs. The checked
baseline is 59.79% and 10.18% slower respectively. Baseline scalar 91.24 ns remains
above the original 55.44 ns limit; SQL 2.263 µs is above 2.2154 µs. The measured
candidate does not satisfy those limits either. No M1 acceptance follows.

### Narrow attribution experiment

The successful AddMany per-point loop retains the same point-load → `fn176` →
status-load sequence and 208-byte frame. Its ten rewritten regions mostly cover
one-time setup/finalization or errors. The two `fn176` terminal regions at baseline
lines 36198 and 36206 execute per point and remove four memory-header reloads.
The 144-byte frame and original guest accesses remain. Three further hash-pinned
controls select the nine sketch-function regions, ten batch-export regions, or
only those two publication regions. These function/line filters are diagnostic
controls, not acceptable production rules.

Repeat seven 500 ms samples and seven reversed one-second samples with the same
frozen before/broad/narrow binaries. This focused experiment covers original
scalar/batch, raw guest, and matched repeated/broad batches; it does not qualify
the narrowed variants across all workloads or native architectures.

| Workload | Before, confirm | Two regions, confirm | Time change, screen → confirm |
| --- | ---: | ---: | ---: |
| Batch 1k | 43.365 µs | 38.220 µs | −11.21% → −11.86%, p=.001/.001 |
| Repeated batch 1024, P1 | 44.792 µs | 39.060 µs | −11.55% → −12.80%, p=.001/.001 |
| Broad batch 8192, P1 | 321.449 µs | 321.337 µs | +0.44% → −0.03%, undetected |
| Checked scalar | 90.02 ns | 88.71 ns | −1.82% → −1.46%, p=.155/.002 |

The two-region confirmation gives about 13.46% more original-batch points/s and
14.68% more repeated-batch points/s. The whole sketch-function control reproduces
the batch gain too. The batch-export-only control instead regresses original
batch by 2.48% and repeated batch by 2.17% in confirmation (both p=.001).
This supports targeting the publication pattern. It does not eliminate placement/
register effects or establish a general per-byte or standalone-scalar benefit.
Narrow scalar improvement is detected only in its own confirmation, not its
screening round. Broad scalar comparisons have no detected gain; the narrowed
candidate has not run the full guard matrix. Absolute latency remains far above M1.

### Streaming controls, review, and next step

At actual P16 with 64-byte fixed writes, the unchanged single-home stream gives
1.242M writes/s in screening and 1.301M in confirmation; sixteen resource homes
give 6.181M and 7.453M. Held-out chunking gives 1.404M/1.391M on one home and
6.811M/7.933M on sixteen homes. These are same-binary synthetic workload controls,
not before/after gains from this Datadog-only probe. Variation from previous
rounds is preserved; it is not evidence of a streaming code regression.

Guest, host, and adversarial reviewers approve diagnostic timing and the retained
harness/evidence scope. Broad retention is deferred because it lacks generic
binding/effect/ownership proofs and executable WAT adversaries. The narrowed
publication pattern is a positive candidate for that implementation round;
see [the production requirements](translation-memory-views.md). No NEW public
contract is proposed or approved in this analysis.

The retained harness/evidence pass `make check` with pinned golangci-lint 2.13.1,
`make race`, `make rust`, `make cross`, `make smoke`, `make fuzz`, and
`make integration`. Two further builds reproduce all three proof packages,
locks, and runtime adapters unchanged. Exact staged-snapshot tests/strict lint
and darna precede the signed local commit. Production qualification of a future
hardener pass remains pending.

## Exact bin reuse: phase 0 no-go (2026-10-11)

Reject both measured default exact-value cache candidates. Retain only the
measurement harness and [raw evidence](measurements/bin-reuse-phase0.txt).
Neither capacity passes the preregistered requirement: the same capacity must
improve complete checked scalar calls by at least 5% on constant inputs **and**
a frozen mixed/phase corpus, versus untouched execution, in both rounds
(p < 0.05). Confirmed median regressions above 1% on miss-heavy, held-out or
original guards also reject a candidate. Smaller detected costs and uncertain
negatives remain reportable. This is a decision about these implementations
as defaults, not a claim that every exact cache or repeated batch is useless.
No production code, generated artifact, lock, public contract or manifest
option changes. No implementation spec is approved or needed for the retained
diagnostic work. M1 performance remains unaccepted.

### Frozen experiment and limits

Baseline is merged `main=origin/main=1510d69`; local work starts on
`task/exact-bin-reuse`. The private diagnostic driver checks the SHA-256 of the
entire pinned translated module, uniquely extracts its existing ln/affine/
floor/saturated-conversion block, and builds temporary Go overlays. It uses
neither host `math.Log` nor copied upstream/reference mapping code. The four
shapes are untouched, extracted/no-cache, extracted/cache1 and
extracted/cache64. These are not retained compiler specializations.

The cache stores only logical i32 indices, keyed by all five u64 words: value
bits plus current gamma, multiplier, minimum and offset bits. Parameter loads
remain inside execution, after unchanged validity/zero-threshold checks.
Every original counter/store/collapse update still executes in order. Tables
are private to a module; replacement creates empty storage. Direct mapping
hashes value bits with two xor-right-33 steps around multiplication by
`0xff51afd7ed558ccd`, then masks by capacity minus one; full-key equality and
an explicit valid flag decide hits. A valid collision replaces the entry.
On amd64 entries occupy 48 B: 48/3,072 B for capacities 1/64, plus a 24 B module
slice field and allocator overhead. Capacity 256 is 12,288 B of entry data.
This host memory is outside guest logical lengths; zero warmed B/op does not
count construction. All extracted variants add the slice field/helper code,
so untouched-to-extracted includes layout, allocation size and placement
effects; it does not isolate a function-call cost.

Go 1.26.8, CGO disabled, GOAMD64=v1, PGO off, default GC/memory, unpinned Ryzen
7 5800HS CPUs 0–15 (8 physical/16 logical). Seven 300 ms samples interleave
roles per leaf, reversing role order each sample. Separate seven 500 ms samples
reverse starting roles, families and leaves on the same frozen binaries.
Hashes remain identical after both rounds. No compilation, lint, fuzz or
reviewer tools overlap timing. New scalar/mapping/matched leaves set, assert
and report actual P1; original serial guards use initial P16. Printed `-16`
suffixes alone are not processor evidence. No samples are filtered. Dispersion
is substantial in several screening cells; p-values are exploratory and
unadjusted across the matrix. An undetected difference is not equivalence.

### Value and bin locality

The 8,192-point corpora are frozen before cache selection/timing. Timed arrays
repeat after a full pass, so uniqueness below is within each finite cycle.
The additional Increasing control and original `SketchAdd` instead continue
through nonrepeating integers; measured N stays below binary64's integer
precision limit. A complete untimed pass warms each new scalar store. Exact
logical bins come from fresh singleton sketches in the unchanged Wasm under
wazero, not collapsed store offsets or approximate host logarithms. Zero
count and logical index zero are distinguished separately by boundary tests.

| Corpus | Distinct values | Logical bins | Adjacent same-bin pairs / 8,191 | Second-pass hits: cap1 / cap64 |
| --- | ---: | ---: | ---: | ---: |
| Repeated (42) | 1 | 1 | 8,191 | 8,192 / 8,192 |
| UniqueCycle (1…8,192) | 8,192 | 377 | 7,815 | 0 / 0 |
| NearbyUniqueCycle (adjacent bits above 42) | 8,192 | 1 | 8,191 | 0 / 0 |
| MostlyUniqueCycle (one hot point per 16) | 7,681 | 3 | 7,167 | 0 / 413 |
| Phases (16 hot integers / unique cold phases) | 4,112 | 18 | 4,092 | 0 / 0 |
| BroadHoldout (unique mantissa/exponent permutation) | 8,192 | 5,373 | 0 | 0 / 0 |
| NegativeBins (existing eight-value cycle) | 8 | 8 | 0 | 0 / 8,192 |

Actual cache probes run outside timing, with first-pass and continuous second-
pass hits, misses, collisions/evictions per 1,024-point phase in the archive.
Capacities 1/16/64/256 are checked. Capacity 64's mostly-unique hit rate is
5.04%, below the nominal 6.25% hot frequency because collisions evict 42.
The 16 hot integers map to only four slots at capacity 64 and thrash, giving
zero hits in hot phases too. Keep this negative; the hash is not rerolled
against the observed corpora. The broad/unique/nearby cycles have zero replay
hits even at capacity 256. These are properties of the frozen policy, not
universal hit-rate bounds for another hash, associativity or capacity.

High adjacent-bin locality supports investigating certified intervals. It
does not certify membership: oracle samples, monotone bit search or approximate
exponential thresholds cannot substitute for a proof of the pinned mapping's
floating-point behavior. An interval design must fall back around boundaries
and uncertified values and retain ordered collapse. None is implemented here.

### Mapping controls and complete checked calls

Raw mapping controls include common indirect method invocation, bypass
ownership/locks/validation/current guest parameter loads, and are not additive
parts of facade latency. Original and hits use 42; OriginalIncreasing and
miss-dominated controls use identical increasing integers. The latter prime
42 once and can have at most one surviving priming hit. The increasing flag
is precomputed outside timing. Screening → confirmation medians (ns/op):

| Control | Screening | Confirmation |
| --- | ---: | ---: |
| Original mapping, 42 | 17.61 | 16.83 |
| Original mapping, increasing | 18.76 | 17.74 |
| Cache1 hit | 12.84 | 11.42 |
| Cache1 miss/fill | 40.32 | 39.30 |
| Cache64 hit | 13.92 | 12.66 |
| Cache64 miss/fill | 40.33 | 39.35 |

The attainable measured raw saving is only about 4–5 ns on 42, while a miss
adds roughly 21–22 ns. A weighted estimate using these matched input classes
needs about 80%/84% hits for capacities 1/64 just to break even. It is not a
full-call ceiling or prediction: instruction layout, direct versus indirect
calls, memory loads and stores interact. The increasing standalone workload
has no exact reuse regardless of hash quality.

Confirmation complete P1 scalar medians, ns/point; all cache miss-heavy losses
listed detect a difference (p ≤ 0.015). Constant cap1's modest gain is p=0.011;
screening detects no constant gain for either capacity.

| Corpus | Untouched | Extracted | Cache1 | Cache64 | Cache64 change: screening / confirmation |
| --- | ---: | ---: | ---: | ---: | --- |
| Increasing | 91.86 | 92.85 | 115.4 | 116.0 | +25.53% / +26.28% |
| Repeated | 91.21 | 92.45 | 88.53 | 91.36 | −3.35% / +0.16% |
| UniqueCycle | 92.39 | 94.02 | 110.8 | 114.7 | +20.02% / +24.15% |
| NearbyUniqueCycle | 91.69 | 94.93 | 112.3 | 115.3 | +27.66% / +25.75% |
| MostlyUniqueCycle | 90.88 | 92.35 | 111.7 | 114.9 | +24.97% / +26.43% |
| Phases | 92.54 | 92.60 | 110.4 | 114.5 | +27.44% / +23.73% |
| BroadHoldout | 91.43 | 94.26 | 116.9 | 120.4 | +29.54% / +31.69% |
| NegativeBins | 90.58 | 95.82 | 111.0 | 91.69 | +5.70% / +1.23% |

The cap64 NegativeBins comparison detects no gain or loss in either round.
Cap1 constant improvement is only 3.60%/2.94%, below the 5% criterion and
unconfirmed in screening. Neither mixed/phase candidate improves. Extraction
alone increases BroadHoldout 3.10% in confirmation (p=0.007), while most other
scalar extraction comparisons detect no difference. Keep the full-shape cost
separate from lookup/fill overhead.
Extraction's Increasing screening cost is 11.46% (p=0.038), without
confirmation (+1.08%, p=0.209). NegativeBins extraction rises 5.78% in
confirmation with benchstat's rounded p=0.050; retain that threshold ambiguity
rather than treating the printed value as proof of p < 0.05.

Original guards and matched batch qualification reject retention too:

| Workload | Confirmation untouched → cache64 | Cache64 change: screening / confirmation | Confirmation p |
| --- | --- | --- | ---: |
| Original increasing scalar | 91.22 → 109.9 ns | +16.42% / +20.48% | 0.001 |
| Original AddMany1k | 43.41 → 57.90 µs | +37.00% / +33.36% | 0.001 |
| Matched repeated N1,024 | 45.18 → 28.96 µs | −32.27% / −35.90% | 0.001 |
| Matched holdout N8,192 | 325.9 → 498.8 µs | +50.08% / +53.05% | 0.001 |
| SQL | 2.175 → 2.244 µs | +0.21% / +3.17% | 1.000 |
| Varied SQL | 2.785 → 2.841 µs | +1.75% / +2.01% | 0.620 |
| Varied scalar | 91.02 → 89.56 ns | +0.84% / −1.60% | 0.318 |
| Negative-bin scalar | 93.01 → 85.43 ns | −10.41% / −8.15% | 0.128 |
| Traces | 2.612 → 2.619 ms | −1.62% / +0.27% | 0.620 |

SQL/varied/trace differences and cap64's confirmation negative-bin gain are
not detected; their point estimates still remain. Cap1 original scalar and
AddMany1k regress 23.22%/34.41% in confirmation, both p=0.001. Its matched
repeated batch gains 35.80%/37.26%, while the matched holdout loses 48.20%/50.21%.
Extraction alone loses 4.94%/7.56% on matched holdout (p=0.002/0.017), and its
repeated-batch screening cost is 2.61% (p=0.038), without confirmation
(p=0.053). Strong repeated-batch gains do not qualify a default scalar cache.

The unchanged streaming binary is shared by all DD-only overlay shapes;
there is no before/after stream implementation difference to test. Fresh P16
64 B fixed/held-out default streams reach 1.275/1.417 million writes/s in
screening, 1.331/1.480 in confirmation. Sixteen-home standalone streams reach
7.555/7.933 then 8.877/10.009 million writes/s. These single-binary controls
show dispersion; they are not cache gains or a resolution of the older default
streaming negative. Existing checked Batch32 remains unchanged.

### Reference, correctness, gates and next work

Rebuild the independent local spike from its hash-recorded unversioned sources:
Rust 1.97.0 `--locked` std/panic_abort, Binaryen 133 meta-DCE/O3, wasm2go 0.4.16
unsafe and its own hardener. Its Wasm again reproduces
`8bda5ff7acff11fd64fd47a0a48165a71d610c28413665e4cc4d4034150d0651`.
Native Go/wazero tests and matched one-iteration Count checks pass. Only corpus
and benchmark code is shared, with no Cocoon implementation transplant or
external source edit. Raw labels/provenance/commands remain in the archive;
only comparison files normalize the reference package label. As in track 1,
the original batch/trace shapes and ABI2 borrowed/background/four-instance
contract differ; new P1 scalar corpora and matched groups use identical inputs.
Fresh original scalar is 57.48 ns reference versus 91.22 ns untouched Cocoon
(+58.70%); SQL is 1.982 versus 2.175 µs (+9.74%). Historical runs with different
harness/binary placement are not paired evidence of an intervening regression.
The real injector benchmark remains unavailable at its original machine path.

Before freeze all three reviewers approve the guest, host and adversarial
controls. Every overlay passes the entire DD test package, including point
validation/errors, late-invalid batch atomicity/recovery, boundary/zero tests
and growth/collapse. New tests check full value/parameter identity, collision
replacement, reset and fresh module isolation. Every corpus compares Count and
full protobuf after two complete ordered passes plus low-bin revisits, through
both standalone scalar and prevalidated batch calls, against original Wasm.
The 5,373-bin holdout necessarily crosses the 2,048-bin store capacity.

`make check`, `race`, `rust`, `cross`, `smoke`, `fuzz` and `integration` pass
using Go 1.27.1, official golangci-lint 2.13.1, Rust 1.97.0 and Binaryen 133.
These gates, including race/fuzz, use unchanged generated modules. The private
overlays receive the full DD test package, not separate race/fuzz qualification.
Cross checks compile rather than execute non-native targets. Race includes
generated consumer subprocesses; fuzz covers parsers, hardening and all
generated wazero adapters plus stateful streaming. Two further builds of each
of the three proofs and locks plus runtime adapter generation reproduce all
57 existing fixture/runtime files and the new harness unchanged. Exact staged
snapshot tests/vet/strict lint and Darna are required before the local signed
measurement commit. Gate/proof logs and source/binary hashes are archived.

Retain the phase-0 harness, decision and evidence in one local measurement
commit. Reject cache1/cache64 and extraction-only as production changes; no
rejected generated module is committed. Next choose whether to specify a
certified interval experiment or pursue translation analysis. Interval work
needs a credible floating-point proof and upstream API or checked compiler
specialization before implementation, original fallback, new holdouts and all
existing correctness/performance guards. Exact-value cache gains on repeated
batches alone do not solve standalone M1. No publication follows from this
round.

## Low-copy numeric inputs: qualified host encoding (2026-10-11)

Retain the private generator stage of track 1. Top-level numeric slices now
encode directly into their checked guest-memory reservation in stateless calls,
resource constructors/methods, and checked commands. The six integer/float types
share the existing checked cursor and bounded view APIs. Argument/aggregate size
checks, UTF-8/record validation order, admission, ownership, per-operation reset
and memory checks, reply validation, and immediate poisoning remain intact.
Rust still decodes an owned vector and validates the entire sketch batch before
mutation. Record codecs remain owned. No public API, ABI, manifest option, or
borrowed guest lifetime contract is added; the [spec](low-copy-batching.md)
precedes implementation in phase-0 commit `b04e04e`.

### Rejected placement and retained placement

[Round 1](measurements/low-copy-round1.txt) interleaved the new helpers with
existing codecs. Its batch gains repeated, but SQL rose 1.07% in confirmation
(2.143 to 2.166 µs, p=0.038), with the same direction in screening. The streaming
holdout P16/64 B rose 1.34% in confirmation (p=0.026). Fixed P16/4 KiB rose 0.54%
in screening (p=0.017), without a detected difference in confirmation. Reject
that candidate before committing it: batch gains and passing safety gates do
not waive the latency guards. All samples, source hashes, and negatives remain.

Round 2 appends the identical private helper definitions after operations and
resources, preserving existing codec/operation relative order. Go 1.26.8 amd64
inspection shows the four SQL function addresses restored to baseline; round 1
had moved them by 352 bytes. Equal bodies and changed placement motivated the
experiment but do not establish cache alignment as the cause. The new placement
qualifies on its own [paired/reversed evidence](measurements/low-copy-round2.txt).
All three reviewers approved the bounded implementation and the ordering delta
before retention. No padding, compiler flags, runtime shortcuts, or isolated
edits to generated code were used.

### Conditions and batch results

Baseline is `main=origin/main=47ae180` with the identical matched harness from
`b04e04e`. Frozen before/after and independently rebuilt spike binaries use Go
1.26.8, CGO disabled, GOAMD64=v1, PGO off, default GC/memory, and unpinned Ryzen
7 5800HS CPUs 0–15 (8 physical/16 logical). Seven 500 ms samples interleave
implementations per leaf, reversing role order on alternate samples. A separate
seven 1 s confirmation reverses the starting role, family and leaf order, without
rebuilding binaries. No compilation, lint, fuzz, or reviewer tools overlap timing.
Actual P1/P16 is set/asserted/reported in new leaves; the initial `-16` suffix
is not evidence of a P16 serial measurement. Original serial guards use initial
P16. P-values are exploratory, unadjusted across the matrix; no samples are
filtered. “No detected regression” does not prove timing equivalence.

The screening matrix covers all five corpora at 1/32/1,024/8,192 points, P1/P16,
one/sixteen homes, the seven original/varied/negative guards, and default/checked
streaming. Confirmation repeats the original guards, repeated/holdout matched
cells, P16 contention, and selected streaming guards including both 64 B holdouts
and fixed P16/4 KiB. Each group repeats after an untimed warmup; distinct values
are unique within a group, not an indefinitely new stream. The collapse oracle
crosses capacity during initial ingestion, not necessarily during timed groups.
Count and full protobuf agreement, invalid-tail atomicity, and recovery are
checked against the original Wasm before timing.

Selected P1 medians (µs/group; all listed batch comparisons p=0.001):

| Corpus / points | Before | After | Screening change | Confirmation change |
| --- | ---: | ---: | ---: | ---: |
| Repeated / 1 | 0.2181 | 0.1893 | −13.20% | −13.34% |
| Repeated / 32 | 1.737 | 1.673 | −3.68% | −4.02% |
| Repeated / 1,024 | 45.99 | 44.29 | −3.70% | −3.24% |
| Repeated / 8,192 | 361.7 | 351.8 | −2.71% | −2.73% |
| Holdout / 1 | 0.2181 | 0.1894 | −13.16% | −13.14% |
| Holdout / 32 | 1.740 | 1.681 | −3.39% | −3.56% |
| Holdout / 1,024 | 41.82 | 40.08 | −4.15% | −4.45% |
| Holdout / 8,192 | 331.6 | 319.6 | −3.61% | −4.75% |

The original 1,000-point batch improves 45.85 to 43.09 µs (−6.00%), then 46.27
to 43.28 µs (−6.46%) in confirmation. One-point `AddMany` is still much slower
than one scalar `Add` (~90 ns); these are batch throughput gains, not a scalar
latency claim. In every matched cell the temporary 8×N-byte Go buffer and its
allocation disappear (one allocation to zero). Rust allocation/copying remains.
The frozen candidate's `PutBytes` assembly checks source/destination equality
and bypasses `memmove` for the same guest view; this is a measured compiler/target
fact, not a portable performance guarantee. All cursor/range checks still run.

At P16 with one home, N32 reaches 18.017 million points/s and N1,024 22.289
million points/s; screening time drops 6.67%/4.78%, confirmed 6.20%/3.57%.
Sixteen homes reach 181.344/240.285 million points/s. The N32 sharded gain is
not detected in confirmation (p=0.053); N1,024 improves 7.61% then 10.28%, with
notable dispersion, so no fixed scaling guarantee follows. Logical guest memory
is unchanged: 1,179,648 B for one warmed home, 18,874,368 B for sixteen; serial
N8,192 uses 1,441,792 B, smaller groups 1,179,648 B. Measured post-warmup growth
is zero. These are logical lengths, not RSS or zero-guest-allocation proofs.

### Standalone and streaming guards

| Workload | Screening before → after | Screening p | Confirmation before → after | Confirmation p |
| --- | --- | ---: | --- | ---: |
| SQL | 2.148 → 2.132 µs | 1.000 | 2.154 → 2.156 µs | 0.620 |
| Scalar Add | 89.10 → 89.65 ns | 0.559 | 89.61 → 89.94 ns | 0.535 |
| Traces | 2.532 → 2.523 ms | 0.620 | 2.565 → 2.557 ms | 0.259 |
| Varied SQL | 2.725 → 2.720 µs | 0.648 | 2.737 → 2.732 µs | 0.438 |
| Varied Add | 89.75 → 89.79 ns | 0.878 | 90.51 → 90.15 ns | 0.902 |
| Negative-bin Add | 90.02 → 90.25 ns | 0.436 | 90.09 → 90.19 ns | 0.735 |

No standalone scalar, SQL, trace, or streaming guard detects a regression in
either retained-candidate run. Streaming has no numeric input and shows no
established benefit from this stage. The fixed P16/4 KiB confirmation still has
a +0.48% point estimate (p=0.053). Fixed P16 BatchHomesP has a +1.8% screening
estimate (p=0.053), not repeated in confirmation (p=0.902); other absent gains
and wide sharded streaming dispersion remain in the archive. This does not
resolve the older default-stream
negative against the pre-session implementation.

For synthetic 64 B streams at P16, the default one home reaches 1.369 million
writes/s (fixed) / 1.544 million (holdout); sixteen homes with standalone calls
reach 10.546/10.587 million, and Batch32 with sixteen homes 9.618/10.463 million.
Confirmation reaches 1.370/1.518 million default, 9.564/9.873 million standalone
sharded, and 8.795/9.979 million Batch sharded. Batch with one home remains
1.184/1.301 million in screening: sessions do not remove serialization. These
are warmed aggregate throughput, not per-request latency. Fixed sharded Batch
still allocates about 49.8 KB/801 allocations per stream versus 17.5 KB/250 for
standalone. The resource-home and Batch tradeoffs from track 3 remain.

### Independent reference, proofs, and next work

The original spike is a hash-recorded local source archive without `.git`.
Fresh Rust 1.97.0 `--locked` std/panic_abort Wasm builds, Binaryen 133 meta-DCE/O3,
wasm2go 0.4.16 unsafe, and its own hardener reproduce Wasm hash
`8bda5ff7acff11fd64fd47a0a48165a71d610c28413665e4cc4d4034150d0651`.
Actual commands, source hashes, native-Go/wazero oracles, and build metadata are
archived. Raw package labels stay intact; only comparison input labels are
normalized for benchstat. New matched P1 harnesses use identical ordered groups.
Original batch values (`i*1.5` versus `i`) and trace shapes (20×50 versus 1×1,000)
differ, and the original ABI2 borrowed decode/mutate and four-instance/background
context contracts differ from Cocoon's owned/prevalidated ABI3 calls. Compare
complete implementations; do not subtract their gap as safety overhead.

Fresh scalar is 55.51 ns reference versus 89.65 ns candidate (+61.50%), SQL
1.945 versus 2.132 µs (+9.61%). Matched reference repeated N1,024/8,192 is
36.28/290.5 µs versus 44.29/351.8 µs (+22.10%/+21.13%); holdout N8,192 is
297.4 versus 319.6 µs (+7.48%). The real injector reference at the original
`/home/bits/inject-bench` path is unavailable on this machine; no fresh
native/translated/wazero real-payload claim follows. **M1 performance remains
unaccepted**, and no default PGO profile is added.

Qualification uses Go 1.27.1, Rust 1.97.0 and strict lint 2.13.1. `make check`,
`make race`, `make rust`, `make cross`, `make smoke`, `make fuzz`, and
`make integration` pass. Integration rebuilds and checks all three
proofs and every shim's native tests/all-feature Clippy. `make fuzz` covers the
parsers/hardener and all three generated wazero adapters plus streaming's own
adapter. Cross checks compile all five targets; they are not native runs.
Compiled generated-consumer fixtures exercise the six types, mixed/unaligned
inputs, exact float bits, constructors/methods and checked commands. Independent
wire goldens and original-Wasm checks cover bounds, malformed tails, retained
owned output, current backing memory, undersized reservations before mutation,
reserve traps/overflow/logical bounds, memory faults, typed errors, and prefix
stop. Two repeated builds reproduce all 57 proof/runtime files and locks byte
for byte. Datadog and streaming Wasm remain byte-identical to baseline; compute
Wasm changes only to add the synthetic numeric proof operations. Retained
commits additionally validate their exact staged snapshots with darna.

Track 1's host stage is qualified. The subsequent exact-bin-reuse measurements
above reject both default cache prototypes; certified intervals or translation
analysis require a separate decision. Guest borrowed/lazy input decoding remains
a separate design and
lifetime/aliasing review requiring operator approval for any new public contract
before implementation. These throughput results do not accept scalar latency
or the remaining guest-decoding cost.

## Low-copy batching: measurements before implementation (2026-10-11)

After all three PRs merged, `main` and `origin/main` matched at `47ae180`.
Track 1 starts with matched inputs and a private checked prototype, before any
generator, runtime, or guest change. [Raw samples, driver, provenance, oracle
checks, and disassembly](measurements/low-copy-phase0.txt) retain every sample.

The serial matrix ingests identical ordered groups through public scalar calls,
public `AddMany`, and the prototype, at 1, 32, 1,024, and 8,192 points. Corpora
cover repeated values, nearby distinct values, integers distinct within each
group, verified nonzero negative bins, and an independent nonmonotonic broad
exponent/mantissa holdout. All groups repeat after one untimed warmup; this is
steady ingestion, not an indefinitely unique stream or a finite growth lifetime.
The Wasm-derived collapse assertion verifies the large holdout crosses the pinned
2,048-bin capacity. Complete protobuf and Count comparisons against ordered
original-Wasm scalar calls cover three repetitions and empty inputs. Prototype
proofs also cover invalid tails, exact errors, healthy recovery, input limits,
closed resources/libraries, and size-error precedence over a closed handle.

Seven isolated 500 ms samples use Go 1.26.8, CGO disabled, GOAMD64=v1, PGO off,
default GC/memory, and unpinned AMD Ryzen 7 5800HS CPUs 0–15. Each measured leaf
sets, asserts, and reports actual GOMAXPROCS, serial P1 or contention P1/P16.
The printed `-16` suffix reflects the initial setting, not a P1 measurement.
Scalar/facade/prototype group order reverses on alternating samples. No builds,
linting, fuzzing, or reviewer tool work overlapped timing. Go 1.26.8 tests and
one-iteration benchmark oracles passed beforehand, as did strict lint 2.13.1.

Selected medians (microseconds per group; no scalar-latency improvement claim):

| Corpus / points | Ordered scalars | Public AddMany | Checked prototype | Prototype vs AddMany |
| --- | ---: | ---: | ---: | --- |
| Repeated / 1 | 0.09297 | 0.2226 | 0.2243 | no detected change, p=0.831 |
| Repeated / 32 | 2.886 | 1.786 | 1.694 | −5.15%, p=0.001 |
| Repeated / 1,024 | 92.241 | 47.089 | 44.421 | −5.67%, p=0.001 |
| Repeated / 8,192 | 737.442 | 372.421 | 353.798 | −5.00%, p=0.001 |
| Nearby / 1,024 | 92.733 | 46.901 | 44.785 | −4.51%, p=0.026 |
| Distinct group / 8,192 | 742.070 | 369.963 | 356.068 | no detected change, p=0.073 |
| Negative bins / 1 | 0.09267 | 0.2251 | 0.2425 | no detected change, p=0.209 |
| Negative bins / 1,024 | 92.714 | 47.550 | 44.961 | −5.44%, p=0.026 |
| Holdout / 1,024 | 90.695 | 43.404 | 40.393 | −6.94%, p=0.001 |
| Holdout / 8,192 | 730.207 | 342.795 | 322.779 | −5.84%, p=0.001 |

Several prototype samples have 20–50% dispersion. Small batches and multiple
corpus/size cells show no detected gain; raw results include them all. These are
exploratory, unadjusted comparisons. Removing one temporary Go allocation saves
8×N B/call in every measured prototype cell; Go allocations become zero. Rust
still clones/decodes owned vectors, so guest execution remains allocating.

Independently timed controls (N=1/32/1,024/8,192) measure encode-only medians
20.93 ns / 80.66 ns / 1.796 µs / 11.55 µs; checked reserve/copy of preencoded
input 37.94 ns / 42.13 ns / 129.0 ns / 1.479 µs; checked direct preparation
51.83 ns / 76.94 ns / 987.9 ns / 7.304 µs; and raw prepared guest execution
94.22 ns / 1.510 µs / 43.42 µs / 347.4 µs. Their preparations, checks, locking,
and state differ: do not add or subtract them into an end-to-end cost model.
Preparation throughput is not ingestion throughput; raw guest calls bypass
supported ownership/execution checks.

The prototype preserves admission, owner resolution, execution locking,
per-operation state/memory checks, canonical reply validation, and poisoning.
It obtains a checked reservation cursor with `PutBytes(nil)`, borrows its current
bounded memory using `Range`, writes little-endian floats, and advances actual
reservation usage with `PutBytes` of that same view. The frozen Go 1.26.8 amd64
`PutBytes` assembly compares source/destination and skips `runtime.memmove` when
equal. This verifies the measured copy bypass, not a cross-toolchain performance
guarantee; the extra range/accounting work remains in the prototype. Inspection
used Go 1.24.1 objdump because the installed Go 1.26 package lacks that tool.

Contention uses one warmed resource per worker, with encoding outside the
instance lock in the facade and inside it in the prototype. At P16, one home
limits N32 to 16.255 vs 17.174 million points/s and N1,024 to 20.868 vs 21.816
million points/s (−5.38% and −4.34% time/op, p=0.001/0.002). Sixteen homes reach
139.462 vs 149.061 million points/s at N32 and 166.589 vs 184.086 at N1,024;
neither sharded timing difference is detected (p=0.073/0.165). Logical guest
memory is 1,179,648 B for one home and 18,874,368 B for sixteen, with measured
zero growth after warmup. Serial N8,192 uses 1,441,792 B; smaller groups use
1,179,648 B, also with zero warmed growth. These are logical lengths, not RSS
or proofs of zero guest allocation.

The measured ceiling justifies a small host-only generator experiment for all
six top-level numeric slice types using existing checked reservation APIs.
Keep Rust owned decoding, record codecs, standalone scalar/SQL bodies, and the
public contract unchanged. Retention still requires fresh paired full-facade,
scalar/SQL/trace/streaming holdouts and independent spike comparisons, fault and
growth proofs, all quality gates, and repeat builds. No production gain or M1
acceptance is established by this phase.

## Default streaming cost investigation (2026-10-11)

After PR #1 merged at `0a5969e`, the first follow-up investigates the repeated
default fixed P1/64 B slowdown. It retains diagnostic benchmarks and evidence;
there is **no production optimization or new public contract**. Default
standalone streams with ResourceInstances=1 bear the repeated fixed-stream cost;
home-sharding throughput gains require opt-in. **M1 performance is not accepted**.

The [raw archive](measurements/default-streaming-phases.txt) contains all samples,
drivers, commands, source/binary hashes, build metadata, and summaries. Conditions
match streaming qualification: Go 1.26.8, CGO disabled, PGO off, GOAMD64=v1,
Ryzen 5800HS, CPUs 0–15 without affinity, and default GC/memory settings. Each
leaf sets and asserts actual P1 despite the process's initial `-16` suffix.
Compilation, checks, fuzzing, profiles, and reviewer tools do not overlap timing.
The baseline remains Phase 0 `36a8e6f`, with production unchanged from `9623228`.
Both sides use the same standalone benchmark, Wasm, and translated guest.

### Frozen full-stream reproduction

Seven alternating 700 ms pairs screen fixed/holdout inputs at 64 B and 4 KiB;
seven reversed-order 1 s pairs confirm 64 B. After diagnostic measurements,
another seven 1 s pairs repeat the original frozen binaries.

| Workload | Frozen screen | Reversed confirmation | Frozen repeat after diagnostics |
|---|---:|---:|---:|
| Fixed 64 B | 157.2 → 160.4 µs, +2.01%, p=0.007 | 156.5 → 161.2 µs, +3.01%, p=0.001 | 157.7 → 161.6 µs, +2.50%, p=0.004 |
| Holdout 64 B | 197.0 → 196.3 µs, p=0.710 | 197.9 → 198.1 µs, p=0.902 | 197.9 → 197.3 µs, p=0.209 |
| Fixed 4 KiB | 5.137 → 5.139 ms, p=1.000 | — | — |
| Holdout 4 KiB | 5.468 → 5.423 ms, p=0.209 | — | — |

The fixed 64 B cost repeats; these sets detect no holdout or 4 KiB regression.
No detected difference is not equivalence. Allocation counts are unchanged.

### Bounded phase controls

`BenchmarkStreamPhases` uses identical source with both facades and the ordinary
checked APIs. NewClose measures a warm complete resource lifetime. Write controls
prime a full 1 KiB window, either before or after injection, then retain only
the trailing 32 owned replies. End and abandoned Close measure groups of 32
fresh resources. Preparation and verification are untimed; cleanup follows
timed End, while abandoned Close times disposal directly. End states cover
empty/short/full windows, prior injection, an entire match, and a partial match
at the tail. Independent whole-body oracles validate preparation and End; untimed
different replies overwrite guest storage to check retained-result ownership.
Live resources never exceed 32, independent of b.N. Close warm-up abandons
pending data without End.

Both binaries pass a 1x oracle run before seven alternating 250 ms pairs.
The following End/Close times divide the raw group ns/op by 32; NewClose and
Writes already measure one call/lifetime. These controls are **not additive**:
priming, output retention, live resources, timer transitions, and GC differ
from a complete stream.

| Control | Baseline | Merged | p-value |
|---|---:|---:|---:|
| NewClose | 551.5 ns | 564.0 ns | 0.128 |
| Write, uninjected, 64 B | 698.9 ns | 692.9 ns | 0.318 |
| Write, uninjected, 4 KiB | 23.60 µs | 23.35 µs | 0.259 |
| Write, uninjected, 16 KiB | 92.98 µs | 92.33 µs | 0.318 |
| Write, injected, 64 B | 637.3 ns | 624.2 ns | 0.012 |
| Write, injected, 4 KiB | 20.72 µs | 19.87 µs | 0.001 |
| Write, injected, 16 KiB | 79.84 µs | 78.18 µs | 0.004 |
| End, empty | 127.4 ns | 124.0 ns | 0.053 |
| End, short | 255.5 ns | 250.5 ns | 0.364 |
| End, uninjected full window | 5.922 µs | 5.881 µs | 0.805 |
| End, injected full window | 5.047 µs | 4.972 µs | 0.165 |
| End, whole match at tail | 239.4 ns | 232.8 ns | 0.053 |
| End, partial match at tail | 5.872 µs | 5.919 µs | 0.535 |
| Close, abandoned full window | 98.56 ns | 98.13 ns | 0.596 |

No phase detects a slowdown in this newly linked comparison. Some injected
Write controls favor the merged version; they do not establish a production
gain. Host Go allocations match: NewClose uses 200 bytes/8 allocations per
lifetime, each steady Write copies its chunk into one owned allocation,
nonempty End copies its tail into one allocation, and empty End/Close allocate
nothing. This does not count Rust/Wasm allocator activity.
Every control reports zero post-warm guest growth. Warm linear memory is
1,179,648 bytes except the 16 KiB Write controls, which use 1,245,184 bytes.
Write growth is sampled before untimed End. Grouped growth covers the
whole diagnostic, including untimed preparation, ownership probes and disposal;
it is neither an End-only measure nor a resident-memory bound.

### Adding benchmark source changes the comparison's sign

Seven reversed-order 700 ms full-stream pairs in the diagnostic binaries give
fixed 64 B 162.2 → 159.2 µs (−1.85%, p=0.038), fixed 4 KiB 5.215 → 5.114 ms
(−1.94%, p=0.017), holdout 64 B 203.1 → 198.5 µs (−2.24%, p=0.026), and holdout
4 KiB 5.438 → 5.401 ms (p=0.383). The holdout candidate has 21% dispersion.
This is one exploratory set, without independent confirmation of a gain.
P-values are unadjusted across exploratory comparisons.
Removing only the new benchmark source and rebuilding reproduces **both frozen
binaries byte for byte**. The post-diagnostic frozen repeat still detects the
fixed slowdown. Consequently, adding test source can change the measured sign
without a production-source change. Link placement and allocation/GC effects
are hypotheses, not an established cause or a fix.

Source review finds unchanged standalone Write/End/Close bodies, Resource and
Lifecycle paths, and original Instance Call/ResultView bodies and footprint.
Normalized frozen instructions agree for outer End, Close, execution/reply
validation, resource use, and guest writes; Write and End's callback differ
only in option-field offsets. Normalization masks unnamed data relocations;
it does not prove equal placement, referenced data, metadata, or performance.
Resource construction has the new home-selection machinery; selection is not
repeated on each write. These observations narrow the search but do not
attribute the complete-stream delta to one check or constructor cost.

Retain the diagnostic controls; retain no speculative production rewrite from
this round. The default fixed-stream negative remains unresolved and visible.
Future candidates need frozen end-to-end guardrails and holdouts even if their
phase controls improve. Resume low-copy batching with matched scalar/batch input
corpora; refresh the independently rebuilt spike before any M1 acceptance claim.

### Qualification

Guest, host, and adversarial reviewers checked the harness, raw evidence, and
conclusions before retention. `make check` (tests, vet, 49 strict linters with
official golangci-lint 2.13.1), `make race`, `make rust`, `make cross`, `make
smoke`, `make fuzz`, and `make integration` pass with Go 1.27.1/Rust 1.97.0/
Binaryen 133. Smoke initially failed because the sandbox denied Go module
verification network access; the authorized network retry passed. Two further
builds of all three proofs plus runtime generation reproduce all 54 existing
tracked proof/runtime files byte for byte, including their locks. A Go 1.27.1
race-instrumented 1x run exercises every new benchmark's oracle and ownership
checks; its timings are validation only, excluded from performance results.
The retained change adds only diagnostics and documentation, with no generated
artifact or lock delta. Darna and exact staged-snapshot tests gate its commit.

## Streaming qualification harness correction (2026-10-11)

The first end-to-end streaming run and the stopped private-layout candidate
used parent-only GOMAXPROCS setters. Go 1.26's `testing.processBench` resets
GOMAXPROCS to its CPU-list value before each measured leaf: these binaries
started with 16 processors, so P1/P4 labels did not describe actual timing.
All streaming qualification conclusions from these runs are excluded, without
relabeling lower-P samples. Heap diagnostics have no P-labelled mismatch but
are superseded too. The corrected harness sets/restores GOMAXPROCS inside each
measured leaf, checks it before and after RunParallel, and reports actual-procs.
The measurement runner must reject a missing or mismatched metric. Identical
corrected standalone source is used on both sides of the new comparison.

The [historical raw archive](measurements/checked-sessions-streaming-before-layout.txt)
preserves the original harness, hashes, samples, and stopped candidate (exit
130). Its independent DD scalar/SQL/batch/trace pairs remain valid: they never
used parent processor setters. The pre-layout negative-bin slowdown repeated
at 0.64% (p=0.036) and 0.97% (p=0.017), so that shape failed standalone
qualification. Phase 0 and R8 runtime measurements remain valid: they set
GOMAXPROCS externally per process, without P-labelled parent setters.

An earlier attempt also overlapped the final race check. It was stopped after
about 26 seconds and all its samples were discarded before restarting in
isolation. The same archive preserves that excluded attempt separately. No
overlapping sample contributes to an accepted measurement.

## Bounded resource streaming qualification (2026-10-11)

The operator-approved [contract](checked-sessions.md) is implemented: finite
opt-in typed command buffers share admission, ownership, and execution locking;
resources can use a separate bounded set of lazy homes. Each command retains
reset, memory checks, reply validation, owned publication, and immediate fault
poisoning. Handles remain attached to their exact home/epoch after poisoned-slot
replacement. Qualification supports **opt-in concurrent streaming**, with the
negative default-path and Batch results below. **M1 performance is not accepted**.
This is a synthetic exact-byte matching payload; performance of the real
Injector and its dd-trace embedding remains separate follow-up work.

### Conditions and comparisons

Measurements use Go 1.26.8, CGO disabled, PGO off, GOAMD64=v1, Ryzen 5800HS,
CPUs 0–15 without affinity, and default GC/memory settings. System Go 1.27.1
and official golangci-lint 2.13.1 qualify code. Builds, checks, fuzzing, and
reviewer tools stop before each measurement. Every stream row reports verified
actual-procs; Go's initial `-16` name suffix is not the leaf's processor count.
The corrected screen takes 155.4 seconds; the option/diagnostic sweep takes
497.2 seconds, with a separate reversed-order standalone confirmation.

The baseline is the Phase 0 snapshot (`36a8e6f`, production unchanged from
`9623228`). Its original CLI builds the same authored streaming shim with the
host-only checked_batches flag omitted. Baseline and candidate Wasm SHA-256:
`9141023ca801ef1cb0825ea155dd42b4d83bcee9ce6c9a6ad453b40badb49955`.
The corrected standalone benchmark is copied unchanged to the baseline:
`4e98e93d5658fae33558b9af70d208031da25bd8cf0d9c59af13e2234ee86b59`.
This compares complete facades around identical guest code.

Seven alternating 500 ms DD pairs cover scalar, SQL, batch, trace, varied
scalar/SQL, and negative bins; a second seven-pair set reverses the first order.
Two seven-pair 250 ms stream sets cover fixed and independent holdout corpora
at P=1/4/16, 64 B/4 KiB/16 KiB chunks. Seven option sets compare N32, then
N=16/64/256 at P16. Options run in fixed within-process order. Three diagnostic
rounds cover mixed stateless traffic (250 ms), competition (500 ms), and
initialized memory (3 iterations). Short windows, GC, frequency effects,
shared-machine conditions, fixed ordering, and repeated exploratory comparisons
limit inference; p-values are unadjusted. Absence of a detected difference is
not an equivalence proof.

A timed operation is a complete stream: New, all writes, retained owned outputs,
End, and Close. Open, lazy initialization, inputs/oracles, and full byte checks
occur outside timing. Fixed input has 256 equal chunks. Holdouts rotate four
corpora with no match, crossing/repeated matches, partial tails, binary bytes,
and uneven/empty partitions; actual write counts vary. Paths retain one finite
output group without history proportional to b.N. Input reaches 4 MiB at the
largest size. Batch includes command construction, plans, decoding/boxing,
and owned results, with MaxCalls=N and 16 MiB input/output budgets. Parallel
ns/op is aggregate stream throughput, not individual caller latency.

The [complete qualification log](measurements/checked-sessions-streaming.txt)
contains every sample, source/binary hashes, exact drivers/commands, independent
repeats, and summaries. The [historical archive](measurements/checked-sessions-streaming-before-layout.txt)
retains invalid processor-labelled sets, earlier scalar negatives, and excluded
procedural attempts. Those stream sets do not support qualification.

### Standalone guardrails and rejected shapes

The final R12 shape restores the original Call and ResultView bodies and
Instance footprint. One authored template generates locked/held-lock adapters
in the original Call region, with reproduction and statement-equivalence tests;
strict lint still covers the authored runtime file. Batch snapshots validated
wire length before status interpretation/typed decoding, keeping its accounting
out of standalone ResultView. The private Library restores the original
lifecycle offset and places new homes/cursor after it. Options offsets change;
these are implementation choices, not proof of cache placement or the cause of
earlier slowdowns.

| Workload | First baseline → R12 | Independent baseline → R12 |
|---|---:|---:|
| SQL | 2169 → 2190 ns, p=0.066 | 2169 → 2180 ns, p=0.090 |
| Scalar | 88.64 → 88.18 ns, p=0.002 | 88.77 → 88.70 ns, p=0.902 |
| Batch 1k | 44.78 → 44.59 µs, p=0.011 | 44.75 → 44.65 µs, p=0.165 |
| Trace 1k | 2.617 → 2.625 ms, p=0.902 | 2.623 → 2.607 ms, p=0.902 |
| Negative bins | 89.31 → 89.02 ns, p=0.128 | 89.57 → 89.16 ns, p=0.535 |
| Varied scalar | 89.28 → 89.36 ns, p=0.435 | 89.62 → 89.00 ns, p=0.128 |
| Varied SQL | 2809 → 2786 ns, p=0.015 | 2806 → 2795 ns, p=0.247 |

Neither set detects a scalar/SQL/batch regression. Allocation counts are
unchanged. SQL medians remain roughly 0.5–1% higher, and the confirmation
candidate includes a 21% SQL dispersion: a small cost is not excluded.
No standalone scalar improvement or blanket no-regressions claim is made.

Default P1/64 B fixed streams regress 155.7 → 160.4 µs (+3.00%, p=0.001),
confirmed at 155.7 → 160.5 µs (+3.05%, p=0.002), with unchanged allocations.
This negative remains part of the retained opt-in throughput trade-off.
P1/16 KiB holdout is +0.67% (p=0.004) in the first set, but not repeated
(21.00 → 20.95 ms, p=0.383). The option sweep's P1/64 B sharded-standalone
holdout is +2.20% (p=0.011); it has no independent confirmation. Other held-out
negative point estimates likewise are not independently confirmed. All point
estimates and samples are retained. Sharding does not promise a
single-stream/default latency improvement.

Additional uncommitted shapes were rejected:

| Shape | Reason | Evidence |
|---|---|---|
| Layout alone | P1/64 B holdout +1.56%, repeated +2.75%; fixed +2.23% in confirmation | [raw samples/patch](measurements/checked-sessions-streaming-layout-rejected.txt) |
| R9, batch-local wire accounting with helper core | Multiple standalone stream regressions of 1–3%; not qualified | [raw samples/patch](measurements/checked-sessions-r9-rejected.txt) |
| R10, cold state predicate and lifecycle prefix | Fixed P1/64 B +3.75%, repeated +4.72%; holdout +0.86% in second set | [raw samples/patch](measurements/checked-sessions-r10-rejected.txt) |
| R11, shared template in separate runtime file | Negative-bin scalar +2.74%; fixed P1/64 B +2.21% | [raw samples/patch](measurements/checked-sessions-r11-rejected.txt) |

R1–R7 and earlier R8 foundation/harness evidence remain below. Rejected designs
are not retained commits or cumulative speedups.

### Streaming ceilings

Medians in **million writes/s**, N32. Old/default and new/default use one home.
Sharded standalone and sharded Batch use P homes; the separate stateless pool
is configured to one and stays unused. P1's final Batch column reuses BatchHomes1.

| Corpus | P | Chunk | Old/default | New/default | Sharded standalone | Batch, one home | Batch, P homes |
|---|---:|---:|---:|---:|---:|---:|---:|
| Fixed | 1 | 64 B | 1.64420 | 1.59633 | 1.59950 | 1.34171 | 1.34171 |
| Fixed | 1 | 4096 B | 0.05054 | 0.05061 | 0.05075 | 0.04977 | 0.04977 |
| Fixed | 1 | 16384 B | 0.01295 | 0.01290 | 0.01278 | 0.01266 | 0.01266 |
| Fixed | 4 | 64 B | 1.44998 | 1.46013 | 4.94357 | 1.23452 | 4.33661 |
| Fixed | 4 | 4096 B | 0.04893 | 0.04916 | 0.19740 | 0.05012 | 0.19342 |
| Fixed | 4 | 16384 B | 0.01253 | 0.01250 | 0.04905 | 0.01268 | 0.04879 |
| Fixed | 16 | 64 B | 1.42854 | 1.44544 | 11.16153 | 1.22629 | 10.36181 |
| Fixed | 16 | 4096 B | 0.04723 | 0.04784 | 0.46550 | 0.04995 | 0.46699 |
| Fixed | 16 | 16384 B | 0.01249 | 0.01239 | 0.11478 | 0.01267 | 0.11729 |
| Holdout | 1 | 64 B | 1.77818 | 1.77923 | 1.73991 | 1.43140 | 1.43140 |
| Holdout | 1 | 4096 B | 0.06745 | 0.06787 | 0.06696 | 0.06572 | 0.06572 |
| Holdout | 1 | 16384 B | 0.01759 | 0.01747 | 0.01753 | 0.01713 | 0.01713 |
| Holdout | 4 | 64 B | 1.61222 | 1.60694 | 5.41892 | 1.38019 | 4.93383 |
| Holdout | 4 | 4096 B | 0.06573 | 0.06511 | 0.25852 | 0.06667 | 0.25834 |
| Holdout | 4 | 16384 B | 0.01713 | 0.01711 | 0.06492 | 0.01743 | 0.06327 |
| Holdout | 16 | 64 B | 1.58146 | 1.60226 | 11.81841 | 1.34872 | 11.60406 |
| Holdout | 16 | 4096 B | 0.06486 | 0.06466 | 0.59915 | 0.06696 | 0.59580 |
| Holdout | 16 | 16384 B | 0.01713 | 0.01707 | 0.15436 | 0.01729 | 0.15277 |

At P16/64 B, sharding raises 1.429 to 11.162 Mwrites/s (7.81×); sharded
Batch32 reaches 10.362 (7.25×). The independent holdout reaches 11.818
standalone and 11.604 batched versus 1.581 old/default. P16/4 KiB fixed
sharded standalone/Batch reach 0.466/0.467 versus 0.0472 old/default.
The gain is primarily placement: translated guest calls run concurrently.

Typed Batch alone loses roughly 15–20% on 64-byte cases. Fixed 64 B/N32
allocates 49.84 KB and 801 allocations per stream versus 17.50 KB and 250
standalone. Reusing three adapter closures per buffer avoids three per command,
but typed command/result boxing and plan creation remain real costs. Runtime
admission ceilings do not establish a small-write facade gain. Prefer sharded
standalone writes for this synthetic small-chunk workload.

P1 batching also loses on larger chunks. Some larger shared-home P4/16 cases
have only small point-estimate gains; P4 sharded Batch generally trails sharded
standalone, including the 16 KiB holdout. At P16 fixed 4/16 KiB Batch is slightly
higher, while holdouts are slightly lower. These are workload-specific choices,
not a general Batch recommendation or per-byte translation improvement.
The external inject-bench checkout is absent here; its supplied 1–2 µs/call
facade cost and ~11× translation overhead are sanity references, not reproduced
or directly comparable measurements. Phase 0 admission primitives in tens of
nanoseconds are a small share of those payload costs.

### Length, contention, and memory negatives

N16/64/256 does not establish monotonic sharded gains. Full length samples are
in the log; larger buffers cannot be chosen from the runtime ceiling alone.
Mixed 4 KiB holdout traffic performs one owned stateless Echo per output group
through a separately warmed P-slot pool. At P16/N32, sharded standalone reaches
0.608 Mwrites/s and 19,925 Echo/s; sharded Batch reaches 0.589 and 19,294,
versus one-home Batch 0.0681 and 2,231. These are coupled rates, not independent
stateless latency. Echo/write is 0.03276 at N32 and 0.005461 at N256; raw
rates must not hide the changed mix.

A dedicated 64-byte standalone probe competes with buffered streams on 4 KiB
holdouts. Timings include scheduling and the clock. Calls begun while workers
are active enter a trailing 1024-sample ring, including a final slow waiter.
Quantiles below are medians of three diagnostic runs, not whole-window
latency distributions or guarantees; maxima are medians of each run's
retained-sample maximum.

| P16 homes | N | Probe p95 | Median of retained maxima | Retained samples |
|---|---:|---:|---:|---:|
| 1 | 16 | 3.99 ms | 4.74 ms | 540 |
| 1 | 32 | 7.82 ms | 9.11 ms | 132 |
| 1 | 64 | 14.72 ms | 15.82 ms | 163 |
| 1 | 256 | 36.95 ms | 47.63 ms | 59 |
| 16 | 16 | 1.49 µs | 4557.50 µs | 1024 |
| 16 | 32 | 1.09 µs | 3.18 µs | 1024 |
| 16 | 64 | 0.75 µs | 4.80 µs | 1024 |
| 16 | 256 | 1.45 µs | 10.53 µs | 1024 |

Larger shared buffers materially increase competing-call latency. Sharded tails
are nonmonotonic, and a limited ring cannot prove fairness. P1 also has scheduler
stalls measured in milliseconds. N32 bounds work, not latency; responsiveness
requires smaller-buffer experiments and workload-specific measurement.

Initialized-memory medians after GC, outputs released, live wrappers kept alive.
The configured one-slot pool stays lazy/uninitialized. MB is decimal; heap
observations are not process RSS or hard bounds.

| Initialized homes | Live streams | Go heap delta | Logical guest memory |
|---|---:|---:|---:|
| 1 | 1 | 1.403 MB | 1.245 MB |
| 1 | 256 | 1.810 MB | 1.704 MB |
| 4 | 4 | 5.610 MB | 4.981 MB |
| 4 | 256 | 5.663 MB | 5.243 MB |
| 16 | 16 | 22.439 MB | 19.923 MB |
| 16 | 256 | 22.489 MB | 19.923 MB |

Sharding trades memory for concurrency. Bounded lazy homes are not one instance
per stream; uneven lifetimes can overload a lane. Per-instance MaxMemory,
temporarily retiring homes, inputs, typed metadata, and host decode overhead
remain separate from wire-output budgets.

### Proof and qualification

The synthetic fixture retains at most 1 KiB between writes, injects once before
the first exact `</head>`, flushes End, and exercises expected errors/traps.
Write working buffers and linear memory exceed that retained window. Native
whole-input oracles, partial-match splits, translated contract/typed tests, bulk-memory proofs, and
valid-handle wazero sequences/fuzzing cover the stateful case. Middle failures
prove Attempted and exact publication, including post-decode memory faults,
exhausted budgets, cancellation, abandonment, aliases, replacement, sibling
survival, and draining Close. Generated differential comparisons check status
as well as reply bytes.

Final gates and retained commits are recorded in the roadmap. Cross checks
compile arm64/386/Darwin/Windows/js targets; they do not claim native execution.
Race instruments actual compiled generated consumers. Both existing proofs and
the streaming fixture rebuild from authored sources/locks; translated modules
are never optimized alone. All requested gates pass on the final R12 tree;
the approved contract is retained under the narrow throughput guidance above.

## Checked resource bridge qualification (2026-10-11)

This section records the earlier runtime-foundation rounds, superseded by the
complete translated-stream qualification above. The [approved contract](checked-sessions.md)
followed Phase 0. Qualification compares frozen binaries against the original standalone API;
the throughput feature must preserve its latency. These exploratory runtime
ceilings do not accept M1 performance.

Two checked-core shapes were rejected before retaining implementation. R1
factored `Instance.Call` into an inline wrapper and a shared boundary taking an
optional-lock flag, and factored `ResultView` into a wrapper over a reply helper.
Seven alternating baseline/candidate pairs show scalar 88.69 → 90.25 ns
(+1.76%, p=0.006), SQL 2167 → 2212 ns (+2.08%, p=0.017), negative bins
89.29 → 91.02 ns (+1.94%, p=0.012), varied scalar 89.33 → 91.71 ns
(+2.66%, p=0.001), and varied SQL 2802 → 2832 ns (+1.07%, p=0.017).
Batch and trace have no detected difference. No allocation count changed.
The [complete R1 samples](measurements/checked-sessions-r1-rejected.txt) include
its bridge throughput measurements and frozen binary hashes.

R2 restored the original successful `ResultView` path, tracking only validated
non-OK wire bytes in reset call state. It still regressed scalar 88.70 →
90.26 ns (+1.76%, p=0.001), negative bins 88.81 → 90.41 ns (+1.80%, p=0.001),
and varied scalar 89.40 → 90.62 ns (+1.36%, p=0.001). SQL, batch, traces,
and varied SQL have no detected difference; allocation counts are unchanged.
The [complete R2 samples](measurements/checked-sessions-r2-rejected.txt) preserve
this negative result. R1/R2 are uncommitted candidate shapes against `bfc77f4`;
neither is an accepted implementation or speedup.

R3 removed the conditional locking branch by putting unconditional lock/unlock
in standalone Call and invoking the shared full boundary in another frame.
It regressed scalar 88.65 → 92.13 ns (+3.93%, p=0.001), negative bins
89.16 → 93.22 ns (+4.55%, p=0.017), and varied scalar 89.47 → 92.39 ns
(+3.26%, p=0.001), despite decreases in batch (-0.39%) and varied SQL (-1.35%).
SQL and trace have no detected difference. This shape is also rejected; the
[complete R3 samples](measurements/checked-sessions-r3-rejected.txt) retain all
seven alternating pairs and binary provenance.

R4 shared inlineable state/reset/memory/protocol helpers, with the recovery
method directly deferred by the two small ordering adapters. It regressed
scalar 88.72 → 90.74 ns (+2.28%, p=0.001), negative bins 88.96 → 91.44 ns
(+2.79%, p=0.001), and varied scalar 89.56 → 91.56 ns (+2.23%, p=0.001).
SQL (-0.92%), traces (-2.08%), and varied SQL (-2.39%) decreased; batch has no
detected difference. This shape is rejected too. The
[complete R4 samples](measurements/checked-sessions-r4-rejected.txt) retain the
negative scalar outcome alongside those decreases.

R5 restored the original direct recovery closure, sharing the cold fault path
instead of calling a deferred method on success. No difference is detected for
scalar (89.99 → 88.88 ns, p=0.274), SQL, batch, traces, or negative bins. Varied
scalar has a small negative result (89.18 → 89.49 ns, +0.35%, p=0.023), while
varied SQL decreases 1.14%. The [complete R5 samples](measurements/checked-sessions-r5.txt)
include the baseline scalar's wider dispersion. This round was refined further
before committing; it supports no standalone latency improvement claim.

R6 made every reset field explicit and skipped state-error construction when
open. Scalar 88.68 → 88.95 ns (+0.30%, p=0.041), negative bins 88.96 →
89.80 ns (+0.94%, p=0.002), and varied scalar 89.42 → 90.04 ns
(+0.69%, p=0.011) have small negative differences. SQL, traces, and varied SQL
decreased; batch has no detected difference. The [complete R6 samples](measurements/checked-sessions-r6.txt)
preserve these negatives; no standalone latency benefit is claimed.

R7 changed the state predicate so healthy calls fall through and restored the
original scalar-before-pointer reset order. Scalar (88.87 → 89.00 ns,
p=0.302) and varied scalar have no detected difference, but negative bins
89.29 → 90.62 ns (+1.49%, p=0.026) regress. SQL, batch, and varied SQL decrease;
traces have no detected difference. The [complete R7 samples](measurements/checked-sessions-r7-rejected.txt)
retain this rejected shape and its negative holdout.

### Retained runtime foundation (R8)

The original R8 commit preserves Call layout and reset. Batch-only validated error
wire bytes live privately on Instance and reset before each command, without
adding standalone successful-call stores. Shared inlineable helpers own state,
reset, memory, and protocol policy; the small locked/unlocked ordering adapters
recover directly in deferred closures and share the cold fault classifier.
Every command retains its memory check, reset, reply validation, and immediate
poisoning. This is a runtime bridge; typed facade commands, optional lazy homes,
and the translated streaming fixture were not yet implemented at that stage.
R12 replaces the helper/Instance-ledger implementation with the shared authored
template and batch-local accounting described above. R8 timings remain evidence
for that historical foundation, not the final typed streaming implementation.

Seven alternating baseline/R8 pairs detect no standalone scalar regression
(88.64 → 88.86 ns, p=0.330), SQL (2180 → 2176 ns, p=0.689), batch
(45.02 → 44.79 µs, p=0.383), traces (2.612 → 2.596 ms, p=0.053), negative
bins (88.85 → 89.07 ns, p=0.165), or varied scalar (89.21 → 89.23 ns,
p=0.929). Varied SQL decreases 0.99% (p=0.038), with a 22% baseline
dispersion. Allocation counts are unchanged. These are standalone guardrails,
not a scalar latency improvement claim or equivalence proof. All comparisons
are exploratory, unadjusted for multiple candidate rounds; final end-to-end
qualification needs independent paired samples.

Runtime medians in Mcalls/s, with plans reused and full bridge checks:

| GOMAXPROCS | Chunk | One home, standalone | One home, Batch32 | Worker-private, standalone | Worker-private, Batch32 |
|---|---:|---:|---:|---:|---:|
| 1 | 64 B | 7.217 | 7.971 | 7.167 | 7.952 |
| 4 | 64 B | 5.694 | 7.817 | 14.202 | 21.760 |
| 16 | 64 B | 4.858 | 7.404 | 5.704 | 34.813 |
| 1 | 4 KiB | 1.527 | 1.484 | 1.533 | 1.510 |
| 4 | 4 KiB | 0.960 | 0.957 | 1.391 | 1.259 |
| 16 | 4 KiB | 0.746 | 0.926 | 1.605 | 1.682 |
| 1 | 16 KiB | 0.426 | 0.399 | 0.427 | 0.395 |
| 4 | 16 KiB | 0.322 | 0.299 | 0.451 | 0.522 |
| 16 | 16 KiB | 0.271 | 0.324 | 0.452 | 0.561 |

At 16 processors, the shared-home 64-byte bridge gains 52% throughput, while
private instances with batching reach 7.17 times that standalone shared ceiling.
The private-instance path remains a diagnostic placement hypothesis, not the
bounded lazy-home implementation. At 4 KiB the respective rates are 0.746,
0.926, and 1.682 Mcalls/s, with far less admission-amortization headroom.
Negative results matter: P1 4/16 KiB batching and P4 shared 16 KiB batching are
slower than standalone; P4 private 4 KiB batching also loses throughput.
Retaining one buffer of results changes live allocation/GC behavior versus
the prototype. Generated typed plans and complete stream lifetimes must be
measured before recommending either option.

The [complete R8 sample log](measurements/checked-sessions-runtime.txt) records
binary/source hashes, every sample, exact commands, and limitations. It includes
zero-byte controls and the checked prototype comparators. Actual command
limits, contexts, byte budgets, owned staging, prefix accounting, and per-call
checks are included; typed command construction and New/End/Close are excluded.
Integer allocation metrics can show 0/call or 31/32-command buffer because the
first holdback replies are empty; these paths are not allocation-free.
No measurements overlap checks. R8 passed strict check/race and both proof
rebuilds retain the existing artifacts/locks. M1 performance remains unaccepted.

Both rounds use Go 1.26.8, CGO disabled, PGO off, GOAMD64=v1, the Phase 0
Ryzen machine without affinity, GOMAXPROCS=16, seven 500 ms samples per workload,
alternating pair order. All builds, checks, and reviewers completed before
timing. Raw files include every sample. Short windows, frequency variation,
GC, and shared-machine conditions still limit inference.

R1's actual runtime bridge, including context and byte budgets, reaches
7.37 Mcalls/s at GOMAXPROCS 16 for 64-byte writes on one home versus 5.25 for
standalone calls; worker-private instances reach 35.15. For 4 KiB these are
0.940, 0.775, and 1.804 Mcalls/s. However, the reusable plans exclude generated
typed-plan construction and complete stream lifetimes. The bridge also retains
one buffer of owned outputs per worker (up to 512 KiB per worker at 16 KiB
chunks), whereas the Phase 0 loop discards copies immediately. P1 4 KiB bridge
is slower than standalone (663 versus 651 ns), and P4 16 KiB shared bridge is
slower (3350 versus 3128 ns). Gains do not justify the standalone regressions;
these measurements explain the rejected candidate, not a retained feature.

## Bounded checked sessions: Phase 0 (2026-10-10)

Measurement precedes the proposed contract in [the review spec](checked-sessions.md).
This phase added only test/benchmark code; runtime, generators, standalone APIs,
Wasm, translated modules, and proof locks remained unchanged for its samples.
Phase 0 implemented or accepted no throughput feature. M1 performance
acceptance remains open.

### Conditions and provenance

The clean starting checkout was `main` at `9623228`, equal to the local
`origin/main` ref. This machine's checkout is `/home/dario/Code/cocoon`, not the
supplied `/home/bits/cocoon`. Measurements use the already installed
`/usr/lib64/go/1.26/bin/go` (Go 1.26.8), linux/amd64, `GOAMD64=v1`, CGO disabled,
and explicit `-pgo=off`. The system Go 1.27.1 is used for quality checks.
The CPU is an AMD Ryzen 7 5800HS with 8 physical/16 logical cores, frequency
boost enabled, CPUs 0–15 allowed, and no affinity. The complete measurement
runner took 288.0 seconds. Compilation, linting, tests, fuzzing, and reviewer
work had finished before the recorded samples; none ran concurrently.

Each of seven 250 ms throughput sets runs GOMAXPROCS 1, 4, and 16, rotating
processor-setting order between sets. Subbenchmark order within a process is
fixed. Primitive layers use five 300 ms samples at GOMAXPROCS 1. The seven
existing Datadog workloads use seven 500 ms samples at GOMAXPROCS 16, followed
by five 300 ms real-guest layer samples at GOMAXPROCS 1. No samples are removed.
These are exploratory medians, without significance or production-speedup
claims; short windows, fixed within-process ordering, GC, and shared-machine
frequency effects limit interpretation.

Build the frozen binaries before timing:

```sh
CGO_ENABLED=0 GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go \
  /usr/lib64/go/1.26/bin/go test -pgo=off -c \
  -o .cache/sessions-phase0/rt.test ./rt
CGO_ENABLED=0 GOTOOLCHAIN=local GOCACHE=$PWD/.cache/go \
  /usr/lib64/go/1.26/bin/go test -pgo=off -c \
  -o .cache/sessions-phase0/dd-baseline.test ./examples/datadog/go/dd
GOMAXPROCS=16 CGO_ENABLED=0 .cache/sessions-phase0/rt.test \
  -test.run '^$' -test.bench '^BenchmarkSessionThroughput$' \
  -test.benchtime=250ms -test.count=1 -test.benchmem
```

With trailing whitespace normalized, full commands, iteration counts, ns/sequence, ns/call, calls/s, input MiB/s,
bytes/sequence, and allocations/sequence are preserved in the committed
[complete raw log](measurements/checked-sessions-phase0.txt). Its SHA-256 is
`517833c82c0cab692a376e6f525aafd04c6dd0693e8ffc4c4dc32d963d32c774`. The ignored individual logs are under `.cache/sessions-phase0`;
the committed log is the durable copy. Binary SHA-256 hashes are:

- Runtime benchmark: `030f8cef86ad3d88ee1100f02763fe8091bf79ea823856dc58df06c4e80258b7`.
- Unchanged Datadog baseline: `aec965fbf86f67ceaf5a916a0df83dd306b4d27e580d8dd3d5b484f43bbb1c32`.
- Authored benchmark source: `9a79138a9e7048299bc223b12f940b0160b24e7df5836b42fd9dd87024e30996`.

The packaged Go toolchain lacks `go tool buildid`; binary hashes provide the
available provenance. Quality uses the official golangci-lint 2.13.1 binary
(revision `6d2288e0`, built with Go 1.27.0), verified against its published
archive checksum. The pre-existing cached linter reports `(devel)` and was
not used to qualify this addition.

### What is measured

`rt/session_bench_test.go` supplies a native-Go synthetic ABI with a 1 KiB
hold-back window per stream, fixed generation-one handles, preallocated memory,
and bounded input/output copying. Empty writes still reserve and encode their
zero-byte variable input. This is a host-runtime ceiling, not translated-Rust
or injector throughput. It omits guest input ownership allocations, real Rust
slab reuse, charset/partial-match parsing, construction, end, and disposal.
Nonempty replies are copied into owned Go bytes in every checked path.

Each parallel worker has its own resource ownership cell. `Home` is a
count-one synthetic checked resource path sharing one instance, including
library admission, ownership, execution lock, reset, host panic metadata,
input checks, owned reply validation, post-call memory checks, and poisoning.
`Admit32` shares one admission and ownership lock but uses 32 existing checked
`Instance.Call`s. `Locked32` also shares the execution lock; its test-only
checked boundary repeats every current per-operation check and stops on the
first error. `Locked4/16/64` are sequence-length controls at 64-byte inputs.
No prototype API is exported.

`Private` and `Private32` give each worker a fixed independent instance while
retaining shared library admission. They are hypothetical placement controls,
not implemented resource APIs. They use 64 KiB linear memory plus a 1 KiB
window per worker, versus one 64 KiB instance and one window per resource in
`Home`. `Pool1` and `PoolP` warm every lazy slot before timing and include both
admissions, context checks, a checked call, and a checked recycler. Synthetic
trim has no guest work. Pool capacity is one or GOMAXPROCS respectively; these
are stateless-envelope controls and cannot maintain a stream across arbitrary
pool borrows.

A benchmark operation is a whole sequence. `ns/call` and `calls/s` normalize
by its command count; parallel ns/op is aggregate throughput, not caller
latency. MiB/s counts input bytes, not output bytes; the initial 1 KiB holdback
is not emitted. B/op and allocs/op remain per sequence. Go reports allocs/op
as a truncated integer: the first holdback writes have no output, so a row
showing zero can still allocate on almost every steady-state nonempty write.
No allocation removal is claimed. Tests independently check mixed-size byte
order, exact/empty window boundaries, tail preservation, owned-output stability,
call-state reset, and actual sequence termination at command two for traps,
excess memory, malformed replies, and reserved status.

### Admission and serialization findings

Primitive medians (and all raw ns/op samples) are:

| Mechanism | Median ns | Raw samples in ns |
| --- | ---: | --- |
| Admission | 6.668 | 6.708, 6.669, 6.668, 6.664, 6.664 |
| Ownership | 11.23 | 11.45, 11.23, 11.2, 11.39, 11.15 |
| GuestHandle | 2.533 | 2.633, 2.533, 2.789, 2.533, 2.532 |
| Mutex | 5.391 | 5.398, 5.39, 5.391, 5.387, 5.391 |
| CheckedEmpty | 15.53 | 15.93, 15.53, 16.46, 15.52, 15.51 |
| ReplyUnit | 9.181 | 9.822, 9.208, 9.181, 9.18, 9.181 |
| Guest | 16.16 | 16.16, 16.18, 16.21, 16.15, 16.16 |

Admission is 6.668 ns without contention; host ownership is 11.23 ns, the bare
mutex is 5.391 ns, and canonical unit reply validation is 9.181 ns. Empty
checked calls include state/reset, host panic metadata, recovery, and memory
checks as well as locking (15.53 ns). Guest handle resolution and the synthetic
kernel remain per command. These independently compiled mechanisms are
not additive cost accounting. They do not justify skipping checks.

Real translated sketch layer medians are 53.54 ns guest, 72.64 ns checked
instance, 82.79 ns resource, and 89.31 ns facade. Guest work therefore remains
material even for a scalar sketch. Layer differences provide rough ceilings,
not exact component shares or a claim about standalone latency improvements.

| Input per write | GOMAXPROCS | Home Mcalls/s | Locked32 Mcalls/s | Private32 Mcalls/s | Home / Locked32 ns per call |
| --- | ---: | ---: | ---: | ---: | --- |
| 64 B | 1 | 7.355 | 9.220 | 9.225 | 136 / 108.5 |
| 64 B | 4 | 6.141 | 9.245 | 25.449 | 162.8 / 108.2 |
| 64 B | 16 | 5.215 | 8.986 | 38.832 | 191.7 / 111.3 |
| 4096 B | 1 | 1.583 | 1.668 | 1.671 | 631.6 / 599.4 |
| 4096 B | 4 | 0.990 | 1.119 | 1.416 | 1010 / 893.4 |
| 4096 B | 16 | 0.770 | 0.974 | 1.667 | 1298 / 1027 |
| 16384 B | 1 | 0.433 | 0.445 | 0.445 | 2310 / 2249 |
| 16384 B | 4 | 0.324 | 0.385 | 0.454 | 3086 / 2599 |
| 16384 B | 16 | 0.278 | 0.345 | 0.467 | 3591 / 2901 |

At GOMAXPROCS 16, the 64-byte shared-home ceiling is 5.22 Mcalls/s, rising to
8.99 Mcalls/s with 32 checked calls per sequence; independent worker instances
with the same sequence reach 38.83 Mcalls/s. Shared-lock sequencing improves
this synthetic path 1.72×, but leaves the shared-home ceiling intact. The
independent-instance sequence is 4.32× the shared-home sequence, with more
instance memory. This supports reviewing both amortization and placement.

For 4 KiB writes at GOMAXPROCS 16, the corresponding ceilings are 0.770,
0.974, and 1.667 Mcalls/s (about 3009, 3803, and 6511 input MiB/s). For 16 KiB,
they are 0.278, 0.345, and 0.467 Mcalls/s. Copy/allocation and serialized guest
work remain, so large-chunk gains are much smaller than the tiny-call ceiling.
These rates are not expectations for translated streaming payloads.

### Negative results and scope limits

- Independent instances alone do not solve tiny-call contention here:
  64-byte `Private` reaches only 5.45 Mcalls/s at GOMAXPROCS 16 versus Home's
  5.22. This is consistent with shared per-call admission contention, but also
  changes ownership/execution locking and loop overhead; the attribution is
  not isolated. At GOMAXPROCS 4 it reaches
  13.94 Mcalls/s. Scaling is not monotonic; more processors are not a guaranteed
  gain for cheap work.
- Sharing only admission/ownership is weaker than also sharing the execution
  lock: 64-byte `Admit32` is 139.3 ns/call at GOMAXPROCS 16 versus Locked32's
  111.3. For 4 KiB, Admit32 is 1212 versus Home's 1298 and Locked32's 1027.
- On one processor, extending shared execution from Admit32 to Locked32 adds
  little for large chunks: 4 KiB is 598.7 versus 599.4 ns/call, and 16 KiB is
  2262 versus 2249. These medians do not establish an isolated benefit.
- At GOMAXPROCS 16, 16 KiB Private32 is 2143 ns/call versus Private's 2121;
  sequencing is slightly slower by the medians. No significance claim is made.
- Longer sequences are not automatically better. At GOMAXPROCS 16, 64-byte
  Locked4/16/32/64 medians are 122.9/112.6/111.3/112.0 ns/call; N=64 gives no
  further measured benefit. Competing-stream tail latency and fairness were
  not measured and must be qualified after approval.
- A lock held while waiting for host network data would invalidate this
  CPU-only ceiling and extend Close indefinitely. Success throughput and the
  prototype fault checks do not establish a production session lifecycle.

The supplied cross-ecosystem reference `/home/bits/inject-bench` is absent on
this machine; an equivalent local directory was not found. Its reported
~1–2 µs/call facade cost and ~11× per-byte translation overhead are operator
context, not independently reproduced measurements. Uncontended admission
here is only about 0.3–0.7 percent of that reported fixed cost. Removing tens
of nanoseconds of host work cannot address an 11× guest translation cost;
contention amortization and resource placement are the relevant hypotheses.
The kernels, wrappers, inputs, and machines are not controlled equivalents.

### Fresh real-workload baseline

All seven existing workloads remain unchanged. These are fresh baseline
samples for a future paired Phase 2 run, not a before/after acceptance result.
Differences from historical sessions are not source improvements/regressions.

| Workload | Median ns/op | Raw samples in ns/op |
| --- | ---: | --- |
| ObfuscateSQL | 2156 | 2186, 2156, 2146, 2173, 2145, 2152, 2157 |
| SketchAdd | 88.36 | 88.69, 88.14, 88.13, 88.12, 89.35, 88.36, 88.47 |
| SketchAddMany1k | 44612 | 44612, 44865, 44547, 44746, 44573, 44343, 44799 |
| ObfuscateTraces1k | 2598198 | 2608015, 2597670, 2571825, 2601919, 2564812, 2634479, 2598198 |
| SketchAddNegativeBins | 88.91 | 88.95, 90.91, 88.91, 88.37, 88.57, 88.46, 88.91 |
| SketchAddVaried | 88.96 | 88.79, 90.06, 89.65, 88.89, 88.96, 88.87, 90.59 |
| ObfuscateSQLVaried | 2779 | 2786, 2770, 2782, 2771, 2773, 2785, 2779 |

SQL remains below its original 2215.4 ns limit in this run; scalar SketchAdd
at 88.36 ns remains above 55.44 ns. No new same-machine reference is available,
and M1 performance is not accepted. At this stage Phase 2 needed matched before/after scalar,
SQL, batch, trace, varied, negative-bin, streaming, and independent holdout
samples, including fairness and live-instance memory under contention.

### Review and qualification status

Guest-side, host-side, and adversarial reviewers inspected the harness before
measurement and retention. Review fixes add current Host/reset work, empty
input reservation, outer pool context checks, correct topology setup, byte
oracles, and middle-of-sequence faults. The single-call-around-a-loop shortcut
was rejected because it postpones memory checks/reset/poisoning. Synthetic
ceilings are retained as diagnostics; no production optimization is retained.

The Phase 0 addition passes `make check` with official golangci-lint 2.13.1,
`make race`, Go 1.26.8 runtime tests, darna, and exact staged-snapshot tests.
The operator explicitly approved the specification on 2026-10-10. Phase 2 was
pending at this point; its final translated-stream evidence and quality gates
are recorded above and in the roadmap.

## Current results

The fifth performance pass on 2026-10-06 investigated exact guest math lowering
and tested a guarded floor/saturated-conversion fast path with separate guest,
host, and adversarial review agents. Neither the unrestricted screen nor the
longer pinned confirmation established a timing improvement. The candidate
was removed; production code, both Wasm proofs, translated modules, facades,
and locks are unchanged. Performance acceptance remains open.

Two independent test commits remain: exact sketch-bin transitions and a verified
negative-bin benchmark (`1672671`), and executable numeric lowering contracts
included in `make smoke` (`b03d81b`). These are correctness/measurement additions,
not a claimed speedup. No profile or compiler-default change was retained.

### Exact guest math and conversion trial

The pinned libdatadog mapping at `libdd-ddsketch/src/lib.rs:295` computes a bin
from `floor(value.ln() * multiplier + index_offset)`. Its Wasm logarithm comes
from Rust 1.97's compiler-builtins/libm implementation, not Go's `math.Log`.
The translated hot path is `fn176`; the baseline compound floor/conversion is
at `examples/datadog/go/dd/internal/wasm/module.go:35812`.

Assembly inspection found that the dense log polynomial already stays in XMM
registers without floating-point spills or helper calls. The explicit float64
conversions do not add standalone instructions there; they preserve rounding
boundaries and prevent multiply/add fusion or reassociation. `math.Floor` is
already a guarded SSE4.1 `ROUNDSD`, and saturated conversion is inlined. The
58.29-percent guest, 43.58-percent math, and 10.16-percent result-validation
profile categories overlap. The roughly 14-percent flat polynomial and
2.14-percent flat conversion samples are attribution, not available speedups.
No exact, beneficial polynomial rewrite was identified.

The trial rewrote only a direct, correctly bound
`i32_trunc_sat_f64_s(math.Floor(expr))` into a helper with this guarded body:

```go
if f >= 0 && f < 0x1p31 {
    return int32(f)
}
return i32_trunc_sat_f64_s(math.Floor(f))
```

In that representable interval, truncation toward zero equals floor; all
negative, NaN, infinite, and out-of-range values use the original fallback.
The operand AST, including its explicit float conversions, was preserved and
evaluated once. See the [Go numeric-conversion rules](https://go.dev/ref/spec#Conversions_between_numeric_types),
[rounding rules](https://go.dev/ref/spec#Floating-point_operators), and
[Wasm saturated-conversion definition](https://webassembly.github.io/spec/core/exec/numerics.html#op-trunc-sat).
This is an integer-result equivalence argument, not a floating-point-environment
or NaN-payload equivalence claim.

The experimental AST pass required the exact unique helper signature/body,
real unshadowed math import, bound callee, one argument, and no ellipsis or
reserved-name collision. Review found that text alone also permits shadowed
built-in numeric types: `type int32 = float64` changes the fast path's result.
The trial therefore rejected bound non-predeclared numeric types, with a
regression test. Conflicting explicit math aliases were also rejected. The
fallback was appended after original-site rewriting to prevent recursion.
Datadog had one match; compute had none. These guards were tested but are not
shipped because the entire optimization was withdrawn.

Values below one can still populate positive-index bins. All existing varied
values do; they do not qualify the negative-index fallback. The new eight-value
corpus spans `1e-12` through `2e-12`, and the original Wasm independently proves
that each value populates a negative bin.

### Fifth-pass measurements

All binaries use Go 1.26.8, CGO disabled, `GOAMD64=v1`, and explicit `-pgo=off`
on the AMD Ryzen 7 5800HS. Seven rotated before/after/reference triples run
without compilation, lint, tests, fuzzing, or agent work. No samples are removed.
The independent reference is unchanged; only aligned SQL/scalar inputs support
overhead comparisons. Reference batch/trace inputs and pool size differ, and
the new negative-bin workload has no matched reference benchmark.

The first screen uses 16 Go processors, no affinity, and 500 ms per benchmark:

| Workload | Original | Candidate | Result |
| --- | --- | --- | --- |
| SQL | 1.905 µs | 1.913 µs | No detected change, p=0.927 |
| SketchAdd | 76.23 ns | 75.62 ns | No detected change, p=0.512 |
| SketchAddMany1k | 37.09 µs | 36.82 µs | No detected change, p=0.710 |
| ObfuscateTraces1k | 2.285 ms | 2.332 ms | No detected change, p=0.209 |
| SketchAddVaried | 76.58 ns | 75.27 ns | No detected change, p=0.209 |
| ObfuscateSQLVaried | 2.450 µs | 2.430 µs | No detected change, p=0.383 |
| SketchAddNegativeBins | 76.09 ns | 79.47 ns | No detected change, p=0.097 |

The longer confirmation uses one Go processor, CPU 14, and one-second samples:

| Workload | Original | Candidate | Result |
| --- | --- | --- | --- |
| SQL | 1.821 µs | 1.820 µs | No detected change, p=0.710 |
| SketchAdd | 74.37 ns | 73.28 ns | No detected change, p=0.165 |
| SketchAddMany1k | 35.22 µs | 35.89 µs | No detected change, p=0.097 |
| ObfuscateTraces1k | 2.157 ms | 2.196 ms | No detected change, p=0.209 |
| SketchAddVaried | 74.74 ns | 74.08 ns | No detected change, p=0.644 |
| ObfuscateSQLVaried | 2.339 µs | 2.352 µs | No detected change, p=0.833 |
| SketchAddNegativeBins | 73.97 ns | 75.69 ns | No detected change, p=0.165 |

The apparent scalar shifts are −0.80 and −1.47 percent; negative-bin shifts
are +4.44 and +2.33 percent. None establishes a gain or regression. Variability
remains substantial even with affinity, including 24-percent scalar and
75-percent trace confidence-interval bounds in the confirmation. The rejection
is for insufficient evidence, not proof of equivalence or impossibility.
All Go allocation counts remain unchanged; trace bytes vary slightly with
amortized setup and collection. P-values are exploratory and unadjusted.

Original/reference SQL and scalar medians are 1.905/1.669 µs and 76.23/48.40 ns
in the screen: 14.1 and 57.5 percent overhead. Pinned medians are
1.821/1.621 µs and 74.37/46.05 ns: 12.3 and 61.5 percent overhead. Neither
fresh same-run 10-percent target is met. SQL remains below its original
2.2154 µs limit; scalar remains above 55.44 ns. Differences from the preceding
pass are session variation, not a retained source improvement.

### Retained contracts and qualification

Nine adjacent binary64 boundaries were located by bit-pattern search against
the unchanged Wasm interpreter, not a host logarithm. They span negative bins,
the zero/positive-bin transitions, values around one, very small and large
exponents, and the zero-counter threshold. Fresh singletons prevent the
2048-bin collapse policy from hiding a wrong bin. Scalar and batch paths check
exact Count bits and complete protobuf bytes; triples and zero/tiny/negative-bin
mixes are also compared. The authored protobuf reader includes leading zero
counts when locating the first populated bin, with its own regression.

The independent WAT fixture runs pinned assembly, optimization, actual
wasm2go translation, and actual hardening before compiling the resulting Go.
Both Go and wazero check explicit integer/bit goldens: integer-limit neighbors,
signed zero, subnormals, finite extremes, infinities, signed signaling/quiet
NaNs, separate f32/f64 multiply/add rounding, addition order, once-only side
effects, bounds traps, and post-trap continuation. A deliberately broken helper
must fail with a semantic mismatch, not merely a compilation error. The fixture
also passes against the original hardener, independently of the candidate.

Candidate qualification passed strict check, both real proof builds and repeat
builds, Rust checks, race tests, five differential/structural fuzz targets,
external/relocation/Go-only smoke, and all five cross-compilation targets.
Numeric contracts ran natively on linux/386 and amd64-v3 as well as amd64-v1.
Arm64 was cross-compiled, not natively executed in this session. The smoke
fixture's CGO-disabled child is not race-instrumented by a parent race run.
Candidate hardening coverage was 94.6 percent, with the new pass at 95.5 percent;
restored hardening remains 94.0 percent. No coverage gain is claimed from code
that was removed. Restored production passes the retained contracts and proof
rebuilds, and its benchmark binary is byte-identical to the frozen baseline.
Each retained commit passes darna and exact staged-snapshot tests.

### Fifth-pass provenance

The before/after binaries include identical authored Datadog tests from
`1672671`. Their SHA-256 hashes are
`4ac489cc3c8e993570706aa4259cd64ed7e1da27f47a2dc66d901e47fa3a0ffa` and
`81bb7a7248e8758c540429ecb6e3d95bc970c0591af30d2a7fb8bad30aa99237`.
Their build IDs are `22fd5ce8f45a38453adc4f12c11106e2ea46992e` and
`9b7b88349f8771e52228a4fbc3420faf06e7f9fb`. The independent reference binary
hash is `bf3f15d3241684b35b0b0b448799373e8d66951f955827cce38fd7e365237dd8`.
Among generated proof artifacts, the candidate changed only translated Go
and its lock digest, to
`0e16305e4792902b3b6bd307bcd47ddc3e0783b4c8db918978233dfab8afb631`;
the retained Go hash is again
`f50d20e7fe091569353ac1ced7865a53cd23bc9add668c3aac0f5f3a0ac8eed4`.
Datadog Wasm remains
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`.
Compute and all other proof artifacts/lock fields are unchanged.

Complete raw logs and summaries are in ignored `.cache` under
`round12-floor-screen` and `round12-floor-pinned`; the rejected source patch
is saved as `.cache/round12-floor-candidate.patch`. It is experimental evidence,
not maintained production code. The retained commits are local and unpushed.

## Fourth performance pass

The fourth performance pass on 2026-10-06 tested profile-guided compilation
(PGO) and a fused checked resource-call path, with separate host, guest, and
adversarial review agents. No production source change or default profile
was retained.
The useful additions are held-out varied workloads (`2d95490`), ownership and
nested execution regressions (`f2d0158`), and a compiled generated-facade shutdown
regression (`85b55d0`). Runtime coverage increased from 97.0 to 97.8 percent.
The Wasm, translated modules, generated facades, and proof locks are unchanged.
The longer unrestricted comparison confirms a 5.36-percent scalar PGO gain,
including a 4.98-percent held-out gain, but batches regress 1.26 percent. PGO
remains an opt-in trade-off, and performance acceptance remains open.

### Profile guided compilation

Both Cocoon and the independent reference trained their own CPU profile, with
PGO explicitly disabled. Only the original SQL, increasing-value scalar,
1,000-value batch, and 1,000-trace benchmarks were selected. Each class ran for
two seconds with 16 Go processors and no affinity. This is equal nominal time
per class, not a production request mix. Cocoon's new varied-value and varied-SQL
benchmarks were held out from training. The reference retains the earlier
independently built/hardened overlay; its batch and trace inputs and pool size
differ from Cocoon's, so reference overhead comparisons use aligned SQL and
scalar workloads only.

The AMD Ryzen 7 5800HS comparisons use Go 1.26.8, CGO disabled, and explicit
`-pgo=off` or an absolute path to that implementation's own profile. Seven
sample sets rotate the four variants: Cocoon off/on and reference off/on.
Compilation, tests, fuzzing, lint, and agent work stop before timing. Profiles
are captured separately from unprofiled measurements; outliers are not removed.
All p-values are exploratory and unadjusted for repeated comparisons.

The initial unrestricted 500 ms screen did not establish a scalar improvement
(75.99 to 74.41 ns, p=0.383) or a varied-scalar improvement
(76.46 to 74.36 ns, p=0.318). Varied SQL improved 4.56 percent (p=0.038);
the other original workloads had no detected timing change. A longer pinned
confirmation used one Go processor, CPU 14, and one-second samples:

| Workload | PGO off | PGO on | Pinned result |
| --- | --- | --- | --- |
| SQL | 1.833 µs | 1.733 µs | −5.46%, p=0.010 |
| SketchAdd | 74.34 ns | 70.14 ns | −5.65%, p=0.026 |
| SketchAddMany1k | 35.27 µs | 35.29 µs | No detected change, p=0.902 |
| ObfuscateTraces1k | 2.164 ms | 2.083 ms | No detected change, p=0.097 |
| SketchAddVaried | 73.53 ns | 69.99 ns | −4.81%, p=0.001 |
| ObfuscateSQLVaried | 2.320 µs | 2.245 µs | −3.23%, p=0.011 |

The matched PGO reference scalar median is 45.36 ns, leaving 54.6 percent
overhead in this pinned run. Its SQL improves from 1.587 to 1.517 µs
(−4.41%, p=0.002); scalar, batch, and traces have no detected reference gain.
PGO does not close either the original 55.44 ns scalar limit or the pinned
same-machine 49.896 ns limit. Allocation counts are unchanged for all six
Cocoon workloads. Amortized construction bytes are not per-call allocations.

The final unrestricted confirmation returned to 16 Go processors and no
affinity, with seven rotated one-second sample sets and the same frozen
profiles and binaries. It establishes a synthetic scalar benefit beyond the
pinned run, but also exposes a batch regression:

| Workload | PGO off | PGO on | Unrestricted result |
| --- | --- | --- | --- |
| SQL | 1.843 µs | 1.765 µs | −4.23%, p=0.001 |
| SketchAdd | 77.37 ns | 73.22 ns | −5.36%, p=0.004 |
| SketchAddMany1k | 35.945 µs | 36.399 µs | +1.26%, p=0.004 |
| ObfuscateTraces1k | 2.164 ms | 2.104 ms | −2.78%, p=0.001 |
| SketchAddVaried | 77.15 ns | 73.31 ns | −4.98%, p=0.003 |
| ObfuscateSQLVaried | 2.376 µs | 2.253 µs | −5.18%, p=0.001 |

Percentages use unrounded medians. The reference improves SQL from 1.655 to
1.572 µs (−5.02%, p=0.001); its scalar is 47.52 versus 47.23 ns (p=0.710),
with no detected batch or trace change. No-PGO-to-no-PGO overhead is 11.4
percent for SQL and 62.8 percent for scalar. PGO-to-own-PGO overhead is 12.3
and 55.0 percent. Both PGO same-run 10-percent limits remain unmet:
1.7292 µs for SQL and 51.953 ns for scalar. SQL is below the original absolute
2.2154 µs limit; scalar still exceeds 55.44 ns. Allocation counts remain
unchanged. The slower batch is a measured trade-off, not an accepted universal
optimization. The earlier screen's inconclusive scalar result remains recorded.

No benchmark profile is installed as `default.pgo`. Synthetic gains do not
establish a production speedup, and profiles belong to the consuming application.
The [profile recipe](#benchmarking-and-profiles) preserves explicit controls and
excludes the held-out workloads. Training weights were not tuned against these
held-out results; further tuning needs independent traffic and a new holdout.

### Rejected fused resource calls

The candidate replaced generated `Resource.Use` → `Instance.Call` adapters
with one callback and a shared checked execution core. Both locks, ownership
and epoch checks, full call reset, panic containment, memory limits, reply
validation, and lifecycle admission/drain remained intact. It added
`Resource.Call` and callback-scoped `Call.Handle` only for this experiment;
neither API remains in the branch.

Seven rotated unrestricted 500 ms triples found no established gain in any
of the six workloads. Scalar medians were 74.81 versus 72.07 ns (p=0.259),
and varied scalar was 78.03 versus 74.71 ns (p=0.535). The longer CPU 14,
one-processor, one-second confirmation also did not improve scalar calls:

| Workload | Original path | Fused candidate | Pinned result |
| --- | --- | --- | --- |
| SQL | 1.851 µs | 1.808 µs | −2.32%, p=0.043 |
| SketchAdd | 73.06 ns | 73.43 ns | No detected change, p=0.318 |
| SketchAddMany1k | 35.04 µs | 35.17 µs | No detected change, p=0.805 |
| ObfuscateTraces1k | 2.140 ms | 2.147 ms | No detected change, p=0.535 |
| SketchAddVaried | 74.34 ns | 73.14 ns | No detected change, p=0.318 |
| ObfuscateSQLVaried | 2.329 µs | 2.340 µs | No detected change, p=0.833 |

The candidate was removed because the intended scalar benefit was not
established. These results do not prove equivalence. The observed SQL change
does not isolate causality: its generated operation and guest parser were
unchanged. All Go allocation counts were unchanged.

The retained tests exercise full-width interleaved handles, contained nested
guest failures, active ownership through GC followed by eventual cleanup,
alias Close and shared destructor results, and shared-instance serialization.
The compiled facade fixture blocks an admitted guest call, observes actual
terminal admission, rejects new resource/constructor/stateless calls, then
checks drain and once-only home release. Its subprocess is CGO-disabled; the
parent race invocation does not instrument that child. Runtime and real proof
tests separately pass under the race detector.

### Profiles and qualification

A working Go-built `pprof` frontend was recovered from
`.cache/go/00/00f01208b59cc8aadfd1faa6e2680739540033a3f6e8fedd79f8e89539b0530e-d/pprof`.
The distribution toolchains still omit that frontend; profile inspection is
no longer blocked. The matching retained pre-holdout scalar profile attributes
58.29 percent cumulatively to the guest export, 43.58 percent to sketch math,
and 10.16 percent to `ResultView`. A separate PGO scalar profile attributes
53.17, 41.80, and 8.99 percent respectively. These cumulative categories
overlap and cannot be added or treated as independent costs. Their difference
does not establish which compiler optimization caused a timing change.

The matching retained SQL profile puts 85.15 percent cumulatively in the
guest export and 77.87 percent in translated function `fn96` (49.02 percent
flat). Both retained profiles match build ID
`5005a40a835d17d4d76c9237ae6169b3c810febe` and the pre-holdout binary hash
`7657ad19a6fdb499d9650bede60c2f25df31e5ff9cfe2eed06dbac805e965fe2`.

In the retained profile, the log polynomial's dense source line accounts for
about 14 percent flat and saturated float conversion for about 2.14 percent
flat. Those are sampled attribution, not promised optimization gains. The fifth
pass above investigates exact lowering; preserve rounding and sketch bin boundaries.
Never substitute approximate math or edit generated modules alone.

Restored production is byte-identical to the frozen pre-experiment benchmark
binaries, both with and without PGO. Full check, integration, cross-platform
compilation, external/relocation/Go-only smoke, all five fuzz targets, Go 1.26.8
PGO-enabled tests and race checks, and native linux/386 targeted tests pass.
Both proofs rebuild to the original artifacts and locks. An independently
authored external consumer also builds and runs with PGO off/on, exercising
numeric boundaries, atomic batch errors, SQL decoding, and terminal admission.
Each retained test commit passes darna and exact staged-snapshot Go/proof tests.
The pass's commits are local and unpushed.

The Cocoon profile SHA-256 is
`f75e852793200b22ce61668326860e9047cc7150404cced26cad489bafbc3af2`;
the reference profile is
`1334598cd2ffbd0fd8e3684c9c22ea3909e233ae7bfcc94603958ad8aa0792c7`.
Cocoon off/on binary hashes are
`d5fdf9913af526e3e93779283186146557af9ca3ba4bb6c7328c3bf23b471a80` and
`d9c949317052141c92ee7371cc734a34f66179f077f6ac6101ec8f58bb0a20dc`;
reference off/on hashes are
`bf3f15d3241684b35b0b0b448799373e8d66951f955827cce38fd7e365237dd8` and
`f8f51b711bd9571a09609937e15860e8fdc00fcac927bdde45fdfe777471d06f`.
The rejected fused binary is
`7a8706ad61ca772448bb517b7614459f4b3dce26eed06a9453837ac95206d1fb`.
The off training and PGO scalar profile build IDs are respectively
`83928cd627234b621c12e8f5ac6d8179b7d3fc44` and
`e3b76825c536eb9e92f0fe3e9f17df5058fd90d5`.

Ignored raw logs use `.cache/round10-pgo-{off,on,reference-off,reference-on}-{1..7}.txt`,
`.cache/round10-pgo-confirm-*`, `.cache/round11-fused-{before,after,reference}-{1..7}.txt`,
`.cache/round10-pgo-final-*`, and `.cache/round11-fused-confirm-*`. Training profiles are
`.cache/round10-{cocoon,reference}-mixed.prof`; separate PGO scalar attribution
uses `.cache/round10-pgo-sketch.prof`. Restored qualification logs use
`.cache/round11-restored-*`; staged checks use `.cache/round11-staged-*`.
Matching retained profile summaries are `.cache/round11-retained-*-top.txt`
and `.cache/round11-retained-sql-cumulative.txt`.

## Third performance loop results

The 2026-10-05 and 2026-10-06 multi-agent loop completed nine rounds. No
performance optimization survived the full workload comparison. The branch
retains a destructor correctness fix and stronger callback, handle, numeric,
and optimizer regressions. The range predicate initially retained in `0dd1701`
was withdrawn in `db580ca`; its tests remain. Binaryen remains at 133 with
ordinary `-O3`, and the original Datadog Wasm and translated module are restored.

The retained production binary is byte-identical to the fix-only binary used
in the longer three-way comparison. Seven rotated one-second sample sets used
Go 1.26.8, CGO disabled, no PGO, the AMD Ryzen 7 5800HS, 16 Go processors, and
no CPU affinity. Loop-start, destructor-fix-only, and range-candidate order
rotated; the unchanged independent reference followed each set. All compilation
and quality jobs had finished before measurement. Both Cocoon builds contain
the same final Datadog Go tests, including the newly added boundary tests.

| Version | SQL median | SketchAdd median |
| --- | --- | --- |
| Matched loop-start production (`969dfca`) | 1.782 µs | 74.63 ns |
| Retained production (`db580ca`) | 1.767 µs | 74.33 ns |
| Independent reference | 1.617 µs | 46.65 ns |

No retained timing gain or regression is established: SQL p=0.927, scalar
p=0.710, batch p=1.000, and traces p=0.383, each with n=7. Batch medians are
35.492 versus 35.558 µs; trace medians are 2.1380 versus 2.1595 ms. Go allocation
counts remain one, zero, one, and one respectively. Construction contributes
amortized bytes to stateless benchmarks; bytes/op changes do not imply an
eliminated per-call allocation. These exploratory comparisons have not been
adjusted for testing multiple candidates.

Remaining same-machine reference overhead is 9.3 percent for SQL and
59.3 percent for SketchAdd. SQL's median is below both the original 2.2154 µs
limit and this run's 1.7787 µs limit, but that narrow margin is not a guarantee
across sessions. SketchAdd exceeds both 55.44 ns and this run's 51.315 ns limit.
Overall performance acceptance remains open. The change from the earlier
11.3/66.8-percent overhead figures is not a code improvement: the paired
loop-start comparison does not establish one.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Loop start | 1755, 1767, 1834, 1796, 1800, 1782, 1741 | 75.38, 73.68, 71.93, 74.84, 74.63, 75.22, 71.98 |
| Retained | 1738, 1823, 1846, 1767, 1763, 1766, 1809 | 71.32, 73.17, 74.33, 75.80, 78.12, 76.28, 72.41 |
| Reference | 1609, 1597, 1675, 1617, 1560, 1647, 1640 | 45.23, 47.91, 47.09, 46.65, 45.34, 47.50, 45.97 |

The final binary SHA-256 is
`7657ad19a6fdb499d9650bede60c2f25df31e5ff9cfe2eed06dbac805e965fe2`;
the matched loop-start binary is
`4315e367c165904a824280a531420205e8ba665c16d51984860735933047d5e4`;
the unchanged reference binary is
`bf3f15d3241684b35b0b0b448799373e8d66951f955827cce38fd7e365237dd8`.
The retained binary also matches the earlier `round3-point-base.test`, despite
later test-only commits and regenerated source locks. The Datadog Wasm hash is
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`;
translated Go is
`f50d20e7fe091569353ac1ced7865a53cd23bc9add668c3aac0f5f3a0ac8eed4`.

Full check, integration, race, cross-platform compilation, external CLI and
Go-only smoke, and all five differential/parser/hardening fuzz targets pass
after restoring the candidates. Go 1.26.8 and native linux/386 targeted tests
pass. Both proof packages reproduce their artifacts, generated tests, and
locks byte-for-byte. Runtime coverage remains 97.0 percent; Go and Rust
generator coverage remains 96.7 and 94.6 percent. Each code/test commit passed
darna and exact staged-snapshot Go tests. The loop's commits are local and have
not been pushed; the preceding pass's CI completed successfully.

Raw acceptance samples are `.cache/round8-attribution-{before,fix,reference}-{1..7}.txt`.
The superseded 500 ms final screen is `.cache/loop-final-{before,final,reference}-*`.
Profiles were captured separately from acceptance samples in
`.cache/loop-retained-{sketch,sql}.prof`. A cached frontend recovered during the
fourth pass makes these available for attribution; the retained scalar and SQL
findings are recorded above. Existing earlier attributed profiles remain below.

### Multi agent experiment results

Rounds 1 through 7 used seven alternating 500 ms pairs, one Go processor and
CPU 14 affinity for screening. Longer confirmations used one-second pairs.
Rounds 8 and 9 used the unrestricted 16-processor configuration above.
No benchmark ran concurrently with compilation, lint, tests, fuzzing, or agent
work. Outliers remain in the samples; inconclusive results are not proof of
equivalence. Screening gains are not additive and do not override the final
workload comparison.

| Round | Experiment | Evidence and disposition |
| --- | --- | --- |
| 1 | Conditional reused Call reset | No detected change in any workload; removed. Callback replacement/reservation regressions retained in `c1c5f52`. |
| 2 | Specialized successful unit decoding | Confirmation: scalar −2.20% (p=0.024), traces +1.64% (p=0.004); removed. |
| 3 | Ordered finite range predicate | Scalar screen −1.93% (p=0.008), no detected SQL/trace change; initially retained, later withdrawn after rounds 8 and final screening. Numeric regressions retained. |
| 4 | Single checked mutable slab lookup | Confirmation: scalar −5.35%, SQL +1.93%, batch +1.54%, traces +2.71%; removed. Handle retirement/reuse tests retained in `be98f8a`. |
| 5 | Binaryen O4 | Confirmation: batch −0.46% (p=0.026), no established SQL/scalar gain; kept O3. Exact policy and semantic contract tests retained. |
| 6 | O3 without StackIR | No established gain; noisy baseline outliers retained. Removed. Eager-select and operand-order tests retained. |
| 7 | O3 to convergence | No established gain, with noisy candidate outliers; removed. Convergence semantic variant retained. |
| 8 | Three-way attribution | Baseline→range build SQL +2.13% (p=0.011), no detected scalar change. No detected baseline→fix-only change; fix-only→range SQL inconclusive (p=0.128). Withdrew the range change without claiming its isolated causality. |
| 9 | Unsigned float-bit classifier | Versus fix-only: batch −2.85% (p=0.007), SQL +2.55% (p=0.035), no detected scalar change; removed. |

The initial unrestricted range screen also found SQL +3.03 percent (p=0.038)
without establishing a scalar gain. Static inspection found the SQL export
and parser source unchanged by either the destructor fix or predicate changes.
Code layout, cache, or compiler effects are hypotheses, not demonstrated causes.
Do not manually pad or edit generated modules to chase those timings.

The destructor fix (`3fb2ffc`) closes a pre-existing canonical-unit gap. Its
`ResultUnit` helper delegates to fully checked borrowed `ResultView` and rejects
nonempty successful replies. Generated destructors use it; ordinary unit
methods retain their existing borrowed-view/empty checks. It does not add work
to either timed SQL or scalar benchmark path. A compiled mock-facade regression
demonstrably fails against the old destructor generator. Mock fixture subprocess
tests use CGO-disabled Go; the parent race invocation does not instrument that
subprocess. Runtime and real proof tests separately run under the race detector.

Binaryen candidates kept all existing features, validation, meta-DCE, memory
limits, and AST hardening. An independent WAT/wazero fixture asserts expected
numeric, signed-zero, NaN transport, rounding, trap ordering, memory effects,
eager-select, and operand-order contracts for O3, O4, no-StackIR, and convergence.
This supplements Go-versus-the-same-Wasm differential tests, which cannot detect
a shared optimizer error. `make smoke` includes the fixture in CI.

The O4 diagnostic optimization run took 1.36 seconds/127,916 KiB peak RSS
versus O3's 1.01 seconds/118,328 KiB; these are single observations, not stable
build-cost estimates. Three alternating convergence runs took 2.19–2.21 seconds
versus O3's 1.05 seconds. Convergence keeps its final non-improving iteration;
smaller output is not guaranteed by that policy.

| Datadog optimizer | Wasm bytes | Translated Go bytes |
| --- | --- | --- |
| Range-based O3 experiment baseline | 293861 | 2240411 |
| O4 candidate | 294285 | 2235047 |
| No StackIR candidate | 294759 | 2242288 |
| Convergence candidate | 293312 | 2238105 |

These sizes compare the same range-predicate source, not the restored final
source. Candidate Wasm hashes are O4
`4084e0a45346fe69eba53cd9493211bc0ac5339133da33e426f8a1bb83a7d7d8`,
no StackIR `5108d1e3eccca33c6f59d9e4fcdeb6cdff650749caa98b2af97e76fe8fa8702d`,
convergence `b71afc1c54900a191020da9e83dd7d99f519fbd9712bb00d911cc111e6e7c310`,
and the float-bit classifier
`626db603bf535387ab4e3c91eb7c6cb3eafb820fb70c4bd64cc11ebce4c6ceeb`.
Ignored raw logs use `.cache/round1-call-*`, `.cache/round2-unit-*`,
`.cache/round3-point-*`, `.cache/round4-slab-*`, `.cache/round5-o4-*`,
`.cache/round6-no-stack-ir-*`, `.cache/round7-converge-*`,
`.cache/round8-attribution-*`, and `.cache/round9-bits-*`.

## Second performance pass

The second 2026-10-05 pass retained four host-side optimizations, with no change
to the Wasm artifacts or translator pin. Seven interleaved 500 ms sample sets
used Go 1.26.8, CGO disabled, no PGO, the AMD Ryzen 7 5800HS, 16 Go processors,
no CPU affinity, and the same query and increasing-value workloads. Pre-pass
and current order alternated; the independent reference followed each pair.
All compilation, linting, tests, and fuzzing had finished before measurement.

| Version | SQL median | SketchAdd median |
| --- | --- | --- |
| Before second pass (`6082ad7`) | 1.880 µs | 80.00 ns |
| Current (`0c653b3`) | 1.839 µs | 77.31 ns |
| Independent reference | 1.652 µs | 46.34 ns |

SketchAdd improves by 3.36 percent (p=0.001, n=7); SQL has no detected change
(p=0.128). Batch sketch medians are 36.402 versus 36.468 µs (p=0.902);
trace medians are 2.1968 versus 2.1637 ms (p=0.209). None establishes a timing
regression. The pre-pass trace samples include a 2.976 ms outlier, retained
rather than filtered. Allocation counts are unchanged: one for SQL, zero for
scalar adds, one for batch adds, and one for traces. Stateless bytes/op include
amortized lazy guest construction; changes in that metric are not evidence
of an eliminated per-call allocation.

Remaining same-machine reference overhead is 11.3 percent for SQL and
66.8 percent for SketchAdd. Both exceed this run's 10-percent limits of
1.8172 µs and 50.974 ns. SQL is below the original absolute 2.2154 µs limit
in this run; SketchAdd remains above 55.44 ns. Overall acceptance remains open.
The unchanged pre-pass binary also runs faster here than in the first pass,
so the historical 96.45 to current 77.31 ns difference is not a code improvement.
Use the paired 80.00 to 77.31 ns comparison.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Before | 1904, 1880, 1923, 1856, 1825, 1950, 1815 | 80.28, 84.27, 80.00, 79.24, 79.53, 79.25, 82.58 |
| Current | 1798, 1864, 1873, 1836, 1839, 1845, 1756 | 73.86, 78.76, 77.31, 76.32, 78.99, 77.65, 75.21 |
| Reference | 1638, 1678, 1652, 1670, 1660, 1620, 1652 | 45.87, 46.34, 49.23, 52.08, 47.02, 44.88, 45.25 |

Five separate 300 ms layer samples give medians of 46.07 ns for the guest,
65.90 ns for the checked instance, 72.94 ns for resource use, and 80.00 ns
for the diagnostic facade. Empty instance/resource callbacks take 14.23 and
22.06 ns; lifecycle enter/leave takes 6.198 ns. Five separate 500 ms samples
put the ready pool at 73.75 ns. These callbacks differ from the public benchmark
and are not exact additive cost accounting.

Separate three-second profiles attribute 55.6 percent of SketchAdd samples
cumulatively to the translated export, 39.2 percent to the sketch algorithm,
and 11.8 percent to reply validation. SQL's translated export accounts for
85.9 percent cumulatively; its parser accounts for 79.6 percent. These percentages
overlap, describe sampled execution rather than independent costs, and do not
establish the next optimization's benefit. The final profile build ID is
`c09b4d52203a98b361e0e66e1a3f7a18f8e66559`.

`make check`, `make race`, `make cross`, `make integration`, `make smoke`, and
`make fuzz` pass. Both proofs reproduce on repeat builds. Targeted tests also
pass with Go 1.26.8 and native linux/386. Runtime coverage is now 97.0 percent,
up from 95.6 percent before this pass. Each code or test commit passed darna
and exact staged-snapshot Go tests. Bounds, reply checks, panic containment,
shared ownership, cancellation, and terminal draining Close remain intact.
Raw final samples use `.cache/pass2-final-{before,final,reference}-{1..7}.txt`.
The review also found a pre-existing destructor unit-reply strictness gap;
it was subsequently fixed in the multi-agent loop above.

## First performance pass

The first 2026-10-05 pass retained three small optimizations: inline unit success
publication (`e099514`), a checked successful-status fast path (`14e056c`), and
cold unit-error publication (`078c9c9`). The final seven interleaved 500 ms sample
sets used Go 1.26.8, CGO disabled, no PGO, the AMD Ryzen 7 5800HS, and the same
query and increasing-value workloads. Release and current order alternated
between sets; each set also measured the independent reference. All compilation
and quality checks had finished before timing.

| Version | SQL median | SketchAdd median |
| --- | --- | --- |
| Release artifact | 2.269 µs | 114.4 ns |
| Current | 2.272 µs | 96.45 ns |
| Independent reference | 1.959 µs | 55.48 ns |

SketchAdd improves by 15.69 percent (p=0.001, n=7); SQL has no detected change
(p=0.710). Go allocation counts remain one for SQL and zero for SketchAdd.
Current overhead relative to the same-machine reference is 16.0 percent for
SQL and 73.8 percent for SketchAdd. Neither the original absolute limits
(2.2154 µs and 55.44 ns) nor the current same-machine 10-percent limits
(2.1549 µs and 61.028 ns) are met. M1 performance acceptance remains open.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Release | 2320, 2250, 2269, 2281, 2285, 2256, 2261 | 116.2, 114.4, 113.5, 113.2, 114.0, 115.9, 115.1 |
| Current | 2267, 2279, 2240, 2259, 2274, 2273, 2272 | 95.42, 96.51, 95.87, 96.85, 96.85, 96.25, 96.45 |
| Reference | 1946, 1948, 1974, 1970, 1948, 1975, 1959 | 55.18, 55.97, 55.45, 54.88, 55.48, 57.10, 55.63 |

Final five-sample layer medians are 54.86 ns for the direct guest, 78.08 ns for
the checked instance, 89.81 ns for the resource layer, and 97.83 ns for the
complete facade. These diagnostic callbacks differ from the public benchmark
and are not exact additive accounting. Separate three-second profiles attribute
53.7 percent of SketchAdd cumulatively to its translated export, 38.6 percent
to the sketch algorithm, and 7.4 percent to reply validation. SQL attributes
80.5 percent to its translated export. Cumulative percentages overlap.

`make check`, `make race`, `make cross`, `make integration`, and `make smoke`
pass for the final code. Both proofs reproduce across repeat builds; their
generated differential fuzz tests pass. Targeted tests also pass with Go
1.26.8 and native linux/386. Runtime statement coverage remains 95.6 percent.
Every code commit passed darna and the exact staged Go snapshot tests. No safety
checks or shutdown guarantees were removed, and no translator pin was changed.

## Release baseline

On 2026-10-05, the released Datadog artifact was remeasured after the source-path
and compiler-metadata reproducibility fixes. Five interleaved, unprofiled samples
used the same machine, Go 1.26.8, CGO disabled, no PGO, and 500 ms per benchmark.
Compilation, linting, tests, and fuzzing were finished before measurement.

| Version | SQL median | SQL Go allocations | SketchAdd median | SketchAdd Go allocations |
| --- | --- | --- | --- | --- |
| Cocoon release artifact | 2.225 µs | 1 | 113.5 ns | 0 |
| Independently rebuilt spike | 1.954 µs | 1 | 55.41 ns | 0 |

This is 13.9 percent SQL overhead and 104.8 percent SketchAdd overhead relative
to the same-machine reference. Both original absolute targets and the fresh
same-machine 10-percent targets remain unmet. Historical pre-release numbers
below must not be treated as the release's current performance.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Release | 2267, 2245, 2218, 2225, 2221 | 114.7, 115.6, 113.0, 113.5, 112.9 |
| Spike | 1943, 1954, 1960, 1942, 1994 | 56.26, 56.08, 55.41, 55.25, 55.12 |

### Call layer measurements

`BenchmarkSketchAddLayers` executes the same increasing-value workload through
the guest alone, instance serialization and reply validation, resource ownership,
and the complete facade. Each subbenchmark checks the resulting guest count
outside the timer. Only the facade is a supported public call; the other layers
deliberately bypass protections for measurement, not as alternative APIs.

Five 500 ms samples produced these medians, all with zero Go allocations:

| Layer | Time |
| --- | --- |
| Direct guest | 58.68 ns |
| Instance and checked reply | 83.56 ns |
| Resource and checked instance | 97.70 ns |
| Complete facade | 117.9 ns |
| Empty instance callback | 18.58 ns |
| Empty resource and instance callback | 27.51 ns |
| Lifecycle enter and leave | 7.208 ns |

These diagnostics include callback dispatch and have different compiler layouts
from the public benchmark. Differences between rows are approximate cumulative
costs, not independent additive guarantees. The release SketchAdd profile puts
53.9 percent cumulatively in the translated export and 6.4 percent in reply
validation. A separate SQL profile puts 79.2 percent in its translated export.
Profiles used three-second runs and are not acceptance timings.

### Unit reply inlining

Forcing Rust to inline `reply_unit` was measured independently before any other
change. Both proof artifacts and locks were regenerated. The paired medians
were 113.8 to 98.53 ns for SketchAdd, a 13.4 percent reduction with no Go
allocations. SQL moved from 2.243 to 2.262 µs, within about one percent; this
experiment does not establish a SQL improvement.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Before | 2228, 2281, 2232, 2243, 2244 | 113.8, 113.7, 114.3, 114.2, 113.0 |
| Inline unit reply | 2262, 2261, 2292, 2258, 2269 | 103.2, 98.53, 98.56, 98.04, 98.27 |

The descriptor, error encoding, output limit, and canonical empty success
contract are unchanged. Native unit-reply tests and an additional translated
versus wazero test cover data-to-unit and error-to-unit transitions.

### Successful reply status fast path

`ResultView` now skips general status dispatch for `OK` only after validating
the descriptor, output limit, and complete logical memory range. Error messages
still become owned Go data, and reserved or unknown statuses still poison the
instance. New regressions cover negative and unknown codes and a zero-length
success reply beyond logical memory.

A five-pair 500 ms screening run was followed by seven interleaved one-second
samples with execution order alternated between pairs. The longer run's medians
were 97.61 to 95.90 ns for SketchAdd and 2.298 to 2.296 µs for SQL. `benchstat`
reported a 1.75 percent SketchAdd reduction (p=0.026, n=7) and no detected SQL
change (p=0.594). Allocation counts remain zero and one respectively. This is
a modest improvement, not evidence that the remaining overhead is solved.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Before | 2270, 2301, 2291, 2298, 2309, 2288, 2301 | 98.10, 97.36, 97.73, 97.61, 97.46, 97.60, 100.1 |
| Status fast path | 2279, 2299, 2276, 2296, 2310, 2282, 2297 | 95.62, 100.8, 95.76, 95.73, 95.90, 96.08, 96.26 |

### Cold unit error publication

A broader seven-pair comparison caught a tradeoff missed by the first screening:
unit inlining plus the host fast path reduced SketchAdd by 15.05 percent but
increased SQL from 2.234 to 2.293 µs (2.64 percent, p=0.001). A weaker `#[inline]`
hint produced byte-identical Datadog Wasm, so it could not address that slowdown.

Moving only unit-error publication into a private `#[cold]` helper retained the
inline success path. Seven further 500 ms pairs alternated release and candidate
order, followed each pair with the previous implementation, and included batch
sketches and trace obfuscation. The release-to-candidate medians were:

| Benchmark | Release | Cold unit helper | Detected change |
| --- | --- | --- | --- |
| SQL | 2.230 µs | 2.241 µs | None, p=0.251 |
| SketchAdd | 113.2 ns | 95.89 ns | −15.29 percent, p=0.001 |
| SketchAddMany1k | 46.39 µs | 45.90 µs | None, p=0.165 |
| ObfuscateTraces1k | 2.561 ms | 2.574 ms | None, p=0.620 |

All allocation counts are unchanged. The trace samples include a 3.091 ms
candidate outlier; do not interpret “none detected” as proof of identical
performance under every workload. The SQL recovery is measured code-generation
evidence, not a claim that a unit-error helper directly speeds up the SQL parser.
Native tests cover all four error statuses, embedded NUL and Unicode messages,
stable descriptor addresses, output limits, and error-to-unit transitions.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Release | 2234, 2230, 2221, 2426, 2217, 2234, 2217 | 113.3, 113.2, 113.3, 113.1, 113.8, 113.2, 112.8 |
| Previous | 2330, 2314, 2290, 2286, 2284, 2295, 2293 | 97.12, 97.85, 96.63, 96.31, 95.72, 96.02, 101.7 |
| Cold unit helper | 2270, 2229, 2224, 2241, 2241, 2232, 2255 | 95.89, 95.17, 95.17, 96.02, 99.55, 96.45, 95.54 |

### Discarded guest layout experiments

Adding `#[cold]` to the argument, application, limit, and handle error constructors
did not establish a repeatable public-call improvement. SQL medians were
2.301 versus 2.278 µs; paired differences varied in sign. SketchAdd medians were
98.65 versus 99.49 ns. The annotations were removed rather than retaining a
code-layout change without convincing evidence.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Inline unit baseline | 2259, 2318, 2301, 2309, 2273 | 98.95, 100.5, 98.65, 98.40, 98.12 |
| Cold errors | 2288, 2278, 2239, 2229, 2290 | 97.77, 99.49, 98.46, 99.85, 101.6 |

Forcing both `Slab::index` and `Slab::get_mut` to inline also failed to establish
a benefit. SketchAdd medians were 99.12 versus 99.41 ns; SQL was 2.291 versus
2.273 µs, again with mixed-sign paired differences. Those annotations were
removed independently of the cold error experiment.

| Version | SQL samples in ns | SketchAdd samples in ns |
| --- | --- | --- |
| Inline unit baseline | 2276, 2263, 2291, 2326, 2300 | 99.02, 99.12, 98.29, 102.8, 102.3 |
| Inline resource lookup | 2266, 2296, 2273, 2250, 2299 | 99.41, 99.00, 99.53, 98.62, 101.4 |

## Second performance pass

The second 2026-10-05 pass retained four host-side changes. All screening
comparisons used Go 1.26.8, CGO disabled, no PGO, seven alternating 500 ms
sample pairs, CPU 14 affinity, and one Go processor. CPU affinity made the
small scalar differences easier to distinguish. These screening values must
not be compared directly with the earlier unrestricted 16-processor results.

| Change | Measured operation | Before | After | Detected change |
| --- | --- | --- | --- | --- |
| Skip error matching for successful callbacks (`c7e148e`) | SketchAdd | 79.80 ns | 78.24 ns | −1.95 percent, p=0.001 |
| Inline checked memory ranges (`469e865`) | SketchAdd | 79.27 ns | 77.83 ns | −1.82 percent, p=0.026 |
| Direct lifecycle admission for resource calls (`aec4d6d`) | SketchAdd | 78.61 ns | 75.78 ns | −3.60 percent, p=0.001 |
| Receive ready pool slots without a blocking select (`b30d62e`) | PoolAvailable | 109.60 ns | 72.37 ns | −33.97 percent, p=0.001 |

These percentages describe separate comparisons and must not be added. No
screening comparison established a significant SQL improvement. The pool
experiment moved SQL from 1.861 to 1.802 µs, but p=0.079 does not establish a
full-facade gain. The isolated pool benchmark is a diagnostic, not a substitute
for SQL acceptance.

Successful callbacks still undergo the post-call memory check; wrapped and
joined protocol errors still poison their instance. Range errors now construct
a private error value and format their unchanged diagnostic on demand. The
smaller success path inlines with Go 1.26.8 while preserving logical-length,
overflow, spare-capacity, and zero-length checks and
`errors.Is(err, rt.ErrProtocol)`.

Resource methods have no caller context, so checking `context.Background().Err()`
was redundant. They now enter the same lifecycle directly, retaining nil-wrapper,
nil-library, ownership, epoch, terminal admission, and draining Close behavior.
Stateless operations still check their caller context. Both facades and locks
were regenerated; both Wasm artifacts and translated guest sources are unchanged.

The pool retains context checks before and after borrowing. A ready slot uses
a nonblocking receive; an unavailable slot still waits on slot availability or
context cancellation. The existing panic-safe release path is unchanged. New
tests deterministically cancel after both lazy and initialized ready borrows,
cancel while waiting, and verify capacity and instance reuse.

| Comparison | Before samples in ns | After samples in ns |
| --- | --- | --- |
| Successful callback | 79.53, 80.37, 79.50, 80.21, 79.80, 81.09, 79.79 | 78.04, 79.00, 78.24, 78.07, 78.41, 78.20, 79.11 |
| Checked ranges | 79.72, 79.27, 79.86, 80.07, 78.81, 78.59, 78.56 | 77.22, 77.06, 77.83, 77.20, 78.65, 78.74, 78.96 |
| Resource admission | 79.28, 77.77, 77.18, 79.48, 80.20, 78.61, 77.45 | 76.87, 76.41, 75.55, 74.96, 75.78, 76.77, 74.69 |
| Ready pool | 109.0, 109.6, 109.1, 112.3, 110.2, 111.4, 109.5 | 71.14, 74.13, 71.01, 73.20, 72.83, 71.05, 72.37 |

### Rejected second pass experiments

Publishing an encoded reply by taking its vector instead of copying into the
cached output buffer passed native tests but regressed SQL from 1.946 to
2.055 µs (5.60 percent, p=0.007) and traces from 2.205 to 2.278 ms
(3.27 percent, p=0.001). Allocation counts were unchanged. Changing ownership
also changes buffer lifetimes; the comparison does not isolate the regression's
cause. The candidate and its regenerated artifacts were removed. Its Datadog
Wasm hash was `d1eaa8b1dd37afd1fcf715f719df04a7de18447a10b925276b348959c4575d5a`.

Combining resource unlock and KeepAlive defers moved SketchAdd from 75.82 to
74.71 ns but did not establish a significant gain (p=0.259). The production
change was removed. Its panic-unlock and GC-during-use regressions passed against
the original implementation and were retained independently (`0c653b3`).

An unrestricted initial range-inlining screen was inconclusive. Forcing 16 Go
processors onto one CPU produced heavy scheduling variance and is not retained
as performance evidence. The table above uses the cleaner one-processor
confirmation, not either noisy screen. Raw logs remain in ignored `.cache`
under `pass2-pinned`, `pass2-bounds-confirm`, `pass2-admission`, `pass2-pool`,
`pass2-owned`, and `pass2-resource-exit` prefixes.

## Historical pre release results

On 2026-10-04, five interleaved samples of the pre-change binary, independently
rebuilt spike, and current binary ran on the local AMD Ryzen 7 5800HS. Each
sample used Go 1.26.8, CGO disabled, no PGO, and 500 ms per benchmark. Compilation,
linting, tests, and fuzzing were finished before measurement. SQL used the same
query; SketchAdd used increasing values. Medians are:

| Version | SQL time | SQL Go allocations | SketchAdd time | SketchAdd Go allocations |
| --- | --- | --- | --- | --- |
| Cocoon before this pass (`e9052fd`) | 2.299 µs | 2 | 114.4 ns | 0 |
| Independently rebuilt spike | 1.950 µs | 1 | 55.18 ns | 0 |
| Cocoon after this pass (`ec3398d`) | 2.298 µs | 1 | 103.2 ns | 0 |

SketchAdd improved by about 9.8 percent. SQL eliminated one allocation and
reduced allocated bytes from 105–106 to 57–58 per operation, but its runtime
did not improve beyond sample variation. A scalar Count reply now also decodes
without Go allocation, protected by an allocation regression test.

The original spike medians, recorded on a different cloud VM, were 2.014 µs
and 50.4 ns. The original within-10-percent limits remain **2.2154 µs and
55.44 ns**. Neither was met in that session. Those results also exceeded 110 percent of the
fresh same-machine spike medians; do not mark M1 performance accepted.

For reproducible accounting, the five unprofiled samples in nanoseconds were:

| Version | SQL samples | SketchAdd samples |
| --- | --- | --- |
| Before | 2313, 2296, 2303, 2298, 2299 | 114.7, 121.1, 113.2, 113.8, 114.4 |
| Spike | 1936, 1944, 1950, 1961, 1983 | 54.87, 55.53, 55.18, 55.23, 55.18 |
| After | 2315, 2292, 2298, 2306, 2298 | 103.6, 103.0, 103.2, 103.2, 102.9 |

## Comparison provenance

The spike was built from its own unchanged Rust sources and Cargo lock using
Rust 1.97.0, rebuilt `std,panic_abort`, Binaryen 133 meta-DCE and `-O3`, the
268435456-byte memory cap, and wasm2go v0.4.16 with `-unsafe`. An independently
written AST preparation step applied its bulk-helper length bound. Go overlays
supplied the generated module and embedded Wasm without modifying reference
sources or copying its implementation into Cocoon. Its Go tests and wazero
differential tests passed before the final comparison.

Both builds use libdatadog revision
`7f3b16fe1b4bfc2a016ef869459e915ee02d6d6a`, but their Cargo dependency versions
and guest contracts differ. The spike uses four library instances and a
background context; Cocoon uses one instance and the benchmark context. The
benchmarks are serial. This is a comparison of the complete implementations,
not a controlled estimate of safety-check costs alone.

The reference Cargo lock SHA-256 is
`c39203ded9bc15f520da886bf3d1bcab8e0b6addd583d91edbdde3cfe5519c77`;
its rebuilt Wasm SHA-256 is
`8bda5ff7acff11fd64fd47a0a48165a71d610c28413665e4cc4d4034150d0651`.
The historical pre-release Datadog Wasm SHA-256 was
`2b0b48e84f8d38f705ce49ee52609a49a7e5112dfc59d68d0d257ce3d7becef6`.
The release baseline Wasm SHA-256 is
`9af0f6a8dcf934b3eb977dfdabe11ef5224bc1cded91353ee5f6687b86fd9f95`.
The retained unit-inlining artifact SHA-256 is
`d9b0eee8b99d2a728dc231ed10e4579fba22d4602c9f2860c5c0ed93e93ffd7e`.
The host status fast path does not change the guest artifact.
The final cold unit-error helper artifact SHA-256 is
`ae0119ca34206c2fcd8b69ae4e81421214417abb364d589c3afabcf16d8c6333`.

## Implemented optimizations and next experiments

The runtime now reuses call state under its instance lock, admits lifecycle
calls atomically, avoids unused scalar input reservations, copies string inputs
directly, and consumes guest-owned string/byte inputs without a second clone.
SketchAdd's initial ~162 ns/one-allocation measurement fell to ~114 ns/zero
allocations before the earlier pass. That pass added typed unit replies, cached
the fixed output descriptor address, and decoded borrowed replies under the guest
lock before copying retained fields. Both native Rust and generated Go retain
their validation, and public `Result` still returns an owned copy.

The earlier post-change five-second SketchAdd CPU profile attributed about
52 percent of samples to translated guest execution, including 34 percent to the sketch
algorithm. Reply validation accounted for about 9 percent cumulatively.
Resource ownership, instance serialization, lifecycle admission, and generated
callbacks account for much of the remainder; cumulative percentages overlap.
These observations suggest experiments, not proof that a particular rewrite
will help.

The first 2026-10-05 pass added the isolated benchmarks and separate SQL profile above,
retained unit-reply inlining, successful-reply status dispatch, and a focused
cold unit-error helper. It rejected general cold error and resource lookup
annotations that did not establish an improvement. The second pass optimized
successful callback bookkeeping, checked range inlining, resource admission,
and ready pool borrowing. It rejected owned reply publication and combined
resource exit defers. The third loop retained correctness and optimizer-contract
tests, not a timing gain. The fourth pass found a synthetic PGO scalar benefit
with a batch regression and rejected fused resource calls. The fifth found no
justified log-polynomial rewrite and no established guarded-conversion gain.
Next, prioritize measured batch-only or guest SQL experiments and representative
application profiles; adding screening gains is not valid accounting.
Retain bounds checks, canonical replies, trap containment, shared ownership,
and draining Close, with staged-snapshot tests for each atomic change. See the
[roadmap](roadmap.md) for other pending acceptance work.

## Benchmarking and profiles

Run repeatable samples with `make bench`. To repeat the measured configuration,
use the pinned comparison compiler and finish compilation before measurement:

```sh
CGO_ENABLED=0 /usr/lib64/go/1.26/bin/go test -pgo=off -c \
  -o .cache/dd-bench.test ./examples/datadog/go/dd
.cache/dd-bench.test -test.run '^$' \
  -test.bench '^(BenchmarkObfuscateSQL|BenchmarkSketchAdd)$' \
  -test.benchtime=500ms -test.count=5 -test.benchmem
```

Collect profiles separately from unprofiled acceptance samples. For the
synthetic four-class experiment, explicitly exclude the varied held-out and
call-layer diagnostic benchmarks:

```sh
CGO_ENABLED=0 /usr/lib64/go/1.26/bin/go test -pgo=off -c \
  -o .cache/dd-profile.test ./examples/datadog/go/dd
GOMAXPROCS=16 .cache/dd-profile.test -test.run '^$' \
  -test.bench '^(BenchmarkObfuscateSQL|BenchmarkSketchAdd|BenchmarkSketchAddMany1k|BenchmarkObfuscateTraces1k)$' \
  -test.benchtime=2s -test.cpuprofile=.cache/dd-mixed.prof
go tool pprof -top .cache/dd-profile.test .cache/dd-mixed.prof
```

This gives each class equal nominal time, not a representative request mix.
For deployment, collect profiles from the consuming application's workload.
Merge compatible profiles with `go tool pprof -proto a.prof b.prof > default.pgo`,
place `default.pgo` beside that application's main package, or pass
`go build -pgo=/absolute/path/to/profile ./cmd/app`. A library does not enable
PGO globally for its consumers; Cocoon does not ship a default profile.
See the [Go PGO guide](https://go.dev/doc/pgo) for consuming-main placement and
profile selection.

Go PGO changes host compilation, not the pinned Rust/Wasm translation or
schema. Compare with explicit `-pgo=off`, profile each implementation separately,
and rerun tests and race checks after changing compiler options. Benchmark
profiles remain experimental until representative of deployed traffic.
Profiles can contain application symbols and paths; review them before sharing.
