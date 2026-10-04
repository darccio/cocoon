package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cocoon.dev/cocoon/internal/cli"
)

func TestHelpAndInvalidCommands(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	if err := cli.Run(t.Context(), nil, &output); err != nil || !strings.Contains(output.String(), "cocoon") {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"unknown"}, {"gen", "--bad"}, {"build", "--manifest", "/nonexistent/cocoon.toml"}, {"init", "--name", "../../escape"}} {
		if err := cli.Run(t.Context(), args, &output); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestManifestAndArgumentErrors(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "cocoon.toml")
	data := []byte("[package]\nname=\"test\"\ngo_import=\"example.com/test\"\nrust_crate=\"test-shim\"\n[limits]\nmax_input=\"64KiB\"\nmax_output=\"64KiB\"\nmax_memory=\"16MiB\"\ninstances=\"1\"\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"gen", "build", "doctor", "verify"} {
		var output bytes.Buffer
		args := []string{command, "--manifest", path}
		if command != "verify" {
			args = append(args, "unexpected")
		}
		if err := cli.Run(t.Context(), args, &output); err == nil {
			t.Fatalf("accepted invalid %s arguments", command)
		}
		if err := cli.Run(t.Context(), []string{command, "--help"}, &output); err != nil {
			t.Fatal(err)
		}
	}
	invalid := filepath.Join(directory, "invalid.wasm")
	if err := os.WriteFile(invalid, []byte("invalid wasm"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	for _, file := range []string{invalid, filepath.Join(directory, "missing.wasm")} {
		if err := cli.Run(t.Context(), []string{"verify", "--manifest", path, file}, &output); err == nil {
			t.Fatal("invalid wasm accepted")
		}
	}
	if err := os.WriteFile(path, []byte("malformed TOML ="), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cli.Run(t.Context(), []string{"gen", "--manifest", path}, &output); err == nil {
		t.Fatal("malformed manifest accepted")
	}
}

func TestInitPreflight(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	guest := filepath.Join(directory, "rust", "cocoon-guest")
	if err := os.MkdirAll(guest, 0o750); err != nil {
		t.Fatal(err)
	}
	for path, data := range map[string]string{
		"go.mod":                       "// leading comment\nmodule example.com/consumer\ngo 1.26\n",
		"rust/cocoon-guest/Cargo.toml": "[package]\nname=\"cocoon-guest\"\n",
		"cocoon.toml":                  "authored",
	} {
		if err := os.WriteFile(filepath.Join(directory, path), []byte(data), 0o600); err != nil { // #nosec G304 -- Explicit fixture paths below t.TempDir.
			t.Fatal(err)
		}
	}
	var output bytes.Buffer
	for _, args := range [][]string{
		{"init", directory},
		{"init", "--name", "../escape", directory},
		{"init", "--guest", filepath.Join(directory, "missing"), directory},
		{"init", directory, "extra"},
		{"init", "--bad"},
	} {
		if err := cli.Run(t.Context(), args, &output); err == nil {
			t.Fatalf("accepted invalid initialization %v", args)
		}
	}
	if err := cli.Run(t.Context(), []string{"init", "--help"}, &output); err != nil {
		t.Fatal(err)
	}
	preserved, err := os.ReadFile(filepath.Join(directory, "cocoon.toml")) // #nosec G304 -- Read the known authored fixture below t.TempDir.
	if err != nil || string(preserved) != "authored" {
		t.Fatal("preflight modified authored manifest")
	}
}
