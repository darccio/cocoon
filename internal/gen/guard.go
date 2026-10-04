package gen

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

// Guard excludes concurrent generation and builds for one project. The returned
// release is idempotent; interrupted processes leave an explicit stale lock.
func Guard(directory string) (func() error, error) {
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	if directoryErr := root.MkdirAll(".cocoon-build", 0o750); directoryErr != nil {
		return nil, errors.Join(directoryErr, root.Close())
	}
	info, err := root.Lstat(".cocoon-build")
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.Join(fmt.Errorf("symlinked project build directory"), root.Close())
	}
	file, err := root.OpenFile(".cocoon-build/lock", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("build or generation already running or stale .cocoon-build/lock: %w", err), root.Close())
	}
	var once sync.Once
	var releaseErr error
	return func() error {
		once.Do(func() { releaseErr = errors.Join(file.Close(), root.Remove(".cocoon-build/lock"), root.Close()) })
		return releaseErr
	}, nil
}
