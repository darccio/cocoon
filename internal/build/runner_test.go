package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darccio/cocoon/internal/manifest"
)

func TestBoundedDiagnostics(t *testing.T) {
	t.Parallel()
	var buffer boundedBuffer
	first := bytes.Repeat([]byte("a"), 1048570)
	if n, err := buffer.Write(first); err != nil || n != len(first) {
		t.Fatal(n, err)
	}
	if n, err := buffer.Write([]byte("boundary-overflow")); err != nil || n != len("boundary-overflow") {
		t.Fatal(n, err)
	}
	if buffer.Len() != 1048576 || !bytes.Equal(buffer.Bytes()[len(first):], []byte("bounda")) {
		t.Fatal("diagnostics were not truncated at exactly one MiB")
	}
	if n, err := buffer.Write(first); err != nil || n != len(first) || buffer.Len() != 1048576 {
		t.Fatal("a full buffer must keep draining tool output", n, err)
	}
}

func TestRunnerEnvironmentAndMissingExecutable(t *testing.T) {
	t.Parallel()
	runner := ExecRunner{Environment: []string{"GOWORK=off", "GOENV=off", "GOTOOLCHAIN=local"}}
	output, err := runner.Run(t.Context(), t.TempDir(), "go", "env", "GOWORK")
	if err != nil || strings.TrimSpace(string(output)) != "off" {
		t.Fatal("runner did not pass its environment", err)
	}
	if _, err := runner.Run(t.Context(), ".", "cocoon-nonexistent-test-tool"); err == nil {
		t.Fatal("missing executable accepted")
	}
}

func TestExecutableDigest(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "tool")
	content := []byte("version 133, build identity")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if hash, err := executableDigest(path); err != nil || hash != digest(content) {
		t.Fatal("incorrect executable fingerprint", hash, err)
	}
	if _, err := executableDigest(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing executable fingerprint accepted")
	}
}

func TestPinnedDoctorFingerprints(t *testing.T) {
	t.Parallel()
	pins := manifest.Toolchain{Rust: manifest.RustVersion, Binaryen: manifest.BinaryenVersion, Wasm2Go: manifest.Wasm2GoVersion}
	versions, err := Doctor(t.Context(), ExecRunner{}, ".", pins)
	if err != nil {
		t.Skip("the pinned toolchain is needed only for executable-fingerprint integration", err)
	}
	if len(versions.BinaryenDigests) != 3 {
		t.Fatal("doctor omitted executable identities")
	}
	for _, program := range []string{"wasm-as", "wasm-metadce", "wasm-opt"} {
		if len(versions.BinaryenDigests[program]) != 64 {
			t.Fatalf("invalid fingerprint for %s", program)
		}
	}
}
