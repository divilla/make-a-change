package app

import (
	"cli/internal/dto"
	"cli/internal/styles"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func TestAllConfirmationsDisplayPlainChoices(t *testing.T) {
	for _, tc := range []struct {
		name     string
		state    State
		previous State
		target   State
	}{
		{"change list", ChangeDeleteConfirmation, ChangesListState, ChangesListState},
		{"change details", ChangeDeleteConfirmation, ChangeDetailsState, ChangesListState},
		{"testcase", TestCaseDeleteConfirmation, ChangeDetailsState, ChangeDetailsState},
		{"epic", EpicDeleteConfirmation, EpicDetailsState, EpicsListState},
		{"project", ProjectDeleteConfirmation, ProjectDetailsState, ProjectsListState},
		{"document", ChangeDetailsState, ChangeDetailsState, ChangeDetailsState},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := Model{state: tc.previous}
			if tc.name == "document" {
				next, _ := m.openDocumentConfirmation(9)
				m = next.(Model)
			} else {
				m.openConfirmation(tc.state, tc.previous, tc.target)
			}
			for _, width := range []int{20, 80, 240} {
				for _, selected := range []int{0, 1} {
					m.dropdown.highlighted = selected
					require.Contains(t, m.dropdownView(width), styles.Default.MenuPromptIndicator.Render(" Are you sure? > "))
					lines := strings.Split(stripANSI(m.dropdownView(width)), "\n")
					require.Len(t, lines, 5)
					require.Equal(t, strings.Repeat("▄", width), lines[0])
					require.Contains(t, lines[1], " Are you sure? > ")
					require.Equal(t, strings.Repeat("▀", width), lines[2])
					require.Equal(t, "yes", strings.TrimSpace(lines[3]))
					require.Equal(t, "no", strings.TrimSpace(lines[4]))
				}
			}
			// Both plain input and existing slash-prefixed input select the same choice.
			for _, answer := range []string{"yes", "no", "/yes", "/no"} {
				m.dropdown.filter = answer
				m.dropdown.highlighted = 0
				require.Equal(t, "/"+strings.TrimPrefix(answer, "/"), m.selectedOption().ID)
			}
			m.dropdown.filter = ""
			m.dropdown.highlighted = 0
			m, _ = sendKey(m, tea.KeyDown)
			require.Equal(t, "/no", m.selectedOption().ID)
			m, cmd := sendKey(m, tea.KeyEnter)
			require.Nil(t, cmd)
			require.Equal(t, tc.previous, m.state)
			require.False(t, m.hasDropdown())
		})
	}
}

func TestTestcaseConfirmationStaysBelowChangeDetails(t *testing.T) {
	for _, height := range []int{24, 40} {
		m := newChangeTestModel(&fakeClient{})
		m.width, m.height = 100, height
		m.state = ChangeDetailsState
		m.changeList = m.changeList.WithDetail(dto.ChangeView{
			ID: "12", Title: "Visible change",
			TestCases: []dto.TestCase{{ID: 31, Scenario: "Visible testcase", ChangeID: 12}},
		})
		m = selectTestcase(m, "31")
		m, cmd := sendKey(m, tea.KeyDelete)
		require.Nil(t, cmd)
		view := stripANSI(m.View())
		require.Contains(t, view, "ChangeDetailsScreen")
		require.NotContains(t, view, "TestCaseDeleteConfirmationScreen")
		require.Contains(t, view, "Visible change")
		require.Contains(t, view, "Visible testcase")
		require.Less(t, strings.Index(view, "Visible testcase"), strings.Index(view, "Are you sure? >"))
		require.Equal(t, height, lipgloss.Height(view))
		lines := strings.Split(view, "\n")
		promptRow := -1
		for row, line := range lines {
			if strings.Contains(line, "Are you sure? >") {
				promptRow = row
			}
		}
		require.GreaterOrEqual(t, promptRow, height-8, "confirmation stays at the bottom above feedback and footer")
		require.Equal(t, "yes", strings.TrimSpace(lines[promptRow+2]))
		require.Equal(t, "no", strings.TrimSpace(lines[promptRow+3]))
		m, cmd = sendKey(m, tea.KeyEsc)
		require.Nil(t, cmd)
		require.Equal(t, ChangeDetailsState, m.state)
		require.Len(t, m.changeList.Detail.TestCases, 1)
	}
}
