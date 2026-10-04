package manifest_test

import (
	"strings"
	"testing"

	"cocoon.dev/cocoon/internal/manifest"
)

const example = `
[package]
name="example"
go_import="example.com/example"
rust_crate="example-shim"
[limits]
max_input="64KiB"
max_output="64KiB"
max_memory="16MiB"
instances="1"
[[func]]
name="echo"
params=[{name="input",type="string"}]
returns="string"
fallible=true
`

func TestParseAndSchema(t *testing.T) {
	t.Parallel()
	m, err := manifest.Parse([]byte(example))
	if err != nil {
		t.Fatal(err)
	}
	hash, full, err := m.SchemaHash()
	if err != nil || hash == 0 || len(full) != 64 {
		t.Fatalf("hash: %v", err)
	}
	n, err := manifest.Parse([]byte(strings.ReplaceAll(example, "64KiB", "65536B")))
	if err != nil {
		t.Fatal(err)
	}
	n.Package.GoImport = "elsewhere.test/moved"
	other, _, err := n.SchemaHash()
	if err != nil || other != hash {
		t.Fatal("nonsemantic edit changed schema")
	}
	n.Functions[0].Returns = "bytes"
	other, _, err = n.SchemaHash()
	if err != nil || other == hash {
		t.Fatal("semantic edit did not change schema")
	}
	if manifest.GoName("obfuscate_sql") != "ObfuscateSQL" {
		t.Fatal("Go initialism conversion")
	}
}

func TestRejectInvalidManifests(t *testing.T) {
	t.Parallel()
	for _, replacement := range []struct{ old, new string }{
		{"name=\"echo\"", "name=\"Open\""},
		{"type=\"string\"", "type=\"[]string\""},
		{"64KiB", "0"},
		{"16MiB", "1B"},
		{"instances=\"1\"", "instances=\"0\""},
		{"fallible=true", "async=true"},
		{"fallible=true", "typo=true"},
		{"example.com/example", "../escape"},
		{"example-shim", "../bad"},
		{"example.com/example", "."},
		{"name=\"echo\"", "name=\"abi_version\""},
		{"name=\"echo\"", "name=\"schema_hash\""},
	} {
		if _, err := manifest.Parse([]byte(strings.ReplaceAll(example, replacement.old, replacement.new))); err == nil {
			t.Fatalf("accepted %q", replacement.new)
		}
	}
	for _, limit := range []string{"", "-1", "18446744073709551615GiB", "1MB", "4294967296B"} {
		if _, err := manifest.ParseSize(limit); err == nil {
			t.Fatalf("accepted %q", limit)
		}
	}
	for _, extra := range []string{
		"\n[[func]]\nname=\"echo\"", "\n[[resource]]\nname=\"Sketch\"\ncopy=\"shared\"",
		"\n[[record]]\nname=\"Config\"\nfields=[{name=\"again\",type=\"Config\"}]",
	} {
		if _, err := manifest.Parse([]byte(example + extra)); err == nil {
			t.Fatal("invalid declarations accepted")
		}
	}
}

func TestInstanceCountsAreCanonicalDecimal(t *testing.T) {
	t.Parallel()
	var expected string
	for _, count := range []string{"10", "010", "00010"} {
		m, err := manifest.Parse([]byte(strings.ReplaceAll(example, `instances="1"`, `instances="`+count+`"`)))
		if err != nil {
			t.Fatal(err)
		}
		if m.Limits.Instances != "10" {
			t.Fatalf("instance count %q normalized to %q", count, m.Limits.Instances)
		}
		_, full, err := m.SchemaHash()
		if err != nil {
			t.Fatal(err)
		}
		if expected == "" {
			expected = full
		} else if full != expected {
			t.Fatal("equivalent instance counts changed schema")
		}
	}
}

func TestGeneratedNamesCannotCollide(t *testing.T) {
	t.Parallel()
	resource := `
[[resource]]
name="Sketch"
[[resource.method]]
name="new"
returns="Sketch"
[[resource.method]]
name="close"
`
	for _, source := range []string{
		strings.ReplaceAll(example, `name="echo"`, `name="sketch_new"`) + resource,
		example + strings.ReplaceAll(resource, "Sketch", "Service"),
		example + strings.ReplaceAll(resource, "Sketch", "sketch"),
		example + `
[[record]]
name="String"
fields=[{name="text",type="string"}]
`,
		example + `
[[record]]
name="Empty"
`,
	} {
		if _, err := manifest.Parse([]byte(source)); err == nil {
			t.Fatal("accepted generated name collision or unsupported empty record")
		}
	}
}

func TestGeneratedRustPreludeNamesAreReserved(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"Default", "Ok", "Err"} {
		source := example + `
[[record]]
name="` + name + `"
fields=[{name="value",type="i32"}]
`
		if _, err := manifest.Parse([]byte(source)); err == nil {
			t.Fatalf("accepted Rust prelude collision %q", name)
		}
	}
}

func TestSourcePinsAndEmptyParameterNormalization(t *testing.T) {
	t.Parallel()
	const pins = `
[[source]]
name="upstream"
path="../upstream"
revision="0123456789012345678901234567890123456789"
`
	source := example + pins
	m, err := manifest.Parse([]byte(source))
	if err != nil || string(m.Content) != source {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.ReplaceAll(source, "revision=", "unknown="), strings.ReplaceAll(source, "0123456789012345678901234567890123456789", "main"), source + pins} {
		if _, parseErr := manifest.Parse([]byte(bad)); parseErr == nil {
			t.Fatal("invalid source pin accepted")
		}
	}
	m.Functions[0].Params = nil
	first, _, err := m.SchemaHash()
	if err != nil {
		t.Fatal(err)
	}
	m.Functions[0].Params = []manifest.Param{}
	m.Sources = nil
	second, _, err := m.SchemaHash()
	if err != nil || first != second {
		t.Fatal("nonsemantic source pins or empty slices changed schema")
	}
}

func TestResourceValidation(t *testing.T) {
	t.Parallel()
	source := example + `
[[resource]]
name="Sketch"
copy="shared"
[[resource.method]]
name="new"
returns="Sketch"
[[resource.method]]
name="add_many"
params=[{name="values",type="[]f64"}]
[[resource.method]]
name="close"
`
	m, err := manifest.Parse([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if !m.Variable("[]f64") || m.Variable("i32") {
		t.Fatal("variable type classification")
	}
	for _, replacement := range []struct{ old, new string }{{"copy=\"shared\"", "copy=\"unique\""}, {"returns=\"Sketch\"", "returns=\"bytes\""}, {"name=\"close\"", "name=\"Close\""}} {
		if _, err := manifest.Parse([]byte(strings.ReplaceAll(source, replacement.old, replacement.new))); err == nil {
			t.Fatalf("accepted %q", replacement.new)
		}
	}
}

func FuzzParse(f *testing.F) {
	f.Add([]byte(example))
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > 65536 {
			return
		}
		if m, err := manifest.Parse(input); err == nil {
			if _, _, hashErr := m.SchemaHash(); hashErr != nil {
				t.Fatal(hashErr)
			}
		}
	})
}
