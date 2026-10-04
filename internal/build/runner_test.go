package build

import (
	"bytes"
	"strings"
	"testing"
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
