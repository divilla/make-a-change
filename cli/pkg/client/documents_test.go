package client

import (
	"cli/internal/dto"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const documentTime = `"2026-09-28T10:00:00Z"`

func documentJSON(id int, table string, owner int, _ bool) string {
	return fmt.Sprintf(`{"id":%d,"ref_id":%d,"ref_table":%q,"doc_type":"spec","body":" /save\\nraw ","agent_edit":false,"deleted_at":%s,"created_at":%s,"updated_at":%s,"html":"<p>server</p>"}`, id, owner, table, "null", documentTime, documentTime)
}

func TestP601DocumentRoutesPayloadStatusAndOneRequest(t *testing.T) {
	const wide = 9007199254740993
	tests := []struct {
		route, payload string
		status         int
		body           string
		call           func(HTTPClient) error
	}{
		{"list", `{"ref_id":12,"ref_table":"epic"}`, 200, "[]", func(c HTTPClient) error {
			r, e := c.ListDocuments(context.Background(), 12, "epic")
			require.Empty(t, r)
			return e
		}},
		{"list-active", `{"ref_id":12,"ref_table":"change"}`, 200, "[]", func(c HTTPClient) error {
			r, e := c.ActiveDocuments(context.Background(), 12, "change")
			require.Empty(t, r)
			return e
		}},
		{"details", fmt.Sprintf(`{"id":%d}`, wide), 200, documentJSON(wide, "project", 7, false), func(c HTTPClient) error {
			r, e := c.DocumentDetails(context.Background(), wide)
			require.Equal(t, wide, r.ID)
			return e
		}},
		{"insert", `{"ref_id":12,"ref_table":"change","doc_type":"spec","body":" /save\nraw ","agent_edit":false}`, 201, fmt.Sprintf(`{"id":%d}`, wide), func(c HTTPClient) error {
			id, e := c.InsertDocument(context.Background(), dto.DocumentInput{RefID: 12, RefTable: "change", DocType: "spec", Body: " /save\nraw ", AgentEdit: false})
			require.Equal(t, wide, id)
			return e
		}},
	}
	for _, tt := range tests {
		t.Run(tt.route, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, "POST", r.Method)
				require.Equal(t, "/api/v1/doc/"+tt.route, r.URL.Path)
				var raw json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
				require.JSONEq(t, tt.payload, string(raw))
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			require.NoError(t, tt.call(NewHTTPClient(server.URL)))
			require.Equal(t, 1, calls)
		})
	}
}

func TestP601FullRowsHistoryCurrentAndWideIDs(t *testing.T) {
	const wide = 9007199254740993
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/list") {
			_, _ = fmt.Fprintf(w, "[%s,%s]", documentJSON(wide+1, "change", 12, true), documentJSON(wide, "change", 12, false))
		} else {
			_, _ = fmt.Fprintf(w, "[%s]", documentJSON(wide+1, "change", 12, true))
		}
	}))
	defer server.Close()
	c := NewHTTPClient(server.URL)
	rows, err := c.ListDocuments(context.Background(), 12, "change")
	require.NoError(t, err)
	require.Len(t, rows, 2)
	require.Equal(t, wide+1, rows[0].ID)
	require.Equal(t, wide, rows[1].ID)
	require.Nil(t, rows[1].DeletedAt)
	require.False(t, rows[1].AgentEdit)
	require.Equal(t, " /save\\nraw ", rows[1].Body)
	require.Equal(t, "<p>server</p>", rows[1].HTML)
	require.Equal(t, time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC), rows[1].CreatedAt)
	require.Equal(t, rows[1].CreatedAt, rows[1].UpdatedAt)
	current, err := c.ActiveDocuments(context.Background(), 12, "change")
	require.NoError(t, err)
	require.Len(t, current, 1)
	require.Nil(t, current[0].DeletedAt)
}

func TestP601MalformedDocumentResponsesAndOwnerMismatch(t *testing.T) {
	valid := documentJSON(9, "change", 12, true)
	cases := []struct {
		body    string
		details bool
	}{
		{"null", false},
		{"{}", false},
		{`[{"id":9}]`, false},
		{"[" + documentJSON(9, "epic", 12, true) + "]", false},
		{"[" + documentJSON(9, "change", 13, true) + "]", false},
		{"[" + valid + "," + valid + "]", false},
		{"[" + documentJSON(8, "change", 12, true) + "," + valid + "]", false},
		{"[" + strings.Replace(valid, `"deleted_at":null`, `"deleted_at":"2026-09-28T11:00:00Z"`, 1) + "]", false},
		{documentJSON(10, "change", 12, true), true},
		{"null", true},
		{strings.Replace(valid, `"body":" /save\\nraw "`, `"body":null`, 1), true},
		{strings.Replace(valid, `"agent_edit":false`, `"agent_edit":"false"`, 1), true},
		{strings.Replace(valid, `"created_at":`+documentTime, `"created_at":null`, 1), true},
		{strings.Replace(valid, `"html":"<p>server</p>"`, `"html":null`, 1), true},
		{strings.TrimSuffix(valid, "}") + `,"surprise":1}`, true},
	}
	for _, tt := range cases {
		t.Run(tt.body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(tt.body)) }))
			defer server.Close()
			c := NewHTTPClient(server.URL)
			var err error
			if tt.details {
				_, err = c.DocumentDetails(context.Background(), 9)
			} else {
				_, err = c.ActiveDocuments(context.Background(), 12, "change")
			}
			var h *HTTPError
			require.ErrorAs(t, err, &h)
			require.Equal(t, 200, h.Status)
		})
	}
	for _, id := range []int{0, -1} {
		_, err := NewHTTPClient("http://unused").DocumentDetails(context.Background(), id)
		require.Error(t, err)
	}
	for _, table := range []string{"", "task"} {
		_, err := NewHTTPClient("http://unused").ListDocuments(context.Background(), 12, table)
		require.Error(t, err)
	}
}

func TestP601DocumentStatusCausesAndCancellation(t *testing.T) {
	for _, status := range []int{400, 404, 409, 500} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status); _, _ = w.Write([]byte("failure")) }))
		_, err := NewHTTPClient(server.URL).ListDocuments(context.Background(), 12, "change")
		server.Close()
		var h *HTTPError
		require.ErrorAs(t, err, &h)
		require.Equal(t, status, h.Status)
		require.Error(t, h.Cause)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	_, err := NewHTTPClient(server.URL).DocumentDetails(ctx, 9)
	require.ErrorIs(t, err, context.Canceled)
	var h *HTTPError
	require.ErrorAs(t, err, &h)
	require.Zero(t, h.Status)
	require.Zero(t, calls)
	_, err = NewHTTPClient(server.URL).InsertDocument(context.Background(), dto.DocumentInput{RefID: 12, RefTable: "change", DocType: "spec", Body: " "})
	require.Error(t, err)
	require.Zero(t, calls)
	for _, body := range []string{`{}`, `{"id":0}`, `{"id":1.5}`, `{"id":9,"unexpected":true}`} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201); _, _ = w.Write([]byte(body)) }))
		_, err := NewHTTPClient(s.URL).InsertDocument(context.Background(), dto.DocumentInput{RefID: 12, RefTable: "change", DocType: "spec", Body: "x"})
		s.Close()
		require.Error(t, err)
	}
	require.True(t, errors.Is(&HTTPError{Cause: context.Canceled}, context.Canceled))
}

func TestP601EveryRequiredFullRowFieldRejectsMissingOrWrongType(t *testing.T) {
	base := map[string]any{"id": 9, "ref_id": 12, "ref_table": "change", "doc_type": "spec", "body": "raw", "agent_edit": false, "deleted_at": nil, "created_at": "2026-09-28T10:00:00Z", "updated_at": "2026-09-28T11:00:00Z", "html": "<p>rendered</p>"}
	for _, field := range []string{"id", "ref_id", "ref_table", "doc_type", "body", "agent_edit", "created_at", "updated_at", "html"} {
		for _, mode := range []string{"missing", "null", "wrong type"} {
			t.Run(field+"/"+mode, func(t *testing.T) {
				row := map[string]any{}
				for k, v := range base {
					row[k] = v
				}
				switch mode {
				case "missing":
					delete(row, field)
				case "null":
					row[field] = nil
				case "wrong type":
					row[field] = []string{"unexpected"}
				}
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					require.NoError(t, json.NewEncoder(w).Encode([]any{row}))
				}))
				defer server.Close()
				_, err := NewHTTPClient(server.URL).ActiveDocuments(context.Background(), 12, "change")
				var h *HTTPError
				require.ErrorAs(t, err, &h)
				require.Equal(t, 200, h.Status)
			})
		}
	}
}
