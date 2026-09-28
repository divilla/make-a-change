package app

import (
	"errors"
	"os"
	"path/filepath"
)

func replaceFileAtomically(path string, mode os.FileMode, write func(*os.File) error) error {
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	closed := false
	defer func() {
		if !closed {
			_ = temporary.Close()
		}
		_ = os.Remove(temporaryPath)
	}()

	if err := temporary.Chmod(mode); err != nil {
		return err
	}
	if err := write(temporary); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	closeErr := temporary.Close()
	closed = true
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(temporaryPath, path)
}
