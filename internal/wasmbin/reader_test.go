package wasmbin_test

import (
	"bytes"
	"testing"

	"dario.cat/cocoon/internal/wasmbin"
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

func smallModule(sections ...[]byte) []byte {
	data := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	for _, section := range sections {
		data = append(data, section...)
	}
	return data
}

func smallSection(id byte, payload ...byte) []byte {
	if len(payload) > 127 {
		panic("test section requires a one-byte size")
	}
	return append([]byte{id, byte(len(payload))}, payload...) // #nosec G115 -- Checked one-byte section size above.
}

func TestImportKindsTablesStartAndFeatures(t *testing.T) {
	t.Parallel()
	types := smallSection(1, 1, 0x60, 4, 0x7f, 0x7e, 0x7d, 0x7c, 0)
	imports := smallSection(2, 4,
		1, 'x', 1, 'f', 0, 0,
		1, 'x', 1, 't', 1, 0x70, 1, 0, 2,
		1, 'x', 1, 'm', 2, 1, 1, 2,
		1, 'x', 1, 'g', 3, 0x7e, 1)
	tables := smallSection(4, 1, 0x70, 0, 0)
	exports := smallSection(7, 4, 1, 'f', 0, 0, 1, 't', 1, 0, 1, 'm', 2, 0, 1, 'g', 3, 0)
	features := append([]byte{15}, "target_features"...)
	features = append(features, 2, '+', 8, 's', 'i', 'g', 'n', '-', 'e', 'x', 't', '-', 4, 's', 'i', 'm', 'd')
	m, err := wasmbin.Read(smallModule(types, imports, tables, exports, smallSection(8, 0), smallSection(12, 0), smallSection(0, features...)))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Imports) != 4 || len(m.Memories) != 1 || !m.HasStart || len(m.Features) != 1 || m.Features[0] != "sign-ext" {
		t.Fatalf("metadata: %+v", m)
	}
	signature, err := m.Signature(m.Exports[0])
	if err != nil || len(signature.Params) != 4 {
		t.Fatal(signature, err)
	}
	if _, err := m.Signature(wasmbin.Export{Kind: 0, Index: 99}); err == nil {
		t.Fatal("out-of-range signature accepted")
	}
}

func TestInvalidSectionSemantics(t *testing.T) {
	t.Parallel()
	for name, data := range map[string][]byte{
		"type-tag":          smallModule(smallSection(1, 1, 0x61, 0, 0)),
		"value-type":        smallModule(smallSection(1, 1, 0x60, 1, 0x70, 0)),
		"vector-limit":      smallModule(smallSection(1, 0xa1, 0x8d, 0x06)),
		"import-kind":       smallModule(smallSection(2, 1, 1, 'x', 1, 'y', 4)),
		"global-type":       smallModule(smallSection(2, 1, 1, 'x', 1, 'y', 3, 0x70, 0)),
		"global-mutability": smallModule(smallSection(2, 1, 1, 'x', 1, 'y', 3, 0x7f, 2)),
		"table-type":        smallModule(smallSection(4, 1, 0x6f, 0, 0)),
		"table-flags":       smallModule(smallSection(4, 1, 0x70, 2)),
		"table-limits":      smallModule(smallSection(4, 1, 0x70, 1, 2, 1)),
		"memory-limits":     smallModule(smallSection(5, 1, 1, 2, 1)),
		"memory-64":         smallModule(smallSection(5, 1, 5, 1, 2)),
		"export-kind":       smallModule(smallSection(7, 1, 1, 'x', 4, 0)),
		"export-index":      smallModule(smallSection(7, 1, 1, 'x', 0, 0)),
		"memory-index":      smallModule(smallSection(7, 1, 1, 'x', 2, 0)),
		"duplicate-export":  smallModule(smallSection(7, 2, 1, 'x', 3, 0, 1, 'x', 3, 0)),
		"non-utf8":          smallModule(smallSection(7, 1, 1, 255, 3, 0)),
		"start-index":       smallModule(smallSection(8, 0)),
		"trailing-section":  smallModule(smallSection(5, 0, 0)),
		"code-body":         smallModule(smallSection(10, 1, 2, 0, 0)),
		"type-index":        smallModule(smallSection(3, 1, 9), smallSection(10, 1, 2, 0, 0x0b)),
		"count-mismatch":    smallModule(smallSection(3, 1, 0)),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := wasmbin.Read(data); err == nil {
				t.Fatal("invalid metadata accepted")
			}
		})
	}
	features := append([]byte{15}, "target_features"...)
	features = append(features, 1, '!', 1, 'x')
	if _, err := wasmbin.Read(smallModule(smallSection(0, features...))); err == nil {
		t.Fatal("invalid feature prefix accepted")
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
