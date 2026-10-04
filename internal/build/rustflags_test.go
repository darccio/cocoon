package build

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestPathRemapsAndEncodedFlags(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	module := filepath.Join(root, "author with spaces")
	guest := filepath.Join(module, "rust", "cocoon-guest")
	flags, err := pathFlags(module, filepath.Join(root, "rust"), filepath.Join(root, "cargo"), 16777216, []sourcePath{{name: "crates/cocoon-guest", path: guest}})
	if err != nil {
		t.Fatal(err)
	}
	if flags[0] != "-C" || flags[1] != "link-arg=--max-memory=16777216" {
		t.Fatal("remapping lost the bounded-memory linker argument")
	}
	moduleFlag := "--remap-path-prefix=" + module + "=/cocoon/module"
	guestFlag := "--remap-path-prefix=" + guest + "=/cocoon/crates/cocoon-guest"
	if slices.Index(flags, moduleFlag) < 0 || slices.Index(flags, guestFlag) <= slices.Index(flags, moduleFlag) {
		t.Fatal("specific source remap must override the enclosing module")
	}
	if decoded := strings.Split(strings.Join(flags, "\x1f"), "\x1f"); !slices.Equal(decoded, flags) {
		t.Fatal("encoded flags did not preserve paths containing spaces")
	}
	for _, invalid := range []string{"", "relative", filepath.Join(root, "injected") + "\x1fflag"} {
		if _, err := pathFlags(module, invalid, filepath.Join(root, "cargo"), 16777216, nil); err == nil {
			t.Fatal("invalid sysroot accepted")
		}
	}
}

type sysrootRunner struct {
	path string
	fail bool
}

func (r sysrootRunner) Run(context.Context, string, string, ...string) ([]byte, error) {
	if r.fail {
		return nil, fmt.Errorf("sysroot unavailable")
	}
	return []byte(r.path + "\n"), nil
}

func TestCargoEnvironment(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	t.Setenv("CARGO_HOME", "relative cache")
	environment, err := cargoEnvironment(t.Context(), sysrootRunner{path: filepath.Join(root, "sysroot")}, project, root, "1.97.0", 16777216, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(environment, "RUSTC_BOOTSTRAP=1") || !slices.Contains(environment, "CARGO_TARGET_DIR="+filepath.Join(project, ".cocoon-build", "target")) {
		t.Fatal("missing isolated build environment")
	}
	if !strings.Contains(environment[2], "--remap-path-prefix="+filepath.Join(project, "shim", "relative cache")+"=/cocoon/cargo") {
		t.Fatal("relative Cargo home was not resolved against the Cargo working directory")
	}
	t.Setenv("CARGO_HOME", "")
	if _, err := cargoEnvironment(t.Context(), sysrootRunner{path: root}, project, root, "1.97.0", 16777216, nil); err != nil {
		t.Fatal("default Cargo home unavailable", err)
	}
	if _, err := cargoEnvironment(t.Context(), sysrootRunner{fail: true}, project, root, "1.97.0", 16777216, nil); err == nil {
		t.Fatal("missing sysroot accepted")
	}
}
