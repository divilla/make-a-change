// Package documentprocess captures syntax-colored document output in owned files.
package documentprocess

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Bat prints the full Markdown body without handing terminal ownership away.
type Bat struct {
	Executable string
	TempRoot   string
}

// Print preserves bat's ANSI output and removes its operation-owned directory.
func (b Bat) Print(ctx context.Context, body string) (output string, err error) {
	root, err := os.MkdirTemp(b.TempRoot, "mch-history-")
	if err != nil {
		return "", fmt.Errorf("history file: %w", err)
	}
	defer func() { err = errors.Join(err, os.RemoveAll(root)) }()
	path := filepath.Join(root, "document.md")
	if err = os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", fmt.Errorf("history file: %w", err)
	}
	executable := b.Executable
	if executable == "" {
		executable = "bat"
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	data, err := exec.CommandContext(ctx, executable, "-pp", "--color=always", path).Output()
	if err != nil {
		return "", fmt.Errorf("history bat: %w", errors.Join(err, ctx.Err()))
	}
	return string(data), nil
}
