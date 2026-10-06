package client

import (
	"cli/internal/dto"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test031DocumentWireContract(t *testing.T) {
	for _, deleted := range []string{"null", `"2026-09-28T12:00:00+02:00"`} {
		body := strings.Replace(documentJSON(9, "change", 12, false), `"deleted_at":null`, `"deleted_at":`+deleted, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, err := fmt.Fprint(w, body); require.NoError(t, err) }))
		row, err := NewHTTPClient(server.URL).DocumentDetails(context.Background(), 9)
		server.Close()
		require.NoError(t, err)
		require.Equal(t, deleted != "null", row.DeletedAt != nil)
		encoded, err := json.Marshal(row)
		require.NoError(t, err)
		require.NotContains(t, string(encoded), `"current"`)
	}
	for _, value := range []string{`{}`, `"bad"`, `1`} {
		body := strings.Replace(documentJSON(9, "change", 12, false), `"deleted_at":null`, `"deleted_at":`+value, 1)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, err := fmt.Fprint(w, body); require.NoError(t, err) }))
		_, err := NewHTTPClient(server.URL).DocumentDetails(context.Background(), 9)
		server.Close()
		require.Error(t, err)
	}
	body := strings.Replace(documentJSON(9, "change", 12, false), `,"deleted_at":null`, "", 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, err := fmt.Fprint(w, body); require.NoError(t, err) }))
	defer server.Close()
	_, err := NewHTTPClient(server.URL).DocumentDetails(context.Background(), 9)
	require.Error(t, err)
}

func Test031DocumentReadRoutesAndOwnerValidation(t *testing.T) {
	for _, route := range []string{"list", "list-active", "comment-list"} {
		t.Run(route, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/v1/doc/"+route, r.URL.Path)
				var in map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
				require.Equal(t, map[string]any{"ref_id": float64(12), "ref_table": "change"}, in)
				_, err := fmt.Fprint(w, "[]")
				require.NoError(t, err)
			}))
			defer server.Close()
			c := NewHTTPClient(server.URL)
			var err error
			switch route {
			case "list":
				_, err = c.ListDocuments(context.Background(), 12, "change")
			case "list-active":
				_, err = c.ActiveDocuments(context.Background(), 12, "change")
			case "comment-list":
				_, err = c.ListComments(context.Background(), 12, "change")
			}
			require.NoError(t, err)
		})
	}
	for _, body := range []string{"null", "[" + documentJSON(9, "project", 12, false) + "]", "[" + documentJSON(9, "change", 13, false) + "]", "[" + documentJSON(9, "change", 12, false) + "," + documentJSON(8, "change", 12, false) + "]"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, err := fmt.Fprint(w, body); require.NoError(t, err) }))
		_, err := NewHTTPClient(server.URL).ActiveDocuments(context.Background(), 12, "change")
		server.Close()
		require.Error(t, err)
	}
}

func Test031CommentRoutesPayloadsAndStatuses(t *testing.T) {
	tests := []struct {
		route, payload string
		status         int
		call           func(HTTPClient) error
	}{
		{"comment-insert", `{"ref_id":12,"ref_table":"change","body":" exact\nbody ","agent_edit":false}`, 201, func(c HTTPClient) error {
			id, err := c.InsertComment(context.Background(), 12, "change", " exact\nbody ")
			if err == nil {
				require.Equal(t, 91, id)
			}
			return err
		}},
		{"comment-update", `{"id":91,"body":""}`, 204, func(c HTTPClient) error { return c.UpdateComment(context.Background(), 91, "") }},
		{"comment-undelete", `{"id":91}`, 204, func(c HTTPClient) error { return c.UndeleteComment(context.Background(), 91) }},
		{"active-set", `{"id":91}`, 204, func(c HTTPClient) error { return c.ActivateDocument(context.Background(), 91) }},
		{"delete", `{"id":91}`, 204, func(c HTTPClient) error { return c.DeleteDocument(context.Background(), 91) }},
	}
	for _, tt := range tests {
		t.Run(tt.route, func(t *testing.T) {
			for _, status := range []int{tt.status, 400, 404, 500} {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls++
					require.Equal(t, "POST", r.Method)
					require.Equal(t, "/api/v1/doc/"+tt.route, r.URL.Path)
					var raw json.RawMessage
					require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
					require.JSONEq(t, tt.payload, string(raw))
					w.WriteHeader(status)
					if status == 201 {
						_, err := fmt.Fprint(w, `{"id":91}`)
						require.NoError(t, err)
					}
				}))
				err := tt.call(NewHTTPClient(server.URL))
				server.Close()
				require.Equal(t, 1, calls)
				if status == tt.status {
					require.NoError(t, err)
				} else {
					var httpErr *HTTPError
					require.ErrorAs(t, err, &httpErr)
					require.Equal(t, status, httpErr.Status)
				}
			}
		})
	}
	c := NewHTTPClient("http://unused")
	for _, id := range []int{0, -1} {
		require.Error(t, c.UpdateComment(context.Background(), id, ""))
		require.Error(t, c.DeleteDocument(context.Background(), id))
		require.Error(t, c.ActivateDocument(context.Background(), id))
		require.Error(t, c.UndeleteComment(context.Background(), id))
	}
	for _, table := range []string{"", "other"} {
		_, err := c.InsertComment(context.Background(), 12, table, "body")
		require.Error(t, err)
	}
	_, err := c.InsertComment(context.Background(), 12, "change", " \n")
	require.Error(t, err)
	_, err = c.InsertDocument(context.Background(), dto.DocumentInput{RefID: 12, RefTable: "change", DocType: "comment", Body: "body"})
	require.Error(t, err)
}

func Test031ProjectEpicChangeWireContracts(t *testing.T) {
	row := changeFixture()
	delete(row, "active")
	delete(row, "created_at")
	delete(row, "pr_url")
	delete(row, "after_change_id")
	delete(row, "after_change_name")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/change/list", r.URL.Path)
		var in map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&in))
		require.Equal(t, float64(7), in["project_id"])
		require.Equal(t, false, in["active"])
		require.NoError(t, json.NewEncoder(w).Encode([]any{row}))
	}))
	defer server.Close()
	rows, err := NewHTTPClient(server.URL).ListInactiveChanges(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	// Details still require the explicit active field, while list rows do not.
	var wire changeWire
	data, err := json.Marshal(row)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &wire))
	_, err = wire.value(true)
	require.Error(t, err)
	var p projectWire
	require.NoError(t, json.Unmarshal([]byte(projectJSON), &p))
	project, err := p.value()
	require.NoError(t, err)
	require.Equal(t, "custom", project.ConfigSlug)
	require.True(t, project.Active)
	var e epicWire
	require.NoError(t, json.Unmarshal([]byte(epicJSON), &e))
	epic, err := e.value()
	require.NoError(t, err)
	require.True(t, epic.Active)
	p.Active = nil
	_, err = p.value()
	require.Error(t, err)
	e.Active = nil
	_, err = e.value()
	require.Error(t, err)
}

func Test031UpdateActiveRouteAndFalse(t *testing.T) {
	for _, active := range []bool{true, false} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/api/v1/change/update-active", r.URL.Path)
			var raw map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
			require.Equal(t, map[string]any{"id": float64(12), "active": active}, raw)
			w.WriteHeader(204)
		}))
		require.NoError(t, NewHTTPClient(server.URL).UpdateChangeActive(context.Background(), 12, active))
		server.Close()
	}
}
