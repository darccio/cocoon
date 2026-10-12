package harden_test

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"slices"
	"strings"
	"testing"
)

const containedReads = `
func (m *Module) reads(base int32) int32 {
 first := int32(load32(m.memory, uint64(uint32(base))+8))
 last := int32(load32(m.memory, uint32(base)))
 return first ^ last
}
`

const containedHelper = "cocoonLoad32InRange"

func containedCalls(t *testing.T, source []byte, method string) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "contained.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	var calls []string
	found := false
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || function.Name.Name != method {
			continue
		}
		found = true
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name, ok := call.Fun.(*ast.Ident)
			if ok && (name.Name == "load32" || name.Name == containedHelper) {
				calls = append(calls, name.Name)
			}
			return true
		})
	}
	if !found {
		t.Fatalf("missing method %s", method)
	}
	return calls
}

func assertContainedCalls(t *testing.T, source []byte, method string, want ...string) {
	t.Helper()
	if got := containedCalls(t, source, method); !slices.Equal(got, want) {
		t.Fatalf("%s load order = %v, want %v", method, got, want)
	}
}

func TestContainedLoadPatterns(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		method string
		want   []string
	}{
		{"inferredConverted", containedReads, []string{"load32", containedHelper}},
		{"differentMethodName", strings.Replace(containedReads, "reads(", "otherPayload(", 1), []string{"load32", containedHelper}},
		{"inferredUnsigned", `func (m *Module) reads(base int32) uint32 {
 first := load32(m.memory, uint64(uint32(base))+8)
 last := load32(m.memory, uint32(base))
 return first ^ last
}`, []string{"load32", containedHelper}},
		{"explicitSigned", `func (m *Module) reads(base int32) int32 {
 var first, last int32
 first = int32(load32(m.memory, uint64(uint32(base))+8))
 last = int32(load32(m.memory, uint32(base)))
 return first ^ last
}`, []string{"load32", containedHelper}},
		{"explicitUnsigned", `func (m *Module) reads(base int32) uint32 {
 var first, last uint32
 first = load32(m.memory, uint64(uint32(base))+8)
 last = load32(m.memory, uint32(base))
 return first ^ last
}`, []string{"load32", containedHelper}},
		{"firstAndNewMaximum", `func (m *Module) reads(base int32) int32 {
 first := int32(load32(m.memory, uint64(uint32(base))+4))
 highest := int32(load32(m.memory, uint64(uint32(base))+8))
 last := int32(load32(m.memory, uint32(base)))
 return first ^ highest ^ last
}`, []string{"load32", "load32", containedHelper}},
		{"sameEndpoint", strings.Replace(containedReads, "uint32(base)))", "uint64(uint32(base))+8))", 1), []string{"load32", containedHelper}},
		{"maximumLiteralOffset", strings.Replace(containedReads, "+8", "+0xffffffff", 1), []string{"load32", containedHelper}},
		{"risingEndpoints", `func (m *Module) reads(base int32) int32 {
 first := int32(load32(m.memory, uint32(base)))
 middle := int32(load32(m.memory, uint64(uint32(base))+4))
 last := int32(load32(m.memory, uint64(uint32(base))+8))
 return first ^ middle ^ last
}`, []string{"load32", "load32", "load32"}},
		{"nestedFreshRegion", `func (m *Module) reads(base int32) int32 {
 {
  first := int32(load32(m.memory, uint64(uint32(base))+8))
  last := int32(load32(m.memory, uint32(base)))
  return first ^ last
 }
}`, []string{"load32", containedHelper}},
		{"typedDeclarationsBarrier", strings.Replace(containedReads, " last :=", ` var copied int32
 copied = i32(base + i32(1))
 var wide int64
 wide = i64(int64(copied))
 var fraction float64
 fraction = float64(wide)
 copied = int32(fraction)
 last :=`, 1), []string{"load32", "load32"}},
		{"typedPureLocals", `func (m *Module) reads(base int32) int32 {
 var copied int32
 var wide int64
 var fraction float64
 first := int32(load32(m.memory, uint64(uint32(base))+8))
 copied = i32(base + i32(1))
 wide = i64(int64(copied))
 fraction = float64(wide)
 copied = int32(fraction)
 last := int32(load32(m.memory, uint32(base)))
 return first ^ last ^ copied ^ int32(wide) ^ int32(fraction)
}`, []string{"load32", containedHelper}},
		{"inferredPureLocals", strings.Replace(containedReads, " last :=", ` copied := i32(base + i32(1))
 copied = copied ^ base
 wide := i64(int64(copied))
 wide = int64(uint64(wide) >> 1)
 fraction := float64(wide)
 copied = int32(fraction)
 last :=`, 1), []string{"load32", containedHelper}},
		{"rhsBeforeBaseKill", `func (m *Module) reads(base int32) int32 {
 first := int32(load32(m.memory, uint64(uint32(base))+8))
 base = int32(load32(m.memory, uint32(base)))
 last := int32(load32(m.memory, uint32(base)))
 return first ^ last ^ base
}`, []string{"load32", containedHelper, "load32"}},
		{"explicitLocalBase", `func (m *Module) reads(input int32) int32 {
 var base int32
 base = input
 first := int32(load32(m.memory, uint64(uint32(base))+8))
 last := int32(load32(m.memory, uint32(base)))
 return first ^ last
}`, []string{"load32", containedHelper}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			method := test.method
			want := test.want
			name := "reads"
			if test.name == "differentMethodName" {
				name = "otherPayload"
			}
			output := rewriteViewFixture(t, viewFixture(t, method))
			assertContainedCalls(t, output, name, want...)
			if !slices.Contains(want, containedHelper) && bytes.Contains(output, []byte("func "+containedHelper+"[")) {
				t.Fatal("unneeded private helper was appended")
			}
		})
	}
}

func TestContainedLoadBarriers(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		between string
	}{
		{"baseReassignment", "base = i32(base + i32(1))"},
		{"compoundAssignment", "base += 1"},
		{"increment", "base++"},
		{"decrement", "base--"},
		{"multiAssignment", "base, first = first, base"},
		{"declaration", "var copied int32\n copied = base\n _ = copied"},
		{"block", "{ copied := base; _ = copied }"},
		{"conditional", "if base != 0 { base = 1 }"},
		{"loop", "for index := 0; index < 1; index++ {}"},
		{"switch", "switch base { case 0: base = 1 }"},
		{"call", "m.Xunreachable()"},
		{"store", "store32(m.memory, uint32(base), uint32(first))"},
		{"headerWrite", "m.memory = nil"},
		{"receiverFieldWrite", "m.g0 = base"},
		{"growth", "_ = memory_grow(&m.memory, int64(1), m.maxMem)"},
		{"bulk", "memory_zero(m.memory, uint32(base), uint32(0))"},
		{"pointerEscape", "address := &base\n _ = address"},
		{"closure", "callback := func() int32 { return base }\n _ = callback"},
		{"defer", "defer m.Xunreachable()"},
		{"goroutine", "go m.Xunreachable()"},
		{"label", "join:"},
		{"goto", "goto join\njoin:"},
		{"return", "return first"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			method := strings.Replace(containedReads, " last :=", test.between+"\n last :=", 1)
			output := rewriteViewFixture(t, viewFixture(t, method))
			assertContainedCalls(t, output, "reads", "load32", "load32")
			if bytes.Contains(output, []byte("func "+containedHelper+"[")) {
				t.Fatal("barrier-only fixture emitted private helper")
			}
		})
	}
}

func TestContainedLoadDistinctBindings(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		method string
		want   []string
	}{
		{"groupedParameters", `func (m *Module) reads(base, other int32) int32 {
 first := int32(load32(m.memory, uint64(uint32(base))+8))
 last := int32(load32(m.memory, uint32(other)))
 return first ^ last
}`, []string{"load32", "load32"}},
		{"groupedVariables", `func (m *Module) reads(input, next int32) int32 {
 var base, other int32
 base = input
 other = next
 first := int32(load32(m.memory, uint64(uint32(base))+8))
 last := int32(load32(m.memory, uint32(other)))
 return first ^ last
}`, []string{"load32", "load32"}},
		{"independentParameterAssignment", `func (m *Module) reads(base, other int32) int32 {
 first := int32(load32(m.memory, uint64(uint32(base))+8))
 other = i32(0)
 last := int32(load32(m.memory, uint32(base)))
 return first ^ last ^ other
}`, []string{"load32", containedHelper}},
		{"loadedSiblingDoesNotKillBase", `func (m *Module) reads(base, other int32) int32 {
 first := int32(load32(m.memory, uint64(uint32(base))+8))
 other = int32(load32(m.memory, uint32(base)))
 last := int32(load32(m.memory, uint32(base)))
 return first ^ last ^ other
}`, []string{"load32", containedHelper, containedHelper}},
		{"independentVariableAssignment", `func (m *Module) reads(input int32) int32 {
 var base, other int32
 base = input
 first := int32(load32(m.memory, uint64(uint32(base))+8))
 other = i32(0)
 last := int32(load32(m.memory, uint32(base)))
 return first ^ last ^ other
}`, []string{"load32", containedHelper}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			output := rewriteViewFixture(t, viewFixture(t, test.method))
			assertContainedCalls(t, output, "reads", test.want...)
		})
	}
}

func TestContainedLoadEarlierEscapes(t *testing.T) {
	t.Parallel()
	for _, before := range []string{
		"address := &base\n _ = address",
		"go func() { base = -1 }()",
		"escaped := func() { base = -1 }\n _ = escaped",
	} {
		method := strings.Replace(containedReads, " first :=", before+"\n first :=", 1)
		output := rewriteViewFixture(t, viewFixture(t, method))
		assertContainedCalls(t, output, "reads", "load32", "load32")
	}
}

func TestContainedLoadUnsupportedBindings(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		method string
	}{
		{"wideBase", strings.Replace(containedReads, "base int32", "base int64", 1)},
		{"inferredBase", strings.Replace(containedReads, "reads(base int32) int32 {", "reads(input int32) int32 {\n base := input", 1)},
		{"namedBase", "type word int32\n" + strings.Replace(containedReads, "base int32", "base word", 1)},
		{"globalBase", "var base int32\n" + strings.Replace(containedReads, "base int32", "", 1)},
		{"globalDestination", "var last int32\n" + strings.Replace(containedReads, "last :=", "last =", 1)},
		{"blankDestination", strings.ReplaceAll(strings.Replace(containedReads, "last :=", "_ =", 1), "first ^ last", "first")},
		{"namedDestination", "type word int32\n" + strings.Replace(strings.Replace(containedReads, "last := int32(", "var last word\n last = word(", 1), "first ^ last", "first ^ int32(last)", 1)},
		{"wrappingOffset", strings.Replace(containedReads, "uint64(uint32(base))+8", "uint32(base+i32(8))", 1)},
		{"signedWidening", strings.Replace(containedReads, "uint64(uint32(base))+8", "uint64(base)+8", 1)},
		{"oversizedOffset", strings.Replace(containedReads, "+8", "+0x100000000", 1)},
		{"convertedOffset", strings.Replace(containedReads, "+8", "+uint64(8)", 1)},
		{"effectfulBase", strings.Replace(containedReads, "uint32(base)))", "uint32(m.Xi32_overflow())))", 1)},
		{"otherMemory", strings.Replace(strings.Replace(containedReads, "reads(base int32)", "reads(base int32, other []byte)", 1), "last := int32(load32(m.memory", "last := int32(load32(other", 1)},
		{"aliasMemory", strings.Replace(strings.Replace(containedReads, " first :=", " other := m.memory\n first :=", 1), "last := int32(load32(m.memory", "last := int32(load32(other", 1)},
		{"otherReceiver", strings.Replace(strings.Replace(containedReads, "reads(base int32)", "reads(base int32, other *Module)", 1), "last := int32(load32(m.memory", "last := int32(load32(other.memory", 1)},
		{"shadowedHelper", strings.Replace(strings.Replace(containedReads, " first :=", " load32 := func(mem []byte, addr uint64) uint32 { return 0 }\n first :=", 1), "uint32(base)))", "uint64(uint32(base))+0))", 1)},
		{"shadowedConversion", strings.Replace(containedReads, " first :=", " uint32 := func(value int32) uint32 { return 0 }\n first :=", 1)},
		{"shadowedIdentity", strings.Replace(strings.Replace(strings.Replace(containedReads, " first :=", " i32 := func(value int32) int32 { return 0 }\n first :=", 1), " last :=", " copied := i32(base)\n last :=", 1), "first ^ last", "first ^ last ^ copied", 1)},
		{"closureBody", `func (m *Module) reads(base int32) int32 {
 callback := func() int32 {
  first := int32(load32(m.memory, uint64(uint32(base))+8))
  last := int32(load32(m.memory, uint32(base)))
  return first ^ last
 }
 return callback()
}`},
		{"nestedEntry", `func (m *Module) reads(base int32) int32 {
 first := int32(load32(m.memory, uint64(uint32(base))+8))
 { copied := int32(load32(m.memory, uint32(base))); first ^= copied }
 last := int32(load32(m.memory, uint32(base)))
 return first ^ last
}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			output := rewriteViewFixture(t, viewFixture(t, test.method))
			if slices.Contains(containedCalls(t, output, "reads"), containedHelper) || bytes.Contains(output, []byte("func "+containedHelper+"[")) {
				t.Fatal("unsupported binding was transformed")
			}
		})
	}
}

func TestContainedLoadIdentityPurity(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		declaration string
		between     string
		before      string
		after       string
	}{
		{"", "copied := i32(base)", "func i32(x int32) int32 { return x }", "func i32(x int32) int32 { return x + 1 }"},
		{"", "copied := i64(int64(base))", "func i64(x int64) int64 { return x }", "func i64(x int64) int64 { return x + 1 }"},
		{"var copied int64", "copied = i64(int64(base))", "func i64(x int64) int64 { return x }", "func i64(x int64) int64 { return x + 1 }"},
	} {
		method := strings.Replace(containedReads, " last :=", test.between+"\n last :=", 1)
		if test.declaration != "" {
			method = strings.Replace(method, " first :=", test.declaration+"\n first :=", 1)
		}
		method = strings.Replace(method, "first ^ last", "first ^ last ^ int32(copied)", 1)
		source := viewFixture(t, method)
		output := rewriteViewFixture(t, source)
		assertContainedCalls(t, output, "reads", "load32", containedHelper)
		if !bytes.Contains(source, []byte(test.before)) {
			t.Fatalf("missing identity target %s", test.before)
		}
		changed := bytes.Replace(source, []byte(test.before), []byte(test.after), 1)
		if slices.Contains(containedCalls(t, rewriteViewFixture(t, changed), "reads"), containedHelper) {
			t.Fatal("impure identity wrapper retained range facts")
		}
	}
}

func TestContainedLoadReservedIdentifiers(t *testing.T) {
	t.Parallel()
	for _, declaration := range []string{
		"var cocoonLoad32InRange int32\n",
		"func cocoonLoad32InRange() {}\n",
		"type collision struct { cocoonLoad32InRange int32 }\n",
		"func other(cocoonLoad32InRange int32) {}\n",
		"func other() { cocoonLoad32InRange := 1; _ = cocoonLoad32InRange }\n",
		"func other() { cocoonLoad32InRange: for { break cocoonLoad32InRange } }\n",
	} {
		output := rewriteViewFixture(t, viewFixture(t, declaration+containedReads+publication))
		assertContainedCalls(t, output, "reads", "load32", "load32")
		if bytes.Count(output, []byte("store32(cocoonMemoryView1,")) != 2 {
			t.Fatal("contained-name collision disabled the retained publication pass")
		}
	}
	// String payload bytes do not collide with a Go identifier namespace.
	output := rewriteViewFixture(t, viewFixture(t, "const payload = \"cocoonLoad32InRange\"\n"+containedReads))
	assertContainedCalls(t, output, "reads", "load32", containedHelper)
}

func TestContainedLoadMandatoryGuards(t *testing.T) {
	t.Parallel()
	for _, mutation := range [][2]string{
		{"_ = mem[uint64(addr)+3]", "_ = mem[uint64(addr)+2]"},
		{"func i32(x int32) int32 { return x }", "func i32(x int32) int32 { return x + 1 }"},
		{"runtime.GOARCH == \"ppc64\"", "runtime.GOARCH == \"other\""},
		{"\"encoding/binary\"", "\"example.com/binary\""},
		{"\"unsafe\"", "native \"unsafe\""},
		{"m.memory = make([]byte, 65536)", "m.memory = unsafe.Slice((*byte)(unsafe.Pointer(m)),65536)"},
		{"return m\n}", "return new(Module)\n}"},
		{"return m\n}", "m = new(Module)\n return m\n}"},
		{"return m\n}", "p := &m\n *p = new(Module)\n return m\n}"},
		{"return m\n}", "p := m\n *p = Module{}\n return m\n}"},
		{"return m\n}", "defer func() { m.memory = nil }()\n return m\n}"},
		{"func New() *Module {", "func New() (result *Module) {"},
		{"func New() *Module {", "type uint32 = float64\n func New() *Module {"},
		{"//go:nosplit\nfunc load32", "//go:linkname load32 other\nfunc load32"},
		{"base int32", "base int32, uint32 func(int32) uint32"},
	} {
		source := viewFixture(t, containedReads)
		if !bytes.Contains(source, []byte(mutation[0])) {
			t.Fatalf("missing guard target %s", mutation[0])
		}
		changed := bytes.Replace(source, []byte(mutation[0]), []byte(mutation[1]), 1)
		if slices.Contains(containedCalls(t, rewriteViewFixture(t, changed), "reads"), containedHelper) {
			t.Fatalf("mandatory proof drift was accepted: %s", mutation[1])
		}
	}
}

func TestContainedLoadPublicationComposition(t *testing.T) {
	t.Parallel()
	method := strings.Replace(publication, " store32", ` high := int32(load32(receiver.memory, uint64(uint32(base))+8))
 low := int32(load32(receiver.memory, uint32(base)))
 value = high ^ low
 store32`, 1)
	output := rewriteViewFixture(t, viewFixture(t, method))
	assertContainedCalls(t, output, "publish", "load32", containedHelper)
	if bytes.Count(output, []byte("cocoonMemoryView1 = receiver.memory")) != 1 ||
		bytes.Count(output, []byte("store32(cocoonMemoryView1,")) != 2 ||
		bytes.Count(output, []byte("func "+containedHelper+"[")) != 1 {
		t.Fatal("contained helper disabled/duplicated the retained publication capture")
	}
}

func TestContainedLoadHelperPreservesFallback(t *testing.T) {
	t.Parallel()
	output := rewriteViewFixture(t, viewFixture(t, containedReads))
	file, err := parser.ParseFile(token.NewFileSet(), "contained.go", output, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	var original, unchecked *ast.FuncDecl
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv != nil {
			continue
		}
		switch function.Name.Name {
		case "load32":
			original = function
		case containedHelper:
			unchecked = function
		}
	}
	if original == nil || unchecked == nil {
		t.Fatal("missing checked or contained helper")
	}
	found := false
	if unchecked.Doc != nil {
		for _, comment := range unchecked.Doc.List {
			found = found || comment.Text == "//go:nosplit"
		}
	}
	if !found {
		t.Fatal("private helper lost its nosplit directive")
	}
	var statements []ast.Stmt
	removed := 0
	for _, statement := range original.Body.List {
		var encoded bytes.Buffer
		if err = format.Node(&encoded, token.NewFileSet(), statement); err != nil {
			t.Fatal(err)
		}
		if encoded.String() == "_ = mem[uint64(addr)+3]" {
			removed++
			continue
		}
		statements = append(statements, statement)
	}
	if removed != 1 {
		t.Fatalf("checked helper has %d endpoint guards", removed)
	}
	original.Body.List = statements
	var want, got bytes.Buffer
	if err = format.Node(&want, token.NewFileSet(), original.Type); err != nil {
		t.Fatal(err)
	}
	if err = format.Node(&want, token.NewFileSet(), original.Body); err != nil {
		t.Fatal(err)
	}
	if err = format.Node(&got, token.NewFileSet(), unchecked.Type); err != nil {
		t.Fatal(err)
	}
	if err = format.Node(&got, token.NewFileSet(), unchecked.Body); err != nil {
		t.Fatal(err)
	}
	if want.String() != got.String() {
		t.Fatalf("private helper changed more than the redundant unsafe guard\nwant:\n%s\ngot:\n%s", &want, &got)
	}
}
