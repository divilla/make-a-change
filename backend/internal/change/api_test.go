package change

import (
	"context"
	"encoding/json"
	"errors"
	"mch_api/internal/app"
	"mch_api/internal/domain"
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
	for _, path := range []string{"documents", "set-document", "update-brief", "update-spec", "update-pr", "rendered-artifacts", "get", "assign-flow", "start-run", "update-run", "reset-claim", "update-def", "update-agent-edit", "update-def-agent-edit"} {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/change/"+path, strings.NewReader(`{}`)))
		require.Equal(t, 404, rec.Code)
	}
}

func TestChangeAPIContracts(t *testing.T) {
	mutations := map[string]struct {
		calls   []string
		request any
	}{
		"update-after-change": {[]string{"UpdateAfterChange"}, domain.ChangeUpdateAfterChangeRequest{ID: 7, AfterChangeID: intPtr(4)}},
		"update-epic":         {[]string{"Project", "EpicProject", "UpdateEpic"}, domain.ChangeUpdateEpicRequest{ID: 7, EpicID: intPtr(4)}},
		"update-phase":        {[]string{"Project", "UpdatePhase"}, domain.ChangeUpdatePhaseRequest{ID: 7, ChangePhase: "review"}},
		"update-open":         {[]string{"UpdateOpen"}, domain.ChangeUpdateOpenRequest{ID: 7, Open: boolPtr(false)}},
		"update-types":        {[]string{"Project", "UpdateTypes"}, domain.ChangeUpdateTypesRequest{ID: 7, ChangeTypes: []string{"fix"}}},
		"update-title":        {[]string{"Exists", "UpdateTitle"}, domain.ChangeUpdateTitleRequest{ID: 7, Title: "Title"}},
		"update-pr-url":       {[]string{"UpdatePRUrl"}, domain.ChangeUpdatePRUrlRequest{ID: 7, PRUrl: "https://pr"}},
	}
	cases := []struct {
		path, body string
		status     int
	}{
		{"list", `{"project_id":9}`, 200},
		{"details", `{"id":7}`, 200},
		{"create", `{"project_id":9,"title":"Title","brief":"Brief"}`, 201},
		{"update-after-change", `{"id":7,"after_change_id":4}`, 204},
		{"update-epic", `{"id":7,"epic_id":4}`, 204},
		{"update-phase", `{"id":7,"change_phase":"review"}`, 204},
		{"update-open", `{"id":7,"open":false}`, 204},
		{"update-types", `{"id":7,"change_types":["fix"]}`, 204},
		{"update-title", `{"id":7,"title":"Title"}`, 204},
		{"update-pr-url", `{"id":7,"pr_url":"https://pr"}`, 204},
		{"delete", `{"id":7}`, 204},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			inputs := []struct {
				name, body string
				err        error
				status     int
			}{{"success", tc.body, nil, tc.status}, {"malformed", "{", nil, 400}, {"wrong type", `{"id":[],"project_id":[],"ids":"bad"}`, nil, 400}, {"invalid", `{"id":-1,"project_id":-1,"ids":[0]}`, nil, 400}, {"missing", tc.body, app.ErrChangeNotFound, 404}, {"failure", tc.body, errors.New("private details"), 500}}
			for _, input := range inputs {
				t.Run(input.name, func(t *testing.T) {
					repo := &fakeChangeRepository{projectID: 9, epicProjectID: 9, err: input.err, details: domain.ChangeDetails{ChangeListItem: domain.ChangeListItem{ID: 7, ChangeTypes: []string{}}}}
					mutation, isMutation := mutations[tc.path]
					e := echo.New()
					config := defaultConfig()
					NewAPI(e, NewService(repo, config))
					var returned error
					e.HTTPErrorHandler = func(c *echo.Context, err error) {
						returned = err
						echo.DefaultHTTPErrorHandler(false)(c, err)
					}
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					req := httptest.NewRequest("POST", "/api/v1/change/"+tc.path, strings.NewReader(input.body)).WithContext(ctx)
					req.Header.Set("Content-Type", "application/json")
					rec := httptest.NewRecorder()
					e.ServeHTTP(rec, req)
					if input.err != nil {
						require.ErrorIs(t, returned, input.err)
					}
					if isMutation && input.body == tc.body {
						if input.err == nil {
							require.Equal(t, mutation.calls, repo.calls)
							require.Equal(t, mutation.request, repo.requests[len(repo.requests)-1])
						} else {
							require.Equal(t, mutation.calls[:1], repo.calls)
						}
						for _, got := range repo.contexts {
							require.Same(t, ctx, got)
						}
						for _, got := range config.contexts {
							require.Same(t, ctx, got)
						}
					}
					require.Equal(t, input.status, rec.Code, rec.Body.String())
					if input.status == 204 {
						require.Zero(t, rec.Body.Len())
					}
					if input.name == "success" {
						switch tc.path {
						case "create":
							require.JSONEq(t, `{"id":2}`, rec.Body.String())
						case "details":
							var body map[string]any
							require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
							keys := []string{}
							for k := range body {
								keys = append(keys, k)
							}
							require.ElementsMatch(t, []string{"id", "ref_uuid", "ref", "slug", "project_id", "change_phase", "change_types", "epic_id", "epic_name", "title", "open", "done_tc", "total_tc", "completed", "updated_at", "pr_url", "created_at", "after_change_id"}, keys)
							for _, k := range []string{"ref", "slug", "epic_id", "epic_name"} {
								require.Nil(t, body[k])
							}
						case "list":
							require.JSONEq(t, `[]`, rec.Body.String())
						}
					}
					if input.status == 500 {
						require.JSONEq(t, `{"message":"Internal Server Error"}`, rec.Body.String())
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
		{app.ErrChangeInvalidInput, http.StatusBadRequest},
		{app.ErrChangeInvalidReference, http.StatusBadRequest},
		{app.ErrChangeNotFound, http.StatusNotFound},
	} {
		assert.Equal(t, tc.status, echo.StatusCode(app.HTTP(tc.err)))
	}
	err := errors.New("database unavailable")
	assert.ErrorIs(t, app.HTTP(err), err)
}

func TestChangeHandlerReturnCauses(t *testing.T) {
	cause := errors.New("private database details")
	for _, tc := range []struct {
		err     error
		code    int
		message string
	}{
		{cause, 500, "Internal Server Error"},
		{app.Wrap(app.ErrChangeNotFound, "nested"), 404, "change not found"},
		{app.ErrChangeInvalidReference, 400, "invalid change reference"},
	} {
		e := echo.New()
		a := NewAPI(e, NewService(&fakeChangeRepository{err: tc.err}, nil))
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"id":7}`))
		req.Header.Set("Content-Type", "application/json")
		err := a.details(e.NewContext(req, httptest.NewRecorder()))
		require.ErrorIs(t, err, tc.err)
		var he *echo.HTTPError
		require.ErrorAs(t, err, &he)
		require.Equal(t, tc.code, he.Code)
		require.Equal(t, tc.message, he.Message)
	}
	e := echo.New()
	a := NewAPI(e, nil)
	req := httptest.NewRequest("POST", "/", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	err := a.details(e.NewContext(req, httptest.NewRecorder()))
	var he *echo.HTTPError
	require.ErrorAs(t, err, &he)
	require.NotNil(t, errors.Unwrap(he))
	require.Equal(t, "invalid change details payload", he.Message)
}
