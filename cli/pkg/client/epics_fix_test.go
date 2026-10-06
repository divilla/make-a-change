package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test034ActivityRequestsAndScopedLists(t *testing.T) {
	for _, active := range []bool{true, false} {
		calls := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			switch r.URL.Path {
			case "/api/v1/change/list":
				require.Equal(t, map[string]any{"project_id": float64(7), "active": active}, body)
				row := changeFixture()
				delete(row, "active")
				require.NoError(t, json.NewEncoder(w).Encode([]any{row}))
			case "/api/v1/epic/list":
				require.Equal(t, map[string]any{"project_id": float64(7)}, body)
				a, b := epicFixture(), epicFixture()
				b["id"] = 4
				b["active"] = false
				require.NoError(t, json.NewEncoder(w).Encode([]any{a, b}))
			case "/api/v1/epic/update-active", "/api/v1/change/update-active":
				require.Equal(t, map[string]any{"id": float64(3), "active": active}, body)
				w.WriteHeader(204)
			default:
				t.Errorf("unexpected route %s", r.URL.Path)
				http.NotFound(w, r)
			}
		}))
		c := NewHTTPClient(server.URL)
		list := c.ListChangeRows
		if !active {
			list = c.ListInactiveChanges
		}
		rows, err := list(context.Background(), 7)
		if active {
			require.NoError(t, err)
			require.True(t, rows[0].Active)
		} else {
			require.NoError(t, err)
			require.False(t, rows[0].Active)
		}
		epics, err := c.ListEpics(context.Background(), 7)
		require.NoError(t, err)
		require.Len(t, epics, 2)
		require.False(t, epics[1].Active)
		require.NoError(t, c.UpdateEpicActive(context.Background(), 3, active))
		require.NoError(t, c.UpdateChangeActive(context.Background(), 3, active))
		require.Error(t, c.UpdateEpicActive(context.Background(), 0, active))
		require.Equal(t, 4, calls)
		server.Close()
	}
}
