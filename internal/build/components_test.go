package build_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"dario.cat/cocoon/internal/build"
	"dario.cat/cocoon/internal/manifest"
)

type componentRunner struct {
	err        error
	components string
}

func (r componentRunner) Run(ctx context.Context, directory, program string, args ...string) ([]byte, error) {
	if program == "rustup" && args[0] == "component" {
		return []byte(r.components), r.err
	}
	return (fakeRunner{}).Run(ctx, directory, program, args...)
}

func TestDoctorInstalledComponents(t *testing.T) {
	t.Parallel()
	pins := manifest.Toolchain{Rust: manifest.RustVersion, Binaryen: manifest.BinaryenVersion, Wasm2Go: manifest.Wasm2GoVersion}
	for _, scenario := range []struct {
		name, components string
		valid            bool
	}{
		{"line-feed", "rust-src\nrust-std-wasm32-unknown-unknown\n", true},
		{"carriage-return", "rust-src\r\nrust-std-wasm32-unknown-unknown\r\n", true},
		{"no-final-newline", "rust-src\nrust-std-wasm32-unknown-unknown", true},
		{"missing-src", "rust-std-wasm32-unknown-unknown\n", false},
		{"missing-target", "rust-src\n", false},
		{"wrong-src", "not-rust-src\nrust-std-wasm32-unknown-unknown\n", false},
		{"wrong-target", "rust-src\nnot-rust-std-wasm32-unknown-unknown\n", false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			versions, err := build.Doctor(t.Context(), componentRunner{components: scenario.components}, ".", pins)
			if scenario.valid {
				if err != nil || versions.Rust != pins.Rust {
					t.Fatal("installed components rejected", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), "requires rust-src") {
				t.Fatal("missing or partial component name accepted", err)
			}
		})
	}
	failure := errors.New("component discovery failed")
	if _, err := build.Doctor(t.Context(), componentRunner{err: failure}, ".", pins); !errors.Is(err, failure) {
		t.Fatal("component discovery error identity lost", err)
	}
}
