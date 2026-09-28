package app

import (
	"cli/internal/dto"
	"cli/internal/epics"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type epicAppClient struct {
	fakeClient
	names   []string
	deletes int
	detail  dto.Epic
	read    func(context.Context, int) (dto.Epic, error)
	list    func(context.Context, int) ([]dto.Epic, error)
}

func (f *epicAppClient) ListEpics(ctx context.Context, id int) ([]dto.Epic, error) {
	if f.list != nil {
		return f.list(ctx, id)
	}
	return f.fakeClient.ListEpics(ctx, id)
}

func (f *epicAppClient) CreateEpic(_ context.Context, p int, name string) (int, error) {
	f.names = append(f.names, name)
	if f.createErr == nil {
		f.detail = dto.Epic{ID: 3, ProjectID: p, Name: name}
	}
	return 3, f.createErr
}

func (f *epicAppClient) UpdateEpic(_ context.Context, _ int, name string) error {
	f.names = append(f.names, name)
	if f.updateErr == nil {
		f.detail.Name = name
	}
	return f.updateErr
}

func (f *epicAppClient) GetEpic(ctx context.Context, id int) (dto.Epic, error) {
	if f.read != nil {
		return f.read(ctx, id)
	}
	return f.detail, f.getErr
}
func (f *epicAppClient) DeleteEpic(context.Context, int) error { f.deletes++; return f.err }

func epicApp(f *epicAppClient) Model {
	m := NewModelWithClient(f)
	m.currentProject = dto.Option{ID: "7", Label: "Seven"}
	m.appConfig.ProjectID = 7
	m.epicList = epics.Model{ProjectID: 7, Detail: f.detail, DetailLoaded: true}
	return m
}

func TestP303EpicEditorRawDraftRetryCancelAndNoOp(t *testing.T) {
	for _, state := range []State{EpicCreateState, EpicUpdateState} {
		for _, raw := range []string{"/cancel", "/api/v1/health returns 200", "\t" + strings.Repeat("long\n", 10001)} {
			t.Run(string(state)+raw[:min(len(raw), 20)], func(t *testing.T) {
				f := &epicAppClient{fakeClient: fakeClient{createErr: errors.New("offline"), updateErr: errors.New("offline")}, detail: dto.Epic{ID: 3, ProjectID: 7, Name: "before"}}
				m := epicApp(f)
				m.state = state
				m.changeDetailLoaded = true
				next, cmd := m.Update(editorFinishedMsg{source: state, content: raw})
				m = applyCommand(next.(Model), cmd)
				assert.Equal(t, "save failed", m.status)
				assert.Equal(t, raw, m.promptValue())
				assert.Equal(t, raw, m.epicList.Draft)
				next, cmd = m.Update(editorFinishedMsg{source: state, original: raw, content: raw})
				m = applyCommand(next.(Model), cmd)
				assert.Equal(t, "save failed", m.status)
				assert.Equal(t, raw, m.promptValue())
				f.createErr = nil
				f.updateErr = nil
				m, cmd = sendKey(m, tea.KeyEnter)
				m = applyCommand(m, cmd)
				require.Equal(t, EpicDetailsState, m.state)
				assert.Equal(t, "saved epic", m.status)
				assert.Nil(t, m.editorDraft)
				assert.Equal(t, []string{raw, raw, raw}, f.names)
				next, cmd = m.executeCommand("/edit")
				require.Nil(t, cmd)
				m = next.(Model)
				assert.Equal(t, raw, m.promptValue())
				next, cmd = m.Update(editorFinishedMsg{source: EpicUpdateState, original: raw, content: raw})
				m = applyCommand(next.(Model), cmd)
				assert.Equal(t, "unchanged", m.status)
				assert.Len(t, f.names, 3)
				next, _ = m.handleListSelection()
				m = next.(Model)
				next, cmd = m.executeCommand("/cancel")
				m = applyCommand(next.(Model), cmd)
				assert.Equal(t, EpicDetailsState, m.state)
				assert.Empty(t, m.promptValue())
				assert.Len(t, f.names, 3)
			})
		}
	}
}

func TestP302EpicKeyboardCRUDHelpConfirmationAndScope(t *testing.T) {
	f := &epicAppClient{fakeClient: fakeClient{epics: []dto.Option{{ID: "3", Label: "Epic"}}}, detail: dto.Epic{ID: 3, ProjectID: 7, Name: "Epic", Completed: 63}}
	m := epicApp(f)
	next, cmd := m.executeCommand("/epics")
	m = applyCommand(next.(Model), cmd)
	assert.Contains(t, m.View(), "Epic")
	assert.Equal(t, "loaded epics", m.status)
	m, _ = sendKey(m, tea.KeyDown)
	m, _ = sendKey(m, tea.KeyUp)
	m, cmd = sendKey(m, tea.KeyEnter)
	m = applyCommand(m, cmd)
	assert.Equal(t, EpicDetailsState, m.state)
	assert.Contains(t, m.View(), "Completed: 63")
	for _, key := range []tea.KeyType{tea.KeyEsc, tea.KeyCtrlC, tea.KeyEnter} {
		next, cmd = m.executeCommand("/delete")
		require.Nil(t, cmd)
		m = next.(Model)
		if key == tea.KeyEnter {
			m, _ = sendKey(m, tea.KeyDown)
		}
		m, cmd = sendKey(m, key)
		assert.Nil(t, cmd)
		assert.Equal(t, EpicDetailsState, m.state)
		assert.Zero(t, f.deletes)
	}
	next, _ = m.executeCommand("/delete")
	m = next.(Model)
	m, cmd = sendKey(m, tea.KeyEnter)
	m = applyCommand(m, cmd)
	assert.Equal(t, 1, f.deletes)
	assert.Equal(t, EpicsListState, m.state)
	assert.Equal(t, "deleted epic", m.status)
	next, cmd = m.executeCommand("/help")
	m = applyCommand(next.(Model), cmd)
	assert.Contains(t, m.View(), "/new-epic")
	assert.Contains(t, m.View(), "/delete asks for confirmation")
	m = NewModelWithClient(f)
	next, cmd = m.executeCommand("/epics")
	require.Nil(t, cmd)
	m = next.(Model)
	assert.Contains(t, m.err, "/select-project")
	next, cmd = m.executeCommand("/new-epic")
	require.Nil(t, cmd)
	assert.Contains(t, next.(Model).err, "/select-project")
	m = epicApp(f)
	m.state = EpicDetailsState
	m.epicList.DetailLoaded = false
	next, cmd = m.executeCommand("/edit")
	require.Nil(t, cmd)
	assert.Contains(t, next.(Model).err, "/retry")
}

func TestP304EpicObsoleteReadCannotChangeShellOrDraft(t *testing.T) {
	for _, command := range []string{"/return", "/help", "/new-epic"} {
		t.Run(command, func(t *testing.T) {
			f := &epicAppClient{detail: dto.Epic{ID: 3, ProjectID: 7, Name: "old"}}
			f.read = func(ctx context.Context, _ int) (dto.Epic, error) {
				require.ErrorIs(t, ctx.Err(), context.Canceled)
				return dto.Epic{ID: 3, ProjectID: 7, Name: "stale"}, errors.New("stale error")
			}
			m := epicApp(f)
			m.state = EpicsListState
			next, held := m.beginEpic(epics.Details, 3, "")
			m = next.(Model)
			next, cmd := m.executeCommand(command)
			m = applyCommand(next.(Model), cmd)
			m = m.setPromptValue("new draft")
			state, status := m.state, m.status
			m = applyMsg(m, held())
			assert.Equal(t, state, m.state)
			assert.Equal(t, status, m.status)
			assert.Equal(t, "new draft", m.promptValue())
			assert.NotContains(t, m.err, "stale")
		})
	}
	f := &epicAppClient{}
	m := epicApp(f)
	next, held := m.beginEpic(epics.List, 0, "")
	m = next.(Model)
	m.currentProject = dto.Option{ID: "8"}
	m.epicList = m.epicList.Scope(8)
	m = applyMsg(m, held())
	assert.Empty(t, m.epicList.Rows)
	assert.Equal(t, 8, m.epicList.ProjectID)
}

func TestP304EpicEmptyFindRestartsCanceledRead(t *testing.T) {
	for _, state := range []State{EpicsListState, EpicDetailsState} {
		for _, query := range []string{"", " \t "} {
			t.Run(string(state)+"/"+query, func(t *testing.T) {
				f := &epicAppClient{detail: dto.Epic{ID: 3, ProjectID: 7, Name: "Fresh epic"}}
				var contexts []context.Context
				f.read = func(ctx context.Context, id int) (dto.Epic, error) {
					contexts = append(contexts, ctx)
					assert.Equal(t, 3, id)
					assert.NoError(t, ctx.Err())
					return f.detail, nil
				}
				f.list = func(ctx context.Context, id int) ([]dto.Epic, error) {
					contexts = append(contexts, ctx)
					assert.Equal(t, 7, id)
					assert.NoError(t, ctx.Err())
					return []dto.Epic{f.detail}, nil
				}
				m := epicApp(f)
				next, cmd := m.arrive(state, "")
				m = next.(Model)
				// Hold the response until after navigation cancels its request.
				stale := cmd().(epics.Result)
				m, _ = sendCommand(m, "/find")
				require.Equal(t, FindInputState, m.state)
				assert.ErrorIs(t, contexts[0].Err(), context.Canceled)
				m = m.setPromptValue(query)
				m, cmd = sendKey(m, tea.KeyEnter)
				require.NotNil(t, cmd, "empty find must restart the canceled read")
				assert.Equal(t, state, m.state)
				assert.Equal(t, "find text is required", m.err)
				assert.True(t, m.epicList.Loading)
				stale.Err = errors.New("obsolete read failed")
				m = applyMsg(m, stale)
				assert.True(t, m.epicList.Loading)
				assert.Equal(t, "find text is required", m.err)
				m = applyCommand(m, cmd)
				assert.Len(t, contexts, 2)
				assert.Equal(t, state, m.state)
				assert.False(t, m.epicList.Loading)
				assert.Empty(t, m.err)
				if state == EpicsListState {
					assert.Equal(t, []dto.Epic{f.detail}, m.epicList.Rows)
					assert.Equal(t, "loaded epics", m.status)
				} else {
					assert.True(t, m.epicList.DetailLoaded)
					assert.Equal(t, f.detail, m.epicList.Detail)
					assert.Equal(t, "loaded epic", m.status)
				}
				assert.Contains(t, m.View(), "Fresh epic")
				m = applyMsg(m, stale)
				assert.Empty(t, m.err)
				assert.Contains(t, m.View(), "Fresh epic")
			})
		}
	}
}

func TestP305EpicSelectorUsesTypedValuesAndObsoleteScope(t *testing.T) {
	f := &fakeClient{epics: []dto.Option{{ID: "3", Label: "Name"}}}
	m := NewModelWithClient(f)
	m.currentProject = dto.Option{ID: "7"}
	m.state = ChangesListState
	next, cmd := m.executeCommand("/epic-filter")
	m = next.(Model)
	result := cmd()
	assert.Equal(t, 1, f.epicCalls)
	assert.Equal(t, "7", f.projectID)
	m = applyMsg(m, result)
	assert.Equal(t, "3", m.dropdown.options[0].ID)
	m.currentProject = dto.Option{ID: "8"}
	m.dropdown.options = nil
	m = applyMsg(m, result)
	assert.Empty(t, m.dropdown.options)
}

func TestP304EpicSelectorReopenRejectsOlderSameProjectResult(t *testing.T) {
	for _, state := range []State{ChangesListState, ChangeDetailsState} {
		for _, lateErr := range []error{nil, errors.New("obsolete failure")} {
			name := string(state) + "/success"
			if lateErr != nil {
				name = string(state) + "/error"
			}
			t.Run(name, func(t *testing.T) {
				f := &fakeClient{epics: []dto.Option{{ID: "3", Label: "Old epic"}}}
				m := NewModelWithClient(f)
				m.currentProject = dto.Option{ID: "7"}
				m.state = state
				m.changeDetailLoaded = true
				m.changeList.Detail = dto.ChangeView{ID: "12"}
				command := "/epic-filter"
				if state == ChangeDetailsState {
					command = "/epic"
				}
				next, cmd := m.executeCommand(command)
				m = next.(Model)
				older := cmd().(selectorLoadedMsg)
				older.err = lateErr
				m, _ = sendKey(m, tea.KeyEsc)
				f.epics = []dto.Option{{ID: "4", Label: "New epic"}}
				m.changeList.Detail.ID = "13"
				next, cmd = m.executeCommand(command)
				m = applyCommand(next.(Model), cmd)
				m = applyMsg(m, older)
				require.NotEmpty(t, m.dropdown.options)
				assert.Equal(t, "4", m.dropdown.options[0].ID)
				assert.Empty(t, m.err)
				assert.False(t, m.dropdown.loading)
			})
		}
	}
}

func TestP303LoadedEpicNamesRemainLiteralOnEnter(t *testing.T) {
	for _, name := range []string{"/save", "/cancel", "/editor", "/return"} {
		t.Run(name, func(t *testing.T) {
			f := &epicAppClient{detail: dto.Epic{ID: 3, ProjectID: 7, Name: name}}
			m := epicApp(f)
			m.state = EpicDetailsState
			next, cmd := m.executeCommand("/edit")
			require.Nil(t, cmd)
			m, cmd = sendKey(next.(Model), tea.KeyEnter)
			require.Nil(t, cmd)
			assert.Equal(t, EpicDetailsState, m.state)
			assert.Equal(t, "unchanged", m.status)
			assert.Empty(t, m.err)
			assert.Empty(t, f.names)
			next, _ = m.executeCommand("/edit")
			m = next.(Model).insertPromptLiteral(" changed")
			m, cmd = sendKey(m, tea.KeyEnter)
			m = applyCommand(m, cmd)
			assert.Equal(t, EpicDetailsState, m.state)
			assert.Equal(t, []string{name + " changed"}, f.names)
		})
	}
}

func TestP302EpicListFitsTerminalWhileMovingAndResizing(t *testing.T) {
	f := &epicAppClient{}
	m := epicApp(f)
	m.state = EpicsListState
	for i := 1; i <= 40; i++ {
		m.epicList.Rows = append(m.epicList.Rows, dto.Epic{ID: i, ProjectID: 7, Name: fmt.Sprintf("Epic-%02d", i)})
	}
	for _, size := range []tea.WindowSizeMsg{{Width: 100, Height: 24}, {Width: 60, Height: 16}, {Width: 120, Height: 40}} {
		m = applyMsg(m, size)
		m.err = "A read failed; retry is available"
		for _, key := range []tea.KeyType{tea.KeyDown, tea.KeyUp} {
			for range 40 {
				view := m.View()
				assert.LessOrEqual(t, lipgloss.Height(view), size.Height)
				assert.Contains(t, view, m.epicList.Rows[m.epicList.Selected].Name)
				m, _ = sendKey(m, key)
			}
		}
	}
	f.detail = m.epicList.Rows[m.epicList.Selected]
	m, cmd := sendKey(m, tea.KeyEnter)
	m = applyCommand(m, cmd)
	assert.Equal(t, EpicDetailsState, m.state)
	assert.Equal(t, f.detail.ID, m.epicList.Detail.ID)
}

func TestP302EpicDetailsFitsTerminalAndScrollsEveryField(t *testing.T) {
	f := &epicAppClient{detail: dto.Epic{ID: 3, ProjectID: 7}}
	for i := range 40 {
		f.detail.Name += fmt.Sprintf("name-line-%02d\n", i)
	}
	m := epicApp(f)
	m.state = EpicDetailsState
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 60, Height: 16}, {Width: 120, Height: 40}} {
		m = applyMsg(m, size)
		m.epicList.DetailOffset = 0
		assert.Contains(t, m.View(), "ID: 3")
		assert.Contains(t, m.View(), "Project ID: 7")
		assert.Contains(t, m.View(), "Name: name-line-00")
		seen := ""
		m.err = "Read failed; use /retry to try again"
		for range 55 {
			view := m.View()
			require.LessOrEqual(t, lipgloss.Height(view), size.Height)
			seen += view
			m, _ = sendKey(m, tea.KeyDown)
		}
		for i := range 40 {
			assert.Contains(t, seen, fmt.Sprintf("name-line-%02d", i))
		}
		for _, label := range []string{"Done TC:", "Total TC:", "Completed:", "Changes:", "Created:", "Modified:"} {
			assert.Contains(t, seen, label)
		}
		for range 55 {
			m, _ = sendKey(m, tea.KeyUp)
		}
		assert.Zero(t, m.epicList.DetailOffset)
		m, _ = sendKey(m, tea.KeyPgDown)
		assert.Positive(t, m.epicList.DetailOffset)
		m, _ = sendKey(m, tea.KeyPgUp)
		assert.Zero(t, m.epicList.DetailOffset)
	}
	m = m.setPromptValue("draft")
	m, _ = sendKey(m, tea.KeyDown)
	assert.Zero(t, m.epicList.DetailOffset, "prompt input retains focus")
}
