package gogen_test

import (
	"bytes"
	"testing"

	"github.com/darccio/cocoon/internal/gen/gogen"
	"github.com/darccio/cocoon/internal/manifest"
)

func TestFacadeAndAdapter(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(apiManifest))
	if err != nil {
		t.Fatal(err)
	}
	first, err := gogen.Facade(m)
	if err != nil {
		t.Fatal(err)
	}
	second, err := gogen.Facade(m)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("nondeterministic facade")
	}
	for _, expected := range []string{"ObfuscateSQL", "NewSketch", "AddMany", "rt.NewResource", "l.life.Close", "rt.InputSize", "PutString(_arg0)", "utf8.ValidString(_arg0)", "_call.ResultView(", "append([]byte(nil), data...)"} {
		if !bytes.Contains(first, []byte(expected)) {
			t.Fatalf("missing %s", expected)
		}
	}
	countStart := bytes.Index(first, []byte("func (r *Sketch) Count"))
	countEnd := bytes.Index(first[countStart:], []byte("func (r *Sketch) Encode"))
	if countStart < 0 || countEnd < 0 || bytes.Contains(first[countStart:countStart+countEnd], []byte("PrepareInput")) {
		t.Fatal("scalar-only operation reserves unused input")
	}
	for _, imports := range []bool{false, true} {
		adapter, adapterErr := gogen.Adapter(m, imports)
		if adapterErr != nil || !bytes.Contains(adapter, []byte("Xrandom_get")) {
			t.Fatalf("adapter: %v", adapterErr)
		}
	}
}

const apiManifest = `
[package]
name="dd"
go_import="example.com/dd"
rust_crate="dd-shim"
[limits]
max_input="64KiB"
max_output="64KiB"
max_memory="16MiB"
instances="1"
[capabilities]
log=true
random=true
clock=true
[[record]]
name="Config"
fields=[{name="label",type="string"},{name="value",type="f64"},{name="many",type="[]u32"}]
[[func]]
name="obfuscate_sql"
params=[{name="query",type="string"},{name="configuration",type="Config"},{name="flag",type="bool"},{name="signed",type="i64"},{name="unsigned",type="u64"}]
returns="string"
fallible=true
[[func]]
name="sum"
params=[{name="values",type="[]f32"}]
returns="f32"
[[func]]
name="set"
params=[{name="value",type="u32"}]
[[resource]]
name="Sketch"
copy="shared"
[[resource.method]]
name="new"
returns="Sketch"
[[resource.method]]
name="add_many"
params=[{name="values",type="[]f64"}]
[[resource.method]]
name="count"
returns="f64"
[[resource.method]]
name="encode"
returns="bytes"
[[resource.method]]
name="close"
`
