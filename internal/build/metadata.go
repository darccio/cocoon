package build

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// Embedding the normalizer fingerprints its exact implementation without
// tying deterministic artifacts to the host Go compiler or binary path.
//
//go:embed metadata.go
var compilerNormalizationSource []byte

// RustcWrapper executes Cargo's compiler with location-independent crate IDs.
// Cargo hashes absolute paths for crates outside the workspace, even when rustc
// remaps source paths. Keep its output filenames, but normalize symbol metadata.
func RustcWrapper(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("missing Rust compiler")
	}
	arguments, err := compilerArguments(args[1:], os.Getenv("CARGO_PKG_NAME"), os.Getenv("CARGO_PKG_VERSION"), os.Getenv("CARGO_MANIFEST_DIR"), os.Getenv("COCOON_METADATA_SEED"))
	if err != nil {
		return err
	}
	command := exec.CommandContext(ctx, args[0], arguments...) // #nosec G204 -- Cargo supplies its pinned compiler and argument vector; no shell is invoked.
	command.Stdin = os.Stdin
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("rust compiler: %w", err)
	}
	return nil
}

func compilerArguments(args []string, name, version, directory, seed string) ([]string, error) {
	target := slices.Index(args, "--target")
	wasm := slices.Contains(args, "--target=wasm32-unknown-unknown") || (target >= 0 && target+1 < len(args) && args[target+1] == "wasm32-unknown-unknown")
	if name == "" || !wasm {
		return args, nil // Compiler probes and host build scripts retain Cargo's IDs.
	}
	if seed == "" {
		return nil, fmt.Errorf("missing Cocoon compiler metadata seed")
	}
	var prefixes []sourcePath
	for _, argument := range args {
		if value, ok := strings.CutPrefix(argument, "--remap-path-prefix="); ok {
			separator := strings.LastIndex(value, "=")
			if separator <= 0 || separator+1 == len(value) {
				return nil, fmt.Errorf("invalid compiler source remapping")
			}
			prefixes = append(prefixes, sourcePath{path: value[:separator], name: value[separator+1:]})
		}
	}
	identity, mapped := compilerPath(directory, prefixes)
	if !mapped {
		return nil, fmt.Errorf("unmapped compiler source root for %s", name)
	}
	canonical := []string{"cocoon-metadata-v1", seed, name, version, identity}
	arguments := make([]string, 0, len(args)+1)
	for index := 0; index < len(args); index++ {
		argument := args[index]
		if argument == "-C" && index+1 < len(args) {
			option := args[index+1]
			if strings.HasPrefix(option, "metadata=") {
				index++
				continue
			}
			if strings.HasPrefix(option, "extra-filename=") {
				arguments = append(arguments, argument, option)
				index++
				continue
			}
		}
		if strings.HasPrefix(argument, "-Cmetadata=") {
			continue
		}
		arguments = append(arguments, argument)
		option, _, joined := strings.Cut(argument, "=")
		if compilerDiagnostic(option) {
			if !joined {
				if index+1 == len(args) {
					return nil, fmt.Errorf("missing compiler argument for %s", argument)
				}
				index++
				arguments = append(arguments, args[index])
			}
			continue
		}
		if strings.HasPrefix(argument, "--remap-path-prefix=") || strings.HasPrefix(argument, "-Cextra-filename=") {
			continue
		}
		if argument == "--out-dir" || argument == "-L" || argument == "--extern" {
			if index+1 == len(args) {
				return nil, fmt.Errorf("missing compiler argument for %s", argument)
			}
			index++
			value := args[index]
			arguments = append(arguments, value)
			if argument == "--extern" {
				dependency, _, _ := strings.Cut(value, "=")
				canonical = append(canonical, argument, dependency)
			}
			continue
		}
		if value, ok := compilerPath(argument, prefixes); ok {
			argument = value
		}
		canonical = append(canonical, argument)
	}
	return append(arguments, "-Cmetadata="+digest([]byte(strings.Join(canonical, "\x00")))), nil
}

func compilerDiagnostic(option string) bool {
	switch option {
	case "--error-format", "--json", "--diagnostic-width", "--color":
		return true
	default:
		return false
	}
}

func compilerPath(path string, prefixes []sourcePath) (string, bool) {
	for index := len(prefixes) - 1; index >= 0; index-- {
		prefix := prefixes[index]
		if path == prefix.path {
			return prefix.name, true
		}
		if strings.HasPrefix(path, prefix.path+string(filepath.Separator)) {
			return prefix.name + filepath.ToSlash(strings.TrimPrefix(path, prefix.path)), true
		}
	}
	return path, false
}
