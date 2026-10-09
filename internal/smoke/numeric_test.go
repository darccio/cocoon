package smoke_test

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"math"
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

//go:embed testdata/numeric/module_test.go.txt
var numericModuleTests string

type numericCase struct {
	Name string `json:"name"`
	Bits uint64 `json:"bits"`
	Want int32  `json:"want"`
}

// TestNumericLoweringSemantics executes actual translated and hardened Go, not
// an authored replacement for the production numeric helper. Its CGO-disabled
// child is not race-instrumented by a parent invocation using the race detector.
func TestNumericLoweringSemantics(t *testing.T) {
	if os.Getenv("COCOON_SMOKE") != "1" {
		t.Skip("set COCOON_SMOKE=1 with the pinned Binaryen and wasm2go tools installed")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
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
	environment := []string{"CGO_ENABLED=0", "GOWORK=off", "GOTOOLCHAIN=local", "GOENV=off", "GOPROXY=off", "GOSUMDB=off", "GOVCS=*:off", "GOFLAGS=-buildvcs=false"}
	if os.Getenv("GOCACHE") == "" {
		environment = append(environment, "GOCACHE="+filepath.Join(work, "go-cache"))
	}
	// Keep the inherited/default module cache: the offline pinned go tool must
	// reuse the wasm2go dependencies already downloaded by the test setup.
	runner := build.ExecRunner{Environment: environment}
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
	raw, optimized := filepath.Join(work, "raw.wasm"), filepath.Join(work, "module.wasm")
	arguments := append([]string{filepath.Join(root, "internal", "smoke", "testdata", "numeric.wat"), "-o", raw}, build.Features()...)
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
	cases := numericCases()
	encoded, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	write(t, translatedPath, string(hardened))
	write(t, filepath.Join(work, "go.mod"), "module example.com/numeric-fixture\n\ngo 1.26.0\n")
	write(t, filepath.Join(work, "module_test.go"), numericModuleTests)
	write(t, filepath.Join(work, "cases.json"), string(encoded))
	runNumeric(ctx, t, runner, work, goExecutable, "test", "-pgo=off", "-count=1", "-timeout=45s", "./...")

	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	t.Cleanup(func() {
		if closeErr := runtime.Close(context.WithoutCancel(t.Context())); closeErr != nil {
			t.Error(closeErr)
		}
	})
	module, err := runtime.Instantiate(ctx, read(t, optimized))
	if err != nil {
		t.Fatal(err)
	}
	checkNumericInterpreter(ctx, t, module, cases)

	// Mutation occurs after rewriting: fail-closed source validation must not
	// mask whether the executable oracle actually detects incorrect semantics.
	write(t, translatedPath, string(breakNumericHelper(t, hardened)))
	output, mutationErr := runner.Run(ctx, work, goExecutable, "test", "-pgo=off", "-count=1", "-timeout=45s", "-run", "^TestNumericFloorSat$", "./...")
	if mutationErr == nil || !bytes.Contains(output, []byte("floor-sat mismatch")) {
		t.Fatalf("numeric oracle did not reject a broken helper: %v\n%s", mutationErr, output)
	}
}

func runNumeric(ctx context.Context, t *testing.T, runner build.ExecRunner, directory, program string, arguments ...string) []byte {
	t.Helper()
	output, err := runner.Run(ctx, directory, program, arguments...)
	if err != nil {
		t.Fatal(err)
	}
	return output
}

func numericCases() []numericCase {
	cases := []numericCase{
		{"positive zero", 0, 0},
		{"negative zero", 1 << 63, 0},
		{"positive subnormal", 1, 0},
		{"negative subnormal", 1<<63 | 1, -1},
		{"positive largest subnormal", 0x000f_ffff_ffff_ffff, 0},
		{"negative largest subnormal", 0x800f_ffff_ffff_ffff, -1},
		{"positive smallest normal", 0x0010_0000_0000_0000, 0},
		{"negative smallest normal", 0x8010_0000_0000_0000, -1},
		{"positive fraction", math.Float64bits(0.75), 0},
		{"negative fraction", math.Float64bits(-0.75), -1},
		{"positive integer fraction", math.Float64bits(1.75), 1},
		{"negative integer fraction", math.Float64bits(-1.75), -2},
		{"below one", math.Float64bits(math.Nextafter(1, math.Inf(-1))), 0},
		{"one", math.Float64bits(1), 1},
		{"above one", math.Float64bits(math.Nextafter(1, math.Inf(1))), 1},
		{"below negative one", math.Float64bits(math.Nextafter(-1, math.Inf(-1))), -2},
		{"negative one", math.Float64bits(-1), -1},
		{"above negative one", math.Float64bits(math.Nextafter(-1, math.Inf(1))), -1},
		{"below maximum integer", math.Float64bits(math.Nextafter(math.MaxInt32, math.Inf(-1))), math.MaxInt32 - 1},
		{"maximum integer", math.Float64bits(math.MaxInt32), math.MaxInt32},
		{"above maximum integer", math.Float64bits(math.Nextafter(math.MaxInt32, math.Inf(1))), math.MaxInt32},
		{"maximum integer plus fraction", math.Float64bits(math.MaxInt32 + 0.75), math.MaxInt32},
		{"below upper clamp", math.Float64bits(math.Nextafter(0x1p31, math.Inf(-1))), math.MaxInt32},
		{"upper clamp", math.Float64bits(0x1p31), math.MaxInt32},
		{"above upper clamp", math.Float64bits(math.Nextafter(0x1p31, math.Inf(1))), math.MaxInt32},
		{"below lower clamp", math.Float64bits(math.Nextafter(math.MinInt32, math.Inf(-1))), math.MinInt32},
		{"lower clamp", math.Float64bits(math.MinInt32), math.MinInt32},
		{"above lower clamp", math.Float64bits(math.Nextafter(math.MinInt32, math.Inf(1))), math.MinInt32},
		{"largest positive finite", math.Float64bits(math.MaxFloat64), math.MaxInt32},
		{"largest negative finite", math.Float64bits(-math.MaxFloat64), math.MinInt32},
		{"positive infinity", math.Float64bits(math.Inf(1)), math.MaxInt32},
		{"negative infinity", math.Float64bits(math.Inf(-1)), math.MinInt32},
	}
	for _, bits := range []uint64{0x7ff0_0000_0000_0001, 0x7ff7_ffff_ffff_ffff, 0x7ff8_0000_0000_0000, 0x7fff_ffff_ffff_ffff} {
		cases = append(cases, numericCase{"positive NaN", bits, 0}, numericCase{"negative NaN", bits | 1<<63, 0})
	}
	return cases
}

func checkNumericInterpreter(ctx context.Context, t *testing.T, module api.Module, cases []numericCase) {
	t.Helper()
	for _, test := range cases {
		want := uint64(uint32(test.Want)) // #nosec G115 -- Compare the Wasm i32 result's unsigned bits.
		if got := numericResult(ctx, t, module, "floor_sat", test.Bits); got != want {
			t.Fatalf("floor-sat mismatch for %s: %08x, want %08x", test.Name, got, want)
		}
		if _, err := module.ExportedFunction("reset").Call(ctx); err != nil {
			t.Fatal(err)
		}
		if got := numericResult(ctx, t, module, "floor_effect", test.Bits); got != want || numericResult(ctx, t, module, "counter") != 1 {
			t.Fatal("floor-sat argument was not evaluated exactly once", test.Name)
		}
	}
	if got := numericResult(ctx, t, module, "f64_unfused", math.Float64bits(1+0x1p-27), math.Float64bits(1-0x1p-27), math.Float64bits(-1)); got != 0 {
		t.Fatalf("f64 multiply/add rounding changed: %016x", got)
	}
	if got := numericResult(ctx, t, module, "f32_unfused", uint64(math.Float32bits(1+0x1p-13)), uint64(math.Float32bits(1-0x1p-13)), uint64(math.Float32bits(-1))); got != 0 {
		t.Fatalf("f32 multiply/add rounding changed: %08x", got)
	}
	if got := numericResult(ctx, t, module, "f64_unfused", 1<<63, math.Float64bits(1), 1<<63); got != 1<<63 {
		t.Fatalf("signed zero rounding changed: %016x", got)
	}
	if got := numericResult(ctx, t, module, "rounding_order", math.Float64bits(1)); got != 0 {
		t.Fatalf("floating additions reassociated: %016x", got)
	}
	for _, test := range []struct{ input, want uint64 }{{1, 0}, {3, 2}} {
		if got := numericResult(ctx, t, module, "subnormal_half", test.input); got != test.want {
			t.Fatalf("subnormal rounding changed: %016x, want %016x", got, test.want)
		}
	}
	for _, test := range []struct {
		name    string
		args    []uint64
		counter uint64
	}{
		{"floor_trap", []uint64{65536}, 7},
		{"effect_then_trap", []uint64{65536, math.Float64bits(1.75)}, 1},
	} {
		if _, err := module.ExportedFunction("reset").Call(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := module.ExportedFunction(test.name).Call(ctx, test.args...); err == nil {
			t.Fatal("numeric argument lost its bounds trap", test.name)
		}
		if got := numericResult(ctx, t, module, "counter"); got != test.counter {
			t.Fatal("numeric argument trap reordered effects", test.name, got)
		}
		if got := numericResult(ctx, t, module, "floor_sat", math.Float64bits(1.75)); got != 1 {
			t.Fatal("numeric trap made the module unusable")
		}
	}
	if got := numericResult(ctx, t, module, "floor_trap", 8); got != 0 || numericResult(ctx, t, module, "counter") != 9 {
		t.Fatal("valid numeric argument did not complete its following effect")
	}
}

func numericResult(ctx context.Context, t *testing.T, module api.Module, name string, args ...uint64) uint64 {
	t.Helper()
	function := module.ExportedFunction(name)
	if function == nil {
		t.Fatal("missing numeric export", name)
	}
	result, err := function.Call(ctx, args...)
	if err != nil || len(result) != 1 {
		t.Fatalf("%s: %v: %v", name, result, err)
	}
	return result[0]
}

func breakNumericHelper(t *testing.T, source []byte) []byte {
	t.Helper()
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "module.go", source, 0)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Recv == nil && function.Name.Name == "i32_trunc_sat_f64_s" {
			count++
			function.Body = &ast.BlockStmt{List: []ast.Stmt{&ast.ReturnStmt{Results: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: "0"}}}}}
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one generated saturation helper, got %d", count)
	}
	var output bytes.Buffer
	if err := format.Node(&output, files, file); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
