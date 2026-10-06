package epics

import (
	"cli/internal/dto"
	"cli/internal/styles"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

func Test034EpicTableColumnsColorsAndBounds(t *testing.T) {
	profile := lipgloss.ColorProfile()
	t.Cleanup(func() { lipgloss.SetColorProfile(profile) })
	lipgloss.SetColorProfile(termenv.TrueColor)
	header := epicTableLine("ID", "Name", "DoneTC", "Compl", "Chngs", "Active")
	require.Equal(t, "  ID Name                           DoneTC Compl Chngs Active  ", header)
	rows := []dto.Epic{{ID: 7, Name: "Alpha", Active: true, DoneTC: 2, TotalTC: 8, Completed: 100, ChangeCount: 1}, {ID: 1234, Name: "Beta", Active: false, DoneTC: 1, TotalTC: 3, Completed: 33, ChangeCount: 12}}
	for selected := range rows {
		view := TableView(Model{Rows: rows, Selected: selected}, 100, 10)
		for i, e := range rows {
			state := ""
			if !e.Active {
				state = "inactive"
			}
			line := epicTableLine(map[int]string{7: "7", 1234: "1234"}[e.ID], e.Name, []string{"2/8", "1/3"}[i], []string{"100%", "33%"}[i], []string{"1", "12"}[i], state)
			require.Contains(t, view, epicRowStyle(e, i == selected).Render(line))
			cells := []struct {
				start, end int
				value      string
			}{{0, 4, []string{"   7", "1234"}[i]}, {5, 35, e.Name + strings.Repeat(" ", 30-len(e.Name))}, {36, 42, []string{"   2/8", "   1/3"}[i]}, {43, 48, []string{" 100%", "  33%"}[i]}, {49, 54, []string{"    1", "   12"}[i]}, {55, 63, []string{"        ", "inactive"}[i]}}
			for _, c := range cells {
				require.Equal(t, c.value, line[c.start:c.end])
			}
			want := styles.Foreground
			if !e.Active {
				want = styles.AccentRed
			}
			require.Equal(t, want, epicRowStyle(e, i == selected).GetForeground())
			if i == selected {
				require.Equal(t, styles.MutedPurple, epicRowStyle(e, true).GetBackground())
			}
		}
	}
	for _, name := range []string{strings.Repeat("a", 50), strings.Repeat("界", 30), "A\nB\tC"} {
		line := epicTableLine("7", name, "2/8", "33%", "12", "inactive")
		require.Equal(t, 63, ansi.StringWidth(line))
		require.True(t, strings.HasSuffix(line, "   2/8   33%    12 inactive"))
	}
	for _, width := range []int{20, 30, 40, 65, 100} {
		for _, height := range []int{1, 2, 4, 5, 10} {
			view := TableView(Model{Rows: rows}, width, height)
			require.LessOrEqual(t, len(strings.Split(view, "\n")), height)
			for _, line := range strings.Split(view, "\n") {
				require.LessOrEqual(t, ansi.StringWidth(line), width)
			}
		}
	}
}

func Test034EpicToggleMutationAndReadOnlyRecovery(t *testing.T) {
	for _, active := range []bool{true, false} {
		for _, mode := range []string{"success", "write failure", "refresh failure"} {
			t.Run(strings.Join([]string{map[bool]string{true: "active", false: "inactive"}[active], mode}, "/"), func(t *testing.T) {
				api := &epicAPI{rows: []dto.Epic{{ID: 3, ProjectID: 7, Active: !active}}}
				if mode == "write failure" {
					api.writeErr = errors.New("rejected")
				}
				if mode == "refresh failure" {
					api.readErr = errors.New("refresh failed")
				}
				m := Model{ProjectID: 7, Rows: []dto.Epic{{ID: 3, ProjectID: 7, Active: active}}}
				m, cmd := m.Begin(context.Background(), api, ToggleActive, 7, 3, "")
				r := cmd().(Result)
				require.Equal(t, !active, r.Epic.Active)
				m, ok := m.Apply(r)
				require.True(t, ok)
				require.Equal(t, 1, api.writes)
				if mode == "write failure" {
					require.Error(t, m.Err)
					require.Zero(t, api.reads)
					require.Empty(t, m.Outcome)
					require.Equal(t, active, m.Rows[0].Active)
					return
				}
				if mode == "success" {
					require.NoError(t, m.Err)
					require.Equal(t, !active, m.Rows[0].Active)
					return
				}
				require.Contains(t, m.Status, "epic activity saved")
				require.Contains(t, m.Status, "/retry reads only")
				m, cmd = m.Begin(context.Background(), api, List, 7, 0, "")
				m, ok = m.Apply(cmd().(Result))
				require.True(t, ok)
				require.Contains(t, m.Status, "epic activity saved")
				api.readErr = nil
				m, cmd = m.Begin(context.Background(), api, List, 7, 0, "")
				m, ok = m.Apply(cmd().(Result))
				require.True(t, ok)
				require.NoError(t, m.Err)
				require.Equal(t, 1, api.writes)
				require.Equal(t, !active, m.Rows[0].Active)
			})
		}
	}
}

func Test034EpicToggleRequiresLoadedScopedSelection(t *testing.T) {
	api := &epicAPI{}
	for _, m := range []Model{{ProjectID: 7}, {ProjectID: 7, Loading: true, Rows: []dto.Epic{{ID: 3, ProjectID: 7}}}, {ProjectID: 7, Rows: []dto.Epic{{ID: 3, ProjectID: 8}}}} {
		m, cmd := m.Begin(context.Background(), api, ToggleActive, 7, 3, "")
		require.Nil(t, cmd)
		require.Error(t, m.Err)
		require.Zero(t, api.writes)
	}
}
