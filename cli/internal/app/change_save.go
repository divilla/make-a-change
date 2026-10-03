package app

import (
	"cli/internal/changes"
	"cli/internal/documents"
	"cli/internal/dto"
	"cli/internal/testcases"
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) beginTestCase(op testcases.Operation, rawID, scenario string, done bool) (tea.Model, tea.Cmd) {
	projectID, err := currentProjectNumericID(m.currentProject.ID)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	changeID, err := changeNumericID(m.changeList.Detail)
	if err != nil {
		m.err = err.Error()
		return m, nil
	}
	var cmd tea.Cmd
	switch op {
	case testcases.Create, testcases.Refresh:
		m.testCase, cmd = m.testCase.Begin(m.ctx, m.client, projectID, changeID, op, 0, scenario, done)
	case testcases.Edit, testcases.Delete:
		m.testCase, cmd = m.testCase.BeginTarget(m.ctx, m.client, projectID, changeID, op, scenario, done)
	default:
		m.testCase, cmd = m.testCase.BeginRow(m.ctx, m.client, projectID, changeID, op, rawID, scenario, done)
	}
	if op != testcases.Refresh && cmd != nil {
		m.changeDocuments = m.changeDocuments.Invalidate()
		m.changeDocuments.Committed, m.changeDocuments.Err = "", nil
	}
	m.status = m.testCase.Status
	m.err = ""
	if m.testCase.Err != nil {
		m.err = m.testCase.Err.Error()
	}
	return m, cmd
}

func (m Model) saveTestCaseCreateValue(scenario string) (tea.Model, tea.Cmd) {
	return m.beginTestCase(testcases.Create, "", scenario, false)
}

func (m Model) saveTestCaseUpdateValue(scenario string) (tea.Model, tea.Cmd) {
	return m.beginTestCase(testcases.Edit, "", scenario, false)
}

func (m Model) applyTestCaseResult(r testcases.Result) (tea.Model, tea.Cmd) {
	if m.currentProject.ID != strconv.Itoa(r.ProjectID) || m.changeList.Detail.ID != strconv.Itoa(r.ChangeID) {
		return m, nil
	}
	next, ok := m.testCase.Apply(r)
	if !ok {
		return m, nil
	}
	m.testCase = next
	m.status, m.err = next.Status, ""
	if next.Err != nil {
		m.err = next.Err.Error()
	}
	if r.Err != nil {
		return m, nil
	}
	old := m.changeList.Detail
	selected, offset := m.changeList.DetailSelected, m.changeList.DetailOffset
	selectedID := ""
	if row, ok := changes.DetailRowAtSelection(old, selected); ok {
		selectedID = row.TestCaseID
	}
	refreshed := old
	if r.ChangeErr == nil {
		refreshed = changes.Present(r.Change)
		refreshed.Documents = old.Documents
		refreshed.Comments, refreshed.DocumentTypes = old.Comments, old.DocumentTypes
		refreshed.Brief, refreshed.Spec, refreshed.PR = old.Brief, old.Spec, old.PR
	}
	if r.RowsErr == nil {
		refreshed.TestCases = r.Rows
	}
	m.changeList = m.changeList.WithDetail(refreshed)
	m.changeList.DetailSelected, m.changeList.DetailOffset = selected, offset
	if selectedID != "" {
		for i, row := range changes.DetailRows(refreshed) {
			if row.TestCaseID == selectedID {
				m.changeList.DetailSelected = i
				break
			}
		}
	}
	m.changeList = m.changeList.ClampDetailSelection(m.changeTableRows(), terminalWidth(m.width))
	m.changeDetailLoaded = r.RefreshErr == nil
	m.state = ChangeDetailsState
	m.detailEditField = ""
	m.testCase = m.testCase.ClearForm()
	m = m.setPromptValue("")
	return m, nil
}

func changeNumericID(change dto.ChangeView) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(change.ID))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("change ID must be a valid positive number")
	}
	return id, nil
}

func currentProjectNumericID(projectID string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(projectID))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("current project must be numeric")
	}
	return id, nil
}

func toggleChangeType(current []string, selected dto.Option) []string {
	selectedID := strings.TrimSpace(selected.ID)
	if selectedID == "" {
		selectedID = strings.TrimSpace(selected.Label)
	}
	next := make([]string, 0, len(current)+1)
	removed := false
	for _, changeType := range current {
		if changeType == selectedID || changeType == selected.Label {
			removed = true
			continue
		}
		next = append(next, changeType)
	}
	if !removed && selectedID != "" {
		next = append(next, selectedID)
	}
	sort.Strings(next)
	return next
}

func normalizeTypeSet(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	next := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		next = append(next, value)
	}
	sort.Strings(next)
	return next
}

func selectedEpicID(selected dto.Option) (*int, error) {
	if selected.ID == "@none" {
		return nil, nil
	}
	epicID, err := strconv.Atoi(strings.TrimSpace(selected.ID))
	if err != nil || epicID <= 0 {
		return nil, fmt.Errorf("epic ID must be numeric")
	}
	return &epicID, nil
}

func (m Model) beginChange(op changes.Operation, id int, in changes.Input) (tea.Model, tea.Cmd) {
	project, _ := strconv.Atoi(m.currentProject.ID)
	var cmd tea.Cmd
	m.changeList, cmd = m.changeList.Begin(m.ctx, m.client, documents.Access{API: m.client, Types: m.optionCatalog.config.ChangeDocs}, op, project, id, in, m.optionCatalog.config)
	if op != changes.List && op != changes.Details && cmd != nil {
		m.changeDocuments = m.changeDocuments.Invalidate()
		m.changeDocuments.Committed, m.changeDocuments.Err = "", nil
	}
	if op == changes.Details && cmd != nil {
		m.changeDetailLoaded = false
		m.testCase = m.testCase.Invalidate()
		m.testCase.Loaded = false
	}
	m.status = m.changeList.Status
	m.err = ""
	if m.changeList.Err != nil {
		m.err = m.changeList.Err.Error()
	}
	if cmd == nil && m.status == "unchanged" {
		m.state = ChangeDetailsState
		m.detailEditField = ""
		m = m.setPromptValue("")
	}
	return m, cmd
}

func (m Model) applyChangeResult(r changes.Result) (tea.Model, tea.Cmd) {
	if strconv.Itoa(r.ProjectID) != m.currentProject.ID {
		return m, nil
	}
	next, ok := m.changeList.Apply(r)
	if !ok {
		return m, nil
	}
	selected, offset := m.changeList.DetailSelected, m.changeList.DetailOffset
	returning := m.historyDetailReload && !m.historyOpen && m.state == ChangeDetailsState && r.Operation == changes.Details && r.ProjectID == m.history.ProjectID && r.ID == m.history.OwnerID
	next.Detail.DocumentTypes = append([]string(nil), m.optionCatalog.config.ChangeDocs...)
	m.changeList = next
	m.changeDetailLoaded = next.DetailLoaded
	if r.Operation == changes.List && r.Err == nil {
		m.restoreSelectedChange()
	}
	if r.Operation == changes.Details && next.DetailLoaded {
		m.testCase.Rows = append([]dto.TestCase(nil), next.Detail.TestCases...)
		m.testCase.Loaded = true
	}
	if returning || (r.Operation != changes.Create && r.Operation != changes.Details && r.Operation != changes.List && r.Operation != changes.Delete) {
		m.changeList.DetailSelected = selected
		m.changeList.DetailOffset = offset
		if returning {
			m.changeList = m.changeList.ClampDetailSelection(m.changeTableRows(), terminalWidth(m.width))
		}
	}
	m.status = next.Status
	if r.Operation == changes.Details && next.DetailLoaded && len(next.Detail.TestCases) == 0 {
		m.status += "; " + testcases.Summary(m.testCase)
	}
	m.err = ""
	if next.Err != nil {
		m.err = next.Err.Error()
	}
	if returning {
		m.status = "returned from history; " + m.status
		if m.history.Committed != "" {
			m.status = m.history.Committed + "; " + m.status
		}
		m.historyDetailReload = !next.DetailLoaded
	}
	if r.Err == nil && r.Operation != changes.List && r.Operation != changes.Details {
		m.state = ChangeDetailsState
		if r.Operation == changes.Delete || r.Operation == changes.Reactivate {
			m.state = ChangesListState
		}
		m.detailEditField = ""
		m = m.setPromptValue("")
	}
	if m.state == ChangesListState && !m.changeList.Inactive {
		m.status = strings.ReplaceAll(m.status, "/retry reads only", "return to Main and reopen /changes")
	}
	return m, nil
}

func (m Model) saveChangeCreate() (tea.Model, tea.Cmd) {
	return m.saveChangeCreateValue(m.promptValue())
}

func (m Model) saveChangeCreateValue(brief string) (tea.Model, tea.Cmd) {
	m.changeList = m.changeList.PrepareCreate(brief)
	in := m.changeList.Draft
	return m.beginChange(changes.Create, 0, in)
}

func (m Model) saveChangeUpdate() (tea.Model, tea.Cmd) {
	return m.saveChangeUpdateValue(m.promptValue())
}

func (m Model) saveChangeUpdateValue(value string) (tea.Model, tea.Cmd) {
	if m.detailEditField == "" {
		m.detailEditField = detailEditSpec
	}
	return m.saveChangeDetailTextValue(value)
}

func (m Model) saveChangeDetailTextValue(value string) (tea.Model, tea.Cmd) {
	id, _ := changeNumericID(m.changeList.Detail)
	in := changes.Input{Value: value}
	op := changes.Document
	switch m.detailEditField {
	case detailEditComment:
		op := documents.EditComment
		if m.commentID == 0 {
			op = documents.NewComment
		}
		return m.beginChangeDocumentMutation(op, m.commentID, value)
	case detailEditTitle:
		op = changes.Title
	case detailEditSlug:
		op = changes.Slug
	case detailEditPRUrl:
		op = changes.PRURL
	case detailEditAfterChange:
		op = changes.AfterChange
		var err error
		in.Association, err = changes.AssociationInput(value)
		if err != nil {
			m.err = err.Error()
			return m, nil
		}
	case detailEditDocument:
		in.DocumentType = m.changeList.Draft.DocumentType
	case detailEditBrief:
		in.DocumentType = "brief"
	case detailEditSpec:
		in.DocumentType = "spec"
	case detailEditPullRequest:
		in.DocumentType = "pr"
	case detailCreateTitle:
		m.changeList.Draft.Title = value
		m.detailEditField = ""
		m = m.setPromptValue(m.changeList.Draft.Value)
		raw := m.changeList.Draft.Value
		m.editorDraft = &raw
		return m, nil
	case detailCreateUUID:
		m.changeList.Draft.UUID = value
		m.detailEditField = ""
		m = m.setPromptValue(m.changeList.Draft.Value)
		raw := m.changeList.Draft.Value
		m.editorDraft = &raw
		return m, nil
	}
	return m.beginChange(op, id, in)
}

func (m Model) beginChangeField(field detailEditField) (tea.Model, tea.Cmd) {
	if !m.changeDetailLoaded {
		m.err = "load change details with /retry before editing"
		return m, nil
	}
	m.changeList = m.changeList.Invalidate()
	m.previousState = ChangeDetailsState
	m.state = ChangeDetailsState
	m.detailEditField = field
	value := m.changeList.Detail.Title
	switch field {
	case detailEditSlug:
		m.slugPrefix = ""
		value = ""
		if prefix, suffix, ok := strings.Cut(m.changeList.Detail.RefSlug, "-"); ok {
			m.slugPrefix = prefix + "-"
			value = suffix
		}
	case detailEditAfterChange:
		value = m.changeList.Detail.AfterChangeID
		if value == "null" {
			value = ""
		}
	case detailEditPRUrl:
		value = m.changeList.Detail.PRUrl
	}
	m = m.setPromptValue(value)
	m.editorDraft = &value
	m.input.Placeholder = "Enter value (Ctrl+C/Esc cancel)"
	return m, nil
}
