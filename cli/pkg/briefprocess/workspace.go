package briefprocess

import (
	"cli/internal/dto"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Workspace owns a single exclusive scratch directory for an editor/agent operation.
type Workspace struct {
	TempDir  string
	dir      string
	identity os.FileInfo
}

// Prepare refuses existing paths, including unowned directories and symlinks.
func (w *Workspace) Prepare(ref, body string) (string, error) {
	if !uuid.MatchString(ref) {
		return "", fmt.Errorf("change reference must be a UUID")
	}
	base := w.TempDir
	if base == "" {
		base = os.TempDir()
	}
	parent := filepath.Join(base, "mch")
	if err := os.Mkdir(parent, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return "", err
	}
	info, err := os.Lstat(parent)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("refuse scratch ancestor %s", parent)
	}
	dir := filepath.Join(parent, ref)
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", fmt.Errorf("refuse scratch directory reuse %s: %w", dir, err)
	}
	w.dir = dir
	w.identity, err = os.Lstat(dir)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "brief.md")
	if err := writeExclusive(path, []byte(body)); err != nil {
		return path, err
	}
	return path, nil
}

func (w *Workspace) owned(path string) error {
	if w.dir == "" || filepath.Dir(path) != w.dir {
		return fmt.Errorf("refuse unowned operation file %s", path)
	}
	parent, err := os.Lstat(filepath.Dir(w.dir))
	if err != nil {
		return err
	}
	if !parent.IsDir() || parent.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refuse unowned scratch ancestor")
	}
	info, err := os.Lstat(w.dir)
	if err != nil {
		return err
	}
	if !info.IsDir() || !os.SameFile(w.identity, info) {
		return fmt.Errorf("scratch ownership changed")
	}
	return nil
}

// Stamp rejects non-regular draft files and records missing output explicitly.
func (w *Workspace) Stamp(path string) (dto.FileStamp, error) {
	if err := w.owned(path); err != nil {
		return dto.FileStamp{}, err
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return dto.FileStamp{}, nil
	}
	if err != nil {
		return dto.FileStamp{}, err
	}
	if !info.Mode().IsRegular() {
		return dto.FileStamp{}, fmt.Errorf("draft is not a regular file: %s", path)
	}
	return dto.FileStamp{Exists: true, Time: info.ModTime()}, nil
}

// Read returns exact draft bytes from the owned directory.
func (w *Workspace) Read(path string) (string, error) {
	stamp, err := w.Stamp(path)
	if err != nil {
		return "", err
	}
	if !stamp.Exists {
		return "", fmt.Errorf("draft missing: %s", path)
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

// Prompt substitutes only the brief placeholder, in memory.
func (w *Workspace) Prompt(root, name, path string) (string, error) {
	if name != "brief-rewrite" && name != "spec-write" {
		return "", fmt.Errorf("unsupported prompt %s", name)
	}
	data, err := os.ReadFile(filepath.Join(root, ".mch", "default", "prompts", name+".md"))
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(string(data), "[brief-file-path.md]", path), nil
}

// Cleanup validates ownership and removes only the operation's known files.
func (w *Workspace) Cleanup(path string) error {
	if err := w.owned(path); err != nil {
		return err
	}
	paths := []string{path, filepath.Join(w.dir, "spec.md"), filepath.Join(w.dir, "final.txt")}
	// Validate everything before deleting anything.
	for _, path := range paths {
		if _, err := w.Stamp(path); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != "brief.md" && entry.Name() != "spec.md" && entry.Name() != "final.txt" {
			return fmt.Errorf("refuse cleanup of unexpected operation file %s", entry.Name())
		}
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return os.Remove(w.dir)
}

func writeExclusive(path string, body []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
