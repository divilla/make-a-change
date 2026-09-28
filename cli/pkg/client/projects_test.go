package client

import (
	"cli/internal/dto"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	projectJSON = `{"id":7,"name":"Seven","config":"custom","last_ref":42,"created_at":"2026-09-28T10:00:00Z","updated_at":"2026-09-28T11:00:00Z","change_count":3}`
	configJSON  = `{"slug":"custom","project_docs":["readme"],"epic_docs":["brief"],"change_docs":["brief","spec"],"change_phases":["todo","done"],"change_colors":["12"],"change_types":["fix","feature"]}`
)

func TestP201ProjectRoutesShapesAndExactlyOneRequest(t *testing.T) {
	cases := []struct {
		route, body, response string
		status                int
		call                  func(HTTPClient) error
	}{
		{"list", "{}", "[" + projectJSON + "]", 200, func(c HTTPClient) error {
			rows, err := c.ListProjectRows(context.Background())
			require.Len(t, rows, 1)
			p := rows[0]
			assert.Equal(t, 7, p.ID)
			assert.Equal(t, int32(42), p.LastRef)
			assert.Equal(t, "custom", p.Config)
			assert.Equal(t, 3, p.ChangeCount)
			assert.Equal(t, "Seven", p.Name)
			assert.Equal(t, 2026, p.CreatedAt.Year())
			assert.Equal(t, 11, p.UpdatedAt.Hour())
			return err
		}},
		{"details", `{"id":7}`, projectJSON, 200, func(c HTTPClient) error { _, err := c.GetProject(context.Background(), 7); return err }},
		{"create", `{"name":"New\nname"}`, `{"id":7}`, 201, func(c HTTPClient) error {
			id, err := c.CreateProject(context.Background(), "New\nname")
			assert.Equal(t, 7, id)
			return err
		}},
		{"update", `{"id":7,"name":"Edited"}`, "ignored body", 204, func(c HTTPClient) error { return c.UpdateProject(context.Background(), 7, "Edited") }},
		{"delete", `{"id":7}`, "", 204, func(c HTTPClient) error { return c.DeleteProject(context.Background(), 7) }},
		{"config", `{"id":7}`, configJSON, 200, func(c HTTPClient) error {
			cfg, err := c.GetProjectConfig(context.Background(), 7)
			assert.Equal(t, dto.ProjectConfig{Slug: "custom", ProjectDocs: []string{"readme"}, EpicDocs: []string{"brief"}, ChangeDocs: []string{"brief", "spec"}, ChangePhases: []string{"todo", "done"}, ChangeColors: []string{"12"}, ChangeTypes: []string{"fix", "feature"}}, cfg)
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.route, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/v1/project/"+tc.route, r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.JSONEq(t, tc.body, string(body))
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.response)
			}))
			defer server.Close()
			require.NoError(t, tc.call(NewHTTPClient(server.URL)))
			assert.Equal(t, 1, calls)
		})
	}
}

func TestP202ContractErrorsPreserveStatusAndCause(t *testing.T) {
	for _, tc := range []struct {
		route, body string
		status      int
	}{
		{"list", `{"projects":[]}`, 200},
		{"list", "null", 200},
		{"list", "[{}]", 200},
		{"list", "[null]", 200},
		{"details", "{}", 200},
		{"details", "null", 200},
		{"details", "[]", 200},
		{"details", strings.Replace(projectJSON, `"id":7`, `"id":"7"`, 1), 200},
		{"details", strings.Replace(projectJSON, `"id":7`, `"id":8`, 1), 200},
		{"details", strings.Replace(projectJSON, "2026-09-28T10:00:00Z", "bad", 1), 200},
		{"details", projectJSON + ` {}`, 200},
		{"details", `{`, 200},
		{"details", strings.Replace(projectJSON, `"name":"Seven",`, "", 1), 200},
		{"create", `{}`, 201},
		{"create", `{"project":{"id":7}}`, 201},
		{"create", `{"id":"7"}`, 201},
		{"config", strings.Replace(configJSON, `["readme"]`, `[null]`, 1), 200},
		{"config", `{}`, 200},
		{"config", strings.Replace(configJSON, `"project_docs":["readme"]`, `"project_docs":null`, 1), 200},
		{"config", strings.Replace(configJSON, `["readme"]`, `{}`, 1), 200},
	} {
		t.Run(tc.route+tc.body, func(t *testing.T) {
			c := HTTPClient{BaseURL: "http://test", Client: &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})}}
			var err error
			switch tc.route {
			case "list":
				_, err = c.ListProjectRows(context.Background())
			case "details":
				_, err = c.GetProject(context.Background(), 7)
			case "create":
				_, err = c.CreateProject(context.Background(), "name")
			case "config":
				_, err = c.GetProjectConfig(context.Background(), 7)
			}
			var contract *ContractError
			require.ErrorAs(t, err, &contract)
			require.NotNil(t, errors.Unwrap(contract))
			var status *HTTPError
			require.ErrorAs(t, err, &status)
			assert.Equal(t, tc.status, status.Status)
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type failingBody struct {
	io.Reader
	readErr, closeErr error
}

func (b failingBody) Read(p []byte) (int, error) {
	if b.readErr != nil {
		return 0, b.readErr
	}
	return b.Reader.Read(p)
}
func (b failingBody) Close() error { return b.closeErr }

func TestP202TransportFailuresCancellationAndFiniteDeadline(t *testing.T) {
	cause := errors.New("transport cause")
	for _, tc := range []struct {
		name    string
		status  int
		body    io.ReadCloser
		network error
	}{
		{"network", 0, nil, cause}, {"decode", 200, failingBody{readErr: cause}, nil}, {"close", 200, failingBody{Reader: strings.NewReader(projectJSON), closeErr: cause}, nil}, {"status read and close", 503, failingBody{readErr: cause, closeErr: cause}, nil}, {"empty close", 204, failingBody{closeErr: cause}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := HTTPClient{BaseURL: "http://test", Client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				deadline, ok := r.Context().Deadline()
				require.True(t, ok)
				assert.LessOrEqual(t, time.Until(deadline), 15*time.Second)
				if tc.network != nil {
					return nil, tc.network
				}
				return &http.Response{StatusCode: tc.status, Body: tc.body}, nil
			})}}
			var err error
			if tc.status == 204 {
				err = c.DeleteProject(context.Background(), 7)
			} else {
				_, err = c.GetProject(context.Background(), 7)
			}
			require.ErrorIs(t, err, cause)
			var status *HTTPError
			require.ErrorAs(t, err, &status)
			assert.Equal(t, tc.status, status.Status)
		})
	}
	c := NewHTTPClient(":invalid")
	_, err := c.GetProject(context.Background(), 7)
	require.Error(t, err)
	err = c.projectRequest(context.Background(), "/encode", make(chan int), 200, nil)
	require.ErrorContains(t, err, "unsupported type")
	started := make(chan struct{})
	finished := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
		close(finished)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := NewHTTPClient(server.URL).GetProject(ctx, 7); result <- err }()
	<-started
	cancel()
	require.ErrorIs(t, <-result, context.Canceled)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("HTTP work did not cancel")
	}
}

func TestP202EmptyListAndInvalidIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "[]") }))
	defer server.Close()
	c := NewHTTPClient(server.URL)
	c.Client = nil
	rows, err := c.ListProjectRows(context.Background())
	require.NoError(t, err)
	assert.Empty(t, rows)
	_, err = c.GetProject(context.Background(), 0)
	require.Error(t, err)
	_, err = c.GetProjectConfig(context.Background(), -1)
	require.Error(t, err)
	require.Error(t, c.UpdateProject(context.Background(), 0, "name"))
	require.Error(t, c.DeleteProject(context.Background(), 0))
}

func TestP202RedirectIsStatusErrorWithoutAlternateRequest(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Redirect(w, r, "/alternate", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	_, err := NewHTTPClient(server.URL).GetProject(context.Background(), 7)
	var status *HTTPError
	require.ErrorAs(t, err, &status)
	assert.Equal(t, 307, status.Status)
	assert.Equal(t, 1, calls)
}
