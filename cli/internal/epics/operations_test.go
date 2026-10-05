package epics

import (
	"cli/internal/dto"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type epicAPI struct {
	writes, reads     int
	writeErr, readErr error
	name              string
	rows              []dto.Epic
	get               func(context.Context, int) (dto.Epic, error)
}

func (f *epicAPI) ListEpics(context.Context, int) ([]dto.Epic, error) {
	f.reads++
	return f.rows, f.readErr
}

func (f *epicAPI) GetEpic(ctx context.Context, id int) (dto.Epic, error) {
	f.reads++
	if f.get != nil {
		return f.get(ctx, id)
	}
	return dto.Epic{ID: id, ProjectID: 7, Name: f.name, Completed: 63}, f.readErr
}

func (f *epicAPI) CreateEpic(_ context.Context, _ int, name string) (int, error) {
	f.writes++
	f.name = name
	return 9, f.writeErr
}

func (f *epicAPI) UpdateEpic(_ context.Context, _ int, name string) error {
	f.writes++
	f.name = name
	return f.writeErr
}
func (f *epicAPI) DeleteEpic(context.Context, int) error { f.writes++; return f.writeErr }

func TestP302EpicOperationsValidationFormsAndPresentation(t *testing.T) {
	ctx := context.Background()
	for _, op := range []Operation{List, Details, Create, Edit, Delete} {
		t.Run(string(op), func(t *testing.T) {
			api := &epicAPI{}
			m, cmd := (Model{}).Begin(ctx, api, op, 7, 3, "new")
			require.NotNil(t, cmd)
			assert.True(t, m.Loading)
			m, ok := m.Apply(cmd().(Result))
			require.True(t, ok)
			assert.False(t, m.Loading)
			require.NoError(t, m.Err)
			if op == List {
				assert.Equal(t, "no epics", m.Status)
			}
			_, cmd = m.Begin(ctx, api, op, 0, 3, "new")
			assert.Nil(t, cmd)
			if op != List && op != Create {
				_, cmd = m.Begin(ctx, api, op, 7, 0, "new")
				assert.Nil(t, cmd)
			}
			if op == Create || op == Edit {
				invalid, cmd := m.Begin(ctx, api, op, 7, 3, " \t\n")
				assert.Nil(t, cmd)
				assert.Equal(t, " \t\n", invalid.Draft)
				require.ErrorContains(t, invalid.Err, "name is required")
			}
		})
	}
	e := dto.Epic{ID: 3, ProjectID: 7, Name: "Exact\nname", DoneTC: 2, TotalTC: 8, Completed: 63, ChangeCount: 4, CreatedAt: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 9, 28, 11, 0, 0, 0, time.UTC)}
	m := Model{ProjectID: 7, Detail: e, DetailLoaded: true, Rows: []dto.Epic{e, {ID: 4, ProjectID: 7, Name: "Other"}}}
	for _, part := range []string{"ID: 3", "Project ID: 7", "Name: Exact\nname", "Done TC: 2", "Total TC: 8", "Completed: 63", "Changes: 4", "Created: " + e.CreatedAt.Local().Format("2006-01-02 15:04"), "Modified: " + e.UpdatedAt.Local().Format("2006-01-02 15:04")} {
		assert.Contains(t, DetailsView(m, 160), part)
	}
	assert.Contains(t, TableView(m, 160, 24), "3  Exact name  2/8  63  4")
	assert.Contains(t, TableView(Model{}, 80, 24), "No epics")
	assert.Contains(t, TableView(Model{Loading: true}, 80, 24), "loading")
	assert.Contains(t, DetailsView(Model{Loading: true}, 80), "Loading")
	assert.Empty(t, DetailsView(Model{}, 80))
	assert.Contains(t, DetailsView(Model{Detail: e}, 80), "Details unavailable")
	form, ok := m.Form(true)
	require.True(t, ok)
	assert.Equal(t, e.Name, form.Draft)
	form, ok = m.Form(false)
	require.True(t, ok)
	assert.Empty(t, form.Draft)
	_, ok = (Model{}).Form(false)
	assert.False(t, ok)
	_, ok = (Model{ProjectID: 7}).Form(true)
	assert.False(t, ok)
	moved := m.MoveSelection(100)
	assert.Equal(t, 1, moved.Selected)
	moved = moved.MoveSelection(-100)
	assert.Zero(t, moved.Selected)
	selected, e2, ok := moved.SelectDetail()
	assert.True(t, ok)
	assert.Equal(t, e, e2)
	assert.Equal(t, e, selected.Detail)
	assert.Zero(t, (Model{}).MoveSelection(2).Selected)
	_, _, ok = (Model{Loading: true, Rows: []dto.Epic{e}}).SelectDetail()
	assert.False(t, ok)
	assert.Equal(t, []dto.Option{{ID: "3", Label: "Exact\nname"}, {ID: "4", Label: "Other"}}, Options(m.Rows))
	assert.Contains(t, ListCommands(), "/new-epic")
	for _, cmd := range []string{"/edit", "/delete", "/retry"} {
		assert.Contains(t, DetailCommands(), cmd)
	}
	assert.Contains(t, ListTitle(), "Epics")
	assert.Contains(t, DetailTitle(), "Epic")
}

func TestP303EpicMutationsExactlyOnceFailureRecoveryAndNoOp(t *testing.T) {
	for _, op := range []Operation{Create, Edit, Delete} {
		for _, mode := range []string{"success", "write failure", "refresh failure"} {
			t.Run(string(op)+mode, func(t *testing.T) {
				api := &epicAPI{rows: []dto.Epic{{ID: 10, ProjectID: 7, Name: "Unrelated"}}}
				failure := errors.New("unavailable")
				if mode == "write failure" {
					api.writeErr = failure
				}
				if mode == "refresh failure" {
					api.readErr = failure
					api.rows = nil
				}
				m := Model{ProjectID: 7, Rows: []dto.Epic{{ID: 3, ProjectID: 7}, {ID: 10, ProjectID: 7, Name: "Unrelated"}}, Detail: dto.Epic{ID: 3, ProjectID: 7, Name: "before"}}
				raw := "/cancel\n\t"
				m, cmd := m.Begin(context.Background(), api, op, 7, 3, raw)
				require.NotNil(t, cmd)
				_, duplicate := m.Begin(context.Background(), api, op, 7, 3, raw)
				assert.Nil(t, duplicate)
				r := cmd().(Result)
				m, ok := m.Apply(r)
				require.True(t, ok)
				assert.Equal(t, 1, api.writes)
				_, accepted := m.Apply(r)
				assert.False(t, accepted)
				if mode == "write failure" {
					require.ErrorIs(t, m.Err, failure)
					assert.Zero(t, api.reads)
					if op != Delete {
						assert.Equal(t, raw, m.Draft)
					}
					return
				}
				assert.Equal(t, 1, api.reads)
				assert.Empty(t, m.Draft)
				if op == Delete {
					assert.Equal(t, 10, m.Rows[0].ID)
					assert.Zero(t, m.Detail.ID)
				} else {
					assert.Equal(t, raw, m.Detail.Name)
					assert.Positive(t, m.Detail.ID)
				}
				if op == Create {
					assert.Equal(t, 9, m.Detail.ID)
				}
				if mode == "refresh failure" {
					require.ErrorIs(t, m.Err, failure)
					assert.Contains(t, m.Status, "refresh failed")
					assert.Contains(t, m.Status, "/retry reads only")
				}
				retry := Details
				id := m.Detail.ID
				if op == Delete {
					retry = List
					id = 0
				}
				m, cmd = m.Begin(context.Background(), api, retry, 7, id, "")
				m, ok = m.Apply(cmd().(Result))
				require.True(t, ok)
				if mode == "refresh failure" {
					assert.Contains(t, m.Status, "refresh failed")
					assert.NotEmpty(t, m.Outcome)
				}
				api.readErr = nil
				m, cmd = m.Begin(context.Background(), api, retry, 7, id, "")
				m, ok = m.Apply(cmd().(Result))
				require.True(t, ok)
				require.NoError(t, m.Err)
				assert.Equal(t, 1, api.writes)
			})
		}
	}
	api := &epicAPI{}
	m := Model{ProjectID: 7, Detail: dto.Epic{ID: 3, ProjectID: 7, Name: " /cancel\n"}, DetailLoaded: true}
	m, cmd := m.Begin(context.Background(), api, Edit, 7, 3, m.Detail.Name)
	assert.Nil(t, cmd)
	assert.Equal(t, "unchanged", m.Status)
	assert.Zero(t, api.writes)
}

func TestP304EpicIdentityCancellationAndHiddenRows(t *testing.T) {
	api := &epicAPI{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, cmd := (Model{}).Begin(ctx, api, Details, 7, 3, "")
	api.get = func(ctx context.Context, id int) (dto.Epic, error) {
		require.ErrorIs(t, ctx.Err(), context.Canceled)
		return dto.Epic{ID: id, ProjectID: 7, Name: "stale"}, nil
	}
	m = m.Invalidate()
	stale := cmd().(Result)
	next, accepted := m.Apply(stale)
	assert.False(t, accepted)
	assert.Empty(t, next.Detail.Name)
	m, cmd = m.Begin(ctx, api, Details, 7, 4, "")
	forged := Result{Generation: m.Generation, Operation: Details, ProjectID: 8, ID: 4}
	_, accepted = m.Apply(forged)
	assert.False(t, accepted)
	forged.ProjectID = 7
	forged.ID = 3
	_, accepted = m.Apply(forged)
	assert.False(t, accepted)
	forged.ID = 4
	forged.Operation = List
	_, accepted = m.Apply(forged)
	assert.False(t, accepted)
	m = m.Scope(8)
	stale = cmd().(Result)
	_, accepted = m.Apply(stale)
	assert.False(t, accepted)
	assert.False(t, m.Loading)
	api.get = nil
	m = Model{ProjectID: 7, Rows: []dto.Epic{{ID: 3, ProjectID: 7}}}
	m, cmd = m.Begin(ctx, api, List, 7, 0, "")
	assert.Empty(t, m.Rows)
	_, _, ok := m.SelectDetail()
	assert.False(t, ok)
	m, ok = m.Apply(cmd().(Result))
	assert.True(t, ok)
	assert.Equal(t, "no epics", m.Status)
	api.get = func(context.Context, int) (dto.Epic, error) { return dto.Epic{ID: 3, ProjectID: 8}, nil }
	m, cmd = m.Begin(ctx, api, Details, 7, 3, "")
	m, ok = m.Apply(cmd().(Result))
	assert.True(t, ok)
	require.ErrorContains(t, m.Err, "selected project")
	cancel()
	api.get = func(ctx context.Context, _ int) (dto.Epic, error) { return dto.Epic{}, ctx.Err() }
	m, cmd = m.Begin(ctx, api, Details, 7, 3, "")
	m, ok = m.Apply(cmd().(Result))
	assert.True(t, ok)
	require.ErrorIs(t, m.Err, context.Canceled)
}
