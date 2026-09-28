// Package client provides the backend HTTP adapter used by the CLI.
package client

import (
	"bytes"
	"cli/internal/dto"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HTTPClient calls the Project Manager backend over HTTP.
type HTTPClient struct {
	BaseURL string
	Client  *http.Client
}

// NewHTTPClient creates an HTTP backend client for a base URL.
func NewHTTPClient(baseURL string) HTTPClient {
	return HTTPClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateTestCase creates a test case for a change and returns refreshed change data.
func (c HTTPClient) CreateTestCase(changeID int, scenario string) (dto.ChangeView, error) {
	if changeID <= 0 {
		return dto.ChangeView{}, fmt.Errorf("change ID must be a valid positive number")
	}
	return c.postChange("/api/v1/test-case/create", map[string]any{
		"change_id": changeID,
		"scenario":  scenario,
	})
}

// UpdateTestCase updates a test case scenario and returns refreshed change data.
func (c HTTPClient) UpdateTestCase(id int, scenario string) (dto.ChangeView, error) {
	if id <= 0 {
		return dto.ChangeView{}, fmt.Errorf("test case ID must be a valid positive number")
	}
	return c.postChange("/api/v1/test-case/update", map[string]any{
		"id":       id,
		"scenario": scenario,
	})
}

// UpdateTestCaseDone updates the test case done flag and returns refreshed change data.
func (c HTTPClient) UpdateTestCaseDone(id int, done bool) (dto.ChangeView, error) {
	if id <= 0 {
		return dto.ChangeView{}, fmt.Errorf("test case ID must be a valid positive number")
	}
	return c.postChange("/api/v1/test-case/update-done", map[string]any{
		"id":   id,
		"done": done,
	})
}

// DeleteTestCase deletes a test case and returns refreshed change data.
func (c HTTPClient) DeleteTestCase(id int) (dto.ChangeView, error) {
	if id <= 0 {
		return dto.ChangeView{}, fmt.Errorf("test case ID must be a valid positive number")
	}
	return c.postChange("/api/v1/test-case/delete", map[string]any{"id": id})
}

func (c HTTPClient) postChange(path string, payload any) (dto.ChangeView, error) {
	data, err := c.postJSON(path, payload)
	if err != nil {
		return dto.ChangeView{}, err
	}
	change, ok := findChange(data)
	if !ok {
		return dto.ChangeView{}, fmt.Errorf("change response missing change")
	}
	return change, nil
}

func (c HTTPClient) postJSON(path string, payload any) (any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var data map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
			if message := firstString(data, "message", "error"); message != "" {
				return nil, fmt.Errorf("%s", message)
			}
		}
		return nil, fmt.Errorf("backend returned %s", resp.Status)
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var data any
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data, nil
}

func findChange(value any) (dto.ChangeView, bool) {
	switch typed := value.(type) {
	case []any:
		changes := changesFromArray(typed)
		if len(changes) > 0 {
			return changes[0], true
		}
	case map[string]any:
		for _, key := range []string{"change", "data", "result"} {
			if candidate, ok := typed[key]; ok {
				if change, found := findChange(candidate); found {
					if testCases := findTestCases(typed); len(testCases) > 0 {
						change.TestCases = testCases
					}
					return change, true
				}
			}
		}
		change := changeFromMap(typed)
		if change.ID != "" || change.Title != "" {
			if testCases := findTestCases(typed); len(testCases) > 0 {
				change.TestCases = testCases
			}
			return change, true
		}
		for _, candidate := range typed {
			if change, found := findChange(candidate); found {
				return change, true
			}
		}
	}
	return dto.ChangeView{}, false
}

func findTestCases(value any) []dto.TestCase {
	switch typed := value.(type) {
	case []any:
		return testCasesFromArray(typed)
	case map[string]any:
		for key, candidate := range typed {
			if key == "test_cases" {
				if list, ok := candidate.([]any); ok {
					return testCasesFromArray(list)
				}
			}
		}
	}
	return nil
}

func changesFromArray(values []any) []dto.ChangeView {
	changes := make([]dto.ChangeView, 0, len(values))
	for _, value := range values {
		typed, ok := value.(map[string]any)
		if !ok {
			continue
		}
		change := changeFromMap(typed)
		if change.ID == "" && change.Title == "" {
			continue
		}
		changes = append(changes, change)
	}
	return changes
}

func testCasesFromArray(values []any) []dto.TestCase {
	testCases := make([]dto.TestCase, 0, len(values))
	for _, value := range values {
		typed, ok := value.(map[string]any)
		if !ok {
			continue
		}
		testCase := dto.TestCase{
			ID:       firstString(typed, "id", "test_case_id"),
			Scenario: firstString(typed, "scenario"),
			Done:     firstBool(typed, "done"),
			ChangeID: firstString(typed, "change_id"),
		}
		if testCase.ID == "" && testCase.Scenario == "" {
			continue
		}
		testCases = append(testCases, testCase)
	}
	return testCases
}

func changeFromMap(values map[string]any) dto.ChangeView {
	return dto.ChangeView{
		ID:          firstString(values, "id", "change_id"),
		RefUUID:     firstString(values, "ref_uuid"),
		Ref:         firstString(values, "ref"),
		Slug:        firstString(values, "slug"),
		ProjectID:   firstString(values, "project_id"),
		EpicID:      firstString(values, "epic_id"),
		EpicName:    firstString(values, "epic_name", "epic_title", "epic"),
		ChangePhase: firstString(values, "change_phase", "phase"),
		ChangeTypes: firstStringSlice(values, "change_types", "types"),
		Title:       firstString(values, "title", "name"),
		Brief:       firstString(values, "brief"),
		Spec:        firstString(values, "spec"),
		PR:          firstString(values, "pr"),
		PRUrl:       firstString(values, "pr_url"),

		Open:      firstBool(values, "open"),
		Done:      int64(firstInt(values, "done_tc", "done")),
		Total:     int64(firstInt(values, "total_tc", "total")),
		Completed: int64(firstInt(values, "completed", "completed_pct")),
		Created:   firstString(values, "created", "created_at"),
		Modified:  firstString(values, "modified", "updated", "updated_at"),
	}
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			return typed
		case float64:
			return strconv.FormatInt(int64(typed), 10)
		case int:
			return strconv.Itoa(typed)
		}
	}
	return ""
}

func firstInt(values map[string]any, keys ...string) int {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return int(typed)
		case int:
			return typed
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(typed))
			if err == nil {
				return parsed
			}
		}
	}
	return 0
}

func firstStringSlice(values map[string]any, keys ...string) []string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case []any:
			items := make([]string, 0, len(typed))
			for _, item := range typed {
				switch value := item.(type) {
				case string:
					items = append(items, value)
				case float64:
					items = append(items, strconv.FormatInt(int64(value), 10))
				}
			}
			return items
		case []string:
			return append([]string(nil), typed...)
		case string:
			if strings.TrimSpace(typed) == "" {
				return nil
			}
			return strings.Split(typed, "|")
		}
	}
	return nil
}

func firstBool(values map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := values[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			return typed == "true"
		}
	}
	return false
}
