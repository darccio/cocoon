package harden

import (
	"go/ast"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/cfg"
)

// removeDeadReturns removes only returns proven unreachable by Go's control-flow
// graph. wasm2go can emit these between terminating blocks and live goto labels;
// preserving the labels is essential. All other statements remain untouched.
func removeDeadReturns(file *ast.File) {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		graph := cfg.New(function.Body, func(call *ast.CallExpr) bool {
			name, named := call.Fun.(*ast.Ident)
			return !named || name.Name != "panic"
		})
		dead := make(map[ast.Node]bool)
		for _, block := range graph.Blocks {
			if !block.Live {
				for _, node := range block.Nodes {
					if _, isReturn := node.(*ast.ReturnStmt); isReturn {
						dead[node] = true
					}
				}
			}
		}
		astutil.Apply(function.Body, func(cursor *astutil.Cursor) bool {
			if dead[cursor.Node()] && cursor.Index() >= 0 {
				cursor.Delete()
				return false
			}
			return true
		}, nil)
	}
}
