package testcases

import (
	"cli/internal/dto"
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

type fakeAPI struct {
	calls                        []string
	rows                         []dto.TestCase
	listErr, changeErr, writeErr error
	done                         bool
	honorContext                 bool
	changeProject                int
}

func (f *fakeAPI) ListTestCases(ctx context.Context, _ int) ([]dto.TestCase, error) {
	f.calls = append(f.calls, "list")
	if f.honorContext && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return f.rows, f.listErr
}

func (f *fakeAPI) GetChange(ctx context.Context, id int) (dto.Change, error) {
	f.calls = append(f.calls, "change")
	if f.honorContext && ctx.Err() != nil {
		return dto.Change{}, ctx.Err()
	}
	project := f.changeProject
	if project == 0 {
		project = 7
	}
	return dto.Change{ID: id, ProjectID: project, DoneTC: 1, TotalTC: 2}, f.changeErr
}

func (f *fakeAPI) CreateTestCase(_ context.Context, _ int, _ string) (int, error) {
	f.calls = append(f.calls, "create")
	return 31, f.writeErr
}

func (f *fakeAPI) UpdateTestCase(_ context.Context, _ int, _ string) error {
	f.calls = append(f.calls, "edit")
	return f.writeErr
}

func (f *fakeAPI) UpdateTestCaseDone(_ context.Context, _ int, done bool) error {
	f.calls = append(f.calls, "done")
	f.done = done
	return f.writeErr
}

func (f *fakeAPI) DeleteTestCase(_ context.Context, _ int) error {
	f.calls = append(f.calls, "delete")
	return f.writeErr
}

func TestP502TestCaseFormValidationAndLiteralDraft(t *testing.T) {
	f := &fakeAPI{}
	m, cmd := (Model{}).Begin(context.Background(), f, 7, 12, Create, 0, " \n ", false)
	require.Nil(t, cmd)
	require.Error(t, m.Err)
	require.Empty(t, f.calls)
	require.Equal(t, " \n ", m.Draft)
	m, cmd = m.Begin(context.Background(), f, 7, 12, Create, 0, "/delete", false)
	require.NotNil(t, cmd)
	r := cmd().(Result)
	require.Equal(t, 31, r.ID)
	require.Equal(t, []string{"create", "list", "change"}, f.calls)
	m, ok := m.Apply(r)
	require.True(t, ok)
	require.Equal(t, 31, m.CommittedID)
	require.Empty(t, m.Draft)
	require.Contains(t, m.Status, "saved")
	_, err := ParseID("nope")
	require.Error(t, err)
	m = m.OpenEdit("31", "/delete literal")
	require.Equal(t, "31", m.TargetID)
	require.Equal(t, "/delete literal", m.Draft)
	m = m.ClearForm()
	require.Empty(t, m.TargetID)
	require.Empty(t, m.Draft)
	m = m.OpenDelete("invalid")
	m, cmd = m.BeginTarget(context.Background(), f, 7, 12, Delete, "", false)
	require.Nil(t, cmd)
	require.Error(t, m.Err)
	require.Equal(t, []string{"create", "list", "change"}, f.calls)
	m = m.OpenCreate()
	require.Empty(t, m.TargetID)
}

func TestP502CreateEditToggleDeleteNavigation(t *testing.T) {
	f := &fakeAPI{rows: []dto.TestCase{{ID: 31, ChangeID: 12, Scenario: "text"}}}
	m := Model{}
	for _, op := range []Operation{Create, Edit, SetDone, SetDone, Delete} {
		done := op == SetDone && !f.done
		var command tea.Cmd
		m, command = m.Begin(context.Background(), f, 7, 12, op, 31, "literal", done)
		require.NotNil(t, command)
		r := command().(Result)
		require.True(t, r.Committed)
		m, _ = m.Apply(r)
		require.True(t, m.Loaded)
	}
	require.Equal(t, []string{"create", "list", "change", "edit", "list", "change", "done", "list", "change", "done", "list", "change", "delete", "list", "change"}, f.calls)
	require.False(t, f.done)
}

func TestP503FeatureOwnsTestCaseTransitionsAndMessages(t *testing.T) {
	f := &fakeAPI{}
	m, cmd := (Model{}).Begin(context.Background(), f, 7, 12, Create, 0, "data", false)
	r := cmd().(Result)
	require.Equal(t, Create, r.Operation)
	require.Equal(t, 31, r.ID)
	m, ok := m.Apply(r)
	require.True(t, ok)
	require.Equal(t, 31, m.CommittedID)
	require.False(t, m.Busy)
	require.True(t, m.Loaded)
}

func TestP503FeatureSuppliesCreateAndEditFormText(t *testing.T) {
	create, edit := CreateForm(), EditForm()
	require.Equal(t, "Write a Scenario", create.Placeholder)
	require.Contains(t, create.Help, "<return> save")
	require.Contains(t, create.Help, "<esc> cancel")
	require.Equal(t, "new test case", create.Status)
	require.Equal(t, create.Placeholder, edit.Placeholder)
	require.Equal(t, create.Help, edit.Help)
	require.Equal(t, "editing test case", edit.Status)
}

func TestP504CommittedTestCaseWriteSurvivesRefreshFailure(t *testing.T) {
	readErr := errors.New("read unavailable")
	f := &fakeAPI{listErr: readErr}
	m, cmd := (Model{}).Begin(context.Background(), f, 7, 12, Create, 0, "data", false)
	r := cmd().(Result)
	require.True(t, r.Committed)
	require.Equal(t, 31, r.ID)
	require.ErrorIs(t, r.RefreshErr, readErr)
	m, ok := m.Apply(r)
	require.True(t, ok)
	require.Contains(t, m.Status, "saved test case; refresh failed")
	require.Equal(t, 31, m.CommittedID)
	require.False(t, m.Loaded)
	require.Equal(t, []string{"create", "list", "change"}, f.calls)
	f.listErr = nil
	m, cmd = m.Begin(context.Background(), f, 7, 12, Refresh, 0, "", false)
	r = cmd().(Result)
	require.False(t, r.Committed)
	m, _ = m.Apply(r)
	require.True(t, m.Loaded)
	require.Equal(t, 1, count(f.calls, "create"))
}

func TestP504CommittedOutcomeAcrossRepeatedReadOnlyFailures(t *testing.T) {
	for _, tc := range []struct {
		name, write, status string
		op                  Operation
	}{
		{name: "create", write: "create", status: "saved test case", op: Create},
		{name: "delete", write: "delete", status: "deleted test case", op: Delete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeAPI{listErr: errors.New("read unavailable")}
			m, cmd := (Model{}).Begin(context.Background(), f, 7, 12, tc.op, 31, "scenario", false)
			m, ok := m.Apply(cmd().(Result))
			require.True(t, ok)
			suffix := ""
			if tc.op == Create {
				suffix = " (#31)"
			}
			for attempt := 0; attempt < 2; attempt++ {
				require.Equal(t, tc.status+"; refresh failed"+suffix, m.Status)
				require.Equal(t, 1, count(f.calls, tc.write))
				if tc.op == Create {
					require.Equal(t, 31, m.CommittedID)
				}
				m, cmd = m.Begin(context.Background(), f, 7, 12, Refresh, 0, "", false)
				m, ok = m.Apply(cmd().(Result))
				require.True(t, ok)
			}
			invalidated := m.Invalidate()
			require.False(t, invalidated.NeedsRefresh())
			require.Zero(t, invalidated.CommittedID)
			newWrite, newCommand := m.Begin(context.Background(), f, 7, 12, Edit, 31, "replacement", false)
			require.NotNil(t, newCommand)
			require.False(t, newWrite.NeedsRefresh())
			require.Zero(t, newWrite.CommittedID)
			f.listErr = nil
			m, cmd = m.Begin(context.Background(), f, 7, 12, Refresh, 0, "", false)
			m, ok = m.Apply(cmd().(Result))
			require.True(t, ok)
			require.True(t, m.Loaded)
			require.Equal(t, tc.status+"; refreshed test cases"+suffix, m.Status)
			require.Equal(t, 1, count(f.calls, tc.write))
			m, cmd = m.Begin(context.Background(), f, 7, 12, Refresh, 0, "", false)
			m, ok = m.Apply(cmd().(Result))
			require.True(t, ok)
			require.Equal(t, "refreshed test cases", m.Status)
		})
	}
}

func TestP504FailedWriteRetryAndBusyDeduplication(t *testing.T) {
	f := &fakeAPI{writeErr: errors.New("offline")}
	m, cmd := (Model{}).Begin(context.Background(), f, 7, 12, Edit, 31, "draft", false)
	_, duplicate := m.Begin(context.Background(), f, 7, 12, Edit, 31, "draft", false)
	require.Nil(t, duplicate)
	r := cmd().(Result)
	require.False(t, r.Committed)
	require.Equal(t, []string{"edit"}, f.calls)
	m, _ = m.Apply(r)
	require.Equal(t, "draft", m.Draft)
	require.Equal(t, "save failed", m.Status)
	f.writeErr = nil
	m, cmd = m.Begin(context.Background(), f, 7, 12, Edit, 31, m.Draft, false)
	r = cmd().(Result)
	m, _ = m.Apply(r)
	require.Equal(t, 2, count(f.calls, "edit"))
	require.True(t, m.Loaded)
}

func TestP504StaleProjectChangeAndRevisionResults(t *testing.T) {
	f := &fakeAPI{}
	m, cmd := (Model{}).Begin(context.Background(), f, 7, 12, Refresh, 0, "", false)
	stale := cmd().(Result)
	m = m.Invalidate()
	_, ok := m.Apply(stale)
	require.False(t, ok)
	m, cmd = m.Begin(context.Background(), f, 7, 13, Refresh, 0, "", false)
	wrong := cmd().(Result)
	wrong.ProjectID = 8
	_, ok = m.Apply(wrong)
	require.False(t, ok)
	wrong.ProjectID = 7
	wrong.ChangeID = 12
	_, ok = m.Apply(wrong)
	require.False(t, ok)
}

func TestP504CanceledTestCaseWorkAndReadOnlyRetry(t *testing.T) {
	f := &fakeAPI{honorContext: true}
	ctx, cancel := context.WithCancel(context.Background())
	m, cmd := (Model{}).Begin(ctx, f, 7, 12, Refresh, 0, "", false)
	cancel()
	require.NotNil(t, cmd)
	m = m.Invalidate()
	require.False(t, m.Busy)
	r := cmd().(Result)
	require.ErrorIs(t, r.RefreshErr, context.Canceled)
	_, ok := m.Apply(r)
	require.False(t, ok)
}

func count(items []string, want string) int {
	n := 0
	for _, s := range items {
		if s == want {
			n++
		}
	}
	return n
}

func TestP504InvalidScopeOwnerAndReadFailure(t *testing.T) {
	f := &fakeAPI{}
	m, cmd := (Model{}).Begin(context.Background(), f, 0, 12, Create, 0, "x", false)
	require.Nil(t, cmd)
	require.Error(t, m.Err)
	require.Empty(t, f.calls)
	m, cmd = m.Begin(context.Background(), f, 7, 12, SetDone, 0, "", true)
	require.Nil(t, cmd)
	require.Error(t, m.Err)
	require.Empty(t, f.calls)
	f.changeErr = errors.New("details offline")
	m, cmd = m.Begin(context.Background(), f, 7, 12, Refresh, 0, "", false)
	r := cmd().(Result)
	require.ErrorIs(t, r.RefreshErr, f.changeErr)
	m, _ = m.Apply(r)
	require.Equal(t, "refresh failed", m.Status)
	f.changeErr = nil
	f.changeProject = 8
	m, cmd = m.Begin(context.Background(), f, 7, 12, Refresh, 0, "", false)
	r = cmd().(Result)
	require.ErrorContains(t, r.RefreshErr, "does not belong")
	m, _ = m.Apply(r)
	require.False(t, m.Loaded)
	require.Equal(t, "no test cases", Summary(Model{Loaded: true}))
	require.Equal(t, "2 test cases", Summary(Model{Loaded: true, Rows: []dto.TestCase{{}, {}}}))
}
