package change_test

import (
	"aipm/api-tests/shared"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
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
		payload := map[string]any{"project_id": projectID, "title": name, "def": "# " + name}
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
	ID             int      `json:"id"`
	Version        int16    `json:"version"`
	RefUUID        string   `json:"ref_uuid"`
	Ref            *int32   `json:"ref"`
	Slug           *string  `json:"slug"`
	ProjectID      int      `json:"project_id"`
	EpicID         *int     `json:"epic_id"`
	EpicName       *string  `json:"epic_name"`
	ChangePhase    string   `json:"change_phase"`
	ChangeTypes    []string `json:"change_types"`
	Title          string   `json:"title"`
	Def            string   `json:"def"`
	Spec           string   `json:"spec"`
	SpecHTML       string   `json:"spec_html"`
	PR             string   `json:"pr"`
	PRHtml         string   `json:"pr_html"`
	PRUrl          string   `json:"pr_url"`
	AgentEdit      bool     `json:"agent_edit"`
	FlowStages     []string `json:"flow_stages"`
	FlowStageModes []string `json:"flow_stage_modes"`
	RunClaimID     *string  `json:"run_claim_id"`
	RunFlowStage   string   `json:"run_flow_stage"`
	RunTaskStep    string   `json:"run_task_step"`
	RunTaskStatus  string   `json:"run_task_status"`
	RunError       string   `json:"run_error"`
	RunIsCompleted bool     `json:"run_is_completed"`
	RunStartedAt   *string  `json:"run_started_at"`
	RunUpdatedAt   *string  `json:"run_updated_at"`
	Open           bool     `json:"open"`
	DoneTC         int16    `json:"done_tc"`
	TotalTC        int16    `json:"total_tc"`
	Completed      int16    `json:"completed"`
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

type runClaimResponse struct {
	ClaimID *string `json:"claim_id"`
}

type runUpdateResponse struct {
	ChangeID *int `json:"change_id"`
}

type assignFlowResult struct {
	status int
	change change
	err    error
}

type startRunResult struct {
	status   int
	response runClaimResponse
	err      error
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
	def := "Created by change API integration test."
	var created change
	status = client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      title,
		"def":        def,
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	require.NotEmpty(t, created.ID)
	assert.NotEmpty(t, created.RefUUID)
	assert.Nil(t, created.Ref)
	assert.Nil(t, created.Slug)
	assert.Equal(t, title, created.Title)
	assert.Equal(t, def, created.Def)
	assert.Equal(t, "backlog", created.ChangePhase)
	assert.Empty(t, created.Spec)
	assert.Empty(t, created.SpecHTML)
	assert.Empty(t, created.PR)
	assert.Empty(t, created.PRHtml)
	assert.Empty(t, created.PRUrl)
	assert.False(t, created.AgentEdit)
	assert.True(t, created.Open)
	assert.Empty(t, created.ChangeTypes)
	assert.Nil(t, created.EpicID)

	status = client.Post(t, removedUpdateIdeaPath, map[string]any{
		"id": created.ID, "idea": "legacy definition", "agent_edit": false,
	}, nil)
	require.Equal(t, http.StatusNotFound, status)
	status = client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID, "title": "legacy field", "idea": "legacy definition",
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
	assert.NotContains(t, listedFields[0], "def")
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
	assert.Equal(t, def, fetched.Change.Def)
	assert.Empty(t, fetched.Change.SpecHTML)

	var referenced change
	status = client.Post(t, "/api/v1/change/assign-flow", map[string]any{"id": created.ID}, &referenced)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, referenced.Ref)
	require.NotNil(t, referenced.Slug)
	require.NotEmpty(t, referenced.FlowStages)
	require.NotEmpty(t, referenced.FlowStageModes)
	assert.Equal(t, def, referenced.Def)

	firstRef := *referenced.Ref
	firstSlug := *referenced.Slug
	firstFlowStages := append([]string(nil), referenced.FlowStages...)
	firstFlowStageModes := append([]string(nil), referenced.FlowStageModes...)
	var projectAfterFirstReference project
	status = client.Post(t, "/api/v1/project/get", map[string]any{"id": projectID}, &projectAfterFirstReference)
	require.Equal(t, http.StatusOK, status)

	status = client.Post(t, "/api/v1/change/assign-flow", map[string]any{"id": created.ID}, &referenced)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, referenced.Ref)
	require.NotNil(t, referenced.Slug)
	assert.Equal(t, firstRef, *referenced.Ref)
	assert.Equal(t, firstSlug, *referenced.Slug)
	assert.Equal(t, firstFlowStages, referenced.FlowStages)
	assert.Equal(t, firstFlowStageModes, referenced.FlowStageModes)
	var projectAfterSecondReference project
	status = client.Post(t, "/api/v1/project/get", map[string]any{"id": projectID}, &projectAfterSecondReference)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, projectAfterFirstReference.LastRef, projectAfterSecondReference.LastRef)

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
	require.NotNil(t, updated.Ref)
	require.NotNil(t, updated.Slug)
	assert.Equal(t, firstRef, *updated.Ref)
	assert.Equal(t, firstSlug, *updated.Slug)

	status = client.Post(t, "/api/v1/change/assign-flow", map[string]any{"id": created.ID}, &referenced)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, referenced.Ref)
	require.NotNil(t, referenced.Slug)
	assert.Equal(t, firstRef, *referenced.Ref)
	assert.Equal(t, fmt.Sprintf("%03d-perf-json-pooled-buffer-json-deserialize", firstRef), *referenced.Slug)
	var projectAfterSlugRefresh project
	status = client.Post(t, "/api/v1/project/get", map[string]any{"id": projectID}, &projectAfterSlugRefresh)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, projectAfterSecondReference.LastRef, projectAfterSlugRefresh.LastRef)

	status = client.Post(t, "/api/v1/change/reference", map[string]any{"id": created.ID}, nil)
	require.Equal(t, http.StatusNotFound, status)

	status = client.Post(t, "/api/v1/change/update-def", map[string]any{
		"id":         created.ID,
		"def":        "Focused definition update.",
		"agent_edit": false,
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Focused definition update.", updated.Def)
	assert.False(t, updated.AgentEdit)

	status = client.Post(t, "/api/v1/change/update-def", map[string]any{
		"id":         created.ID,
		"def":        "Agent rewritten definition.",
		"agent_edit": true,
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "Agent rewritten definition.", updated.Def)
	assert.True(t, updated.AgentEdit)

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
	assert.True(t, updated.AgentEdit)

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

func TestChangeDefinitionUpdatesPreserveVersionAndHistory(t *testing.T) {
	client := shared.NewClient(t)
	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	var created change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      "Definition history",
		"def":        "Initial definition.",
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	assert.Equal(t, int16(0), created.Version)

	var userUpdated change
	status = client.Post(t, "/api/v1/change/update-def", map[string]any{
		"id":         created.ID,
		"def":        "User definition update.",
		"agent_edit": false,
	}, &userUpdated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, int16(1), userUpdated.Version)
	assert.Equal(t, "User definition update.", userUpdated.Def)
	assert.False(t, userUpdated.AgentEdit)

	var agentUpdated change
	status = client.Post(t, "/api/v1/change/update-def", map[string]any{
		"id":         created.ID,
		"def":        "Agent definition update.",
		"agent_edit": true,
	}, &agentUpdated)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, int16(2), agentUpdated.Version)
	assert.Equal(t, "Agent definition update.", agentUpdated.Def)
	assert.True(t, agentUpdated.AgentEdit)

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
		{Version: 0, DocType: "def", Body: "Initial definition.", AgentEdit: false},
		{Version: 1, DocType: "def", Body: "User definition update.", AgentEdit: false},
		{Version: 2, DocType: "def", Body: "Agent definition update.", AgentEdit: true},
	}, history)
}

func TestChangeAssignFlowConcurrentRequestsPreserveSingleRef(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	var created change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      "Concurrent reference",
		"def":        "# Concurrent reference\n\nAssign one reference.",
	}, &created)
	require.Equal(t, http.StatusCreated, status)

	var projectBefore project
	status = client.Post(t, "/api/v1/project/get", map[string]any{"id": projectID}, &projectBefore)
	require.Equal(t, http.StatusOK, status)

	const requestCount = 12
	start := make(chan struct{})
	results := make(chan assignFlowResult, requestCount)
	var wg sync.WaitGroup
	for range requestCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			status, referenced, err := postChangeAssignFlow(client.BaseURL(), created.ID)
			results <- assignFlowResult{status: status, change: referenced, err: err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var firstRef int32
	var firstSlug string
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, http.StatusOK, result.status)
		require.NotNil(t, result.change.Ref)
		require.NotNil(t, result.change.Slug)
		if firstRef == 0 {
			firstRef = *result.change.Ref
			firstSlug = *result.change.Slug
		}
		assert.Equal(t, firstRef, *result.change.Ref)
		assert.Equal(t, firstSlug, *result.change.Slug)
	}

	var projectAfter project
	status = client.Post(t, "/api/v1/project/get", map[string]any{"id": projectID}, &projectAfter)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, projectBefore.LastRef+1, projectAfter.LastRef)
}

func postChangeAssignFlow(baseURL string, changeID int) (int, change, error) {
	payload, err := json.Marshal(map[string]any{"id": changeID})
	if err != nil {
		return 0, change{}, err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/change/assign-flow", bytes.NewReader(payload))
	if err != nil {
		return 0, change{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return 0, change{}, err
	}
	defer res.Body.Close()

	var referenced change
	if res.StatusCode != http.StatusOK {
		return res.StatusCode, referenced, nil
	}
	if err := json.NewDecoder(res.Body).Decode(&referenced); err != nil {
		return res.StatusCode, change{}, err
	}
	return res.StatusCode, referenced, nil
}

func TestChangeStartRunConcurrentRequestsPreserveSingleClaim(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	var created change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      "Concurrent run claim",
		"def":        "Claim the run once.",
	}, &created)
	require.Equal(t, http.StatusCreated, status)

	var assigned change
	status = client.Post(t, "/api/v1/change/assign-flow", map[string]any{"id": created.ID}, &assigned)
	require.Equal(t, http.StatusOK, status)

	const requestCount = 12
	start := make(chan struct{})
	results := make(chan startRunResult, requestCount)
	var wg sync.WaitGroup
	for range requestCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			status, response, err := postChangeStartRun(client.BaseURL(), created.ID)
			results <- startRunResult{status: status, response: response, err: err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	var winningClaim string
	claimedCount := 0
	noClaimCount := 0
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, http.StatusOK, result.status)
		if result.response.ClaimID == nil {
			noClaimCount++
			continue
		}
		claimedCount++
		require.NotEmpty(t, *result.response.ClaimID)
		winningClaim = *result.response.ClaimID
	}
	require.Equal(t, 1, claimedCount)
	assert.Equal(t, requestCount-1, noClaimCount)

	var fetched detail
	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, fetched.Change.RunClaimID)
	assert.Equal(t, winningClaim, *fetched.Change.RunClaimID)
	assert.NotNil(t, fetched.Change.RunStartedAt)
}

func postChangeStartRun(baseURL string, changeID int) (int, runClaimResponse, error) {
	payload, err := json.Marshal(map[string]any{"id": changeID})
	if err != nil {
		return 0, runClaimResponse{}, err
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/v1/change/start-run", bytes.NewReader(payload))
	if err != nil {
		return 0, runClaimResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return 0, runClaimResponse{}, err
	}
	defer res.Body.Close()

	var response runClaimResponse
	if res.StatusCode != http.StatusOK {
		return res.StatusCode, response, nil
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return res.StatusCode, runClaimResponse{}, err
	}
	return res.StatusCode, response, nil
}

func TestChangeRunLifecycle(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	var created change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      "Run lifecycle",
		"def":        "Run lifecycle def.",
	}, &created)
	require.Equal(t, http.StatusCreated, status)

	var assigned change
	status = client.Post(t, "/api/v1/change/assign-flow", map[string]any{"id": created.ID}, &assigned)
	require.Equal(t, http.StatusOK, status)
	require.NotEmpty(t, assigned.FlowStages)
	require.NotEmpty(t, assigned.FlowStageModes)

	var started runClaimResponse
	status = client.Post(t, "/api/v1/change/start-run", map[string]any{"id": created.ID}, &started)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, started.ClaimID)
	require.NotEmpty(t, *started.ClaimID)

	var fetched detail
	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, fetched.Change.RunClaimID)
	assert.Equal(t, *started.ClaimID, *fetched.Change.RunClaimID)
	assert.NotNil(t, fetched.Change.RunStartedAt)

	var detailFields map[string]any
	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &detailFields)
	require.Equal(t, http.StatusOK, status)
	changeFields, ok := detailFields["change"].(map[string]any)
	require.True(t, ok)
	for _, field := range []string{
		"flow_stages",
		"flow_stage_modes",
		"run_claim_id",
		"run_flow_stage",
		"run_task_step",
		"run_task_status",
		"run_error",
		"run_is_completed",
		"run_started_at",
		"run_updated_at",
	} {
		assert.Contains(t, changeFields, field)
	}

	var duplicateStart runClaimResponse
	status = client.Post(t, "/api/v1/change/start-run", map[string]any{"id": created.ID}, &duplicateStart)
	require.Equal(t, http.StatusOK, status)
	assert.Nil(t, duplicateStart.ClaimID)

	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, fetched.Change.RunClaimID)
	assert.Equal(t, *started.ClaimID, *fetched.Change.RunClaimID)

	var runUpdate runUpdateResponse
	status = client.Post(t, "/api/v1/change/update-run", map[string]any{
		"id":               created.ID,
		"run_claim_id":     " " + *started.ClaimID + " ",
		"run_flow_stage":   " artifact ",
		"run_task_step":    " agent ",
		"run_task_status":  " running ",
		"run_error":        " latest error ",
		"run_is_completed": false,
	}, &runUpdate)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, runUpdate.ChangeID)
	assert.Equal(t, created.ID, *runUpdate.ChangeID)

	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "artifact", fetched.Change.RunFlowStage)
	assert.Equal(t, "agent", fetched.Change.RunTaskStep)
	assert.Equal(t, "running", fetched.Change.RunTaskStatus)
	assert.Equal(t, "latest error", fetched.Change.RunError)
	assert.False(t, fetched.Change.RunIsCompleted)
	assert.NotNil(t, fetched.Change.RunUpdatedAt)

	var reset runClaimResponse
	status = client.Post(t, "/api/v1/change/reset-claim", map[string]any{"id": created.ID}, &reset)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, reset.ClaimID)
	require.NotEqual(t, *started.ClaimID, *reset.ClaimID)

	var staleUpdate runUpdateResponse
	status = client.Post(t, "/api/v1/change/update-run", map[string]any{
		"id":               created.ID,
		"run_claim_id":     *started.ClaimID,
		"run_flow_stage":   "docs",
		"run_task_step":    "done",
		"run_task_status":  "completed",
		"run_error":        "",
		"run_is_completed": true,
	}, &staleUpdate)
	require.Equal(t, http.StatusOK, status)
	assert.Nil(t, staleUpdate.ChangeID)

	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "artifact", fetched.Change.RunFlowStage)
	assert.Equal(t, "agent", fetched.Change.RunTaskStep)
	assert.Equal(t, "running", fetched.Change.RunTaskStatus)
	assert.Equal(t, "latest error", fetched.Change.RunError)
	assert.False(t, fetched.Change.RunIsCompleted)
	require.NotNil(t, fetched.Change.RunClaimID)
	assert.Equal(t, *reset.ClaimID, *fetched.Change.RunClaimID)

	var informationalUpdate runUpdateResponse
	status = client.Post(t, "/api/v1/change/update-run", map[string]any{
		"id":               created.ID,
		"run_claim_id":     *reset.ClaimID,
		"run_flow_stage":   "worker-local-stage",
		"run_task_step":    "worker-local-step",
		"run_task_status":  "worker-local-status",
		"run_error":        "",
		"run_is_completed": false,
	}, &informationalUpdate)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, informationalUpdate.ChangeID)
	assert.Equal(t, created.ID, *informationalUpdate.ChangeID)

	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "worker-local-stage", fetched.Change.RunFlowStage)
	assert.Equal(t, "worker-local-step", fetched.Change.RunTaskStep)
	assert.Equal(t, "worker-local-status", fetched.Change.RunTaskStatus)
	assert.False(t, fetched.Change.RunIsCompleted)

	var completed runUpdateResponse
	status = client.Post(t, "/api/v1/change/update-run", map[string]any{
		"id":               created.ID,
		"run_claim_id":     *reset.ClaimID,
		"run_flow_stage":   "docs",
		"run_task_step":    "done",
		"run_task_status":  "completed",
		"run_error":        "",
		"run_is_completed": true,
	}, &completed)
	require.Equal(t, http.StatusOK, status)
	require.NotNil(t, completed.ChangeID)
	assert.Equal(t, created.ID, *completed.ChangeID)

	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.Nil(t, fetched.Change.RunClaimID)
	assert.Equal(t, "docs", fetched.Change.RunFlowStage)
	assert.Equal(t, "done", fetched.Change.RunTaskStep)
	assert.Equal(t, "completed", fetched.Change.RunTaskStatus)
	assert.Empty(t, fetched.Change.RunError)
	assert.True(t, fetched.Change.RunIsCompleted)
	assert.NotNil(t, fetched.Change.RunUpdatedAt)

	status = client.Post(t, "/api/v1/change/update-run", map[string]any{
		"id":               created.ID,
		"run_claim_id":     "",
		"run_flow_stage":   "docs",
		"run_task_step":    "done",
		"run_task_status":  "completed",
		"run_error":        "",
		"run_is_completed": true,
	}, nil)
	require.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/update-run", map[string]any{
		"id":               created.ID,
		"run_claim_id":     "not-a-uuid",
		"run_flow_stage":   "docs",
		"run_task_step":    "done",
		"run_task_status":  "completed",
		"run_error":        "",
		"run_is_completed": true,
	}, nil)
	require.Equal(t, http.StatusBadRequest, status)
}

func TestChangeListOrdersByModifiedDescending(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	var older change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      fmt.Sprintf("api-test-older-change-%d", time.Now().UnixNano()),
		"def":        "Older def",
	}, &older)
	require.Equal(t, http.StatusCreated, status)

	time.Sleep(10 * time.Millisecond)

	var newer change
	status = client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      fmt.Sprintf("api-test-newer-change-%d", time.Now().UnixNano()),
		"def":        "Newer def",
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
		"def":        "Test case ordering def",
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
		"def":        "Boolean update definition",
	}, &created)
	require.Equal(t, http.StatusCreated, status)
	assert.False(t, created.AgentEdit)
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
	status = client.Post(t, removedUpdateDefAgentEditPath, map[string]any{"id": created.ID, "def": " ", "agent_edit": true}, nil)
	require.Equal(t, http.StatusNotFound, status)

	artifactRequests := []struct {
		path  string
		field string
		body  string
	}{
		{path: "/api/v1/change/update-def", field: "def", body: "Definition without provenance"},
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
	assert.Equal(t, "Boolean update definition", fetched.Change.Def)
	assert.Empty(t, fetched.Change.Spec)
	assert.Empty(t, fetched.Change.PR)
	assert.False(t, fetched.Change.AgentEdit)

	var updated change
	status = client.Post(t, "/api/v1/change/update-def", map[string]any{
		"id":         created.ID,
		"def":        "Agent-edited def.",
		"agent_edit": true,
	}, &updated)
	require.Equal(t, http.StatusOK, status)
	require.True(t, updated.AgentEdit)

	status = client.Post(t, "/api/v1/change/get", map[string]any{"id": created.ID}, &fetched)
	require.Equal(t, http.StatusOK, status)
	assert.True(t, fetched.Change.AgentEdit)
}

func TestChangeArtifactUpdatesRejectNullAndEmptyWithoutMutation(t *testing.T) {
	client := shared.NewClient(t)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	const originalDef = "Artifact validation def"
	var created change
	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      fmt.Sprintf("api-test-artifact-validation-%d", time.Now().UnixNano()),
		"def":        originalDef,
	}, &created)
	require.Equal(t, http.StatusCreated, status)

	requests := []struct {
		name  string
		path  string
		field string
		value any
	}{
		{name: "null def", path: "/api/v1/change/update-def", field: "def", value: nil},
		{name: "empty def", path: "/api/v1/change/update-def", field: "def", value: ""},
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
			assert.Equal(t, originalDef, fetched.Change.Def)
			assert.Empty(t, fetched.Change.Spec)
			assert.Empty(t, fetched.Change.PR)
			assert.Empty(t, fetched.Change.PRUrl)
			assert.False(t, fetched.Change.AgentEdit)
		})
	}
}

func TestChangeCreateRejectsInvalidInput(t *testing.T) {
	client := shared.NewClient(t)

	status := client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": 999999999,
		"title":      "orphan change",
		"def":        "Orphan def",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	projectID := createProject(t, client)
	defer shared.CleanupProject(t, client, projectID)

	status = client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      "   ",
		"def":        "Blank title def",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, status)

	status = client.Post(t, "/api/v1/change/create", map[string]any{
		"project_id": projectID,
		"title":      "blank def change",
		"def":        "   ",
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

	status = client.Post(t, removedUpdateDefAgentEditPath, map[string]any{"id": 999999999, "def": "missing", "agent_edit": true}, nil)
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
