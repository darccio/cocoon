package harden_test

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"dario.cat/cocoon/internal/harden"
)

const publication = `
func (receiver *Module) publish(base, value, stack int32) {
 store32(receiver.memory, uint32(base), uint32(value))
 store32(receiver.memory, uint64(uint32(base))+4, uint32(i32(7)))
 receiver.g0 = stack + i32(16)
}
`

// The trap fixture has only a load32. Include the pinned translator's store32
// shape here; the executable WAT proof obtains both helpers from wasm2go.
const publicationStore = `
//go:nosplit
func store32[T uint32 | uint64](mem []byte, addr T, val uint32) {
 if !unalignedOK { binary.LittleEndian.PutUint32(mem[addr:], val); return }
 if big { val = bits.ReverseBytes32(val) }
 _ = mem[uint64(addr)+3]
 *(*uint32)(unsafe.Add(unsafe.Pointer(unsafe.SliceData(mem)), uintptr(addr))) = val
}
`

func viewFixture(t *testing.T, method string) []byte {
	t.Helper()
	source, err := os.ReadFile("../../rt/internal/trapfix/testdata/raw.go.txt")
	if err != nil {
		t.Fatal(err)
	}
	source = bytes.Replace(source, []byte("type Module struct {"), []byte("type Module struct {\n g0 int32"), 1)
	return append(source, []byte(publicationStore+method)...)
}

func rewriteViewFixture(t *testing.T, source []byte) []byte {
	t.Helper()
	names, err := harden.Required(source)
	if err != nil {
		t.Fatal(err)
	}
	output, err := harden.Rewrite(source, names)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func TestPublicationMemoryViews(t *testing.T) {
	t.Parallel()
	for _, method := range []string{
		publication,
		strings.ReplaceAll(publication, "publish", "differentPayload"),
	} {
		output := rewriteViewFixture(t, viewFixture(t, method))
		if bytes.Count(output, []byte("cocoonMemoryView1 = receiver.memory")) != 1 ||
			bytes.Count(output, []byte("store32(cocoonMemoryView1,")) != 2 ||
			!bytes.Contains(output, []byte("var cocoonMemoryView1 []byte")) {
			t.Fatal("structural publication tail was not rewritten")
		}
	}
	source := bytes.ReplaceAll(viewFixture(t, publication), []byte("g0"), []byte("anotherGlobal"))
	if !bytes.Contains(rewriteViewFixture(t, source), []byte("store32(cocoonMemoryView1,")) {
		t.Fatal("global name affected eligibility")
	}
}

func TestPublicationEntryAndLiveLoad(t *testing.T) {
	t.Parallel()
	method := `
func (m *Module) publish(base, value, stack int32) {
 if value == 0 { goto join }
 store32(m.memory, uint64(uint32(base))+4, uint32(value))
 store32(m.memory, uint32(base), uint32(i32(9)))
 m.g0 = stack + i32(16)
 goto join
join:
 value = int32(load32(m.memory, uint32(base)))
 loaded := int32(load32(m.memory, uint64(uint32(base))+4))
 copied := loaded
 store32(m.memory, uint32(base), uint32(copied))
 store32(m.memory, uint64(uint32(base))+4, uint32(value))
 m.g0 = stack + i32(16)
}
`
	output := rewriteViewFixture(t, viewFixture(t, method))
	if bytes.Count(output, []byte("= m.memory")) != 2 ||
		!bytes.Contains(output, []byte("join:\n\tvalue = int32(load32(m.memory,")) ||
		!bytes.Contains(output, []byte("load32(cocoonMemoryView2,")) ||
		bytes.Count(output, []byte("store32(cocoonMemoryView")) != 4 {
		t.Fatal("label entry or live-memory access changed", string(output[len(output)-1500:]))
	}
}

func TestPublicationUnsupportedShapesStayUnchanged(t *testing.T) {
	t.Parallel()
	for _, change := range [][2]string{
		{"uint64(uint32(base))+4", "uint64(uint32(base))+8"},
		{"uint64(uint32(base))+4", "uint32(base+i32(4))"},
		{"uint64(uint32(base))+4", "uint64(base)+4"},
		{"uint64(uint32(base))+4", "uint64(uint32(base))+0xffffffffffffffff"},
		{"receiver.g0 = stack + i32(16)", "receiver.g0 = stack"},
		{"uint32(value)", "uint32(value+i32(1))"},
		{"uint32(value)", "uint32(load32(receiver.memory, uint32(base)))"},
		{"store32(receiver.memory, uint64", "receiver.fn0()\n store32(receiver.memory, uint64"},
		{"store32(receiver.memory, uint64", "between:\n store32(receiver.memory, uint64"},
		{"receiver.g0 = stack + i32(16)", "receiver.g0 = stack + i32(16)\n return int32(load32(receiver.memory, uint32(base)))"},
		{"receiver.g0 = stack + i32(16)", "receiver.g0 = stack + i32(16)\n receiver.fn0()"},
		{"base, value, stack int32", "base int64, value, stack int32"},
		{"base, value, stack int32", "base, value, stack int32, byte int32"},
		{"base, value, stack int32", "base, value, stack int32, cocoonMemoryView1 int32"},
	} {
		source := viewFixture(t, strings.Replace(publication, change[0], change[1], 1))
		if bytes.Contains(rewriteViewFixture(t, source), []byte("var cocoonMemoryView")) {
			t.Fatalf("unproved publication shape rewritten: %q", change[1])
		}
	}
}

func TestPublicationHelperAndOwnershipDrift(t *testing.T) {
	t.Parallel()
	for _, change := range [][2]string{
		{"_ = mem[uint64(addr)+3]", "_ = mem[uint64(addr)+2]"},
		{"func i32(x int32) int32 { return x }", "func i32(x int32) int32 { return x + 1 }"},
		{"runtime.GOARCH == \"ppc64\"", "runtime.GOARCH == \"other\""},
		{"\"encoding/binary\"", "\"example.com/binary\""},
		{"\"unsafe\"", "native \"unsafe\""},
		{"m.memory = make([]byte, 65536)", "m.memory = make([]byte, 65536)\n m.memory = unsafe.Slice((*byte)(unsafe.Pointer(m)),65536)"},
		{"return m\n}", "m = new(Module)\n return m\n}"},
		{"return m\n}", "return new(Module)\n}"},
		{"return m\n}", "p := &m\n *p = new(Module)\n return m\n}"},
		{"return m\n}", "*(&m) = new(Module)\n return m\n}"},
		{"return m\n}", "p := m\n *p = Module{}\n return m\n}"},
		{"return m\n}", "for _, m = range []*Module{new(Module)} {}\n return m\n}"},
		{"func New() *Module {", "func New() (result *Module) {"},
		{"return m\n}", "defer func() { m.memory = nil }()\n return m\n}"},
		{"return m\n}", "((m)) = new(Module)\n return m\n}"},
		{"return m\n}", "p := m\n (*p) = Module{}\n return m\n}"},
		{"return m\n}", "(m.memory) = nil\n return m\n}"},
		{"return m\n}", "p := &(m)\n (*p) = new(Module)\n return m\n}"},
		{"return m\n}", "p := &(m.memory)\n (*p) = nil\n return m\n}"},
		{"return m\n}", "for _, (m) = range []*Module{new(Module)} {}\n return m\n}"},
		{"return m\n}", "for _, m.memory = range [][]byte{nil} {}\n return m\n}"},
		{"return m\n}", "for _, (m.memory) = range [][]byte{nil} {}\n return m\n}"},
		{"func New() *Module {", "var clearModule = func(p *Module) { *p = Module{} }\n func New() *Module {"},
		{"func New() *Module {", "type int int64\n func New() *Module {"},
		{"//go:nosplit\nfunc load32", "//go:linkname load32 other\nfunc load32"},
		{"base, value, stack int32", "base, value, stack int32, uint32 func(int32) uint32"},
		{"base, value, stack int32", "base, value, stack int32, store32 func([]byte,uint32,uint32)"},
		{"base, value, stack int32", "base, value, stack int32, other []byte"},
	} {
		source := viewFixture(t, publication)
		if !bytes.Contains(source, []byte(change[0])) {
			t.Fatalf("missing drift target %q", change[0])
		}
		source = bytes.Replace(source, []byte(change[0]), []byte(change[1]), 1)
		if strings.Contains(change[1], "other []byte") {
			source = bytes.ReplaceAll(source, []byte("store32(receiver.memory"), []byte("store32(other"))
		}
		if bytes.Contains(rewriteViewFixture(t, source), []byte("var cocoonMemoryView")) {
			t.Fatalf("unproved binding/construction rewritten: %q", change[1])
		}
	}
}

func TestPublicationShadowedBindings(t *testing.T) {
	t.Parallel()
	// The first store's explicit widened zero offset keeps these fake helper
	// signatures valid Go while retaining the same structural offset pair.
	stores := `
 store32(receiver.memory, uint64(uint32(base))+0, uint32(value))
 store32(receiver.memory, uint64(uint32(base))+4, uint32(i32(7)))
 receiver.g0 = stack + i32(16)
 return
`
	fakeStore := `func(memory []byte, address uint64, value uint32) {}`
	for _, scope := range []string{
		"store32 := " + fakeStore + stores,
		"if store32 := " + fakeStore + "; value != 0 {" + stores + "}",
		"for _, store32 := range []func([]byte, uint64, uint32){" + fakeStore + "} {" + stores + "}",
		"switch store32 := " + fakeStore + "; value { case 1:" + stores + "}",
		"{ uint32 := func(value int32) uint32 { return 0 };" + stores + "}",
		"{ uint64 := func(value uint32) uint64 { return 0 };" + stores + "}",
		"{ i32 := func(value int32) int32 { return 0 };" + stores + "}",
		"{ receiver := new(Module);" + stores + "}",
		"{ type word int32; var base word;" + stores + "}",
	} {
		method := "func (receiver *Module) publish(base, value, stack int32) {\n" + scope + "\n}"
		if bytes.Contains(rewriteViewFixture(t, viewFixture(t, method)), []byte("var cocoonMemoryView")) {
			t.Fatalf("shadowed declaration transformed: %s", scope)
		}
	}
}

func TestPublicationLabelledBlockCapturesAfterEntry(t *testing.T) {
	t.Parallel()
	method := `
func (receiver *Module) publish(base, value, stack int32) {
 goto join
join: {
 store32(receiver.memory, uint32(base), uint32(value))
 store32(receiver.memory, uint64(uint32(base))+4, uint32(i32(7)))
 receiver.g0 = stack + i32(16)
 return
}
}
`
	output := rewriteViewFixture(t, viewFixture(t, method))
	if !bytes.Contains(output, []byte("join:\n\t{\n\t\tcocoonMemoryView1 = receiver.memory")) ||
		bytes.Count(output, []byte("store32(cocoonMemoryView1,")) != 2 {
		t.Fatal("labelled-block capture was not placed after entry")
	}
}
