package build

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/darccio/cocoon/internal/manifest"
)

// SourceHash identifies a local source tree without leaking build-machine paths.
type SourceHash struct {
	Name     string `json:"name"`
	Revision string `json:"revision,omitempty"`
	SHA256   string `json:"sha256"`
}

func sourceIdentities(ctx context.Context, runner Runner, directory string, m *manifest.Manifest) ([]SourceHash, error) {
	identities := make([]SourceHash, 0, len(m.Sources)+1)
	for _, source := range m.Sources {
		path := filepath.Join(directory, filepath.FromSlash(source.Path))
		arguments := []string{"--git-dir=" + filepath.Join(path, ".git"), "--work-tree=" + path}
		output, err := runner.Run(ctx, directory, "git", append(slices.Clone(arguments), "rev-parse", "HEAD")...)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(output)) != source.Revision {
			return nil, fmt.Errorf("source %s must be at revision %s", source.Name, source.Revision)
		}
		if _, diffErr := runner.Run(ctx, directory, "git", append(arguments, "diff", "--quiet", "HEAD", "--")...); diffErr != nil {
			return nil, fmt.Errorf("source %s has modified tracked files: %w", source.Name, diffErr)
		}
		hash, err := sourceDigest(path)
		if err != nil {
			return nil, err
		}
		identities = append(identities, SourceHash{Name: source.Name, Revision: source.Revision, SHA256: hash})
	}
	metadata, err := runner.Run(ctx, filepath.Join(directory, "shim"), "rustup", "run", m.Toolchain.Rust, "cargo", "metadata", "--locked", "--offline", "--no-deps", "--format-version=1")
	if err != nil {
		return nil, err
	}
	var cargo struct {
		Packages []struct {
			Dependencies []struct {
				Name string `json:"name"`
				Path string `json:"path"`
			} `json:"dependencies"`
		} `json:"packages"`
	}
	if decodeErr := json.Unmarshal(metadata, &cargo); decodeErr != nil {
		return nil, fmt.Errorf("read Cargo metadata: %w", decodeErr)
	}
	for _, pkg := range cargo.Packages {
		for _, dependency := range pkg.Dependencies {
			if dependency.Path == "" {
				continue
			}
			hash, hashErr := sourceDigest(dependency.Path)
			if hashErr != nil {
				return nil, hashErr
			}
			identities = append(identities, SourceHash{Name: dependency.Name, SHA256: hash})
		}
	}
	if !slices.ContainsFunc(identities, func(source SourceHash) bool { return source.Name == "cocoon-guest" }) {
		return nil, fmt.Errorf("shim must depend on the local cocoon-guest crate")
	}
	slices.SortFunc(identities, func(a, b SourceHash) int { return strings.Compare(a.Name, b.Name) })
	return identities, nil
}
