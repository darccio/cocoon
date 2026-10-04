package cocoontest_test

//go:generate wasm-as testdata/reference.wat -o testdata/reference.wasm

import (
	"encoding/binary"
	"errors"
	"os"
	"sync"
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

func TestI32ResultsAfterWideScalar(t *testing.T) {
	t.Parallel()
	wasm, err := os.ReadFile("testdata/reference.wasm")
	if err != nil {
		t.Fatal(err)
	}
	guest := cocoontest.New(t, wasm)
	for _, value := range []uint64{1 << 32, 0xffffffffffffffff, 7} {
		status, output, err := guest.Operation(t.Context(), "store", []cocoontest.Arg{{Value: value}}, 64, 64)
		if err != nil || status != rt.OK || len(output) != 8 || binary.LittleEndian.Uint64(output) != value {
			t.Fatalf("wide scalar %x: %d %x %v", value, status, output, err)
		}
	}
	if _, _, err := guest.Operation(t.Context(), "store", []cocoontest.Arg{{Value: 1}}, 64, 7); !errors.Is(err, rt.ErrTooLarge) {
		t.Fatal(err)
	}
}

func TestConcurrentReferenceConstruction(t *testing.T) {
	t.Parallel()
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			guest, err := cocoontest.Instantiate(t.Context(), []byte{0, 97, 115, 109, 1, 0, 0, 0})
			if err != nil {
				t.Error(err)
				return
			}
			if err := guest.Close(t.Context()); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
}
