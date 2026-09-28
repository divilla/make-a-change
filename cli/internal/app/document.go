package app

import (
	"cli/internal/documents"
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) openDocuments(source State) (tea.Model, tea.Cmd) {
	var projectID, ownerID int
	var table string
	switch source {
	case ProjectDetailsState:
		if m.projectList.Loading || !m.projectList.DetailLoaded || m.projectList.Detail.ID <= 0 {
			m.err = "load project details before opening documents"
			return m, nil
		}
		projectID, ownerID, table = m.projectList.Detail.ID, m.projectList.Detail.ID, "project"
	case EpicDetailsState:
		if m.epicList.Loading || !m.epicList.DetailLoaded || m.epicList.Detail.ID <= 0 {
			m.err = "load epic details before opening documents"
			return m, nil
		}
		projectID, ownerID, table = m.epicList.Detail.ProjectID, m.epicList.Detail.ID, "epic"
		if m.currentProject.ID != strconv.Itoa(projectID) {
			m.err = "epic is outside the selected project"
			return m, nil
		}
	case ChangeDetailsState:
		if !m.changeDetailLoaded {
			m.err = "load change details before opening documents"
			return m, nil
		}
		changeID, err := changeNumericID(m.changeList.Detail)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		projectID, err = currentProjectNumericID(m.currentProject.ID)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
		if m.changeList.Detail.ProjectID != strconv.Itoa(projectID) {
			m.err = "change is outside the selected project"
			return m, nil
		}
		ownerID, table = changeID, "change"
	default:
		m.err = "documents are available from owner details"
		return m, nil
	}
	var cmd tea.Cmd
	m.document, cmd = m.document.BeginOpen(m.ctx, m.client, projectID, ownerID, table)
	if cmd == nil {
		m.err = m.document.Err.Error()
		return m, nil
	}
	m.documentReturn, m.state, m.documentForm = source, DocumentState, false
	m.status, m.err = m.document.Status, ""
	m = m.setPromptValue("")
	return m, cmd
}

func (m Model) documentScopeCurrent() bool {
	if m.state != DocumentState {
		return false
	}
	switch m.documentReturn {
	case ProjectDetailsState:
		return m.projectList.Detail.ID == m.document.OwnerID
	case EpicDetailsState:
		return m.currentProject.ID == strconv.Itoa(m.document.ProjectID) && m.epicList.Detail.ID == m.document.OwnerID && m.epicList.Detail.ProjectID == m.document.ProjectID
	case ChangeDetailsState:
		return m.currentProject.ID == strconv.Itoa(m.document.ProjectID) && m.changeList.Detail.ID == strconv.Itoa(m.document.OwnerID) && m.changeList.Detail.ProjectID == strconv.Itoa(m.document.ProjectID)
	}
	return false
}

func (m Model) applyDocumentResult(r documents.Result) (tea.Model, tea.Cmd) {
	if !m.documentScopeCurrent() {
		return m, nil
	}
	next, ok := m.document.Apply(r)
	if !ok {
		return m, nil
	}
	m.document = next
	m.status, m.err = next.Status, ""
	if next.Err != nil {
		m.err = next.Err.Error()
	}
	if (r.Operation == documents.Open || r.Operation == documents.Refresh) && next.Loaded {
		m.document = m.document.KeepSelectedVisible(m.documentViewportHeight())
	}
	if r.Operation == documents.Insert && r.Err == nil && r.ID > 0 {
		m.documentForm = false
		m = m.setPromptValue("")
		var cmd tea.Cmd
		m.document, cmd = m.document.BeginRefresh(m.ctx, m.client)
		m.status = m.document.Status
		return m, cmd
	}
	return m, nil
}

func (m Model) beginDocumentInsert(body string) (tea.Model, tea.Cmd) {
	if !m.documentScopeCurrent() {
		m.err = "document owner selection changed"
		return m, nil
	}
	var cmd tea.Cmd
	m.document, cmd = m.document.BeginInsert(m.ctx, m.client, body, false)
	m.status, m.err = m.document.Status, ""
	if m.document.Err != nil {
		m.err = m.document.Err.Error()
	}
	return m, cmd
}

func (m Model) beginDocumentRefresh() (tea.Model, tea.Cmd) {
	if !m.documentScopeCurrent() {
		m.err = "document owner selection changed"
		return m, nil
	}
	var cmd tea.Cmd
	m.document, cmd = m.document.BeginRefresh(m.ctx, m.client)
	m.status, m.err = m.document.Status, ""
	if m.document.Err != nil {
		m.err = m.document.Err.Error()
	}
	return m, cmd
}

func (m Model) documentCommand(command string) (tea.Model, tea.Cmd) {
	if command != "/return" && command != "/cancel" && !m.documentScopeCurrent() {
		m.err = "document owner selection changed"
		return m, nil
	}
	switch command {
	case "/new-document":
		if m.document.Busy || !m.document.Loaded {
			m.err = "load documents before appending"
			return m, nil
		}
		if len(m.document.Types) == 0 {
			m.err = "no document types configured for this owner"
			return m, nil
		}
		m.documentForm = true
		m.document = m.document.Back()
		m.status = fmt.Sprintf("new %s document for %s #%d (human)", m.document.DraftType, m.document.OwnerTable, m.document.OwnerID)
		m.input.Placeholder = "Write document body or Ctrl+E editor"
		m = m.setPromptValue(m.document.DraftBody)
		m.document = m.document.KeepSelectedVisible(m.documentViewportHeight())
		return m, nil
	case "/type":
		m.document = m.document.NextType()
		if m.document.Err != nil {
			m.err = m.document.Err.Error()
		} else {
			m.status = "selected document type: " + m.document.DraftType
		}
		return m, nil
	case "/retry":
		return m.beginDocumentRefresh()
	case "/cancel":
		if m.documentForm {
			m.documentForm = false
			m.document.DraftBody = ""
			m = m.setPromptValue("")
			m.status = "document draft canceled"
			return m, nil
		}
		if m.document.ShowingDetail {
			m.document = m.document.Back()
			m.status = "document history"
			m.document = m.document.KeepSelectedVisible(m.documentViewportHeight())
			return m, nil
		}
		return m.leaveDocuments()
	case "/return":
		if m.document.ShowingDetail {
			m.document = m.document.Back()
			m.status = "document history"
			m.document = m.document.KeepSelectedVisible(m.documentViewportHeight())
			return m, nil
		}
		return m.leaveDocuments()
	}
	m.err = "unknown document command"
	return m, nil
}

func (m Model) leaveDocuments() (tea.Model, tea.Cmd) {
	target := m.documentReturn
	m.document = m.document.Invalidate()
	m.documentForm = false
	m = m.setPromptValue("")
	m.input.Placeholder = defaultInputPlaceholder
	return m.arrive(target, "return")
}

func (m Model) documentKey(key string, msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.state != DocumentState || m.documentForm || m.input.Value() != "" {
		return m, nil, false
	}
	if key != "up" && key != "down" && key != "pgup" && key != "pgdown" && key != "enter" && msg.Type != tea.KeyCtrlJ {
		return m, nil, false
	}
	if !m.documentScopeCurrent() {
		m.err = "document owner selection changed"
		return m, nil, true
	}
	height := m.documentViewportHeight()
	if key == "up" || key == "down" || key == "pgup" || key == "pgdown" {
		delta := 1
		if key == "pgup" || key == "pgdown" {
			delta = max(1, height)
		}
		if key == "up" || key == "pgup" {
			delta = -delta
		}
		if m.document.ShowingDetail {
			m.document = m.document.Scroll(delta, terminalWidth(m.width), height)
		} else {
			m.document = m.document.Move(delta)
			m.document = m.document.KeepSelectedVisible(height)
		}
		return m, nil, true
	}
	if key == "enter" || msg.Type == tea.KeyCtrlJ {
		if m.document.ShowingDetail {
			return m, nil, true
		}
		var cmd tea.Cmd
		m.document, cmd = m.document.BeginDetails(m.ctx, m.client)
		m.status, m.err = m.document.Status, ""
		if m.document.Err != nil {
			m.err = m.document.Err.Error()
		}
		return m, cmd, true
	}
	return m, nil, false
}

func (m Model) documentViewportHeight() int {
	lines, _ := m.viewLines()
	return m.epicViewportHeight(lines)
}
