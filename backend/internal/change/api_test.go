package change

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemovedChangeRoutes(t *testing.T) {
	e := echo.New()
	NewAPI(e, nil)
	for _, path := range []string{"assign-flow", "start-run", "update-run", "reset-claim", "update-def"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/change/"+path, strings.NewReader(`{}`)))
			assert.Equal(t, http.StatusNotFound, rec.Code)
		})
	}
}

func TestChangeAPIContracts(t *testing.T) {
	cases := []struct {
		path   string
		body   string
		status int
	}{
		{"list", `{"project_id":1}`, http.StatusOK},
		{"get", `{"id":2}`, http.StatusOK},
		{"rendered-artifacts", `{"ids":[2]}`, http.StatusOK},
		{"create", `{"project_id":1,"title":"Title","brief":"Brief"}`, http.StatusCreated},
		{"update-epic", `{"id":2,"epic_id":3}`, http.StatusOK},
		{"update-phase", `{"id":2,"change_phase":"review"}`, http.StatusOK},
		{"update-open", `{"id":2,"open":false}`, http.StatusOK},
		{"update-change-types", `{"id":2,"change_types":["fix"]}`, http.StatusOK},
		{"update-title", `{"id":2,"title":"Title"}`, http.StatusOK},
		{"update-brief", `{"id":2,"brief":" Brief ","agent_edit":true}`, http.StatusOK},
		{"update-spec", `{"id":2,"spec":"Spec","agent_edit":false}`, http.StatusOK},
		{"update-pr", `{"id":2,"pr":"PR","agent_edit":false}`, http.StatusOK},
		{"update-pr-url", `{"id":2,"pr_url":"https://example.test/pr"}`, http.StatusOK},
		{"delete", `{"id":2}`, http.StatusNoContent},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			for _, input := range []struct {
				name   string
				body   string
				err    error
				status int
			}{
				{"success", tc.body, nil, tc.status},
				{"invalid JSON", `{`, nil, http.StatusBadRequest},
				{"repository error", tc.body, ErrNotFound, http.StatusNotFound},
			} {
				t.Run(input.name, func(t *testing.T) {
					repo := &fakeChangeRepository{err: input.err, availableTypes: []string{"fix"}}
					e := echo.New()
					NewAPI(e, NewService(repo, Renderer{}))
					req := httptest.NewRequest(http.MethodPost, "/api/v1/change/"+tc.path, strings.NewReader(input.body))
					req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
					rec := httptest.NewRecorder()
					e.ServeHTTP(rec, req)
					require.Equal(t, input.status, rec.Code, rec.Body.String())
					if tc.path == "update-brief" && input.err == nil && input.status == http.StatusOK {
						assert.Equal(t, "Brief", repo.updateBriefReq.Brief)
						require.NotNil(t, repo.updateBriefReq.AgentEdit)
						assert.True(t, *repo.updateBriefReq.AgentEdit)
						var body map[string]any
						require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
						assert.Equal(t, "Brief", body["brief"])
						for _, removed := range []string{"def", "agent_edit", "flow_stages", "flow_stage_modes", "run_claim_id", "run_flow_stage", "run_task_step", "run_task_status", "run_error", "run_is_completed", "run_started_at", "run_updated_at"} {
							assert.NotContains(t, body, removed)
						}
					}
				})
			}
		})
	}
}

func TestChangeError(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{ErrInvalidInput, http.StatusBadRequest},
		{ErrInvalidReference, http.StatusBadRequest},
		{ErrNotFound, http.StatusNotFound},
	} {
		assert.Equal(t, tc.status, echo.StatusCode(changeError(tc.err)))
	}
	err := errors.New("database unavailable")
	assert.ErrorIs(t, changeError(err), err)
}
