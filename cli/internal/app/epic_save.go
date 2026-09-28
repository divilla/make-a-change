package app

import (
	"cli/internal/epics"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) beginEpic(op epics.Operation, id int, name string) (tea.Model, tea.Cmd) {
	projectID, _ := strconv.Atoi(m.currentProject.ID)
	var cmd tea.Cmd
	m.epicList, cmd = m.epicList.Begin(m.ctx, m.client, op, projectID, id, name)
	m.status = m.epicList.Status
	m.err = ""
	if m.epicList.Err != nil {
		m.err = m.epicList.Err.Error()
	}
	if cmd == nil && m.status == "unchanged" {
		m.state = EpicDetailsState
		m = m.setPromptValue("")
	}
	return m, cmd
}

func (m Model) epicForm(edit bool) (tea.Model, tea.Cmd) {
	projectID, _ := strconv.Atoi(m.currentProject.ID)
	if m.epicList.ProjectID != projectID {
		m.epicList = m.epicList.Scope(projectID)
	}
	var ok bool
	m.epicList, ok = m.epicList.Form(edit)
	if !ok {
		m.err = m.epicList.Err.Error()
		return m, nil
	}
	m.state = EpicCreateState
	if edit {
		m.state = EpicUpdateState
	}
	m = m.setPromptValue(m.epicList.Draft)
	// Loaded names are literal data, including names that match commands.
	if edit {
		raw := m.epicList.Draft
		m.editorDraft = &raw
	}
	m.input.Placeholder = "Write a Name"
	m.status = "epic name"
	return m, nil
}

func (m Model) applyEpicResult(r epics.Result) (tea.Model, tea.Cmd) {
	if strconv.Itoa(r.ProjectID) != m.currentProject.ID {
		return m, nil
	}
	next, accepted := m.epicList.Apply(r)
	if !accepted {
		return m, nil
	}
	m.epicList = next
	m.status = next.Status
	m.err = ""
	if next.Err != nil {
		m.err = next.Err.Error()
	}
	if r.Err == nil {
		switch r.Operation {
		case epics.Create, epics.Edit:
			m.state = EpicDetailsState
			m = m.setPromptValue("")
		case epics.Delete:
			m.state = EpicsListState
			m = m.setPromptValue("")
		}
	}
	return m, nil
}
