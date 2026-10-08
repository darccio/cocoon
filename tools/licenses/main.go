// Copyright 2026 Cocoon contributors
// SPDX-License-Identifier: Apache-2.0

// Command licenses inventories resolved dependencies using only the Go standard
// library, go list/mod download, and cargo metadata. See README.md in this directory.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type review struct {
	Go      []goReview     `json:"go"`
	Rust    []rustReview   `json:"rust"`
	Notices []noticeReview `json:"notices"`
}

type rustReview struct {
	Package string `json:"package"`
	License string `json:"license"`
	File    string `json:"file"`
	SHA256  string `json:"sha256"`
}

type goReview struct {
	Module  string `json:"module"`
	License string `json:"license"`
	File    string `json:"file"`
	SHA256  string `json:"sha256"`
}

type noticeReview struct {
	Dependency string `json:"dependency"`
	Version    string `json:"version"`
	License    string `json:"license"`
	Source     string `json:"source"`
	File       string `json:"file"`
	SHA256     string `json:"sha256"`
	Pins       []pin  `json:"pins"`
}

type pin struct {
	File string `json:"file"`
	Text string `json:"text"`
}

type evidence struct {
	Name string
	Data []byte
}

type dependency struct {
	Ecosystem string
	Name      string
	Version   string
	License   string
	Source    string
	UsedBy    []string
	Files     []evidence
}

type inventory map[string]*dependency

type goModule struct {
	Replace *goModule `json:"Replace"`
	Error   *struct {
		Err string `json:"Err"`
	} `json:"Error"`
	Path    string `json:"Path"`
	Version string `json:"Version"`
	Dir     string `json:"Dir"`
	Main    bool   `json:"Main"`
}

type cargoPackage struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	License     string `json:"license"`
	LicenseFile string `json:"license_file"`
	Source      string `json:"source"`
	Manifest    string `json:"manifest_path"`
}

type cargoMetadata struct {
	Packages         []cargoPackage `json:"packages"`
	WorkspaceRoot    string         `json:"workspace_root"`
	WorkspaceMembers []string       `json:"workspace_members"`
}

func main() {
	check := flag.Bool("check", false, "fail if the committed inventory or notice bundle differs")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	err := run(ctx, *check)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, check bool) error {
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	var reviewed review
	data, err := os.ReadFile(filepath.Join(root, "tools", "licenses", "reviewed.json")) // #nosec G304 -- Fixed repository configuration, relative to the invocation directory.
	if err != nil {
		return fmt.Errorf("run from the repository root: %w", err)
	}
	if decodeErr := json.Unmarshal(data, &reviewed); decodeErr != nil {
		return decodeErr
	}
	manifests, err := discover(root)
	if err != nil {
		return err
	}
	all := make(inventory)
	workspaces := make(map[string]bool)
	for _, manifest := range manifests {
		fmt.Fprintln(os.Stderr, "Collecting", relative(root, manifest))
		switch filepath.Base(manifest) {
		case "go.mod":
			err = collectGo(ctx, root, manifest, reviewed.Go, all)
		case "Cargo.toml":
			err = collectCargo(ctx, root, manifest, manifests, workspaces, reviewed.Rust, all)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", relative(root, manifest), err)
		}
	}
	if noticeErr := collectNotices(root, reviewed.Notices, all); noticeErr != nil {
		return noticeErr
	}
	outputs, err := render(all)
	if err != nil {
		return err
	}
	if err := save(root, outputs, check); err != nil {
		return err
	}
	fmt.Printf("License inventory: %d entries; CSV and upstream notices are current.\n", len(all))
	return nil
}

func discover(root string) ([]string, error) {
	var manifests []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && path != root && ignoredDirectory(entry.Name()) {
			return fs.SkipDir
		}
		if entry.Type().IsRegular() && (entry.Name() == "go.mod" || entry.Name() == "Cargo.toml") {
			manifests = append(manifests, path)
		}
		return nil
	})
	if err == nil && len(manifests) == 0 {
		return nil, errors.New("no dependency manifests found")
	}
	slices.Sort(manifests)
	return manifests, err
}

func ignoredDirectory(name string) bool {
	return strings.HasPrefix(name, ".") || slices.Contains([]string{"target", "vendor", "node_modules"}, name)
}

func command(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- Executables and arguments are constructed by this tool; no shell is invoked.
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w\n%s", name, strings.Join(args, " "), err, stderr.String())
	}
	return data, nil
}

func collectGo(ctx context.Context, root, manifest string, reviews []goReview, all inventory) error {
	dir := filepath.Dir(manifest)
	data, err := command(ctx, dir, "go", "list", "-mod=readonly", "-m", "-json", "all")
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var module goModule
		if err = decoder.Decode(&module); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
		if module.Main {
			continue
		}
		if module.Error != nil {
			return errors.New(module.Error.Err)
		}
		dep, depErr := goDependency(ctx, root, module, reviews)
		if depErr != nil {
			return depErr
		}
		dep.UsedBy = []string{relative(root, manifest)}
		if addErr := all.add(&dep); addErr != nil {
			return addErr
		}
	}
}

func goDependency(ctx context.Context, root string, module goModule, reviews []goReview) (dependency, error) {
	actual := module
	if module.Replace != nil {
		actual = *module.Replace
	}
	if actual.Dir == "" {
		downloaded, err := downloadGoModule(ctx, actual.Path, actual.Version)
		if err != nil {
			return dependency{}, err
		}
		actual = downloaded
	}
	if actual.Error != nil {
		return dependency{}, errors.New(actual.Error.Err)
	}
	if actual.Dir == "" {
		return dependency{}, fmt.Errorf("no source directory for %s", actual.Path)
	}
	var approved *goReview
	reviewName := actual.Path
	if actual.Version == "" {
		reviewName = module.Path
	}
	for i := range reviews {
		if reviews[i].Module == reviewName {
			if approved != nil {
				return dependency{}, fmt.Errorf("duplicate review for %s", actual.Path)
			}
			approved = &reviews[i]
		}
	}
	if approved == nil {
		return dependency{}, fmt.Errorf("unreviewed Go module %s@%s: inspect its license and add a fingerprint to tools/licenses/reviewed.json", actual.Path, actual.Version)
	}
	if err := verifyFile(filepath.Join(actual.Dir, approved.File), approved.License, approved.SHA256); err != nil {
		return dependency{}, err
	}
	files, err := licenseFiles(actual.Dir, true, actual.Dir)
	if err != nil {
		return dependency{}, err
	}
	source := "module:" + actual.Path + "@" + actual.Version
	if actual.Version == "" {
		source = "path:" + relative(root, actual.Dir)
	}
	return dependency{Ecosystem: "Go", Name: module.Path, Version: module.Version, License: approved.License, Source: source, Files: files}, nil
}

func downloadGoModule(ctx context.Context, path, version string) (module goModule, err error) {
	// Outside a module, go mod download cannot add sums to a project's go.sum.
	// Source downloads still use the normal module cache and checksum database.
	tmp, err := os.MkdirTemp("", "cocoon-licenses-*")
	if err != nil {
		return module, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(tmp)) }()
	data, err := command(ctx, tmp, "go", "mod", "download", "-json", path+"@"+version)
	if err != nil {
		return module, err
	}
	err = json.Unmarshal(data, &module)
	return module, err
}

func collectCargo(ctx context.Context, root, manifest string, manifests []string, workspaces map[string]bool, reviews []rustReview, all inventory) error {
	if workspaces[manifest] {
		return nil
	}
	data, err := command(ctx, root, "cargo", "metadata", "--locked", "--all-features", "--format-version=1", "--manifest-path", manifest)
	if err != nil {
		return err
	}
	var metadata cargoMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return err
	}
	workspaceManifest := filepath.Join(metadata.WorkspaceRoot, "Cargo.toml")
	workspaces[workspaceManifest] = true
	for _, pkg := range metadata.Packages {
		// Packages authored in this repository are not third-party dependencies.
		if pkg.Source == "" && slices.Contains(manifests, pkg.Manifest) {
			if pkg.License != "Apache-2.0" {
				return fmt.Errorf("cocoon package %s must declare Apache-2.0", pkg.Name)
			}
			workspaces[pkg.Manifest] = true
			continue
		}
		dep, depErr := rustDependency(root, &pkg, reviews)
		if depErr != nil {
			return depErr
		}
		dep.UsedBy = []string{relative(root, workspaceManifest)}
		if addErr := all.add(&dep); addErr != nil {
			return addErr
		}
	}
	return nil
}

func rustDependency(root string, pkg *cargoPackage, reviews []rustReview) (dependency, error) {
	license := strings.TrimSpace(strings.ReplaceAll(pkg.License, "/", " OR "))
	dir := filepath.Dir(pkg.Manifest)
	boundary := dir
	if pkg.Source == "" || strings.HasPrefix(pkg.Source, "git+") {
		boundary = repositoryRoot(dir)
	}
	var extra *evidence
	if license == "" || license == "UNKNOWN" || license == "UNLICENSED" {
		var approved *rustReview
		for i := range reviews {
			if reviews[i].Package == pkg.Name {
				if approved != nil {
					return dependency{}, fmt.Errorf("duplicate review for Rust package %s", pkg.Name)
				}
				approved = &reviews[i]
			}
		}
		if approved == nil {
			return dependency{}, fmt.Errorf("rust package %s@%s has no declared license: inspect its terms and add a fingerprint to the rust section of tools/licenses/reviewed.json", pkg.Name, pkg.Version)
		}
		text, err := readEvidence(filepath.Join(dir, approved.File), approved.File, boundary)
		if err != nil {
			return dependency{}, err
		}
		if approved.License == "" || digest(text.Data) != approved.SHA256 {
			return dependency{}, fmt.Errorf("rust license review does not match %s@%s", pkg.Name, pkg.Version)
		}
		license = approved.License
		extra = &text
	}
	files, err := licenseFiles(dir, true, boundary)
	if err != nil {
		return dependency{}, err
	}
	if extra != nil && !slices.ContainsFunc(files, func(f evidence) bool { return f.Name == extra.Name }) {
		files = append(files, *extra)
	}
	if pkg.LicenseFile != "" {
		path := pkg.LicenseFile
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		text, readErr := readEvidence(path, relative(dir, path), boundary)
		if readErr != nil {
			return dependency{}, readErr
		}
		if !slices.ContainsFunc(files, func(f evidence) bool { return f.Name == text.Name }) {
			files = append(files, text)
		}
	}
	// Path/git workspace crates commonly inherit the repository's root license.
	// Also retain repository notices when the crate already has its own license.
	if boundary != dir {
		for parent := filepath.Dir(dir); ; parent = filepath.Dir(parent) {
			inherited, readErr := licenseFiles(parent, false, boundary)
			if readErr != nil {
				return dependency{}, readErr
			}
			for _, f := range inherited {
				f.Name = relative(dir, filepath.Join(parent, f.Name))
				files = append(files, f)
			}
			if parent == boundary {
				break
			}
		}
	}
	if len(files) == 0 {
		return dependency{}, fmt.Errorf("no license/notice texts for %s@%s", pkg.Name, pkg.Version)
	}
	source := pkg.Source
	if source == "" {
		source = "path:" + relative(root, dir)
	}
	return dependency{Ecosystem: "Rust", Name: pkg.Name, Version: pkg.Version, License: license, Source: source, Files: files}, nil
}

func repositoryRoot(dir string) string {
	workspace := dir
	for parent := dir; ; parent = filepath.Dir(parent) {
		if _, err := os.Stat(filepath.Join(parent, ".git")); err == nil {
			return parent
		}
		manifest, err := os.ReadFile(filepath.Join(parent, "Cargo.toml")) // #nosec G304 -- Looking for an ancestor workspace manifest, never publishing its contents.
		if err == nil && bytes.Contains(manifest, []byte("[workspace]")) {
			workspace = parent
		}
		if parent == filepath.Dir(parent) {
			return workspace
		}
	}
}

func licenseFiles(dir string, recursive bool, boundary string) ([]evidence, error) {
	var files []evidence
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != dir && (!recursive || ignoredDirectory(entry.Name())) {
				return fs.SkipDir
			}
			return nil
		}
		name := strings.ToUpper(entry.Name())
		if !slices.ContainsFunc([]string{"LICENSE", "LICENCE", "UNLICENSE", "COPYING", "NOTICE", "COPYRIGHT", "AUTHORS", "PATENTS"}, func(prefix string) bool {
			return name == prefix || strings.HasPrefix(name, prefix+".") || strings.HasPrefix(name, prefix+"-") || strings.HasPrefix(name, prefix+"_")
		}) {
			return nil
		}
		if strings.HasSuffix(name, ".CSV") || strings.HasSuffix(name, ".JSON") {
			return nil // Inventories are not upstream license texts.
		}
		text, err := readEvidence(path, relative(dir, path), boundary)
		if err != nil {
			return err
		}
		files = append(files, text)
		return nil
	})
	return files, err
}

func readEvidence(path, name, boundary string) (text evidence, err error) {
	root, err := os.OpenRoot(boundary)
	if err != nil {
		return evidence{}, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	rel, err := filepath.Rel(boundary, path)
	if err != nil {
		return evidence{}, err
	}
	info, err := root.Stat(rel)
	if err != nil {
		return evidence{}, err
	}
	if !info.Mode().IsRegular() {
		return evidence{}, fmt.Errorf("nonregular license/notice file %s requires review", path)
	}
	data, err := root.ReadFile(rel)
	if err != nil {
		return evidence{}, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return evidence{}, fmt.Errorf("empty license/notice file %s", path)
	}
	return evidence{Name: name, Data: data}, nil
}

func verifyFile(path, license, expected string) error {
	if license == "" || len(expected) != sha256.Size*2 {
		return fmt.Errorf("invalid license review for %s", path)
	}
	text, err := readEvidence(path, path, filepath.Dir(path))
	if err != nil {
		return err
	}
	if actual := digest(text.Data); actual != expected {
		return fmt.Errorf("license text changed: %s (SHA-256 %s); review before updating tools/licenses/reviewed.json", path, actual)
	}
	return nil
}

func collectNotices(root string, notices []noticeReview, all inventory) error {
	for _, notice := range notices {
		for _, p := range notice.Pins {
			data, err := os.ReadFile(filepath.Join(root, p.File)) // #nosec G304 -- Version-pin paths come from reviewed repository configuration.
			if err != nil {
				return err
			}
			if !bytes.Contains(data, []byte(p.Text)) {
				return fmt.Errorf("%s version pin changed in %s: refresh the reviewed toolchain notice", notice.Dependency, p.File)
			}
		}
		path := filepath.Join(root, notice.File)
		if err := verifyFile(path, notice.License, notice.SHA256); err != nil {
			return err
		}
		text, err := readEvidence(path, notice.File, root)
		if err != nil {
			return err
		}
		if err := all.add(&dependency{Ecosystem: "Toolchain", Name: notice.Dependency, Version: notice.Version, License: notice.License, Source: notice.Source, UsedBy: []string{"generation; see file-specific exceptions in notices"}, Files: []evidence{text}}); err != nil {
			return err
		}
	}
	return nil
}

func (all inventory) add(dep *dependency) error {
	slices.SortFunc(dep.Files, func(a, b evidence) int { return strings.Compare(a.Name, b.Name) })
	dep.Files = slices.CompactFunc(dep.Files, func(a, b evidence) bool { return a.Name == b.Name && bytes.Equal(a.Data, b.Data) })
	key := strings.Join([]string{dep.Ecosystem, dep.Name, dep.Version, dep.Source}, "\x00")
	if old, ok := all[key]; ok {
		if old.License != dep.License || !slices.EqualFunc(old.Files, dep.Files, func(a, b evidence) bool { return a.Name == b.Name && bytes.Equal(a.Data, b.Data) }) {
			return fmt.Errorf("conflicting license evidence for %s@%s", dep.Name, dep.Version)
		}
		old.UsedBy = append(old.UsedBy, dep.UsedBy...)
		return nil
	}
	copyDep := *dep
	copyDep.Files = slices.Clone(dep.Files)
	copyDep.UsedBy = slices.Clone(dep.UsedBy)
	all[key] = &copyDep
	return nil
}

func render(all inventory) (map[string][]byte, error) {
	keys := make([]string, 0, len(all))
	for key := range all {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var table, bundle bytes.Buffer
	writer := csv.NewWriter(&table)
	if err := writer.Write([]string{"ecosystem", "dependency", "version", "license", "source", "used_by", "license_files", "evidence_sha256"}); err != nil {
		return nil, err
	}
	bundle.WriteString("Generated by make licenses. Do not edit.\nSee LICENSE-3rdparty.csv and tools/licenses/README.md for scope.\nUpstream license and attribution texts follow verbatim.\nRust standard-library HTML includes file-specific license exceptions.\n\n")
	for _, key := range keys {
		dep := all[key]
		usedBy := slices.Clone(dep.UsedBy)
		slices.Sort(usedBy)
		usedBy = slices.Compact(usedBy)
		var names, hashes []string
		fmt.Fprintf(&bundle, "===== %s: %s@%s =====\nLicense: %s\nSource: %s\nUsed by: %s\n\n", dep.Ecosystem, dep.Name, dep.Version, dep.License, dep.Source, strings.Join(usedBy, "; "))
		for _, file := range dep.Files {
			names = append(names, file.Name)
			hashes = append(hashes, file.Name+"="+digest(file.Data))
			fmt.Fprintf(&bundle, "----- %s (SHA-256 %s) -----\n", file.Name, digest(file.Data))
			bundle.Write(file.Data)
			bundle.WriteString("\n\n")
		}
		if err := writer.Write([]string{dep.Ecosystem, dep.Name, dep.Version, dep.License, dep.Source, strings.Join(usedBy, "; "), strings.Join(names, "; "), strings.Join(hashes, "; ")}); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	return map[string][]byte{"LICENSE-3rdparty.csv": table.Bytes(), "LICENSE-3rdparty.txt": bundle.Bytes()}, writer.Error()
}

func save(root string, outputs map[string][]byte, check bool) error {
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		names = append(names, name)
	}
	slices.Sort(names)
	var failures []error
	for _, name := range names {
		path := filepath.Join(root, name)
		if check {
			committed, err := os.ReadFile(path) // #nosec G304 -- Output names are fixed by render, relative to the repository root.
			if err != nil || !bytes.Equal(committed, outputs[name]) {
				failures = append(failures, fmt.Errorf("%s is missing or stale; run make licenses and include both inventory files", name))
			}
			continue
		}
		// Resolve and validate everything before replacing any output. Each rename
		// is atomic, so an interrupted write cannot truncate a committed file.
		if err := writeAtomic(path, outputs[name]); err != nil {
			return err
		}
	}
	return errors.Join(failures...)
}

func writeAtomic(path string, data []byte) (err error) {
	file, err := os.CreateTemp(filepath.Dir(path), ".licenses-*")
	if err != nil {
		return err
	}
	defer func() {
		removeErr := os.Remove(file.Name())
		if !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	_, writeErr := file.Write(data)
	err = errors.Join(writeErr, file.Chmod(0o644), file.Close())
	if err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func digest(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func relative(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
