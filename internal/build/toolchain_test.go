package build_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/darccio/cocoon/internal/build"
	"github.com/darccio/cocoon/internal/manifest"
)

type fakeRunner struct{ fail string }

func (f fakeRunner) Run(_ context.Context, _, program string, args ...string) ([]byte, error) {
	if program == f.fail {
		return nil, fmt.Errorf("tool unavailable")
	}
	switch program {
	case "go":
		return []byte("wasm2go v0.4.16\n"), nil
	case "rustup":
		if args[0] == "component" {
			return []byte("rust-src\nrust-std-wasm32-unknown-unknown\n"), nil
		}
		return []byte("rustc 1.97.0 (revision date)\n"), nil
	default:
		return []byte(program + " version 133\n"), nil
	}
}

func TestDoctor(t *testing.T) {
	t.Parallel()
	pins := manifest.Toolchain{Rust: manifest.RustVersion, Binaryen: manifest.BinaryenVersion, Wasm2Go: manifest.Wasm2GoVersion}
	versions, err := build.Doctor(t.Context(), fakeRunner{}, ".", pins)
	if err != nil || versions.Rust != manifest.RustVersion || len(versions.CompilerMetadata) != 64 {
		t.Fatal(err)
	}
	for _, tool := range []string{"go", "rustup", "wasm-opt", "wasm-metadce", "wasm-as"} {
		if _, err := build.Doctor(t.Context(), fakeRunner{fail: tool}, ".", pins); err == nil {
			t.Fatalf("missing %s accepted", tool)
		}
	}
	pins.Rust = "1.94.0"
	if _, err := build.Doctor(t.Context(), fakeRunner{}, ".", pins); err == nil {
		t.Fatal("wrong version accepted")
	}
}

func TestRunnerCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := (build.ExecRunner{}).Run(ctx, ".", "go", "version"); err == nil || !strings.Contains(err.Error(), "go") {
		t.Fatal("cancellation ignored")
	}
}
