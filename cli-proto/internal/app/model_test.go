package app

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptLineFillsWidth(t *testing.T) {
	input := textinput.New()
	input.Placeholder = "/change-new or /cancel"
	m := model{input: input}

	line := m.promptLine(40)

	assert.Equal(t, 40, lipgloss.Width(line))
}

func TestPromptLineRendersCursorInsideValue(t *testing.T) {
	input := textinput.New()
	input.SetValue("/change-new")
	input.SetCursor(7)
	m := model{input: input}

	line := m.promptLine(30)

	assert.Equal(t, 30, lipgloss.Width(line))
}

func TestSlashOpensMenuAtBeginningOfLine(t *testing.T) {
	input := textinput.New()
	m := model{input: input, screen: screenPlanning}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	require.Nil(t, cmd)
	result := updated.(model)

	assert.True(t, result.menuOpen)
	assert.Equal(t, 0, result.menuIndex)
	assert.Equal(t, "/", result.input.Value())
}

func TestSlashDoesNotOpenMenuAfterInput(t *testing.T) {
	input := textinput.New()
	input.SetValue("hello")
	m := model{input: input, screen: screenPlanning}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	result := updated.(model)

	assert.False(t, result.menuOpen)
}

func TestChangeNewMenuSelectionIsRejectedMidFlow(t *testing.T) {
	input := textinput.New()
	m := model{input: input, screen: screenIdea, menuOpen: true}

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	result := updated.(model)

	assert.False(t, result.menuOpen)
	assert.Contains(t, result.errText, "only available")
}

func TestLoadingViewShowsElapsedSeconds(t *testing.T) {
	m := newModel("", "", Config{}, nil, []Project{{ID: 1, Name: "demo"}}, nil, nil, nil)
	m.elapsed = 3

	view := m.loadingView()

	assert.Contains(t, view, "Waiting for Codex")
	assert.Contains(t, view, "3 seconds")
}
