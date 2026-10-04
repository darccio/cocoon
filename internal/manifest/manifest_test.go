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
