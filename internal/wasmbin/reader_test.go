package wasmbin_test

import (
	"bytes"
	"testing"

	"cocoon.dev/cocoon/internal/wasmbin"
)

func fixture() []byte {
	return []byte{
		0, 97, 115, 109, 1, 0, 0, 0,
		1, 5, 1, 0x60, 0, 1, 0x7f,
		3, 2, 1, 0,
		5, 4, 1, 1, 1, 2,
		7, 7, 1, 3, 'r', 'u', 'n', 0, 0,
		10, 6, 1, 4, 0, 0x41, 0, 0x0b,
	}
}

func TestRead(t *testing.T) {
	t.Parallel()
	m, err := wasmbin.Read(fixture())
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Exports) != 1 || m.Memories[0].Max != 2 {
		t.Fatal("incorrect sections")
	}
	sig, err := m.Signature(m.Exports[0])
	if err != nil || len(sig.Results) != 1 || sig.Results[0] != wasmbin.I32 {
		t.Fatalf("signature: %v", err)
	}
	if _, err := m.Signature(wasmbin.Export{Kind: 2}); err == nil {
		t.Fatal("memory treated as function")
	}
}

func TestMalformed(t *testing.T) {
	t.Parallel()
	data := fixture()
	for size := range data {
		if size == 8 || size == 15 || size == 19 || size == 25 {
			continue
		}
		if _, err := wasmbin.Read(data[:size]); err == nil {
			t.Fatalf("truncation %d accepted", size)
		}
	}
	for _, bad := range [][]byte{append(bytes.Clone(data), 1, 1, 0), append(bytes.Clone(data), 13, 0), append(bytes.Clone(data), 0, 0), append(bytes.Clone(data), 0, 255, 255, 255, 255, 16)} {
		if _, err := wasmbin.Read(bad); err == nil {
			t.Fatal("malformed section accepted")
		}
	}
	bad := bytes.Clone(data)
	bad[22] = 0
	if _, err := wasmbin.Read(bad); err == nil {
		t.Fatal("unbounded memory accepted")
	}
}

func FuzzRead(f *testing.F) {
	f.Add(fixture())
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) <= 1048576 {
			module, err := wasmbin.Read(input)
			if err == nil && module == nil {
				t.Fatal("successful parse returned nil")
			}
		}
	})
}
