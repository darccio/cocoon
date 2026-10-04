# Adversarial review of the implementation plan

The first milestone covers synchronous compute and resources. Async operations,
HTTP, cancellation of guest futures, and the pipeline are a separate milestone.
The implementation is new; reference repositories supply evidence and test cases,
not source code.

## Contracts resolved before implementation

1. **Records need an actual wire format.** Fixed field order alone cannot detect
   duplicate or unknown fields. ABI v3 uses little-endian records: a u32 field
   count followed by ordered `(u32 field ID, u32 byte length, payload)` entries.
   IDs start at one. All fields are required. Decoders reject unknown, duplicate,
   missing, out-of-order, truncated, oversized, and trailing data. Strings are
   UTF-8. Numeric slices use little-endian elements, never Go memory layouts.
2. **Input limits are aggregate.** Sum every variable input, including record
   headers, with overflow checks before allocation or encoding. Bound scalar
   slices before multiplication. Bound outputs before copying and validate the
   complete descriptor against the current memory length.
3. **Schema identity must be reproducible.** Hash the normalized, validated
   semantic manifest using SHA-256; use the first eight digest bytes interpreted
   little-endian as the ABI schema identifier. Store the full digest in the lock.
   Exclude comments, TOML formatting, and build paths from schema identity.
4. **Pool resources cannot migrate.** Stateless functions use the pool. Every
   resource belongs to one serialized home instance and its epoch. Resource
   methods never borrow a different pooled instance. Shared wrapper copies refer
   to the same ownership cell; closing any copy invalidates every copy.
5. **Close must have one admission boundary.** Library admission and terminal
   closing state share a lock. Close rejects new calls, waits for admitted calls,
   then destroys resources and instances exactly once. Resource destructors run
   outside ownership-cell locks. Cleanup is a fallback, attached to that shared
   cell, with a token that does not retain the cell itself.
6. **Faults are terminal for an instance.** Call contains every recoverable guest
   panic, including unknown values. Host panics are shielded separately. A fault
   increments the instance epoch so existing handles fail consistently. A pool
   callback panic discards its instance, restores capacity in a defer, then
   propagates the caller panic. Initialization must also be contained.
7. **Verification is more than matching names.** Check function signatures,
   memory count, explicit memory maximum, namespace and signatures of capability
   imports, forbidden start functions, duplicate sections/names, section order,
   valid indexes, and the allowed feature set. Never accept unknown opcodes.
   Binaryen validates code before translation; the reader validates ABI sections.
8. **Hardening must tolerate dead-code elimination without tolerating drift.**
   wasm2go emits only needed helpers. Derive the expected helper set from the
   actual bulk instructions; require each expected helper exactly once, validate
   its AST shape, and reject missing, duplicate, or unexpected memory helpers.
   Bound both memory and data slices by length. Exercise zero-length operations
   beyond the memory end, overlapping copies, memory growth, and spare capacity.
9. **Generation and builds must not leave mixed artifacts.** Validate the entire
   manifest before writing. Generate into temporary directories, format and
   validate, then replace complete files. Reject path traversal and reserved Go,
   Rust, ABI, and generated API names. Build with locked dependencies, bounded
   output capture, and a cancellable context. Do not silently fall back from pins.
10. **Reproducibility requires recording inputs.** The lock records full manifest,
    guest/shim source, Cargo.lock, dependency revision, tool versions, final Wasm,
    and translated Go hashes. Exclude timestamps and absolute paths. Test two
    consecutive builds and compare bytes. Generated Wasm for differential tests
    must be the exact bytes translated into Go.
11. **Trusted payload is an explicit limit.** Generated Go cannot recover fatal
    stack exhaustion or preempt infinite guest loops. Per-instance memory limits
    are not a process-wide budget. Context cancellation controls pool admission;
    synchronous guest execution is not interruptible.

## Environment findings

The workspace starts empty. Its `.git` directory is mounted read-only; Git
metadata is stored in `.cocoon-git` for this session. Commands use
`GIT_DIR=$PWD/.cocoon-git GIT_WORK_TREE=$PWD`. Commit signing is disabled only in
this repository because the configured 1Password signing agent is unavailable.

Go 1.27.1, golangci-lint 2.11.4, darna, Binaryen 133, Rust 1.94.0 with rust-src,
and the wasm32 target are installed. Rust 1.97.0 and the libdatadog checkout at
revision 7f3b16f are absent. Terminal DNS/network access is unavailable. Exact
toolchain and Datadog proof checks must report those prerequisites as missing;
passing tests with available tools does not establish the missing proof.

The golangci-lint configuration enables 49 independent correctness, security,
API, documentation, and style checks with all vet analyzers and unchecked-error
checks. Generated code alone is excluded from style analysis; it remains covered
by compilation, vet, tests, and differential validation. Configuration validation
through the linter runs offline; its separate JSON-schema command requires the
network and is not part of the local quality target.
