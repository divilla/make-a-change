package app

import (
	"cli/internal/changes"
	"cli/internal/dto"
	"cli/internal/styles"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func TestChangesListPromptFiltersAsTypedAndCanSelectResult(t *testing.T) {
	client := &fakeClient{gotChange: dto.ChangeView{ID: "2", Title: "Add DB Migrations"}}
	m := newChangeTestModel(client)
	m.state = ChangesListState
	m.changeList = m.changeList.WithRows([]dto.ChangeView{
		{ID: "1", Ref: "1", Title: "Other", ChangeTypes: []string{"feature"}},
		{ID: "2", Ref: "2", Title: "Add DB Migrations"},
	})

	for _, r := range "AD mig" {
		m, _ = sendRune(m, r)
	}
	require.Equal(t, "AD mig", m.input.Value())
	require.Equal(t, "AD mig", m.changeFilters().Find)
	require.Empty(t, m.changesFilters.find)
	require.Equal(t, "Add DB Migrations", changes.FilteredRows(m.changeList.Rows, m.changeFilters())[0].Title)
	require.Len(t, changes.FilteredRows(m.changeList.Rows, m.changeFilters()), 1)
	m, _ = sendRune(m, 'x')
	require.Empty(t, changes.FilteredRows(m.changeList.Rows, m.changeFilters()))
	require.Contains(t, m.View(), "No changes match filters.")
	require.Contains(t, m.View(), "Rows 0-0 of 0")
	m, _ = sendKey(m, tea.KeyBackspace)
	require.Len(t, changes.FilteredRows(m.changeList.Rows, m.changeFilters()), 1)

	m, _ = sendKey(m, tea.KeyBackspace)
	require.Equal(t, "AD mi", m.input.Value())
	require.Len(t, changes.FilteredRows(m.changeList.Rows, m.changeFilters()), 1)
	m, _ = sendKey(m, tea.KeyCtrlC)
	require.Empty(t, m.input.Value())
	require.Empty(t, m.changesFilters.find)
	require.Len(t, changes.FilteredRows(m.changeList.Rows, m.changeFilters()), 2)

	for _, r := range "ad mig" {
		m, _ = sendRune(m, r)
	}
	m, cmd := sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	require.Equal(t, ChangeDetailsState, m.state)
	require.Empty(t, m.input.Value())
	require.Empty(t, m.changeFilters().Find)
	require.Empty(t, m.changesFilters.find)
	require.Equal(t, "2", m.changeSelectionID)
	m = applyMsg(m, cmd())
	require.Equal(t, []int{2}, client.changeGetIDs)
}

func TestFindFilterAndSelectedChangeSurviveDetailAndMainNavigation(t *testing.T) {
	rows := []dto.ChangeView{
		{ID: "1", Title: "Add DB Migrations", ChangePhase: "backlog", EpicID: "5", EpicName: "Project", ChangeTypes: []string{"feature"}},
		{ID: "2", Title: "Other work", ChangePhase: "done", EpicID: "6", EpicName: "Other", ChangeTypes: []string{"fix"}},
		{ID: "3", Title: "Add Storage Migrations", ChangePhase: "backlog", EpicID: "5", EpicName: "Project", ChangeTypes: []string{"feature"}},
	}
	client := &fakeClient{changeRows: rows, gotChange: dto.ChangeView{Title: "Add Storage Migrations"}}
	m := newChangeTestModel(client)
	m.state = ChangesListState
	m.changeList = m.changeList.WithRows(rows)
	m.changesFilters.phase = dto.Option{ID: "backlog", Label: "backlog"}
	m.changesFilters.typ = dto.Option{ID: "feature", Label: "feature"}
	m.changesFilters.epic = dto.Option{ID: "5", Label: "Project"}

	m, _ = sendCommand(m, "/find-filter")
	require.Equal(t, FindInputState, m.state)
	m = m.setPromptValue("AD mig")
	m, _ = sendKey(m, tea.KeyEnter)
	require.Equal(t, ChangesListState, m.state)
	require.Equal(t, "AD mig", m.changesFilters.find)
	require.Equal(t, []string{"1", "3"}, []string{changes.FilteredRows(m.changeList.Rows, m.changeFilters())[0].ID, changes.FilteredRows(m.changeList.Rows, m.changeFilters())[1].ID})
	require.Contains(t, stripANSI(m.View()), "AD mig")

	m, _ = sendKey(m, tea.KeyDown)
	require.Equal(t, 1, m.changeList.Selected)
	m, cmd := sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	m = applyMsg(m, cmd())
	require.Equal(t, ChangeDetailsState, m.state)
	require.Equal(t, "3", m.changeSelectionID)

	m, cmd = sendCommand(m, "/return")
	require.NotNil(t, cmd)
	m = applyMsg(m, cmd())
	require.Equal(t, ChangesListState, m.state)
	require.Equal(t, "AD mig", m.changesFilters.find)
	require.Equal(t, "backlog", m.changesFilters.phase.ID)
	require.Equal(t, "feature", m.changesFilters.typ.ID)
	require.Equal(t, "5", m.changesFilters.epic.ID)
	require.Equal(t, 1, m.changeList.Selected)
	require.Equal(t, "3", changes.FilteredRows(m.changeList.Rows, m.changeFilters())[m.changeList.Selected].ID)

	m, _ = sendCommand(m, "/return")
	require.Equal(t, MainState, m.state)
	m, cmd = sendCommand(m, "/changes")
	require.NotNil(t, cmd)
	m = applyMsg(m, cmd())
	require.Equal(t, ChangesListState, m.state)
	require.Equal(t, "AD mig", m.changesFilters.find)
	require.Equal(t, "backlog", m.changesFilters.phase.ID)
	require.Equal(t, "feature", m.changesFilters.typ.ID)
	require.Equal(t, "5", m.changesFilters.epic.ID)
	require.Equal(t, 1, m.changeList.Selected)
	require.Equal(t, "3", changes.FilteredRows(m.changeList.Rows, m.changeFilters())[m.changeList.Selected].ID)

	m, _ = sendCommand(m, "/find-filter")
	m = m.setPromptValue("")
	m, _ = sendKey(m, tea.KeyEnter)
	require.Equal(t, "AD mig", m.changesFilters.find)
	require.Equal(t, "backlog", m.changesFilters.phase.ID)
	require.Equal(t, "feature", m.changesFilters.typ.ID)
	require.Equal(t, "5", m.changesFilters.epic.ID)
	m, _ = sendCommand(m, "/clear-filters")
	require.Empty(t, m.changesFilters.find)
	require.Empty(t, m.changesFilters.phase.ID)
	require.Empty(t, m.changesFilters.typ.ID)
	require.Empty(t, m.changesFilters.epic.ID)
	require.Equal(t, 2, m.changeList.Selected)
}

func TestChangesListPromptQueryDoesNotReplaceSavedFindFilter(t *testing.T) {
	rows := []dto.ChangeView{
		{ID: "1", Title: "Add DB Migrations", ChangeTypes: []string{"feature"}},
		{ID: "2", Title: "Add DB Migrations", ChangeTypes: []string{"fix"}},
	}
	m := newChangeTestModel(&fakeClient{changeRows: rows})
	m.state = ChangesListState
	m.changeList = m.changeList.WithRows(rows)
	m.changesFilters.find = "ad mig"
	for _, r := range "FEA" {
		m, _ = sendRune(m, r)
	}
	require.Equal(t, "ad mig FEA", m.changeFilters().Find)
	require.Len(t, changes.FilteredRows(m.changeList.Rows, m.changeFilters()), 1)
	require.Equal(t, "ad mig", m.changesFilters.find)

	next, _ := m.arrive(MainState, "return")
	m = next.(Model)
	require.Empty(t, m.input.Value())
	require.Equal(t, "ad mig", m.changesFilters.find)
	next, cmd := m.arrive(ChangesListState, "return")
	m = next.(Model)
	require.NotNil(t, cmd)
	m = applyMsg(m, cmd())
	require.Equal(t, "ad mig", m.changeFilters().Find)
	require.Len(t, changes.FilteredRows(m.changeList.Rows, m.changeFilters()), 2)
}

func TestSelectorClearBroadensOnlyItsOwnFilter(t *testing.T) {
	rows := []dto.ChangeView{
		{ID: "1", Title: "One", ChangePhase: "done", ChangeTypes: []string{"test"}, EpicID: "1", EpicName: "First"},
		{ID: "2", Title: "Two", ChangePhase: "backlog", ChangeTypes: []string{"test"}, EpicID: "1", EpicName: "First"},
		{ID: "3", Title: "Three", ChangePhase: "done", ChangeTypes: []string{"feature"}, EpicID: "1", EpicName: "First"},
		{ID: "4", Title: "Four", ChangePhase: "done", ChangeTypes: []string{"test"}, EpicID: "2", EpicName: "Second"},
	}
	client := &fakeClient{
		phases: []dto.Option{{ID: "done", Label: "done"}},
		types:  []dto.Option{{ID: "test", Label: "test"}},
		epics:  []dto.Option{{ID: "1", Label: "First"}},
	}
	m := newChangeTestModel(client)
	m.state = ChangesListState
	m.changeList = m.changeList.WithRows(rows)
	m.changesFilters = changesFilters{
		phase: dto.Option{ID: "done", Label: "done"},
		typ:   dto.Option{ID: "test", Label: "test"},
		epic:  dto.Option{ID: "1", Label: "First"},
	}
	require.Len(t, changes.FilteredRows(rows, m.changeFilters()), 1)
	for _, tc := range []struct {
		command string
		count   int
	}{
		{command: "/phase-filter", count: 2},
		{command: "/types-filter", count: 3},
		{command: "/epic-filter", count: 4},
	} {
		var cmd tea.Cmd
		m, cmd = sendCommand(m, tc.command)
		require.NotNil(t, cmd)
		m = applyMsg(m, cmd())
		for _, letter := range "@clear" {
			m, _ = sendRune(m, letter)
		}
		m, _ = sendKey(m, tea.KeyEnter)
		require.Len(t, changes.FilteredRows(rows, m.changeFilters()), tc.count, tc.command)
	}
}

func TestChangeSlugPromptKeepsPrefixAndRejectsInvalidCharacters(t *testing.T) {
	change := dto.ChangeView{ID: "12", ProjectID: "7", Ref: "111", RefSlug: "111-old-slug", Title: "Change"}
	client := &fakeClient{gotChange: change}
	m := newChangeTestModel(client)
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(change)
	m.changeList.DetailSelected = 0

	m, cmd := sendKey(m, tea.KeyEnter)
	require.Nil(t, cmd)
	require.Equal(t, detailEditSlug, m.detailEditField)
	require.Equal(t, "old-slug", m.input.Value())
	require.Equal(t, "111-", m.slugPrefix)
	require.Contains(t, stripANSI(m.View()), "Slug > 111-old-slug")

	for _, key := range []tea.KeyType{tea.KeySpace, tea.KeyCtrlE} {
		m, cmd = sendKey(m, key)
		require.Nil(t, cmd)
	}
	m, cmd = sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("A[] .é")})
	require.Nil(t, cmd)
	require.Equal(t, "old-slug", m.input.Value())
	m, cmd = sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	require.Nil(t, cmd)
	require.Equal(t, "old-slug", m.input.Value())

	canceled, cmd := sendKey(m, tea.KeyEsc)
	require.Nil(t, cmd)
	require.Equal(t, ChangeDetailsState, canceled.state)
	require.Empty(t, canceled.input.Value())
	require.Empty(t, canceled.detailEditField)
	require.Equal(t, "111-old-slug", canceled.changeList.Detail.RefSlug)

	for range "old-slug" {
		m, _ = sendKey(m, tea.KeyBackspace)
	}
	for _, r := range "new-slug_2" {
		m, _ = sendRune(m, r)
	}
	require.Equal(t, "new-slug_2", m.input.Value())
	m, cmd = sendKey(m, tea.KeyEnter)
	require.NotNil(t, cmd)
	m = applyMsg(m, cmd())
	require.Equal(t, "111-new-slug_2", client.gotChange.RefSlug)
	require.Equal(t, "111-new-slug_2", m.changeList.Detail.RefSlug)
	require.Empty(t, m.detailEditField)
}

func TestChangeSlugWithoutRefSlugHasNoInventedPrefix(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", Ref: "null", RefSlug: "null", Title: "Change"})
	m.changeList.DetailSelected = 0
	m, cmd := sendKey(m, tea.KeyEnter)
	require.Nil(t, cmd)
	require.Empty(t, m.slugPrefix)
	require.Equal(t, "", m.input.Value())
	require.Contains(t, stripANSI(m.View()), "Slug >")
}

func TestAfterChangeRowShowsNameAndPromptUsesAssociationID(t *testing.T) {
	for _, tc := range []struct {
		name       string
		id         string
		label      string
		wantRow    string
		wantPrompt string
	}{
		{"associated", "3", "First change #3", "After Change │ First change #3", "After Change > 3"},
		{"unassociated", "null", "null", "After Change │ -", "After Change > "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newChangeTestModel(&fakeClient{})
			m.state = ChangeDetailsState
			m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", RefUUID: "uuid", RefSlug: "006-some-slug", AfterChangeID: tc.id, AfterChangeName: tc.label, Title: "Change"})
			view := stripANSI(m.View())
			require.Less(t, strings.Index(view, "Ref UUID │ uuid"), strings.Index(view, "Slug │ 006-some-slug"))
			require.Less(t, strings.Index(view, "Slug │ 006-some-slug"), strings.Index(view, tc.wantRow))
			m.changeList.DetailSelected = 4
			active, cmd := sendKey(m, tea.KeyEnter)
			require.Nil(t, cmd)
			require.Equal(t, detailEditAfterChange, active.detailEditField)
			require.Contains(t, stripANSI(active.View()), tc.wantPrompt)
			require.Equal(t, strings.TrimPrefix(tc.wantPrompt, "After Change > "), active.input.Value())
			canceled, cmd := sendKey(active, tea.KeyEsc)
			require.Nil(t, cmd)
			require.Empty(t, canceled.input.Value())
			require.Empty(t, canceled.detailEditField)
		})
	}
}

func TestPRURLRowUsesInlinePromptAndBothCancelKeys(t *testing.T) {
	change := dto.ChangeView{ID: "12", Ref: "111", Title: "Change", PRUrl: "https://example.test/pr"}
	m := newChangeTestModel(&fakeClient{})
	m.state = ChangeDetailsState
	m.changeList = m.changeList.WithDetail(change)
	m.changeList.DetailSelected = 9
	active, cmd := sendKey(m, tea.KeyEnter)
	require.Nil(t, cmd)
	require.Equal(t, ChangeDetailsState, active.state)
	require.Equal(t, detailEditPRUrl, active.detailEditField)
	require.Contains(t, stripANSI(active.View()), "PR URL > https://example.test/pr")
	line := active.renderPromptLineWithStyle(active.input.Value(), true, styles.Default.InputBand, styles.Gray)
	require.Contains(t, line, styles.Default.InputBand.Foreground(styles.AccentPurple).Render(" PR URL > "))
	require.Contains(t, line, styles.Default.InputBand.Foreground(styles.AccentGreen).Render(change.PRUrl))
	for _, key := range []tea.KeyType{tea.KeyEsc, tea.KeyCtrlC} {
		canceled, cancelCmd := sendKey(active, key)
		require.Nil(t, cancelCmd)
		require.Equal(t, ChangeDetailsState, canceled.state)
		require.Empty(t, canceled.detailEditField)
		require.Empty(t, canceled.input.Value())
		require.Equal(t, defaultInputPlaceholder, canceled.input.Placeholder)
		previous, _ := sendKey(canceled, key)
		require.Equal(t, ChangesListState, previous.state)
	}
}

func TestTextPromptLabelsAndInputColorsAcrossForms(t *testing.T) {
	cases := []struct {
		state State
		label string
	}{
		{ProjectCreateState, "Name"},
		{EpicUpdateState, "Name"},
		{TestCaseCreateState, "Scenario"},
		{ChangeCreateState, "Brief"},
		{FindInputState, "Find"},
	}
	for _, testCase := range cases {
		t.Run(testCase.label+fmt.Sprint(testCase.state), func(t *testing.T) {
			m := newChangeTestModel(&fakeClient{})
			m.state = testCase.state
			m.input.SetValue("value")
			line := m.renderPromptLineWithStyle("value", false, styles.Default.InputBand, styles.Gray)
			require.Contains(t, line, styles.Default.InputBand.Foreground(styles.AccentPurple).Render(" "+testCase.label+" > "))
			require.Contains(t, line, styles.Default.InputBand.Foreground(styles.AccentGreen).Render("value"))
		})
	}
}

func TestChangeDetailPromptAndFooterMeetBoxWithoutBlankLine(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	m.state = ChangeDetailsState
	m.width, m.height = 80, 24
	m.changeList = m.changeList.WithDetail(dto.ChangeView{ID: "12", Ref: "111", RefSlug: "111-change", Title: "Change"})
	lines := strings.Split(stripANSI(m.View()), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "└") {
			require.Less(t, i+1, len(lines))
			require.True(t, strings.HasPrefix(lines[i+1], "▄"))
		}
		if strings.HasPrefix(line, "▀") {
			require.Less(t, i+1, len(lines))
			require.Contains(t, lines[i+1], "<ctrl+n>")
		}
	}
}

func TestP403ClearingUpdatePromptCancelsField(t *testing.T) {
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
			active := m
			m, cmd := sendKey(m, tea.KeyCtrlC)
			require.Nil(t, cmd)
			require.Equal(t, ChangeDetailsState, m.state)
			require.Empty(t, m.promptValue())
			require.Empty(t, m.detailEditField)
			for _, cancel := range []tea.KeyType{tea.KeyCtrlC, tea.KeyEsc} {
				canceled, _ := sendKey(active, cancel)
				require.Equal(t, ChangeDetailsState, canceled.state)
				require.Empty(t, canceled.detailEditField)
			}
			m, _ = sendCommand(m, tc.command)
			m = m.setPromptValue("")
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
			next, cmd := m.Update(editorFinishedMsg{source: ChangeDetailsState, content: title})
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
				command := "/find"
				if state == ChangesListState {
					command = "/find-filter"
				}
				m, _ = sendCommand(m, command)
				require.Equal(t, FindInputState, m.state)
				m = applyMsg(m, pending)
				require.Equal(t, FindInputState, m.state)
				m = m.setPromptValue(query)
				m, cmd = sendKey(m, tea.KeyEnter)
				require.Equal(t, state, m.state)
				if state == ChangesListState {
					require.NotNil(t, cmd)
					m = applyMsg(m, cmd())
				} else {
					require.NotNil(t, cmd)
					m = applyMsg(m, cmd())
				}
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
			for _, value := range []string{"ID │ 12", "server-uuid", "visible title", "https://example.test/pr", "Modified"} {
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
		m.changeList.Detail = dto.ChangeView{ID: "12", ProjectID: "7", RefUUID: "uuid", Ref: "null", RefSlug: "server-slug", EpicID: "null", EpicName: "null", AfterChangeID: "4", Title: title.String(), PRUrl: "https://example.test/pr", Completed: 73, Done: 2, Total: 9, Created: "created", Modified: "modified"}
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
		for _, value := range []string{"server-slug", "After Change", "https://example.test/pr", "73%", "Created", "Modified"} {
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
	require.Equal(t, ChangeDetailsState, m.state)
	m.currentProject.ID = "8"
	m.changeList = m.changeList.Scope(8)
	m = applyMsg(m, pending)
	require.Empty(t, m.changeList.Detail.ID)
}
