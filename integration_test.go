package cocoon_test

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"dario.cat/cocoon/internal/build"
	"dario.cat/cocoon/internal/gen/gogen"
	"dario.cat/cocoon/internal/harden"
	"dario.cat/cocoon/internal/manifest"

	"dario.cat/cocoon/rt"
	"dario.cat/cocoon/testdata/compute/go/compute"
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
	data, err := os.ReadFile("testdata/compute/cocoon.toml")
	if err != nil {
		t.Fatal(err)
	}
	m, err := manifest.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	wasm, err := os.ReadFile("testdata/compute/go/compute/testdata/module.wasm")
	if err != nil {
		t.Fatal(err)
	}
	module, err := build.Verify(m, wasm)
	if err != nil {
		t.Fatal(err)
	}
	translated, err := os.ReadFile("testdata/compute/go/compute/internal/wasm/module.go")
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
		got, readErr := os.ReadFile("testdata/compute/go/compute/" + path) // #nosec G304 -- The map above contains only fixed golden fixture filenames.
		if generationErr != nil || readErr != nil || !bytes.Equal(want, got) {
			t.Fatalf("stale golden %s: %v %v", path, generationErr, readErr)
		}
	}
}
