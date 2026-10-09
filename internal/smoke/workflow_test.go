package smoke_test

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/darccio/cocoon/internal/build"
)

// TestExternalWorkflow exercises the installed CLI and a Go-only distribution.
// It is opt-in because it needs the full pinned Rust/Binaryen toolchain.
func TestExternalWorkflow(t *testing.T) {
	if os.Getenv("COCOON_SMOKE") != "1" {
		t.Skip("run make smoke with the pinned build toolchain installed")
	}
	root, rootErr := filepath.Abs("../..")
	if rootErr != nil {
		t.Fatal(rootErr)
	}
	goExecutable, lookErr := exec.LookPath("go")
	if lookErr != nil {
		t.Fatal(lookErr)
	}
	work := t.TempDir()
	bin := filepath.Join(work, "bin")
	author := filepath.Join(work, "author")
	mkdir(t, bin)
	mkdir(t, author)
	environment := []string{"CGO_ENABLED=0", "GOWORK=off", "GOTOOLCHAIN=local"}
	run(t, root, append(environment, "GOBIN="+bin), goExecutable, "install", "./cmd/cocoon")
	cocoon := filepath.Join(bin, "cocoon")
	run(t, author, environment, goExecutable, "mod", "init", "example.com/cocoon-author")
	run(t, author, environment, goExecutable, "mod", "edit", "-go=1.26.0", "-replace=github.com/darccio/cocoon="+root)
	run(t, author, environment, goExecutable, "get", "github.com/darccio/cocoon@v0.0.0")
	run(t, author, environment, goExecutable, "get", "-tool", "github.com/ncruces/wasm2go@v0.4.16")
	run(t, author, environment, cocoon, "init", "--guest", filepath.Join(root, "rust", "cocoon-guest"), "project")
	manifest := filepath.Join(author, "project", "cocoon.toml")
	run(t, author, environment, cocoon, "doctor", "--manifest", manifest)
	run(t, author, environment, cocoon, "gen", "--manifest", manifest)
	run(t, author, environment, cocoon, "build", "--manifest", manifest)
	run(t, author, environment, cocoon, "verify", "--manifest", manifest, filepath.Join(author, "project", "go", "example", "testdata", "module.wasm"))
	run(t, author, environment, "rustup", "run", "1.97.0", "cargo", "fmt", "--manifest-path", "project/shim/Cargo.toml", "--check")
	run(t, author, environment, "rustup", "run", "1.97.0", "cargo", "clippy", "--manifest-path", "project/shim/Cargo.toml", "--all-targets", "--", "-D", "warnings")
	run(t, author, environment, goExecutable, "mod", "tidy")
	run(t, author, environment, goExecutable, "test", "./project/go/example/...")
	outputs := filepath.Join(author, "project", "go")
	first := snapshot(t, outputs)
	lockPath := filepath.Join(author, "project", "cocoon.lock.json")
	firstLock := read(t, lockPath)
	run(t, author, environment, cocoon, "build", "--manifest", manifest)
	if !reflect.DeepEqual(first, snapshot(t, outputs)) || !bytes.Equal(firstLock, read(t, lockPath)) {
		t.Fatal("rebuilding the external project changed generated artifacts")
	}
	if bytes.Contains(firstLock, []byte(work)) || bytes.Contains(firstLock, []byte(root)) {
		t.Fatal("lock contains machine-specific source paths")
	}
	firstWasm := read(t, filepath.Join(outputs, "example", "testdata", "module.wasm"))
	if bytes.Contains(firstWasm, []byte(work)) || bytes.Contains(firstWasm, []byte(root)) {
		t.Fatal("Wasm contains machine-specific source paths")
	}

	// Relocate both the project and guest support crate. This catches path
	// strings in compiled Rust that same-directory rebuilding cannot detect.
	relocated := filepath.Join(work, "relocated author with spaces")
	guestRoot := filepath.Join(work, "relocated cocoon with spaces")
	guest := filepath.Join(guestRoot, "rust", "cocoon-guest")
	mkdir(t, relocated)
	for relative, data := range snapshot(t, filepath.Join(root, "rust")) {
		write(t, filepath.Join(guestRoot, "rust", relative), string(data))
	}
	for _, metadata := range []string{"Cargo.toml", "Cargo.lock"} {
		write(t, filepath.Join(guestRoot, metadata), string(read(t, filepath.Join(root, metadata))))
	}
	for _, metadata := range []string{"go.mod", "go.sum"} {
		write(t, filepath.Join(relocated, metadata), string(read(t, filepath.Join(author, metadata))))
	}
	run(t, relocated, environment, cocoon, "init", "--guest", guest, "project")
	relocatedManifest := filepath.Join(relocated, "project", "cocoon.toml")
	run(t, relocated, environment, cocoon, "build", "--manifest", relocatedManifest)
	if !reflect.DeepEqual(first, snapshot(t, filepath.Join(relocated, "project", "go"))) || !bytes.Equal(firstLock, read(t, filepath.Join(relocated, "project", "cocoon.lock.json"))) {
		t.Fatal("relocating source directories changed generated artifacts")
	}

	// The consumer receives only production Go files: no Rust, Wasm fixture,
	// translator, generated differential tests, or checkout-only dependency.
	distribution := filepath.Join(work, "distribution")
	runtimeOnly := filepath.Join(work, "runtime")
	consumer := filepath.Join(work, "consumer")
	goOnlyBin := filepath.Join(work, "go-only-bin")
	mkdir(t, consumer)
	mkdir(t, goOnlyBin)
	copyProductionGo(t, outputs, filepath.Join(distribution, "project", "go"))
	copyProductionGo(t, filepath.Join(root, "rt"), filepath.Join(runtimeOnly, "rt"))
	copyProductionGo(t, filepath.Join(root, "examples", "datadog", "go", "dd"), filepath.Join(runtimeOnly, "examples", "datadog", "go", "dd"))
	write(t, filepath.Join(distribution, "go.mod"), "module example.com/cocoon-author\ngo 1.26.0\nrequire github.com/darccio/cocoon v0.0.0\n")
	write(t, filepath.Join(runtimeOnly, "go.mod"), "module github.com/darccio/cocoon\ngo 1.26.0\n")
	write(t, filepath.Join(consumer, "go.mod"), fmt.Sprintf("module example.com/cocoon-consumer\ngo 1.26.0\nrequire (\nexample.com/cocoon-author v0.0.0\ngithub.com/darccio/cocoon v0.0.0\n)\nreplace example.com/cocoon-author => %q\nreplace github.com/darccio/cocoon => %q\n", distribution, runtimeOnly))
	write(t, filepath.Join(consumer, "consumer_test.go"), consumerTest)
	if linkErr := os.Symlink(goExecutable, filepath.Join(goOnlyBin, "go")); linkErr != nil {
		t.Fatal(linkErr)
	}
	consumerEnvironment := append(slices.Clone(environment), "PATH="+goOnlyBin, "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off", "GOENV=off")
	run(t, consumer, consumerEnvironment, goExecutable, "mod", "tidy")
	run(t, consumer, consumerEnvironment, goExecutable, "test", "./...")
	dependencies := run(t, consumer, consumerEnvironment, goExecutable, "list", "-deps", "-test", "-f", "{{if and (not .Standard) (not .ForTest)}}{{.ImportPath}}{{end}}", "./...")
	imports := strings.Fields(string(dependencies))
	for _, required := range []string{"example.com/cocoon-author/project/go/example", "github.com/darccio/cocoon/rt", "github.com/darccio/cocoon/examples/datadog/go/dd"} {
		if !slices.Contains(imports, required) {
			t.Fatalf("consumer dependency audit omitted %s", required)
		}
	}
	for _, dependency := range imports {
		if !strings.HasPrefix(dependency, "example.com/cocoon-") && dependency != "github.com/darccio/cocoon/rt" && !strings.HasPrefix(dependency, "github.com/darccio/cocoon/examples/datadog/go/dd") {
			t.Fatalf("consumer unexpectedly depends on %s", dependency)
		}
	}
}

func run(t *testing.T, directory string, environment []string, program string, args ...string) []byte {
	t.Helper()
	t.Logf("%s %s", filepath.Base(program), strings.Join(args, " "))
	output, err := (build.ExecRunner{Environment: environment}).Run(t.Context(), directory, program, args...)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil { // #nosec G703 -- Test fixture paths are contained below t.TempDir.
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path) // #nosec G304 -- Read only explicitly selected test or distribution fixtures.
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func snapshot(t *testing.T, directory string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			relative, relativeErr := filepath.Rel(directory, path)
			if relativeErr != nil {
				return relativeErr
			}
			files[relative] = read(t, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func copyProductionGo(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && entry.Name() == "testdata" {
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			relative, relativeErr := filepath.Rel(source, path)
			if relativeErr != nil {
				return relativeErr
			}
			write(t, filepath.Join(destination, relative), string(read(t, path)))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

const consumerTest = `package consumer_test
import (
    "context"
    "errors"
    "strings"
    "testing"
    example "example.com/cocoon-author/project/go/example"
    "github.com/darccio/cocoon/examples/datadog/go/dd"
    "github.com/darccio/cocoon/rt"
)
func TestInstalledOutput(t *testing.T) {
    library, err := example.Open(example.Options{Instances: 1})
    if err != nil { t.Fatal(err) }
    t.Cleanup(func() { if closeErr := library.Close(); closeErr != nil { t.Error(closeErr) } })
    input := "hello\x00世界\nkey=value"
    if output, callErr := library.Echo(t.Context(), input); callErr != nil || output != input { t.Fatal(output, callErr) }
    ctx, cancel := context.WithCancel(t.Context())
    cancel()
    if _, callErr := library.Echo(ctx, input); !errors.Is(callErr, context.Canceled) { t.Fatal(callErr) }
    if closeErr := library.Close(); closeErr != nil { t.Fatal(closeErr) }
    if _, callErr := library.Echo(t.Context(), input); !errors.Is(callErr, rt.ErrClosed) { t.Fatal(callErr) }
    datadog, openErr := dd.Open(dd.Options{Instances: 1})
    if openErr != nil { t.Fatal(openErr) }
    t.Cleanup(func() { if closeErr := datadog.Close(); closeErr != nil { t.Error(closeErr) } })
    if sql, sqlErr := datadog.ObfuscateSQL(t.Context(), "SELECT * FROM users WHERE id = 424242"); sqlErr != nil || strings.Contains(sql, "424242") || !strings.Contains(sql, "?") { t.Fatal(sql, sqlErr) }
    sketch, sketchErr := datadog.NewSketch()
    if sketchErr != nil { t.Fatal(sketchErr) }
    if addErr := sketch.AddMany([]float64{1, 2, 3}); addErr != nil { t.Fatal(addErr) }
    if count, countErr := sketch.Count(); countErr != nil || count != 3 { t.Fatal(count, countErr) }
    if closeErr := sketch.Close(); closeErr != nil { t.Fatal(closeErr) }
}
`
