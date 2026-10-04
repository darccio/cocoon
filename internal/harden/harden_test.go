package harden_test

import (
	"bytes"
	"strings"
	"testing"

	"cocoon.dev/cocoon/internal/harden"
)

const bulk = `package example
func memory_copy[T uint32 | uint64](mem []byte, dest, src, n T) {
 x,z:=uint64(dest),uint64(src)
 copy(mem[x:x+uint64(n)],mem[z:z+uint64(n)])
}
func call(mem []byte) { memory_copy(mem,uint32(0),uint32(0),uint32(1)) }
`

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

func FuzzRewrite(f *testing.F) {
	f.Add([]byte(bulk))
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
