package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestP501TestCaseRoutesPayloadStatusAndOneRequest(t *testing.T) {
	routes := []struct {
		path, payload string
		status        int
		call          func(HTTPClient) error
	}{
		{"list", `{"change_id":12}`, 200, func(c HTTPClient) error {
			rows, err := c.ListTestCases(context.Background(), 12)
			require.Empty(t, rows)
			return err
		}},
		{"create", `{"change_id":12,"scenario":" /save "}`, 201, func(c HTTPClient) error {
			id, err := c.CreateTestCase(context.Background(), 12, " /save ")
			require.Equal(t, 31, id)
			return err
		}},
		{"update", `{"id":31,"scenario":"text"}`, 204, func(c HTTPClient) error { return c.UpdateTestCase(context.Background(), 31, "text") }},
		{"update-done", `{"id":31,"done":true}`, 204, func(c HTTPClient) error { return c.UpdateTestCaseDone(context.Background(), 31, true) }},
		{"update-done", `{"id":31,"done":false}`, 204, func(c HTTPClient) error { return c.UpdateTestCaseDone(context.Background(), 31, false) }},
		{"delete", `{"id":31}`, 204, func(c HTTPClient) error { return c.DeleteTestCase(context.Background(), 31) }},
	}
	for _, tt := range routes {
		t.Run(tt.path+tt.payload, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/api/v1/test-case/"+tt.path, r.URL.Path)
				var raw json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
				require.JSONEq(t, tt.payload, string(raw))
				w.WriteHeader(tt.status)
				if tt.status == 200 {
					_, _ = w.Write([]byte("[]"))
				}
				if tt.status == 201 {
					_, _ = w.Write([]byte(`{"id":31}`))
				}
			}))
			defer server.Close()
			require.NoError(t, tt.call(NewHTTPClient(server.URL)))
			require.Equal(t, 1, calls)
		})
	}
}

func TestP501ListFieldsOwnerOrderAndLargeIDs(t *testing.T) {
	const large = 9007199254740993
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, `[{"id":%d,"change_id":12,"scenario":"first","done":false,"created_at":"2026-09-28T10:00:00Z","updated_at":"2026-09-28T11:00:00Z"},{"id":%d,"change_id":12,"scenario":"second","done":true,"created_at":"2026-09-28T10:00:00Z","updated_at":"2026-09-28T11:00:00Z"}]`, large, large+1)
	}))
	defer server.Close()
	rows, err := NewHTTPClient(server.URL).ListTestCases(context.Background(), 12)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, 9007199254740993, rows[0].ID)
	require.Equal(t, 12, rows[0].ChangeID)
	require.Equal(t, "first", rows[0].Scenario)
	require.False(t, rows[0].Done)
	require.True(t, rows[1].Done)
	require.Equal(t, time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), rows[0].CreatedAt)
	require.Equal(t, time.Date(2026, 9, 28, 11, 0, 0, 0, time.UTC), rows[0].UpdatedAt)
}

func TestP501MalformedTestCaseResponsesAndStatusCauses(t *testing.T) {
	tests := []struct {
		body   string
		status int
	}{
		{`null`, 200},
		{`{}`, 200},
		{`[{"id":1}]`, 200},
		{`[{"id":1,"change_id":13,"scenario":"x","done":false,"created_at":"2026-09-28T10:00:00Z","updated_at":"2026-09-28T10:00:00Z"}]`, 200},
		{`[{"id":2,"change_id":12,"scenario":"x","done":false,"created_at":"2026-09-28T10:00:00Z","updated_at":"2026-09-28T10:00:00Z"},{"id":1,"change_id":12,"scenario":"x","done":false,"created_at":"2026-09-28T10:00:00Z","updated_at":"2026-09-28T10:00:00Z"}]`, 200},
		{`{"message":"missing"}`, 404},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status, tt.body), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			_, err := NewHTTPClient(server.URL).ListTestCases(context.Background(), 12)
			var h *HTTPError
			require.ErrorAs(t, err, &h)
			require.Equal(t, tt.status, h.Status)
			require.Error(t, h.Cause)
		})
	}
	for _, body := range []string{`{}`, `{"id":0}`, `{"id":null}`, `{"id":1.5}`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201); _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			_, err := NewHTTPClient(server.URL).CreateTestCase(context.Background(), 12, "x")
			var h *HTTPError
			require.ErrorAs(t, err, &h)
			require.Equal(t, 201, h.Status)
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("unexpected")) }))
	defer server.Close()
	err := NewHTTPClient(server.URL).DeleteTestCase(context.Background(), 31)
	var h *HTTPError
	require.ErrorAs(t, err, &h)
	require.Equal(t, 200, h.Status)
	malformed := NewHTTPClient("http://example.test")
	malformed.Client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 204, Body: io.NopCloser(strings.NewReader("body")), Header: make(http.Header)}, nil
	})}
	err = malformed.DeleteTestCase(context.Background(), 31)
	require.ErrorAs(t, err, &h)
	require.Equal(t, 204, h.Status)
	require.Error(t, NewHTTPClient(server.URL).UpdateTestCase(context.Background(), 0, "x"))
	require.Error(t, NewHTTPClient(server.URL).UpdateTestCase(context.Background(), 1, " "))
}

func TestP501CanceledTestCaseRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	_, err := NewHTTPClient(server.URL).CreateTestCase(ctx, 12, "x")
	require.ErrorIs(t, err, context.Canceled)
	var h *HTTPError
	require.ErrorAs(t, err, &h)
	require.Zero(t, h.Status)
	require.Zero(t, calls)
	require.True(t, errors.Is(err, context.Canceled))
}

func TestP501EachRequiredListFieldRejectsMissingOrInvalid(t *testing.T) {
	base := map[string]any{"id": 31, "change_id": 12, "scenario": "scenario", "done": false, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z"}
	for _, field := range []string{"id", "change_id", "scenario", "done", "created_at", "updated_at"} {
		for _, invalid := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-invalid-%t", field, invalid), func(t *testing.T) {
				row := map[string]any{}
				for k, v := range base {
					row[k] = v
				}
				if invalid {
					row[field] = nil
				} else {
					delete(row, field)
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					require.NoError(t, json.NewEncoder(w).Encode([]any{row}))
				}))
				defer server.Close()
				_, err := NewHTTPClient(server.URL).ListTestCases(context.Background(), 12)
				var h *HTTPError
				require.ErrorAs(t, err, &h)
				require.Equal(t, 200, h.Status)
			})
		}
	}
}
