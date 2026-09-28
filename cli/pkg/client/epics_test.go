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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const epicJSON = `{"id":3,"project_id":7,"name":"Epic Three","done_tc":2,"total_tc":8,"completed":63,"change_count":4,"created_at":"2026-09-28T10:00:00Z","updated_at":"2026-09-28T11:00:00Z"}`

func epicFixture() map[string]any {
	var e map[string]any
	_ = json.Unmarshal([]byte(epicJSON), &e)
	return e
}

func TestP301EpicRoutesShapesAndExactlyOneOperation(t *testing.T) {
	expected := dto.Epic{ID: 3, ProjectID: 7, Name: "Epic Three", DoneTC: 2, TotalTC: 8, Completed: 63, ChangeCount: 4, CreatedAt: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 9, 28, 11, 0, 0, 0, time.UTC)}
	for _, tc := range []struct {
		route, payload, response string
		status                   int
		call                     func(HTTPClient) error
	}{
		{"list", `{"project_id":7}`, "[" + epicJSON + "]", 200, func(c HTTPClient) error {
			rows, err := c.ListEpics(context.Background(), 7)
			assert.Equal(t, []dto.Epic{expected}, rows)
			return err
		}},
		{"details", `{"id":3}`, epicJSON, 200, func(c HTTPClient) error {
			e, err := c.GetEpic(context.Background(), 3)
			assert.Equal(t, expected, e)
			return err
		}},
		{"create", `{"project_id":7,"name":" /cancel\n\t"}`, `{"id":3}`, 201, func(c HTTPClient) error {
			id, err := c.CreateEpic(context.Background(), 7, " /cancel\n\t")
			assert.Equal(t, 3, id)
			return err
		}},
		{"update", `{"id":3,"name":" /cancel\n\t"}`, "", 204, func(c HTTPClient) error { return c.UpdateEpic(context.Background(), 3, " /cancel\n\t") }},
		{"delete", `{"id":3}`, "", 204, func(c HTTPClient) error { return c.DeleteEpic(context.Background(), 3) }},
	} {
		t.Run(tc.route, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				assert.Equal(t, "POST", r.Method)
				assert.Equal(t, "/api/v1/epic/"+tc.route, r.URL.Path)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				assert.JSONEq(t, tc.payload, string(body))
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.response)
			}))
			defer server.Close()
			require.NoError(t, tc.call(NewHTTPClient(server.URL)))
			assert.Equal(t, 1, calls)
		})
	}
}

func TestP301EpicRequiredFieldsAndMalformedResponses(t *testing.T) {
	cases := []string{"null", "{}", "[]", "[null]", `{"epics":[]}`, epicJSON + " {}", strings.Replace(epicJSON, `"id":3`, `"id":"3"`, 1), strings.Replace(epicJSON, "2026-09-28T10:00:00Z", "yesterday", 1), strings.Replace(epicJSON, `"id":3`, `"id":4`, 1)}
	for key := range epicFixture() {
		for _, mode := range []string{"missing", "null", "wrong type"} {
			e := epicFixture()
			switch mode {
			case "missing":
				delete(e, key)
			case "null":
				e[key] = nil
			default:
				e[key] = []int{}
			}
			body, err := json.Marshal(e)
			require.NoError(t, err)
			cases = append(cases, string(body))
		}
	}
	for _, body := range cases {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, body) }))
			defer server.Close()
			_, err := NewHTTPClient(server.URL).GetEpic(context.Background(), 3)
			var contract *ContractError
			require.ErrorAs(t, err, &contract)
			var response *HTTPError
			require.ErrorAs(t, err, &response)
			assert.Equal(t, 200, response.Status)
			assert.NotNil(t, errors.Unwrap(contract))
		})
	}
	for _, body := range []string{"null", "[null]", "[{}]", `{"epics":[]}`, "[" + strings.Replace(epicJSON, `"project_id":7`, `"project_id":8`, 1) + "]"} {
		t.Run("list "+body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, body) }))
			defer server.Close()
			_, err := NewHTTPClient(server.URL).ListEpics(context.Background(), 7)
			var contract *ContractError
			require.ErrorAs(t, err, &contract)
		})
	}
	for _, body := range []string{"null", "{}", `{"id":0}`, `{"id":null}`, `{"id":"3"}`} {
		t.Run("create "+body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201); _, _ = io.WriteString(w, body) }))
			defer server.Close()
			_, err := NewHTTPClient(server.URL).CreateEpic(context.Background(), 7, "name")
			var contract *ContractError
			require.ErrorAs(t, err, &contract)
			var response *HTTPError
			require.ErrorAs(t, err, &response)
			assert.Equal(t, 201, response.Status)
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "[]") }))
	defer server.Close()
	rows, err := NewHTTPClient(server.URL).ListEpics(context.Background(), 7)
	require.NoError(t, err)
	assert.Empty(t, rows)
	assert.NotNil(t, rows)
}

func TestP301EpicFailuresStatusesCancellationAndInvalidIDs(t *testing.T) {
	operations := []func(HTTPClient, context.Context, int) error{
		func(c HTTPClient, ctx context.Context, id int) error { _, err := c.ListEpics(ctx, id); return err },
		func(c HTTPClient, ctx context.Context, id int) error { _, err := c.GetEpic(ctx, id); return err },
		func(c HTTPClient, ctx context.Context, id int) error {
			_, err := c.CreateEpic(ctx, id, "name")
			return err
		},
		func(c HTTPClient, ctx context.Context, id int) error { return c.UpdateEpic(ctx, id, "name") },
		func(c HTTPClient, ctx context.Context, id int) error { return c.DeleteEpic(ctx, id) },
	}
	for index, call := range operations {
		for _, status := range []int{200, 201, 204, 400, 404, 500} {
			expected := []int{200, 200, 201, 204, 204}[index]
			if status == expected {
				continue
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = io.WriteString(w, "backend rejected")
			}))
			err := call(NewHTTPClient(server.URL), context.Background(), 7)
			var response *HTTPError
			require.ErrorAs(t, err, &response)
			assert.Equal(t, status, response.Status)
			server.Close()
		}
		c := NewHTTPClient("http://example.invalid")
		ctx, cancel := context.WithCancel(context.Background())
		c.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			deadline, ok := r.Context().Deadline()
			assert.True(t, ok)
			assert.LessOrEqual(t, time.Until(deadline), 15*time.Second)
			cancel()
			<-r.Context().Done()
			return nil, r.Context().Err()
		})}
		require.ErrorIs(t, call(c, ctx, 7), context.Canceled)
		for _, id := range []int{0, -1} {
			require.Error(t, call(c, context.Background(), id))
		}
	}
}
