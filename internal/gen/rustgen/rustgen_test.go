package rustgen_test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/darccio/cocoon/internal/gen/rustgen"
	"github.com/darccio/cocoon/internal/manifest"
)

func TestDatadogGeneration(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(apiManifest))
	if err != nil {
		t.Fatal(err)
	}
	m.Functions = append(m.Functions,
		manifest.Function{Name: "echo_bytes", Params: []manifest.Param{{Name: "data", Type: "bytes"}}, Returns: "bytes"},
		manifest.Function{Name: "echo_text", Params: []manifest.Param{{Name: "data", Type: "string"}}, Returns: "string"},
	)
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
	for _, forbidden := range []string{"-> ()", "let __value=__state.service.sketch_close", "(|| ->", "Ok(__value.encode()?)", "crate::Shim::default()"} {
		if bytes.Contains(first, []byte(forbidden)) {
			t.Fatalf("generated lint regression: %s", forbidden)
		}
	}
	if !bytes.Contains(first, []byte("let arg0=__arg0;")) || !bytes.Contains(first, []byte("String::from_utf8(__arg0)")) {
		t.Fatal("owned buffer copied during argument decoding")
	}
	if rustfmt, lookErr := exec.LookPath("rustfmt"); lookErr == nil {
		command := exec.CommandContext(t.Context(), rustfmt, "--edition", "2024", "--emit", "stdout") // #nosec G204 -- The executable is a resolved formatter path.
		command.Stdin = bytes.NewReader(first)
		if output, runErr := command.CombinedOutput(); runErr != nil {
			t.Fatalf("Rust syntax: %v: %s", runErr, output)
		}
	}
}

func TestUnitExportsUseTypedEmptyReplies(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(apiManifest))
	if err != nil {
		t.Fatal(err)
	}
	m.Functions = append(m.Functions,
		manifest.Function{Name: "notify"},
		manifest.Function{Name: "try_notify", Fallible: true},
	)
	generated, err := rustgen.Generate(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"notify", "try_notify", "sketch_add_many", "sketch_close"} {
		_, body, found := strings.Cut(string(generated), "fn cocoon_"+name+"(")
		if !found {
			t.Fatalf("missing unit export %s", name)
		}
		body, _, _ = strings.Cut(body, "#[unsafe(no_mangle)]")
		for _, required := range []string{"-> Result<()>", "Ok(())", "cocoon_guest::reply_unit(__result)"} {
			if !strings.Contains(body, required) {
				t.Errorf("%s missing %s", name, required)
			}
		}
		if strings.Contains(body, "Vec::new()") || strings.Contains(body, "Result<Vec<u8>>") {
			t.Errorf("%s constructs a byte-vector reply", name)
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
