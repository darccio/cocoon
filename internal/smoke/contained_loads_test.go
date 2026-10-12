package smoke_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dario.cat/cocoon/internal/build"
	"dario.cat/cocoon/internal/harden"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed testdata/contained-loads/module_test.go.txt
var containedLoadModuleTests string

const containedNativeMethods = `
var containedBase int32
func (m *Module) XshadowFunc(v0 int32) int32 {
 i32 := m._probe.Ximpure
 a := int32(load32(m.memory, uint64(uint32(v0))+8))
 x := i32(v0)
 b := int32(load32(m.memory, uint32(v0)))
 return a*3+b*5+x
}
func (m *Module) XglobalBase() int32 {
 a := int32(load32(m.memory, uint64(uint32(containedBase))+8))
 b := int32(load32(m.memory, uint32(containedBase)))
 return a*3+b*5
}
func (m *Module) Xlabel(v0, flag int32) int32 {
 var a, b int32
 if flag == 0 { goto lower }
 a = int32(load32(m.memory, uint64(uint32(v0))+8))
lower:
 b = int32(load32(m.memory, uint32(v0)))
 return a*3+b*5
}
`

type containedEvent struct {
	Kind   uint32 `json:"kind"`
	Before uint32 `json:"before"`
	Length uint32 `json:"length"`
}

type containedCase struct {
	Export string           `json:"export"`
	Trap   string           `json:"trap"`
	Memory []byte           `json:"memory"`
	Events []containedEvent `json:"events"`
	Base   uint32           `json:"base"`
	Flag   uint32           `json:"flag"`
	Result uint32           `json:"result"`
	Prefix uint32           `json:"prefix"`
}

func TestContainedLoadSemantics(t *testing.T) {
	if os.Getenv("COCOON_SMOKE") != "1" {
		t.Skip("set COCOON_SMOKE=1 with the pinned Binaryen and wasm2go tools installed")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	goExecutable, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	runner := memoryViewRunner(work)
	memoryViewTools(ctx, t, runner, root, goExecutable)
	wasmPath, translatedPath := filepath.Join(work, "module.wasm"), filepath.Join(work, "module.go")
	// Keep explicit load32 sites: optimizer coalescing is not proof of this pass.
	arguments := append([]string{filepath.Join(root, "internal", "smoke", "testdata", "contained-loads.wat"), "-o", wasmPath}, build.Features()...)
	runNumeric(ctx, t, runner, root, "wasm-as", arguments...)
	runNumeric(ctx, t, runner, root, goExecutable, "tool", "wasm2go", "-unsafe", "-pkg", "wasm", "-o", translatedPath, wasmPath)
	translated := append(bytes.Clone(read(t, translatedPath)), containedNativeMethods...)
	required, err := harden.Required(translated)
	if err != nil {
		t.Fatal(err)
	}
	hardened, err := harden.Rewrite(translated, required)
	if err != nil {
		t.Fatal(err)
	}
	assertContainedLoadRegions(t, hardened)
	encoded, err := json.Marshal(containedLoadGoldens(ctx, t, read(t, wasmPath)))
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(work, "go.mod"), "module example.com/contained-load-fixture\n\ngo 1.26.0\n")
	write(t, filepath.Join(work, "module_test.go"), containedLoadModuleTests)
	write(t, filepath.Join(work, "cases.json"), string(encoded))
	for _, role := range []struct {
		name   string
		source []byte
	}{
		{"original", translated},
		{"hardened", hardened},
		{"recheck", recheckContainedLoad(t, hardened)},
		{"checked-fallback", fallbackContainedLoad(t, hardened)},
	} {
		t.Run(role.name, func(t *testing.T) {
			write(t, translatedPath, string(role.source))
			runNumeric(ctx, t, runner, work, goExecutable, "test", "-pgo=off", "-count=1", "-timeout=45s", "./...")
		})
	}
	checkContainedLoadMutants(ctx, t, runner, work, goExecutable, translatedPath, hardened)
}

func containedFixtureBytes(length int) []byte {
	data := make([]byte, length)
	for index := range data {
		data[index] = byte(index*7 + 3)
	} // #nosec G115 -- The fixture deliberately cycles all byte values.
	return data
}

func containedLoadBases() []uint32 {
	values := []uint32{0, 1, 2, 3, 4, 0x7fffffff, 0xfffffff0, 0xfffffffc, 0xffffffff}
	for distance := range 17 {
		values = append(values, 65536-uint32(distance)) // #nosec G115 -- distance is bounded to 0–16.
	}
	return values
}

func containedLoadGoldens(ctx context.Context, t *testing.T, wasm []byte) []containedCase {
	t.Helper()
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	t.Cleanup(func() {
		if err := runtime.Close(context.WithoutCancel(ctx)); err != nil {
			t.Error(err)
		}
	})
	var events []containedEvent
	_, err := runtime.NewHostModuleBuilder("probe").NewFunctionBuilder().WithFunc(func(_ context.Context, module api.Module) {
		before, ok := module.Memory().ReadUint32Le(0)
		if !ok {
			t.Fatal("contained-load import memory unavailable")
		}
		events = append(events, containedEvent{1, before, module.Memory().Size()})
		if !module.Memory().WriteUint32Le(0, 0x11223345) {
			t.Fatal("contained-load import write failed")
		}
	}).Export("touch").NewFunctionBuilder().WithFunc(func(value uint32) uint32 { return value }).Export("impure").Instantiate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		t.Fatal(err)
	}
	var cases []containedCase
	for _, name := range []string{"covered", "triple", "rising", "rebind", "wrapped", "imported", "stored", "conditional", "grown", "two_bases"} {
		for _, base := range containedLoadBases() {
			for _, flag := range []uint32{0, 1} {
				events = nil
				entry := containedCase{Export: name, Base: base, Flag: flag}
				module, instantiateErr := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(fmt.Sprintf("%s-%x-%d", name, base, flag)))
				if instantiateErr != nil {
					t.Fatal(instantiateErr)
				}
				if !module.Memory().Write(0, containedFixtureBytes(65536)) {
					t.Fatal("contained-load seed failed")
				}
				containedLoadGoldenStep(ctx, t, module, &entry)
				entry.Events = append([]containedEvent(nil), events...)
				assertContainedGolden(t, &entry)
				cases = append(cases, entry)
				if closeErr := module.Close(ctx); closeErr != nil {
					t.Fatal(closeErr)
				}
			}
		}
	}
	return cases
}

func containedLoadGoldenStep(ctx context.Context, t *testing.T, module api.Module, entry *containedCase) {
	t.Helper()
	args := []uint64{uint64(entry.Base)}
	if entry.Export == "conditional" || entry.Export == "two_bases" {
		args = append(args, uint64(entry.Flag))
	}
	function := module.ExportedFunction(entry.Export)
	if function == nil {
		t.Fatal("missing contained-load export", entry.Export)
	}
	result, err := function.Call(ctx, args...)
	if err != nil {
		if !strings.Contains(err.Error(), "out of bounds memory access") {
			t.Fatal("unexpected contained-load Wasm trap", err)
		}
		entry.Trap = "memory"
	} else {
		if len(result) != 1 {
			t.Fatal("contained-load result arity mismatch")
		}
		entry.Result = api.DecodeU32(result[0])
	}
	prefix, err := module.ExportedFunction("prefix").Call(ctx)
	if err != nil || len(prefix) != 1 {
		t.Fatal("contained-load prefix unavailable", prefix, err)
	}
	entry.Prefix = api.DecodeU32(prefix[0])
	view, ok := module.Memory().Read(0, module.Memory().Size())
	if !ok {
		t.Fatal("contained-load full memory unavailable")
	}
	entry.Memory = bytes.Clone(view)
}

func assertContainedGolden(t *testing.T, entry *containedCase) {
	t.Helper()
	if entry.Prefix != 1 {
		t.Fatal("original Wasm lost contained-load observable prefix")
	}
	if entry.Base != 0 || entry.Flag != 1 {
		return
	}
	data := containedFixtureBytes(32)
	low, middle, high := binary.LittleEndian.Uint32(data), binary.LittleEndian.Uint32(data[4:]), binary.LittleEndian.Uint32(data[8:])
	want := high*3 + low*5
	switch entry.Export {
	case "triple":
		want = middle*3 + high*5 + low*7
	case "rising":
		want = low*3 + middle*5 + high*7
	case "rebind":
		want = high*3 + binary.LittleEndian.Uint32(data[16:])*5
	case "wrapped":
		if entry.Trap != "memory" {
			t.Fatal("original Wasm wrapped widened address unexpectedly")
		}
		return
	case "imported":
		want = high*3 + uint32(0x11223345)*5
	case "stored":
		want = high*3 + 999*5
	case "two_bases":
		want = high*3 + binary.LittleEndian.Uint32(data[1:])*5
	}
	if entry.Trap != "" || entry.Result != want {
		t.Fatal("original Wasm weighted golden mismatch", entry.Export, entry.Result, want, entry.Trap)
	}
}

func assertContainedLoadRegions(t *testing.T, source []byte) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "module.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]int{"Xcovered": 1, "Xtriple": 1, "Xrising": 0, "Xrebind": 0, "Xwrapped": 0, "Ximported": 0, "Xstored": 0, "Xconditional": 0, "Xgrown": 0, "Xtwo_bases": 0, "XshadowFunc": 0, "XglobalBase": 0, "Xlabel": 0}
	helpers := 0
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function.Name.Name == "cocoonLoad32InRange" {
			helpers++
		}
		want, required := wanted[function.Name.Name]
		if !required {
			continue
		}
		if function.Name.Name == "Xtwo_bases" && (len(function.Type.Params.List) != 1 || len(function.Type.Params.List[0].Names) != 2) {
			t.Fatal("contained-load two-base fixture must exercise grouped parameter bindings")
		}
		count := 0
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if call, ok := node.(*ast.CallExpr); ok {
				if name, ok := call.Fun.(*ast.Ident); ok && name.Name == "cocoonLoad32InRange" {
					if !containedSelectedAddress(call, function) {
						t.Fatal("contained-load selected the wrong read or address", function.Name.Name)
					}
					count++
				}
			}
			return true
		})
		if count != want {
			t.Fatalf("contained-load method %s selected %d reads, want %d; ownership proof must not skip the fixture", function.Name.Name, count, want)
		}
		delete(wanted, function.Name.Name)
	}
	if helpers != 1 || len(wanted) != 0 {
		t.Fatal("contained-load missing production helper or fixture methods", helpers, wanted)
	}
}

func containedSelectedAddress(call *ast.CallExpr, function *ast.FuncDecl) bool {
	if len(call.Args) != 2 || function.Type.Params == nil || len(function.Type.Params.List) == 0 || len(function.Type.Params.List[0].Names) == 0 {
		return false
	}
	address, ok := call.Args[1].(*ast.CallExpr)
	if !ok || len(address.Args) != 1 {
		return false
	}
	conversion, ok := address.Fun.(*ast.Ident)
	if !ok || conversion.Name != "uint32" || conversion.Obj != nil {
		return false
	}
	base, ok := address.Args[0].(*ast.Ident)
	return ok && base.Obj == function.Type.Params.List[0].Names[0].Obj
}

func fallbackContainedLoad(t *testing.T, source []byte) []byte {
	t.Helper()
	return mutateContainedLoad(t, source, func(file *ast.File) bool {
		for _, declaration := range file.Decls {
			group, ok := declaration.(*ast.GenDecl)
			if !ok || group.Tok != token.CONST {
				continue
			}
			for _, specification := range group.Specs {
				value, ok := specification.(*ast.ValueSpec)
				if ok && len(value.Names) == 1 && value.Names[0].Name == "unalignedOK" && len(value.Values) == 1 {
					value.Values[0] = ast.NewIdent("false")
					return true
				}
			}
		}
		return false
	})
}

func recheckContainedLoad(t *testing.T, source []byte) []byte {
	t.Helper()
	return mutateContainedLoad(t, source, func(file *ast.File) bool {
		var original, unchecked *ast.FuncDecl
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok {
				switch function.Name.Name {
				case "load32":
					original = function
				case "cocoonLoad32InRange":
					unchecked = function
				}
			}
		}
		if original == nil || unchecked == nil {
			return false
		}
		unchecked.Body = original.Body
		return true
	})
}

func mutateContainedLoad(t *testing.T, source []byte, edit func(*ast.File) bool) []byte {
	t.Helper()
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "module.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	if !edit(file) {
		t.Fatal("contained-load semantic mutation did not find its target")
	}
	var output bytes.Buffer
	if err := format.Node(&output, files, file); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func uncheckedContainedRead(file *ast.File, method string, target int) bool {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Name.Name != method {
			continue
		}
		read, changed := 0, false
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			name, ok := call.Fun.(*ast.Ident)
			if !ok || name.Name != "load32" && name.Name != "cocoonLoad32InRange" {
				return true
			}
			if read == target {
				call.Fun = ast.NewIdent("cocoonLoad32InRange")
				changed = true
			}
			read++
			return true
		})
		return changed
	}
	return false
}

func checkContainedLoadMutants(ctx context.Context, t *testing.T, runner build.ExecRunner, work, goExecutable, path string, source []byte) {
	t.Helper()
	for _, mutant := range []struct {
		name, method, test string
		read               int
	}{
		{"first-check deletion", "Xcovered", "TestContainedLoadLogicalLength", 0},
		{"import crossing", "Ximported", "TestContainedLoadHeaderBarrier", 1},
		{"name-only purity", "XshadowFunc", "TestContainedLoadShadowBarrier", 1},
	} {
		// Mutate after validation: binding/drift rejection is not an executable proof.
		modified := mutateContainedLoad(t, source, func(file *ast.File) bool { return uncheckedContainedRead(file, mutant.method, mutant.read) })
		write(t, path, string(modified))
		output, err := runner.Run(ctx, work, goExecutable, "test", "-pgo=off", "-count=1", "-timeout=45s", "-run", "^"+mutant.test+"$", "./...")
		if err == nil || !bytes.Contains(output, []byte("--- FAIL: "+mutant.test)) || !bytes.Contains(output, []byte("contained-load")) || !bytes.Contains(output, []byte("mismatch")) {
			t.Fatalf("contained-load runtime proof did not reject %s: %v\n%s", mutant.name, err, output)
		}
		t.Logf("contained-load semantic mutant %q rejected by %s", mutant.name, mutant.test)
	}
}
