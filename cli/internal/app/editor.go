package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) openPromptEditor(source State) (tea.Model, tea.Cmd) {
	return m.openTextEditor(source, m.promptValue())
}

func (m Model) openTextEditor(source State, original string) (tea.Model, tea.Cmd) {
	file, err := os.CreateTemp("", "mch-project-*.md")
	if err != nil {
		m.err = fmt.Errorf("failed to create editor file: %w", err).Error()
		return m, nil
	}
	path := file.Name()
	if _, err := file.WriteString(original); err != nil {
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
	return m.openEditorPath(source, path, true, original)
}

func (m Model) openEditorPath(source State, path string, removeAfter bool, original string) (tea.Model, tea.Cmd) {
	m.status = "editor"
	m.editorGeneration++
	generation, projectID, ownerID, field := m.editorGeneration, m.currentProject.ID, m.changeList.Detail.ID, m.detailEditField
	documentRevision, documentOwner, documentTable, documentType, commentID := m.document.Revision, m.document.OwnerID, m.document.OwnerTable, m.changeList.Draft.DocumentType, m.commentID
	if source == DocumentState {
		documentType = m.document.DraftType
	}
	cmd := tea.ExecProcess(editorCommand(path), func(err error) tea.Msg {
		content, readErr := os.ReadFile(path)
		if removeAfter {
			_ = os.Remove(path)
		}
		if err != nil {
			return editorFinishedMsg{generation: generation, projectID: projectID, ownerID: ownerID, field: field, documentRevision: documentRevision, documentOwner: documentOwner, documentTable: documentTable, documentType: documentType, commentID: commentID, source: source, err: err}
		}
		if readErr != nil {
			return editorFinishedMsg{generation: generation, projectID: projectID, ownerID: ownerID, field: field, documentRevision: documentRevision, documentOwner: documentOwner, documentTable: documentTable, documentType: documentType, commentID: commentID, source: source, err: readErr}
		}
		return editorFinishedMsg{generation: generation, projectID: projectID, ownerID: ownerID, field: field, documentRevision: documentRevision, documentOwner: documentOwner, documentTable: documentTable, documentType: documentType, commentID: commentID, source: source, original: original, content: string(content)}
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
