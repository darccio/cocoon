package build

import (
	"encoding/binary"
	"os"
	"slices"
	"strings"
	"testing"

	"dario.cat/cocoon/internal/manifest"
)

func replaceSection(t *testing.T, module []byte, wanted byte, transform func([]byte) []byte) []byte {
	t.Helper()
	output := slices.Clone(module[:8])
	changed := false
	for remaining := module[8:]; len(remaining) != 0; {
		id := remaining[0]
		size, n := binary.Uvarint(remaining[1:])
		if n <= 0 || n >= len(remaining) {
			t.Fatal("malformed test fixture section")
		}
		remaining = remaining[1+n:]
		if size > uint64(len(remaining)) {
			t.Fatal("truncated test fixture section")
		}
		body := slices.Clone(remaining[:size])
		if !changed && id > wanted {
			output = section(output, wanted, transform(nil))
			changed = true
		} else if id == wanted {
			body = transform(body)
			changed = true
		}
		output = section(output, id, body)
		remaining = remaining[size:]
	}
	return output
}

func TestVerifyComputeResourceAndCapabilitySignatures(t *testing.T) {
	t.Parallel()
	manifestData, err := os.ReadFile("../../testdata/compute/cocoon.toml")
	if err != nil {
		t.Fatal(err)
	}
	m, err := manifest.Parse(manifestData)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../testdata/compute/go/compute/testdata/module.wasm")
	if err != nil {
		t.Fatal(err)
	}
	m.Capabilities.Clock, m.Capabilities.Random = true, true
	if _, err := Verify(m, data); err != nil {
		t.Fatal("valid scalar, record, resource, or capability signature rejected", err)
	}
	m.Capabilities.Log = false
	if _, err := Verify(m, data); err == nil || !strings.Contains(err.Error(), "undeclared") {
		t.Fatal("disabled log capability admitted", err)
	}
}

func TestVerifyRejectsStartExtraExportsAndMultipleMemories(t *testing.T) {
	t.Parallel()
	valid := abiModule(t, "", 2)
	withStart := replaceSection(t, valid, 8, func([]byte) []byte { return []byte{0} })
	extra := replaceSection(t, valid, 7, func(body []byte) []byte {
		body[0]++
		return append(name(body, "extra"), 0, 0)
	})
	memories := replaceSection(t, valid, 5, func([]byte) []byte { return []byte{2, 1, 1, 2, 1, 1, 2} })
	for _, scenario := range []struct {
		name, diagnostic string
		data             []byte
	}{
		{"start", "start functions", withStart},
		{"extra-export", "unexpected export", extra},
		{"multiple-memories", "exactly one memory", memories},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Verify(testManifest(t), scenario.data); err == nil || !strings.Contains(err.Error(), scenario.diagnostic) {
				t.Fatalf("missing %s policy error: %v", scenario.name, err)
			}
		})
	}
	if _, err := Verify(&manifest.Manifest{}, valid); err == nil {
		t.Fatal("invalid manifest accepted by verification")
	}
}

func TestVerifyRejectsCapabilitySignatureAndDuplicateImports(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name, diagnostic string
		count            byte
		correct          bool
	}{
		{"wrong-signature", "import signature mismatch", 1, false},
		{"duplicate", "duplicate import", 2, true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			m := testManifest(t)
			m.Capabilities.Log = true
			valid := abiModule(t, "", 2)
			var index byte
			valid = replaceSection(t, valid, 1, func(body []byte) []byte {
				index = body[0]
				body[0]++
				if scenario.correct {
					return append(body, 0x60, 3, 0x7f, 0x7f, 0x7f, 0)
				}
				return append(body, 0x60, 0, 0)
			})
			imports := []byte{scenario.count}
			for range scenario.count {
				imports = name(name(imports, "cocoon"), "log")
				imports = append(imports, 0, index)
			}
			end := 10 + int(valid[9])
			valid = append(section(slices.Clone(valid[:end]), 2, imports), valid[end:]...)
			if _, err := Verify(m, valid); err == nil || !strings.Contains(err.Error(), scenario.diagnostic) {
				t.Fatal("capability policy diagnostic lost", err)
			}
		})
	}
}
