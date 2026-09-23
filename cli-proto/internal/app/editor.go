package app

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

type Editor interface {
	Edit(body string) (string, error)
}

type ExternalEditor struct{}

func (ExternalEditor) Edit(body string) (string, error) {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		return "", errors.New("$EDITOR is unset")
	}
	file, err := os.CreateTemp("", "mch-requirement-*.md")
	if err != nil {
		return "", err
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.WriteString(body); err != nil {
		file.Close()
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return "", errors.New("$EDITOR is unset")
	}
	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
