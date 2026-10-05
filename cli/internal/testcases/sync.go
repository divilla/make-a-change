package testcases

import (
	"cli/internal/dto"
	"context"
	"fmt"
	"slices"
	"strings"
)

// SyncAPI exposes only the testcase operations used after spec saves.
type SyncAPI interface {
	ListTestCases(context.Context, int) ([]dto.TestCase, error)
	CreateTestCase(context.Context, int, string) (int, error)
	DeleteTestCase(context.Context, int) error
}

// SyncError means synchronization may have reached the API and cached rows may be stale,
// even when a failed request did not report whether its write was committed.
type SyncError struct{ Err error }

func (e *SyncError) Error() string { return e.Err.Error() }

func (e *SyncError) Unwrap() error { return e.Err }

// ParseSpecCases validates the entire section before any persistence is attempted.
func ParseSpecCases(spec string) ([]string, error) {
	var scenarios []string
	inside, found := false, false
	var fenceCharacter byte
	fenceLength := 0
	for _, raw := range strings.Split(strings.ReplaceAll(spec, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if !inside {
			if len(line) >= 3 && (line[0] == '`' || line[0] == '~') {
				character := line[0]
				length := len(line) - len(strings.TrimLeft(line, string(character)))
				if fenceLength == 0 && length >= 3 {
					fenceCharacter, fenceLength = character, length
				} else if character == fenceCharacter && length >= fenceLength && strings.TrimSpace(line[length:]) == "" {
					fenceLength = 0
				}
			}
			if fenceLength == 0 && line == "## Testcases" {
				inside, found = true, true
			}
			continue
		}
		if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "# ") {
			break
		}
		if line == "" {
			continue
		}
		if raw != strings.TrimLeft(raw, " \t") || !strings.HasPrefix(line, "- ") {
			return nil, fmt.Errorf("testcase error: expected single-line action → expected result bullet")
		}
		scenario := strings.TrimSpace(strings.TrimPrefix(line, "- "))
		action, expected, ok := strings.Cut(scenario, "→")
		if !ok || strings.TrimSpace(action) == "" || strings.TrimSpace(expected) == "" {
			return nil, fmt.Errorf("testcase error: bullet requires action and expected result")
		}
		scenarios = append(scenarios, scenario)
	}
	if !found || len(scenarios) == 0 {
		return nil, fmt.Errorf("testcase error: Testcases section is missing or empty")
	}
	return scenarios, nil
}

// Synchronize preserves exact matches, preferring checked copies for duplicate occurrences.
func Synchronize(ctx context.Context, api SyncAPI, change int, spec string) (err error) {
	scenarios, err := ParseSpecCases(spec)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = &SyncError{Err: err}
		}
	}()
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("spec saved; testcase synchronization cancelled: %w", err)
	}
	rows, err := api.ListTestCases(ctx, change)
	if err != nil {
		return fmt.Errorf("spec saved; testcase listing failed: %w", err)
	}
	// Validate every identity before a deletion can affect another change.
	for _, row := range rows {
		if row.ID <= 0 || row.ChangeID != change {
			return fmt.Errorf("spec saved; invalid testcase identity or owner")
		}
	}
	slices.SortStableFunc(rows, func(a, b dto.TestCase) int {
		if a.Done == b.Done {
			return 0
		}
		if a.Done {
			return -1
		}
		return 1
	})
	needed := map[string]int{}
	for _, scenario := range scenarios {
		needed[scenario]++
	}
	mutations := 0
	fail := func(err error) error {
		return fmt.Errorf("spec saved; testcase synchronization failed after %d successful mutations (partial persistence): %w", mutations, err)
	}
	for _, row := range rows {
		if needed[row.Scenario] > 0 {
			needed[row.Scenario]--
			continue
		}
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		if err := api.DeleteTestCase(ctx, row.ID); err != nil {
			return fail(err)
		}
		mutations++
	}
	for _, scenario := range scenarios {
		if needed[scenario] == 0 {
			continue
		}
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		if _, err := api.CreateTestCase(ctx, change, scenario); err != nil {
			return fail(err)
		}
		needed[scenario]--
		mutations++
	}
	return nil
}
