package build_test

import (
	"encoding/binary"
	"slices"
	"testing"

	"cocoon.dev/cocoon/internal/build"
	"cocoon.dev/cocoon/internal/manifest"
)

func testManifest(t *testing.T) *manifest.Manifest {
	t.Helper()
	m, err := manifest.Parse([]byte(`[package]
name="test"
go_import="example.com/test"
rust_crate="test-shim"
[limits]
max_input="64KiB"
max_output="64KiB"
max_memory="128KiB"
instances="1"
[[func]]
name="echo"
params=[{name="input",type="string"}]
returns="string"
`))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func section(data []byte, id byte, body []byte) []byte {
	data = append(data, id)
	data = binary.AppendUvarint(data, uint64(len(body)))
	return append(data, body...)
}

func name(data []byte, value string) []byte {
	data = binary.AppendUvarint(data, uint64(len(value)))
	return append(data, value...)
}

func abiModule(t *testing.T, omit string, maximum byte) []byte {
	t.Helper()
	exports := build.ExportSet(testManifest(t))
	names := make([]string, 0, len(exports))
	for key := range exports {
		names = append(names, key)
	}
	slices.Sort(names)
	count := byte(len(names)) // #nosec G115 -- The fixture has eight fixed ABI exports.
	types, functions, exported, code := []byte{count}, []byte{count}, []byte{count + 1}, []byte{count}
	for index, key := range names {
		signature := exports[key]
		types = append(types, 0x60, byte(len(signature.Params))) // #nosec G115 -- Fixed fixture signatures have at most two parameters.
		for _, parameter := range signature.Params {
			types = append(types, byte(parameter))
		}
		types = append(types, byte(len(signature.Results))) // #nosec G115 -- Each ABI export has at most one result.
		for _, result := range signature.Results {
			types = append(types, byte(result))
		}
		functions = append(functions, byte(index)) // #nosec G115 -- Fixed ABI export indexes fit in one byte.
		exportName := key
		if key == omit {
			exportName = "unexpected"
		}
		exported = name(exported, exportName)
		exported = append(exported, 0, byte(index)) // #nosec G115 -- Fixed indexes fit in one byte.
		body := []byte{0}
		for _, result := range signature.Results {
			if result == 0x7e {
				body = append(body, 0x42, 0)
			} else {
				body = append(body, 0x41, 0)
			}
		}
		body = append(body, 0x0b)
		code = append(code, byte(len(body))) // #nosec G115 -- Bodies have at most four bytes.
		code = append(code, body...)         // #nosec G115 -- Bodies have at most four bytes.
	}
	exported = name(exported, "memory")
	exported = append(exported, 2, 0)
	data := []byte{0, 97, 115, 109, 1, 0, 0, 0}
	data = section(data, 1, types)
	data = section(data, 3, functions)
	data = section(data, 5, []byte{1, 1, 1, maximum})
	data = section(data, 7, exported)
	return section(data, 10, code)
}

func TestVerifyABI(t *testing.T) {
	t.Parallel()
	if _, err := build.Verify(testManifest(t), abiModule(t, "", 2)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{abiModule(t, "cocoon_out", 2), abiModule(t, "", 3), {0}} {
		if _, err := build.Verify(testManifest(t), bad); err == nil {
			t.Fatal("invalid ABI accepted")
		}
	}
}
