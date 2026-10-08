# Dependency license inventory

From the repository root, run:

```sh
make licenses        # regenerate LICENSE-3rdparty.csv and LICENSE-3rdparty.txt
make licenses-check  # resolve again and compare without changing either file
```

The equivalent commands are `go run -mod=readonly ./tools/licenses` and
`go run -mod=readonly ./tools/licenses -check`. The collector uses only Go's
standard library and the installed Go/Cargo toolchains; there is no license
scanner to install. Network access is needed until module/crate caches are warm.
The Datadog example requires the sibling `../libdatadog` checkout described in
the root README, at commit `7f3b16fe1b4bfc2a016ef869459e915ee02d6d6a`.
CI checks out that exact revision automatically.

The collector discovers every `go.mod` and `Cargo.toml` under the repository,
including examples and test fixtures, excluding hidden/cache/build/vendor
directories. `go list -mod=readonly -m -json all` inventories the complete
selected module graph, including indirect, test and tool modules. Local and
versioned replacements use the replacement's actual license evidence. Each
Go module is resolved independently with `GOWORK=off`, so a developer's workspace
cannot silently change the result.

`cargo metadata --locked --all-features --format-version=1` inventories all
resolved packages across each workspace, including indirect, development,
build and target-specific dependencies. It does not filter by the host target.
Cargo's `OR`/`AND` expressions are retained; legacy `/` separators mean `OR`.
External path and Git dependencies are included, with inherited repository
licenses and notices. Cocoon's own Cargo packages must declare `Apache-2.0`.
The selected Go/Cargo graphs are authoritative: unused historical entries in a
lockfile are not dependencies. Alternative versions under mutually exclusive
feature resolutions require their own audited build configuration.

Rows identify the ecosystem, package, resolved version, source, consuming
manifests, license expression, evidence filenames and SHA-256 hashes. Multiple
versions or sources remain separate rows. The CSV's license expression is the
package's main license, **not a claim that every file has identical terms**.
The text bundle preserves discovered `LICENSE`, `LICENCE`, `UNLICENSE`, `COPYING`, `NOTICE`,
`COPYRIGHT`, `AUTHORS` and `PATENTS` files, including nested files and explicitly
declared Cargo `license_file` paths. For example, crossbeam-channel's
`LICENSE-THIRD-PARTY` includes CC-BY-3.0 and other attributions in addition to
the crate's `MIT OR Apache-2.0` declaration. Read those notices when redistributing.

The bundle also includes Binaryen's pinned license and the Rust 1.97.0 standard
library's complete upstream copyright document, including its file-specific
exceptions and build dependencies. Rust standard-library code is embedded in
generated Go payloads. These reviewed documents are vendored under `notices/`,
with their provenance, fingerprints and version-pin checks in `reviewed.json`.
Other compiler/toolchain internals and system libraries are outside the package
inventory. It covers this repository; a user's own shim and dependency graph need
a separate inventory. `rt` currently imports only Go's standard library; the full inventory
also covers Cocoon's CLI, tests, generation tooling and examples.

## Dependency updates and review

1. Update dependencies and commit their manifest/lockfile changes together.
2. Run `make licenses`. No manifest or lockfile is rewritten by the collector.
3. Review changes to both generated files, including upstream copyright and
   additional notice texts. Include both in the change for CI.

Go does not provide license metadata. `reviewed.json` therefore binds each Go
module's reviewed SPDX expression to the exact SHA-256 of its upstream license
file. A new module, unknown license or changed reviewed license fails closed;
inspect the full upstream terms before adding/updating that entry. Do not copy a
hash merely to make CI pass. Rust crates without a license declaration or license
texts also fail and require review. For legitimate `license_file`-only crates,
the `rust` section accepts reviewed `package`, `license`, `file` and `sha256`
fields; the exact upstream file must match before that expression is used.
This check does not choose
between dual licenses or prohibit particular license families.

CI compares the **entire** CSV and bundle, catching removed/added dependencies,
versions, sources, scope and changed evidence, including hand-deleted rows. Check
mode never writes the outputs. Generation resolves and validates all inputs
before replacing files, and uses atomic writes. Output contains no timestamps
or machine-specific cache paths.

For distributions, ship Cocoon's `LICENSE` and `NOTICE` plus the applicable
upstream licenses and attribution notices from `LICENSE-3rdparty.txt`. A
generated package retains the licenses of its Rust inputs and standard library;
Cocoon's Apache-2.0 license does not relicense those inputs. Consumers should
retain the notices that apply to their chosen runtime, shim and embedded payload.
