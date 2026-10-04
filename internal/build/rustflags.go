package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

func cargoEnvironment(ctx context.Context, runner Runner, directory, moduleRoot, rust string, memory uint64, sources []sourcePath) ([]string, error) {
	output, err := runner.Run(ctx, directory, "rustup", "run", rust, "rustc", "--print", "sysroot")
	if err != nil {
		return nil, err
	}
	cargoHome := os.Getenv("CARGO_HOME")
	if cargoHome == "" {
		homeDirectory, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return nil, homeErr
		}
		cargoHome = filepath.Join(homeDirectory, ".cargo")
	} else if !filepath.IsAbs(cargoHome) {
		cargoHome = filepath.Join(directory, "shim", cargoHome)
	}
	sysroot := strings.TrimSpace(string(output))
	flags, err := pathFlags(moduleRoot, sysroot, cargoHome, memory, sources)
	if err != nil {
		return nil, err
	}
	lock, err := os.ReadFile(filepath.Join(directory, "shim", "Cargo.lock")) // #nosec G304 -- The selected project's locked dependency graph supplies compiler identity.
	if err != nil {
		return nil, err
	}
	seed := digest(append([]byte(digest(compilerNormalizationSource)+"\x00"+rust+"\x00"), lock...))
	// Rustflags are fingerprinted by Cargo, unlike arbitrary environment
	// variables. This also invalidates cached units when normalization changes.
	flags = append(flags, "-Cmetadata="+seed)
	executable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	compiler := filepath.Join(sysroot, "bin", "rustc")
	if runtime.GOOS == "windows" {
		compiler += ".exe"
	}
	return []string{"RUSTC_BOOTSTRAP=1", "CARGO_TARGET_DIR=" + filepath.Join(directory, ".cocoon-build", "target"), "CARGO_ENCODED_RUSTFLAGS=" + strings.Join(flags, "\x1f"), "RUSTC=" + compiler, "RUSTC_WRAPPER=" + executable, "RUSTC_WORKSPACE_WRAPPER=", "COCOON_RUSTC_WRAPPER=1", "COCOON_METADATA_SEED=" + seed}, nil
}

func pathFlags(moduleRoot, sysroot, cargoHome string, memory uint64, sources []sourcePath) ([]string, error) {
	paths := append([]sourcePath{
		{name: "module", path: moduleRoot},
		{name: "rust", path: sysroot},
		{name: "cargo", path: cargoHome},
	}, sources...)
	for index, source := range paths {
		if !filepath.IsAbs(source.path) || strings.ContainsAny(source.path, "\x00\x1f") {
			return nil, fmt.Errorf("invalid Rust source prefix for %s", source.name)
		}
		paths[index].path = filepath.Clean(source.path)
	}
	// rustc uses the last matching prefix, so more specific roots come last.
	slices.SortFunc(paths, func(a, b sourcePath) int {
		if length := len(a.path) - len(b.path); length != 0 {
			return length
		}
		return strings.Compare(a.name, b.name)
	})
	flags := []string{"-C", fmt.Sprintf("link-arg=--max-memory=%d", memory)}
	for _, source := range paths {
		flags = append(flags, "--remap-path-prefix="+source.path+"=/cocoon/"+source.name)
	}
	return flags, nil
}
