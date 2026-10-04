package build

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/darccio/cocoon/internal/manifest"
)

// Runner is injectable so toolchain checks can be tested without launching tools.
type Runner interface {
	Run(ctx context.Context, directory, program string, args ...string) ([]byte, error)
}

// ExecRunner executes trusted tool names with bounded output and no shell expansion.
type ExecRunner struct{ Environment []string }

type boundedBuffer struct{ bytes.Buffer }

func (b *boundedBuffer) Write(data []byte) (int, error) {
	size := len(data)
	if b.Len() < 1048576 {
		_, err := b.Buffer.Write(data[:min(len(data), 1048576-b.Len())])
		if err != nil {
			return 0, err
		}
	}
	return size, nil
}

// Run invokes a tool in a specific directory and retains at most one MiB of diagnostics.
func (r ExecRunner) Run(ctx context.Context, directory, program string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, program, args...) // #nosec G204 -- Programs are selected by the build pipeline; no shell is invoked.
	command.Dir = directory
	command.Env = append(os.Environ(), r.Environment...)
	var output boundedBuffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		return output.Bytes(), fmt.Errorf("%s: %w: %s", program, err, output.String())
	}
	return output.Bytes(), nil
}

// ToolVersions records checked version identities without absolute paths or timestamps.
type ToolVersions struct {
	BinaryenDigests  map[string]string `json:"binaryen_executable_sha256,omitempty"`
	CompilerMetadata string            `json:"compiler_metadata_sha256"`
	Rust             string            `json:"rust"`
	Binaryen         string            `json:"binaryen"`
	Wasm2Go          string            `json:"wasm2go"`
}

// Doctor verifies every pinned executable and the required Rust target components.
func Doctor(ctx context.Context, runner Runner, directory string, pins manifest.Toolchain) (ToolVersions, error) {
	checks := []struct {
		program  string
		pattern  string
		expected string
		args     []string
	}{
		{"rustup", `^rustc ([0-9.]+) `, pins.Rust, []string{"run", pins.Rust, "rustc", "--version"}},
		{"wasm-opt", `^wasm-opt version ([0-9]+)`, pins.Binaryen, []string{"--version"}},
		{"wasm-metadce", `^wasm-metadce version ([0-9]+)`, pins.Binaryen, []string{"--version"}},
		{"wasm-as", `^wasm-as version ([0-9]+)`, pins.Binaryen, []string{"--version"}},
		{"go", `^wasm2go (v[0-9.]+)`, pins.Wasm2Go, []string{"tool", "wasm2go", "-version"}},
	}
	for _, check := range checks {
		output, err := runner.Run(ctx, directory, check.program, check.args...)
		if err != nil {
			return ToolVersions{}, err
		}
		match := regexp.MustCompile(check.pattern).FindStringSubmatch(strings.TrimSpace(string(output)))
		if len(match) != 2 || match[1] != check.expected {
			return ToolVersions{}, fmt.Errorf("%s version mismatch: expected %s, got %s", check.program, check.expected, bytes.TrimSpace(output))
		}
	}
	components, err := runner.Run(ctx, directory, "rustup", "component", "list", "--installed", "--toolchain", pins.Rust)
	if err != nil {
		return ToolVersions{}, err
	}
	if !strings.Contains(string(components), "rust-src\n") || !strings.Contains(string(components), "rust-std-wasm32-unknown-unknown\n") {
		return ToolVersions{}, fmt.Errorf("rust %s requires rust-src and wasm32-unknown-unknown", pins.Rust)
	}
	versions := ToolVersions{Rust: pins.Rust, Binaryen: pins.Binaryen, Wasm2Go: pins.Wasm2Go, CompilerMetadata: digest(compilerNormalizationSource)}
	if _, executable := runner.(ExecRunner); executable {
		versions.BinaryenDigests = make(map[string]string)
		for _, program := range []string{"wasm-as", "wasm-metadce", "wasm-opt"} {
			path, pathErr := exec.LookPath(program)
			if pathErr != nil {
				return ToolVersions{}, pathErr
			}
			hash, hashErr := executableDigest(path)
			if hashErr != nil {
				return ToolVersions{}, hashErr
			}
			versions.BinaryenDigests[program] = hash
		}
	}
	return versions, nil
}

func executableDigest(path string) (encoded string, err error) {
	file, err := os.Open(path) // #nosec G304 -- Fingerprint the executable selected by the same PATH lookup used to invoke it.
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
