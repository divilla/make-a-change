package app

import (
	"cli/internal/changes"
	"cli/internal/documents"
	"cli/internal/dto"
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

type itemPrinter struct {
	body string
	err  error
}

func (p *itemPrinter) Print(_ context.Context, body string) (string, error) {
	p.body = body
	return body, p.err
}

func TestSelectedItemsViewFullBodyAndReturnWithoutMutation(t *testing.T) {
	for _, kind := range []string{"brief", "spec", "comment"} {
		for _, exit := range []tea.KeyType{tea.KeyEsc, tea.KeyCtrlC} {
			t.Run(kind+string(rune(exit)), func(t *testing.T) {
				m, a := app031Model(t)
				body := "first\n" + strings.Repeat("full body line\n", 50) + "tail"
				id := 9
				if kind == "brief" {
					id = 8
				}
				if kind == "comment" {
					id = 7
					m.changeList.Detail.Comments[0].Body = body
				} else {
					for i := range m.changeList.Detail.Documents {
						if m.changeList.Detail.Documents[i].ID == id {
							m.changeList.Detail.Documents[i].Body = body
						}
					}
				}
				m = select031Row(t, m, kind, id)
				selection := m.changeList.DetailSelected
				printer := &itemPrinter{}
				m.historyPrinter = printer
				m, cmd := sendKey(m, tea.KeySpace)
				require.NotNil(t, cmd)
				require.True(t, m.historyOpen)
				require.True(t, m.history.Preview)
				result := cmd()
				m = applyMsg(m, result)
				require.Equal(t, body, printer.body)
				require.Contains(t, m.View(), "ItemViewScreen")
				m, _ = sendKey(m, tea.KeyPgDown)
				require.Positive(t, m.history.Offset)
				m, _ = sendKey(m, tea.KeyPgUp)
				require.Zero(t, m.history.Offset)
				m, _ = sendKey(m, tea.KeyDown)
				require.Equal(t, 1, m.history.Offset)
				m, _ = sendKey(m, tea.KeyUp)
				require.Zero(t, m.history.Offset)
				m, _ = sendKey(m, tea.KeySpace)
				require.Empty(t, a.deletes)
				m, _ = sendKey(m, exit)
				require.False(t, m.historyOpen)
				require.Equal(t, selection, m.changeList.DetailSelected)
				require.Equal(t, ChangeDetailsState, m.state)
				m = applyMsg(m, result)
				require.False(t, m.historyOpen)
			})
		}
	}
}

func TestSelectedItemViewErrorAndEmptySlot(t *testing.T) {
	m, _ := app031Model(t)
	m = select031Row(t, m, "notes", 0)
	m, cmd := sendKey(m, tea.KeySpace)
	require.Nil(t, cmd)
	require.Contains(t, m.err, "no saved content")
	m = select031Row(t, m, "spec", 9)
	m.historyPrinter = &itemPrinter{err: errors.New("bat unavailable")}
	m, cmd = sendKey(m, tea.KeySpace)
	require.NotNil(t, cmd)
	m = applyMsg(m, cmd())
	require.Contains(t, m.err, "bat unavailable")
	m, _ = sendKey(m, tea.KeyCtrlC)
	require.False(t, m.historyOpen)
}

func TestHistoryKeysOnDocsAndUnavailableTestcases(t *testing.T) {
	for _, key := range []tea.KeyMsg{{Type: tea.KeyCtrlH}, {Type: tea.KeyRunes, Runes: []rune{'h'}}} {
		m, _ := app031Model(t)
		m = select031Row(t, m, "brief", 8)
		next, cmd := m.Update(key)
		require.NotNil(t, cmd)
		require.True(t, next.(Model).historyOpen)
		m.changeList.Detail.TestCases = []dto.TestCase{{ID: 31}}
		m = selectTestcase(m, "31")
		next, cmd = m.Update(key)
		require.Nil(t, cmd)
		require.Contains(t, next.(Model).err, "testcase history unavailable")
		require.False(t, next.(Model).historyOpen)
	}
}

func TestEnterOpensEditorForAllSelectedItemKinds(t *testing.T) {
	for _, kind := range []string{"brief", "spec", "comment", "testcase"} {
		t.Run(kind, func(t *testing.T) {
			m, _ := app031Model(t)
			if kind == "testcase" {
				m.changeList.Detail.TestCases = []dto.TestCase{{ID: 31, Scenario: "full\nscenario"}}
				m = selectTestcase(m, "31")
			} else {
				id := 9
				if kind == "brief" {
					id = 8
				}
				if kind == "comment" {
					id = 7
				}
				m = select031Row(t, m, kind, id)
			}
			next, cmd := sendKey(m, tea.KeyEnter)
			require.NotNil(t, cmd)
			require.Equal(t, "editor", next.status)
			if kind == "testcase" {
				require.Equal(t, TestCaseUpdateState, next.state)
				require.Equal(t, "full\nscenario", next.promptValue())
			}
		})
	}
}

func TestViewRejectsStaleScope(t *testing.T) {
	m, _ := app031Model(t)
	m = select031Row(t, m, "spec", 9)
	m.historyPrinter = &itemPrinter{}
	next, cmd := m.viewSelectedDetail(changes.DetailRow{DocumentID: 9, DocumentType: "spec"})
	m = next.(Model)
	result := cmd()
	m.currentProject = dto.Option{ID: "8"}
	m = applyMsg(m, result)
	require.Empty(t, m.history.Output)
	m.history = m.history.Invalidate()
	require.IsType(t, documents.HistoryResult{}, result)
}
