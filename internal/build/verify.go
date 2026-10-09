// Package build orchestrates the pinned Rust, Binaryen, and wasm2go toolchain.
package build

import (
	"fmt"
	"slices"
	"strings"

	"dario.cat/cocoon/internal/manifest"
	"dario.cat/cocoon/internal/wasmbin"
)

// ExportSet derives the exact ABI export signatures from a validated manifest.
func ExportSet(m *manifest.Manifest) map[string]wasmbin.Type {
	i32 := []wasmbin.Value{wasmbin.I32}
	exports := map[string]wasmbin.Type{
		"cocoon_abi_version": {Results: i32},
		"cocoon_schema_hash": {Results: []wasmbin.Value{wasmbin.I64}},
		"cocoon_init":        {}, "cocoon_out": {Results: i32},
		"cocoon_in_reserve": {Params: i32, Results: i32},
		"cocoon_trim":       {Params: i32}, "cocoon_alloc": {Params: i32, Results: i32},
	}
	add := func(name string, function manifest.Function, handle bool) {
		var parameters []wasmbin.Value
		if handle {
			parameters = append(parameters, wasmbin.I64)
		}
		for _, parameter := range function.Params {
			if m.Variable(parameter.Type) {
				parameters = append(parameters, wasmbin.I32, wasmbin.I32)
				continue
			}
			value := wasmbin.I32
			switch parameter.Type {
			case "i64", "u64":
				value = wasmbin.I64
			case "f32":
				value = wasmbin.F32
			case "f64":
				value = wasmbin.F64
			}
			parameters = append(parameters, value)
		}
		exports[name] = wasmbin.Type{Params: parameters, Results: i32}
	}
	for _, function := range m.Functions {
		add("cocoon_"+function.Name, function, false)
	}
	for _, resource := range m.Resources {
		for _, method := range resource.Methods {
			add("cocoon_"+strings.ToLower(resource.Name)+"_"+method.Name, method, method.Name != "new")
		}
	}
	return exports
}

// Verify checks imports, exports, exact signatures, memory policy, and feature metadata.
// It complements Binaryen's instruction and initializer validation.
func Verify(m *manifest.Manifest, data []byte) (*wasmbin.Module, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	module, err := wasmbin.Read(data)
	if err != nil {
		return nil, err
	}
	_, _, maximum, err := m.ByteLimits()
	if err != nil {
		return nil, err
	}
	if module.HasStart {
		return nil, fmt.Errorf("start functions are forbidden")
	}
	if len(module.Memories) != 1 || uint64(module.Memories[0].Max)*65536 != maximum {
		return nil, fmt.Errorf("module must have exactly one memory with the declared maximum")
	}
	allowed := make(map[string]wasmbin.Type)
	if m.Capabilities.Log {
		allowed["log"] = wasmbin.Type{Params: []wasmbin.Value{wasmbin.I32, wasmbin.I32, wasmbin.I32}}
	}
	if m.Capabilities.Random {
		allowed["random_get"] = wasmbin.Type{Params: []wasmbin.Value{wasmbin.I32, wasmbin.I32}, Results: []wasmbin.Value{wasmbin.I32}}
	}
	if m.Capabilities.Clock {
		allowed["clock_nanos"] = wasmbin.Type{Results: []wasmbin.Value{wasmbin.I64}}
	}
	imports := make(map[string]bool)
	for _, imported := range module.Imports {
		signature, exists := allowed[imported.Name]
		if imported.Module != "cocoon" || imported.Kind != 0 || !exists || imports[imported.Name] {
			return nil, fmt.Errorf("undeclared or duplicate import %s.%s", imported.Module, imported.Name)
		}
		if !sameType(signature, module.Types[imported.Type]) {
			return nil, fmt.Errorf("import signature mismatch: %s", imported.Name)
		}
		imports[imported.Name] = true
	}
	expected := ExportSet(m)
	seen := make(map[string]bool)
	for _, exported := range module.Exports {
		if exported.Name == "memory" && exported.Kind == 2 && exported.Index == 0 {
			seen["memory"] = true
			continue
		}
		signature, exists := expected[exported.Name]
		if !exists {
			return nil, fmt.Errorf("unexpected export %s", exported.Name)
		}
		actual, signatureErr := module.Signature(exported)
		if signatureErr != nil {
			return nil, signatureErr
		}
		if !sameType(signature, actual) {
			return nil, fmt.Errorf("export signature mismatch: %s", exported.Name)
		}
		seen[exported.Name] = true
	}
	if !seen["memory"] {
		return nil, fmt.Errorf("missing memory export")
	}
	for name := range expected {
		if !seen[name] {
			return nil, fmt.Errorf("missing export %s", name)
		}
	}
	for _, feature := range module.Features {
		if !slices.Contains([]string{"mutable-globals", "bulk-memory", "bulk-memory-opt", "sign-ext", "nontrapping-fptoint", "reference-types", "multivalue", "call-indirect-overlong"}, feature) {
			return nil, fmt.Errorf("unsupported wasm feature %s", feature)
		}
	}
	return module, nil
}

func sameType(a, b wasmbin.Type) bool {
	return slices.Equal(a.Params, b.Params) && slices.Equal(a.Results, b.Results)
}
