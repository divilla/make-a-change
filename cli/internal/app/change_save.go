package app

import (
	"cli/internal/changes"
	"cli/internal/documents"
	"cli/internal/dto"
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) saveTestCaseCreateValue(scenario string) (tea.Model, tea.Cmd) {
	changeID, err := changeNumericID(m.changeList.Detail)
	if err != nil {
		m.err = err.Error()
		m.status = "validation failed"
		return m, nil
	}
	if strings.TrimSpace(scenario) == "" {
		m.err = "test case scenario is required"
		m.status = "validation failed"
		return m, nil
	}
	m.status = "saving test case"
	return m, testCaseCreateCommand(m.client, changeID, scenario)
}

func (m Model) saveTestCaseUpdateValue(scenario string) (tea.Model, tea.Cmd) {
	testCaseID, err := testCaseNumericID(m.activeTestCase.ID)
	if err != nil {
		m.err = err.Error()
		m.status = "validation failed"
		return m, nil
	}
	if strings.TrimSpace(scenario) == "" {
		m.err = "test case scenario is required"
		m.status = "validation failed"
		return m, nil
	}
	m.status = "saving test case"
	return m, testCaseUpdateCommand(m.client, testCaseID, scenario)
}

func testCaseCreateCommand(client appClient, changeID int, scenario string) tea.Cmd {
	return func() tea.Msg {
		change, err := client.CreateTestCase(changeID, scenario)
		return changeSavedMsg{source: TestCaseCreateState, change: change, err: err}
	}
}

func testCaseUpdateCommand(client appClient, testCaseID int, scenario string) tea.Cmd {
	return func() tea.Msg {
		change, err := client.UpdateTestCase(testCaseID, scenario)
		return changeSavedMsg{source: TestCaseUpdateState, change: change, err: err}
	}
}

func testCaseDeleteCommand(client appClient, testCase dto.TestCase) tea.Cmd {
	return func() tea.Msg {
		testCaseID, err := testCaseNumericID(testCase.ID)
		if err != nil {
			return changeSavedMsg{source: ChangeDetailsState, err: err}
		}
		change, err := client.DeleteTestCase(testCaseID)
		return changeSavedMsg{source: ChangeDetailsState, change: change, err: err}
	}
}

func changeDetailTestCaseDoneUpdateCommand(client appClient, change dto.ChangeView, row changes.DetailRow) tea.Cmd {
	return func() tea.Msg {
		changeID, err := changeNumericID(change)
		if err != nil {
			return changeSavedMsg{source: ChangeDetailsState, err: err}
		}
		testCaseID, err := testCaseNumericID(row.TestCaseID)
		if err != nil {
			return changeSavedMsg{source: ChangeDetailsState, err: err}
		}
		if _, err := client.UpdateTestCaseDone(testCaseID, !row.TestCaseDone); err != nil {
			return changeSavedMsg{source: ChangeDetailsState, err: err}
		}
		wire, err := client.GetChange(context.Background(), changeID)
		change := changes.Present(wire)
		if err == nil {
			change.TestCases, err = client.ListTestCases(context.Background(), changeID)
		}
		return changeSavedMsg{source: ChangeDetailsState, change: change, err: err}
	}
}

func changeNumericID(change dto.ChangeView) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(change.ID))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("change ID must be a valid positive number")
	}
	return id, nil
}

func testCaseNumericID(idValue string) (int, error) {
	id, err := strconv.Atoi(strings.TrimSpace(idValue))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("test case ID must be a valid positive number")
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
	if op == changes.Details && cmd != nil {
		m.changeDetailLoaded = false
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
	m.changeList = next
	m.changeDetailLoaded = next.DetailLoaded
	if r.Operation != changes.Create && r.Operation != changes.Details && r.Operation != changes.List && r.Operation != changes.Delete {
		m.changeList.DetailSelected = selected
		m.changeList.DetailOffset = offset
	}
	m.status = next.Status
	m.err = ""
	if next.Err != nil {
		m.err = next.Err.Error()
	}
	if r.Err == nil && r.Operation != changes.List && r.Operation != changes.Details {
		m.state = ChangeDetailsState
		if r.Operation == changes.Delete {
			m.state = ChangesListState
		}
		m.detailEditField = ""
		m = m.setPromptValue("")
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
	case detailEditTitle:
		op = changes.Title
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
	m.state = ChangeUpdateState
	m.detailEditField = field
	value := m.changeList.Detail.Title
	switch field {
	case detailEditAfterChange:
		value = m.changeList.Detail.AfterChangeID
	case detailEditPRUrl:
		value = m.changeList.Detail.PRUrl
	}
	m = m.setPromptValue(value)
	m.editorDraft = &value
	m.input.Placeholder = "Enter value (Ctrl+C clears, Esc cancels)"
	return m, nil
}
