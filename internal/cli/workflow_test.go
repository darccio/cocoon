package cli_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dario.cat/cocoon/internal/cli"
)

func TestCommandWorkflowsWithFixtureTools(t *testing.T) {
	directory, manifestPath := cliProject(t)
	work := t.TempDir()
	for _, variable := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(variable, work)
	}
	for _, command := range []string{"doctor", "build", "verify"} {
		t.Run(command, func(t *testing.T) {
			args := []string{command, "--manifest", manifestPath}
			if command == "verify" {
				args = append(args, filepath.Join(directory, "fixture.wasm"))
			}
			var output bytes.Buffer
			if err := cli.Run(t.Context(), args, &output); err != nil {
				t.Fatal(err)
			}
			if output.Len() == 0 {
				t.Fatal("successful command did not report its result")
			}
			if err := cli.Run(t.Context(), args, failedOutput{}); !errors.Is(err, io.ErrClosedPipe) {
				t.Fatal("output error lost", err)
			}
		})
	}
	for _, path := range []string{"cocoon.lock.json", "go/compute/cocoon_gen.go", "go/compute/internal/wasm/module.go", "go/compute/testdata/module.wasm"} {
		if _, err := os.Stat(filepath.Join(directory, path)); err != nil {
			t.Fatalf("successful build omitted %s: %v", path, err)
		}
	}
	entries, err := os.ReadDir(work)
	if err != nil || len(entries) != 0 {
		t.Fatal("verification retained its temporary artifacts", err)
	}
	assertNoCLIFile(t, filepath.Join(directory, ".cocoon-build", "lock"))
	if err := cli.Run(t.Context(), []string{"help"}, failedOutput{}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("help output error lost", err)
	}
}

func TestCommandToolFailuresAndRetry(t *testing.T) {
	directory, manifestPath := cliProject(t)
	for _, scenario := range []struct{ command, failure string }{
		{"doctor", "wasm-as"}, {"build", "build"}, {"gen", "rustfmt"}, {"verify", "wasm-opt"},
	} {
		t.Run(scenario.command, func(t *testing.T) {
			t.Setenv("COCOON_CLI_TEST_FAILURE", scenario.failure)
			args := []string{scenario.command, "--manifest", manifestPath}
			if scenario.command == "verify" {
				args = append(args, filepath.Join(directory, "fixture.wasm"))
			}
			err := cli.Run(t.Context(), args, io.Discard)
			if err == nil || !strings.Contains(err.Error(), "fixture failure: "+scenario.failure) {
				t.Fatal("tool diagnostics lost", err)
			}
			assertNoCLIFile(t, filepath.Join(directory, ".cocoon-build", "lock"))
			t.Setenv("COCOON_CLI_TEST_FAILURE", "")
			if err := cli.Run(t.Context(), args, io.Discard); err != nil {
				t.Fatal("tool failure prevented retry", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := cli.Run(ctx, []string{"build", "--manifest", manifestPath}, io.Discard); !errors.Is(err, context.Canceled) {
		t.Fatal("build cancellation lost", err)
	}
}

func TestDoctorWithoutManifestAndWithoutModule(t *testing.T) {
	directory, _ := cliProject(t)
	if err := cli.Run(t.Context(), []string{"doctor", "--manifest", filepath.Join(directory, "missing.toml")}, io.Discard); err != nil {
		t.Fatal("doctor requires an existing manifest", err)
	}
	if err := cli.Run(t.Context(), []string{"doctor", "--manifest", filepath.Join(t.TempDir(), "missing.toml")}, io.Discard); err == nil || !strings.Contains(err.Error(), "no go.mod") {
		t.Fatal("doctor accepted a missing module", err)
	}
	if err := cli.Run(t.Context(), []string{"doctor", "--manifest", directory}, io.Discard); err == nil {
		t.Fatal("doctor ignored a non-missing manifest read error")
	}
}

func TestVerifyRejectsUnavailableTemporaryDirectory(t *testing.T) {
	directory, manifestPath := cliProject(t)
	missing := filepath.Join(t.TempDir(), "missing")
	for _, variable := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(variable, missing)
	}
	if err := cli.Run(t.Context(), []string{"verify", "--manifest", manifestPath, filepath.Join(directory, "fixture.wasm")}, io.Discard); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("verification ignored temporary directory failure", err)
	}
}

func TestInitReportsGenerationAndLockfileFailures(t *testing.T) {
	directory, _ := cliProject(t)
	for _, failure := range []string{"rustfmt", "generate-lockfile"} {
		t.Run(failure, func(t *testing.T) {
			t.Setenv("COCOON_CLI_TEST_FAILURE", failure)
			project := filepath.Join(directory, failure)
			err := cli.Run(t.Context(), []string{"init", "--guest", filepath.Join(directory, "guest"), "--import", "example.com/author/go/example", project}, io.Discard)
			if err == nil || !strings.Contains(err.Error(), "fixture failure: "+failure) {
				t.Fatal("initialization lost tool diagnostics", err)
			}
			assertNoCLIFile(t, filepath.Join(project, "shim", "Cargo.lock"))
			assertNoCLIFile(t, filepath.Join(project, ".cocoon-build", "lock"))
			// Initialization's authored scaffold is deliberately retained. A
			// retry must refuse to overwrite it rather than discard user edits.
			if _, err := os.Stat(filepath.Join(project, "shim", "src", "implementation.rs")); err != nil {
				t.Fatal("failed initialization lost the authored scaffold", err)
			}
			t.Setenv("COCOON_CLI_TEST_FAILURE", "")
			if err := cli.Run(t.Context(), []string{"init", "--guest", filepath.Join(directory, "guest"), project}, io.Discard); err == nil || !strings.Contains(err.Error(), "refuse to overwrite") {
				t.Fatal("initialization retry overwrote the existing scaffold", err)
			}
		})
	}
	project := filepath.Join(directory, "blocked")
	writeCLIFile(t, project, []byte("authored file"))
	if err := cli.Run(t.Context(), []string{"init", "--guest", filepath.Join(directory, "guest"), filepath.Join(project, "child")}, io.Discard); err == nil {
		t.Fatal("initialization accepted a non-directory ancestor")
	}
}
