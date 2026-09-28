// Package client provides the backend HTTP adapter used by the CLI.
package client

import (
	"context"
	"errors"
	"net/http"
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
	return HTTPClient{BaseURL: baseURL, Client: &http.Client{Timeout: 15 * time.Second}}
}

// CreateTestCase returns the committed testcase ID. The caller refreshes reads separately.
func (c HTTPClient) CreateTestCase(ctx context.Context, changeID int, scenario string) (int, error) {
	if changeID <= 0 || strings.TrimSpace(scenario) == "" {
		return 0, errors.New("valid change ID and scenario required")
	}
	return c.insertID(ctx, "/api/v1/test-case/create", struct {
		ChangeID int    `json:"change_id"`
		Scenario string `json:"scenario"`
	}{changeID, scenario})
}

// UpdateTestCase changes only the scenario.
func (c HTTPClient) UpdateTestCase(ctx context.Context, id int, scenario string) error {
	if id <= 0 || strings.TrimSpace(scenario) == "" {
		return errors.New("valid testcase ID and scenario required")
	}
	return c.projectRequest(ctx, "/api/v1/test-case/update", struct {
		ID       int    `json:"id"`
		Scenario string `json:"scenario"`
	}{id, scenario}, http.StatusNoContent, nil)
}

// UpdateTestCaseDone sets the explicit completion value.
func (c HTTPClient) UpdateTestCaseDone(ctx context.Context, id int, done bool) error {
	if id <= 0 {
		return errors.New("testcase ID must be positive")
	}
	return c.projectRequest(ctx, "/api/v1/test-case/update-done", struct {
		ID   int  `json:"id"`
		Done bool `json:"done"`
	}{id, done}, http.StatusNoContent, nil)
}

// DeleteTestCase removes one testcase.
func (c HTTPClient) DeleteTestCase(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("testcase ID must be positive")
	}
	return c.projectRequest(ctx, "/api/v1/test-case/delete", struct {
		ID int `json:"id"`
	}{id}, http.StatusNoContent, nil)
}
