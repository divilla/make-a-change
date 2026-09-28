package client

import (
	"cli/internal/dto"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func configFixture(slug string) dto.BackendConfig {
	return dto.BackendConfig{Slug: slug, ProjectDocs: []string{"readme", "notes"}, EpicDocs: []string{}, ChangeDocs: []string{"brief", "spec"}, ChangePhases: []string{"todo", "done"}, ChangeColors: []string{"12", "9"}, ChangeTypes: []string{"fix", "feature"}}
}

type testRoundTrip func(*http.Request) (*http.Response, error)

func (f testRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestP701ConfigRoutesPayloadsStatusesAndSingleRequest(t *testing.T) {
	row := configFixture("a")
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		switch r.URL.Path {
		case "/api/v1/config/list":
			assert.Empty(t, body)
			_ = json.NewEncoder(w).Encode([]dto.BackendConfig{row})
		case "/api/v1/config/details":
			assert.Equal(t, "a", body["slug"])
			_ = json.NewEncoder(w).Encode(row)
		case "/api/v1/config/insert":
			assert.Len(t, body, 7)
			assert.Equal(t, []any{}, body["epic_docs"])
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(map[string]string{"slug": "a"})
		case "/api/v1/config/update":
			assert.Len(t, body, 7)
			w.WriteHeader(204)
		case "/api/v1/config/delete":
			assert.Equal(t, map[string]any{"slug": "a"}, body)
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	c := NewHTTPClient(server.URL)
	rows, err := c.ListConfigurations(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []dto.BackendConfig{row}, rows)
	got, err := c.GetConfiguration(context.Background(), "a")
	require.NoError(t, err)
	assert.Equal(t, row, got)
	slug, err := c.InsertConfiguration(context.Background(), row)
	require.NoError(t, err)
	assert.Equal(t, "a", slug)
	require.NoError(t, c.UpdateConfiguration(context.Background(), row))
	require.NoError(t, c.DeleteConfiguration(context.Background(), "a"))
	assert.EqualValues(t, 5, calls.Load())
}

func TestP701SixArraysExplicitEmptyAndMalformedResponses(t *testing.T) {
	empty := configFixture("empty")
	empty.ProjectDocs = []string{}
	empty.EpicDocs = []string{}
	empty.ChangeDocs = []string{}
	empty.ChangePhases = []string{}
	empty.ChangeColors = []string{}
	empty.ChangeTypes = []string{}
	data, _ := json.Marshal(empty)
	for _, body := range []string{`null`, `{}`, strings.Replace(string(data), `"project_docs":[]`, `"project_docs":null`, 1), strings.Replace(string(data), `"project_docs":[]`, `"project_docs":[null]`, 1), strings.Replace(string(data), `"change_types":[]`, `"change_types":[" "]`, 1), strings.Replace(string(data), `"epic_docs":[]`, `"epic_docs":{}`, 1)} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
		_, err := NewHTTPClient(s.URL).GetConfiguration(context.Background(), "empty")
		require.Error(t, err, body)
		s.Close()
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) }))
	defer s.Close()
	got, err := NewHTTPClient(s.URL).GetConfiguration(context.Background(), "empty")
	require.NoError(t, err)
	assert.Equal(t, empty, got)
	orderedByBackend := []dto.BackendConfig{configFixture("a"), configFixture("B"), configFixture("ä")}
	s = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(orderedByBackend)
	}))
	rows, err := NewHTTPClient(s.URL).ListConfigurations(context.Background())
	require.NoError(t, err)
	assert.Equal(t, orderedByBackend, rows, "preserve the database collation order")
	s.Close()
	for _, duplicate := range [][]dto.BackendConfig{{configFixture("a"), configFixture("a")}, {configFixture("a"), configFixture("B"), configFixture("a")}} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(duplicate) }))
		_, err := NewHTTPClient(s.URL).ListConfigurations(context.Background())
		require.ErrorContains(t, err, "unique")
		s.Close()
	}
}

func TestP701ConfigErrorsAndCancellation(t *testing.T) {
	for _, status := range []int{400, 404, 409, 500} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			_, _ = w.Write([]byte("config is in use"))
		}))
		err := NewHTTPClient(s.URL).DeleteConfiguration(context.Background(), "in-use")
		var typed *HTTPError
		require.ErrorAs(t, err, &typed)
		assert.Equal(t, status, typed.Status)
		assert.Contains(t, typed.Error(), "config is in use")
		s.Close()
	}
	for _, op := range []string{"list", "details", "insert", "update", "delete"} {
		calls := 0
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			assert.Equal(t, "/api/v1/config/"+op, r.URL.Path)
			http.Error(w, "operation rejected", http.StatusConflict)
		}))
		c := NewHTTPClient(s.URL)
		var err error
		switch op {
		case "list":
			_, err = c.ListConfigurations(context.Background())
		case "details":
			_, err = c.GetConfiguration(context.Background(), "x")
		case "insert":
			_, err = c.InsertConfiguration(context.Background(), configFixture("x"))
		case "update":
			err = c.UpdateConfiguration(context.Background(), configFixture("x"))
		case "delete":
			err = c.DeleteConfiguration(context.Background(), "x")
		}
		var typed *HTTPError
		require.ErrorAs(t, err, &typed)
		assert.Equal(t, http.StatusConflict, typed.Status)
		assert.Equal(t, 1, calls)
		s.Close()
	}
	_, err := NewHTTPClient("http://unused.invalid").InsertConfiguration(context.Background(), dto.BackendConfig{Slug: " "})
	require.Error(t, err)
	started := make(chan struct{})
	release := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer s.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := NewHTTPClient(s.URL).ListConfigurations(ctx); done <- err }()
	<-started
	cancel()
	err = <-done
	close(release)
	var typed *HTTPError
	require.ErrorAs(t, err, &typed)
	assert.Equal(t, 0, typed.Status)
	assert.True(t, errors.Is(err, context.Canceled))
	for _, op := range []string{"update", "delete"} {
		c := NewHTTPClient("http://backend.test")
		c.Client.Transport = testRoundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader("unexpected")), Header: make(http.Header)}, nil
		})
		if op == "update" {
			err = c.UpdateConfiguration(context.Background(), configFixture("x"))
		} else {
			err = c.DeleteConfiguration(context.Background(), "x")
		}
		require.Error(t, err)
	}
}

func TestP701ConfigIdentityAndUnexpectedSuccessResponses(t *testing.T) {
	for _, op := range []string{"details", "insert"} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if op == "insert" {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"slug":"other"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(configFixture("other"))
		}))
		c := NewHTTPClient(s.URL)
		var err error
		if op == "insert" {
			_, err = c.InsertConfiguration(context.Background(), configFixture("wanted"))
		} else {
			_, err = c.GetConfiguration(context.Background(), "wanted")
		}
		var typed *HTTPError
		require.ErrorAs(t, err, &typed)
		assert.ErrorContains(t, err, "differs from request")
		s.Close()
	}
}

func TestP701BothHealthRoutesHealthyDegradedAndMalformed(t *testing.T) {
	var calls atomic.Int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, int64(0), r.ContentLength)
		if r.URL.Path == "/api/health" {
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"status":"degraded","api":"ok","database":"error","error":"database unavailable"}`))
		} else {
			_, _ = w.Write([]byte(`{"status":"ok","api":"ok","database":"ok"}`))
		}
	}))
	defer s.Close()
	c := NewHTTPClient(s.URL)
	healthy, err := c.CheckHealth(context.Background(), "/api/v1/health")
	require.NoError(t, err)
	assert.Equal(t, 200, healthy.HTTPStatus)
	degraded, err := c.CheckHealth(context.Background(), "/api/health")
	require.NoError(t, err)
	assert.Equal(t, 503, degraded.HTTPStatus)
	assert.Equal(t, "database unavailable", degraded.Error)
	assert.EqualValues(t, 2, calls.Load())
	for _, tc := range []struct {
		status int
		body   string
	}{{200, `{}`}, {200, `{"status":"degraded","api":"ok","database":"error","error":"database unavailable"}`}, {503, `{"status":"ok","api":"ok","database":"ok"}`}, {503, `{"status":"degraded","api":"ok","database":"error"}`}, {302, `{"status":"ok","api":"ok","database":"ok"}`}, {200, `{"status":"ok","api":"ok","database":"ok"} {}`}} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		}))
		_, err := NewHTTPClient(s.URL).CheckHealth(context.Background(), "/api/health")
		require.Error(t, err)
		var typed *HTTPError
		require.ErrorAs(t, err, &typed)
		assert.Equal(t, tc.status, typed.Status)
		s.Close()
	}
	_, err = c.CheckHealth(context.Background(), "/api/unsupported")
	require.Error(t, err)
	started := make(chan struct{})
	c.Client.Transport = testRoundTrip(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := c.CheckHealth(ctx, "/api/health"); done <- err }()
	<-started
	cancel()
	err = <-done
	var typed *HTTPError
	require.ErrorAs(t, err, &typed)
	assert.Zero(t, typed.Status)
	assert.True(t, errors.Is(err, context.Canceled))
}
