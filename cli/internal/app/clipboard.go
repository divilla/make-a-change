package app

import (
    "fmt"

    "cli/internal/changes"

    "github.com/atotto/clipboard"
    tea "github.com/charmbracelet/bubbletea"
)

type detailCopiedMsg struct {
    label string
    err   error
}

var writeClipboard = clipboard.WriteAll

func (m Model) handleDetailCopy() (tea.Model, tea.Cmd) {
    next, row, ok := m.changeList.SelectDetailRow(m.changeTableRows(), terminalWidth(m.width))
    m.changeList = next
    if !ok {
        m.err = "no change details selectable"
        return m, nil
    }
    value := changes.DetailCopyValue(row)
    m.status = "copying " + row.Label
    return m, detailCopyCommand(row.Label, value)
}

func detailCopyCommand(label string, value string) tea.Cmd {
    return func() tea.Msg {
        if err := writeClipboard(value); err != nil {
            return detailCopiedMsg{label: label, err: err}
        }
        return detailCopiedMsg{label: label}
    }
}

func detailCopyStatus(label string) string {
    return fmt.Sprintf("copied %s", label)
}
