package harden

import (
	"fmt"
	"go/ast"
	"go/token"
	"math"
	"strconv"
	"strings"
)

// reusePublicationMemory retains every guest access while reusing its header
// only in a validated, short publication tail. Unsupported code stays intact.
func reusePublicationMemory(file *ast.File) {
	bindings := scalarViewBindings(file)
	if bindings == nil {
		return
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		receiver := bindings.receiver(function)
		if receiver == nil || !eligibleViewNames(function) {
			continue
		}
		method := bindings.method(function)
		var names []*ast.Ident
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if _, nested := node.(*ast.FuncLit); nested {
				return false
			}
			block, ok := node.(*ast.BlockStmt)
			if ok {
				names = method.publicationTails(block, block == function.Body, receiver, names)
			}
			return true
		})
		if len(names) != 0 {
			declaration := &ast.DeclStmt{Decl: &ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{
				&ast.ValueSpec{Names: names, Type: &ast.ArrayType{Elt: ast.NewIdent("byte")}},
			}}}
			function.Body.List = append([]ast.Stmt{declaration}, function.Body.List...)
		}
	}
}

func eligibleViewNames(function *ast.FuncDecl) bool {
	valid := true
	ast.Inspect(function, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok &&
			(strings.HasPrefix(identifier.Name, "cocoonMemoryView") ||
				identifier.Name == "byte" && viewDeclaration(identifier) != nil) {
			valid = false
		}
		return valid
	})
	return valid
}

func (b *memoryViewBindings) method(function *ast.FuncDecl) *memoryViewBindings {
	method := *b
	method.locals = make(map[any]bool)
	for _, fields := range []*ast.FieldList{function.Type.Params, function.Type.Results} {
		if fields != nil {
			for _, field := range fields.List {
				method.locals[field] = true
			}
		}
	}
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ValueSpec:
			method.locals[node] = true
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				method.locals[node] = true
			}
		}
		return true
	})
	return &method
}

func (b *memoryViewBindings) publicationTails(block *ast.BlockStmt, outer bool, receiver *ast.Ident, names []*ast.Ident) []*ast.Ident {
	var output []ast.Stmt
	for index := range len(block.List) {
		statement := block.List[index]
		if !b.restoration(statement, receiver) || len(output) < 2 || !b.tailExit(block.List, index, outer) {
			output = append(output, statement)
			continue
		}
		second, secondBase, secondOffset := b.publicationStore(output[len(output)-1], receiver)
		first, firstBase, firstOffset := b.publicationStore(output[len(output)-2], receiver)
		adjacentWords := firstOffset == 0 && secondOffset == 4 || firstOffset == 4 && secondOffset == 0
		if first == nil || second == nil || !sameViewIdentifier(firstBase, secondBase) || !adjacentWords {
			output = append(output, statement)
			continue
		}
		start := len(output) - 2
		loads := 0
		for count := 0; start > 0 && count < 4; count++ {
			load, eligible := b.publicationLocal(output[start-1], receiver)
			if !eligible || load && loads == 1 {
				break
			}
			if load {
				loads++
			}
			start--
		}
		// Copies before the first load do not need a header capture.
		for start < len(output)-2 {
			load, _ := b.publicationLocal(output[start], receiver)
			if load {
				break
			}
			start++
		}
		name := fmt.Sprintf("cocoonMemoryView%d", len(names)+1)
		names = append(names, ast.NewIdent(name))
		capture := &ast.AssignStmt{Tok: token.ASSIGN, Lhs: []ast.Expr{ast.NewIdent(name)}, Rhs: []ast.Expr{
			&ast.SelectorExpr{X: ast.NewIdent(receiver.Name), Sel: ast.NewIdent("memory")},
		}}
		for _, node := range output[start:] {
			ast.Inspect(node, func(node ast.Node) bool {
				if call, ok := node.(*ast.CallExpr); ok &&
					(b.boundCall(call, "load32") || b.boundCall(call, "store32")) {
					call.Args[0] = ast.NewIdent(name)
				}
				return true
			})
		}
		tail := append([]ast.Stmt{capture}, output[start:]...)
		output = append(output[:start], tail...)
		output = append(output, statement)
	}
	block.List = output
	return names
}

func (b *memoryViewBindings) tailExit(statements []ast.Stmt, index int, outer bool) bool {
	if index+1 == len(statements) {
		return outer
	}
	switch next := statements[index+1].(type) {
	case *ast.BranchStmt:
		return next.Tok == token.GOTO
	case *ast.ReturnStmt:
		for _, result := range next.Results {
			if identifier, ok := result.(*ast.Ident); ok && b.localI32(identifier, 0) {
				continue
			}
			if _, ok := viewInteger(result, -1<<31, 1<<31-1); !ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func (b *memoryViewBindings) restoration(statement ast.Stmt, receiver *ast.Ident) bool {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.ASSIGN || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return false
	}
	field, ok := assignment.Lhs[0].(*ast.SelectorExpr)
	if !ok || b.globals[field.Sel.Name] == nil {
		return false
	}
	base, ok := field.X.(*ast.Ident)
	if !ok || !sameViewIdentifier(base, receiver) {
		return false
	}
	addition, ok := assignment.Rhs[0].(*ast.BinaryExpr)
	if !ok || addition.Op != token.ADD {
		return false
	}
	local, ok := addition.X.(*ast.Ident)
	if !ok || !b.localI32(local, 0) {
		return false
	}
	call, ok := addition.Y.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || !b.boundCall(call, "i32") {
		return false
	}
	_, ok = viewInteger(call.Args[0], 0, math.MaxInt32)
	return ok
}

func (b *memoryViewBindings) publicationStore(statement ast.Stmt, receiver *ast.Ident) (*ast.CallExpr, *ast.Ident, uint64) {
	expression, ok := statement.(*ast.ExprStmt)
	if !ok {
		return nil, nil, 0
	}
	call, ok := expression.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 3 || !b.boundCall(call, "store32") || !b.memoryArgument(call.Args[0], receiver) ||
		!b.publicationValue(call.Args[2]) {
		return nil, nil, 0
	}
	base, offset := b.publicationAddress(call.Args[1])
	if base == nil {
		return nil, nil, 0
	}
	return call, base, offset
}

func (b *memoryViewBindings) publicationValue(expression ast.Expr) bool {
	value, ok := b.conversion(expression, "uint32")
	if !ok {
		return false
	}
	if identifier, identified := value.(*ast.Ident); identified {
		return b.localI32(identifier, 0)
	}
	call, ok := value.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || !b.boundCall(call, "i32") {
		return false
	}
	_, ok = viewInteger(call.Args[0], math.MinInt32, math.MaxInt32)
	return ok
}

func (b *memoryViewBindings) publicationAddress(expression ast.Expr) (base *ast.Ident, offset uint64) {
	if addition, ok := expression.(*ast.BinaryExpr); ok && addition.Op == token.ADD {
		constant, ok := viewInteger(addition.Y, 0, math.MaxUint32)
		if !ok {
			return nil, 0
		}
		offset = uint64(constant) // #nosec G115 -- The nonnegative constant is bounded to MaxUint32.
		expression = addition.X
		value, ok := b.conversion(expression, "uint64")
		if !ok {
			return nil, 0
		}
		expression = value
	}
	value, ok := b.conversion(expression, "uint32")
	if !ok {
		return nil, 0
	}
	base, ok = value.(*ast.Ident)
	if !ok || !b.localI32(base, 0) {
		return nil, 0
	}
	// Zero extension plus a u32 offset is below 2^33. Keeping the exact
	// expression preserves wrapping distinctions; the helper's +3 is safe.
	return base, offset
}

func (b *memoryViewBindings) publicationLocal(statement ast.Stmt, receiver *ast.Ident) (load, eligible bool) {
	assignment, ok := statement.(*ast.AssignStmt)
	if !ok || (assignment.Tok != token.ASSIGN && assignment.Tok != token.DEFINE) ||
		len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return false, false
	}
	local, ok := assignment.Lhs[0].(*ast.Ident)
	if !ok || !b.localI32(local, 0) {
		return false, false
	}
	if copied, identified := assignment.Rhs[0].(*ast.Ident); identified {
		return false, b.localI32(copied, 0)
	}
	value, ok := b.conversion(assignment.Rhs[0], "int32")
	if !ok {
		return false, false
	}
	call, ok := value.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 || !b.boundCall(call, "load32") || !b.memoryArgument(call.Args[0], receiver) {
		return false, false
	}
	base, _ := b.publicationAddress(call.Args[1])
	return base != nil, base != nil
}

func viewInteger(expression ast.Expr, minimum, maximum int64) (int64, bool) {
	sign := int64(1)
	if unary, ok := expression.(*ast.UnaryExpr); ok && unary.Op == token.SUB {
		sign = -1
		expression = unary.X
	}
	literal, ok := expression.(*ast.BasicLit)
	if !ok || literal.Kind != token.INT {
		return 0, false
	}
	value, err := strconv.ParseInt(literal.Value, 0, 64)
	if err != nil {
		return 0, false
	}
	value *= sign
	return value, value >= minimum && value <= maximum
}
