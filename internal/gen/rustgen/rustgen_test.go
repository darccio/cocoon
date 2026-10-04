package rustgen_test

import (
	"bytes"
	"os/exec"
	"testing"

	"cocoon.dev/cocoon/internal/gen/rustgen"
	"cocoon.dev/cocoon/internal/manifest"
)

func TestDatadogGeneration(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(apiManifest))
	if err != nil {
		t.Fatal(err)
	}
	first, err := rustgen.Generate(m)
	if err != nil {
		t.Fatal(err)
	}
	second, err := rustgen.Generate(m)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("nondeterministic Rust generation")
	}
	if !bytes.Contains(first, []byte("fn sketch_add_many")) || !bytes.Contains(first, []byte("cocoon_schema_hash")) {
		t.Fatal("missing exports")
	}
	if rustfmt, lookErr := exec.LookPath("rustfmt"); lookErr == nil {
		command := exec.CommandContext(t.Context(), rustfmt, "--edition", "2024", "--emit", "stdout") // #nosec G204 -- The executable is a resolved formatter path.
		command.Stdin = bytes.NewReader(first)
		if output, runErr := command.CombinedOutput(); runErr != nil {
			t.Fatalf("Rust syntax: %v: %s", runErr, output)
		}
	}
}

const apiManifest = `
[package]
name="example"
go_import="example.com/example"
rust_crate="example-shim"
[limits]
max_input="64KiB"
max_output="64KiB"
max_memory="16MiB"
instances="1"
[[record]]
name="Config"
fields=[{name="text",type="string"},{name="data",type="bytes"},{name="ok",type="bool"},{name="number",type="i32"},{name="values",type="[]f64"}]
[[func]]
name="configure"
params=[{name="resource",type="Config"},{name="resource_ptr",type="u32"},{name="service",type="bool"}]
returns="Config"
fallible=true
[[resource]]
name="Sketch"
[[resource.method]]
name="new"
returns="Sketch"
[[resource.method]]
name="add_many"
params=[{name="values",type="[]f64"}]
fallible=true
[[resource.method]]
name="count"
returns="f64"
[[resource.method]]
name="close"
`
