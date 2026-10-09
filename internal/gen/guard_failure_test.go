package gen_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"dario.cat/cocoon/internal/gen"
)

func TestGuardRejectsNonDirectoryBuildPath(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	path := filepath.Join(directory, ".cocoon-build")
	content := []byte("authored file")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if release, err := gen.Guard(directory); err == nil || release != nil {
		t.Fatal("non-directory build path admitted", err)
	}
	preserved, err := os.ReadFile(path) // #nosec G304 -- Read this test's own fixture below t.TempDir.
	if err != nil || !bytes.Equal(preserved, content) {
		t.Fatal("failed guard altered the existing file", err)
	}
}

func TestGuardReleaseRetainsFailureAndAllowsRetry(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	release, err := gen.Guard(directory)
	if err != nil {
		t.Fatal(err)
	}
	if removeErr := os.Remove(filepath.Join(directory, ".cocoon-build", "lock")); removeErr != nil {
		t.Fatal(removeErr)
	}
	first := release()
	if !errors.Is(first, os.ErrNotExist) {
		t.Fatal("missing lock removal error lost", first)
	}
	if again := release(); !errors.Is(again, first) {
		t.Fatal("idempotent release did not retain its first result", again)
	}
	next, err := gen.Guard(directory)
	if err != nil {
		t.Fatal("release failure prevented reacquiring the guard", err)
	}
	if err := next(); err != nil {
		t.Fatal(err)
	}
}
