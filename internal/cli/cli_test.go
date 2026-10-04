package cli_test

import (
	"bytes"
	"os"
	"os/exec"
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

func TestInitAndGenWithPinnedFormatter(t *testing.T) {
	t.Parallel()
	rustup, lookErr := exec.LookPath("rustup")
	if lookErr != nil {
		t.Skip("Rust is needed only for source-generation integration")
	}
	probe := exec.CommandContext(t.Context(), rustup, "run", "1.97.0", "rustfmt", "--version") // #nosec G204 -- Invoke a resolved tool with fixed probe arguments.
	if probeErr := probe.Run(); probeErr != nil {
		t.Skip("pinned Rust formatter unavailable")
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "go.mod"), []byte("// leading comment\nmodule example.com/consumer\ngo 1.26\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	guest, guestErr := filepath.Abs("../../rust/cocoon-guest")
	if guestErr != nil {
		t.Fatal(guestErr)
	}
	project := filepath.Join(directory, "project")
	var output bytes.Buffer
	if err := cli.Run(t.Context(), []string{"init", "--guest", guest, project}, &output); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(project, "cocoon.toml")
	data, err := os.ReadFile(manifestPath) // #nosec G304 -- Read the initialized fixture below t.TempDir.
	if err != nil || !bytes.Contains(data, []byte("example.com/consumer/project/go/example")) {
		t.Fatal("import inference", err)
	}
	if err := cli.Run(t.Context(), []string{"gen", "--manifest", manifestPath}, &output); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, "shim", "Cargo.lock")); err != nil {
		t.Fatal(err)
	}
	if err := cli.Run(t.Context(), []string{"init", "--guest", guest, project}, &output); err == nil {
		t.Fatal("existing initialized files overwritten")
	}
}

func TestManifestAndArgumentErrors(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, "cocoon.toml")
	data := []byte("[package]\nname=\"test\"\ngo_import=\"example.com/test\"\nrust_crate=\"test-shim\"\n[limits]\nmax_input=\"64KiB\"\nmax_output=\"64KiB\"\nmax_memory=\"16MiB\"\ninstances=\"1\"\n")
	data = append(data, []byte("[[func]]\nname=\"echo\"\nparams=[{name=\"input\",type=\"string\"}]\nreturns=\"string\"\n")...)
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
