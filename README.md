# Cocoon

Cocoon turns a safe Rust shim into an ordinary Go package. Its checked-in output
needs no Rust compiler, Wasm engine, cgo, or native shared library at runtime.
The synchronous M1 framework supports typed values, records, shared resources,
bounded calls, terminal shutdown, and declared host capabilities. Async and HTTP
are deliberately not implemented yet. The [roadmap](docs/roadmap.md) records
remaining work and how to resume development in a later conversation.

## Private repository access

The repository is private and no license has been selected yet. Installation
requires authorized GitHub access. Authenticate Git first, for example with
`gh auth login` and `gh auth setup-git`, then exclude this private module from
the public Go proxy and checksum service:

```sh
GOPRIVATE=github.com/darccio/cocoon go install github.com/darccio/cocoon/cmd/cocoon@main
GOPRIVATE=github.com/darccio/cocoon go get github.com/darccio/cocoon@main
```

Use a release tag instead of `main` when one is available. Keep any other
existing `GOPRIVATE` entries when configuring your environment. Consumers of
checked-in Go packages need only Go; authoring a new Rust shim also requires
a matching source checkout for `rust/cocoon-guest` and the tools below.

## Try the checked-in packages

Go 1.26 or newer is sufficient for consumers:

```sh
CGO_ENABLED=0 go test ./...
CGO_ENABLED=0 go test ./testdata/compute/go/compute/...
go build -o bin/cocoon ./cmd/cocoon
```

The proof consumer is `github.com/darccio/cocoon/examples/datadog/go/dd`:

```go
library, err := dd.Open(dd.Options{Instances: 1})
if err != nil {
    return err
}
defer func() { _ = library.Close() }()
query, err := library.ObfuscateSQL(ctx, "SELECT * FROM users WHERE id = 42")
```

It exposes SQL and v0.4 MessagePack trace obfuscation and pointer-only `*Sketch`
resources. Explicitly close resources before closing their library. Go wrapper
copies share one ownership cell, but `go vet` warns against making such copies.
The small [compute fixture](testdata/compute/cocoon.toml) covers every numeric
scalar/slice, nested records, multiple input buffers, and a real safe Rust panic.

## Build or author a shim

Generation requires Rust **1.97.0**, Binaryen **133**, and the Go tool dependency
`github.com/ncruces/wasm2go` **v0.4.16**. This module already pins the translator.
For a separate author module, first add that tool with
`go get -tool github.com/ncruces/wasm2go@v0.4.16` and add the Cocoon Go dependency.

```sh
rustup toolchain install 1.97.0 --profile minimal
rustup component add rust-src rustfmt clippy --toolchain 1.97.0
rustup target add wasm32-unknown-unknown --toolchain 1.97.0
```

Install Binaryen 133 and put `wasm-opt`, `wasm-as`, and `wasm-metadce` on `PATH`.
Use the [official release bundle](https://github.com/WebAssembly/binaryen/releases/tag/version_133)
so local proof rebuilds and CI use identical executables; a version number alone
does not identify a binary. On linux/amd64, install the exact bundle used by CI:

```sh
curl --fail --location https://github.com/WebAssembly/binaryen/releases/download/version_133/binaryen-version_133-x86_64-linux.tar.gz -o binaryen.tar.gz
echo '2dc9c7813f5375db93d96ead4b78222fcc3e2677bbb832297af4797782a37489  binaryen.tar.gz' | sha256sum --check
tar -xzf binaryen.tar.gz
export PATH="$PWD/binaryen-version_133/bin:$PATH"
```

From a Cocoon source checkout, check the tools and initialize a project:

```sh
go run ./cmd/cocoon doctor
go run ./cmd/cocoon init --name example testdata/my-example
```

`init` works inside an existing Go module. Outside this repository, provide
`--guest /path/to/cocoon/rust/cocoon-guest`. It creates an independent Cargo
workspace, an echo manifest, safe implementation, generated trait/facade, and
Cargo lockfile. It refuses existing authored files; it does not initialize Git
or alter your module dependencies.

For a separate author module, after installing the CLI and the pinned tools:

```sh
go mod init example.com/my-library
GOPRIVATE=github.com/darccio/cocoon go get github.com/darccio/cocoon@main
go get -tool github.com/ncruces/wasm2go@v0.4.16
cocoon init --guest /absolute/path/to/cocoon/rust/cocoon-guest project
cocoon doctor --manifest project/cocoon.toml
cocoon build --manifest project/cocoon.toml
go mod tidy
CGO_ENABLED=0 go test ./project/go/example/...
```

Edit `cocoon.toml`, run `cocoon gen`, then implement the generated Rust `API`
trait in `shim/src/implementation.rs`. That module has `forbid(unsafe_code)`;
pointer handling stays in generated glue and the reviewed guest support crate.
Use the [Datadog manifest](examples/datadog/cocoon.toml) as a resource example.

```sh
go run ./cmd/cocoon gen --manifest testdata/my-example/cocoon.toml
go run ./cmd/cocoon build --manifest testdata/my-example/cocoon.toml
go run ./cmd/cocoon verify --manifest testdata/my-example/cocoon.toml \
  testdata/my-example/go/example/testdata/module.wasm
```

`gen` writes source contracts; `build` finalizes the translated module, adapter,
generated contract/differential/bulk-memory tests, exact reference Wasm, and
`cocoon.lock.json`. Commit generated output with its manifest and authored shim.
Only generated-marked source files may be replaced. All artifacts are staged
before publication and earlier replacements roll back on failure. Individual
renames are atomic; the group is not an atomic filesystem snapshot. Do not read
or compile output during publication. `gen` and `build` exclude one another with
`.cocoon-build/lock`; after an interrupted process, remove that exact stale lock
only after confirming no build is running.

The build uses `--locked`, build-std, panic=abort, a bounded linear memory, exact
manifest export roots, meta-DCE, `wasm-opt -O3`, structural/feature verification,
and the unsafe translator followed by checked AST hardening. `RUSTC_BOOTSTRAP=1`
is scoped to the pinned build-std subprocess, not a general nightly toolchain.
Compiler source roots for the module, local dependencies, Rust sysroot, and
Cargo cache are remapped to stable `/cocoon/...` paths. Encoded compiler flags
also preserve paths containing spaces. These mappings normalize compiler
output, not arbitrary strings emitted by your shim or a build script; see
[Rust source path remapping](https://doc.rust-lang.org/rustc/remap-source-paths.html).
Cargo also hashes absolute paths of dependencies outside the shim workspace
into compiler metadata. Cocoon wraps target Rust compilation to derive stable
crate identities from the locked graph, canonical source roots, package
versions, and compiler settings. It preserves Cargo's expected output filenames
and keeps host build scripts and compiler probes unchanged. Features and the
standard-library compilation role remain distinct; diagnostic presentation
does not affect artifact identity. The exact normalizer source is fingerprinted
in the lock and compiler flags, so changing it invalidates Cargo's cached units.
The build subprocess selects the pinned compiler explicitly and replaces
ambient Cargo compiler wrappers; these changes do not affect your shell.
If an offline machine lacks standard-library dependencies, fetch them on a
networked machine with:

```sh
RUSTC_BOOTSTRAP=1 cargo +1.97.0 fetch --locked \
  --manifest-path "$(rustc +1.97.0 --print sysroot)/lib/rustlib/src/rust/library/Cargo.toml"
```

## Datadog rebuild

The generated Go consumer does not need libdatadog. Rebuilding it needs a clean
local checkout at `../libdatadog`, revision
`7f3b16fe1b4bfc2a016ef869459e915ee02d6d6a`:

```sh
cargo fetch --manifest-path examples/datadog/shim/Cargo.toml
make integration
```

Locks record full schema/raw manifest hashes, exact tools, compiler metadata
normalizer and Binaryen executable hashes, local source content and revisions,
Cargo lock and shim hashes, and hashes of Wasm, translated Go,
facade, adapter, and generated tests. They contain no build-machine paths or
timestamps. The independently authored getrandom 0.2 compatibility crate routes
Wasm entropy through Cocoon even when a dependency enables its `js` feature;
it is not a general replacement for every upstream getrandom backend.

## Quality and limits

```sh
make check       # tests, vet, and 49 explicitly selected strict linters
make race        # runtime and generated consumers
make rust        # native guest/getrandom tests and all-feature Clippy
make cross       # compile tests + vet for arm64, 386, Darwin, Windows, js/wasm
make smoke       # installed CLI, deterministic new shim, Go-only consumer
make fuzz        # parsers, hardener, and generated wazero differential adapters
make coverage
make bench
```

Use golangci-lint **2.13.1**; older analyzer builds cannot read newer Go export
data. CI is configured to execute linux/amd64 and native arm64 tests and rebuild
both proof packages with pinned tools. Wazero is test-only; production imports
stay in the Go runtime and standard library. See [ABI and lifecycle](docs/abi.md),
[security decisions](docs/design-review.md), and [performance/PGO](docs/performance.md).

Cocoon targets trusted, reviewed Rust payloads. Recoverable traps poison the
instance and invalidate its handles; host-import panics are distinct errors.
It is not a hostile-code sandbox: fatal Go stack exhaustion/process OOM cannot
be recovered, and synchronous execution cannot be forcibly interrupted.
Context cancellation prevents admission or stops waiting for a pool slot; an
already executing call runs to completion. Callbacks/imports must not reenter
their instance or library shutdown. Linear-memory limits are per instance, not
a process-wide budget; Go capacity, copies, compilation caches, and concurrent
instances consume additional memory. Limits must also fit the host's Go `int`.
