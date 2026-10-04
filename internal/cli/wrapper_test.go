package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompilerWrapperDispatch(t *testing.T) {
	t.Setenv("COCOON_RUSTC_WRAPPER", "1")
	var output bytes.Buffer
	if err := Run(t.Context(), nil, &output); err == nil || !strings.Contains(err.Error(), "missing Rust compiler") {
		t.Fatal("compiler wrapper was not selected", err)
	}
	t.Setenv("COCOON_RUSTC_WRAPPER", "")
	if err := Run(t.Context(), []string{"help"}, &output); err != nil || !strings.Contains(output.String(), "usage: cocoon") {
		t.Fatal("ordinary CLI dispatch changed", err)
	}
}
