package app

import (
	"cli/internal/dto"
	"cli/internal/styles"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandMenuLayoutAndColors(t *testing.T) {
	m := NewModel()
	m.state = ChangeDetailsState
	m.openCommandDropdown()
	m.dropdown.options = append(m.dropdown.options, dto.Option{ID: "/example", Label: "/example"})
	require.Len(t, m.dropdown.options, 20)
	m.dropdown.highlighted = 3

	view := m.dropdownView(80)
	lines := strings.Split(view, "\n")
	require.Len(t, lines, 12) // prompt with half-row edges, eight options, one counter
	assert.Equal(t, strings.Repeat("▄", 80), stripANSI(lines[0]))
	assert.True(t, strings.HasPrefix(stripANSI(lines[1]), " > /"))
	assert.Equal(t, strings.Repeat("▀", 80), stripANSI(lines[2]))
	assert.True(t, strings.HasPrefix(stripANSI(lines[3]), "    new-comment"))
	assert.Contains(t, stripANSI(lines[6]), "title")
	assert.Contains(t, stripANSI(lines[6]), "Edit the title")
	assert.Equal(t, 80, lipgloss.Width(lines[1]))
	assert.Contains(t, lines[1], promptCursorWithStyle(styles.Default.MenuPrompt))
	assert.Equal(t, 80, lipgloss.Width(lines[6]))
	assert.Equal(t, "(4/20)", stripANSI(lines[11]))
	assert.NotContains(t, stripANSI(view), "▲")
	assert.NotContains(t, stripANSI(view), "▼")
	assert.Equal(t, styles.InputBackground, styles.Default.MenuPrompt.GetBackground())
	assert.Equal(t, styles.AccentGreen, styles.Default.MenuPrompt.GetForeground())
	assert.Equal(t, styles.AccentPurple, styles.Default.MenuPromptIndicator.GetForeground())
	assert.Equal(t, styles.InputBackground, styles.Default.PromptEdge.GetForeground())
	assert.Equal(t, styles.Gray, styles.Default.MenuItem.GetForeground())
	assert.Equal(t, styles.AccentGreen, styles.Default.MenuSelected.GetForeground())
	assert.Equal(t, styles.MenuBackground, styles.Default.MenuSelected.GetBackground())
	assert.Equal(t, styles.DarkGray, styles.Default.MenuCounter.GetForeground())
}

func TestMenuCounterHidesBelowTenOptionsAndFollowsSelection(t *testing.T) {
	m := NewModel()
	m.state = ChangeDetailsState
	m.openCommandDropdown()
	m.dropdown.highlighted = 18
	lines := strings.Split(stripANSI(m.dropdownView(80)), "\n")
	assert.Equal(t, "(19/19)", lines[len(lines)-1])
	assert.Contains(t, strings.Join(lines, "\n"), "brief-clarify")

	m.dropdown.options = m.dropdown.options[:9]
	m.dropdown.highlighted = 8
	view := stripANSI(m.dropdownView(80))
	assert.Len(t, strings.Split(view, "\n"), 12) // three prompt rows and all nine options
	assert.NotContains(t, view, "(9/9)")
	assert.Contains(t, view, "active")
}

func TestEveryCommandMenuItemHasHelp(t *testing.T) {
	for state, commands := range commandsByState {
		for _, command := range commands {
			assert.NotEmpty(t, commandDescriptions[command], "%s in %s", command, state)
		}
	}
}

func TestCommandMenuCanCloseFromEveryScreen(t *testing.T) {
	for state := range commandsByState {
		for _, key := range []tea.KeyType{tea.KeyBackspace, tea.KeyDelete, tea.KeyCtrlC} {
			t.Run(fmt.Sprintf("%s/%s", state, key), func(t *testing.T) {
				m := NewModel()
				m.state = state
				m.openCommandDropdown()
				got, _ := sendKey(m, key)
				assert.Equal(t, state, got.state)
				assert.False(t, got.hasDropdown())
			})
		}
	}
}

func TestCommandMenuDeletesFilterBeforeSlash(t *testing.T) {
	m := NewModel()
	m.state = ChangesListState
	m.openCommandDropdown()
	m.dropdown.filter = "é"
	got, _ := sendKey(m, tea.KeyBackspace)
	assert.True(t, got.hasDropdown())
	assert.Empty(t, got.dropdown.filter)
	got, _ = sendKey(got, tea.KeyBackspace)
	assert.False(t, got.hasDropdown())
}

func TestCommandMenuShowsEachTypedCharacterAndRestoresMatches(t *testing.T) {
	m := NewModel()
	m.state = MainState
	m.width = 80
	m, _ = sendRune(m, '/')
	assert.Contains(t, stripANSI(m.dropdownView(80)), " > / ")
	assert.NotEmpty(t, m.filteredOptions())

	m, _ = sendRune(m, '/')
	view := stripANSI(m.dropdownView(80))
	assert.Contains(t, view, " > // ")
	assert.Contains(t, view, "Commands: no options")
	assert.Empty(t, m.filteredOptions())

	m, _ = sendKey(m, tea.KeyBackspace)
	view = stripANSI(m.dropdownView(80))
	assert.Contains(t, view, " > / ")
	assert.NotContains(t, view, "Commands: no options")
	assert.NotEmpty(t, m.filteredOptions())

	m, _ = sendRune(m, 'q')
	assert.Contains(t, stripANSI(m.dropdownView(80)), " > /q ")
	require.Len(t, m.filteredOptions(), 1)
	assert.Equal(t, "/quit", m.filteredOptions()[0].ID)
	m, _ = sendKey(m, tea.KeyBackspace)
	assert.Contains(t, stripANSI(m.dropdownView(80)), " > / ")

	m, _ = sendKey(m, tea.KeySpace)
	assert.Contains(t, stripANSI(m.dropdownView(80)), " > /  ")
	assert.Empty(t, m.filteredOptions())
}

func TestMenuCursorRemainsVisibleAsFilterChanges(t *testing.T) {
	for _, kind := range []dropdownKind{dropdownCommand, dropdownSelect} {
		t.Run(string(kind), func(t *testing.T) {
			m := NewModel()
			m.state = MainState
			if kind == dropdownCommand {
				m.openCommandDropdown()
			} else {
				m.openDropdown(SelectPhaseDropDown, dropdownSelect, MainState, MainState, "Phase", nil, false)
			}
			for _, filter := range []string{"", "q", strings.Repeat("x", 100)} {
				m.dropdown.filter = filter
				line := strings.Split(m.dropdownView(20), "\n")[1]
				assert.Contains(t, line, promptCursorWithStyle(styles.Default.MenuPrompt))
				assert.Equal(t, 20, lipgloss.Width(line))
			}
		})
	}
}

func TestChangesMenuKeepsMaximumVisibleTableHeight(t *testing.T) {
	m := NewModel()
	m.state = ChangesListState
	m.width, m.height = 120, 30
	for i := 1; i <= 30; i++ {
		m.changeList.Rows = append(m.changeList.Rows, dto.ChangeView{ID: fmt.Sprint(i), Ref: fmt.Sprint(i), Title: fmt.Sprintf("Change %02d", i)})
	}
	closed := stripANSI(m.View())
	closedBottom := strings.Index(closed, "└")
	require.NotEqual(t, -1, closedBottom)

	m.openCommandDropdown()
	opened := stripANSI(m.View())
	require.LessOrEqual(t, lipgloss.Height(m.View()), m.height)
	assert.Contains(t, opened, "#Ref")
	assert.Contains(t, opened, "Change 01")
	assert.Contains(t, opened, "    new-change")
	openBottom := strings.Index(opened, "└")
	require.NotEqual(t, -1, openBottom)
	assert.Less(t, openBottom, closedBottom)
	lines := strings.Split(opened, "\n")
	for i, line := range lines[:len(lines)-1] {
		if strings.Contains(line, "└") {
			assert.True(t, strings.HasPrefix(lines[i+1], "▄"), "menu should follow the table border")
			break
		}
	}
}
