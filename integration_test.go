package cocoon_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"dario.cat/cocoon/internal/build"
	"dario.cat/cocoon/internal/gen/gogen"
	"dario.cat/cocoon/internal/harden"
	"dario.cat/cocoon/internal/manifest"

	"dario.cat/cocoon/rt"
	"dario.cat/cocoon/testdata/compute/go/compute"
	"dario.cat/cocoon/testdata/streaming/go/streaming"
)

// Importing the checked-in fixture makes ordinary root tests exercise the full
// safe Rust -> Wasm -> translated Go -> typed facade path without build tools.
func TestCheckedInComputeConsumer(t *testing.T) {
	t.Parallel()
	library, openErr := compute.Open(compute.Options{Instances: 1})
	if openErr != nil {
		t.Fatal(openErr)
	}
	t.Cleanup(func() {
		if err := library.Close(); err != nil {
			t.Error(err)
		}
	})
	if got, err := library.Join(t.Context(), "pure ", "Go"); err != nil || got != "pure Go" {
		t.Fatal(got, err)
	}
	counter, counterErr := library.NewCounter(42)
	if counterErr != nil {
		t.Fatal(counterErr)
	}
	if value, err := counter.Add(1); err != nil || value != 43 {
		t.Fatal(value, err)
	}
	if err := library.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := library.Echo(t.Context(), "closed"); !errors.Is(err, rt.ErrClosed) {
		t.Fatal(err)
	}
}

func TestCheckedInGeneratorGoldens(t *testing.T) {
	t.Parallel()
	for _, proof := range []struct{ project, pkg string }{
		{"testdata/compute", "compute"},
		{"examples/datadog", "dd"},
		{"testdata/streaming", "streaming"},
	} {
		t.Run(proof.pkg, func(t *testing.T) {
			t.Parallel()
			assertGeneratorGoldens(t, proof.project, proof.pkg)
		})
	}
}

func assertGeneratorGoldens(t *testing.T, project, pkg string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(project, "cocoon.toml")) // #nosec G304 -- Callers use only the three fixed checked-in proof paths.
	if err != nil {
		t.Fatal(err)
	}
	m, err := manifest.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(project, "go", pkg)
	wasm, err := os.ReadFile(filepath.Join(base, "testdata", "module.wasm")) // #nosec G304 -- Fixed proof paths above.
	if err != nil {
		t.Fatal(err)
	}
	module, err := build.Verify(m, wasm)
	if err != nil {
		t.Fatal(err)
	}
	translated, err := os.ReadFile(filepath.Join(base, "internal", "wasm", "module.go")) // #nosec G304 -- Fixed proof paths above.
	if err != nil {
		t.Fatal(err)
	}
	helpers, err := harden.Required(translated)
	if err != nil {
		t.Fatal(err)
	}
	for path, generate := range map[string]func() ([]byte, error){
		"cocoon_gen.go":                 func() ([]byte, error) { return gogen.Facade(m) },
		"zz_adapter.go":                 func() ([]byte, error) { return gogen.Adapter(m, len(module.Imports) != 0) },
		"zz_contract_test.go":           func() ([]byte, error) { return gogen.Contracts(m) },
		"zz_wazero_test.go":             func() ([]byte, error) { return gogen.WazeroTests(m) },
		"internal/wasm/zz_bulk_test.go": func() ([]byte, error) { return gogen.BulkTests(helpers) },
	} {
		want, generationErr := generate()
		got, readErr := os.ReadFile(filepath.Join(base, path)) // #nosec G304 -- The map and callers above contain only fixed proof filenames.
		if generationErr != nil || readErr != nil || !bytes.Equal(want, got) {
			t.Fatalf("stale golden %s: %v %v", path, generationErr, readErr)
		}
	}
}

func TestCheckedInStreamingConsumer(t *testing.T) {
	t.Parallel()
	library, err := streaming.Open(streaming.Options{ResourceInstances: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := library.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})
	stream, err := library.NewStream()
	if err != nil {
		t.Fatal(err)
	}
	results, err := stream.Batch(t.Context(), []streaming.StreamCommand{
		streaming.StreamWriteCommand([]byte("prefix</hea")),
		streaming.StreamWriteCommand([]byte("d>suffix")),
		streaming.StreamEndCommand(),
	}, rt.BatchLimits{})
	if err != nil || len(results) != 3 {
		t.Fatal(results, err)
	}
	tail, ok := results[2].(streaming.StreamEndResult)
	if !ok || !bytes.Equal(tail.Value, []byte("prefix<cocoon></head>suffix")) {
		t.Fatal(results)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
}
