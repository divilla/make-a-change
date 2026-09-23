package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const defaultPromptCharLimit = 240

func (m Model) setPromptValue(value string) Model {
	m.applyPromptLimit()
	m.input.SetValue(value)
	m.movePromptCursorToEnd()
	return m
}

func (m *Model) applyPromptLimit() {
	if m.state == ChangeCreateState || m.state == ChangeUpdateState ||
		m.detailEditField == detailEditDef || m.detailEditField == detailEditSpec ||
		m.detailEditField == detailEditPullRequest {
		m.input.CharLimit = 0
		return
	}
	m.input.CharLimit = defaultPromptCharLimit
}

func (m *Model) preparePromptInput() {
	m.applyPromptLimit()
	m.input.SetWidth(terminalWidth(m.width))
	m.input.SetHeight(len(promptValueLines(m.input.Value())))
	m.clampPromptCursor()
}

func (m *Model) movePromptCursorToEnd() {
	lines := promptValueLines(m.input.Value())
	m.promptCursorRow = len(lines) - 1
	m.promptCursorCol = runeCount(lines[m.promptCursorRow])
}

func (m *Model) clampPromptCursor() {
	lines := promptValueLines(m.input.Value())
	if m.promptCursorRow < 0 {
		m.promptCursorRow = 0
	}
	if m.promptCursorRow >= len(lines) {
		m.promptCursorRow = len(lines) - 1
	}
	width := runeCount(lines[m.promptCursorRow])
	if m.promptCursorCol < 0 {
		m.promptCursorCol = 0
	}
	if m.promptCursorCol > width {
		m.promptCursorCol = width
	}
}

func (m *Model) insertPromptCursorText(value string) {
	parts := strings.Split(value, "\n")
	if len(parts) == 1 {
		m.promptCursorCol += runeCount(parts[0])
		return
	}
	m.promptCursorRow += len(parts) - 1
	m.promptCursorCol = runeCount(parts[len(parts)-1])
}

func (m *Model) movePromptCursorHorizontal(delta int) {
	lines := promptValueLines(m.input.Value())
	m.clampPromptCursor()
	nextCol := m.promptCursorCol + delta
	width := runeCount(lines[m.promptCursorRow])
	switch {
	case nextCol >= 0 && nextCol <= width:
		m.promptCursorCol = nextCol
	case delta < 0 && m.promptCursorRow > 0:
		m.promptCursorRow--
		m.promptCursorCol = runeCount(lines[m.promptCursorRow])
	case delta > 0 && m.promptCursorRow < len(lines)-1:
		m.promptCursorRow++
		m.promptCursorCol = 0
	}
}

func (m *Model) movePromptCursorVertical(delta int) {
	lines := promptValueLines(m.input.Value())
	m.clampPromptCursor()
	m.promptCursorRow += delta
	if m.promptCursorRow < 0 {
		m.promptCursorRow = 0
	}
	if m.promptCursorRow >= len(lines) {
		m.promptCursorRow = len(lines) - 1
	}
	if width := runeCount(lines[m.promptCursorRow]); m.promptCursorCol > width {
		m.promptCursorCol = width
	}
}

func (m *Model) movePromptCursorAfterBackspace() {
	lines := promptValueLines(m.input.Value())
	m.clampPromptCursor()
	if m.promptCursorCol > 0 {
		m.promptCursorCol--
		return
	}
	if m.promptCursorRow > 0 {
		m.promptCursorRow--
		m.promptCursorCol = runeCount(lines[m.promptCursorRow])
	}
}

func (m *Model) mirrorPromptKey(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyRunes:
		m.insertPromptCursorText(string(msg.Runes))
	case tea.KeySpace:
		m.insertPromptCursorText(" ")
	case tea.KeyBackspace, tea.KeyCtrlH:
		m.movePromptCursorAfterBackspace()
	case tea.KeyLeft, tea.KeyCtrlB:
		m.movePromptCursorHorizontal(-1)
	case tea.KeyRight, tea.KeyCtrlF:
		m.movePromptCursorHorizontal(1)
	case tea.KeyUp, tea.KeyCtrlP:
		m.movePromptCursorVertical(-1)
	case tea.KeyDown, tea.KeyCtrlN:
		m.movePromptCursorVertical(1)
	case tea.KeyHome, tea.KeyCtrlA:
		m.promptCursorCol = 0
	case tea.KeyEnd:
		m.clampPromptCursor()
		lines := promptValueLines(m.input.Value())
		m.promptCursorCol = runeCount(lines[m.promptCursorRow])
	}
}

func (m Model) updatePromptInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	m.preparePromptInput()
	m.mirrorPromptKey(msg)
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.clampPromptCursor()
	return m, cmd
}

func (m Model) insertPromptNewline() Model {
	m.preparePromptInput()
	m.input.InsertString("\n")
	m.insertPromptCursorText("\n")
	m.clampPromptCursor()
	return m
}

func (m Model) handlePendingShiftEnter(msg tea.KeyMsg) (Model, bool) {
	if !m.pendingAltO {
		return m, false
	}
	m.pendingAltO = false
	if msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] == 'M' && !msg.Alt {
		return m.insertPromptNewline(), true
	}
	m = m.insertPromptLiteral("O")
	return m, false
}

func (m Model) insertPromptLiteral(value string) Model {
	m.preparePromptInput()
	m.input.InsertString(value)
	m.insertPromptCursorText(value)
	m.clampPromptCursor()
	return m
}

func isShiftEnterPrefix(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyRunes && msg.Alt && len(msg.Runes) == 1 && msg.Runes[0] == 'O'
}

func runeCount(value string) int {
	return len([]rune(value))
}
