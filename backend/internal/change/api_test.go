package change

import (
	"encoding/json"
	"errors"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gookit/validate/v2"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemovedChangeRoutes(t *testing.T) {
	e := echo.New()
	NewAPI(e, nil)
	for _, path := range []string{"assign-flow", "start-run", "update-run", "reset-claim", "update-def", "update-agent-edit", "update-def-agent-edit"} {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest("POST", "/api/v1/change/"+path, strings.NewReader(`{}`)))
		require.Equal(t, 404, rec.Code)
	}
}

func TestChangeAPIContracts(t *testing.T) {
	cases := []struct {
		path, body string
		status     int
	}{
		{"list", `{"project_id":9}`, 200},
		{"get", `{"id":7}`, 200},
		{"documents", `{"id":7}`, 200},
		{"rendered-artifacts", `{"ids":[7]}`, 200},
		{"create", `{"project_id":9,"title":"Title","brief":"Brief"}`, 201},
		{"update-epic", `{"id":7,"epic_id":4}`, 204},
		{"update-phase", `{"id":7,"change_phase":"review"}`, 204},
		{"update-open", `{"id":7,"open":false}`, 204},
		{"update-change-types", `{"id":7,"change_types":["fix"]}`, 204},
		{"update-title", `{"id":7,"title":"Title"}`, 204},
		{"update-brief", `{"id":7,"brief":"Brief","agent_edit":false}`, 204},
		{"update-spec", `{"id":7,"spec":"Spec","agent_edit":true}`, 204},
		{"update-pr", `{"id":7,"pr":"PR","agent_edit":false}`, 204},
		{"update-pr-url", `{"id":7,"pr_url":"https://pr"}`, 204},
		{"set-document", `{"id":7,"doc_type":"plan","body":"Raw","agent_edit":false}`, 204},
		{"delete", `{"id":7}`, 204},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			inputs := []struct {
				name, body string
				err        error
				status     int
			}{{"success", tc.body, nil, tc.status}, {"malformed", "{", nil, 400}, {"wrong type", `{"id":[],"project_id":[],"ids":"bad"}`, nil, 400}, {"invalid", `{"id":-1,"project_id":-1,"ids":[0]}`, nil, 400}, {"missing", tc.body, apperror.ErrChangeNotFound, 404}, {"failure", tc.body, errors.New("private details"), 500}}
			for _, input := range inputs {
				t.Run(input.name, func(t *testing.T) {
					repo := &fakeChangeRepository{projectID: 9, epicProjectID: 9, err: input.err, details: domain.ChangeDetails{ChangeListItem: domain.ChangeListItem{ID: 7, ChangeTypes: []string{}}}}
					e := echo.New()
					NewAPI(e, NewService(repo, Renderer{}, defaultConfig()))
					req := httptest.NewRequest("POST", "/api/v1/change/"+tc.path, strings.NewReader(input.body))
					req.Header.Set("Content-Type", "application/json")
					rec := httptest.NewRecorder()
					e.ServeHTTP(rec, req)
					require.Equal(t, input.status, rec.Code, rec.Body.String())
					if input.status == 204 {
						require.Zero(t, rec.Body.Len())
					}
					if input.name == "success" {
						switch tc.path {
						case "create":
							require.JSONEq(t, `{"id":2}`, rec.Body.String())
						case "get":
							var body map[string]any
							require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
							keys := []string{}
							for k := range body {
								keys = append(keys, k)
							}
							require.ElementsMatch(t, []string{"id", "ref_uuid", "ref", "slug", "project_id", "change_phase", "change_types", "epic_id", "epic_name", "title", "open", "done_tc", "total_tc", "completed", "modified", "pr_url", "created"}, keys)
							for _, k := range []string{"ref", "slug", "epic_id", "epic_name"} {
								require.Nil(t, body[k])
							}
						case "list", "documents":
							require.JSONEq(t, `[]`, rec.Body.String())
						case "rendered-artifacts":
							require.JSONEq(t, `{"artifacts":[]}`, rec.Body.String())
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

func TestDocumentAPIShapeAndExplicitBooleans(t *testing.T) {
	e := echo.New()
	NewAPI(e, NewService(&fakeChangeRepository{docs: []domain.ChangeDocument{{ID: 8, DocType: "spec", Body: "raw", AgentEdit: false}}}, Renderer{}, defaultConfig()))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/change/documents", strings.NewReader(`{"id":7}`))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(rec, req)
	require.Equal(t, 200, rec.Code)
	var docs []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &docs))
	require.Len(t, docs, 1)
	keys := []string{}
	for k := range docs[0] {
		keys = append(keys, k)
	}
	require.ElementsMatch(t, []string{"id", "doc_type", "body", "agent_edit", "created", "html"}, keys)
	for _, path := range []string{"update-open", "update-brief", "update-spec", "update-pr", "set-document"} {
		for _, value := range []string{"null", `"false"`} {
			body := `{"id":7,"open":` + value + `,"agent_edit":` + value + `,"brief":"Raw","spec":"Raw","pr":"Raw","doc_type":"spec","body":"Raw"}`
			req := httptest.NewRequest("POST", "/api/v1/change/"+path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			require.Equal(t, 400, rec.Code)
		}
	}
	// Validation retains both the module identity and the real field/rule cause.
	a := NewAPI(e, nil)
	req = httptest.NewRequest("POST", "/", strings.NewReader(`{"id":0}`))
	req.Header.Set("Content-Type", "application/json")
	err := a.getChange(e.NewContext(req, httptest.NewRecorder()))
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	var validation validate.Errors
	require.ErrorAs(t, err, &validation)
	require.NotEmpty(t, validation["id"]["required"])
	var he *echo.HTTPError
	require.ErrorAs(t, err, &he)
	require.NotNil(t, errors.Unwrap(he))
}

func TestChangeError(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
	}{
		{apperror.ErrChangeInvalidInput, http.StatusBadRequest},
		{apperror.ErrChangeInvalidReference, http.StatusBadRequest},
		{apperror.ErrChangeNotFound, http.StatusNotFound},
	} {
		assert.Equal(t, tc.status, echo.StatusCode(apperror.HTTP(tc.err)))
	}
	err := errors.New("database unavailable")
	assert.ErrorIs(t, apperror.HTTP(err), err)
}

func TestChangeHandlerReturnCauses(t *testing.T) {
	cause := errors.New("private database details")
	for _, tc := range []struct {
		err     error
		code    int
		message string
	}{
		{cause, 500, "Internal Server Error"},
		{apperror.Wrap(apperror.ErrChangeNotFound, "nested"), 404, "change not found"},
		{apperror.ErrChangeInvalidReference, 400, "invalid change reference"},
	} {
		e := echo.New()
		a := NewAPI(e, NewService(&fakeChangeRepository{err: tc.err}, Renderer{}, nil))
		req := httptest.NewRequest("POST", "/", strings.NewReader(`{"id":7}`))
		req.Header.Set("Content-Type", "application/json")
		err := a.getChange(e.NewContext(req, httptest.NewRecorder()))
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
	err := a.getChange(e.NewContext(req, httptest.NewRecorder()))
	var he *echo.HTTPError
	require.ErrorAs(t, err, &he)
	require.NotNil(t, errors.Unwrap(he))
	require.Equal(t, "invalid change get payload", he.Message)
}
