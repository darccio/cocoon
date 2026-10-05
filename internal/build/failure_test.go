package build

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/darccio/cocoon/internal/manifest"
)

type failureRunner struct {
	fail   error
	runner *pipelineRunner
	stage  string
}

func (r *failureRunner) Run(ctx context.Context, directory, program string, args ...string) ([]byte, error) {
	if slices.Contains(args, "--version") || slices.Contains(args, "-version") {
		return r.runner.Run(ctx, directory, program, args...)
	}
	if (r.stage == "metadce" && program == "wasm-metadce") || (r.stage == "optimize" && program == "wasm-opt") {
		return nil, r.fail
	}
	output, err := r.runner.Run(ctx, directory, program, args...)
	if err != nil {
		return output, err
	}
	if r.stage == "export-graph" && program == "rustup" && slices.Contains(args, "build") {
		return nil, os.Mkdir(filepath.Join(directory, "..", ".cocoon-build", "exports.json"), 0o700)
	}
	index := slices.Index(args, "-o")
	if index < 0 {
		return output, nil
	}
	path := args[index+1]
	if (r.stage == "missing-wasm" && program == "wasm-opt") || (r.stage == "missing-translation" && program == "go") {
		return nil, os.Remove(path)
	}
	if program == "go" {
		switch r.stage {
		case "invalid-translation":
			return nil, os.WriteFile(path, []byte("invalid Go"), 0o600) // #nosec G304 -- Corrupt the fake translator's own temporary output.
		case "helper-drift":
			return nil, os.WriteFile(path, []byte("package wasm\nfunc memory_fill() {}\n"), 0o600) // #nosec G304 -- Exercise fail-closed helper validation on a temporary output.
		}
	}
	return output, nil
}

func TestFailedBuildPreservesEveryPublishedArtifact(t *testing.T) {
	t.Parallel()
	for _, stage := range []string{"metadce", "optimize", "missing-wasm", "missing-translation", "invalid-translation", "helper-drift"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			directory, runner := makeProject(t)
			p := pipeline{runner: runner, sources: fakeSources}
			if _, err := p.run(t.Context(), directory, testManifest(t)); err != nil {
				t.Fatal(err)
			}
			published := publishedArtifacts(t, directory)
			failure := errors.New("tool stage failed")
			p.runner = &failureRunner{runner: &runner, stage: stage, fail: failure}
			p.sources = func(ctx context.Context, m *manifest.Manifest) ([]byte, []byte, error) {
				rust, facade, err := fakeSources(ctx, m)
				return append(rust, "// changed build\n"...), append(facade, "// changed build\n"...), err
			}
			lock, err := p.run(t.Context(), directory, testManifest(t))
			if err == nil || lock != nil {
				t.Fatal("failed build returned success or a lock", err)
			}
			switch stage {
			case "metadce", "optimize":
				if !errors.Is(err, failure) {
					t.Fatal("tool failure identity lost", err)
				}
			case "missing-wasm", "missing-translation":
				if !errors.Is(err, os.ErrNotExist) {
					t.Fatal("missing tool output error lost", err)
				}
			case "invalid-translation", "helper-drift":
				if !strings.Contains(err.Error(), "translated Go") && !strings.Contains(err.Error(), "helper") {
					t.Fatal("invalid translation diagnostics lost", err)
				}
			}
			for path, original := range published {
				data, readErr := os.ReadFile(filepath.Join(directory, path)) // #nosec G304 -- Compare only artifacts produced by this test in t.TempDir.
				if readErr != nil || !bytes.Equal(data, original) {
					t.Fatalf("failed %s build changed %s: %v", stage, path, readErr)
				}
			}
			if _, statErr := os.Stat(filepath.Join(directory, ".cocoon-build", "lock")); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatal("failed build stranded its guard", statErr)
			}
			p.runner = runner
			if _, retryErr := p.run(t.Context(), directory, testManifest(t)); retryErr != nil {
				t.Fatal("failed build prevented retry", retryErr)
			}
		})
	}
}

func TestExportGraphFailureRestoresGlue(t *testing.T) {
	t.Parallel()
	directory, runner := makeProject(t)
	p := pipeline{runner: &failureRunner{runner: &runner, stage: "export-graph"}, sources: fakeSources}
	if lock, err := p.run(t.Context(), directory, testManifest(t)); err == nil || lock != nil {
		t.Fatal("blocked export graph accepted", err)
	}
	for _, path := range []string{"shim/src/cocoon_gen.rs", ".cocoon-build/lock", "cocoon.lock.json"} {
		if _, err := os.Stat(filepath.Join(directory, path)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("failed graph write retained %s: %v", path, err)
		}
	}
}

func TestPublicBuildRejectsInvalidInputsBeforeTools(t *testing.T) {
	t.Parallel()
	if lock, err := Project(t.Context(), t.TempDir(), &manifest.Manifest{}); err == nil || lock != nil {
		t.Fatal("public build accepted an invalid manifest", err)
	}
	if lock, err := Project(t.Context(), t.TempDir(), testManifest(t)); err == nil || lock != nil || !strings.Contains(err.Error(), "no go.mod") {
		t.Fatal("public build accepted a missing module", err)
	}
}

func publishedArtifacts(t *testing.T, directory string) map[string][]byte {
	t.Helper()
	artifacts := make(map[string][]byte)
	for _, path := range []string{
		"shim/src/cocoon_gen.rs", "cocoon.lock.json", "go/test/cocoon_gen.go",
		"go/test/zz_adapter.go", "go/test/zz_contract_test.go", "go/test/zz_wazero_test.go",
		"go/test/internal/wasm/module.go", "go/test/internal/wasm/zz_bulk_test.go", "go/test/testdata/module.wasm",
	} {
		data, err := os.ReadFile(filepath.Join(directory, path)) // #nosec G304 -- Snapshot this test's generated temporary artifacts.
		if err != nil {
			t.Fatal(err)
		}
		artifacts[path] = data
	}
	return artifacts
}
