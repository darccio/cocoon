package rtgen_test

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"dario.cat/cocoon/internal/gen/rtgen"
)

func TestUpdateOnlyCheckedRegion(t *testing.T) {
	t.Parallel()
	existing := []byte("package rt\nconst before=17\n// BEGIN generated checked adapters\nstale code\n// END generated checked adapters\nconst after=42\n")
	before := bytes.Clone(existing)
	updated, err := rtgen.Update(existing)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(existing, before) {
		t.Fatal("update changed its input")
	}
	text := string(updated)
	if strings.Contains(text, "stale code") || !strings.Contains(text, "const before = 17") || !strings.Contains(text, "const after = 42") {
		t.Fatal("update did not preserve authored surroundings")
	}
	again, err := rtgen.Update(updated)
	if err != nil || !bytes.Equal(updated, again) {
		t.Fatal("update is not reproducible", err)
	}
}

func TestUpdateRejectsAmbiguousSource(t *testing.T) {
	t.Parallel()
	const start = "// BEGIN generated checked adapters\n"
	const end = "// END generated checked adapters\n"
	for name, source := range map[string]string{
		"missing":    "package rt\n" + start,
		"duplicate":  "package rt\n" + start + start + end,
		"reversed":   "package rt\n" + end + start,
		"embedded":   "package rt\n// not a marker " + start + end,
		"raw string": "package rt\nconst text = `\n" + start + end + "`\n",
		"block":      "package rt\n/*\n" + start + end + "*/\n",
		"invalid Go": "package broken {\n" + start + end,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if output, err := rtgen.Update([]byte(source)); err == nil || output != nil {
				t.Fatal("invalid source must fail without output", err)
			}
		})
	}
}

func TestAdaptersShareEveryCheckedStatement(t *testing.T) {
	t.Parallel()
	source, err := rtgen.Generate()
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "instance.go", append([]byte("package rt\n"), source...), 0)
	if err != nil {
		t.Fatal(err)
	}
	bodies := make(map[string]string)
	for _, declaration := range file.Decls {
		method, ok := declaration.(*ast.FuncDecl)
		if !ok {
			continue
		}
		if method.Name.Name == "Call" {
			if len(method.Body.List) < 2 {
				t.Fatal("missing locked adapter prefix")
			}
			var prefix strings.Builder
			for _, statement := range method.Body.List[:2] {
				if err := format.Node(&prefix, token.NewFileSet(), statement); err != nil {
					t.Fatal(err)
				}
				prefix.WriteByte('\n')
			}
			if prefix.String() != "i.mu.Lock()\ndefer i.mu.Unlock()\n" {
				t.Fatal("locked adapter must acquire and defer release of execution mutex")
			}
			method.Body.List = method.Body.List[2:]
		}
		var body strings.Builder
		if err := format.Node(&body, token.NewFileSet(), method.Body); err != nil {
			t.Fatal(err)
		}
		bodies[method.Name.Name] = body.String()
	}
	if len(bodies) != 2 || bodies["Call"] == "" || bodies["Call"] != bodies["checked"] {
		t.Fatal("adapters must differ only by the execution lock prefix")
	}
}
