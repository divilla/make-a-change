package app

import (
	"cli/internal/dto"
	"cli/pkg/briefprocess"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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
	draftProject, _ := strconv.Atoi(projectID)
	draftOwner, _ := strconv.Atoi(ownerID)
	draftTable := "change"
	switch field {
	case detailEditBrief:
		documentType = "brief"
	case detailEditSpec:
		documentType = "spec"
	}
	if source == ChangeUpdateState && field == "" {
		documentType = "spec"
	}
	if source == DocumentState {
		draftProject, draftOwner, draftTable = m.document.ProjectID, documentOwner, documentTable
	}
	retain := source == ChangeUpdateState || (source == ChangeDetailsState && (field == detailEditBrief || field == detailEditSpec || (field == detailEditDocument && (documentType == "brief" || documentType == "spec")))) || (source == DocumentState && (documentType == "brief" || documentType == "spec"))
	retry := m.editorDraft != nil
	cmd := tea.ExecProcess(editorProcess(m.ctx, m.appConfig.Editor, path), func(err error) tea.Msg {
		var content []byte
		info, readErr := os.Lstat(path)
		if readErr == nil {
			if !info.Mode().IsRegular() {
				readErr = fmt.Errorf("editor draft is not a regular file")
			} else {
				content, readErr = os.ReadFile(path)
			}
		}
		var scratch *editorScratch
		if retain {
			scratch = &editorScratch{path: path, identity: info, projectID: draftProject, document: dto.Document{RefID: draftOwner, RefTable: draftTable, DocType: documentType, Body: string(content)}}
		}
		if removeAfter && (!retain || (err == nil && readErr == nil && string(content) == original && !retry)) {
			scratch = nil
			_ = os.Remove(path)
		}
		if err != nil {
			return editorFinishedMsg{scratch: scratch, generation: generation, projectID: projectID, ownerID: ownerID, field: field, documentRevision: documentRevision, documentOwner: documentOwner, documentTable: documentTable, documentType: documentType, commentID: commentID, source: source, err: err}
		}
		if readErr != nil {
			return editorFinishedMsg{scratch: scratch, generation: generation, projectID: projectID, ownerID: ownerID, field: field, documentRevision: documentRevision, documentOwner: documentOwner, documentTable: documentTable, documentType: documentType, commentID: commentID, source: source, err: readErr}
		}
		return editorFinishedMsg{scratch: scratch, generation: generation, projectID: projectID, ownerID: ownerID, field: field, documentRevision: documentRevision, documentOwner: documentOwner, documentTable: documentTable, documentType: documentType, commentID: commentID, source: source, original: original, content: string(content)}
	})
	return m, cmd
}

func editorCommand(path string) *exec.Cmd { return configuredEditorCommand("", path) }

func configuredEditorCommand(editor, path string) *exec.Cmd {
	editor = strings.TrimSpace(editor)
	if editor == "" {
		editor = strings.TrimSpace(os.Getenv("EDITOR"))
	}
	if editor == "" {
		return exec.Command("nano", path)
	}
	cmd := exec.Command("sh", "-c", "$EDITOR \"$1\"", "mch-editor", path)
	cmd.Env = append(os.Environ(), "EDITOR="+editor)
	return cmd
}

func (m Model) cleanupEditorDraft(project int, saved dto.Document) Model {
	if m.editorScratch == nil {
		return m
	}
	draft := m.editorScratch
	if draft.projectID != project || draft.document.RefID != saved.RefID || draft.document.RefTable != saved.RefTable || draft.document.DocType != saved.DocType || strings.TrimSpace(draft.document.Body) != saved.Body || saved.AgentEdit {
		return m
	}
	info, err := os.Lstat(draft.path)
	if err == nil {
		if info.Mode().IsRegular() && draft.identity != nil && os.SameFile(info, draft.identity) {
			err = os.Remove(draft.path)
		} else {
			err = fmt.Errorf("editor draft ownership changed; refuse cleanup")
		}
	}
	if err != nil {
		m.err = strings.TrimPrefix(m.err+"; editor draft cleanup failed: "+err.Error(), "; ")
	}
	m.editorScratch = nil
	return m
}

func editorProcess(ctx context.Context, editor, path string) *exec.Cmd {
	configured := configuredEditorCommand(editor, path)
	cmd := briefprocess.InteractiveCommand(ctx, configured.Path, configured.Args[1:]...)
	cmd.Env = configured.Env
	return cmd
}
