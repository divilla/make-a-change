package testcases

import (
	"cli/internal/dto"
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type syncAPI struct {
	rows   []dto.TestCase
	change int
	calls  []string
	fail   string
	next   int
}

func (a *syncAPI) ListTestCases(_ context.Context, id int) ([]dto.TestCase, error) {
	a.change = id
	a.calls = append(a.calls, "list")
	if a.fail == "list" {
		return nil, errors.New("list failed")
	}
	return slices.Clone(a.rows), nil
}

func (a *syncAPI) DeleteTestCase(_ context.Context, id int) error {
	call := fmt.Sprintf("delete:%d", id)
	a.calls = append(a.calls, call)
	if a.fail == call {
		return errors.New("delete failed")
	}
	a.rows = slices.DeleteFunc(a.rows, func(r dto.TestCase) bool { return r.ID == id })
	return nil
}

func (a *syncAPI) CreateTestCase(_ context.Context, id int, text string) (int, error) {
	a.calls = append(a.calls, "create:"+text)
	if a.fail == "create:"+text {
		return 0, errors.New("create failed")
	}
	a.next++
	a.rows = append(a.rows, dto.TestCase{ID: a.next, ChangeID: id, Scenario: text, Done: false})
	return a.next, nil
}

func specCases(bullets string) string {
	return "# Spec\n## Testcases\n" + bullets + "\n## Notes\nIgnored notes\n"
}

func Test032TestcaseSectionValidatedBeforeAPI(t *testing.T) {
	for _, text := range []string{"", "## Testcases\n", "# Testcases\n- Q → R", "## Testcases\n- Q → R\nbad entry", "## Testcases\n- Q", "## Testcases\n- → R", "## Testcases\n- Q → ", "## Testcases\n  - Q → R", "## Testcases\n- Q → R\n continuation", "```\n## Testcases\n- Q → R\n```"} {
		t.Run(text, func(t *testing.T) {
			a := &syncAPI{rows: []dto.TestCase{{ID: 1, ChangeID: 12, Scenario: "old", Done: true}}}
			old := slices.Clone(a.rows)
			require.ErrorContains(t, Synchronize(context.Background(), a, 12, text), "testcase error")
			require.Empty(t, a.calls)
			require.Equal(t, old, a.rows)
		})
	}
	parsed, err := ParseSpecCases(specCases("\n- Click `save` → saved ✓.\n- Click `save` → saved ✓."))
	require.NoError(t, err)
	require.Equal(t, []string{"Click `save` → saved ✓.", "Click `save` → saved ✓."}, parsed)
}

func Test032FencedExamplesCannotSupplyTestcases(t *testing.T) {
	for _, tc := range []struct {
		name, opening, embedded, closing string
	}{
		{"shorter backticks", "````markdown", "```", "````"},
		{"shorter tildes", "~~~~markdown", "~~~", "~~~~"},
		{"different character", "```markdown", "~~~", "```"},
		{"different character tildes", "~~~markdown", "```", "~~~"},
		{"closing with text", "```markdown", "``` example", "```"},
		{"longer closing fence", "````markdown", "```", "`````   "},
		{"unclosed fence", "````markdown", "```", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			example := "# Spec\n" + tc.opening + "\n" + tc.embedded + "\n## Testcases\n- Example action → example result\n## Notes\nExample notes\n" + tc.closing + "\n"
			checked := dto.TestCase{ID: 1, ChangeID: 12, Scenario: "Real action → real result", Done: true, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0)}
			api := &syncAPI{rows: []dto.TestCase{checked}}
			require.ErrorContains(t, Synchronize(context.Background(), api, 12, example), "testcase error")
			require.Empty(t, api.calls)
			require.Equal(t, []dto.TestCase{checked}, api.rows)

			// A matching closer must allow the real section to be recognized.
			if tc.closing == "" {
				return
			}
			require.NoError(t, Synchronize(context.Background(), api, 12, example+"## Testcases\n- "+checked.Scenario))
			require.Equal(t, []string{"list"}, api.calls)
			require.Equal(t, []dto.TestCase{checked}, api.rows)
		})
	}
}

func Test032ExactMatchesAndDuplicateOccurrencesPreserveCheckedRecords(t *testing.T) {
	q, r := "Q → result", "R → result"
	checked := dto.TestCase{ID: 2, ChangeID: 12, Scenario: q, Done: true, CreatedAt: time.Unix(1, 0), UpdatedAt: time.Unix(2, 0)}
	unchecked := dto.TestCase{ID: 1, ChangeID: 12, Scenario: q}
	for _, tc := range []struct {
		name             string
		rows             []dto.TestCase
		bullets          string
		keep             []int
		creates, deletes int
	}{
		{"matching", []dto.TestCase{checked}, "- " + q, []int{2}, 0, 0},
		{"remove absent", []dto.TestCase{checked, {ID: 3, ChangeID: 12, Scenario: r}}, "- " + q, []int{2}, 0, 1},
		{"unmatched", nil, "- " + q, nil, 1, 0},
		{"similar", []dto.TestCase{checked}, "- Q → another result", nil, 1, 1},
		{"checked before unchecked", []dto.TestCase{unchecked, checked}, "- " + q, []int{2}, 0, 1},
		{"duplicate spec", []dto.TestCase{checked}, "- " + q + "\n- " + q, []int{2}, 1, 0},
		{"three copies", []dto.TestCase{unchecked, checked, {ID: 3, ChangeID: 12, Scenario: q, Done: true}}, "- " + q, []int{2}, 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &syncAPI{rows: slices.Clone(tc.rows), next: 100}
			require.NoError(t, Synchronize(context.Background(), a, 12, specCases(tc.bullets)))
			require.Equal(t, 12, a.change)
			creates, deletes := 0, 0
			for _, call := range a.calls {
				if len(call) > 7 && call[:7] == "create:" {
					creates++
				}
				if len(call) > 7 && call[:7] == "delete:" {
					deletes++
				}
			}
			require.Equal(t, tc.creates, creates)
			require.Equal(t, tc.deletes, deletes)
			for _, id := range tc.keep {
				found := false
				for _, row := range a.rows {
					if row.ID == id {
						require.Equal(t, checked, row)
						found = true
					}
				}
				require.True(t, found)
			}
			for _, row := range a.rows {
				require.Equal(t, 12, row.ChangeID)
				if row.ID > 100 {
					require.False(t, row.Done)
				}
			}
		})
	}
}

func Test032SynchronizationErrorsExposePartialPersistence(t *testing.T) {
	for _, tc := range []struct {
		fail      string
		want      []string
		remaining int
	}{
		{"list", []string{"list"}, 2},
		{"delete:1", []string{"list", "delete:1"}, 2},
		{"delete:2", []string{"list", "delete:1", "delete:2"}, 1},
		{"create:Q → R", []string{"list", "delete:1", "delete:2", "create:Q → R"}, 0},
	} {
		t.Run(tc.fail, func(t *testing.T) {
			a := &syncAPI{rows: []dto.TestCase{{ID: 1, ChangeID: 12, Scenario: "old"}, {ID: 2, ChangeID: 12, Scenario: "old2"}}, fail: tc.fail}
			err := Synchronize(context.Background(), a, 12, specCases("- Q → R"))
			require.Error(t, err)
			var syncErr *SyncError
			require.ErrorAs(t, err, &syncErr)
			require.Contains(t, err.Error(), "spec saved")
			if tc.fail == "delete:2" {
				require.Contains(t, err.Error(), "1 successful mutations (partial persistence)")
			}
			require.Equal(t, tc.want, a.calls)
			require.Len(t, a.rows, tc.remaining)
		})
	}
	for _, row := range []dto.TestCase{{ID: 1, ChangeID: 99, Scenario: "other"}, {ID: 0, ChangeID: 12, Scenario: "bad"}} {
		a := &syncAPI{rows: []dto.TestCase{row}}
		require.ErrorContains(t, Synchronize(context.Background(), a, 12, specCases("- Q → R")), "identity or owner")
		require.Equal(t, []string{"list"}, a.calls)
	}
}

func Test032CanceledSyncDoesNotReadOrMutateCases(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	api := &syncAPI{rows: []dto.TestCase{{ID: 1, ChangeID: 12, Scenario: "old"}}}
	err := Synchronize(ctx, api, 12, specCases("- Q → R"))
	require.ErrorIs(t, err, context.Canceled)
	var syncErr *SyncError
	require.ErrorAs(t, err, &syncErr)
	require.Empty(t, api.calls)
	require.Len(t, api.rows, 1)
}

func Test032MalformedSpecDoesNotInvalidateUnchangedCachedCases(t *testing.T) {
	api := &syncAPI{rows: []dto.TestCase{{ID: 1, ChangeID: 12, Scenario: "old"}}}
	err := Synchronize(context.Background(), api, 12, "# Saved without testcases")
	require.ErrorContains(t, err, "testcase error")
	var syncErr *SyncError
	require.False(t, errors.As(err, &syncErr))
	require.Empty(t, api.calls)
}
