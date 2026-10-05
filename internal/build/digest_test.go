package build

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceDigestTracksContainedRelativeLinks(t *testing.T) {
	t.Parallel()
	directory := t.TempDir()
	for _, name := range []string{"first", "second"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("identical contents"), 0o600); err != nil { // #nosec G304 -- Explicit fixtures below t.TempDir.
			t.Fatal(err)
		}
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink("first", link); err != nil {
		t.Skip("filesystem does not support symlink fixtures", err)
	}
	first, err := sourceDigest(directory)
	if err != nil {
		t.Fatal("contained relative link rejected", err)
	}
	if removeErr := os.Remove(link); removeErr != nil {
		t.Fatal(removeErr)
	}
	if linkErr := os.Symlink("second", link); linkErr != nil {
		t.Fatal(linkErr)
	}
	second, err := sourceDigest(directory)
	if err != nil || first == second {
		t.Fatal("link target identity was not hashed", err)
	}
	if linkErr := os.Symlink("missing", filepath.Join(directory, "broken")); linkErr != nil {
		t.Fatal(linkErr)
	}
	if _, err := sourceDigest(directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("broken source link error lost", err)
	}
}

func TestSourceDigestPermissionFailures(t *testing.T) {
	t.Parallel()
	for _, scenario := range []string{"file", "directory"} {
		t.Run(scenario, func(t *testing.T) {
			t.Parallel()
			directory := t.TempDir()
			path := filepath.Join(directory, "source.rs")
			if err := os.WriteFile(path, []byte("authored"), 0o600); err != nil {
				t.Fatal(err)
			}
			denied := path
			if scenario == "directory" {
				denied = directory
			}
			if err := os.Chmod(denied, 0); err != nil {
				t.Skip("filesystem does not support permission fixtures", err)
			}
			t.Cleanup(func() {
				if err := os.Chmod(denied, 0o700); err != nil { // #nosec G302 -- Restore access to an owned test fixture for cleanup.
					t.Error(err)
				}
			})
			if _, err := os.ReadFile(path); !errors.Is(err, os.ErrPermission) { // #nosec G304 -- Probe permissions on this test's own fixture.
				t.Skip("filesystem or process privileges do not enforce fixture permissions")
			}
			if _, err := sourceDigest(directory); !errors.Is(err, os.ErrPermission) {
				t.Fatal("unreadable source input error lost", err)
			}
		})
	}
}

func TestExecutableDigestRejectsDirectory(t *testing.T) {
	t.Parallel()
	if hash, err := executableDigest(t.TempDir()); err == nil || hash != "" {
		t.Fatal("directory accepted as an executable identity", hash, err)
	}
}
