package build

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCompilerMetadataRelocation(t *testing.T) {
	t.Parallel()
	makeArguments := func(root, cargoHash string) []string {
		return []string{"--crate-name", "guest", filepath.Join(root, "guest", "src", "lib.rs"), "--target", "wasm32-unknown-unknown", "-C", "metadata=" + cargoHash, "-C", "extra-filename=-" + cargoHash, "--out-dir", filepath.Join(root, "target"), "-L", "dependency=" + filepath.Join(root, "target"), "--extern", "dependency=" + filepath.Join(root, "target", "libdependency-"+cargoHash+".rlib"), "--cfg", `feature="resource"`, "--remap-path-prefix=" + root + "=/cocoon/module"}
	}
	firstRoot, secondRoot := filepath.Join(t.TempDir(), "first"), filepath.Join(t.TempDir(), "relocated = with spaces")
	first, err := compilerArguments(makeArguments(firstRoot, "111"), "guest", "0.1.0", filepath.Join(firstRoot, "guest"), "locked-graph")
	if err != nil {
		t.Fatal(err)
	}
	second, err := compilerArguments(makeArguments(secondRoot, "222"), "guest", "0.1.0", filepath.Join(secondRoot, "guest"), "locked-graph")
	if err != nil {
		t.Fatal(err)
	}
	if first[len(first)-1] != second[len(second)-1] {
		t.Fatal("compiler metadata depends on build locations or Cargo's absolute-path hash")
	}
	if !slices.Contains(first, "extra-filename=-111") || !slices.Contains(second, "extra-filename=-222") {
		t.Fatal("normalization must preserve Cargo's expected output filenames")
	}
	diagnostics := append(makeArguments(firstRoot, "111"), "--diagnostic-width=120", "--color", "always", "--json=artifacts", "--error-format=json")
	withDiagnostics, err := compilerArguments(diagnostics, "guest", "0.1.0", filepath.Join(firstRoot, "guest"), "locked-graph")
	if err != nil || withDiagnostics[len(withDiagnostics)-1] != first[len(first)-1] || !slices.Contains(withDiagnostics, "always") {
		t.Fatal("diagnostic presentation changed metadata or was not preserved", err)
	}
	for _, variant := range []struct {
		name, version, seed string
		additional          []string
	}{
		{"other", "0.1.0", "locked-graph", nil},
		{"guest", "0.2.0", "locked-graph", nil},
		{"guest", "0.1.0", "different-lock", nil},
		{"guest", "0.1.0", "locked-graph", []string{"--cfg", `feature="other"`}},
		{"guest", "0.1.0", "locked-graph", []string{"-Z", "force-unstable-if-unmarked"}},
	} {
		arguments := append(makeArguments(firstRoot, "111"), variant.additional...)
		changed, changeErr := compilerArguments(arguments, variant.name, variant.version, filepath.Join(firstRoot, "guest"), variant.seed)
		if changeErr != nil || changed[len(changed)-1] == first[len(first)-1] {
			t.Fatal("distinct package, dependency graph, features, or standard-library role lost its identity", changeErr)
		}
	}
}

func TestCompilerMetadataProbesAndFailures(t *testing.T) {
	t.Parallel()
	for _, arguments := range [][]string{{"-vV"}, {"--target", "x86_64-unknown-linux-gnu", "-Cmetadata=host"}, {"wasm32-unknown-unknown.rs", "--target"}} {
		actual, err := compilerArguments(arguments, "guest", "0.1.0", "unmapped", "")
		if err != nil || !slices.Equal(actual, arguments) {
			t.Fatal("compiler probe or host compilation was rewritten", err)
		}
	}
	root := t.TempDir()
	base := []string{"--target", "wasm32-unknown-unknown", "-Cmetadata=old", "--remap-path-prefix=" + root + "=/cocoon/module"}
	for _, missing := range []string{"--extern", "-L", "--out-dir", "--color"} {
		if _, err := compilerArguments(append(slices.Clone(base), missing), "guest", "1", root, "seed"); err == nil {
			t.Fatal("incomplete compiler arguments accepted")
		}
	}
	for _, arguments := range [][]string{append(slices.Clone(base), "--remap-path-prefix=broken"), append(slices.Clone(base), "--remap-path-prefix==empty")} {
		if _, err := compilerArguments(arguments, "guest", "1", root, "seed"); err == nil {
			t.Fatal("invalid path remapping accepted")
		}
	}
	if _, err := compilerArguments(base, "guest", "1", root, ""); err == nil {
		t.Fatal("missing build identity accepted")
	}
	if _, err := compilerArguments(base, "guest", "1", filepath.Dir(root), "seed"); err == nil {
		t.Fatal("unmapped package root accepted")
	}
	actual, err := compilerArguments(append(slices.Clone(base), "-Cextra-filename=-old"), "guest", "1", root, "seed")
	if err != nil || !slices.Contains(actual, "-Cextra-filename=-old") || slices.Contains(actual, "-Cmetadata=old") || !strings.HasPrefix(actual[len(actual)-1], "-Cmetadata=") {
		t.Fatal("joined compiler options were not handled", err)
	}
	if _, err := compilerArguments([]string{"--target=wasm32-unknown-unknown", "--remap-path-prefix=" + root + "=/cocoon/module"}, "guest", "1", root, "seed"); err != nil {
		t.Fatal("joined target option rejected", err)
	}
}

func TestCompilerPathBoundaries(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	prefixes := []sourcePath{{path: root, name: "/cocoon/module"}, {path: filepath.Join(root, "guest"), name: "/cocoon/crates/guest"}}
	if actual, ok := compilerPath(filepath.Join(root, "guest", "src", "lib.rs"), prefixes); !ok || actual != "/cocoon/crates/guest/src/lib.rs" {
		t.Fatal("more specific source identity did not win", actual)
	}
	if _, ok := compilerPath(root+"-different", prefixes); ok {
		t.Fatal("remapping crossed a path component boundary")
	}
}

func TestRustcWrapperExecution(t *testing.T) {
	t.Setenv("CARGO_PKG_NAME", "")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := RustcWrapper(t.Context(), []string{executable, "-test.run=^TestCompilerPathBoundaries$"}, &stdout, &stderr); err != nil {
		t.Fatal(err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "PASS") {
		t.Fatal("compiler output was not streamed")
	}
	for _, arguments := range [][]string{nil, {filepath.Join(t.TempDir(), "missing")}} {
		if err := RustcWrapper(t.Context(), arguments, &stdout, &stderr); err == nil {
			t.Fatal("missing compiler accepted")
		}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := RustcWrapper(ctx, []string{executable}, &stdout, &stderr); err == nil {
		t.Fatal("canceled compiler launched")
	}
	t.Setenv("CARGO_PKG_NAME", "guest")
	t.Setenv("COCOON_METADATA_SEED", "")
	if err := RustcWrapper(t.Context(), []string{executable, "--target", "wasm32-unknown-unknown"}, &stdout, &stderr); err == nil {
		t.Fatal("invalid target compiler arguments launched")
	}
}
