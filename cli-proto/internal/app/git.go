package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
)

func resolveRepoRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New("resolve git repository root: " + msg)
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", errors.New("resolve git repository root: empty path")
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("resolve git repository root: path is not a directory")
	}
	return root, nil
}
