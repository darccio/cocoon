package build

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"
)

type graphRunner struct {
	pipelineRunner
	graph []byte
}

func (r *graphRunner) Run(ctx context.Context, directory, program string, args ...string) ([]byte, error) {
	if program == "rustup" && slices.Contains(args, "metadata") {
		if slices.Contains(args, "--no-deps") {
			return nil, context.Canceled
		}
		return r.graph, nil
	}
	return r.pipelineRunner.Run(ctx, directory, program, args...)
}

func TestSourceGraphRemapsTransitiveCrates(t *testing.T) {
	t.Parallel()
	directory, runner := makeProject(t)
	m := testManifest(t)
	transitive := filepath.Join(directory, "transitive", "Cargo.toml")
	packages := []map[string]any{
		{"name": m.Package.RustCrate, "manifest_path": filepath.Join(directory, "shim", "Cargo.toml"), "dependencies": []map[string]string{{"name": "cocoon-guest", "path": runner.guest}}},
		{"name": "cocoon-guest", "manifest_path": filepath.Join(runner.guest, "Cargo.toml")},
		{"name": "transitive", "manifest_path": transitive},
		{"name": "registry", "manifest_path": filepath.Join(directory, "registry", "Cargo.toml"), "source": "registry+https://github.com/rust-lang/crates.io-index"},
	}
	graph, err := json.Marshal(map[string]any{"packages": packages})
	if err != nil {
		t.Fatal(err)
	}
	identities, paths, err := sourceIdentities(t.Context(), &graphRunner{pipelineRunner: runner, graph: graph}, directory, m)
	if err != nil || len(identities) != 1 || len(paths) != 3 {
		t.Fatal("complete source graph was not mapped", paths, err)
	}
	if !slices.ContainsFunc(paths, func(source sourcePath) bool {
		return source.name == "crates/transitive" && source.path == filepath.Dir(transitive)
	}) {
		t.Fatal("transitive local package omitted")
	}
	packages = append(packages, map[string]any{"name": "transitive", "manifest_path": filepath.Join(directory, "different", "Cargo.toml")})
	graph, err = json.Marshal(map[string]any{"packages": packages})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := sourceIdentities(t.Context(), &graphRunner{pipelineRunner: runner, graph: graph}, directory, m); err == nil {
		t.Fatal("ambiguous local source identity accepted")
	}
}
