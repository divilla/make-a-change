package app

import (
	"cli/internal/changes"
	"cli/internal/dto"
	"cli/internal/styles"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/stretchr/testify/require"
)

func selectDetailLabel(m Model, label string) Model {
	for i, row := range changes.DetailRows(m.changeList.Detail) {
		if row.Label == label {
			m.changeList.DetailSelected = i
			return m
		}
	}
	panic("missing detail label: " + label)
}

func selectTestcase(m Model, id string) Model {
	for i, row := range changes.DetailRows(m.changeList.Detail) {
		if row.TestCaseID == id {
			m.changeList.DetailSelected = i
			return m
		}
	}
	panic("missing testcase: " + id)
}

func TestErrorRowReservedAcrossScreens(t *testing.T) {
	for state := range commandsByState {
		t.Run(string(state), func(t *testing.T) {
			m := NewModelWithClient(&fakeClient{})
			m.state, m.width = state, 160
			empty, _ := m.viewLines()
			require.Empty(t, empty[len(empty)-2])
			m.err = "first\nsecond" + string(rune(27)) + "[2J"
			filled, _ := m.viewLines()
			require.Len(t, filled, len(empty))
			require.Contains(t, stripANSI(filled[len(filled)-2]), "Error: first second")
			require.NotContains(t, filled[len(filled)-2], "\n")
		})
	}
	m := NewModelWithClient(&fakeClient{})
	m.historyOpen = true
	lines, _ := m.viewLines()
	require.Empty(t, lines[len(lines)-2])
	require.Contains(t, stripANSI(lines[len(lines)-3]), " > ")
}

func TestSelectedItemFooterActions(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", DocumentTypes: []string{"brief"}, TestCases: []dto.TestCase{{ID: 31}}, Comments: []dto.Document{{ID: 7}}})
	for _, label := range []string{"Docs", "Testcases", "Comments"} {
		m = selectDetailLabel(m, label)
		require.Contains(t, m.helpText(), "<return> editor")
		if label == "Testcases" {
			require.Contains(t, m.helpText(), "<space> toggle done")
			require.NotContains(t, m.helpText(), "<space> view")
			require.NotContains(t, m.helpText(), "<d>")
		} else {
			require.Contains(t, m.helpText(), "<space> view")
		}
		require.Contains(t, m.helpText(), "<del> delete")
		if label != "Comments" {
			require.Contains(t, m.helpText(), "<h> or <ctrl+h> history")
		}
	}
	m = selectDetailLabel(m, "Active")
	require.Contains(t, m.helpText(), "toggle active")
	require.NotContains(t, m.helpText(), "delete")
}

func TestLongErrorKeepsCauseAndOperationInOneRedLine(t *testing.T) {
	m := NewModelWithClient(&fakeClient{})
	m.err = "/api/v1/test-case/list: " + strings.Repeat("long prefix ", 20) + "read unavailable"
	value := m.errorLine(80)
	require.LessOrEqual(t, lipgloss.Width(value), 80)
	require.NotContains(t, value, "\n")
	require.Contains(t, stripANSI(value), "/api/v1/test-case/list")
	require.Contains(t, stripANSI(value), "read unavailable")
	require.Contains(t, stripANSI(value), "…")
	require.Equal(t, styles.AccentRed, styles.Default.Error.GetForeground())
	m.helpQuery = "query"
	lines, _ := m.viewLines()
	require.Contains(t, lines[len(lines)-3], "▀")
}
