package harden

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"golang.org/x/tools/go/ast/astutil"
)

// hardenTables gives table bounds traps their own category and bounds bulk
// operations by length, including tables with spare capacity after growth.
func hardenTables(file *ast.File) error {
	tables := make(map[string]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		declaration, ok := node.(*ast.TypeSpec)
		if !ok || declaration.Name.Name != "Module" {
			return true
		}
		structure, ok := declaration.Type.(*ast.StructType)
		if !ok {
			return false
		}
		for _, field := range structure.Fields.List {
			if isTable(field.Type) {
				for _, name := range field.Names {
					tables[name.Name] = true
				}
			}
		}
		return false
	})
	indexed, sliced := false, false
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if function.Name.Name == "cocoon_table_index" || function.Name.Name == "cocoon_table_slice" {
			return fmt.Errorf("unexpected preexisting table hardening helper")
		}
		parameters := make(map[string]bool)
		if function.Type.Params != nil {
			for _, parameter := range function.Type.Params.List {
				if isTable(parameter.Type) {
					for _, name := range parameter.Names {
						parameters[name.Name] = true
					}
				}
			}
		}
		astutil.Apply(function.Body, func(cursor *astutil.Cursor) bool {
			switch expression := cursor.Node().(type) {
			case *ast.IndexExpr:
				selector, selectorOK := expression.X.(*ast.SelectorExpr)
				if selectorOK && tables[selector.Sel.Name] {
					expression.Index = &ast.CallExpr{Fun: ast.NewIdent("cocoon_table_index"), Args: []ast.Expr{convert64(expression.Index), length(expression.X)}}
					indexed = true
					return false
				}
			case *ast.SliceExpr:
				identifier, identifierOK := expression.X.(*ast.Ident)
				if identifierOK && parameters[identifier.Name] {
					low, high := expression.Low, expression.High
					if low == nil {
						low = &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(0)}
					}
					if high == nil {
						high = length(expression.X)
					}
					cursor.Replace(&ast.CallExpr{Fun: ast.NewIdent("cocoon_table_slice"), Args: []ast.Expr{expression.X, convert64(low), convert64(high)}})
					sliced = true
					return false
				}
			}
			return true
		}, nil)
	}
	helper := "package wasm\n"
	if indexed {
		helper += `
func cocoon_table_index(index uint64, length int) int {
if index >= uint64(length) { panic("out of bounds table access") }
return int(index)
}`
	}
	if sliced {
		helper += `
func cocoon_table_slice(table []any, low, high uint64) []any {
if low > high || high > uint64(len(table)) { panic("out of bounds table access") }
return table[low:high:len(table)]
}`
	}
	parsed, err := parser.ParseFile(token.NewFileSet(), "tables.go", helper, 0)
	if err != nil {
		return err
	}
	file.Decls = append(file.Decls, parsed.Decls...)
	return nil
}

func isTable(expression ast.Expr) bool {
	array, ok := expression.(*ast.ArrayType)
	if !ok || array.Len != nil {
		return false
	}
	element, ok := array.Elt.(*ast.Ident)
	return ok && element.Name == "any"
}

func convert64(expression ast.Expr) ast.Expr {
	return &ast.CallExpr{Fun: ast.NewIdent("uint64"), Args: []ast.Expr{expression}}
}

func length(expression ast.Expr) ast.Expr {
	return &ast.CallExpr{Fun: ast.NewIdent("len"), Args: []ast.Expr{expression}}
}
