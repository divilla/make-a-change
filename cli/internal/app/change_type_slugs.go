package app

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "sync"

    "cli/internal/dto"
)

const changeTypeSlugsPrompt = "change-types.md"

var changeTypesCache = struct {
    sync.RWMutex
    options []dto.Option
}{}

// ChangeTypes returns a copy of the Change type options loaded at application startup.
func ChangeTypes() []dto.Option {
    changeTypesCache.RLock()
    defer changeTypesCache.RUnlock()
    return append([]dto.Option(nil), changeTypesCache.options...)
}

func cacheChangeTypes(options []dto.Option) {
    changeTypesCache.Lock()
    defer changeTypesCache.Unlock()
    changeTypesCache.options = append([]dto.Option(nil), options...)
}

func rebuildChangeTypeSlugsFile(flowDir string, options []dto.Option) error {
    var content strings.Builder
    content.WriteString("# Change Types\n")
    for _, option := range options {
        content.WriteString("\n- ")
        content.WriteString(option.ID)
    }
    content.WriteByte('\n')

    path := filepath.Join(flowDir, "prompts", changeTypeSlugsPrompt)
    if err := replaceFileAtomically(path, 0o644, func(file *os.File) error {
        _, err := file.WriteString(content.String())
        return err
    }); err != nil {
        return fmt.Errorf("rebuild Change type slugs file %s: %w", path, err)
    }
    return nil
}

func replaceFileAtomically(path string, mode os.FileMode, write func(*os.File) error) error {
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
    closeErr := temporary.Close()
    closed = true
    if closeErr != nil {
        return closeErr
    }
    return os.Rename(temporaryPath, path)
}
