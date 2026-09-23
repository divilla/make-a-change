package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type APIClient struct {
	baseURL string
	http    *http.Client
}

type Project struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Epic struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type ReferenceOption struct {
	Slug string `json:"slug"`
}

type ChangeReferences struct {
	Types []ReferenceOption `json:"types"`
}

type ChangeCreateInput struct {
	ProjectID   int      `json:"project_id"`
	EpicID      *int     `json:"epic_id"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	ChangePhase string   `json:"change_phase"`
	ChangeTypes []string `json:"change_types"`
}

type Change struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
}

func newAPIClient(baseURL string) *APIClient {
	return &APIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *APIClient) ListProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	err := c.post(ctx, "/api/v1/project/list", map[string]any{}, &projects)
	return projects, err
}

func (c *APIClient) ListEpics(ctx context.Context, projectID int) ([]Epic, error) {
	var epics []Epic
	err := c.post(ctx, "/api/v1/epic/list", map[string]any{"project_id": projectID}, &epics)
	return epics, err
}

func (c *APIClient) ChangeReferences(ctx context.Context) (ChangeReferences, error) {
	var refs ChangeReferences
	err := c.post(ctx, "/api/v1/change/reference", map[string]any{}, &refs)
	return refs, err
}

func (c *APIClient) CreateChange(ctx context.Context, input ChangeCreateInput) (Change, error) {
	var change Change
	err := c.post(ctx, "/api/v1/change/create", input, &change)
	return change, err
}

func (c *APIClient) post(ctx context.Context, path string, body any, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("backend %s returned %d: %s", path, res.StatusCode, strings.TrimSpace(string(data)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(data, out)
}
