// Package gen writes validated and formatted Cocoon generated artifacts.
package gen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"

	"dario.cat/cocoon/internal/gen/gogen"
	"dario.cat/cocoon/internal/gen/rustgen"
	"dario.cat/cocoon/internal/manifest"
)

// Project generates the trait and facade; build finalizes the typed adapter.
func Project(ctx context.Context, directory string, m *manifest.Manifest) (err error) {
	release, err := Guard(directory)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, release()) }()
	rust, facade, err := Sources(ctx, m)
	if err != nil {
		return err
	}
	return WriteSet([]Artifact{
		{Path: filepath.Join(directory, "shim", "src", "cocoon_gen.rs"), Data: rust, Generated: true},
		{Path: filepath.Join(directory, "go", m.Package.Name, "cocoon_gen.go"), Data: facade, Generated: true},
	})
}

// Sources validates and formats all source artifacts without changing the project.
func Sources(ctx context.Context, m *manifest.Manifest) (rustSource, facadeSource []byte, err error) {
	rust, err := rustgen.Generate(m)
	if err != nil {
		return nil, nil, err
	}
	facade, err := gogen.Facade(m)
	if err != nil {
		return nil, nil, err
	}
	command := exec.CommandContext(ctx, "rustup", "run", m.Toolchain.Rust, "rustfmt", "--edition", "2024", "--emit", "stdout") // #nosec G204 -- Validate enforces the supported Rust pin; no shell is invoked.
	command.Stdin = bytes.NewReader(rust)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	formatted, err := command.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("format Rust glue: %w: %s", err, stderr.String())
	}
	return formatted, facade, nil
}
