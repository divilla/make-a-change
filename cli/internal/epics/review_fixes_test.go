package epics

import (
	"cli/internal/dto"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestP302EpicViewportKeepsSelectionVisible(t *testing.T) {
	m := Model{}
	for i := 1; i <= 40; i++ {
		m.Rows = append(m.Rows, dto.Epic{ID: i, Name: fmt.Sprintf("Epic-%02d", i)})
	}
	for _, height := range []int{1, 2, 8, 24, 50} {
		for selected := range m.Rows {
			m.Selected = selected
			view := TableView(m, 80, height)
			assert.LessOrEqual(t, len(strings.Split(view, "\n")), height)
			assert.Contains(t, view, m.Rows[selected].Name)
			if height > 1 {
				assert.Contains(t, view, "ID Name")
			}
		}
	}
	assert.Empty(t, TableView(m, 80, 0))
	assert.Empty(t, TableView(m, 80, -1))
}

func TestP303EpicOutcomeBelongsOnlyToPendingRefresh(t *testing.T) {
	for _, op := range []Operation{Create, Edit, Delete} {
		for _, refreshFails := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/refreshFails=%t", op, refreshFails), func(t *testing.T) {
				api := &epicAPI{}
				if refreshFails {
					api.readErr = errors.New("refresh unavailable")
				}
				m, cmd := (Model{ProjectID: 7}).Begin(context.Background(), api, op, 7, 3, "name")
				m, ok := m.Apply(cmd().(Result))
				require.True(t, ok)
				if refreshFails {
					require.NotEmpty(t, m.Outcome)
				} else {
					require.Empty(t, m.Outcome)
				}
				pending := m
				retry, id := Details, m.Detail.ID
				if op == Delete {
					retry, id = List, 0
				}
				if refreshFails {
					m, cmd = m.Begin(context.Background(), api, retry, 7, id, "")
					m, ok = m.Apply(cmd().(Result))
					require.True(t, ok)
					assert.Contains(t, m.Status, "refresh failed")
					assert.NotEmpty(t, m.Outcome)
					api.readErr = nil
					m, cmd = m.Begin(context.Background(), api, retry, 7, id, "")
					m, ok = m.Apply(cmd().(Result))
					require.True(t, ok)
					assert.Empty(t, m.Outcome)
				}
				api.readErr = errors.New("unrelated read unavailable")
				for _, prior := range []Model{m, pending, pending.Invalidate()} {
					next, read := prior.Begin(context.Background(), api, Details, 7, 10, "")
					next, ok = next.Apply(read().(Result))
					require.True(t, ok)
					assert.Equal(t, "load failed", next.Status)
					assert.Empty(t, next.Outcome)
				}
				// A recovered refresh must not be blamed for a later failure of the same read.
				m, cmd = m.Begin(context.Background(), api, retry, 7, id, "")
				m, ok = m.Apply(cmd().(Result))
				require.True(t, ok)
				assert.Equal(t, "load failed", m.Status)
				assert.Equal(t, 1, api.writes)
			})
		}
	}
}
