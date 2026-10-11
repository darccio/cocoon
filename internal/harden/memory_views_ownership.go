package harden

import (
	"go/ast"
	"go/token"
)

// The translator owns its backing allocation. Native Init/Memory adapters may
// replace it only under the existing trusted, nonconcurrent host precondition.
// Generated code cannot rebind the receiver or install an unproved header.
func (b *memoryViewBindings) ownedConstruction(file *ast.File) bool {
	constructor := b.functions["New"]
	if constructor == nil || constructor.Body == nil || constructor.Type.TypeParams != nil ||
		constructor.Type.Results == nil || len(constructor.Type.Results.List) != 1 ||
		len(constructor.Type.Results.List[0].Names) != 0 ||
		!b.modulePointer(constructor.Type.Results.List[0].Type) || len(constructor.Body.List) < 3 {
		return false
	}
	allocation, ok := constructor.Body.List[0].(*ast.AssignStmt)
	if !ok || allocation.Tok != token.DEFINE || len(allocation.Lhs) != 1 || len(allocation.Rhs) != 1 {
		return false
	}
	owner, ok := allocation.Lhs[0].(*ast.Ident)
	if !ok {
		return false
	}
	call, ok := allocation.Rhs[0].(*ast.CallExpr)
	if !ok || !b.builtin(call.Fun, "new") || call.Ellipsis.IsValid() || len(call.Args) != 1 {
		return false
	}
	module, ok := call.Args[0].(*ast.Ident)
	if !ok || viewDeclaration(module) != b.module {
		return false
	}
	final, ok := constructor.Body.List[len(constructor.Body.List)-1].(*ast.ReturnStmt)
	if !ok || len(final.Results) != 1 {
		return false
	}
	returned, ok := final.Results[0].(*ast.Ident)
	if !ok || !sameViewIdentifier(returned, owner) {
		return false
	}
	allocations := 0
	valid := true
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok {
			ast.Inspect(declaration, func(node ast.Node) bool {
				if _, literal := node.(*ast.FuncLit); literal {
					valid = false
				}
				if selector, ok := node.(*ast.SelectorExpr); ok {
					if name, ok := selector.X.(*ast.Ident); ok && name.Name == "unsafe" {
						valid = false
					}
				}
				return valid
			})
			if !valid {
				return false
			}
			continue
		}
		if function.Body == nil {
			continue
		}
		// These complete bodies were validated by support. Their indirect byte
		// stores and owned growth are the only proved pointer-write operations.
		if function.Recv == nil && (viewScalarHelper(function.Name.Name) || function.Name.Name == "memory_grow") {
			continue
		}
		receiver := b.receiver(function)
		if function == constructor {
			receiver = owner
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if !valid {
				return false
			}
			switch node := node.(type) {
			case *ast.DeferStmt:
				valid = valid && function != constructor
			case *ast.RangeStmt:
				if node.Tok == token.ASSIGN {
					valid = valid && !viewReceiverTarget(node.Key, receiver) && !viewReceiverTarget(node.Value, receiver) &&
						!viewMemoryField(node.Key) && !viewMemoryField(node.Value)
				}
			case *ast.SelectorExpr:
				if name, ok := node.X.(*ast.Ident); ok && name.Name == "unsafe" {
					// Full scalar helper bodies are validated above. No other
					// translated function may fabricate a native backing alias.
					valid = valid && function.Recv == nil && viewScalarHelper(function.Name.Name)
				}
			case *ast.AssignStmt:
				for i, target := range node.Lhs {
					target = viewUnwrap(target)
					if node != allocation && viewReceiverTarget(target, receiver) {
						valid = false
					}
					if b.memoryArgument(target, receiver) {
						valid = valid && function == constructor && node.Tok == token.ASSIGN &&
							len(node.Lhs) == 1 && len(node.Rhs) == 1 && b.ownedMemory(node.Rhs[i])
						allocations++
					}
					if field, ok := target.(*ast.SelectorExpr); ok && field.Sel.Name == "memory" &&
						!b.memoryArgument(target, receiver) {
						valid = false
					}
				}
			case *ast.ReturnStmt:
				if function == constructor && node != final {
					valid = false
				}
			case *ast.UnaryExpr:
				addressed := viewUnwrap(node.X)
				if name, ok := addressed.(*ast.Ident); ok && node.Op == token.AND && sameViewIdentifier(name, receiver) {
					valid = false
				}
				if node.Op == token.AND && viewMemoryField(addressed) {
					valid = valid && b.memoryArgument(addressed, receiver) && b.provenMemoryAddress(function, node)
				}
			}
			return valid
		})
		if !valid {
			return false
		}
	}
	return valid && allocations == 1
}

func viewReceiverTarget(expression ast.Expr, receiver *ast.Ident) bool {
	expression = viewUnwrap(expression)
	if name, ok := expression.(*ast.Ident); ok {
		return sameViewIdentifier(name, receiver)
	}
	_, indirect := expression.(*ast.StarExpr)
	return indirect
}

func viewUnwrap(expression ast.Expr) ast.Expr {
	for {
		parenthesized, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression
		}
		expression = parenthesized.X
	}
}

func viewMemoryField(expression ast.Expr) bool {
	field, ok := viewUnwrap(expression).(*ast.SelectorExpr)
	return ok && field.Sel.Name == "memory"
}

func viewScalarHelper(name string) bool {
	switch name {
	case "load16", "store16", "load32", "store32", "load64", "store64":
		return true
	default:
		return false
	}
}

func (b *memoryViewBindings) modulePointer(expression ast.Expr) bool {
	pointer, ok := expression.(*ast.StarExpr)
	if !ok {
		return false
	}
	name, ok := pointer.X.(*ast.Ident)
	return ok && viewDeclaration(name) == b.module
}

func (b *memoryViewBindings) ownedMemory(expression ast.Expr) bool {
	call, ok := expression.(*ast.CallExpr)
	return ok && b.builtin(call.Fun, "make") && !call.Ellipsis.IsValid() &&
		(len(call.Args) == 2 || len(call.Args) == 3) && b.byteSlice(call.Args[0])
}

func (b *memoryViewBindings) provenMemoryAddress(function *ast.FuncDecl, address *ast.UnaryExpr) bool {
	proven := false
	ast.Inspect(function.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if ok && len(call.Args) == 3 && call.Args[0] == address && b.boundCall(call, "memory_grow") {
			proven = true
		}
		return true
	})
	if proven {
		return true
	}
	// The canonical exported adapter hands the header to the trusted host. It
	// does not create backing storage or a snapshot usable by this pass.
	if function.Name.Name != "Xmemory" || len(function.Body.List) != 1 {
		return false
	}
	statement, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(statement.Results) != 1 {
		return false
	}
	call, ok := statement.Results[0].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || call.Args[0] != address {
		return false
	}
	conversion, ok := call.Fun.(*ast.ParenExpr)
	if !ok {
		return false
	}
	pointer, ok := conversion.X.(*ast.StarExpr)
	if !ok {
		return false
	}
	name, ok := pointer.X.(*ast.Ident)
	if !ok || name.Name != "wasmMemory" {
		return false
	}
	typeSpec, ok := viewDeclaration(name).(*ast.TypeSpec)
	return ok && !typeSpec.Assign.IsValid() && typeSpec.TypeParams == nil && b.byteSlice(typeSpec.Type)
}
