// Copyright 2026 Cocoon contributors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestGoGraphIncludesIndirectTestAndToolModulesWithoutEditingLocks(t *testing.T) {
	root := t.TempDir()
	proxy := filepath.Join(root, "proxy")
	cache := filepath.Join(root, "cache")
	proxyURL := &url.URL{Scheme: "file", Path: filepath.ToSlash(proxy)}
	t.Setenv("GOPROXY", proxyURL.String())
	t.Setenv("GOSUMDB", "off") // Invented local fixture modules have no checksum database entries.
	t.Setenv("GOMODCACHE", cache)
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOFLAGS", strings.TrimSpace(os.Getenv("GOFLAGS")+" -modcacherw"))
	var reviews []goReview
	for _, name := range []string{"library", "indirect", "testhelper", "tool"} {
		module := "example.org/" + name
		mod := "module " + module + "\ngo 1.26.0\n"
		source := "package fixture\n"
		if name == "library" {
			mod += "require example.org/indirect v1.0.0\n"
			source += "import _ \"example.org/indirect\"\n"
		}
		if name == "tool" {
			source = "package main\nfunc main() {}\n"
		}
		license := "reviewed fixture copyright and license for " + module + "\n"
		writeFixture(t, proxy, module+"/@v/v1.0.0.mod", mod)
		writeFixture(t, proxy, module+"/@v/v1.0.0.info", `{"Version":"v1.0.0","Time":"2026-01-01T00:00:00Z"}`)
		var archive bytes.Buffer
		zw := zip.NewWriter(&archive)
		for file, data := range map[string]string{"go.mod": mod, "source.go": source, "LICENSE": license} {
			entry, err := zw.Create(module + "@v1.0.0/" + file)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = fmt.Fprint(entry, data); err != nil {
				t.Fatal(err)
			}
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		writeFixture(t, proxy, module+"/@v/v1.0.0.zip", archive.String())
		reviews = append(reviews, goReview{Module: module, License: "MIT", File: "LICENSE", SHA256: digest([]byte(license))})
	}
	project := filepath.Join(root, "project")
	manifest := writeFixture(t, project, "go.mod", "module example.org/project\ngo 1.26.0\ntool example.org/tool\nrequire (\nexample.org/library v1.0.0\nexample.org/testhelper v1.0.0\nexample.org/tool v1.0.0\n)\n")
	writeFixture(t, project, "main.go", "package project\nimport _ \"example.org/library\"\n")
	writeFixture(t, project, "main_test.go", "package project\nimport _ \"example.org/testhelper\"\n")
	if _, err := command(t.Context(), project, "go", "mod", "tidy"); err != nil {
		t.Fatal(err)
	}
	var before [][]byte
	for _, name := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(project, name)) // #nosec G304 -- Fixed fixture filenames under the test directory.
		if err != nil {
			t.Fatal(err)
		}
		before = append(before, data)
	}
	// Remove extracted sources to exercise the isolated download path as well.
	if err := os.RemoveAll(filepath.Join(cache, "example.org")); err != nil {
		t.Fatal(err)
	}
	all := make(inventory)
	if err := collectGo(t.Context(), project, manifest, reviews, all); err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("missing direct/indirect/test/tool dependency: %d entries", len(all))
	}
	for i, name := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(project, name)) // #nosec G304 -- Fixed fixture filenames under the test directory.
		if err != nil || !bytes.Equal(data, before[i]) {
			t.Fatalf("collector rewrote %s: %v", name, err)
		}
	}
}

func writeFixture(t *testing.T, root, name, text string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDiscoveryIncludesExamplesAndSkipsCaches(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"go.mod", "Cargo.toml", "examples/shim/Cargo.toml", "testdata/consumer/go.mod", ".cache/mod/go.mod", "target/Cargo.toml", ".cocoon-build/Cargo.toml", "vendor/thirdparty/go.mod"} {
		writeFixture(t, root, name, "manifest")
	}
	files, err := discover(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, path := range files {
		names = append(names, relative(root, path))
	}
	want := []string{"Cargo.toml", "examples/shim/Cargo.toml", "go.mod", "testdata/consumer/go.mod"}
	if !slices.Equal(names, want) {
		t.Fatalf("manifests = %v, want %v", names, want)
	}
}

func TestGoLicenseFailsClosedAndUsesReplacement(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "LICENSE", "reviewed upstream terms\n")
	writeFixture(t, root, "nested/NOTICE", "additional attribution\n")
	reviews := []goReview{{Module: "example.org/replacement", File: "LICENSE", License: "MIT", SHA256: digest([]byte("reviewed upstream terms\n"))}}
	module := goModule{Path: "example.org/original", Version: "v1.0.0", Replace: &goModule{Path: "example.org/replacement", Version: "v2.0.0", Dir: root}}
	dep, err := goDependency(t.Context(), root, module, reviews)
	if err != nil {
		t.Fatal(err)
	}
	if dep.Name != module.Path || dep.Version != module.Version || dep.Source != "module:example.org/replacement@v2.0.0" || len(dep.Files) != 2 {
		t.Fatalf("replacement not inventoried correctly: %+v", dep)
	}
	if _, err = goDependency(t.Context(), root, module, nil); err == nil || !strings.Contains(err.Error(), "unreviewed") {
		t.Fatalf("unreviewed module accepted: %v", err)
	}
	writeFixture(t, root, "LICENSE", "different terms\n")
	if _, err = goDependency(t.Context(), root, module, reviews); err == nil || !strings.Contains(err.Error(), "license text changed") {
		t.Fatalf("changed upstream terms accepted: %v", err)
	}
}

func TestRustInheritedAndAdditionalNotices(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, ".git/HEAD", "fixture")
	writeFixture(t, root, "LICENSE", "workspace Apache terms\n")
	writeFixture(t, root, "NOTICE", "workspace attribution\n")
	manifest := writeFixture(t, root, "crate/Cargo.toml", "crate")
	writeFixture(t, root, "crate/LICENSE-THIRD-PARTY", "CC-BY-3.0 notice\n")
	writeFixture(t, root, "crate/licenses/custom.txt", "explicit license_file\n")
	pkg := cargoPackage{Name: "external", Version: "1.0.0", License: "MIT/Apache-2.0", Manifest: manifest, LicenseFile: "licenses/custom.txt"}
	dep, err := rustDependency(root, &pkg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if dep.License != "MIT OR Apache-2.0" || dep.Source != "path:crate" || len(dep.Files) != 4 {
		t.Fatalf("missing inherited/third-party/explicit text: %+v", dep)
	}
	if !slices.ContainsFunc(dep.Files, func(f evidence) bool { return string(f.Data) == "CC-BY-3.0 notice\n" }) {
		t.Fatal("third-party notice lost")
	}
	pkg.License = ""
	if _, err = rustDependency(root, &pkg, nil); err == nil {
		t.Fatal("missing license declaration accepted")
	}
	reviews := []rustReview{{Package: "external", License: "MIT", File: "licenses/custom.txt", SHA256: digest([]byte("explicit license_file\n"))}}
	if _, err = rustDependency(root, &pkg, reviews); err != nil {
		t.Fatalf("reviewed license_file-only crate rejected: %v", err)
	}
	writeFixture(t, root, "crate/licenses/custom.txt", "changed terms")
	if _, err = rustDependency(root, &pkg, reviews); err == nil {
		t.Fatal("changed reviewed license_file accepted")
	}
	pkg.License = "MIT"
	pkg.Source = "registry+example"
	pkg.Manifest = writeFixture(t, root, "unlicensed/Cargo.toml", "crate")
	pkg.LicenseFile = ""
	if _, err = rustDependency(root, &pkg, nil); err == nil {
		t.Fatal("crate without any license texts accepted")
	}
}

func TestSupplementaryLicenseNamesArePreserved(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"LICENSE-MIT", "UNLICENSE", "LICENSE_THIRD_PARTY", "nested/COPYING.LESSER", "nested/NOTICE_UPSTREAM"} {
		writeFixture(t, root, name, "terms for "+name+"\n")
	}
	files, err := licenseFiles(root, true, root)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 5 {
		t.Fatalf("alternative/supplementary license texts missing: %v", files)
	}
}

func TestInventoryPreservesVersionsSourcesAndRejectsConflicts(t *testing.T) {
	all := make(inventory)
	dep := dependency{Ecosystem: "Rust", Name: "crate", Version: "1.0.0", License: "MIT", Source: "registry+example", UsedBy: []string{"Cargo.toml"}, Files: []evidence{{Name: "LICENSE", Data: []byte("terms")}}}
	for _, consumer := range []string{"Cargo.toml", "example/Cargo.toml"} {
		dep.UsedBy = []string{consumer}
		if err := all.add(&dep); err != nil {
			t.Fatal(err)
		}
	}
	dep.Version = "2.0.0"
	if err := all.add(&dep); err != nil {
		t.Fatal(err)
	}
	dep.Source = "registry+another-source"
	if err := all.add(&dep); err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("versions/sources collapsed: %d rows", len(all))
	}
	dep.License = "GPL-3.0-only"
	if err := all.add(&dep); err == nil {
		t.Fatal("conflicting evidence silently accepted")
	}
}

func TestCheckDetectsMissingDependencyVersionAndNoticeWithoutWriting(t *testing.T) {
	root := t.TempDir()
	all := make(inventory)
	dep := dependency{Ecosystem: "Go", Name: "example.org/transitive", Version: "v1.0.0", License: "MIT", Source: "module:example.org/transitive@v1.0.0", UsedBy: []string{"go.mod"}, Files: []evidence{{Name: "LICENSE", Data: []byte("copyright\nterms\n")}}}
	if err := all.add(&dep); err != nil {
		t.Fatal(err)
	}
	outputs, err := render(all)
	if err != nil {
		t.Fatal(err)
	}
	if err = save(root, outputs, false); err != nil {
		t.Fatal(err)
	}
	if err = save(root, outputs, true); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(bytes.NewReader(outputs["LICENSE-3rdparty.csv"])).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	var missing bytes.Buffer
	writer := csv.NewWriter(&missing)
	if err = writer.Write(records[0]); err != nil {
		t.Fatal(err)
	}
	writer.Flush()
	if err = writer.Error(); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, file string
		data       []byte
	}{
		{"missing dependency", "LICENSE-3rdparty.csv", missing.Bytes()},
		{"stale version", "LICENSE-3rdparty.csv", bytes.ReplaceAll(outputs["LICENSE-3rdparty.csv"], []byte("v1.0.0"), []byte("v0.9.0"))},
		{"missing copyright notice", "LICENSE-3rdparty.txt", []byte("removed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if saveErr := save(root, outputs, false); saveErr != nil {
				t.Fatal(saveErr)
			}
			path := writeFixture(t, root, tc.file, string(tc.data))
			if checkErr := save(root, outputs, true); checkErr == nil {
				t.Fatal("incomplete inventory accepted")
			}
			data, readErr := os.ReadFile(path) // #nosec G304 -- Path names a fixture in this test's temporary directory.
			if readErr != nil || !bytes.Equal(data, tc.data) {
				t.Fatalf("check mode changed the file: %v", readErr)
			}
		})
	}
	first, err := render(all)
	if err != nil {
		t.Fatal(err)
	}
	second, err := render(all)
	if err != nil || !bytes.Equal(first["LICENSE-3rdparty.csv"], second["LICENSE-3rdparty.csv"]) || !bytes.Equal(first["LICENSE-3rdparty.txt"], second["LICENSE-3rdparty.txt"]) {
		t.Fatalf("rendering is nondeterministic: %v", err)
	}
}

func TestReviewedToolchainVersionAndNoticeCannotDrift(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "pin.txt", "version=1.0")
	writeFixture(t, root, "notice.txt", "reviewed notices")
	notices := []noticeReview{{Dependency: "tool", Version: "1.0", License: "MIT", File: "notice.txt", SHA256: digest([]byte("reviewed notices")), Pins: []pin{{File: "pin.txt", Text: "version=1.0"}}}}
	if err := collectNotices(root, notices, make(inventory)); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "pin.txt", "version=2.0")
	if err := collectNotices(root, notices, make(inventory)); err == nil {
		t.Fatal("toolchain bump accepted with stale notices")
	}
}
