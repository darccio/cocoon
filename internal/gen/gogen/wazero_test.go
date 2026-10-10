package gogen_test

import (
	"bytes"
	"testing"

	"dario.cat/cocoon/internal/gen/gogen"
	"dario.cat/cocoon/internal/manifest"
)

func TestDifferentialGeneration(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(apiManifest))
	if err != nil {
		t.Fatal(err)
	}
	// Exercise raw-bit conversions for every scalar type as well as buffers.
	m.Functions = append(m.Functions, manifest.Function{Name: "scalars", Params: []manifest.Param{
		{Name: "a", Type: "i32"}, {Name: "b", Type: "f64"}, {Name: "c", Type: "f32"},
	}})
	first, err := gogen.WazeroTests(m)
	if err != nil {
		t.Fatal(err)
	}
	second, err := gogen.WazeroTests(m)
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("nondeterministic differential adapter")
	}
	for _, expected := range []string{"go:embed testdata/module.wasm", "Xcocoon_sketch_close", "Float32frombits", "Float64frombits", "FuzzGeneratedDifferential", "encodeConfig"} {
		if !bytes.Contains(first, []byte(expected)) {
			t.Fatalf("missing %s", expected)
		}
	}
	// An expected application and argument error can have identical text; status
	// equality must therefore be checked before converting or returning errors.
	statusCheck := bytes.Index(first, []byte("if referenceErr == nil && status != referenceStatus"))
	statusConversion := bytes.Index(first, []byte("referenceErr = rt.FromStatus"))
	errorComparison := bytes.Index(first, []byte("if err.Error() != referenceErr.Error()"))
	if statusCheck < 0 || statusConversion <= statusCheck || errorComparison <= statusConversion {
		t.Fatal("expected-error differential status comparison is missing or too late")
	}
	m.Package.Name = "bad/name"
	if _, err := gogen.WazeroTests(m); err == nil {
		t.Fatal("invalid manifest accepted")
	}
}
