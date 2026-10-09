package build

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"dario.cat/cocoon/internal/manifest"
)

// SourceHash identifies a local source tree without leaking build-machine paths.
type SourceHash struct {
	Name     string `json:"name"`
	Revision string `json:"revision,omitempty"`
	SHA256   string `json:"sha256"`
}

type sourcePath struct {
	name string
	path string
}

func sourceIdentities(ctx context.Context, runner Runner, directory string, m *manifest.Manifest) ([]SourceHash, []sourcePath, error) {
	identities := make([]SourceHash, 0, len(m.Sources)+1)
	var paths []sourcePath
	for _, source := range m.Sources {
		path := filepath.Join(directory, filepath.FromSlash(source.Path))
		arguments := []string{"--git-dir=" + filepath.Join(path, ".git"), "--work-tree=" + path}
		output, err := runner.Run(ctx, directory, "git", append(slices.Clone(arguments), "rev-parse", "HEAD")...)
		if err != nil {
			return nil, nil, err
		}
		if strings.TrimSpace(string(output)) != source.Revision {
			return nil, nil, fmt.Errorf("source %s must be at revision %s", source.Name, source.Revision)
		}
		if _, diffErr := runner.Run(ctx, directory, "git", append(arguments, "diff", "--quiet", "HEAD", "--")...); diffErr != nil {
			return nil, nil, fmt.Errorf("source %s has modified tracked files: %w", source.Name, diffErr)
		}
		hash, err := sourceDigest(path)
		if err != nil {
			return nil, nil, err
		}
		identities = append(identities, SourceHash{Name: source.Name, Revision: source.Revision, SHA256: hash})
		paths = append(paths, sourcePath{name: "sources/" + source.Name, path: path})
	}
	metadata, err := runner.Run(ctx, filepath.Join(directory, "shim"), "rustup", "run", m.Toolchain.Rust, "cargo", "metadata", "--locked", "--offline", "--format-version=1")
	if err != nil {
		return nil, nil, err
	}
	var cargo struct {
		Packages []struct {
			Name         string  `json:"name"`
			ManifestPath string  `json:"manifest_path"`
			Source       *string `json:"source"`
			Dependencies []struct {
				Name string `json:"name"`
				Path string `json:"path"`
			} `json:"dependencies"`
		} `json:"packages"`
	}
	if decodeErr := json.Unmarshal(metadata, &cargo); decodeErr != nil {
		return nil, nil, fmt.Errorf("read Cargo metadata: %w", decodeErr)
	}
	for _, pkg := range cargo.Packages {
		if pkg.Name != "" && pkg.Name != m.Package.RustCrate {
			continue
		}
		for _, dependency := range pkg.Dependencies {
			if dependency.Path == "" {
				continue
			}
			hash, hashErr := sourceDigest(dependency.Path)
			if hashErr != nil {
				return nil, nil, hashErr
			}
			identities = append(identities, SourceHash{Name: dependency.Name, SHA256: hash})
			paths = append(paths, sourcePath{name: "crates/" + dependency.Name, path: dependency.Path})
		}
	}
	// Cargo's complete graph also identifies patches and transitive local
	// crates. They need canonical compiler identities even when their contents
	// are already covered by a declared source repository.
	localPaths := make(map[string]string)
	for _, pkg := range cargo.Packages {
		if pkg.Source != nil || pkg.ManifestPath == "" {
			continue
		}
		path := filepath.Dir(pkg.ManifestPath)
		if previous, exists := localPaths[pkg.Name]; exists && previous != path {
			return nil, nil, fmt.Errorf("ambiguous local crate name %s", pkg.Name)
		}
		localPaths[pkg.Name] = path
		if !slices.ContainsFunc(paths, func(source sourcePath) bool { return source.path == path }) {
			paths = append(paths, sourcePath{name: "crates/" + pkg.Name, path: path})
		}
	}
	if !slices.ContainsFunc(identities, func(source SourceHash) bool { return source.Name == "cocoon-guest" }) {
		return nil, nil, fmt.Errorf("shim must depend on the local cocoon-guest crate")
	}
	slices.SortFunc(identities, func(a, b SourceHash) int { return strings.Compare(a.Name, b.Name) })
	return identities, paths, nil
}
