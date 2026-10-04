package harden_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/darccio/cocoon/internal/harden"
)

const bulk = `package example
func memory_copy[T uint32 | uint64](mem []byte, dest, src, n T) {
 x,z:=uint64(dest),uint64(src)
 copy(mem[x:x+uint64(n)],mem[z:z+uint64(n)])
}
func call(mem []byte) { memory_copy(mem,uint32(0),uint32(0),uint32(1)) }
`

func TestPinnedFixtureSignaturesAndDrift(t *testing.T) {
	t.Parallel()
	source, err := os.ReadFile("../../rt/internal/trapfix/testdata/raw.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	names, err := harden.Required(source)
	if err != nil || len(names) != 4 {
		t.Fatal(names, err)
	}
	if _, err := harden.Rewrite(source, names); err != nil {
		t.Fatal(err)
	}
	for _, change := range [][2]string{
		{"data string", "data []byte"},
		{"dest T, val int32", "dest T, val uint32"},
		{"dest, src, n T", "dest, src, n int32"},
		{"T uint32 | uint64", "T int32 | uint64"},
	} {
		mutated := bytes.ReplaceAll(source, []byte(change[0]), []byte(change[1]))
		if _, err := harden.Rewrite(mutated, names); err == nil {
			t.Fatalf("signature drift accepted: %s", change[1])
		}
	}
}

func TestRewrite(t *testing.T) {
	t.Parallel()
	expected, err := harden.Required([]byte(bulk))
	if err != nil {
		t.Fatal(err)
	}
	if len(expected) != 1 || expected[0] != "memory_copy" {
		t.Fatal(expected)
	}
	output, err := harden.Rewrite([]byte(bulk), expected)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(output, []byte(":len(mem)]")) != 2 {
		t.Fatal(string(output))
	}
	if _, err := harden.Rewrite(output, expected); err == nil {
		t.Fatal("already hardened body must report pin drift")
	}
	for _, source := range []string{strings.ReplaceAll(bulk, "mem []byte", "mem []uint32"), strings.ReplaceAll(bulk, "memory_copy[", "memory_mystery["), "package example", bulk + bulk[strings.Index(bulk, "func memory_copy"):strings.Index(bulk, "func call")]} {
		if _, err := harden.Rewrite([]byte(source), expected); err == nil {
			t.Fatal("helper drift accepted")
		}
	}
	if _, err := harden.Rewrite([]byte(bulk), nil); err == nil {
		t.Fatal("unexpected helper accepted")
	}
	if _, err := harden.Rewrite([]byte("package simple"), nil); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidSource(t *testing.T) {
	t.Parallel()
	if _, err := harden.Required([]byte("broken")); err == nil {
		t.Fatal("invalid Go accepted")
	}
	if _, err := harden.Rewrite([]byte("broken"), nil); err == nil {
		t.Fatal("invalid Go accepted")
	}
	if _, err := harden.Rewrite([]byte(bulk), []string{"memory_copy", "memory_copy"}); err == nil {
		t.Fatal("duplicate expectations accepted")
	}
}

func TestRewriteSuppliesOmittedSliceUpperBound(t *testing.T) {
	t.Parallel()
	source := strings.ReplaceAll(bulk, "x:x+uint64(n)", "x:")
	names, err := harden.Required([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	output, err := harden.Rewrite([]byte(source), names)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(output, []byte("mem[x:len(mem):len(mem)]")) {
		t.Fatal("omitted upper bound was not preserved as the memory length")
	}
	if _, err := harden.Required(output); err != nil {
		t.Fatal("hardening emitted invalid Go", err)
	}
}

func TestTableHardening(t *testing.T) {
	t.Parallel()
	source := `package wasm
type Module struct { t0 []any; memory []byte }
func(m *Module) call(index uint32) { m.t0[index].(func())() }
func(m *Module) set(index uint32,value any) { m.t0[index]=value }
func table_copy(tab,elems []any,low,high uint64) { copy(tab[low:high],elems[:high]) }
`
	output, err := harden.Rewrite([]byte(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"cocoon_table_index(uint64(index), len(m.t0))", "cocoon_table_slice(tab", "cocoon_table_slice(elems", "out of bounds table access", "table[low:high:len(table)]"} {
		if !bytes.Contains(output, []byte(expected)) {
			t.Fatalf("missing %s: %s", expected, output)
		}
	}
	if _, err := harden.Rewrite(output, nil); err == nil {
		t.Fatal("preexisting table hardening accepted")
	}
}

func TestOnlyUnreachableReturnsAreRemoved(t *testing.T) {
	t.Parallel()
	source := `package wasm
func call(flag bool) int {
if flag { goto live }
{ return 1 }
return 99
live:
return 2
}
`
	output, err := harden.Rewrite([]byte(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(output, []byte("return 99")) || !bytes.Contains(output, []byte("live:")) || bytes.Count(output, []byte("return")) != 2 {
		t.Fatalf("changed reachable control flow: %s", output)
	}
}

func FuzzRewrite(f *testing.F) {
	f.Add([]byte(bulk))
	f.Add([]byte(strings.ReplaceAll(bulk, "x:x+uint64(n)", "x:")))
	f.Fuzz(func(t *testing.T, source []byte) {
		if len(source) > 65536 {
			return
		}
		names, err := harden.Required(source)
		if err == nil {
			output, rewriteErr := harden.Rewrite(source, names)
			if rewriteErr == nil {
				if _, parseErr := harden.Required(output); parseErr != nil {
					t.Fatal(parseErr)
				}
			}
		}
	})
}
