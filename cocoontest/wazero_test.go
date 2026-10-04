package cocoontest_test

import (
	"errors"
	"testing"

	"cocoon.dev/cocoon/cocoontest"
	"cocoon.dev/cocoon/rt"
)

func TestReferenceLifecycleAndMissingExports(t *testing.T) {
	t.Parallel()
	wasm := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	guest := cocoontest.New(t, wasm)
	if guest.Memory() != nil {
		t.Fatal("empty fixture has memory")
	}
	if _, err := guest.Call(t.Context(), "missing"); !errors.Is(err, rt.ErrProtocol) {
		t.Fatal(err)
	}
	if _, _, err := guest.Operation(t.Context(), "missing", nil, 64, 64); !errors.Is(err, rt.ErrProtocol) {
		t.Fatal(err)
	}
	if _, _, err := guest.Operation(t.Context(), "missing", []cocoontest.Arg{{Buffer: true, Bytes: make([]byte, 65)}}, 64, 64); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal(err)
	}
	if err := guest.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := guest.Call(t.Context(), "missing"); !errors.Is(err, rt.ErrClosed) {
		t.Fatal(err)
	}
	if err := guest.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := cocoontest.Instantiate(t.Context(), []byte("malformed")); err == nil {
		t.Fatal("malformed module accepted")
	}
}
