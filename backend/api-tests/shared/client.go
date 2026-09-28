// Package shared provides the HTTP client and fixture cleanup for legacy API tests.
package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const apiTestBaseURL = "http://localhost:19080"

// Client sends requests to the backend owned by the API test run.
type Client struct {
	baseURL string
	http    *http.Client
}

type cleanupChange struct {
	ID int `json:"id"`
}

// NewClient checks backend connectivity and creates a client with a five-second timeout.
func NewClient(t *testing.T) *Client {
	t.Helper()

	client := &Client{
		baseURL: apiTestBaseURLFromEnv(),
		http: &http.Client{
			Timeout: 5 * time.Second,
		},
	}

	req, err := http.NewRequest(http.MethodGet, client.baseURL+"/api/v1/health", nil)
	require.NoError(t, err)

	res, err := client.http.Do(req)
	require.NoErrorf(t, err, "backend is not available at %s", client.baseURL)
	defer closeResponseBody(t, res.Body, http.MethodGet, "/api/v1/health")

	return client
}

// BaseURL returns the API base URL used by the test client.
func (c *Client) BaseURL() string {
	return c.baseURL
}

func apiTestBaseURLFromEnv() string {
	if value := os.Getenv("API_TEST_BASE_URL"); value != "" {
		return value
	}

	return apiTestBaseURL
}

// CleanupProject deletes a test project's changes and epics before the project.
func CleanupProject(t *testing.T, client *Client, projectID int) {
	t.Helper()

	var changes []cleanupChange
	status := client.Post(t, "/api/v1/change/list", map[string]any{"project_id": projectID}, &changes)
	if status == http.StatusOK {
		for _, change := range changes {
			status = client.Post(t, "/api/v1/change/delete", map[string]any{"id": change.ID}, nil)
			assert.Contains(t, []int{http.StatusNoContent, http.StatusNotFound}, status)
		}
	}

	var epics []struct {
		ID int `json:"id"`
	}
	status = client.Post(t, "/api/v1/epic/list", map[string]any{"project_id": projectID}, &epics)
	if status == http.StatusOK {
		for _, epic := range epics {
			status = client.Post(t, "/api/v1/epic/delete", map[string]any{"id": epic.ID}, nil)
			assert.Contains(t, []int{http.StatusNoContent, http.StatusNotFound, http.StatusConflict}, status)
		}
	}

	status = client.Post(t, "/api/v1/project/delete", map[string]any{"id": projectID}, nil)
	assert.Contains(t, []int{http.StatusNoContent, http.StatusNotFound}, status)
}

// Get returns the response status and decodes JSON into out when it is nonnil.
func (c *Client) Get(t *testing.T, path string, out any) int {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	require.NoError(t, err)

	res, err := c.http.Do(req)
	require.NoError(t, err)
	defer closeResponseBody(t, res.Body, http.MethodGet, path)

	if out != nil {
		require.NoError(t, json.NewDecoder(res.Body).Decode(out))
	}

	return res.StatusCode
}

// Post sends a JSON body, returns the status, and decodes JSON into nonnil out.
func (c *Client) Post(t *testing.T, path string, body any, out any) int {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	require.NoError(t, err)
	defer closeResponseBody(t, res.Body, http.MethodPost, path)

	if out != nil {
		data, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, out), fmt.Sprintf("response body: %s", data))
	}

	return res.StatusCode
}

func closeResponseBody(t assert.TestingT, body io.Closer, method, path string) {
	assert.NoErrorf(t, body.Close(), "close %s %s response body", method, path)
}
