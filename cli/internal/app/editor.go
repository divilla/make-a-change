package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) openPromptEditor(source State) (tea.Model, tea.Cmd) {
	file, err := os.CreateTemp("", "mch-project-*.md")
	if err != nil {
		m.err = fmt.Errorf("failed to create editor file: %w", err).Error()
		return m, nil
	}
	path := file.Name()
	if _, err := file.WriteString(m.input.Value()); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		m.err = fmt.Errorf("failed to write editor file: %w", err).Error()
		return m, nil
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		m.err = fmt.Errorf("failed to close editor file: %w", err).Error()
		return m, nil
	}
	return m.openEditorPath(source, path, true)
}

func (m Model) openPersistentEditor(source State, path string) (tea.Model, tea.Cmd) {
	return m.openEditorPath(source, path, false)
}

func (m Model) openEditorPath(source State, path string, removeAfter bool) (tea.Model, tea.Cmd) {
	m.status = "editor"
	cmd := tea.ExecProcess(editorCommand(path), func(err error) tea.Msg {
		content, readErr := os.ReadFile(path)
		if removeAfter {
			_ = os.Remove(path)
		}
		if err != nil {
			return editorFinishedMsg{source: source, err: err}
		}
		if readErr != nil {
			return editorFinishedMsg{source: source, err: readErr}
		}
		return editorFinishedMsg{source: source, content: string(content)}
	})
	return m, cmd
}

func editorCommand(path string) *exec.Cmd {
	editor := strings.TrimSpace(os.Getenv("EDITOR"))
	if editor == "" {
		return exec.Command("nano", path)
	}
	cmd := exec.Command("sh", "-c", "$EDITOR \"$1\"", "mch-editor", path)
	cmd.Env = append(os.Environ(), "EDITOR="+editor)
	return cmd
}
