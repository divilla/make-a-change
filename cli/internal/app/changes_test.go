package app

import (
	"cli/internal/changes"
	"cli/internal/dto"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func TestP403ClearingUpdatePromptPreservesField(t *testing.T) {
	for _, tc := range []struct {
		command string
		value   string
		field   detailEditField
		op      changes.Operation
	}{
		{"/pr-url", "https://example.test/replacement", detailEditPRUrl, changes.PRURL},
		{"/after-change", "9", detailEditAfterChange, changes.AfterChange},
	} {
		t.Run(tc.command, func(t *testing.T) {
			original := dto.ChangeView{ID: "12", Spec: "Existing specification", PRUrl: "https://example.test/old", AfterChangeID: "8"}
			client := &fakeClient{gotChange: original}
			m := newChangeTestModel(client)
			m.state = ChangeDetailsState
			m.changeList.Detail = original
			m, _ = sendCommand(m, tc.command)
			m, cmd := sendKey(m, tea.KeyCtrlC)
			require.Nil(t, cmd)
			require.Equal(t, ChangeUpdateState, m.state)
			require.Empty(t, m.promptValue())
			require.Equal(t, tc.field, m.detailEditField)
			for _, cancel := range []tea.KeyType{tea.KeyCtrlC, tea.KeyEsc} {
				canceled, _ := sendKey(m, cancel)
				require.Equal(t, ChangeDetailsState, canceled.state)
				require.Empty(t, canceled.detailEditField)
			}
			m, _ = sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.value)})
			m, cmd = sendKey(m, tea.KeyEnter)
			require.NotNil(t, cmd)
			require.Equal(t, tc.op, m.changeList.Operation)
			m = applyCommand(m, cmd)
			require.Empty(t, m.err)
			require.Contains(t, m.status, "saved "+string(tc.op))
			require.Equal(t, original.Spec, m.changeList.Detail.Spec)
			require.Zero(t, client.changeSpecUpdateCalls)
			if tc.op == changes.PRURL {
				require.Equal(t, []string{tc.value}, client.changePRUrlUpdates)
			}
			require.NotContains(t, client.requestOrder, "doc/insert")
		})
	}
}

func TestP403EditorReturnsToCreateFormBeforeFirstSave(t *testing.T) {
	for _, uuid := range []string{"", "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"} {
		t.Run(uuid, func(t *testing.T) {
			t.Setenv("TMPDIR", t.TempDir())
			client := &fakeClient{createdChange: dto.ChangeView{ID: "12"}, gotChange: dto.ChangeView{ID: "12"}}
			m := newChangeTestModel(client)
			m.state = ChangesListState
			m, _ = sendCommand(m, "/new-change")
			brief := "# Inferred title\n\nExact\tbrief\r\n"
			next, cmd := m.Update(editorFinishedMsg{source: ChangeCreateState, content: brief})
			m = applyCommand(next.(Model), cmd)
			require.Empty(t, client.changeCreateInputs, "editor completion must not create")
			require.Equal(t, ChangeCreateState, m.state)
			require.Equal(t, brief, m.promptValue())
			require.Equal(t, "Inferred title", m.changeList.Draft.Title)
			require.Contains(t, m.View(), "review creation fields")
			m, _ = sendKey(m, tea.KeyCtrlT)
			m = m.setPromptValue("Chosen title")
			m, _ = sendKey(m, tea.KeyEnter)
			m, _ = sendKey(m, tea.KeyCtrlU)
			m = m.setPromptValue(uuid)
			m, _ = sendKey(m, tea.KeyEnter)
			// Reopening and returning from the brief editor also requires confirmation.
			next, cmd = m.Update(editorFinishedMsg{source: ChangeCreateState, original: brief, content: brief})
			m = applyCommand(next.(Model), cmd)
			require.Empty(t, client.changeCreateInputs)
			require.Equal(t, "Chosen title", m.changeList.Draft.Title)
			m, cmd = sendKey(m, tea.KeyEnter)
			m = applyCommand(m, cmd)
			require.Empty(t, m.err)
			require.Equal(t, []dto.ChangeCreateInput{{ProjectID: 7, Title: "Chosen title", Brief: brief, RefUUID: uuid}}, client.changeCreateInputs)
		})
	}
}

func TestP403CanceledCreateSubfieldDoesNotConsumeNextBrief(t *testing.T) {
	for _, key := range []tea.KeyType{tea.KeyCtrlT, tea.KeyCtrlU} {
		for _, cancel := range []string{"escape", "/cancel"} {
			t.Run(fmt.Sprintf("%v/%s", key, cancel), func(t *testing.T) {
				t.Setenv("TMPDIR", t.TempDir())
				client := &fakeClient{createdChange: dto.ChangeView{ID: "12"}, gotChange: dto.ChangeView{ID: "12"}}
				m := newChangeTestModel(client)
				m.state = ChangeCreateState
				m = m.setPromptValue("abandoned brief")
				m, _ = sendKey(m, key)
				if cancel == "escape" {
					m, _ = sendKey(m, tea.KeyEsc)
				} else {
					m, _ = sendCommand(m, cancel)
				}
				require.Equal(t, ChangesListState, m.state)
				require.Empty(t, m.detailEditField)
				m, _ = sendCommand(m, "/new-change")
				require.Empty(t, m.detailEditField)
				brief := "# Fresh title\n\nFresh\tbrief\r\n"
				next, cmd := m.Update(editorFinishedMsg{source: ChangeCreateState, content: brief})
				m = applyCommand(next.(Model), cmd)
				m, cmd = sendKey(m, tea.KeyEnter)
				m = applyCommand(m, cmd)
				require.Empty(t, m.err)
				require.Equal(t, []dto.ChangeCreateInput{{ProjectID: 7, Title: "Fresh title", Brief: brief}}, client.changeCreateInputs)
			})
		}
	}
}

func TestP404SavedLongTitleKeepsDetailViewportUsable(t *testing.T) {
	for _, refreshFails := range []bool{false, true} {
		t.Run(fmt.Sprint(refreshFails), func(t *testing.T) {
			title := strings.Repeat("long title ", 300)
			client := &fakeClient{gotChange: dto.ChangeView{ID: "12", Title: title, PRUrl: "https://example.test/pr"}}
			if refreshFails {
				client.changeGetErr = errors.New("offline")
			}
			m := newChangeTestModel(client)
			m.state = ChangeDetailsState
			m.width, m.height = 80, 24
			m.changeList.Detail = dto.ChangeView{ID: "12", Title: "Old title", PRUrl: "https://example.test/pr"}
			m, _ = sendCommand(m, "/title")
			next, cmd := m.Update(editorFinishedMsg{source: ChangeUpdateState, content: title})
			m = applyCommand(next.(Model), cmd)
			require.Equal(t, []string{title}, client.changeTitleUpdates)
			require.Equal(t, title, m.changeList.Detail.Title)
			require.Contains(t, m.status, "saved title")
			if refreshFails {
				require.Contains(t, m.status, "/retry reads only")
			}
			var seen strings.Builder
			for i := 0; i < 100; i++ {
				view := m.View()
				require.LessOrEqual(t, lipgloss.Height(view), m.height)
				seen.WriteString(stripANSI(view))
				m, _ = sendKey(m, tea.KeyPgDown)
			}
			require.Contains(t, seen.String(), "Title")
			require.Contains(t, seen.String(), "https://example.test/pr")
			require.Contains(t, seen.String(), "Modified")
		})
	}
}

func TestP403ExplicitCreateFieldsRetainRawBrief(t *testing.T) {
	client := &fakeClient{createdChange: dto.ChangeView{ID: "12"}, gotChange: dto.ChangeView{ID: "12"}}
	m := newChangeTestModel(client)
	m.state = ChangeCreateState
	brief := "plain\tbrief\r\n" + strings.Repeat("body\n", 1000)
	m = m.setPromptValue(brief)
	m.editorDraft = &brief
	m, _ = sendKey(m, tea.KeyCtrlT)
	m = m.setPromptValue("Explicit title")
	m, cmd := sendKey(m, tea.KeyEnter)
	require.Nil(t, cmd)
	require.Equal(t, brief, m.promptValue())
	m, _ = sendKey(m, tea.KeyCtrlU)
	m = m.setPromptValue("0198a86f-9b8a-7d89-ae5b-6f25b528b04c")
	m, _ = sendKey(m, tea.KeyEnter)
	require.Equal(t, brief, m.promptValue())
	m, cmd = sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	m = applyMsg(m, cmd())
	require.Empty(t, m.err)
	require.Len(t, client.changeCreateInputs, 1)
	require.Equal(t, dto.ChangeCreateInput{ProjectID: 7, Title: "Explicit title", Brief: brief, RefUUID: "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"}, client.changeCreateInputs[0])
}

func TestP404ChangeEmptyFindAndObsoleteResults(t *testing.T) {
	for _, state := range []State{ChangesListState, ChangeDetailsState} {
		for _, query := range []string{"", "  "} {
			t.Run(string(state)+query, func(t *testing.T) {
				client := &fakeClient{changeRows: []dto.ChangeView{{ID: "12", Title: "Fresh"}}, gotChange: dto.ChangeView{ID: "12", Title: "Fresh"}}
				m := newChangeTestModel(client)
				m.changeList.Detail = dto.ChangeView{ID: "12"}
				next, cmd := m.arrive(state, "read")
				m = next.(Model)
				pending := cmd()
				m, _ = sendCommand(m, "/find")
				require.Equal(t, FindInputState, m.state)
				m = applyMsg(m, pending)
				require.Equal(t, FindInputState, m.state)
				m = m.setPromptValue(query)
				m, cmd = sendKey(m, tea.KeyEnter)
				require.NotNil(t, cmd)
				require.Equal(t, state, m.state)
				m = applyMsg(m, cmd())
				require.Contains(t, m.View(), "Fresh")
				before := m.status
				m = applyMsg(m, pending)
				require.Equal(t, before, m.status)
			})
		}
	}
}

func TestP404ShortDetailShellKeepsAllFieldsReachable(t *testing.T) {
	for _, feedback := range []string{"", strings.Repeat("wrapped error ", 20)} {
		m := newChangeTestModel(&fakeClient{})
		m.state = ChangeDetailsState
		m.err = feedback
		m.width, m.height = 80, 1000
		m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", RefUUID: "server-uuid", Ref: "3", Title: "visible title", PRUrl: "https://example.test/pr"})
		lines, _ := m.viewLines()
		shellHeight := m.height - m.epicViewportHeight(lines)
		for _, height := range []int{3, 4, 12, 3} {
			m = applyMsg(m, tea.WindowSizeMsg{Width: 80, Height: shellHeight + height})
			for i := 0; i < 100; i++ {
				m, _ = sendKey(m, tea.KeyPgUp)
			}
			var seen strings.Builder
			for i := 0; i < 100; i++ {
				view := m.View()
				require.LessOrEqual(t, lipgloss.Height(view), m.height)
				seen.WriteString(stripANSI(view))
				m, _ = sendKey(m, tea.KeyPgDown)
			}
			for _, value := range []string{"ID │ 12", "server-uuid", "Ref │ 3", "visible title", "https://example.test/pr", "Modified"} {
				require.Contains(t, seen.String(), value)
			}
		}
	}
}

func TestP404ChangeViewportsFitAndExposeEveryField(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {60, 16}, {120, 40}} {
		m := newChangeTestModel(&fakeClient{})
		m.state = ChangeDetailsState
		m.width = size[0]
		m.height = size[1]
		var title strings.Builder
		for i := 0; i < 40; i++ {
			fmt.Fprintf(&title, "title line %02d\n", i)
		}
		m.changeList.Detail = dto.ChangeView{ID: "12", ProjectID: "7", RefUUID: "uuid", Ref: "null", Slug: "server-slug", EpicID: "null", EpicName: "null", AfterChangeID: "4", Title: title.String(), PRUrl: "https://example.test/pr", Completed: 73, Done: 2, Total: 9, Created: "created", Modified: "modified"}
		var seen strings.Builder
		for i := 0; i < 100; i++ {
			v := m.View()
			require.LessOrEqual(t, lipgloss.Height(v), m.height)
			seen.WriteString(stripANSI(v))
			m, _ = sendKey(m, tea.KeyPgDown)
		}
		for i := 0; i < 40; i++ {
			require.Contains(t, seen.String(), fmt.Sprintf("title line %02d", i))
		}
		for _, value := range []string{"server-slug", "Project ID", "Epic ID", "After change", "https://example.test/pr", "73%", "Created", "Modified"} {
			require.Contains(t, seen.String(), value)
		}
	}
	m := newChangeTestModel(&fakeClient{})
	m.state = ChangesListState
	m.width = 80
	m.height = 20
	for i := 0; i < 40; i++ {
		m.changeList.Rows = append(m.changeList.Rows, dto.ChangeView{ID: fmt.Sprint(i + 1), Title: fmt.Sprintf("row %02d", i)})
	}
	for i := 0; i < 40; i++ {
		require.LessOrEqual(t, lipgloss.Height(m.View()), m.height)
		require.Contains(t, stripANSI(m.View()), fmt.Sprintf("row %02d", m.changeList.Selected))
		m, _ = sendKey(m, tea.KeyDown)
	}
}

func TestP404ChangeLongRenderingAndLiteralTitleNoOp(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	m.state = ChangeDetailsState
	m.changeList.Detail = dto.ChangeView{ID: "12", Title: strings.Repeat("a", 50000)}
	started := time.Now()
	_ = m.View()
	require.Less(t, time.Since(started), 2*time.Second)
	for _, title := range []string{"/save", "/cancel", "/return", "/editor"} {
		m.changeList.Detail.Title = title
		m, cmd := sendCommand(m, "/title")
		require.Nil(t, cmd)
		require.NotNil(t, m.editorDraft)
		m, cmd = sendKey(m, tea.KeyEnter)
		require.Nil(t, cmd)
		require.Equal(t, "unchanged", m.status)
		require.Equal(t, ChangeDetailsState, m.state)
	}
}

func TestP406ConfiguredDocumentSelectorAndNoCatalog(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	m.state = ChangeDetailsState
	m.changeList.Detail = dto.ChangeView{ID: "12", Documents: []dto.Document{{DocType: "notes", Body: "bytes"}}}
	m.optionCatalog.config.ChangeDocs = []string{"notes", "brief"}
	m, cmd := sendCommand(m, "/document")
	m = applyMsg(m, cmd())
	require.Equal(t, []dto.Option{{ID: "notes", Label: "notes"}, {ID: "brief", Label: "brief"}}, m.dropdown.options)
	m.optionCatalog.config.ChangeDocs = nil
	m.state = ChangeDetailsState
	m, cmd = sendCommand(m, "/document")
	m = applyMsg(m, cmd())
	require.Contains(t, m.err, "no options")
}

func TestP404ChangeReadCannotOverwriteDraftOrSelectedProject(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	m.changeList.Detail = dto.ChangeView{ID: "12", Title: "/save"}
	m.changeList.DetailLoaded = true
	m.state = ChangeDetailsState
	next, cmd := m.beginChange(changes.Details, 12, changes.Input{})
	m = next.(Model)
	pending := cmd()
	next, cmd = m.beginChange(changes.Details, 12, changes.Input{})
	m = applyMsg(next.(Model), cmd())
	m, cmd = sendCommand(m, "/title")
	require.Nil(t, cmd)
	before := m.promptValue()
	m = applyMsg(m, pending)
	require.Equal(t, before, m.promptValue())
	require.Equal(t, ChangeUpdateState, m.state)
	m.currentProject.ID = "8"
	m.changeList = m.changeList.Scope(8)
	m = applyMsg(m, pending)
	require.Empty(t, m.changeList.Detail.ID)
}
