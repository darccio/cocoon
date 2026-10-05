package smoke_test

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/darccio/cocoon/internal/build"
	"github.com/darccio/cocoon/internal/manifest"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// TestOptimizerSemantics checks optimizer behavior against explicit Wasm contracts,
// independently of translating the same optimized artifact into Go.
func TestOptimizerSemantics(t *testing.T) {
	if os.Getenv("COCOON_SMOKE") != "1" {
		t.Skip("set COCOON_SMOKE=1 with the pinned Binaryen tools installed")
	}
	for _, program := range []string{"wasm-as", "wasm-opt"} {
		version := strings.Fields(string(run(t, ".", nil, program, "--version")))
		if len(version) < 3 || version[0] != program || version[1] != "version" || version[2] != manifest.BinaryenVersion {
			t.Fatalf("%s requires version %s, got %v", program, manifest.BinaryenVersion, version)
		}
	}
	work := t.TempDir()
	raw := filepath.Join(work, "input.wasm")
	arguments := append([]string{"testdata/optimizer.wat", "-o", raw}, build.Features()...)
	run(t, ".", nil, "wasm-as", arguments...)
	for _, variant := range []struct {
		name string
		args []string
	}{
		{"O3", []string{"-O3"}},
		{"O4", []string{"-O4"}},
		{"O3-no-stack-ir", []string{"-O3", "--no-stack-ir"}},
		{"O3-converge", []string{"-O3", "--converge"}},
	} {
		t.Run(variant.name, func(t *testing.T) {
			optimized := filepath.Join(work, variant.name+".wasm")
			arguments := append([]string{raw}, variant.args...)
			arguments = append(arguments, "-o", optimized)
			arguments = append(arguments, build.Features()...)
			run(t, ".", nil, "wasm-opt", arguments...)
			runtime := wazero.NewRuntimeWithConfig(t.Context(), wazero.NewRuntimeConfigInterpreter())
			t.Cleanup(func() {
				if err := runtime.Close(context.WithoutCancel(t.Context())); err != nil {
					t.Error(err)
				}
			})
			module, err := runtime.Instantiate(t.Context(), read(t, optimized))
			if err != nil {
				t.Fatal(err)
			}
			checkOptimizerNumbers(t, module)
			checkOptimizerTraps(t, module)
		})
	}
}

func checkOptimizerNumbers(t *testing.T, module api.Module) {
	t.Helper()
	for _, test := range []struct {
		bits  uint64
		valid bool
	}{
		{0, true},
		{1, true},
		{0x000f_ffff_ffff_ffff, true},
		{0x0010_0000_0000_0000, true},
		{math.Float64bits(math.MaxFloat64), true},
		{math.Float64bits(math.Inf(1)), false},
		{0x7ff0_0000_0000_0001, false},
		{0x7ff7_ffff_ffff_ffff, false},
		{0x7ff8_0000_0000_0000, false},
		{0x7fff_ffff_ffff_ffff, false},
	} {
		for _, sign := range []uint64{0, 1 << 63} {
			bits := test.bits | sign
			var want uint64
			if test.valid && (sign == 0 || test.bits == 0) {
				want = 1
			}
			if got := optimizerResult(t, module, "finite_nonnegative", bits); got != want {
				t.Fatalf("ordered domain for %016x: got %d, want %d", bits, got, want)
			}
			if got := optimizerResult(t, module, "bits", bits); got != bits {
				t.Fatalf("transport changed bits: got %016x, want %016x", got, bits)
			}
		}
	}
	for _, zero := range []uint64{0, 1 << 63} {
		if got := optimizerResult(t, module, "min_zero", zero); got != zero {
			t.Fatalf("min zero: got %016x, want %016x", got, zero)
		}
		if got := optimizerResult(t, module, "max_zero", zero); got != 0 {
			t.Fatalf("max zero changed sign: %016x", got)
		}
	}
	if got := optimizerResult(t, module, "rounding_order", math.Float64bits(1)); got != 0 {
		t.Fatalf("floating operations reassociated: %016x", got)
	}
}

func checkOptimizerTraps(t *testing.T, module api.Module) {
	t.Helper()
	const end = 65536
	if got := optimizerResult(t, module, "guarded_load", 0, end); got != 42 {
		t.Fatal("unselected trapping branch executed")
	}
	if got := optimizerResult(t, module, "eager_select", 1, 8); got != 42 {
		t.Fatal("select returned the wrong alternative")
	}
	if got := optimizerResult(t, module, "eager_select", 0, 8); got != 0 {
		t.Fatal("select ignored its condition")
	}
	if got := optimizerResult(t, module, "operand_order"); got != 6 {
		t.Fatal("nested operand values changed")
	}
	if got, ok := module.Memory().ReadUint32Le(8); !ok || got != 123 {
		t.Fatalf("nested operand evaluation order changed: got %d", got)
	}
	if got := optimizerResult(t, module, "divide", 4, 2); got != 2 {
		t.Fatal("valid division changed")
	}
	if got := optimizerResult(t, module, "truncate", math.Float64bits(1.75)); got != 1 {
		t.Fatal("valid float truncation changed")
	}
	if _, err := module.ExportedFunction("empty_fill").Call(t.Context(), end); err != nil {
		t.Fatal("zero-length fill at memory end trapped", err)
	}
	for _, test := range []struct {
		name string
		args []uint64
	}{
		{"guarded_load", []uint64{1, end}},
		// Unlike an if branch, select evaluates an unselected operand too.
		{"eager_select", []uint64{1, end}},
		{"store_then_trap", []uint64{end}},
		{"trap_then_store", []uint64{end}},
		{"divide", []uint64{1, 0}},
		{"divide", []uint64{1 << 31, math.MaxUint32}},
		{"truncate", []uint64{math.Float64bits(math.NaN())}},
		{"truncate", []uint64{math.Float64bits(math.Inf(1))}},
		{"truncate", []uint64{math.Float64bits(1 << 31)}},
		{"empty_fill", []uint64{end + 1}},
	} {
		if _, err := module.ExportedFunction(test.name).Call(t.Context(), test.args...); err == nil {
			t.Fatalf("%s(%v) lost its trap", test.name, test.args)
		}
		if test.name == "store_then_trap" || test.name == "trap_then_store" {
			if got, ok := module.Memory().ReadUint32Le(0); !ok || got != 1 {
				t.Fatalf("%s reordered memory effects: got %d", test.name, got)
			}
		}
		if got := optimizerResult(t, module, "divide", 4, 2); got != 2 {
			t.Fatalf("%s trap left the module unusable", test.name)
		}
	}
}

func optimizerResult(t *testing.T, module api.Module, name string, args ...uint64) uint64 {
	t.Helper()
	results, err := module.ExportedFunction(name).Call(t.Context(), args...)
	if err != nil || len(results) != 1 {
		t.Fatalf("%s(%v): results=%v, err=%v", name, args, results, err)
	}
	return results[0]
}
