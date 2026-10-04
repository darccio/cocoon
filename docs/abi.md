# Cocoon ABI v3

The synchronous ABI uses wasm32 pointers, numeric Wasm parameters, and one
per-instance cached input buffer/output descriptor. It has no WASI imports.

## Identity and exports

Every module exports its memory and exactly these base functions plus manifest
operations (extra exports are rejected):

| Export | Signature |
| --- | --- |
| `cocoon_abi_version` | `() -> i32`, value 3 |
| `cocoon_schema_hash` | `() -> i64` |
| `cocoon_init` | `() -> ()` |
| `cocoon_in_reserve` | `(i32 length) -> i32 pointer` |
| `cocoon_out` | `() -> i32 descriptor` |
| `cocoon_trim` | `(i32 keep) -> ()` |
| `cocoon_alloc` | `(i32 length) -> i32 pointer` |

The facade checks ABI and schema before admitting operations. The schema is
the low-order, little-endian first eight bytes of SHA-256 over canonical JSON
of semantic declarations. Byte units, numeric instance counts, and omitted
default ownership are normalized; declaration order is ignored except parameter
and record field order. Build paths, Cargo crate names, Go import paths, source
revisions, and tool pins are not schema inputs. The lock also records the full SHA-256.

An operation export is `cocoon_<name>`; resource exports are
`cocoon_<lowercase-resource>_<method>`. Each returns an i32 status. Scalars map
directly to Wasm numeric types; bool is i32 with only 0/1 accepted. All variable
arguments use separate `(pointer, byte-length)` pairs within one aggregate
reservation. Resource methods prepend an i64 handle; constructors return that
handle through the normal output buffer.

## Input/output values

The host sums every variable argument's encoded size before allocating or
encoding. Slice sizing checks multiplication overflow. The guest verifies that
every input region belongs to the current reservation, not merely its linear
memory. Strings are valid UTF-8, bytes are uninterpreted, and scalar arrays are
contiguous little-endian values of their natural width (4 or 8 bytes).

A record begins with a little-endian u32 field count. Each field is a u32 ID,
u32 byte length, and exactly those bytes. IDs are the declared 1-based positions
and must appear once, in that exact order. Fields are required: missing,
duplicate, unknown, reordered, truncated, and trailing data are rejected.
Nested records use the same format; scalar fields use little-endian bytes and
bool uses one byte, 0 or 1. Multiple logical results must be a named record.

`cocoon_out` points at two little-endian u32 values: output pointer and byte
length. Its address is fixed after initialization, including across trim and
memory growth; the host resolves it once per instance, without retaining a
memory view. The host checks descriptor bounds, `max_output`, and logical memory
length, then copies or decodes before releasing the execution lock. It always
re-reads current memory after guest allocations; views never expose spare Go
capacity. No view or `*rt.Call` may escape its call.

`rt.Call.Result` returns an owned byte copy. Generated facades instead decode
`ResultView` under the execution lock, copying retained string and byte fields
and allocating owned scalar arrays. Typed results never borrow guest memory.

| Status | Meaning |
| --- | --- |
| 0 `OK` | Success; declared result encoding or empty output for unit |
| 1 `ERR_APP` | Expected application error message |
| 2 `ERR_ARG` | Invalid argument or encoding |
| 3 `ERR_HANDLE` | Stale/unknown handle |
| 4 `PENDING` | Reserved; rejected in M1 |
| 5 `ERR_LIMIT` | Size limit exceeded |

Malformed replies/unknown statuses poison an instance. Expected application,
argument, and handle errors do not. Recoverable panics become classified guest
faults or shielded host errors and poison the instance. With log capability,
Rust's panic hook sends the message at log level 4 before abort becomes a Wasm
unreachable trap.

## Ownership and shutdown

Handles pack a generation in the high 32 bits and slot index plus one in the
low 32 bits. Zero is invalid. Reusing a slot increments its generation; a slot
at the last generation retires rather than wrapping. Separate slabs belong to
each resource type. Host ownership additionally binds the handle to its exact
instance and epoch, preventing cross-instance use.

Stateless operations use a lazy bounded pool. Resources use one serialized home
instance. A poisoned home is replaced for new resources; old handles never
migrate. Wrapper aliases share the ownership cell and destructor token. Cleanup
is attached to that cell, not a copyable wrapper; explicit Close is preferred.
Destruction runs once and outside the cell lock.

Library Close rejects new calls immediately, drains every admitted call, then
releases pool/home instances and invalidates resources. Concurrent Close calls
wait for the same result; Close is terminal and idempotent. Synchronous calls
cannot be preempted; Close has no forced-abort deadline. Pool factory/callback/
recycler panics cannot lose capacity; callback panics discard the guest and
are re-raised for the caller.

## Capabilities and future async work

Only declared `cocoon.log(i32,i32,i32)`, `cocoon.random_get(i32,i32)->i32`, and
`cocoon.clock_nanos()->i64` may survive in M1 imports. The verifier rejects
extra namespaces, duplicate imports, incorrect signatures, unbounded/shared/
memory64 declarations, a start function, or unsupported instruction features.
Binaryen validates instruction bodies independently of the structural reader.

The next milestone needs explicit future cancellation, tracked host operations,
bounded call/byte admission, one terminal loop-fault/close path that unblocks
producers and waiters, budgeted drain, and `max+1` HTTP response reads that reject
overflow. It must add structured configuration and HTTP/sleep capabilities;
none of those contracts is implied by M1's context parameter or reserved status.
