package rt_test

import (
	"bytes"
	"os"
	"testing"

	"dario.cat/cocoon/internal/gen/rtgen"
)

func TestCheckedBoundaryReproduces(t *testing.T) {
	t.Parallel()
	actual, err := os.ReadFile("instance.go")
	if err != nil {
		t.Fatal(err)
	}
	want, err := rtgen.Update(actual)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, want) {
		t.Fatal("checked adapters drifted: update their shared template and run go generate ./rt")
	}
}
