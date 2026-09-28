package app

import (
	"cli/internal/changes"
	"cli/internal/dto"
	"cli/internal/testcases"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testcaseScreen(f *fakeClient) Model {
	m := NewModelWithClient(f)
	m.currentProject = dto.Option{ID: "7", Label: "Project"}
	m.state = ChangeDetailsState
	m.changeDetailLoaded = true
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", ProjectID: "7", Title: "Change", Documents: []dto.Document{{Body: "spec draft"}}, TestCases: f.gotChange.TestCases})
	m.testCase = testcases.Model{ProjectID: 7, ChangeID: 12, Loaded: true, Rows: f.gotChange.TestCases}
	return m
}

func TestP503ShellRoutesTestCaseResultWithoutReimplementingMutation(t *testing.T) {
	f := &fakeClient{gotChange: dto.ChangeView{ID: "12", ProjectID: "7", Title: "Change", Done: 1, Total: 2, Completed: 50, TestCases: []dto.TestCase{{ID: 31, ChangeID: 12, Scenario: "created"}}}}
	m := testcaseScreen(f)
	m.state = TestCaseCreateState
	next, cmd := m.saveTestCaseCreateValue("created")
	require.NotNil(t, cmd)
	m = applyCommand(next.(Model), cmd)
	require.Equal(t, ChangeDetailsState, m.state)
	require.Equal(t, "saved test case (#31)", m.status)
	require.Equal(t, 31, m.testCase.CommittedID)
	require.Equal(t, 1, f.testCaseCreateCalls)
	require.Equal(t, 1, f.changeGetCalls)
	require.Equal(t, []string{"test-case/create", "change/details"}, f.requestOrder)
	require.Equal(t, int64(1), m.changeList.Detail.Done)
	require.Equal(t, int64(2), m.changeList.Detail.Total)
	require.Equal(t, "created", m.changeList.Detail.TestCases[0].Scenario)
}

func TestP503ShellRendersFeatureSuppliedTestCaseForms(t *testing.T) {
	m := testcaseScreen(&fakeClient{gotChange: dto.ChangeView{ID: "12", ProjectID: "7", Title: "Change"}})
	m.width = 160
	next, cmd := m.executeCommandFrom(ChangeDetailsState, "/new-testcase")
	require.Nil(t, cmd)
	m = next.(Model)
	require.Equal(t, TestCaseCreateState, m.state)
	create := testcases.CreateForm()
	require.Equal(t, create.Placeholder, m.input.Placeholder)
	require.Contains(t, m.View(), create.Placeholder)
	require.Contains(t, m.View(), create.Help)
	require.Contains(t, m.View(), create.Status)
	m = m.setPromptValue("literal /delete")
	require.Contains(t, m.View(), "literal /delete")

	next, cmd = m.beginTestCaseScenarioEdit(changes.DetailRow{TestCaseID: "31", TestCaseText: "old scenario"})
	require.Nil(t, cmd)
	m = next.(Model)
	require.Equal(t, TestCaseUpdateState, m.state)
	edit := testcases.EditForm()
	require.Equal(t, edit.Placeholder, m.input.Placeholder)
	require.Equal(t, "old scenario", m.promptValue())
	require.Contains(t, m.View(), edit.Help)
	require.Contains(t, m.View(), edit.Status)
	require.Contains(t, m.View(), "old scenario")
}

func TestP502SelectionAfterRefreshAndReloadSafety(t *testing.T) {
	f := &fakeClient{gotChange: dto.ChangeView{ID: "12", ProjectID: "7", Title: "Change", TestCases: []dto.TestCase{{ID: 31, ChangeID: 12, Scenario: "before"}}}}
	m := testcaseScreen(f)
	m.changeList.DetailSelected = 8
	m.testCase = testcases.Model{ProjectID: 7, ChangeID: 12, Revision: 4, Busy: true}
	r := testcases.Result{ProjectID: 7, ChangeID: 12, Revision: 4, Operation: testcases.Edit, ID: 31, Committed: true, Rows: []dto.TestCase{{ID: 31, ChangeID: 12, Scenario: "after"}}, Change: fakeWire(f.gotChange)}
	next, _ := m.applyTestCaseResult(r)
	m = next.(Model)
	_, row, ok := m.changeList.SelectDetailRow(m.changeTableRows(), terminalWidth(m.width))
	require.True(t, ok)
	require.Equal(t, "31", row.TestCaseID)
	require.Equal(t, "after", row.TestCaseText)
	m.changeDetailLoaded = false
	next, cmd := m.handleDetailSpaceToggle()
	require.Nil(t, cmd)
	require.Contains(t, next.(Model).err, "/retry")
	next, cmd = m.handleDetailDelete()
	require.Nil(t, cmd)
	require.Contains(t, next.(Model).err, "/retry")
	next, cmd = m.executeCommandFrom(ChangeDetailsState, "/new-testcase")
	require.Nil(t, cmd)
	require.Equal(t, ChangeDetailsState, next.(Model).state)
}

func TestP505TestCaseFailureDoesNotCorruptChangeOrDocumentDraft(t *testing.T) {
	f := &fakeClient{changeUpdateErr: errors.New("offline"), gotChange: dto.ChangeView{ID: "12", ProjectID: "7", Title: "Change"}}
	m := testcaseScreen(f)
	m.state = TestCaseCreateState
	m = m.setPromptValue("literal /delete")
	original := m.changeList.Detail
	next, cmd := m.saveTestCaseCreateValue(m.promptValue())
	require.NotNil(t, cmd)
	m = applyCommand(next.(Model), cmd)
	require.Equal(t, TestCaseCreateState, m.state)
	require.Equal(t, "save failed", m.status)
	require.Equal(t, "literal /delete", m.promptValue())
	require.Equal(t, original, m.changeList.Detail)
	require.Zero(t, f.changeGetCalls)
}

func TestP502EmptyLoadingErrorAndScrollableDetails(t *testing.T) {
	f := &fakeClient{gotChange: dto.ChangeView{ID: "12", ProjectID: "7", Title: "Change"}}
	m := testcaseScreen(f)
	require.Equal(t, "no test cases", testcases.Summary(m.testCase))
	m.testCase.Busy = true
	require.Equal(t, "loading test cases", testcases.Summary(m.testCase))
	m.testCase.Busy = false
	m.testCase.Loaded = false
	require.Contains(t, testcases.Summary(m.testCase), "/retry")
	m.err = "read failed"
	require.Contains(t, m.View(), "read failed")
	when := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 20; i++ {
		m.changeList.Detail.TestCases = append(m.changeList.Detail.TestCases, dto.TestCase{ID: 31, Scenario: "case scenario", CreatedAt: when, UpdatedAt: when})
	}
	m.changeList = m.changeList.ScrollDetailViewport(20, 8, 100)
	rows := changes.DetailRows(m.changeList.Detail)
	require.True(t, len(rows) > 20)
	require.Contains(t, m.View(), "case scenario")
	require.Contains(t, strings.Join([]string{rows[8].Text}, ""), "created 2026-09-28")
}
