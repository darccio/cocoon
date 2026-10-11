package manifest_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"dario.cat/cocoon/internal/gen/rustgen"
	"dario.cat/cocoon/internal/manifest"
)

func checkedResource(name, directive string, methods ...string) string {
	var source strings.Builder
	fmt.Fprintf(&source, "\n[[resource]]\nname=%q\n%s\n[[resource.method]]\nname=\"new\"\nreturns=%q\n", name, directive, name)
	for _, method := range methods {
		fmt.Fprintf(&source, "[[resource.method]]\nname=%q\n", method)
	}
	source.WriteString("[[resource.method]]\nname=\"close\"\n")
	return source.String()
}

func TestCheckedBatchesPreserveGuestSchemaAndProvenance(t *testing.T) {
	t.Parallel()
	var wantSchema uint64
	var wantHash string
	var wantRust []byte
	provenance := make(map[[sha256.Size]byte]bool)
	for index, directive := range []string{"", "checked_batches=false", "checked_batches=true"} {
		source := example + checkedResource("Stream", directive, "write", "end")
		m, err := manifest.Parse([]byte(source))
		if err != nil {
			t.Fatal(err)
		}
		if m.Resources[0].CheckedBatches != (index == 2) {
			t.Fatal("checked_batches did not parse its default or explicit value")
		}
		before, err := manifest.Parse([]byte(source))
		if err != nil {
			t.Fatal(err)
		}
		schema, hash, err := m.SchemaHash()
		if err != nil || !reflect.DeepEqual(m, before) {
			t.Fatal("schema hashing mutated the parsed manifest", err)
		}
		rust, err := rustgen.Generate(m)
		if err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			wantSchema, wantHash, wantRust = schema, hash, rust
		} else if schema != wantSchema || hash != wantHash || !bytes.Equal(rust, wantRust) {
			t.Fatal("host-only policy changed guest schema or Rust shim")
		}
		if string(m.Content) != source {
			t.Fatal("full manifest lost host policy provenance")
		}
		provenance[sha256.Sum256(m.Content)] = true
	}
	if len(provenance) != 3 {
		t.Fatal("omitted and explicit directives did not retain distinct full manifests")
	}
	// Declaration order is not schema identity; retain the authored order and flag.
	source := checkedResource("Stream", "checked_batches=true", "end", "write")
	m, err := manifest.Parse([]byte(example + source))
	if err != nil {
		t.Fatal(err)
	}
	before, err := manifest.Parse([]byte(example + source))
	if err != nil {
		t.Fatal(err)
	}
	schema, hash, err := m.SchemaHash()
	if err != nil || schema != wantSchema || hash != wantHash || !reflect.DeepEqual(m, before) {
		t.Fatal("reordered checked methods changed schema or were mutated", err)
	}
}

func TestCheckedBatchNamesAreReservedOnlyWhenEnabled(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, extra string
		methods     []string
	}{
		{"Batch method", "", []string{"batch"}},
		{"command interface record", "\n[[record]]\nname=\"StreamCommand\"\nfields=[{name=\"value\",type=\"bytes\"}]\n", []string{"write"}},
		{"result interface global", "\n[[func]]\nname=\"stream_result\"\n", []string{"write"}},
		{"command constructor global", "\n[[func]]\nname=\"stream_write_command\"\n", []string{"write"}},
		{"result tag record", "\n[[record]]\nname=\"StreamWriteResult\"\nfields=[{name=\"value\",type=\"bytes\"}]\n", []string{"write"}},
		{"interface resource", checkedResource("StreamResult", ""), []string{"write"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			for _, directive := range []string{"", "checked_batches=false", "checked_batches=true"} {
				_, err := manifest.Parse([]byte(example + test.extra + checkedResource("Stream", directive, test.methods...)))
				if (err != nil) != (directive == "checked_batches=true") {
					t.Fatalf("directive %q: unexpected collision result: %v", directive, err)
				}
			}
		})
	}
}

func TestCheckedBatchConcatenatedNamesCannotCollide(t *testing.T) {
	t.Parallel()
	for _, directives := range [][2]string{{"", ""}, {"checked_batches=true", ""}, {"", "checked_batches=true"}, {"checked_batches=true", "checked_batches=true"}} {
		source := example + checkedResource("FooBar", directives[0], "baz") + checkedResource("Foo", directives[1], "bar_baz")
		_, err := manifest.Parse([]byte(source))
		if (err != nil) != (directives[0] != "" && directives[1] != "") {
			t.Fatalf("concatenated command names with %v: %v", directives, err)
		}
	}
}

func TestCheckedBatchesDoNotReserveExcludedOrQualifiedNames(t *testing.T) {
	t.Parallel()
	source := example + checkedResource("Stream", "checked_batches=true", "write")
	for _, name := range []string{"StreamNewCommand", "StreamCloseResult", "BatchLimits", "BatchError"} {
		source += fmt.Sprintf("\n[[record]]\nname=%q\nfields=[{name=\"value\",type=\"bytes\"}]\n", name)
	}
	if _, err := manifest.Parse([]byte(source)); err != nil {
		t.Fatal("reserved an excluded command or qualified runtime type", err)
	}
	if _, err := manifest.Parse([]byte(example + checkedResource("Stream", "checked_batches=\"true\"", "write"))); err == nil {
		t.Fatal("accepted a non-boolean checked_batches directive")
	}
}
