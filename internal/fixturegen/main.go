// Command fixturegen regenerates Cocoon's independent translator contract fixtures.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/darccio/cocoon/internal/build"
	"github.com/darccio/cocoon/internal/gen"
	"github.com/darccio/cocoon/internal/harden"
	"github.com/darccio/cocoon/internal/manifest"
)

func main() {
	if err := generate(context.Background()); err != nil {
		if _, writeErr := fmt.Fprintln(os.Stderr, err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func generate(ctx context.Context) (err error) {
	work, err := os.MkdirTemp("", "cocoon-fixtures-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(work)) }()
	directory, err := os.Getwd()
	if err != nil {
		return err
	}
	runner := build.ExecRunner{}
	if _, doctorErr := build.Doctor(ctx, runner, directory, manifest.Toolchain{Rust: manifest.RustVersion, Binaryen: manifest.BinaryenVersion, Wasm2Go: manifest.Wasm2GoVersion}); doctorErr != nil {
		return doctorErr
	}
	wasmPath := filepath.Join(work, "traps.wasm")
	arguments := append([]string{"traps.wat", "-o", wasmPath}, build.Features()...)
	if _, assembleErr := runner.Run(ctx, directory, "wasm-as", arguments...); assembleErr != nil {
		return assembleErr
	}
	goPath := filepath.Join(work, "module.go")
	if _, translateErr := runner.Run(ctx, directory, "go", "tool", "wasm2go", "-unsafe", "-pkg", "trapfix", "-o", goPath, wasmPath); translateErr != nil {
		return translateErr
	}
	raw, err := os.ReadFile(goPath) // #nosec G304 -- Read the tool output in our private temporary directory.
	if err != nil {
		return err
	}
	required, err := harden.Required(raw)
	if err != nil {
		return err
	}
	translated, err := harden.Rewrite(raw, required)
	if err != nil {
		return err
	}
	wasm, err := os.ReadFile(wasmPath) // #nosec G304 -- Read the assembled fixture in our private temporary directory.
	if err != nil {
		return err
	}
	return gen.WriteSet([]gen.Artifact{
		{Path: "module.go", Data: translated, Generated: true},
		{Path: "testdata/raw.go.txt", Data: raw, Generated: true},
		{Path: "testdata/module.wasm", Data: wasm},
	})
}
