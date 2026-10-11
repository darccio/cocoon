package harden

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

type memoryViewBindings struct {
	module    *ast.TypeSpec
	memory    *ast.Field
	globals   map[string]*ast.Field
	functions map[string]*ast.FuncDecl
	imports   map[string]string
	declared  map[string]bool
	locals    map[any]bool
}

func scalarViewBindings(file *ast.File) *memoryViewBindings {
	b := &memoryViewBindings{
		globals: make(map[string]*ast.Field), functions: make(map[string]*ast.FuncDecl),
		imports:  make(map[string]string),
		declared: make(map[string]bool),
	}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || spec.Name != nil {
			return nil
		}
		name := path[strings.LastIndexByte(path, '/')+1:]
		if b.imports[name] != "" {
			return nil
		}
		b.imports[name] = path
	}
	for _, declaration := range file.Decls {
		if function, ok := declaration.(*ast.FuncDecl); ok && function.Recv == nil {
			if b.functions[function.Name.Name] != nil {
				return nil
			}
			b.functions[function.Name.Name] = function
			b.declared[function.Name.Name] = true
		}
		if declaration, ok := declaration.(*ast.GenDecl); ok {
			for _, spec := range declaration.Specs {
				switch spec := spec.(type) {
				case *ast.TypeSpec:
					b.declared[spec.Name.Name] = true
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						b.declared[name.Name] = true
					}
				}
				if spec, ok := spec.(*ast.TypeSpec); ok && spec.Name.Name == "Module" {
					if b.module != nil {
						return nil
					}
					b.module = spec
				}
			}
		}
	}
	if !b.structure() || !b.support(file) || !b.ownedConstruction(file) {
		return nil
	}
	return b
}

func (b *memoryViewBindings) structure() bool {
	if b.module == nil || b.module.Assign.IsValid() || b.module.TypeParams != nil {
		return false
	}
	structure, ok := b.module.Type.(*ast.StructType)
	if !ok {
		return false
	}
	for _, field := range structure.Fields.List {
		for _, name := range field.Names {
			if name.Name == "memory" {
				if b.memory != nil || !b.byteSlice(field.Type) {
					return false
				}
				b.memory = field
			}
			if b.builtin(field.Type, "int32") {
				b.globals[name.Name] = field
			}
		}
	}
	return b.memory != nil && len(b.globals) != 0
}

func (b *memoryViewBindings) support(file *ast.File) bool {
	for _, comments := range file.Comments {
		if !viewDirectives(comments) {
			return false
		}
	}
	paths := map[string]string{
		"binary": "encoding/binary", "bits": "math/bits", "unsafe": "unsafe", "runtime": "runtime", "math": "math",
	}
	for name, path := range b.imports {
		if paths[name] != path {
			return false
		}
	}
	for _, name := range []string{"binary", "bits", "unsafe", "runtime"} {
		if b.imports[name] != paths[name] {
			return false
		}
	}
	source := memoryViewSupport
	for _, width := range []int{16, 32, 64} {
		source += fmt.Sprintf(memoryViewLoad, width, width/8-1) + fmt.Sprintf(memoryViewStore, width, width/8-1)
	}
	expected, err := parser.ParseFile(token.NewFileSet(), "scalar-shapes.go", source, 0)
	if err != nil {
		return false
	}
	shapes := make(map[string]ast.Node)
	for _, declaration := range expected.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			shapes[declaration.Name.Name] = declaration
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok && len(value.Names) == 1 {
					shapes[value.Names[0].Name] = value.Values[0]
				}
			}
		}
	}
	for _, name := range []string{"load32", "store32", "i32"} {
		if b.functions[name] == nil {
			return false
		}
	}
	for name, function := range b.functions {
		shape := shapes[name]
		if shape == nil {
			continue
		}
		if function.Body == nil || !sameViewShape(function.Type, viewFunctionType(shape)) ||
			!sameViewShape(function.Body, viewFunctionBody(shape)) || !b.supportBindings(function) ||
			!viewDirectives(function.Doc) {
			return false
		}
	}
	for _, name := range []string{"big", "unalignedOK"} {
		value := viewValueDeclaration(file, name)
		if value == nil || !sameViewShape(value, shapes[name]) || !b.supportBindings(value) {
			return false
		}
	}
	return true
}

func viewFunctionType(node ast.Node) *ast.FuncType {
	function, ok := node.(*ast.FuncDecl)
	if !ok {
		return nil
	}
	return function.Type
}

func viewFunctionBody(node ast.Node) *ast.BlockStmt {
	function, ok := node.(*ast.FuncDecl)
	if !ok {
		return nil
	}
	return function.Body
}

func viewValueDeclaration(file *ast.File, name string) ast.Expr {
	var result ast.Expr
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.CONST {
			continue
		}
		for _, spec := range group.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || len(value.Values) != 1 || value.Type != nil || value.Names[0].Name != name {
				continue
			}
			if result != nil {
				return nil
			}
			result = value.Values[0]
		}
	}
	return result
}

func sameViewShape(a, b ast.Node) bool {
	if a == nil || b == nil {
		return false
	}
	var first, second bytes.Buffer
	return format.Node(&first, token.NewFileSet(), a) == nil &&
		format.Node(&second, token.NewFileSet(), b) == nil && first.String() == second.String()
}

func viewDirectives(doc *ast.CommentGroup) bool {
	if doc == nil {
		return true
	}
	for _, comment := range doc.List {
		if strings.HasPrefix(comment.Text, "//go:") && comment.Text != "//go:nosplit" {
			return false
		}
	}
	return true
}

func (b *memoryViewBindings) supportBindings(node ast.Node) bool {
	valid := true
	ast.Inspect(node, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if name, ok := call.Fun.(*ast.Ident); ok {
				switch name.Name {
				case "len", "make", "append", "min":
					valid = valid && b.builtin(name, name.Name)
				}
			}
		}
		identifier, ok := node.(*ast.Ident)
		if !ok {
			return true
		}
		switch identifier.Name {
		case "int", "int32", "int64", "uint16", "uint32", "uint64", "uintptr", "byte", "false":
			valid = valid && b.builtin(identifier, identifier.Name)
		case "binary", "bits", "unsafe", "runtime", "math":
			valid = valid && viewDeclaration(identifier) == nil && b.imports[identifier.Name] != ""
		}
		return valid
	})
	return valid
}

// The selected grammar contains no ambiguous composite-literal keys or type
// expressions. Parser object references identify declarations in these limited
// forms; unsupported forms never reach the transform.
func viewDeclaration(identifier *ast.Ident) any {
	if identifier.Obj != nil {
		return identifier.Obj.Decl
	}
	return nil
}

func (b *memoryViewBindings) builtin(expression ast.Expr, name string) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && identifier.Name == name && viewDeclaration(identifier) == nil &&
		b.imports[name] == "" && !b.declared[name]
}

func (b *memoryViewBindings) byteSlice(expression ast.Expr) bool {
	slice, ok := expression.(*ast.ArrayType)
	return ok && slice.Len == nil && b.builtin(slice.Elt, "byte")
}

func (b *memoryViewBindings) receiver(function *ast.FuncDecl) *ast.Ident {
	if function.Recv == nil || len(function.Recv.List) != 1 || len(function.Recv.List[0].Names) != 1 {
		return nil
	}
	pointer, ok := function.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return nil
	}
	name, ok := pointer.X.(*ast.Ident)
	if !ok || viewDeclaration(name) != b.module {
		return nil
	}
	return function.Recv.List[0].Names[0]
}

func sameViewIdentifier(a, c *ast.Ident) bool {
	return a != nil && c != nil && viewDeclaration(a) != nil && viewDeclaration(a) == viewDeclaration(c) && a.Name == c.Name
}

func (b *memoryViewBindings) memoryArgument(expression ast.Expr, receiver *ast.Ident) bool {
	field, ok := expression.(*ast.SelectorExpr)
	if !ok || field.Sel.Name != "memory" {
		return false
	}
	base, ok := field.X.(*ast.Ident)
	return ok && sameViewIdentifier(base, receiver)
}

func (b *memoryViewBindings) boundCall(call *ast.CallExpr, name string) bool {
	identifier, ok := call.Fun.(*ast.Ident)
	return ok && identifier.Name == name && b.functions[name] != nil &&
		viewDeclaration(identifier) == b.functions[name] && !call.Ellipsis.IsValid()
}

func (b *memoryViewBindings) conversion(expression ast.Expr, name string) (ast.Expr, bool) {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() || !b.builtin(call.Fun, name) {
		return nil, false
	}
	return call.Args[0], true
}

func (b *memoryViewBindings) localI32(identifier *ast.Ident, depth int) bool {
	if identifier == nil || depth > 16 || identifier.Name == "_" {
		return false
	}
	if !b.locals[viewDeclaration(identifier)] {
		return false
	}
	switch declaration := viewDeclaration(identifier).(type) {
	case *ast.Field:
		return b.builtin(declaration.Type, "int32")
	case *ast.ValueSpec:
		return b.builtin(declaration.Type, "int32")
	case *ast.AssignStmt:
		if declaration.Tok != token.DEFINE || len(declaration.Lhs) != len(declaration.Rhs) {
			return false
		}
		for i, expression := range declaration.Lhs {
			name, ok := expression.(*ast.Ident)
			if ok && name.Name == identifier.Name {
				if _, converted := b.conversion(declaration.Rhs[i], "int32"); converted {
					return true
				}
				if name, ok := declaration.Rhs[i].(*ast.Ident); ok {
					return b.localI32(name, depth+1)
				}
			}
		}
	}
	return false
}
