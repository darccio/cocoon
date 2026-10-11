package gogen_test

import (
	_ "embed"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"dario.cat/cocoon/internal/gen/gogen"
	"dario.cat/cocoon/internal/manifest"
)

//go:embed testdata/batch/module.go
var batchModuleSource []byte

//go:embed testdata/batch/module_test.go
var batchModuleTests []byte

//go:embed testdata/batch/consumer_test.go
var batchConsumerTests []byte

//go:embed testdata/batch/numeric_test.go
var batchNumericTests []byte

func TestGeneratedCheckedBatchFixture(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(batchFixtureManifest))
	if err != nil {
		t.Fatal(err)
	}
	facade, err := gogen.Facade(m)
	if err != nil {
		t.Fatal(err)
	}
	assertBatchConstructorExclusions(t, facade)
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	moduleFile := fmt.Sprintf("module example.com/batch-fixture\n\ngo 1.26.0\n\nrequire dario.cat/cocoon v0.0.0\n\nreplace dario.cat/cocoon => %q\n", root)
	for name, data := range map[string][]byte{
		"go.mod":           []byte(moduleFile),
		"cocoon_gen.go":    facade,
		"module.go":        batchModuleSource,
		"module_test.go":   batchModuleTests,
		"consumer_test.go": batchConsumerTests,
		"numeric_test.go":  batchNumericTests,
	} {
		if writeErr := os.WriteFile(filepath.Join(directory, name), data, 0o600); writeErr != nil { // #nosec G703 -- Authored fixture names stay under t.TempDir.
			t.Fatal(writeErr)
		}
	}
	goExecutable, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	arguments := []string{"test", "-mod=mod", "-count=1", "-v"}
	cgo := "0"
	// make race sets CGO explicitly; instrument the actual generated consumer.
	if os.Getenv("CGO_ENABLED") == "1" {
		arguments = append(arguments, "-race")
		cgo = "1"
	}
	arguments = append(arguments, "./...")
	command := exec.CommandContext(t.Context(), goExecutable, arguments...) // #nosec G204 -- Resolved Go tool compiles an authored temporary fixture.
	command.Dir = directory
	command.Env = append(os.Environ(), "CGO_ENABLED="+cgo, "GOWORK=off", "GOTOOLCHAIN=local", "GOENV=off", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off", "GOFLAGS=-buildvcs=false")
	for _, cache := range []string{"GOCACHE", "GOMODCACHE"} {
		if os.Getenv(cache) == "" {
			command.Env = append(command.Env, cache+"="+filepath.Join(directory, cache))
		}
	}
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("generated checked-batch fixture: %v\n%s", err, output)
	}
}

func assertBatchConstructorExclusions(t *testing.T, facade []byte) {
	t.Helper()
	syntax, err := parser.ParseFile(token.NewFileSet(), "cocoon_gen.go", facade, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range syntax.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && (function.Name.Name == "StreamNewCommand" || function.Name.Name == "StreamCloseCommand") {
			t.Fatal("constructor or destructor became a checked command")
		}
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			kind, ok := specification.(*ast.TypeSpec)
			if ok && (kind.Name.Name == "StreamNewResult" || kind.Name.Name == "StreamCloseResult") {
				t.Fatal("constructor or destructor received a checked-result tag")
			}
		}
	}
}

const batchFixtureManifest = `
[package]
name="batchfixture"
go_import="example.com/batch-fixture"
rust_crate="batch-fixture"
[limits]
max_input="4KiB"
max_output="4KiB"
max_memory="128KiB"
instances="1"
[[record]]
name="Leaf"
fields=[{name="label",type="string"},{name="data",type="bytes"},{name="flag",type="bool"}]
[[record]]
name="Envelope"
fields=[{name="leaf",type="Leaf"},{name="values",type="[]u64"},{name="score",type="f64"}]
[[resource]]
name="Stream"
checked_batches=true
[[resource.method]]
name="new"
returns="Stream"
[[resource.method]]
name="echo"
params=[{name="value",type="bytes"}]
returns="bytes"
[[resource.method]]
name="text"
params=[{name="value",type="string"}]
returns="string"
[[resource.method]]
name="count"
params=[{name="value",type="u64"}]
returns="u64"
[[resource.method]]
name="flag"
params=[{name="value",type="bool"}]
returns="bool"
[[resource.method]]
name="float"
params=[{name="value",type="f64"}]
returns="f64"
[[resource.method]]
name="numbers"
params=[{name="value",type="[]f32"}]
returns="[]f32"
[[resource.method]]
name="unit"
[[resource.method]]
name="mix"
params=[{name="signed32",type="i32"},{name="unsigned32",type="u32"},{name="signed64",type="i64"},{name="unsigned64",type="u64"},{name="single",type="f32"},{name="double",type="f64"},{name="flag",type="bool"},{name="text",type="string"},{name="data",type="bytes"},{name="signed32s",type="[]i32"},{name="unsigned32s",type="[]u32"},{name="signed64s",type="[]i64"},{name="unsigned64s",type="[]u64"},{name="singles",type="[]f32"},{name="doubles",type="[]f64"},{name="envelope",type="Envelope"}]
returns="Envelope"
[[resource.method]]
name="close"
[[resource]]
name="Idle"
checked_batches=true
[[resource.method]]
name="new"
returns="Idle"
[[resource.method]]
name="close"
[[resource]]
name="Scalar"
checked_batches=true
[[resource.method]]
name="new"
returns="Scalar"
[[resource.method]]
name="count"
params=[{name="value",type="u64"}]
returns="u64"
[[resource.method]]
name="unit"
[[resource.method]]
name="close"
[[resource]]
name="Vectors"
[[resource.method]]
name="new"
params=[{name="first",type="[]u32"},{name="text",type="string"},{name="last",type="[]u64"}]
returns="Vectors"
[[resource.method]]
name="close"
`
