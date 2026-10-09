package gogen_test

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"dario.cat/cocoon/internal/gen/gogen"
	"dario.cat/cocoon/internal/manifest"
)

//go:embed testdata/unit/module.go
var unitModuleSource []byte

//go:embed testdata/unit/module_test.go
var unitModuleTests []byte

// Exercise the generated constructor's real destructor closure, not a separately
// authored runtime callback or a textual assertion about the generated source.
func TestGeneratedUnitReplyFixture(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(unitFixtureManifest))
	if err != nil {
		t.Fatal(err)
	}
	facade, err := gogen.Facade(m)
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	moduleFile := fmt.Sprintf("module example.com/unit-fixture\n\ngo 1.26.0\n\nrequire dario.cat/cocoon v0.0.0\n\nreplace dario.cat/cocoon => %q\n", root)
	for name, data := range map[string][]byte{
		"go.mod":         []byte(moduleFile),
		"cocoon_gen.go":  facade,
		"module.go":      unitModuleSource,
		"module_test.go": unitModuleTests,
	} {
		if writeErr := os.WriteFile(filepath.Join(directory, name), data, 0o600); writeErr != nil { // #nosec G703 -- Fixed fixture names are written only below t.TempDir.
			t.Fatal(writeErr)
		}
	}
	goExecutable, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(t.Context(), goExecutable, "test", "-mod=mod", "-count=1", "-v", "./...") // #nosec G204 -- Invoke the resolved Go tool on an authored temporary fixture.
	command.Dir = directory
	command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off", "GOTOOLCHAIN=local", "GOENV=off", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off", "GOFLAGS=-buildvcs=false")
	for _, cache := range []string{"GOCACHE", "GOMODCACHE"} {
		if os.Getenv(cache) == "" {
			command.Env = append(command.Env, cache+"="+filepath.Join(directory, cache))
		}
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("generated unit-reply fixture: %v\n%s", err, output)
	}
}

const unitFixtureManifest = `
[package]
name="unitfixture"
go_import="example.com/unit-fixture"
rust_crate="unit-fixture"
[limits]
max_input="32B"
max_output="32B"
max_memory="64KiB"
instances="1"
[[func]]
name="unit"
[[resource]]
name="Item"
copy="shared"
[[resource.method]]
name="new"
returns="Item"
[[resource.method]]
name="add"
[[resource.method]]
name="close"
`
