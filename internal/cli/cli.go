// Package cli implements Cocoon's command-line workflows with testable IO.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"golang.org/x/mod/modfile"

	"dario.cat/cocoon/internal/build"
	"dario.cat/cocoon/internal/gen"
	"dario.cat/cocoon/internal/manifest"
)

// Run executes one Cocoon command and returns errors for the caller to report.
func Run(ctx context.Context, args []string, output io.Writer) (err error) {
	if os.Getenv("COCOON_RUSTC_WRAPPER") == "1" {
		return build.RustcWrapper(ctx, args, output, os.Stderr)
	}
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		_, err = fmt.Fprintln(output, "usage: cocoon <init|gen|build|verify|doctor> [options]")
		return err
	}
	if args[0] == "init" {
		return initialize(ctx, args[1:], output)
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	flags.SetOutput(output)
	manifestPath := flags.String("manifest", "cocoon.toml", "path to the Cocoon manifest")
	if parseErr := flags.Parse(args[1:]); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	if !slices.Contains([]string{"gen", "build", "verify", "doctor"}, args[0]) {
		return fmt.Errorf("unknown command %q", args[0])
	}
	absolute, err := filepath.Abs(*manifestPath)
	if err != nil {
		return err
	}
	directory := filepath.Dir(absolute)
	data, err := os.ReadFile(absolute) // #nosec G304 -- Reading the user-selected manifest is the CLI's explicit purpose.
	var m *manifest.Manifest
	if err != nil {
		if args[0] != "doctor" || !errors.Is(err, os.ErrNotExist) {
			return err
		}
		m = &manifest.Manifest{Toolchain: manifest.Toolchain{Rust: manifest.RustVersion, Binaryen: manifest.BinaryenVersion, Wasm2Go: manifest.Wasm2GoVersion}}
	} else {
		m, err = manifest.Parse(data)
		if err != nil {
			return err
		}
	}
	switch args[0] {
	case "doctor":
		if flags.NArg() != 0 {
			return fmt.Errorf("doctor takes no positional arguments")
		}
		root, moduleErr := build.FindModule(directory)
		if moduleErr != nil {
			return moduleErr
		}
		versions, doctorErr := build.Doctor(ctx, build.ExecRunner{}, root, m.Toolchain)
		if doctorErr != nil {
			return doctorErr
		}
		_, err = fmt.Fprintf(output, "rust %s; binaryen %s; wasm2go %s: ready\n", versions.Rust, versions.Binaryen, versions.Wasm2Go)
		return err
	case "gen":
		if flags.NArg() != 0 {
			return fmt.Errorf("gen takes no positional arguments")
		}
		return gen.Project(ctx, directory, m)
	case "build":
		if flags.NArg() != 0 {
			return fmt.Errorf("build takes no positional arguments")
		}
		lock, buildErr := build.Project(ctx, directory, m)
		if buildErr != nil {
			return buildErr
		}
		_, err = fmt.Fprintf(output, "built %s; wasm sha256 %s\n", m.Package.Name, lock.Wasm)
		return err
	case "verify":
		if flags.NArg() != 1 {
			return fmt.Errorf("verify requires a wasm file")
		}
		wasm, readErr := os.ReadFile(flags.Arg(0))
		if readErr != nil {
			return readErr
		}
		if _, verifyErr := build.Verify(m, wasm); verifyErr != nil {
			return verifyErr
		}
		work, tempErr := os.MkdirTemp("", "cocoon-verify-")
		if tempErr != nil {
			return tempErr
		}
		defer func() {
			if removeErr := os.RemoveAll(work); removeErr != nil {
				err = errors.Join(err, removeErr)
			}
		}()
		wasmPath := filepath.Join(work, "module.wasm")
		if writeErr := os.WriteFile(wasmPath, wasm, 0o600); writeErr != nil { // #nosec G703 -- The destination is a fixed filename inside a newly allocated temporary directory.
			return writeErr
		}
		arguments := append([]string{wasmPath, "-o", filepath.Join(work, "checked.wasm")}, build.Features()...)
		if _, runErr := (build.ExecRunner{}).Run(ctx, directory, "wasm-opt", arguments...); runErr != nil {
			return runErr
		}
		_, err = fmt.Fprintln(output, "verified ABI, capabilities, memory limits, and instruction features")
		return err
	default:
		return fmt.Errorf("unknown command")
	}
}

func initialize(ctx context.Context, args []string, output io.Writer) error {
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.SetOutput(output)
	name := flags.String("name", "example", "generated package name")
	importPath := flags.String("import", "", "generated Go import path (inferred within an existing module)")
	guestPath := flags.String("guest", "", "path to the cocoon-guest Rust crate")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() > 1 {
		return fmt.Errorf("init takes at most one directory")
	}
	directory := "."
	if flags.NArg() == 1 {
		directory = flags.Arg(0)
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return err
	}
	root, err := build.FindModule(directory)
	if err != nil {
		return fmt.Errorf("init requires an existing Go module: %w", err)
	}
	if *guestPath == "" {
		*guestPath = filepath.Join(root, "rust", "cocoon-guest")
	}
	guestAbsolute, err := filepath.Abs(*guestPath)
	if err != nil {
		return err
	}
	*guestPath = guestAbsolute
	if _, err = os.Stat(filepath.Join(*guestPath, "Cargo.toml")); err != nil {
		return fmt.Errorf("supply --guest with the cocoon-guest crate: %w", err)
	}
	if *importPath == "" {
		moduleData, readErr := os.ReadFile(filepath.Join(root, "go.mod")) // #nosec G304 -- FindModule selects the enclosing Go module's metadata.
		if readErr != nil {
			return readErr
		}
		modulePath := modfile.ModulePath(moduleData)
		if modulePath == "" {
			return fmt.Errorf("cannot infer Go module path")
		}
		relative, relErr := filepath.Rel(root, directory)
		if relErr != nil {
			return relErr
		}
		*importPath = modulePath + "/" + filepath.ToSlash(filepath.Join(relative, "go", *name))
	}
	manifestData := fmt.Sprintf("[package]\nname=%q\ngo_import=%q\nrust_crate=%q\n[limits]\nmax_input=\"64KiB\"\nmax_output=\"64KiB\"\nmax_memory=\"16MiB\"\ninstances=\"1\"\n[[func]]\nname=\"echo\"\nparams=[{name=\"input\",type=\"string\"}]\nreturns=\"string\"\nfallible=true\n", *name, *importPath, "cocoon-"+*name+"-shim")
	m, err := manifest.Parse([]byte(manifestData))
	if err != nil {
		return err
	}
	guestRelative, err := filepath.Rel(filepath.Join(directory, "shim"), *guestPath)
	if err != nil {
		return err
	}
	cargo := fmt.Sprintf("[package]\nname=%q\nversion=\"0.1.0\"\nedition=\"2024\"\n[workspace]\n[lib]\ncrate-type=[\"cdylib\",\"rlib\"]\n[dependencies]\ncocoon-guest={path=%q}\n[profile.release]\nopt-level=3\nlto=true\ncodegen-units=1\npanic=\"abort\"\nstrip=\"symbols\"\n", m.Package.RustCrate, filepath.ToSlash(guestRelative))
	files := map[string][]byte{
		"cocoon.toml": []byte(manifestData), "shim/Cargo.toml": []byte(cargo),
		"shim/src/lib.rs":            []byte("mod cocoon_gen;\n#[forbid(unsafe_code)]\nmod implementation;\npub use implementation::Shim;\n"),
		"shim/src/implementation.rs": []byte("#[derive(Default)]\npub struct Shim;\nimpl crate::cocoon_gen::API for Shim {\n    fn echo(&mut self, input: String) -> cocoon_guest::Result<String> {\n        Ok(input)\n    }\n}\n"),
	}
	for path := range files {
		if _, err = os.Lstat(filepath.Join(directory, filepath.FromSlash(path))); err == nil {
			return fmt.Errorf("refuse to overwrite %s", path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	var artifacts []gen.Artifact
	for path, data := range files {
		artifacts = append(artifacts, gen.Artifact{Path: filepath.Join(directory, filepath.FromSlash(path)), Data: data})
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if publicationErr := gen.WriteSet(artifacts); publicationErr != nil {
		return publicationErr
	}
	if generationErr := gen.Project(ctx, directory, m); generationErr != nil {
		return generationErr
	}
	_, err = (build.ExecRunner{}).Run(ctx, filepath.Join(directory, "shim"), "rustup", "run", m.Toolchain.Rust, "cargo", "generate-lockfile", "--offline")
	return err
}
