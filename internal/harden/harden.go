// Package harden validates and rewrites pinned wasm2go memory helper ASTs.
package harden

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"slices"
	"strings"
)

var bulkHelpers = []string{"memory_init", "memory_copy", "memory_fill", "memory_zero"}

var helperSignatures = map[string]string{
	"memory_init": "func[T1, T2 int | uint32 | uint64](mem []byte, data string, dest T1, src, n T2)",
	"memory_copy": "func[T uint32 | uint64](mem []byte, dest, src, n T)",
	"memory_fill": "func[T uint32 | uint64](mem []byte, dest T, val int32, n T)",
	"memory_zero": "func[T uint32 | uint64](mem []byte, dest, n T)",
}

// Required derives the precise bulk-memory helper set from generated calls.
func Required(source []byte) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), "translated.go", source, 0)
	if err != nil {
		return nil, fmt.Errorf("parse translated Go: %w", err)
	}
	names := make(map[string]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		function := call.Fun
		if indexed, indexedOK := function.(*ast.IndexExpr); indexedOK {
			function = indexed.X
		}
		if indexed, indexedOK := function.(*ast.IndexListExpr); indexedOK {
			function = indexed.X
		}
		if name, nameOK := function.(*ast.Ident); nameOK && slices.Contains(bulkHelpers, name.Name) {
			names[name.Name] = true
		}
		return true
	})
	var required []string
	for _, name := range bulkHelpers {
		if names[name] {
			required = append(required, name)
		}
	}
	return required, nil
}

// Rewrite fails on helper drift and bounds every bulk-memory view by length.
func Rewrite(source []byte, expected []string) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "translated.go", source, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse translated Go: %w", err)
	}
	wanted := make(map[string]bool)
	for _, name := range expected {
		if wanted[name] || !slices.Contains(bulkHelpers, name) {
			return nil, fmt.Errorf("invalid expected helper %q", name)
		}
		wanted[name] = true
	}
	counts := make(map[string]int)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		name := function.Name.Name
		if strings.HasPrefix(name, "memory_") && name != "memory_grow" && name != "memory_size" && !slices.Contains(bulkHelpers, name) {
			return nil, fmt.Errorf("unknown memory helper %q", name)
		}
		if !slices.Contains(bulkHelpers, name) {
			continue
		}
		counts[name]++
		if !wanted[name] || counts[name] != 1 {
			return nil, fmt.Errorf("unexpected or duplicate helper %q", name)
		}
		if function.Recv != nil || function.Body == nil || function.Type.TypeParams == nil || function.Type.Results != nil || function.Type.Params == nil || len(function.Type.Params.List) == 0 {
			return nil, fmt.Errorf("helper signature drift: %s", name)
		}
		parameter := function.Type.Params.List[0]
		if len(parameter.Names) != 1 || parameter.Names[0].Name != "mem" {
			return nil, fmt.Errorf("helper memory parameter drift: %s", name)
		}
		var shape bytes.Buffer
		if formatErr := format.Node(&shape, fset, parameter.Type); formatErr != nil {
			return nil, formatErr
		}
		if shape.String() != "[]byte" {
			return nil, fmt.Errorf("helper memory type drift: %s", name)
		}
		shape.Reset()
		if signatureErr := format.Node(&shape, fset, function.Type); signatureErr != nil {
			return nil, signatureErr
		}
		if shape.String() != helperSignatures[name] {
			return nil, fmt.Errorf("helper full signature drift: %s", name)
		}
		views := 0
		ast.Inspect(function.Body, func(node ast.Node) bool {
			view, viewOK := node.(*ast.SliceExpr)
			if !viewOK {
				return true
			}
			memory, memoryOK := view.X.(*ast.Ident)
			if !memoryOK || memory.Name != "mem" {
				return true
			}
			if view.Slice3 {
				return false
			}
			view.Slice3 = true
			if view.High == nil {
				view.High = &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{ast.NewIdent("mem")}}
			}
			view.Max = &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{ast.NewIdent("mem")}}
			views++
			return false
		})
		minimum := 1
		if name == "memory_copy" {
			minimum = 2
		}
		if views != minimum {
			return nil, fmt.Errorf("helper body drift: %s has %d memory views, expected %d", name, views, minimum)
		}
	}
	for name := range wanted {
		if counts[name] != 1 {
			return nil, fmt.Errorf("missing helper %q", name)
		}
	}
	if tableErr := hardenTables(file); tableErr != nil {
		return nil, tableErr
	}
	removeDeadReturns(file)
	var output bytes.Buffer
	if err := format.Node(&output, fset, file); err != nil {
		return nil, fmt.Errorf("format hardened Go: %w", err)
	}
	return output.Bytes(), nil
}
