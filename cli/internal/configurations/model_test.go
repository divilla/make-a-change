package configurations

import (
	"cli/internal/dto"
	"context"
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAPI struct {
	rows     []dto.BackendConfig
	row      dto.BackendConfig
	err      error
	calls    []Operation
	canceled bool
}

func (f *fakeAPI) ListConfigurations(ctx context.Context) ([]dto.BackendConfig, error) {
	f.calls = append(f.calls, List)
	if ctx.Err() != nil {
		f.canceled = true
		return nil, ctx.Err()
	}
	return f.rows, f.err
}

func (f *fakeAPI) GetConfiguration(_ context.Context, _ string) (dto.BackendConfig, error) {
	f.calls = append(f.calls, Details)
	return f.row, f.err
}

func (f *fakeAPI) InsertConfiguration(_ context.Context, r dto.BackendConfig) (string, error) {
	f.calls = append(f.calls, Insert)
	return r.Slug, f.err
}

func (f *fakeAPI) UpdateConfiguration(_ context.Context, _ dto.BackendConfig) error {
	f.calls = append(f.calls, Update)
	return f.err
}

func (f *fakeAPI) DeleteConfiguration(_ context.Context, _ string) error {
	f.calls = append(f.calls, Delete)
	return f.err
}

func fixture() dto.BackendConfig {
	return dto.BackendConfig{Slug: "fixed", ProjectDocs: []string{"a", "b"}, EpicDocs: []string{}, ChangeDocs: []string{"brief"}, ChangePhases: []string{"todo", "done"}, ChangeColors: []string{"1", "2"}, ChangeTypes: []string{"fix"}}
}

func result(t *testing.T, cmd tea.Cmd) Result {
	t.Helper()
	require.NotNil(t, cmd)
	return cmd().(Result)
}

func TestP702CreateUpdateImmutableSlugAndExplicitEmptyArrays(t *testing.T) {
	m := Model{}.OpenForm(false)
	m.Raw[0] = "created"
	row, err := m.ParseDraft()
	require.NoError(t, err)
	assert.Equal(t, []string{}, row.ProjectDocs)
	assert.Equal(t, []string{}, row.ChangeTypes)
	m.Raw[2] = `["first","second"]`
	row, err = m.ParseDraft()
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, row.EpicDocs)
	for _, bad := range []string{"", `null`, `[null]`, `[" "]`, `{}`} {
		m.Raw[2] = bad
		_, err = m.ParseDraft()
		require.Error(t, err, bad)
	}
	m.Detail = fixture()
	m = m.OpenForm(true)
	assert.Equal(t, 1, m.Field)
	m.Raw[0] = "renamed"
	_, err = m.ParseDraft()
	require.ErrorContains(t, err, "immutable")
	m.Raw[0] = "fixed"
	m.Raw[1] = "[]"
	row, err = m.ParseDraft()
	require.NoError(t, err)
	assert.Equal(t, "fixed", row.Slug)
	assert.Equal(t, []string{}, row.ProjectDocs)
}

func TestP702ConfigViewportAndTerminalSafeValues(t *testing.T) {
	row := fixture()
	row.ProjectDocs = []string{"\x1b[2Jsecret\n" + strings.Repeat("long", 100)}
	m := Model{Detail: row, DetailLoaded: true}
	first := View(m, 30, 5)
	assert.NotContains(t, first, "\x1b[2J")
	assert.LessOrEqual(t, len(strings.Split(first, "\n")), 5)
	m.Offset = 10
	later := View(m, 30, 5)
	assert.NotEqual(t, first, later)
	assert.Contains(t, View(Model{Rows: []dto.BackendConfig{fixture()}, Selected: 0}, 80, 10), "> fixed")
}

func TestP702ConfigScrollClampsStoredOffsetToLastPage(t *testing.T) {
	row := fixture()
	row.ProjectDocs = []string{strings.Repeat("long", 100)}
	for name, model := range map[string]Model{
		"details": {Detail: row, DetailLoaded: true},
		"form":    Model{Detail: row}.OpenForm(true),
	} {
		t.Run(name, func(t *testing.T) {
			lastPage := max(0, len(wrappedLines(model, 30))-5)
			require.Positive(t, lastPage)
			for range 30 {
				model = model.Scroll(5, 30, 5)
			}
			assert.Equal(t, lastPage, model.Offset)
			bottom := View(model, 30, 5)
			model = model.Scroll(-5, 30, 5)
			assert.Equal(t, lastPage-5, model.Offset)
			assert.NotEqual(t, bottom, View(model, 30, 5))
			model.Offset = lastPage + 100
			model = model.Scroll(-5, 30, 5)
			assert.Equal(t, lastPage-5, model.Offset)
		})
	}
	short := Model{Detail: fixture(), DetailLoaded: true}
	short = short.Scroll(100, 80, 50)
	assert.Zero(t, short.Offset)
}

func TestP703CommittedConfigWriteSurvivesRepeatedFailedRefresh(t *testing.T) {
	api := &fakeAPI{row: fixture(), rows: []dto.BackendConfig{fixture()}}
	m := Model{Detail: fixture()}.OpenForm(true)
	var cmd tea.Cmd
	m, cmd = m.Begin(context.Background(), api, Update, "fixed", fixture())
	r := result(t, cmd)
	m, ok := m.Apply(r)
	require.True(t, ok)
	assert.Equal(t, "updated configuration fixed", m.Committed)
	assert.Equal(t, List, m.Refresh)
	api.err = errors.New("read failed")
	for range 2 {
		m, cmd = m.Begin(context.Background(), api, List, "", dto.BackendConfig{})
		r = result(t, cmd)
		m, ok = m.Apply(r)
		require.True(t, ok)
		assert.Contains(t, m.Status, "updated configuration fixed; refresh failed")
		assert.Empty(t, m.Rows)
		assert.Equal(t, []Operation{Update}, api.calls[:1])
	}
	api.err = nil
	m, cmd = m.Begin(context.Background(), api, List, "", dto.BackendConfig{})
	m, ok = m.Apply(result(t, cmd))
	require.True(t, ok)
	assert.Equal(t, "updated configuration fixed", m.Status)
	assert.Len(t, m.Rows, 1)
	assert.Equal(t, []Operation{Update, List, List, List}, api.calls)
}

func TestP703StaleConfigResultsAndShutdownCancellation(t *testing.T) {
	api := &fakeAPI{rows: []dto.BackendConfig{fixture()}}
	m, cmd := Model{}.Begin(context.Background(), api, List, "", dto.BackendConfig{})
	old := result(t, cmd)
	m = m.Invalidate()
	_, ok := m.Apply(old)
	assert.False(t, ok)
	m, cmd = m.Begin(context.Background(), api, List, "", dto.BackendConfig{})
	m = m.Invalidate()
	assert.False(t, m.Busy)
	_ = result(t, cmd)
	assert.True(t, api.canceled)
}

func TestP702DeleteConfirmationConflictAndDraftRecovery(t *testing.T) {
	api := &fakeAPI{err: errors.New("config is in use")}
	m := Model{Detail: fixture(), DetailLoaded: true, Confirm: true}
	var cmd tea.Cmd
	m, cmd = m.Begin(context.Background(), api, Delete, "fixed", dto.BackendConfig{})
	m, ok := m.Apply(result(t, cmd))
	require.True(t, ok)
	assert.ErrorContains(t, m.Err, "in use")
	assert.True(t, m.Confirm)
	assert.True(t, m.DetailLoaded)
	m = m.OpenForm(true)
	m.Raw[1] = `["changed"]`
	api.err = errors.New("write failed")
	m, cmd = m.Begin(context.Background(), api, Update, "fixed", fixture())
	m, ok = m.Apply(result(t, cmd))
	require.True(t, ok)
	assert.Equal(t, `["changed"]`, m.Raw[1])
	assert.True(t, m.Form)
}
