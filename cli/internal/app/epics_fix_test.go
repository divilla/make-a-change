package app

import (
	"cli/internal/changes"
	"cli/internal/dto"
	"cli/internal/styles"
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

type activity034Client struct {
	fakeClient
	changes                  []dto.Change
	epics                    []dto.Epic
	changeWrites, epicWrites []bool
	epicDeletes              []int
	readErr, writeErr        error
}

func (a *activity034Client) list(project int, active bool) ([]dto.Change, error) {
	rows := []dto.Change{}
	for _, row := range a.changes {
		if row.ProjectID == project && row.Active == active {
			rows = append(rows, row)
		}
	}
	return rows, a.readErr
}

func (a *activity034Client) ListChangeRows(_ context.Context, project int) ([]dto.Change, error) {
	return a.list(project, true)
}

func (a *activity034Client) ListInactiveChanges(_ context.Context, project int) ([]dto.Change, error) {
	return a.list(project, false)
}

func (a *activity034Client) GetChange(_ context.Context, id int) (dto.Change, error) {
	for _, row := range a.changes {
		if row.ID == id {
			return row, nil
		}
	}
	return dto.Change{}, errors.New("missing change")
}

func (a *activity034Client) UpdateChangeActive(_ context.Context, id int, active bool) error {
	a.changeWrites = append(a.changeWrites, active)
	if a.writeErr != nil {
		return a.writeErr
	}
	for i := range a.changes {
		if a.changes[i].ID == id {
			a.changes[i].Active = active
		}
	}
	return nil
}

func (a *activity034Client) ListEpics(_ context.Context, project int) ([]dto.Epic, error) {
	rows := []dto.Epic{}
	for _, row := range a.epics {
		if row.ProjectID == project {
			rows = append(rows, row)
		}
	}
	return rows, a.readErr
}

func (a *activity034Client) DeleteEpic(_ context.Context, id int) error {
	a.epicDeletes = append(a.epicDeletes, id)
	if a.writeErr != nil {
		return a.writeErr
	}
	a.epics = slices.DeleteFunc(a.epics, func(e dto.Epic) bool { return e.ID == id && e.ChangeCount == 0 })
	for i := range a.epics {
		if a.epics[i].ID == id {
			a.epics[i].Active = false
		}
	}
	return nil
}

func (a *activity034Client) UpdateEpicActive(_ context.Context, id int, active bool) error {
	a.epicWrites = append(a.epicWrites, active)
	if a.writeErr != nil {
		return a.writeErr
	}
	for i := range a.epics {
		if a.epics[i].ID == id {
			a.epics[i].Active = active
		}
	}
	return nil
}

func model034(t *testing.T) (Model, *activity034Client) {
	t.Helper()
	a := &activity034Client{changes: []dto.Change{{ID: 12, ProjectID: 7, Title: "match active", Active: true}, {ID: 13, ProjectID: 7, Title: "match inactive", Active: false}, {ID: 14, ProjectID: 7, Title: "excluded", Active: false}, {ID: 15, ProjectID: 8, Title: "match second inactive", Active: false}}, epics: []dto.Epic{{ID: 3, ProjectID: 7, Name: "Alpha", Active: true}, {ID: 4, ProjectID: 7, Name: "Beta", Active: false}, {ID: 5, ProjectID: 8, Name: "Other", Active: true}}}
	a.projects = []dto.Option{{ID: "7", Label: "Seven"}, {ID: "8", Label: "Eight"}}
	m := newModelWithConfig(a, appConfig{ProjectID: 7})
	m.currentProject = dto.Option{ID: "7", Label: "Seven"}
	m.width, m.height = 140, 40
	m.state = ChangesListState
	next, cmd := m.beginChange(changes.List, 0, changes.Input{})
	m = applyCommand(next.(Model), cmd)
	return m, a
}

func commands034(m Model) []string {
	out := []string{}
	for _, o := range m.commandOptions(ChangesListState) {
		out = append(out, o.ID)
	}
	return out
}

func Test034InactiveFilterPersistenceMenuSummaryAndClear(t *testing.T) {
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.TrueColor)
	m, a := model034(t)
	require.Len(t, m.changeList.Rows, 1)
	require.Equal(t, "12", m.changeList.Rows[0].ID)
	var cmd tea.Cmd
	for _, inactive := range []bool{true, false, true} {
		m.changesFilters.find = "match"
		m.changesFilters.phase = dto.Option{}
		m, cmd = sendCommand(m, "/inactive-filter")
		m = applyCommand(m, cmd)
		require.Equal(t, inactive, m.changesFilters.inactive)
		require.Equal(t, "match", m.changesFilters.find)
		require.Len(t, changes.FilteredRows(m.changeList.Rows, m.changeFilters()), 1)
		commands := commands034(m)
		action := "/del-change"
		if inactive {
			action = "/undel-change"
		}
		require.Equal(t, action, commands[slices.Index(commands, "/new-change")+1])
		require.Equal(t, "/inactive-filter", commands[slices.Index(commands, "/find-filter")+1])
		summary := m.changeFiltersLine("")
		style := styles.Default.Muted
		if inactive {
			style = lipgloss.NewStyle().Foreground(styles.AccentRed)
		}
		require.Contains(t, summary, style.Render("/inactive-filter"))
		require.Greater(t, strings.Index(summary, "/inactive-filter"), strings.Index(summary, "/find-filter"))

	}
	m, cmd = sendKey(m, tea.KeyEnter)
	m = applyCommand(m, cmd)
	require.Equal(t, ChangeDetailsState, m.state)
	m, cmd = sendCommand(m, "/return")
	m = applyCommand(m, cmd)
	require.True(t, m.changesFilters.inactive)
	require.Equal(t, "13", changes.FilteredRows(m.changeList.Rows, m.changeFilters())[0].ID)
	m, cmd = sendCommand(m, "/return")
	m = applyCommand(m, cmd)
	require.Equal(t, MainState, m.state)
	m, cmd = sendCommand(m, "/changes")
	m = applyCommand(m, cmd)
	require.True(t, m.changesFilters.inactive)
	m, cmd = sendCommand(m, "/return")
	m = applyCommand(m, cmd)
	m, cmd = sendCommand(m, "/select-project")
	m = applyCommand(m, cmd)
	m, _ = sendKey(m, tea.KeyDown)
	m, cmd = sendKey(m, tea.KeyEnter)
	m = applyCommand(m, cmd)
	require.Equal(t, "8", m.currentProject.ID)
	m, cmd = sendCommand(m, "/changes")
	m = applyCommand(m, cmd)
	require.True(t, m.changesFilters.inactive)
	require.Equal(t, "15", m.changeList.Rows[0].ID)
	m.changesFilters.phase = dto.Option{ID: "old"}
	m.changesFilters.typ = dto.Option{ID: "fix"}
	m.changesFilters.epic = dto.Option{ID: "3"}
	m, cmd = sendCommand(m, "/clear-filters")
	m = applyCommand(m, cmd)
	require.Equal(t, changesFilters{}, m.changesFilters)
	require.False(t, m.changeList.Inactive)
	require.Empty(t, m.changeList.Rows)
	require.Empty(t, a.changeWrites)
}

func Test034ChangeDeactivationConfirmationAndRestoration(t *testing.T) {
	for _, entry := range []string{"key", "/del-change", "details"} {
		for _, answer := range []string{"/yes", "/no", "cancel"} {
			t.Run(entry+answer, func(t *testing.T) {
				m, a := model034(t)
				switch entry {
				case "details":
					var cmd tea.Cmd
					m, cmd = sendKey(m, tea.KeyEnter)
					m = applyCommand(m, cmd)
					m, _ = sendCommand(m, "/delete")
				case "key":
					m, _ = sendKey(m, tea.KeyDelete)
				default:
					m, _ = sendCommand(m, entry)
				}
				require.Equal(t, ChangeDeleteConfirmation, m.state)
				require.Contains(t, m.View(), "Are you sure?")
				require.NotContains(t, m.View(), "/yes")
				require.Contains(t, m.View(), "yes")
				require.NotContains(t, m.View(), "/no")
				require.Contains(t, m.View(), "no")
				if entry != "details" {
					require.Contains(t, m.View(), "match active")
				}
				var cmd tea.Cmd
				if answer == "cancel" {
					m, cmd = sendKey(m, tea.KeyEsc)
				} else {
					m.dropdown.filter = answer
					m, cmd = sendKey(m, tea.KeyEnter)
				}
				m = applyCommand(m, cmd)
				require.Zero(t, a.changeDeleteCalls)
				if answer != "/yes" {
					require.Empty(t, a.changeWrites)
					require.True(t, a.changes[0].Active)
					return
				}
				require.Equal(t, []bool{false}, a.changeWrites)
				require.False(t, a.changes[0].Active)
				require.Empty(t, m.changeList.Rows)
				m, cmd = sendCommand(m, "/inactive-filter")
				m = applyCommand(m, cmd)
				require.Equal(t, "12", m.changeList.Rows[0].ID)
				m, cmd = sendCommand(m, "/undel-change")
				m = applyCommand(m, cmd)
				require.True(t, a.changes[0].Active)
				require.Equal(t, []bool{false, true}, a.changeWrites)
				require.NotContains(t, m.changeList.Rows, changes.Present(a.changes[0]))
			})
		}
	}
	m, a := model034(t)
	m, cmd := sendKey(m, tea.KeySpace)
	require.Nil(t, cmd)
	require.Empty(t, a.changeWrites)
	m = m.setPromptValue("")
	m, cmd = sendCommand(m, "/inactive-filter")
	m = applyCommand(m, cmd)
	m, cmd = sendKey(m, tea.KeySpace)
	m = applyCommand(m, cmd)
	require.Equal(t, []bool{true}, a.changeWrites)
	require.Equal(t, "14", m.changeList.Rows[0].ID)
}

func Test034ChangeFailureAndCommittedReadOnlyRecovery(t *testing.T) {
	for _, mode := range []string{"write failure", "refresh failure"} {
		t.Run(mode, func(t *testing.T) {
			m, a := model034(t)
			if mode == "write failure" {
				a.writeErr = errors.New("rejected")
			} else {
				a.readErr = errors.New("refresh unavailable")
			}
			m, _ = sendCommand(m, "/del-change")
			m.dropdown.filter = "/yes"
			m, cmd := sendKey(m, tea.KeyEnter)
			m = applyCommand(m, cmd)
			require.Equal(t, []bool{false}, a.changeWrites)
			require.NotEmpty(t, m.err)
			if mode == "write failure" {
				require.True(t, a.changes[0].Active)
				require.NotContains(t, m.status, "deactivated")
				return
			}
			require.Contains(t, m.status, "deactivated change #12")
			require.Contains(t, m.status, "/retry reads only")
			m, cmd = sendCommand(m, "/retry")
			m = applyCommand(m, cmd)
			require.Contains(t, m.status, "deactivated change #12")
			a.readErr = nil
			m, cmd = sendCommand(m, "/retry")
			m = applyCommand(m, cmd)
			require.Empty(t, m.err)
			require.Empty(t, m.changeList.Rows)
			require.Equal(t, []bool{false}, a.changeWrites)
		})
	}
}

func Test034EpicListImmediateDeletionAndSpace(t *testing.T) {
	for _, attached := range []bool{false, true} {
		m, a := model034(t)
		m.state = MainState
		if attached {
			a.epics[0].ChangeCount = 1
		}
		m, cmd := sendCommand(m, "/epics")
		m = applyCommand(m, cmd)
		require.Len(t, m.epicList.Rows, 2)
		m, cmd = sendKey(m, tea.KeyDelete)
		require.NotNil(t, cmd)
		require.False(t, m.hasDropdown())
		m = applyCommand(m, cmd)
		require.Equal(t, []int{3}, a.epicDeletes)
		if attached {
			require.Len(t, m.epicList.Rows, 2)
			require.False(t, m.epicList.Rows[0].Active)
		} else {
			require.Len(t, m.epicList.Rows, 1)
			require.Equal(t, 4, m.epicList.Rows[0].ID)
		}
		for _, active := range []bool{true, false} {
			m, cmd = sendKey(m, tea.KeySpace)
			m = applyCommand(m, cmd)
			require.Equal(t, active, m.epicList.Rows[0].Active)
		}
		require.Equal(t, []bool{true, false}, a.epicWrites)
	}
}

func Test034OtherFiltersSurviveActivityToggleAndActivationRejection(t *testing.T) {
	m, a := model034(t)
	for i := range a.changes {
		a.changes[i].ChangePhase = "backlog"
		a.changes[i].ChangeTypes = []string{"fix"}
		id := 3
		a.changes[i].EpicID = &id
	}
	m.changesFilters = changesFilters{phase: dto.Option{ID: "backlog"}, typ: dto.Option{ID: "fix"}, epic: dto.Option{ID: "3"}, find: "match"}
	saved := m.changesFilters
	m, cmd := sendCommand(m, "/inactive-filter")
	m = applyCommand(m, cmd)
	saved.inactive = true
	require.Equal(t, saved, m.changesFilters)
	require.Len(t, changes.FilteredRows(m.changeList.Rows, m.changeFilters()), 1)
	a.writeErr = errors.New("activation rejected")
	m, cmd = sendCommand(m, "/undel-change")
	m = applyCommand(m, cmd)
	require.Equal(t, "activation rejected", m.err)
	require.False(t, a.changes[1].Active)
	require.NotContains(t, m.status, "activated change")
	require.Equal(t, []bool{true}, a.changeWrites)
}

func Test034InactiveDeleteConfirmsAndRetainsDeactivatedRow(t *testing.T) {
	m, a := model034(t)
	m, cmd := sendCommand(m, "/inactive-filter")
	m = applyCommand(m, cmd)
	m, cmd = sendKey(m, tea.KeyDelete)
	require.Nil(t, cmd)
	require.Equal(t, ChangeDeleteConfirmation, m.state)
	m.dropdown.filter = "/yes"
	a.readErr = errors.New("refresh unavailable")
	m, cmd = sendKey(m, tea.KeyEnter)
	m = applyCommand(m, cmd)
	require.Equal(t, []bool{false}, a.changeWrites)
	require.Contains(t, m.status, "deactivated change #13")
	require.Equal(t, "13", m.changeList.Rows[0].ID)
	a.readErr = nil
	m, cmd = sendCommand(m, "/retry")
	m = applyCommand(m, cmd)
	require.Equal(t, "13", m.changeList.Rows[0].ID)
	require.Empty(t, m.err)
	require.Equal(t, []bool{false}, a.changeWrites)
}
