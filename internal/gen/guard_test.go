package gen_test

import (
	"os"
	"path/filepath"
	"testing"

	"dario.cat/cocoon/internal/gen"
)

func TestSharedProjectGuard(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	release, guardErr := gen.Guard(directory)
	if guardErr != nil {
		t.Fatal(guardErr)
	}
	if _, err := gen.Guard(directory); err == nil {
		t.Fatal("concurrent generation admitted")
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	next, nextErr := gen.Guard(directory)
	if nextErr != nil {
		t.Fatal(nextErr)
	}
	if err := next(); err != nil {
		t.Fatal(err)
	}
	if _, err := gen.Guard(filepath.Join(directory, "missing")); err == nil {
		t.Fatal("missing project admitted")
	}
	linked := t.TempDir()
	if err := os.Symlink(directory, filepath.Join(linked, ".cocoon-build")); err == nil {
		if _, err := gen.Guard(linked); err == nil {
			t.Fatal("symlinked build directory admitted")
		}
	}
}
