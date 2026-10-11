package smoke_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"encoding/json"
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
	"dario.cat/cocoon/internal/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed testdata/memory-views/module_test.go.txt
var memoryViewModuleTests string

type viewEvent struct {
	Kind   uint32 `json:"kind"`
	Before uint32 `json:"before"`
	Pages  uint32 `json:"pages"`
}

type viewStep struct {
	Export string      `json:"export"`
	Trap   string      `json:"trap"`
	Args   []uint32    `json:"args"`
	Memory []byte      `json:"memory"`
	Events []viewEvent `json:"events"`
	Stack  uint32      `json:"stack"`
}

type viewHistory struct {
	Name  string     `json:"name"`
	Steps []viewStep `json:"steps"`
}

// TestMemoryViewSemantics executes the actual translated/hardened module. The
// CGO-disabled Go child is not race-instrumented by a race-enabled parent.
func TestMemoryViewSemantics(t *testing.T) {
	if os.Getenv("COCOON_SMOKE") != "1" {
		t.Skip("set COCOON_SMOKE=1 with the pinned Binaryen and wasm2go tools installed")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
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
	raw, optimized := filepath.Join(work, "raw.wasm"), filepath.Join(work, "module.wasm")
	arguments := append([]string{filepath.Join(root, "internal", "smoke", "testdata", "memory-views.wat"), "-o", raw}, build.Features()...)
	runNumeric(ctx, t, runner, root, "wasm-as", arguments...)
	arguments = append([]string{raw, "-O3", "-o", optimized}, build.Features()...)
	runNumeric(ctx, t, runner, root, "wasm-opt", arguments...)
	translatedPath := filepath.Join(work, "module.go")
	runNumeric(ctx, t, runner, root, goExecutable, "tool", "wasm2go", "-unsafe", "-pkg", "wasm", "-o", translatedPath, optimized)
	translated := read(t, translatedPath)
	required, err := harden.Required(translated)
	if err != nil {
		t.Fatal(err)
	}
	hardened, err := harden.Rewrite(translated, required)
	if err != nil {
		t.Fatal(err)
	}
	assertMemoryViewRegions(t, hardened)
	histories := memoryViewGoldens(ctx, t, read(t, optimized))
	encoded, err := json.Marshal(histories)
	if err != nil {
		t.Fatal(err)
	}
	write(t, translatedPath, string(hardened))
	write(t, filepath.Join(work, "go.mod"), "module example.com/memory-view-fixture\n\ngo 1.26.0\n")
	write(t, filepath.Join(work, "module_test.go"), memoryViewModuleTests)
	write(t, filepath.Join(work, "histories.json"), string(encoded))
	runNumeric(ctx, t, runner, work, goExecutable, "test", "-pgo=off", "-count=1", "-timeout=45s", "./...")
	checkMemoryViewMutants(ctx, t, runner, work, goExecutable, translatedPath, hardened)
}

func memoryViewRunner(work string) build.ExecRunner {
	environment := []string{"CGO_ENABLED=0", "GOWORK=off", "GOTOOLCHAIN=local", "GOENV=off", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off", "GOFLAGS=-buildvcs=false"}
	if os.Getenv("GOCACHE") == "" {
		environment = append(environment, "GOCACHE="+filepath.Join(work, "go-cache"))
	}
	return build.ExecRunner{Environment: environment}
}

func memoryViewTools(ctx context.Context, t *testing.T, runner build.ExecRunner, root, goExecutable string) {
	t.Helper()
	for _, program := range []string{"wasm-as", "wasm-opt"} {
		version := strings.Fields(string(runNumeric(ctx, t, runner, root, program, "--version")))
		if len(version) < 3 || version[0] != program || version[1] != "version" || version[2] != manifest.BinaryenVersion {
			t.Fatalf("%s requires version %s, got %v", program, manifest.BinaryenVersion, version)
		}
	}
	version := strings.Fields(string(runNumeric(ctx, t, runner, root, goExecutable, "tool", "wasm2go", "-version")))
	if len(version) != 2 || version[0] != "wasm2go" || version[1] != manifest.Wasm2GoVersion {
		t.Fatalf("wasm2go requires %s, got %v", manifest.Wasm2GoVersion, version)
	}
}

func memoryViewCases() []viewHistory {
	step := func(name string, args ...uint32) viewStep { return viewStep{Export: name, Args: args} }
	return []viewHistory{
		{Name: "aliases", Steps: []viewStep{
			step("publish", 32, 0x11223344, 0x55667788),
			step("copy_tuple", 32, 32, 0xaabbccdd),
			step("copy_tuple", 34, 36, 0x01234567),
			step("reverse", 33, 0x10203040, 0x50607080),
			step("mixed_tuple", 35, 33),
		}},
		{Name: "exact-end", Steps: []viewStep{step("publish", 65528, 0x11223344, 0x55667788)}},
		{Name: "trap-prefix", Steps: []viewStep{step("publish", 65532, 0x11223344, 0x55667788)}},
		{Name: "reverse-trap", Steps: []viewStep{step("reverse", 65532, 0x11223344, 0x55667788)}},
		{Name: "source-trap", Steps: []viewStep{step("copy_tuple", 0, 65534, 0x55667788)}},
		{Name: "partial-trap", Steps: []viewStep{step("mixed_tuple", 0, 65535)}},
		{Name: "negative-base", Steps: []viewStep{step("publish", 0xffffffff, 0x11223344, 0x55667788)}},
		{Name: "widened-base", Steps: []viewStep{step("reverse", 0xfffffffc, 0x11223344, 0x55667788)}},
		{Name: "wrapping-trap", Steps: []viewStep{step("wrapping_tuple", 0xfffffffc)}},
		{Name: "imports-and-growth", Steps: []viewStep{
			step("after_import", 1, 64), step("after_import", 2, 65536),
			step("grow_tuple", 131072, 1), step("grow_tuple", 80, 0), step("grow_tuple", 88, 8),
		}},
		{Name: "loop-growth", Steps: []viewStep{step("loop_tuples", 2, 65536, 2)}},
		{Name: "loop-observation", Steps: []viewStep{step("loop_tuples", 1, 0, 3)}},
		{Name: "zero-loop", Steps: []viewStep{step("loop_tuples", 2, 65536, 0)}},
		{Name: "conditional-entry", Steps: []viewStep{step("conditional_tuple", 0, 80), step("conditional_tuple", 1, 80)}},
	}
}

func memoryViewGoldens(ctx context.Context, t *testing.T, wasm []byte) []viewHistory {
	t.Helper()
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	t.Cleanup(func() {
		if err := runtime.Close(context.WithoutCancel(ctx)); err != nil {
			t.Error(err)
		}
	})
	var events []viewEvent
	_, err := runtime.NewHostModuleBuilder("env").NewFunctionBuilder().WithFunc(func(_ context.Context, module api.Module, kind uint32) {
		before, ok := module.Memory().ReadUint32Le(0)
		if !ok {
			t.Fatal("fixture import cannot observe memory")
		}
		events = append(events, viewEvent{Kind: kind, Before: before, Pages: module.Memory().Size() >> 16})
		if kind == 0 {
			return
		}
		if kind == 2 {
			if _, grown := module.Memory().Grow(1); !grown {
				t.Fatal("fixture import growth failed")
			}
		}
		count := len(events)
		if count > 1024 {
			t.Fatal("fixture import history exceeded its bound")
		}
		number := uint32(count) // #nosec G115 -- len(events) is nonnegative and bounded to 1024 above.
		if kind > 2 || !module.Memory().WriteUint32Le(32, 0x0a0b0c0d+number) {
			t.Fatal("unsupported fixture import", kind)
		}
	}).Export("touch").Instantiate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		t.Fatal(err)
	}
	histories := memoryViewCases()
	for index := range histories {
		events = nil
		module, instantiateErr := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig().WithName(histories[index].Name))
		if instantiateErr != nil {
			t.Fatal(instantiateErr)
		}
		for step := range histories[index].Steps {
			memoryViewGoldenStep(ctx, t, module, &histories[index].Steps[step])
			// Imports append to events during the operation, so snapshot afterward.
			histories[index].Steps[step].Events = append([]viewEvent(nil), events...)
		}
		assertMemoryViewOracle(t, histories[index])
		if closeErr := module.Close(ctx); closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	return histories
}

func memoryViewGoldenStep(ctx context.Context, t *testing.T, module api.Module, step *viewStep) {
	t.Helper()
	args := make([]uint64, len(step.Args))
	for index, value := range step.Args {
		args[index] = uint64(value)
	}
	function := module.ExportedFunction(step.Export)
	if function == nil {
		t.Fatal("missing fixture export", step.Export)
	}
	result, err := function.Call(ctx, args...)
	if err != nil {
		if !strings.Contains(err.Error(), "out of bounds memory access") {
			t.Fatal("unexpected fixture trap", err)
		}
		step.Trap = "memory"
	} else if len(result) != 0 {
		t.Fatal("unexpected fixture results", result)
	}
	memory, ok := module.Memory().Read(0, module.Memory().Size())
	if !ok {
		t.Fatal("fixture memory snapshot failed")
	}
	step.Memory = bytes.Clone(memory)
	stack, stackErr := module.ExportedFunction("stack_value").Call(ctx)
	if stackErr != nil || len(stack) != 1 {
		t.Fatal(stack, stackErr)
	}
	step.Stack = uint32(stack[0]) // #nosec G115 -- The validated export returns Wasm i32 bits.
}

func assertMemoryViewOracle(t *testing.T, history viewHistory) {
	t.Helper()
	step := history.Steps[0]
	switch history.Name {
	case "trap-prefix":
		if step.Trap != "memory" || binary.LittleEndian.Uint32(step.Memory[65532:]) != 0x11223344 || step.Stack != 49136 {
			t.Fatal("original Wasm did not preserve the first store before its trap")
		}
	case "reverse-trap":
		if step.Trap != "memory" || binary.LittleEndian.Uint32(step.Memory[65532:]) != 0 || step.Stack != 49136 {
			t.Fatal("original Wasm reordered the reverse-store trap")
		}
	case "wrapping-trap":
		if step.Trap != "memory" || binary.LittleEndian.Uint32(step.Memory) != 0x55667788 || step.Stack != 49136 {
			t.Fatal("original Wasm conflated widened and wrapping addresses")
		}
	case "widened-base":
		if step.Trap != "memory" || binary.LittleEndian.Uint32(step.Memory) != 0 || step.Stack != 49136 {
			t.Fatal("original Wasm truncated a widened address above MaxUint32")
		}
	case "partial-trap":
		if step.Trap != "memory" || step.Memory[65535] != 0xab || step.Stack != 49136 {
			t.Fatal("original Wasm lost its partial-width trap prefix")
		}
	}
}

func assertMemoryViewRegions(t *testing.T, source []byte) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "module.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{"Xpublish": true, "Xreverse": true, "Xcopy_tuple": true, "Xmixed_tuple": true, "Xafter_import": true, "Xgrow_tuple": true, "Xloop_tuples": true, "Xconditional_tuple": true}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function.Name.Name == "Xwrapping_tuple" {
			ast.Inspect(function.Body, func(node ast.Node) bool {
				if identifier, named := node.(*ast.Ident); named && strings.HasPrefix(identifier.Name, "cocoonMemoryView") {
					t.Fatal("wrapping address shape must remain ineligible")
				}
				return true
			})
		}
		if !wanted[function.Name.Name] {
			continue
		}
		stores := 0
		ast.Inspect(function.Body, func(node ast.Node) bool {
			call, called := node.(*ast.CallExpr)
			if called && len(call.Args) == 3 {
				name, named := call.Fun.(*ast.Ident)
				view, viewed := call.Args[0].(*ast.Ident)
				if named && name.Name == "store32" && viewed && strings.HasPrefix(view.Name, "cocoonMemoryView") {
					stores++
				}
			}
			return true
		})
		if stores != 2 {
			t.Fatalf("fixture method %s must execute two rewritten publication stores, got %d", function.Name.Name, stores)
		}
		delete(wanted, function.Name.Name)
	}
	if len(wanted) != 0 {
		t.Fatal("missing rewritten fixture methods", wanted)
	}
}

func checkMemoryViewMutants(ctx context.Context, t *testing.T, runner build.ExecRunner, work, goExecutable, path string, source []byte) {
	t.Helper()
	for _, mutation := range []struct {
		edit func(*ast.FuncDecl) bool
		name string
	}{
		{name: "coalesced stores", edit: coalesceMemoryViewStores},
		{name: "capture before import", edit: func(function *ast.FuncDecl) bool { return hoistMemoryViewCapture(function, "Xafter_import") }},
		{name: "capture above loop", edit: func(function *ast.FuncDecl) bool { return hoistMemoryViewCapture(function, "Xloop_tuples") }},
	} {
		// Mutate after production validation so source drift checks cannot stand in
		// for an executable oracle capable of detecting the wrong behavior.
		write(t, path, string(mutateMemoryView(t, source, mutation.edit)))
		output, err := runner.Run(ctx, work, goExecutable, "test", "-pgo=off", "-count=1", "-timeout=45s", "./...")
		if err == nil || !bytes.Contains(output, []byte("--- FAIL: TestMemoryView")) ||
			!bytes.Contains(output, []byte("memory-view")) || !bytes.Contains(output, []byte("mismatch")) {
			t.Fatalf("memory-view oracle did not reject %s: %v\n%s", mutation.name, err, output)
		}
	}
}

func mutateMemoryView(t *testing.T, source []byte, edit func(*ast.FuncDecl) bool) []byte {
	t.Helper()
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "module.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	changed := 0
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && edit(function) {
			changed++
		}
	}
	if changed != 1 {
		t.Fatalf("semantic mutation changed %d methods, want one", changed)
	}
	var output bytes.Buffer
	if err := format.Node(&output, files, file); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func coalesceMemoryViewStores(function *ast.FuncDecl) bool {
	if function.Name.Name != "Xpublish" {
		return false
	}
	for index := 0; index+1 < len(function.Body.List); index++ {
		first := memoryViewStore(function.Body.List[index])
		second := memoryViewStore(function.Body.List[index+1])
		if first == nil || second == nil {
			continue
		}
		word := func(value ast.Expr) ast.Expr {
			return &ast.CallExpr{Fun: ast.NewIdent("uint64"), Args: []ast.Expr{value}}
		}
		first.Fun = ast.NewIdent("store64")
		first.Args[2] = &ast.BinaryExpr{X: word(first.Args[2]), Op: token.OR, Y: &ast.BinaryExpr{X: word(second.Args[2]), Op: token.SHL, Y: &ast.BasicLit{Kind: token.INT, Value: "32"}}}
		function.Body.List = append(function.Body.List[:index+1], function.Body.List[index+2:]...)
		return true
	}
	return false
}

func memoryViewStore(statement ast.Stmt) *ast.CallExpr {
	expression, ok := statement.(*ast.ExprStmt)
	if !ok {
		return nil
	}
	call, ok := expression.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 3 {
		return nil
	}
	name, named := call.Fun.(*ast.Ident)
	view, viewed := call.Args[0].(*ast.Ident)
	if !named || name.Name != "store32" || !viewed || !strings.HasPrefix(view.Name, "cocoonMemoryView") {
		return nil
	}
	return call
}

func hoistMemoryViewCapture(function *ast.FuncDecl, name string) bool {
	if function.Name.Name != name {
		return false
	}
	var captures []ast.Stmt
	ast.Inspect(function.Body, func(node ast.Node) bool {
		block, ok := node.(*ast.BlockStmt)
		if !ok {
			return true
		}
		kept := make([]ast.Stmt, 0, len(block.List))
		for _, statement := range block.List {
			assignment, assigned := statement.(*ast.AssignStmt)
			if assigned && len(assignment.Lhs) == 1 && len(assignment.Rhs) == 1 {
				view, viewed := assignment.Lhs[0].(*ast.Ident)
				memory, selected := assignment.Rhs[0].(*ast.SelectorExpr)
				if viewed && strings.HasPrefix(view.Name, "cocoonMemoryView") && selected && memory.Sel.Name == "memory" {
					captures = append(captures, statement)
					continue
				}
			}
			kept = append(kept, statement)
		}
		block.List = kept
		return true
	})
	if len(captures) != 1 {
		return false
	}
	entry := 0
	for entry < len(function.Body.List) {
		if _, declaration := function.Body.List[entry].(*ast.DeclStmt); !declaration {
			break
		}
		entry++
	}
	function.Body.List = append(function.Body.List[:entry], append(captures, function.Body.List[entry:]...)...)
	return true
}
