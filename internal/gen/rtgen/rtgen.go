// Package rtgen renders the shared per-operation runtime safety boundary.
package rtgen

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/format"
	"go/scanner"
	"go/token"
	"strings"
	"text/template"
)

//go:embed checked.go.tmpl
var checkedTemplate string

// Generate emits locked and caller-held-lock adapters from one safety body.
func Generate() ([]byte, error) {
	source, err := template.New("checked").Parse(checkedTemplate)
	if err != nil {
		return nil, fmt.Errorf("parse checked boundary: %w", err)
	}
	var output strings.Builder
	adapters := []struct {
		Name   string
		Locked bool
	}{{Name: "Call", Locked: true}, {Name: "checked"}}
	if renderErr := source.Execute(&output, adapters); renderErr != nil {
		return nil, fmt.Errorf("render checked boundary: %w", renderErr)
	}
	formatted, err := format.Source([]byte("package rt\n" + output.String()))
	if err != nil {
		return nil, fmt.Errorf("format checked boundary: %w", err)
	}
	return bytes.TrimPrefix(formatted, []byte("package rt\n\n")), nil
}

// Update replaces only the marked adapters in an authored runtime source file.
func Update(existing []byte) ([]byte, error) {
	const start = "// BEGIN generated checked adapters"
	const end = "// END generated checked adapters"
	if bytes.Count(existing, []byte(start)) != 1 || bytes.Count(existing, []byte(end)) != 1 ||
		!bytes.Contains(existing, []byte("\n"+start+"\n")) || !bytes.Contains(existing, []byte("\n"+end+"\n")) {
		return nil, fmt.Errorf("expected exactly one checked adapter region")
	}
	if !commentMarkers(existing, start, end) {
		return nil, fmt.Errorf("checked adapter markers must be line comments")
	}
	prefix, rest, _ := bytes.Cut(existing, []byte(start))
	_, suffix, found := bytes.Cut(rest, []byte(end))
	if !found {
		return nil, fmt.Errorf("checked adapter markers are out of order")
	}
	body, err := Generate()
	if err != nil {
		return nil, err
	}
	output := append([]byte(nil), prefix...)
	output = append(output, start...)
	output = append(output, "\n\n"...)
	output = append(output, body...)
	output = append(output, end...)
	output = append(output, suffix...)
	formatted, err := format.Source(output)
	if err != nil {
		return nil, fmt.Errorf("format runtime source: %w", err)
	}
	return formatted, nil
}

func commentMarkers(source []byte, start, end string) bool {
	file := token.NewFileSet().AddFile("instance.go", -1, len(source))
	var lexer scanner.Scanner
	lexer.Init(file, source, nil, scanner.ScanComments)
	var foundStart, foundEnd bool
	for {
		_, kind, literal := lexer.Scan()
		if kind == token.EOF {
			return foundStart && foundEnd
		}
		if kind == token.COMMENT {
			foundStart = foundStart || literal == start
			foundEnd = foundEnd || literal == end
		}
	}
}
