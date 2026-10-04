package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/darccio/cocoon/internal/gen"
	"github.com/darccio/cocoon/internal/gen/gogen"
	"github.com/darccio/cocoon/internal/harden"
	"github.com/darccio/cocoon/internal/manifest"
)

// Features enables the supported synchronous Rust instruction subset.
func Features() []string {
	return []string{"--mvp-features", "--enable-mutable-globals", "--enable-bulk-memory", "--enable-sign-ext", "--enable-nontrapping-float-to-int", "--enable-reference-types", "--enable-multivalue"}
}

// Lock records deterministic tool, source, and artifact identities.
type Lock struct {
	Tools        ToolVersions `json:"tools"`
	Schema       string       `json:"schema_sha256"`
	Manifest     string       `json:"manifest_sha256"`
	Cargo        string       `json:"cargo_lock_sha256"`
	Rust         string       `json:"shim_sha256"`
	Wasm         string       `json:"wasm_sha256"`
	Go           string       `json:"go_sha256"`
	Facade       string       `json:"facade_sha256"`
	Adapter      string       `json:"adapter_sha256"`
	Contracts    string       `json:"contract_tests_sha256"`
	Differential string       `json:"differential_tests_sha256"`
	Bulk         string       `json:"bulk_tests_sha256"`
	Sources      []SourceHash `json:"sources"`
}

type pipeline struct {
	runner  Runner
	sources func(context.Context, *manifest.Manifest) ([]byte, []byte, error)
}

// Project builds a locked shim, validates and translates it, and records provenance.
func Project(ctx context.Context, directory string, m *manifest.Manifest) (*Lock, error) {
	return (pipeline{runner: ExecRunner{}, sources: gen.Sources}).run(ctx, directory, m)
}

func (p pipeline) run(ctx context.Context, directory string, m *manifest.Manifest) (_lock *Lock, err error) {
	if validationErr := m.Validate(); validationErr != nil {
		return nil, validationErr
	}
	absolute, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	directory = absolute
	moduleRoot, err := FindModule(directory)
	if err != nil {
		return nil, err
	}
	runner := p.runner
	versions, err := Doctor(ctx, runner, moduleRoot, m.Toolchain)
	if err != nil {
		return nil, err
	}
	_, _, memory, err := m.ByteLimits()
	if err != nil {
		return nil, err
	}
	work := filepath.Join(directory, ".cocoon-build")
	release, err := gen.Guard(directory)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, release()) }()
	identities, err := sourceIdentities(ctx, runner, directory, m)
	if err != nil {
		return nil, err
	}
	rust, facade, err := p.sources(ctx, m)
	if err != nil {
		return nil, err
	}
	rustPath := filepath.Join(directory, "shim", "src", "cocoon_gen.rs")
	previous, previousErr := os.ReadFile(rustPath) // #nosec G304 -- Back up the owned generated glue before compilation.
	if previousErr != nil && !errors.Is(previousErr, os.ErrNotExist) {
		return nil, previousErr
	}
	if generationErr := gen.WriteGenerated(rustPath, rust); generationErr != nil {
		return nil, generationErr
	}
	defer func() {
		if err != nil {
			if previousErr == nil {
				err = errors.Join(err, gen.WriteGenerated(rustPath, previous))
			} else {
				err = errors.Join(err, os.Remove(rustPath))
			}
		}
	}()
	cargo := runner
	if executable, ok := runner.(ExecRunner); ok {
		executable.Environment = append(executable.Environment, "RUSTC_BOOTSTRAP=1", "CARGO_TARGET_DIR="+filepath.Join(work, "target"), fmt.Sprintf("CARGO_TARGET_WASM32_UNKNOWN_UNKNOWN_RUSTFLAGS=-C link-arg=--max-memory=%d", memory))
		cargo = executable
	}
	if _, err = cargo.Run(ctx, filepath.Join(directory, "shim"), "rustup", "run", m.Toolchain.Rust, "cargo", "build", "--locked", "--release", "-Zbuild-std=std,panic_abort", "--target", "wasm32-unknown-unknown"); err != nil {
		return nil, err
	}
	raw := filepath.Join(work, "target", "wasm32-unknown-unknown", "release", strings.ReplaceAll(m.Package.RustCrate, "-", "_")+".wasm")
	type root struct {
		Name   string `json:"name"`
		Export string `json:"export"`
		Root   bool   `json:"root"`
	}
	var roots []root
	for name := range ExportSet(m) {
		roots = append(roots, root{Name: name, Export: name, Root: true})
	}
	roots = append(roots, root{Name: "memory", Export: "memory", Root: true})
	slices.SortFunc(roots, func(a, b root) int { return strings.Compare(a.Name, b.Name) })
	graph, err := json.Marshal(roots)
	if err != nil {
		return nil, err
	}
	graphPath := filepath.Join(work, "exports.json")
	if graphErr := os.WriteFile(graphPath, graph, 0o600); graphErr != nil {
		return nil, graphErr
	}
	pruned, final := filepath.Join(work, "pruned.wasm"), filepath.Join(work, "final.wasm")
	args := append([]string{raw, "--graph-file", graphPath, "-o", pruned}, Features()...)
	if _, err = runner.Run(ctx, directory, "wasm-metadce", args...); err != nil {
		return nil, err
	}
	args = append([]string{pruned, "-O3", "-o", final}, Features()...)
	if _, err = runner.Run(ctx, directory, "wasm-opt", args...); err != nil {
		return nil, err
	}
	wasm, err := os.ReadFile(final) // #nosec G304 -- Read the fixed output of the just-completed local build.
	if err != nil {
		return nil, err
	}
	parsed, err := Verify(m, wasm)
	if err != nil {
		return nil, err
	}
	translatedPath := filepath.Join(work, "translated.go")
	if _, err = runner.Run(ctx, moduleRoot, "go", "tool", "wasm2go", "-unsafe", "-pkg", "wasm", "-o", translatedPath, final); err != nil {
		return nil, err
	}
	translated, err := os.ReadFile(translatedPath) // #nosec G304 -- Read the fixed translator output inside the project build directory.
	if err != nil {
		return nil, err
	}
	required, err := harden.Required(translated)
	if err != nil {
		return nil, err
	}
	translated, err = harden.Rewrite(translated, required)
	if err != nil {
		return nil, err
	}
	adapter, err := gogen.Adapter(m, len(parsed.Imports) != 0)
	if err != nil {
		return nil, err
	}
	contracts, err := gogen.Contracts(m)
	if err != nil {
		return nil, err
	}
	differential, err := gogen.WazeroTests(m)
	if err != nil {
		return nil, err
	}
	bulk, err := gogen.BulkTests(required)
	if err != nil {
		return nil, err
	}
	output := filepath.Join(directory, "go", m.Package.Name)
	_, schema, err := m.SchemaHash()
	if err != nil {
		return nil, err
	}
	cargoLock, err := os.ReadFile(filepath.Join(directory, "shim", "Cargo.lock")) // #nosec G304 -- Read the selected project's locked dependency input.
	if err != nil {
		return nil, err
	}
	source, err := sourceDigest(filepath.Join(directory, "shim", "src"))
	if err != nil {
		return nil, err
	}
	lock := &Lock{Tools: versions, Schema: schema, Manifest: digest(m.Content), Cargo: digest(cargoLock), Rust: source, Wasm: digest(wasm), Go: digest(translated), Facade: digest(facade), Adapter: digest(adapter), Contracts: digest(contracts), Differential: digest(differential), Bulk: digest(bulk), Sources: identities}
	lockData, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return nil, err
	}
	if publicationErr := gen.WriteSet([]gen.Artifact{
		{Path: filepath.Join(output, "cocoon_gen.go"), Data: facade, Generated: true},
		{Path: filepath.Join(output, "internal", "wasm", "module.go"), Data: translated, Generated: true},
		{Path: filepath.Join(output, "zz_adapter.go"), Data: adapter, Generated: true},
		{Path: filepath.Join(output, "zz_contract_test.go"), Data: contracts, Generated: true},
		{Path: filepath.Join(output, "zz_wazero_test.go"), Data: differential, Generated: true},
		{Path: filepath.Join(output, "internal", "wasm", "zz_bulk_test.go"), Data: bulk, Generated: true},
		{Path: filepath.Join(output, "testdata", "module.wasm"), Data: wasm},
		{Path: filepath.Join(directory, "cocoon.lock.json"), Data: append(lockData, '\n')},
	}); publicationErr != nil {
		return nil, publicationErr
	}
	return lock, nil
}

// FindModule locates the Go module that declares the pinned translator tool.
func FindModule(directory string) (string, error) {
	for {
		if info, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil && !info.IsDir() {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("no go.mod with the pinned wasm2go tool above %s", directory)
		}
		directory = parent
	}
}
func digest(data []byte) string { hash := sha256.Sum256(data); return hex.EncodeToString(hash[:]) }
func sourceDigest(directory string) (encoded string, err error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	hash := sha256.New()
	err = fs.WalkDir(root.FS(), ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if slices.Contains([]string{".git", "target", ".cache", ".cocoon-build"}, entry.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		var contentHash []byte
		if entry.Type()&os.ModeSymlink != 0 {
			// Hash relative links as links. Stat enforces containment; the target
			// itself is hashed at its normal location by the same tree walk.
			if _, statErr := root.Stat(path); statErr != nil {
				return statErr
			}
			target, linkErr := root.Readlink(path)
			if linkErr != nil {
				return linkErr
			}
			sum := sha256.Sum256([]byte("symlink\x00" + target))
			contentHash = sum[:]
		} else {
			if !entry.Type().IsRegular() {
				return fmt.Errorf("nonregular source %s", path)
			}
			file, openErr := root.Open(path)
			if openErr != nil {
				return openErr
			}
			fileHash := sha256.New()
			_, copyErr := io.Copy(fileHash, file)
			if closeErr := errors.Join(copyErr, file.Close()); closeErr != nil {
				return closeErr
			}
			contentHash = fileHash.Sum(nil)
		}
		if _, writeErr := hash.Write([]byte(path)); writeErr != nil {
			return writeErr
		}
		if _, writeErr := hash.Write([]byte{0}); writeErr != nil {
			return writeErr
		}
		_, writeErr := hash.Write(contentHash)
		return writeErr
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
