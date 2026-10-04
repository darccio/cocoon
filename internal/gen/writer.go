package gen

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Artifact is a complete output file. Generated protects authored source files;
// false is reserved for owned binary and lock artifacts, not authored sources.
type Artifact struct {
	Path      string
	Data      []byte
	Generated bool
}

type pending struct {
	path   string
	stage  string
	backup string
}

// WriteGenerated atomically replaces an artifact and refuses authored files.
func WriteGenerated(path string, data []byte) error {
	return WriteSet([]Artifact{{Path: path, Data: data, Generated: true}})
}

// WriteSet preflights every output and stages complete files before publishing.
// A failed publication rolls back earlier replacements. Each rename is atomic;
// concurrent readers must not inspect a project while it is being regenerated.
func WriteSet(artifacts []Artifact) (err error) {
	files := make([]pending, 0, len(artifacts))
	defer func() {
		for _, file := range files {
			for _, path := range []string{file.stage, file.backup} {
				if path != "" {
					if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
						err = errors.Join(err, removeErr)
					}
				}
			}
		}
	}()
	seen := make(map[string]bool)
	for _, artifact := range artifacts {
		path := filepath.Clean(artifact.Path)
		if seen[path] {
			return fmt.Errorf("duplicate output %s", path)
		}
		seen[path] = true
		if parentErr := checkParents(filepath.Dir(path)); parentErr != nil {
			return parentErr
		}
		var previous []byte
		mode := os.FileMode(0o600)
		info, statErr := os.Lstat(path)
		if statErr == nil {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("refuse nonregular artifact %s", path)
			}
			previous, err = os.ReadFile(path) // #nosec G304 -- Artifact destinations are deliberately selected by the validated generator.
			if err != nil {
				return err
			}
			if artifact.Generated && !bytes.Contains(previous[:min(len(previous), 128)], []byte("Code generated")) {
				return fmt.Errorf("refuse to overwrite authored file %s", path)
			}
			mode = info.Mode().Perm()
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return statErr
		}
		if directoryErr := os.MkdirAll(filepath.Dir(path), 0o750); directoryErr != nil {
			return directoryErr
		}
		files = append(files, pending{path: path})
		file := &files[len(files)-1]
		file.stage, err = stageFile(path, artifact.Data, mode)
		if err != nil {
			return err
		}
		if statErr == nil {
			file.backup, err = stageFile(path, previous, mode)
			if err != nil {
				return err
			}
		}
	}
	for index, file := range files {
		if renameErr := os.Rename(file.stage, file.path); renameErr != nil {
			for restored := index - 1; restored >= 0; restored-- {
				prior := files[restored]
				if prior.backup == "" {
					err = errors.Join(err, os.Remove(prior.path))
				} else {
					err = errors.Join(err, os.Rename(prior.backup, prior.path))
				}
			}
			return errors.Join(renameErr, err)
		}
	}
	return nil
}

func stageFile(path string, data []byte, mode os.FileMode) (name string, err error) {
	file, err := os.CreateTemp(filepath.Dir(path), ".cocoon-*")
	if err != nil {
		return "", err
	}
	name = file.Name()
	defer func() {
		if err != nil {
			err = errors.Join(err, os.Remove(name))
		}
	}()
	_, writeErr := file.Write(data)
	err = errors.Join(writeErr, file.Chmod(mode), file.Close())
	return name, err
}

func checkParents(directory string) error {
	for {
		info, err := os.Lstat(directory)
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refuse artifact parent symlink %s", directory)
		}
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return nil
		}
		directory = parent
	}
}
