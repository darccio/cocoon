# Building and distributing Cocoon packages

Applications that import a generated package need only Go 1.26 or newer. This
guide is for package authors and contributors rebuilding the Rust inputs.
Start with the [README walkthrough](../README.md#build-your-own-package) to
create your first package.

## Install the build tools

The supported toolchain is pinned:

| Tool | Version | Used for |
| --- | --- | --- |
| Go | 1.26 or newer | CLI, translation, and generated packages. |
| Rust | 1.97.0 | Compile the shim and standard library to Wasm. |
| Binaryen | 133 | Optimize and validate Wasm. |
| wasm2go | v0.4.16 | Translate Wasm into Go source. |

Install [Rust through rustup](https://www.rust-lang.org/tools/install), then:

```sh
rustup toolchain install 1.97.0 --profile minimal
rustup component add rust-src rustfmt clippy --toolchain 1.97.0
rustup target add wasm32-unknown-unknown --toolchain 1.97.0
```

Install Binaryen from the
[official version 133 release](https://github.com/WebAssembly/binaryen/releases/tag/version_133).
Choose the archive for your platform and put `wasm-opt`, `wasm-as`, and
`wasm-metadce` on `PATH`. A version number alone does not identify the
executables; the official bundle makes rebuilds comparable with CI.

For Linux on amd64, these commands install the exact bundle used by CI:

```sh
curl --fail --location --retry 3 \
  https://github.com/WebAssembly/binaryen/releases/download/version_133/binaryen-version_133-x86_64-linux.tar.gz \
  -o binaryen.tar.gz
echo '2dc9c7813f5375db93d96ead4b78222fcc3e2677bbb832297af4797782a37489  binaryen.tar.gz' | sha256sum --check
tar -xzf binaryen.tar.gz
export PATH="$PWD/binaryen-version_133/bin:$PATH"
```

An author module must pin the translator with:

```sh
go get -tool github.com/ncruces/wasm2go@v0.4.16
```

Use the same Cocoon version for the CLI, Go runtime, and Rust guest support
crate. The README derives the CLI version and source checkout tag from the
selected Go module. The CLI is installed in `GOBIN`, or `$(go env GOPATH)/bin`
when `GOBIN` is unset; include that directory on your `PATH`.

### Offline preparation

The build uses Rust's pinned `build-std` support. If an offline machine lacks
standard-library dependencies, fetch them on a networked machine with:

```sh
RUSTC_BOOTSTRAP=1 cargo +1.97.0 fetch --locked \
  --manifest-path "$(rustc +1.97.0 --print sysroot)/lib/rustlib/src/rust/library/Cargo.toml"
```

Also warm the project's own caches before going offline:

```sh
go mod download
cargo fetch --locked --manifest-path shim/Cargo.toml
```

## Authoring workflow

`cocoon init` requires an existing Go module. It creates a manifest, an
independent Cargo workspace, a safe echo implementation, generated Rust/Go
contracts, and a Cargo lockfile. It refuses existing authored files and does
not initialize Git or change your Go dependencies.

Inside a Cocoon checkout, the guest crate is discovered at
`rust/cocoon-guest`. In another module, provide `--guest` with the path to the
matching crate. The README uses a version-matched sibling checkout so the Cargo
dependency has a portable relative path. Recreate that layout and version when
rebuilding on another machine.

After initialization:

1. Declare functions, records, resources, capabilities, and limits in
   `cocoon.toml`. The [Datadog manifest](../examples/datadog/cocoon.toml) includes
   stateful resources; the [compute manifest](../testdata/compute/cocoon.toml)
   covers numeric values, slices, and nested records.
2. Run `cocoon gen` after API changes. Implement the generated `API` trait in
   `shim/src/implementation.rs`, which is compiled with `forbid(unsafe_code)`.
3. Run `cocoon doctor`, then `cocoon build`.
4. Run `go mod tidy` and the generated package's tests with `CGO_ENABLED=0`.

Commands default to the current directory's `cocoon.toml`. For another project:

```sh
cocoon gen --manifest project/cocoon.toml
cocoon build --manifest project/cocoon.toml
cocoon verify --manifest project/cocoon.toml \
  project/go/example/testdata/module.wasm
```

`verify` requires Binaryen and checks the supplied Wasm module's ABI,
capabilities, memory limits, exports, and instruction features against the
manifest.

## Generated files and distribution

For a project named `hello`, the main layout is:

```text
cocoon.toml                  API, capabilities, limits, and tool pins
cocoon.lock.json             Source, tool, and generated artifact fingerprints
shim/
  Cargo.toml
  Cargo.lock
  src/implementation.rs      Your Rust implementation
  src/cocoon_gen.rs          Generated Rust contract and glue
go/hello/
  cocoon_gen.go              Generated Go facade
  zz_adapter.go              Adapter to the translated module
  internal/wasm/
    module.go                Translated Rust payload
    zz_bulk_test.go          Generated bulk-memory tests
  zz_*_test.go               Generated contract and differential tests
  testdata/module.wasm       Reference Wasm used by the tests
```

Keep the authored inputs and generated output together in version control.
This lets package consumers build with Go and lets authors reproduce the
output. The reference Wasm and generated differential tests are test assets;
the production package runs the translated Go source. Differential tests use
wazero through Cocoon's `cocoontest` helper.

Retain applicable upstream licenses and copyright/attribution notices when
distributing Rust inputs or a generated payload. The repository's inventory
covers Cocoon and its checked-in examples; an author's new dependencies need
their own inventory. See the [license inventory guide](../tools/licenses/README.md).

### Publishing generated output safely

`gen` writes source contracts. `build` finalizes the translated module,
adapter, contract/differential/bulk-memory tests, reference Wasm, and lock.
Only source files marked as generated may be replaced.

All artifacts are staged before publication, and earlier replacements roll
back if publication fails. Individual renames are atomic; the group is not an
atomic filesystem snapshot. Avoid compiling or reading generated output
while publishing it.

`gen` and `build` exclude one another with `.cocoon-build/lock`. After an
interrupted process, remove that exact stale lock only after confirming no
build is running.

## Reproducibility and build internals

The build uses `--locked`, `build-std`, `panic=abort`, bounded linear memory,
exact manifest export roots, meta-DCE, `wasm-opt -O3`, structural/feature
verification, and the unsafe translator followed by checked AST hardening.
`RUSTC_BOOTSTRAP=1` is scoped to the pinned `build-std` subprocess.

Compiler source roots for the module, local dependencies, Rust sysroot, and
Cargo cache are remapped to stable `/cocoon/...` paths. Encoded compiler flags
preserve paths containing spaces. These mappings normalize compiler output,
not arbitrary strings emitted by a shim or build script; see
[Rust source path remapping](https://doc.rust-lang.org/rustc/remap-source-paths.html).

Cargo hashes absolute paths of dependencies outside the shim workspace into
compiler metadata. Cocoon wraps target compilation to derive stable crate
identities from the locked graph, canonical source roots, package versions,
and compiler settings. It preserves Cargo's expected output filenames and
keeps host build scripts and compiler probes unchanged. Features and the
standard-library compilation role remain distinct; diagnostic presentation
does not affect artifact identity.

The exact normalizer source is fingerprinted in the lock and compiler flags,
so changing it invalidates Cargo's cached units. The subprocess selects the
pinned compiler explicitly and replaces ambient Cargo compiler wrappers;
these changes do not affect your shell.

Locks record full schema and raw manifest hashes, exact tools, compiler
metadata normalizer and Binaryen executable hashes, local source content
and revisions, Cargo lock and shim hashes, and hashes of Wasm, translated Go,
facade, adapter, and generated tests. They contain no build-machine paths or
timestamps. CI rebuilds both checked-in examples and checks that the generated
files remain identical.

## Rebuild the Datadog example

The generated Go package does not need libdatadog at build or runtime.
Rebuilding its Rust inputs requires a clean sibling checkout at
`../libdatadog`, revision `7f3b16fe1b4bfc2a016ef869459e915ee02d6d6a`.
From a Cocoon source checkout, create that sibling checkout:

```sh
git init ../libdatadog
git -C ../libdatadog fetch --depth=1 https://github.com/DataDog/libdatadog.git \
  7f3b16fe1b4bfc2a016ef869459e915ee02d6d6a
git -C ../libdatadog checkout --detach FETCH_HEAD
cargo fetch --locked --manifest-path examples/datadog/shim/Cargo.toml
make integration
```

If the sibling directory already contains your own libdatadog checkout, use a
separate workspace for the example or check its revision and tracked changes
before rebuilding. The build verifies both the revision and clean source tree.

The independently authored getrandom 0.2 compatibility crate routes Wasm
entropy through Cocoon even when a dependency enables its `js` feature. It is
specific to this integration, rather than a replacement for all getrandom
backends.

## Contributor checks

Clone Cocoon and install the build tools above. Install golangci-lint
**2.13.1** for `make check`/`make lint`; older analyzer builds cannot read newer
Go export data. The Makefile also accepts `GOLANGCI_LINT=/path/to/golangci-lint`.

```sh
git clone https://github.com/darccio/cocoon.git
cd cocoon
make test
```

Use the checks relevant to your change:

| Command | Checks |
| --- | --- |
| `make check` | Go tests, vet, formatting, and strict lint. |
| `make race` | Race detector for the runtime and generated consumers. |
| `make rust` | Native guest/getrandom tests, formatting, and all-feature Clippy. |
| `make cross` | Compile tests and vet for Linux arm64/386, Darwin arm64, Windows amd64, and js/wasm. |
| `make integration` | Rebuild and verify both examples, then run Go and native Rust checks. |
| `make smoke` | Installed CLI, separate author module, deterministic rebuilds, and a Go-only consumer. |
| `make fuzz` | Manifest/Wasm parsers, hardener, and generated differential adapters. |
| `make coverage` | Go coverage report. |
| `make bench` | Datadog consumer benchmarks. |
| `make licenses-check` | Dependency inventory and upstream notice evidence. |

Integration and license checks require the pinned libdatadog checkout above.
CI executes native Linux amd64 and arm64 tests and rebuilds both examples with
pinned tools. See [performance notes](performance.md) for benchmark provenance
and [ABI and lifecycle](abi.md) for behavioral contracts.
