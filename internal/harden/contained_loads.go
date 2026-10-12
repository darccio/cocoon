package harden

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
)

const containedLoadHelper = "cocoonLoad32InRange"

// Grouped parameter/var declarations share a node. Include the bound name so
// distinct variables in that declaration never share a fact or invalidation.
type containedBinding struct {
	declaration any
	name        string
}

func containedKey(identifier *ast.Ident) containedBinding {
	return containedBinding{declaration: viewDeclaration(identifier), name: identifier.Name}
}

// elideContainedLoads keeps each read and its original evaluation position.
// Only a successful, earlier endpoint check on an unchanged base can cover it.
// Publication rewriting runs first so its existing ownership guard is unchanged.
func elideContainedLoads(file *ast.File) []byte {
	bindings := scalarViewBindings(file)
	if bindings == nil || containedNameExists(file) {
		return nil
	}
	helper := containedHelperSource(bindings.functions["load32"])
	if helper == nil {
		return nil
	}
	changed := false
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		receiver := bindings.receiver(function)
		if receiver == nil {
			continue
		}
		method := bindings.method(function)
		escaped, concurrent := containedEscapes(function)
		if concurrent {
			continue
		}
		ast.Inspect(function.Body, func(node ast.Node) bool {
			if _, nested := node.(*ast.FuncLit); nested {
				return false
			}
			if block, ok := node.(*ast.BlockStmt); ok {
				changed = method.containedBlock(block, receiver, escaped) || changed
			}
			return true
		})
	}
	if !changed {
		return nil
	}
	return helper
}

func containedNameExists(file *ast.File) bool {
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		if identifier, ok := node.(*ast.Ident); ok && identifier.Name == containedLoadHelper {
			found = true
		}
		return !found
	})
	return found
}

// The complete helper body and bindings were validated before this copy. Keep
// its directives, endian handling, checked fallback and live unsafe read.
func containedHelperSource(function *ast.FuncDecl) []byte {
	var output bytes.Buffer
	// A standalone FuncDecl printer does not reliably attach its original
	// source-positioned Doc. Copy those validated directives explicitly.
	if function.Doc != nil {
		for _, comment := range function.Doc.List {
			output.WriteString(comment.Text)
			output.WriteByte('\n')
		}
	}
	declarationNode := *function
	declarationNode.Doc = nil
	if format.Node(&output, token.NewFileSet(), &declarationNode) != nil {
		return nil
	}
	source := output.Bytes()
	check := []byte("\t_ = mem[uint64(addr)+3]\n")
	declaration := []byte("func load32[")
	if bytes.Count(source, check) != 1 || bytes.Count(source, declaration) != 1 {
		return nil
	}
	source = bytes.Replace(source, declaration, []byte("func "+containedLoadHelper+"["), 1)
	return bytes.Replace(source, check, nil, 1)
}

// A base can escape before a region begins. Clearing a fact at that earlier
// statement would not prevent a later callback from changing the local.
func containedEscapes(function *ast.FuncDecl) (map[containedBinding]bool, bool) {
	escaped := make(map[containedBinding]bool)
	concurrent := false
	mark := func(node ast.Node) {
		ast.Inspect(node, func(node ast.Node) bool {
			if identifier, ok := node.(*ast.Ident); ok && identifier.Obj != nil {
				escaped[containedKey(identifier)] = true
			}
			return true
		})
	}
	ast.Inspect(function.Body, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.GoStmt:
			concurrent = true
		case *ast.FuncLit:
			mark(node.Body)
		case *ast.UnaryExpr:
			if node.Op == token.AND {
				mark(node.X)
			}
		}
		return true
	})
	return escaped, concurrent
}

func (b *memoryViewBindings) containedBlock(block *ast.BlockStmt, receiver *ast.Ident, escaped map[containedBinding]bool) bool {
	facts := make(map[containedBinding]uint64)
	changed := false
	for _, statement := range block.List {
		assignment, ok := statement.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 ||
			(assignment.Tok != token.ASSIGN && assignment.Tok != token.DEFINE) {
			clear(facts)
			continue
		}
		destination, ok := assignment.Lhs[0].(*ast.Ident)
		if !ok || !b.locals[viewDeclaration(destination)] || destination.Name == "_" {
			clear(facts)
			continue
		}
		call, base, offset := b.containedLoad(assignment.Rhs[0], receiver)
		if call != nil && !escaped[containedKey(base)] && b.containedDestination(destination, assignment) {
			endpoint := offset + 3
			key := containedKey(base)
			if previous, seen := facts[key]; seen && endpoint <= previous {
				call.Fun = &ast.Ident{Name: containedLoadHelper, NamePos: call.Fun.Pos()}
				changed = true
			} else {
				facts[key] = endpoint
			}
		} else if b.containedScalarType(destination, 0) == "" || !b.containedPure(assignment.Rhs[0]) {
			clear(facts)
		}
		// The RHS sees the old base; only then does assignment invalidate it.
		delete(facts, containedKey(destination))
	}
	return changed
}

func (b *memoryViewBindings) containedLoad(expression ast.Expr, receiver *ast.Ident) (*ast.CallExpr, *ast.Ident, uint64) {
	if converted, ok := b.conversion(expression, "int32"); ok {
		expression = converted
	}
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 2 || !b.boundCall(call, "load32") || !b.memoryArgument(call.Args[0], receiver) {
		return nil, nil, 0
	}
	base, offset := b.publicationAddress(call.Args[1])
	if base == nil || !b.containedExplicitBase(base) {
		return nil, nil, 0
	}
	// uint32(base) plus a u32 literal and the width end is at most 2^33+1.
	return call, base, offset
}

func (b *memoryViewBindings) containedExplicitBase(base *ast.Ident) bool {
	if !b.locals[viewDeclaration(base)] {
		return false
	}
	switch declaration := viewDeclaration(base).(type) {
	case *ast.Field:
		return b.builtin(declaration.Type, "int32")
	case *ast.ValueSpec:
		return b.builtin(declaration.Type, "int32")
	default:
		return false
	}
}

func (b *memoryViewBindings) containedDestination(destination *ast.Ident, assignment *ast.AssignStmt) bool {
	if assignment.Tok == token.DEFINE && viewDeclaration(destination) == assignment {
		// Its sole defining RHS is the validated uint32 load or int32 wrapper.
		return true
	}
	kind := b.containedScalarType(destination, 0)
	return kind == "int32" || kind == "uint32"
}

func (b *memoryViewBindings) containedScalarType(identifier *ast.Ident, depth int) string {
	if identifier == nil || identifier.Name == "_" || depth > 16 || !b.locals[viewDeclaration(identifier)] {
		return ""
	}
	switch declaration := viewDeclaration(identifier).(type) {
	case *ast.Field:
		return b.containedBuiltinType(declaration.Type)
	case *ast.ValueSpec:
		return b.containedBuiltinType(declaration.Type)
	case *ast.AssignStmt:
		if declaration.Tok == token.DEFINE && len(declaration.Lhs) == 1 && len(declaration.Rhs) == 1 {
			return b.containedExpressionType(declaration.Rhs[0], depth+1)
		}
	}
	return ""
}

func (b *memoryViewBindings) containedBuiltinType(expression ast.Expr) string {
	for _, name := range []string{"int32", "int64", "uint32", "uint64", "float32", "float64"} {
		if b.builtin(expression, name) {
			return name
		}
	}
	return ""
}

func (b *memoryViewBindings) containedExpressionType(expression ast.Expr, depth int) string {
	if identifier, ok := expression.(*ast.Ident); ok {
		return b.containedScalarType(identifier, depth)
	}
	call, ok := expression.(*ast.CallExpr)
	if !ok || call.Ellipsis.IsValid() {
		return ""
	}
	if len(call.Args) == 2 && b.boundCall(call, "load32") {
		return "uint32"
	}
	if len(call.Args) != 1 {
		return ""
	}
	if kind := b.containedBuiltinType(call.Fun); kind != "" {
		return kind
	}
	return b.containedIdentityType(call)
}

func (b *memoryViewBindings) containedIdentityType(call *ast.CallExpr) string {
	if len(call.Args) != 1 || call.Ellipsis.IsValid() {
		return ""
	}
	if b.boundCall(call, "i32") {
		return "int32" // The complete canonical identity declaration is validated.
	}
	if !b.boundCall(call, "i64") {
		return ""
	}
	function := b.functions["i64"]
	if function.Type.TypeParams != nil || function.Type.Params == nil || function.Type.Results == nil ||
		len(function.Type.Params.List) != 1 || len(function.Type.Results.List) != 1 ||
		function.Body == nil || len(function.Body.List) != 1 || !viewDirectives(function.Doc) {
		return ""
	}
	parameter, result := function.Type.Params.List[0], function.Type.Results.List[0]
	if len(parameter.Names) != 1 || len(result.Names) != 0 ||
		!b.builtin(parameter.Type, "int64") || !b.builtin(result.Type, "int64") {
		return ""
	}
	returned, ok := function.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(returned.Results) != 1 {
		return ""
	}
	value, ok := returned.Results[0].(*ast.Ident)
	if !ok || !sameViewIdentifier(value, parameter.Names[0]) {
		return ""
	}
	return "int64"
}

func (b *memoryViewBindings) containedPure(expression ast.Expr) bool {
	switch expression := expression.(type) {
	case *ast.Ident:
		return b.containedScalarType(expression, 0) != ""
	case *ast.BasicLit:
		return expression.Kind == token.INT || expression.Kind == token.FLOAT || expression.Kind == token.CHAR
	case *ast.ParenExpr:
		return b.containedPure(expression.X)
	case *ast.UnaryExpr:
		return (expression.Op == token.ADD || expression.Op == token.SUB || expression.Op == token.XOR) &&
			b.containedPure(expression.X)
	case *ast.BinaryExpr:
		return b.containedPure(expression.X) && b.containedPure(expression.Y)
	case *ast.CallExpr:
		return len(expression.Args) == 1 && !expression.Ellipsis.IsValid() &&
			(b.containedBuiltinType(expression.Fun) != "" || b.containedIdentityType(expression) != "") &&
			b.containedPure(expression.Args[0])
	default:
		return false
	}
}
