package change_test

import (
	"aipm/api-tests/shared"
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChangeCreateIdentityContract(t *testing.T) {
	client := shared.NewClient(t)
	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	create := func(name string, ref any) (int, change) {
		t.Helper()
		payload := map[string]any{"project_id": projectID, "title": name, "brief": "# " + name}
		if ref != "omitted" {
			payload["ref_uuid"] = ref
		}
		var got change
		return client.Post(t, "/api/v1/change/create", payload, &got), got
	}

	suppliedUUID := uuid.Must(uuid.NewV7()).String()
	status, supplied := create("supplied identity", suppliedUUID)
	require.Equal(t, http.StatusCreated, status)
	assert.Equal(t, suppliedUUID, supplied.RefUUID)

	status, omitted := create("omitted identity", "omitted")
	require.Equal(t, http.StatusCreated, status)
	parsedOmitted, err := uuid.FromString(omitted.RefUUID)
	require.NoError(t, err)
	assert.Equal(t, byte(7), parsedOmitted.Version())

	status, explicitNull := create("null identity", nil)
	require.Equal(t, http.StatusCreated, status)
	parsedNull, err := uuid.FromString(explicitNull.RefUUID)
	require.NoError(t, err)
	assert.Equal(t, byte(7), parsedNull.Version())

	for _, invalid := range []string{"", "not-a-uuid"} {
		status, _ = create("invalid identity "+invalid, invalid)
		assert.Equal(t, http.StatusBadRequest, status)
	}

	status, _ = create("duplicate identity", suppliedUUID)
	assert.NotEqual(t, http.StatusCreated, status)

	var listed []change
	status = client.Post(t, "/api/v1/change/list", map[string]any{"project_id": projectID}, &listed)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listed, 3)
	identities := map[int]string{supplied.ID: suppliedUUID, omitted.ID: omitted.RefUUID, explicitNull.ID: explicitNull.RefUUID}
	for _, item := range listed {
		assert.Equal(t, identities[item.ID], item.RefUUID)
	}

	var updated change
	status = client.Post(t, "/api/v1/change/update-title", map[string]any{"id": supplied.ID, "title": "mutated title", "ref_uuid": omitted.RefUUID}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, suppliedUUID, updated.RefUUID)
	var fetched detail
	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": supplied.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, suppliedUUID, fetched.Change.RefUUID)
	assert.Equal(t, int16(0), fetched.Change.Version)
}

type project struct {
	ID      int   `json:"id"`
	LastRef int32 `json:"last_ref"`
}

type epic struct {
	ID        int   `json:"id"`
	DoneTC    int16 `json:"done_tc"`
	TotalTC   int16 `json:"total_tc"`
	Completed int16 `json:"completed"`
}

type changeOption struct {
	Slug  string `json:"slug"`
	Color string `json:"color"`
}

type change struct {
	ID          int      `json:"id"`
	Version     int16    `json:"version"`
	RefUUID     string   `json:"ref_uuid"`
	Ref         *int32   `json:"ref"`
	Slug        *string  `json:"slug"`
	ProjectID   int      `json:"project_id"`
	EpicID      *int     `json:"epic_id"`
	EpicName    *string  `json:"epic_name"`
	ChangePhase string   `json:"change_phase"`
	ChangeTypes []string `json:"change_types"`
	Title       string   `json:"title"`
	Brief       string   `json:"brief"`
	Spec        string   `json:"spec"`
	SpecHTML    string   `json:"spec_html"`
	PR          string   `json:"pr"`
	PRHtml      string   `json:"pr_html"`
	PRUrl       string   `json:"pr_url"`
	Open        bool     `json:"open"`
	DoneTC      int16    `json:"done_tc"`
	TotalTC     int16    `json:"total_tc"`
	Completed   int16    `json:"completed"`
}

type changeHistory struct {
	Version   int16
	DocType   string
	Body      string
	AgentEdit bool
}

type detail struct {
	Change    change     `json:"change"`
	TestCases []testCase `json:"test_cases"`
}

type testCase struct {
	ID int `json:"id"`
}

type testCaseMutation struct {
	TestCase *testCase `json:"test_case"`
}

type renderedArtifacts struct {
	Artifacts []struct {
		ID       int    `json:"id"`
		SpecHTML string `json:"spec_html"`
		PRHtml   string `json:"pr_html"`
	} `json:"artifacts"`
}

var (
	removedUpdateAgentEditPath    = "/api/v1/change/update-agent-" + "edit"
	removedUpdateDefAgentEditPath = "/api/v1/change/update-def-agent-" + "edit"
	removedUpdateIdeaPath         = "/api/v1/change/update-" + "idea"
)

func TestChangeCRUDAndOptions(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)
	epicID := createEpic(t, client, projectID)

	var phases []changeOption
	status := client.Post(t, "/api/v1/options/change-phases-list", map[string]any{}, &phases)
	require.Equal(t, http.StatusOK, status)
	require.NotEmpty(t, phases)
	require.NotEmpty(t, phases[0].Color)
	var types []changeOption
	status = client.Post(t, "/api/v1/options/change-types-list", map[string]any{}, &types)
	require.Equal(t, http.StatusOK, status)
	require.NotEmpty(t, types)

	title := fmt.Sprintf("api-test-change-%d", time.Now().UnixNano())
	brief := "Created by change API integration test."
	var created change
	status = client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      title,
		"brief":      brief,
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, created.ID)
	assert.NotEmpty(t, created.RefUUID)
	assert.Nil(t, created.Ref)
	assert.Nil(t, created.Slug)
	assert.Equal(t, title, created.Title)
	assert.Equal(t, brief, created.Brief)
	assert.Equal(t, "backlog", created.ChangePhase)
	assert.Empty(t, created.Spec)
	assert.Empty(t, created.SpecHTML)
	assert.Empty(t, created.PR)
	assert.Empty(t, created.PRHtml)
	assert.Empty(t, created.PRUrl)

	assert.True(t, created.Open)
	assert.Empty(t, created.ChangeTypes)
	assert.Nil(t, created.EpicID)

	status = client.Post(t, removedUpdateIdeaPath, map[string]any{
		"id": created.ID, "idea": "legacy brief", "agent_edit": false,
	}, nil)
	require.Equal(t, http.StatusNotFound, status)
	status = client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID, "title": "legacy field", "idea": "legacy brief",
	}, nil)
	require.Equal(t, http.StatusBadRequest, status)

	var listed []change
	status = client.Post(t, "/api/v1/change/list", map[string]any{"project_id": projectID}, &listed)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listed, 1)
	assert.Equal(t, created.ID, listed[0].ID)
	assert.Equal(t, created.RefUUID, listed[0].RefUUID)
	assert.Equal(t, created.Ref, listed[0].Ref)
	assert.Equal(t, created.Slug, listed[0].Slug)
	assert.Equal(t, created.Title, listed[0].Title)
	assert.Nil(t, listed[0].EpicID)
	assert.Nil(t, listed[0].EpicName)

	var listedFields []map[string]any
	status = client.Post(t, "/api/v1/change/list", map[string]any{"project_id": projectID}, &listedFields)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listedFields, 1)
	assert.NotContains(t, listedFields[0], "brief")
	assert.NotContains(t, listedFields[0], "agent_edit")
	assert.NotContains(t, listedFields[0], "spec")
	assert.NotContains(t, listedFields[0], "spec_html")
	assert.NotContains(t, listedFields[0], "pr")
	assert.NotContains(t, listedFields[0], "pr_url")
	assert.NotContains(t, listedFields[0], "flow_stages")
	assert.NotContains(t, listedFields[0], "flow_stage_modes")
	assert.NotContains(t, listedFields[0], "run_claim_id")
	assert.NotContains(t, listedFields[0], "run_flow_stage")
	assert.NotContains(t, listedFields[0], "run_task_step")
	assert.NotContains(t, listedFields[0], "run_task_status")
	assert.NotContains(t, listedFields[0], "run_error")
	assert.NotContains(t, listedFields[0], "run_is_completed")
	assert.NotContains(t, listedFields[0], "run_started_at")
	assert.NotContains(t, listedFields[0], "run_updated_at")
	assert.NotContains(t, listedFields[0], "version")
	assert.NotContains(t, listedFields[0], "created")

	var fetched detail
	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, created.ID, fetched.Change.ID)
	assert.Equal(t, created.RefUUID, fetched.Change.RefUUID)
	assert.Equal(t, created.Ref, fetched.Change.Ref)
	assert.Equal(t, created.Slug, fetched.Change.Slug)
	assert.Equal(t, brief, fetched.Change.Brief)
	assert.Empty(t, fetched.Change.SpecHTML)

	var rendered renderedArtifacts
	status = client.Post(t, "/api/v1/change/rendered-artifacts", map[string]any{"ids": []int{created.ID}}, &rendered)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, rendered.Artifacts, 1)
	assert.Empty(t, rendered.Artifacts[0].SpecHTML)

	var updated change
	punctuationTitle := "perf(json): pooled-buffer JSON deserialize"
	status = client.Post(t, "/api/v1/change/update-title", map[string]any{"id": created.ID, "title": punctuationTitle}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, punctuationTitle, updated.Title)
	assert.Nil(t, updated.Ref)
	assert.Nil(t, updated.Slug)

	status = client.Post(t, "/api/v1/change/reference", map[string]any{"id": created.ID}, nil)
	require.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/change/update-brief", map[string]any{
		"id":         created.ID,
		"brief":      "Focused brief update.",
		"agent_edit": false,
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Focused brief update.", updated.Brief)

	status = client.Post(t, "/api/v1/change/update-brief", map[string]any{
		"id":         created.ID,
		"brief":      "Agent rewritten brief.",
		"agent_edit": true,
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Agent rewritten brief.", updated.Brief)

	status = client.Post(t, "/api/v1/change/update-spec", map[string]any{
		"id":         created.ID,
		"spec":       "Focused spec update.",
		"agent_edit": false,
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Focused spec update.", updated.Spec)
	assert.Contains(t, updated.SpecHTML, "<p>Focused spec update.</p>")

	status = client.Post(t, "/api/v1/change/update-spec", map[string]any{
		"id":         created.ID,
		"spec":       nil,
		"agent_edit": false,
	}, nil)
	require.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/update-spec", map[string]any{
		"id":         created.ID,
		"spec":       "",
		"agent_edit": false,
	}, nil)
	require.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/update-pr", map[string]any{
		"id":         created.ID,
		"pr":         "Focused pull request body update.",
		"agent_edit": true,
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Focused pull request body update.", updated.PR)

	status = client.Post(t, "/api/v1/change/update-pr", map[string]any{
		"id":         created.ID,
		"pr":         "",
		"agent_edit": true,
	}, nil)
	require.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/update-pr-url", map[string]any{
		"id":     created.ID,
		"pr_url": "https://example.test/project-manager/pull/1",
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "https://example.test/project-manager/pull/1", updated.PRUrl)

	status = client.Post(t, "/api/v1/change/update-pr-url", map[string]any{
		"id":     created.ID,
		"pr_url": nil,
	}, nil)
	require.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/update-pr-url", map[string]any{
		"id":     created.ID,
		"pr_url": "",
	}, nil)
	require.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/update-pr-url", map[string]any{
		"id":     created.ID,
		"pr_url": "javascript:alert(1)",
	}, nil)
	require.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "https://example.test/project-manager/pull/1", fetched.Change.PRUrl)

	status = client.Post(t, removedUpdateAgentEditPath, map[string]any{
		"id":         created.ID,
		"agent_edit": true,
	}, nil)
	require.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/change/update-change-types", map[string]any{
		"id":           created.ID,
		"change_types": []string{},
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Empty(t, updated.ChangeTypes)

	status = client.Post(t, "/api/v1/change/update-change-types", map[string]any{
		"id":           created.ID,
		"change_types": []string{"docs", "missing-type"},
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, []string{"docs"}, updated.ChangeTypes)

	status = client.Post(t, "/api/v1/change/update-change-types", map[string]any{
		"id":           created.ID,
		"change_types": []string{"missing-type"},
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Empty(t, updated.ChangeTypes)

	status = client.Post(t, "/api/v1/change/update-phase", map[string]any{"id": created.ID, "change_phase": "review"}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "review", updated.ChangePhase)

	status = client.Post(t, "/api/v1/change/update-open", map[string]any{"id": created.ID, "open": false}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.False(t, updated.Open)

	status = client.Post(t, "/api/v1/change/update-epic", map[string]any{"id": created.ID, "epic_id": epicID}, &updated)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, updated.EpicID)
	assert.Equal(t, epicID, *updated.EpicID)

	status = client.Post(t, "/api/v1/change/update-epic", map[string]any{"id": created.ID, "epic_id": nil}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Nil(t, updated.EpicID)

	testCaseID := createTestCase(t, client, created.ID)

	status = client.Post(t, "/api/v1/change/delete", map[string]any{"id": created.ID}, nil)
	require.Equal(t, http.StatusNoContent, status)

	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/test-case/delete", map[string]any{"id": testCaseID}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func TestChangeBriefUpdatesPreserveVersionAndHistory(t *testing.T) {
	client := shared.NewClient(t)
	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	var created change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      "Brief history",
		"brief":      "Initial brief.",
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	assert.Equal(t, int16(0), created.Version)

	var userUpdated change
	status = client.Post(t, "/api/v1/change/update-brief", map[string]any{
		"id":         created.ID,
		"brief":      "User brief update.",
		"agent_edit": false,
	}, &userUpdated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, int16(1), userUpdated.Version)
	assert.Equal(t, "User brief update.", userUpdated.Brief)

	var agentUpdated change
	status = client.Post(t, "/api/v1/change/update-brief", map[string]any{
		"id":         created.ID,
		"brief":      "Agent brief update.",
		"agent_edit": true,
	}, &agentUpdated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, int16(2), agentUpdated.Version)
	assert.Equal(t, "Agent brief update.", agentUpdated.Brief)

	databaseURL := os.Getenv("API_TEST_DB_URL")
	require.NotEmpty(t, databaseURL, "API_TEST_DB_URL must identify the disposable API-test database")

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, conn.Close(ctx))
	}()

	rows, err := conn.Query(
		ctx,
		`select version, doc_type, body, agent_edit
		from public.change_history
		where id = $1
		order by version`,
		created.ID,
	)
	require.NoError(t, err)

	history, err := pgx.CollectRows(rows, pgx.RowToStructByPos[changeHistory])
	require.NoError(t, err)
	assert.Equal(t, []changeHistory{
		{Version: 0, DocType: "brief", Body: "Initial brief.", AgentEdit: false},
		{Version: 1, DocType: "brief", Body: "User brief update.", AgentEdit: false},
		{Version: 2, DocType: "brief", Body: "Agent brief update.", AgentEdit: true},
	}, history)
}

func TestChangeListOrdersByModifiedDescending(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	var older change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      fmt.Sprintf("api-test-older-change-%d", time.Now().UnixNano()),
		"brief":      "Older brief",
	}, &older)
	require.Equal(t, http.StatusCreated, status)

	time.Sleep(10 * time.Millisecond)

	var newer change
	status = client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      fmt.Sprintf("api-test-newer-change-%d", time.Now().UnixNano()),
		"brief":      "Newer brief",
	}, &newer)
	require.Equal(t, http.StatusCreated, status)

	var listed []change
	status = client.Post(t, "/api/v1/change/list", map[string]any{"project_id": projectID}, &listed)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listed, 2)
	assert.Equal(t, newer.ID, listed[0].ID)
	assert.Equal(t, older.ID, listed[1].ID)

	time.Sleep(10 * time.Millisecond)

	var updated change
	status = client.Post(t, "/api/v1/change/update-title", map[string]any{
		"id":    older.ID,
		"title": older.Title + "-updated",
	}, &updated)
	require.Equal(t, http.StatusOK, status)

	status = client.Post(t, "/api/v1/change/list", map[string]any{"project_id": projectID}, &listed)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, listed, 2)
	assert.Equal(t, older.ID, listed[0].ID)
	assert.Equal(t, newer.ID, listed[1].ID)
}

func TestChangeGetReturnsTestCasesOrderedByID(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	var created change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      fmt.Sprintf("api-test-testcase-order-change-%d", time.Now().UnixNano()),
		"brief":      "Test case ordering brief",
	}, &created)
	require.Equal(t, http.StatusCreated, status)

	firstID := createTestCaseWithScenario(t, client, created.ID, "zzz first by id")
	secondID := createTestCaseWithScenario(t, client, created.ID, "aaa second by id")
	thirdID := createTestCaseWithScenario(t, client, created.ID, "mmm third by id")

	var fetched detail
	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, fetched.TestCases, 3)
	assert.Equal(t, []int{firstID, secondID, thirdID}, []int{
		fetched.TestCases[0].ID,
		fetched.TestCases[1].ID,
		fetched.TestCases[2].ID,
	})
}

func TestChangeBooleanUpdatesRequireExplicitFields(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	var created change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      fmt.Sprintf("api-test-boolean-change-%d", time.Now().UnixNano()),
		"brief":      "Boolean update brief",
	}, &created)
	require.Equal(t, http.StatusCreated, status)

	assert.True(t, created.Open)

	status = client.Post(t, "/api/v1/change/update-open", map[string]any{"id": created.ID}, nil)
	require.Equal(t, http.StatusBadRequest, status)
	status = client.Post(t, "/api/v1/change/update-open", map[string]any{"id": created.ID, "closed": true}, nil)
	require.Equal(t, http.StatusBadRequest, status)

	var fetched detail
	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.True(t, fetched.Change.Open)

	status = client.Post(t, removedUpdateAgentEditPath, map[string]any{"id": created.ID}, nil)
	require.Equal(t, http.StatusNotFound, status)
	status = client.Post(t, removedUpdateDefAgentEditPath, map[string]any{"id": created.ID, "brief": " ", "agent_edit": true}, nil)
	require.Equal(t, http.StatusNotFound, status)

	artifactRequests := []struct {
		path  string
		field string
		body  string
	}{
		{path: "/api/v1/change/update-brief", field: "brief", body: "Brief without provenance"},
		{path: "/api/v1/change/update-spec", field: "spec", body: "Spec without provenance"},
		{path: "/api/v1/change/update-pr", field: "pr", body: "PR without provenance"},
	}
	for _, request := range artifactRequests {
		t.Run(request.field+" rejects omitted agent_edit", func(t *testing.T) {
			status := client.Post(t, request.path, map[string]any{
				"id":          created.ID,
				request.field: request.body,
			}, nil)
			require.Equal(t, http.StatusBadRequest, status)
		})
		t.Run(request.field+" rejects null agent_edit", func(t *testing.T) {
			status := client.Post(t, request.path, map[string]any{
				"id":          created.ID,
				request.field: request.body,
				"agent_edit":  nil,
			}, nil)
			require.Equal(t, http.StatusBadRequest, status)
		})
	}
	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Boolean update brief", fetched.Change.Brief)
	assert.Empty(t, fetched.Change.Spec)
	assert.Empty(t, fetched.Change.PR)

	var updated change
	status = client.Post(t, "/api/v1/change/update-brief", map[string]any{
		"id":         created.ID,
		"brief":      "Agent-edited brief.",
		"agent_edit": true,
	}, &updated)
	require.Equal(t, http.StatusOK, status)

	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)

}

func TestChangeArtifactUpdatesRejectNullAndEmptyWithoutMutation(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	const originalBrief = "Artifact validation brief"
	var created change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      fmt.Sprintf("api-test-artifact-validation-%d", time.Now().UnixNano()),
		"brief":      originalBrief,
	}, &created)
	require.Equal(t, http.StatusCreated, status)

	requests := []struct {
		name  string
		path  string
		field string
		value any
	}{
		{name: "null brief", path: "/api/v1/change/update-brief", field: "brief", value: nil},
		{name: "empty brief", path: "/api/v1/change/update-brief", field: "brief", value: ""},
		{name: "null spec", path: "/api/v1/change/update-spec", field: "spec", value: nil},
		{name: "empty spec", path: "/api/v1/change/update-spec", field: "spec", value: ""},
		{name: "null pr", path: "/api/v1/change/update-pr", field: "pr", value: nil},
		{name: "empty pr", path: "/api/v1/change/update-pr", field: "pr", value: ""},
		{name: "null pr url", path: "/api/v1/change/update-pr-url", field: "pr_url", value: nil},
		{name: "empty pr url", path: "/api/v1/change/update-pr-url", field: "pr_url", value: ""},
	}

	for _, request := range requests {
		t.Run(request.name, func(t *testing.T) {
			body := map[string]any{
				"id":          created.ID,
				request.field: request.value,
			}
			if request.field != "pr_url" {
				body["agent_edit"] = true
			}

			status := client.Post(t, request.path, body, nil)
			require.Equal(t, http.StatusBadRequest, status)

			var fetched detail
			status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
			require.Equal(t, http.StatusOK, status)
			assert.Equal(t, originalBrief, fetched.Change.Brief)
			assert.Empty(t, fetched.Change.Spec)
			assert.Empty(t, fetched.Change.PR)
			assert.Empty(t, fetched.Change.PRUrl)

		})
	}
}

func TestChangeCreateRejectsInvalidInput(t *testing.T) {
	client := shared.NewClient(t)

	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": 999999999,
		"title":      "orphan change",
		"brief":      "Orphan brief",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	status = client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      "   ",
		"brief":      "Blank title brief",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      "blank brief change",
		"brief":      "   ",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestChangeRejectsInvalidInputAndMissingRows(t *testing.T) {
	client := shared.NewClient(t)

	status := client.Post(t, "/api/v1/change/list", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": 999999999}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/change/rendered-artifacts", map[string]any{"ids": []int{0}}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/update-title", map[string]any{
		"id":    999999999,
		"title": "missing change",
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/change/update-change-types", map[string]any{
		"id":           999999999,
		"change_types": []string{"missing-type"},
	}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/change/update-epic", map[string]any{"id": 999999999, "epic_id": nil}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/change/update-phase", map[string]any{
		"id":           999999999,
		"change_phase": "missing-phase",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/update-open", map[string]any{"id": 999999999, "open": false}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/change/update-pr-url", map[string]any{"id": 999999999, "pr_url": "https://example.test"}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, removedUpdateAgentEditPath, map[string]any{"id": 999999999, "agent_edit": true}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, removedUpdateDefAgentEditPath, map[string]any{"id": 999999999, "brief": "missing", "agent_edit": true}, nil)
	assert.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/change/delete", map[string]any{}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/delete", map[string]any{"id": 999999999}, nil)
	assert.Equal(t, http.StatusNotFound, status)
}

func createProject(t *testing.T, client *shared.Client) int {
	t.Helper()
	var created project
	status := client.Post(t, "/api/v1/project/create", map[string]string{
		"name": fmt.Sprintf("api-test-change-project-%d", time.Now().UnixNano()),
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, created.ID)
	return created.ID
}

func createEpic(t *testing.T, client *shared.Client, projectID int) int {
	t.Helper()
	var created epic
	status := client.Post(t, "/api/v1/epic/create", map[string]any{
		"project_id": projectID,
		"name":       fmt.Sprintf("api-test-epic-%d", time.Now().UnixNano()),
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, created.ID)
	return created.ID
}

func createTestCase(t *testing.T, client *shared.Client, changeID int) int {
	t.Helper()

	return createTestCaseWithScenario(t, client, changeID, "Change delete removes this test case.")
}

func createTestCaseWithScenario(t *testing.T, client *shared.Client, changeID int, scenario string) int {
	t.Helper()

	var created testCaseMutation
	status := client.Post(t, "/api/v1/test-case/create", map[string]any{
		"change_id": changeID,
		"scenario":  scenario,
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotNil(t, created.TestCase)
	require.NotEmpty(t, created.TestCase.ID)
	return created.TestCase.ID
}
