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

type Client struct {
	baseURL string
	http    *http.Client
}

type cleanupChange struct {
	ID int `json:"id"`
}

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
	defer res.Body.Close()

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

func (c *Client) Get(t *testing.T, path string, out any) int {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	require.NoError(t, err)

	res, err := c.http.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	if out != nil {
		require.NoError(t, json.NewDecoder(res.Body).Decode(out))
	}

	return res.StatusCode
}

func (c *Client) Post(t *testing.T, path string, body any, out any) int {
	t.Helper()

	payload, err := json.Marshal(body)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	if out != nil {
		data, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(data, out), fmt.Sprintf("response body: %s", data))
	}

	return res.StatusCode
}
