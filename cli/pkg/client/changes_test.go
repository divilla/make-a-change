package client

import (
	"cli/internal/dto"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func changeFixture() map[string]any {
	return map[string]any{"id": 12, "project_id": 7, "ref_uuid": "uuid", "ref": int32(42), "slug": "stored-slug", "epic_id": nil, "epic_name": nil, "change_phase": "backlog", "change_types": []string{}, "title": "Title", "open": false, "done_tc": int64(1 << 34), "total_tc": int64(1 << 35), "completed": int64(73), "updated_at": "2026-09-28T11:00:00Z", "after_change_id": nil, "pr_url": "", "created_at": "2026-09-28T10:00:00Z"}
}

func TestP401AllChangeOperationsExactTypedPayloadsAndOneRequest(t *testing.T) {
	ctx := context.Background()
	association := 3
	tests := []struct {
		name   string
		input  string
		status int
		call   func(HTTPClient) error
	}{
		{"list", `{"project_id":7}`, 200, func(c HTTPClient) error {
			rows, e := c.ListChangeRows(ctx, 7)
			if e == nil {
				require.Len(t, rows, 1)
				require.Equal(t, int64(1<<34), rows[0].DoneTC)
				require.Equal(t, int64(73), rows[0].Completed)
				require.Equal(t, int32(42), *rows[0].Ref)
				require.Nil(t, rows[0].EpicID)
			}
			return e
		}},
		{"details", `{"id":12}`, 200, func(c HTTPClient) error {
			v, e := c.GetChange(ctx, 12)
			if e == nil {
				require.Equal(t, "stored-slug", *v.Slug)
				require.False(t, v.Open)
				require.Nil(t, v.AfterChangeID)
				require.Equal(t, "", v.PRUrl)
				require.False(t, v.CreatedAt.IsZero())
			}
			return e
		}},
		{"create", `{"project_id":7,"title":"Title","brief":"plain brief"}`, 201, func(c HTTPClient) error {
			id, e := c.CreateChange(ctx, dto.ChangeCreateInput{ProjectID: 7, Title: "Title", Brief: "plain brief"})
			require.Equal(t, 12, id)
			return e
		}},
		{"update-title", `{"id":12,"title":"/save"}`, 204, func(c HTTPClient) error { return c.UpdateChangeTitle(ctx, 12, "/save") }},
		{"update-types", `{"id":12,"change_types":[]}`, 204, func(c HTTPClient) error { return c.UpdateChangeTypes(ctx, 12, nil) }},
		{"update-phase", `{"id":12,"change_phase":"review"}`, 204, func(c HTTPClient) error { return c.UpdateChangePhase(ctx, 12, "review") }},
		{"update-open", `{"id":12,"open":false}`, 204, func(c HTTPClient) error { return c.UpdateChangeOpen(ctx, 12, false) }},
		{"update-epic", `{"id":12,"epic_id":3}`, 204, func(c HTTPClient) error { return c.UpdateChangeEpic(ctx, 12, &association) }},
		{"update-epic", `{"id":12,"epic_id":null}`, 204, func(c HTTPClient) error { return c.UpdateChangeEpic(ctx, 12, nil) }},
		{"update-after-change", `{"id":12,"after_change_id":3}`, 204, func(c HTTPClient) error { return c.UpdateChangeAfterChange(ctx, 12, &association) }},
		{"update-after-change", `{"id":12,"after_change_id":null}`, 204, func(c HTTPClient) error { return c.UpdateChangeAfterChange(ctx, 12, nil) }},
		{"update-pr-url", `{"id":12,"pr_url":"https://example.test/1"}`, 204, func(c HTTPClient) error { return c.UpdateChangePRUrl(ctx, 12, "https://example.test/1") }},
		{"delete", `{"id":12}`, 204, func(c HTTPClient) error { return c.DeleteChange(ctx, 12) }},
	}
	for _, tt := range tests {
		t.Run(tt.name+tt.input, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/v1/change/"+tt.name, r.URL.Path)
				var body json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.JSONEq(t, tt.input, string(body))
				w.WriteHeader(tt.status)
				if tt.status == 200 {
					if tt.name == "list" {
						_ = json.NewEncoder(w).Encode([]any{changeFixture()})
					} else {
						_ = json.NewEncoder(w).Encode(changeFixture())
					}
				}
				if tt.status == 201 {
					_, _ = w.Write([]byte(`{"id":12}`))
				}
			}))
			defer server.Close()
			require.NoError(t, tt.call(NewHTTPClient(server.URL)))
			require.Equal(t, 1, calls)
		})
	}
}

func TestP401MalformedChangeFieldsAndStatusCauses(t *testing.T) {
	for field, value := range changeFixture() {
		for _, mode := range []string{"missing", "wrong", "null"} {
			t.Run(field+mode, func(t *testing.T) {
				body := changeFixture()
				switch mode {
				case "missing":
					delete(body, field)
				case "wrong":
					body[field] = map[string]int{"wrong": 1}
				case "null":
					body[field] = nil
				}
				nullable := field == "ref" || field == "slug" || field == "epic_id" || field == "epic_name" || field == "after_change_id"
				_ = value
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _ = json.NewEncoder(w).Encode(body) }))
				defer server.Close()
				_, err := NewHTTPClient(server.URL).GetChange(context.Background(), 12)
				if mode == "null" && nullable {
					require.NoError(t, err)
					return
				}
				var status *HTTPError
				var contract *ContractError
				require.ErrorAs(t, err, &status)
				require.Equal(t, 200, status.Status)
				require.ErrorAs(t, err, &contract)
			})
		}
	}
	for _, body := range []string{`null`, `{}`, `{"data":[]}`, `[{}]`, `[null]`, `[{"id":"12"}]`, `[] []`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
			defer server.Close()
			_, err := NewHTTPClient(server.URL).ListChangeRows(context.Background(), 7)
			require.Error(t, err)
		})
	}
	for _, status := range []int{201, 204, 400, 404, 409, 500} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
		_, err := NewHTTPClient(server.URL).GetChange(context.Background(), 12)
		var h *HTTPError
		require.ErrorAs(t, err, &h)
		require.Equal(t, status, h.Status)
		server.Close()
	}
}

func TestP401ChangeCancellationAndUUIDOmission(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := NewHTTPClient("http://unused.test")
	_, err := c.GetChange(ctx, 12)
	require.ErrorIs(t, err, context.Canceled)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var v dto.ChangeCreateInput
		require.NoError(t, json.NewDecoder(r.Body).Decode(&v))
		require.Equal(t, "0198a86f-9b8a-7d89-ae5b-6f25b528b04c", v.RefUUID)
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":12}`))
	}))
	defer server.Close()
	_, err = NewHTTPClient(server.URL).CreateChange(context.Background(), dto.ChangeCreateInput{ProjectID: 7, RefUUID: "0198a86f-9b8a-7d89-ae5b-6f25b528b04c"})
	require.NoError(t, err)
	require.Equal(t, 15*time.Second, c.Client.Timeout)
}

func TestP406DocumentCurrentInsertAndSeparateTestCases(t *testing.T) {
	calls := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		switch r.URL.Path {
		case "/api/v1/doc/current":
			require.Equal(t, map[string]any{"ref_id": float64(12), "ref_table": "change"}, body)
			_ = json.NewEncoder(w).Encode([]dto.Document{{ID: 91, RefID: 12, RefTable: "change", DocType: "brief", Body: "raw\tbytes\n", Current: true, CreatedAt: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), HTML: "<p>raw</p>"}})
		case "/api/v1/doc/insert":
			require.Equal(t, map[string]any{"ref_id": float64(12), "ref_table": "change", "doc_type": "brief", "body": "raw\tbytes\n", "agent_edit": false}, body)
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"id":92}`))
		case "/api/v1/test-case/list":
			require.Equal(t, map[string]any{"change_id": float64(12)}, body)
			_, _ = w.Write([]byte(`[{"id":31,"change_id":12,"scenario":"literal","done":false,"created_at":"2026-09-28T10:00:00Z","updated_at":"2026-09-28T11:00:00Z"}]`))
		default:
			t.Fatalf("unexpected %s", r.URL.Path)
		}
	}))
	defer server.Close()
	c := NewHTTPClient(server.URL)
	docs, err := c.CurrentDocuments(context.Background(), 12, "change")
	require.NoError(t, err)
	require.Equal(t, "raw\tbytes\n", docs[0].Body)
	id, err := c.InsertDocument(context.Background(), dto.DocumentInput{RefID: 12, RefTable: "change", DocType: "brief", Body: docs[0].Body})
	require.NoError(t, err)
	require.Equal(t, 92, id)
	rows, err := c.ListTestCases(context.Background(), 12)
	require.NoError(t, err)
	require.Equal(t, 31, rows[0].ID)
	require.Len(t, calls, 3)
}

func TestP401InvalidIDsNeverRequest(t *testing.T) {
	c := NewHTTPClient(":bad")
	ctx := context.Background()
	negative := -1
	_, e := c.ListChangeRows(ctx, 0)
	require.Error(t, e)
	_, e = c.GetChange(ctx, 0)
	require.Error(t, e)
	_, e = c.CreateChange(ctx, dto.ChangeCreateInput{})
	require.Error(t, e)
	for _, e := range []error{c.UpdateChangeTitle(ctx, 0, "a"), c.UpdateChangePhase(ctx, 0, "a"), c.UpdateChangeTypes(ctx, 0, nil), c.UpdateChangeOpen(ctx, 0, false), c.UpdateChangeEpic(ctx, 0, nil), c.UpdateChangeAfterChange(ctx, 0, nil), c.UpdateChangePRUrl(ctx, 0, ""), c.DeleteChange(ctx, 0), c.UpdateChangeEpic(ctx, 1, &negative), c.UpdateChangeAfterChange(ctx, 1, &negative)} {
		require.Error(t, e)
		var h *HTTPError
		require.False(t, errors.As(e, &h))
	}
}

func TestP406MalformedDocumentAndTestcaseReads(t *testing.T) {
	for _, path := range []string{"/api/v1/doc/current", "/api/v1/test-case/list"} {
		for _, body := range []string{`null`, `{}`, `[null]`, `[{}]`, `[{"id":-1}]`} {
			t.Run(path+body, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, path, r.URL.Path)
					_, _ = w.Write([]byte(body))
				}))
				defer server.Close()
				c := NewHTTPClient(server.URL)
				var err error
				if path == "/api/v1/doc/current" {
					_, err = c.CurrentDocuments(context.Background(), 12, "change")
				} else {
					_, err = c.ListTestCases(context.Background(), 12)
				}
				var h *HTTPError
				var contract *ContractError
				require.ErrorAs(t, err, &h)
				require.Equal(t, 200, h.Status)
				require.ErrorAs(t, err, &contract)
			})
		}
	}
}
