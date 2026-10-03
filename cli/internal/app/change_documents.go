package app

import (
	"cli/internal/changes"
	"cli/internal/documents"
	"cli/internal/dto"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) beginComment(id int) (tea.Model, tea.Cmd) {
	if !m.changeDetailLoaded {
		m.err = "load change details before editing comments"
		return m, nil
	}
	m.commentID = id
	m.detailEditField = detailEditComment
	body := ""
	for _, d := range m.changeList.Detail.Comments {
		if d.ID == id {
			body = d.Body
		}
	}
	m = m.setPromptValue(body)
	return m.openTextEditor(ChangeDetailsState, body)
}

func (m Model) beginChangeDocumentMutation(op documents.Mutation, id int, body string) (tea.Model, tea.Cmd) {
	project, _ := strconv.Atoi(m.currentProject.ID)
	owner, _ := changeNumericID(m.changeList.Detail)
	var cmd tea.Cmd
	m.changeDocuments, cmd = m.changeDocuments.Begin(m.ctx, m.client, project, owner, op, id, body)
	m.status = "saving documents"
	if op == documents.Read {
		m.status = "refreshing documents"
	}
	if m.changeDocuments.Err != nil {
		m.err = m.changeDocuments.Err.Error()
	}
	return m, cmd
}

func (m Model) applyChangeDocumentResult(r documents.ChangeResult) (tea.Model, tea.Cmd) {
	if m.state != ChangeDetailsState || m.currentProject.ID != strconv.Itoa(r.ProjectID) || m.changeList.Detail.ID != strconv.Itoa(r.OwnerID) {
		return m, nil
	}
	next, ok := m.changeDocuments.Apply(r)
	if !ok {
		return m, nil
	}
	m.changeDocuments = next
	m.status, m.err = "documents refreshed", ""
	if next.Committed != "" {
		m.status = next.Committed
	}
	if r.Err == nil && (r.Operation == documents.NewComment || r.Operation == documents.EditComment) {
		m.detailEditField = ""
		m.commentID = 0
		m = m.setPromptValue("")
	}
	if next.Err != nil {
		m.err = next.Err.Error()
		if next.Committed != "" {
			m.status += "; refresh failed; /retry reads only"
		}
		return m, nil
	}
	m.changeList.Detail.Documents, m.changeList.Detail.Comments = r.Active, r.Comments
	m.changeList.Detail.Brief, m.changeList.Detail.Spec, m.changeList.Detail.PR = "", "", ""
	for _, d := range r.Active {
		switch d.DocType {
		case "brief":
			m.changeList.Detail.Brief = d.Body
		case "spec":
			m.changeList.Detail.Spec = d.Body
		case "pr":
			m.changeList.Detail.PR = d.Body
		}
	}
	m.detailEditField = ""
	m = m.setPromptValue("")
	m.changeList = m.changeList.ClampDetailSelection(m.changeTableRows(), terminalWidth(m.width))
	return m, nil
}

func (m Model) openDocumentConfirmation(id int) (tea.Model, tea.Cmd) {
	if id <= 0 {
		return m, nil
	}
	m.deleteDocumentID = id
	m.dropdown = dropdownModel{kind: dropdownConfirm, previous: ChangeDetailsState, onSelect: ChangeDetailsState, label: "Are you sure?", options: []dto.Option{{ID: "/yes", Label: "yes"}, {ID: "/no", Label: "no"}}}
	return m, nil
}

func (m Model) openHistory(kind string) (tea.Model, tea.Cmd) {
	project, _ := strconv.Atoi(m.currentProject.ID)
	owner, table := 0, "change"
	if m.state == DocumentState {
		project, owner, table = m.document.ProjectID, m.document.OwnerID, m.document.OwnerTable
	} else {
		owner, _ = changeNumericID(m.changeList.Detail)
	}
	var cmd tea.Cmd
	m.history, cmd = m.history.Open(m.ctx, m.client, m.historyPrinter, project, owner, table, kind)
	m.historyOpen = cmd != nil
	if m.historyOpen && m.state == ChangeDetailsState {
		m.changeList = m.changeList.Invalidate()
	}
	if m.history.Err != nil {
		m.err = m.history.Err.Error()
	}
	return m, cmd
}

func (m Model) applyHistoryResult(r documents.HistoryResult) (tea.Model, tea.Cmd) {
	if !m.historyOpen || (m.state == DocumentState && !m.documentScopeCurrent()) || (m.state == ChangeDetailsState && (m.currentProject.ID != strconv.Itoa(r.ProjectID) || m.changeList.Detail.ID != strconv.Itoa(r.OwnerID))) {
		return m, nil
	}
	next, ok := m.history.Apply(r)
	if !ok {
		return m, nil
	}
	m.history = next
	if m.historyReturning {
		return m.returnHistory()
	}
	m.status, m.err = next.Status, ""
	if next.Err != nil {
		m.err = next.Err.Error()
	}
	return m, nil
}

func (m Model) historyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "esc" || key == "ctrl+c" {
		var pending bool
		m.history, pending = m.history.CancelMutation()
		if pending {
			m.historyReturning = true
			m.status = "returning from history; waiting for document update"
			return m, nil
		}
		return m.returnHistory()
	}
	if m.history.Busy {
		return m, nil
	}
	var cmd tea.Cmd
	if len(msg.Runes) > 0 && strings.HasPrefix(string(msg.Runes), "/") {
		m.dropdown = dropdownModel{kind: dropdownCommand, state: CommandDropDownState, previous: m.state, onSelect: m.state, filter: strings.TrimPrefix(string(msg.Runes), "/"), options: []dto.Option{{ID: "/retry", Label: "/retry"}, {ID: "/return", Label: "/return"}}}
		return m, nil
	}
	switch key {
	case "left":
		m.history, cmd = m.history.Move(m.ctx, m.client, m.historyPrinter, -1)
	case "right":
		m.history, cmd = m.history.Move(m.ctx, m.client, m.historyPrinter, 1)
	case " ", "space":
		m.history, cmd = m.history.Activate(m.ctx, m.client, m.historyPrinter)
	case "up":
		m.history = m.history.Scroll(-1, m.historyViewportHeight())
	case "down":
		m.history = m.history.Scroll(1, m.historyViewportHeight())
	case "pgup":
		m.history = m.history.Scroll(-max(1, m.historyViewportHeight()), m.historyViewportHeight())
	case "pgdown":
		m.history = m.history.Scroll(max(1, m.historyViewportHeight()), m.historyViewportHeight())
	case "ctrl+r":
		m.history, cmd = m.history.Refresh(m.ctx, m.client, m.historyPrinter)
	case "/":
		m.dropdown = dropdownModel{kind: dropdownCommand, previous: m.state, onSelect: m.state, options: []dto.Option{{ID: "/retry", Label: "/retry"}, {ID: "/return", Label: "/return"}}}
	}
	return m, cmd
}

func (m Model) returnHistory() (tea.Model, tea.Cmd) {
	m.history = m.history.Invalidate()
	m.historyOpen, m.historyReturning = false, false
	m = m.setPromptValue("")
	m.status, m.err = "returned from history", ""
	if m.state == ChangeDetailsState && m.history.Committed != "" {
		m.changeDocuments.ProjectID, m.changeDocuments.OwnerID = m.history.ProjectID, m.history.OwnerID
		m.changeDocuments.Committed = "returned from history; " + m.history.Committed
		return m.beginChangeDocumentMutation(documents.Read, 0, "")
	}
	if m.state == DocumentState {
		m.document = m.document.Back()
		m.document = m.document.KeepSelectedVisible(m.documentViewportHeight())
		if m.history.Committed != "" {
			m.document.Committed = "returned from history; " + m.history.Committed
			m.document.CommittedID = 0
			return m.beginDocumentRefresh()
		}
	}
	return m, tea.ClearScreen
}

func (m Model) historyViewportHeight() int {
	lines, _ := m.viewLines()
	return m.epicViewportHeight(lines)
}

func (m Model) selectedDocumentHistory() (tea.Model, tea.Cmd) {
	row, ok := changes.DetailRowAtSelection(m.changeList.Detail, m.changeList.DetailSelected)
	if !ok || row.DocumentType == "" {
		return m, nil
	}
	return m.openHistory(row.DocumentType)
}

func (m Model) openInactiveChanges() (tea.Model, tea.Cmd) {
	if m.changeList.Inactive {
		return m, nil
	}
	m.inactiveOrigin = m.changeList
	m.changeList = m.changeList.Invalidate()
	m.changeList.Inactive = true
	return m.beginChange(changes.List, 0, changes.Input{})
}

func (m Model) leaveInactiveChanges() (tea.Model, tea.Cmd) {
	generation := m.changeList.Invalidate().Generation
	m.changeList = m.inactiveOrigin
	m.changeList.Generation = generation
	m.changeList.Inactive = false
	m.rememberSelectedChange()
	return m.beginChange(changes.List, 0, changes.Input{})
}
