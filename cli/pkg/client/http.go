package client

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "strconv"
    "strings"

    "cli/internal/dto"
)

// Client defines backend API methods used by mch.
type Client interface {
    ListProjects() ([]dto.Option, error)
    ListProjectRows() ([]dto.Project, error)
    GetProject(id int) (dto.Project, error)
    CreateProject(name string) (dto.Project, error)
    UpdateProject(id int, name string) (dto.Project, error)
    ListChangeRows(projectID string) ([]dto.Change, error)
    GetChange(id int) (dto.Change, error)
    CreateChange(input dto.ChangeCreateInput) (dto.Change, error)
    ReferenceChange(id int) (dto.Change, error)
    UpdateChangeTitle(id int, title string) (dto.Change, error)
    UpdateChangeDef(id int, def string, agentEdit bool) (dto.Change, error)
    UpdateChangeSpec(id int, spec string, agentEdit bool) (dto.Change, error)
    UpdateChangePR(id int, pr string, agentEdit bool) (dto.Change, error)
    UpdateChangePRUrl(id int, prURL string) (dto.Change, error)
    UpdateChangeTypes(id int, changeTypes []string) (dto.Change, error)
    UpdateChangePhase(id int, changePhase string) (dto.Change, error)
    UpdateChangeOpen(id int, open bool) (dto.Change, error)
    UpdateChangeEpic(id int, epicID *int) (dto.Change, error)
    CreateTestCase(changeID int, scenario string) (dto.Change, error)
    UpdateTestCase(id int, scenario string) (dto.Change, error)
    UpdateTestCaseDone(id int, done bool) (dto.Change, error)
    DeleteTestCase(id int) (dto.Change, error)
    DeleteChange(id int) error
    ListEpics(projectID string) ([]dto.Option, error)
    ListPhases() ([]dto.Option, error)
    ListTypes() ([]dto.Option, error)
}

// HTTPClient calls the Project Manager backend over HTTP.
type HTTPClient struct {
    BaseURL string
    Client  *http.Client
}

// NewHTTPClient creates an HTTP backend client for a base URL.
func NewHTTPClient(baseURL string) HTTPClient {
    return HTTPClient{
        BaseURL: baseURL,
        Client:  http.DefaultClient,
    }
}

// ListProjects loads projects as selector options.
func (c HTTPClient) ListProjects() ([]dto.Option, error) {
    projects, err := c.ListProjectRows()
    if err != nil {
        return nil, err
    }
    options := make([]dto.Option, 0, len(projects))
    for _, project := range projects {
        label := project.Name
        if label == "" {
            label = project.ID
        }
        if label == "" {
            continue
        }
        options = append(options, dto.Option{ID: project.ID, Label: label})
    }
    return options, nil
}

// ListProjectRows loads projects with full table row fields.
func (c HTTPClient) ListProjectRows() ([]dto.Project, error) {
    return c.postProjects("/api/v1/project/list", map[string]any{}, "projects")
}

// GetProject loads a single project by numeric ID.
func (c HTTPClient) GetProject(id int) (dto.Project, error) {
    if id <= 0 {
        return dto.Project{}, fmt.Errorf("project ID must be a valid positive number")
    }
    return c.postProject("/api/v1/project/get", map[string]any{"id": id})
}

// CreateProject creates a project with the required name field.
func (c HTTPClient) CreateProject(name string) (dto.Project, error) {
    return c.postProject("/api/v1/project/create", map[string]any{"name": name})
}

// UpdateProject updates a project name by numeric ID.
func (c HTTPClient) UpdateProject(id int, name string) (dto.Project, error) {
    if id <= 0 {
        return dto.Project{}, fmt.Errorf("project ID must be a valid positive number")
    }
    return c.postProject("/api/v1/project/update", map[string]any{
        "id":   id,
        "name": name,
    })
}

// ListChangeRows loads changes for a project.
func (c HTTPClient) ListChangeRows(projectID string) ([]dto.Change, error) {
    numericProjectID, err := numericCurrentProjectID(projectID)
    if err != nil {
        return nil, err
    }
    return c.postChanges("/api/v1/change/list", map[string]any{"project_id": numericProjectID}, "changes")
}

// GetChange loads a single change by numeric ID.
func (c HTTPClient) GetChange(id int) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/get", map[string]any{"id": id})
}

// CreateChange creates a change.
func (c HTTPClient) CreateChange(input dto.ChangeCreateInput) (dto.Change, error) {
    if input.ProjectID <= 0 {
        return dto.Change{}, fmt.Errorf("project ID must be a valid positive number")
    }
    payload := map[string]any{
        "project_id": input.ProjectID,
        "title":      input.Title,
        "def":        input.Def,
    }
    if strings.TrimSpace(input.RefUUID) != "" {
        payload["ref_uuid"] = input.RefUUID
    }
    return c.postChange("/api/v1/change/create", payload)
}

// ReferenceChange assigns or refreshes backend-owned change Flow identity.
func (c HTTPClient) ReferenceChange(id int) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/assign-flow", map[string]any{"id": id})
}

// UpdateChangeTitle updates a change title.
func (c HTTPClient) UpdateChangeTitle(id int, title string) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/update-title", map[string]any{"id": id, "title": title})
}

// UpdateChangeDef updates a change definition.
func (c HTTPClient) UpdateChangeDef(id int, def string, agentEdit bool) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/update-def", map[string]any{
        "id":         id,
        "def":        def,
        "agent_edit": agentEdit,
    })
}

// UpdateChangeSpec updates a change spec.
func (c HTTPClient) UpdateChangeSpec(id int, spec string, agentEdit bool) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/update-spec", map[string]any{
        "id":         id,
        "spec":       spec,
        "agent_edit": agentEdit,
    })
}

// UpdateChangePR updates a change pull request body.
func (c HTTPClient) UpdateChangePR(id int, pr string, agentEdit bool) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/update-pr", map[string]any{
        "id":         id,
        "pr":         pr,
        "agent_edit": agentEdit,
    })
}

// UpdateChangePRUrl updates a change pull request URL.
func (c HTTPClient) UpdateChangePRUrl(id int, prURL string) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/update-pr-url", map[string]any{
        "id":     id,
        "pr_url": prURL,
    })
}

// UpdateChangeTypes updates change type slugs.
func (c HTTPClient) UpdateChangeTypes(id int, changeTypes []string) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/update-change-types", map[string]any{
        "id":           id,
        "change_types": changeTypes,
    })
}

// UpdateChangePhase updates the change phase slug.
func (c HTTPClient) UpdateChangePhase(id int, changePhase string) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/update-phase", map[string]any{
        "id":           id,
        "change_phase": changePhase,
    })
}

// UpdateChangeOpen updates the change open flag.
func (c HTTPClient) UpdateChangeOpen(id int, open bool) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/update-open", map[string]any{
        "id":   id,
        "open": open,
    })
}

// UpdateChangeEpic updates or clears the change epic.
func (c HTTPClient) UpdateChangeEpic(id int, epicID *int) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/change/update-epic", map[string]any{"id": id, "epic_id": epicID})
}

// CreateTestCase creates a test case for a change and returns refreshed change data.
func (c HTTPClient) CreateTestCase(changeID int, scenario string) (dto.Change, error) {
    if changeID <= 0 {
        return dto.Change{}, fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postChange("/api/v1/test-case/create", map[string]any{
        "change_id": changeID,
        "scenario":  scenario,
    })
}

// UpdateTestCase updates a test case scenario and returns refreshed change data.
func (c HTTPClient) UpdateTestCase(id int, scenario string) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("test case ID must be a valid positive number")
    }
    return c.postChange("/api/v1/test-case/update", map[string]any{
        "id":       id,
        "scenario": scenario,
    })
}

// UpdateTestCaseDone updates the test case done flag and returns refreshed change data.
func (c HTTPClient) UpdateTestCaseDone(id int, done bool) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("test case ID must be a valid positive number")
    }
    return c.postChange("/api/v1/test-case/update-done", map[string]any{
        "id":   id,
        "done": done,
    })
}

// DeleteTestCase deletes a test case and returns refreshed change data.
func (c HTTPClient) DeleteTestCase(id int) (dto.Change, error) {
    if id <= 0 {
        return dto.Change{}, fmt.Errorf("test case ID must be a valid positive number")
    }
    return c.postChange("/api/v1/test-case/delete", map[string]any{"id": id})
}

// DeleteChange deletes a change by numeric ID.
func (c HTTPClient) DeleteChange(id int) error {
    if id <= 0 {
        return fmt.Errorf("change ID must be a valid positive number")
    }
    return c.postNoContent("/api/v1/change/delete", map[string]any{"id": id})
}

// ListEpics loads epic selector options for a project.
func (c HTTPClient) ListEpics(projectID string) ([]dto.Option, error) {
    numericProjectID, err := numericCurrentProjectID(projectID)
    if err != nil {
        return nil, err
    }
    return c.postOptions("/api/v1/epic/list", map[string]any{"project_id": numericProjectID}, "epics")
}

// ListPhases loads change phase selector options.
func (c HTTPClient) ListPhases() ([]dto.Option, error) {
    return c.postOptions("/api/v1/options/change-phases-list", map[string]any{}, "phases")
}

// ListTypes loads change type selector options.
func (c HTTPClient) ListTypes() ([]dto.Option, error) {
    return c.postOptions("/api/v1/options/change-types-list", map[string]any{}, "types")
}

func (c HTTPClient) postOptions(path string, payload any, group string) ([]dto.Option, error) {
    data, err := c.postJSON(path, payload)
    if err != nil {
        return nil, err
    }
    return findOptions(data, group), nil
}

func (c HTTPClient) postProjects(path string, payload any, group string) ([]dto.Project, error) {
    data, err := c.postJSON(path, payload)
    if err != nil {
        return nil, err
    }
    return findProjects(data, group), nil
}

func (c HTTPClient) postProject(path string, payload any) (dto.Project, error) {
    data, err := c.postJSON(path, payload)
    if err != nil {
        return dto.Project{}, err
    }
    project, ok := findProject(data)
    if !ok {
        return dto.Project{}, fmt.Errorf("project response missing project")
    }
    return project, nil
}

func (c HTTPClient) postChanges(path string, payload any, group string) ([]dto.Change, error) {
    data, err := c.postJSON(path, payload)
    if err != nil {
        return nil, err
    }
    return findChanges(data, group), nil
}

func (c HTTPClient) postChange(path string, payload any) (dto.Change, error) {
    data, err := c.postJSON(path, payload)
    if err != nil {
        return dto.Change{}, err
    }
    change, ok := findChange(data)
    if !ok {
        return dto.Change{}, fmt.Errorf("change response missing change")
    }
    return change, nil
}

func (c HTTPClient) postNoContent(path string, payload any) error {
    _, err := c.postJSON(path, payload)
    return err
}

func (c HTTPClient) postJSON(path string, payload any) (any, error) {
    body, err := json.Marshal(payload)
    if err != nil {
        return nil, err
    }
    req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")

    client := c.Client
    if client == nil {
        client = http.DefaultClient
    }
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode > 299 {
        var data map[string]any
        if err := json.NewDecoder(resp.Body).Decode(&data); err == nil {
            if message := firstString(data, "message", "error"); message != "" {
                return nil, fmt.Errorf("%s", message)
            }
        }
        return nil, fmt.Errorf("backend returned %s", resp.Status)
    }
    if resp.StatusCode == http.StatusNoContent {
        return nil, nil
    }

    var data any
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return nil, err
    }
    return data, nil
}

func findOptions(value any, group string) []dto.Option {
    switch typed := value.(type) {
    case []any:
        return optionsFromArray(typed)
    case map[string]any:
        for key, candidate := range typed {
            if key == group {
                if list, ok := candidate.([]any); ok {
                    return optionsFromArray(list)
                }
            }
        }
        for _, candidate := range typed {
            if nested := findOptions(candidate, group); len(nested) > 0 {
                return nested
            }
        }
    }
    return nil
}

func findProjects(value any, group string) []dto.Project {
    switch typed := value.(type) {
    case []any:
        return projectsFromArray(typed)
    case map[string]any:
        for key, candidate := range typed {
            if key == group {
                if list, ok := candidate.([]any); ok {
                    return projectsFromArray(list)
                }
            }
        }
        for _, candidate := range typed {
            if nested := findProjects(candidate, group); len(nested) > 0 {
                return nested
            }
        }
    }
    return nil
}

func findChanges(value any, group string) []dto.Change {
    switch typed := value.(type) {
    case []any:
        return changesFromArray(typed)
    case map[string]any:
        for key, candidate := range typed {
            if key == group {
                if list, ok := candidate.([]any); ok {
                    return changesFromArray(list)
                }
            }
        }
        for _, candidate := range typed {
            if nested := findChanges(candidate, group); len(nested) > 0 {
                return nested
            }
        }
    }
    return nil
}

func findChange(value any) (dto.Change, bool) {
    switch typed := value.(type) {
    case []any:
        changes := changesFromArray(typed)
        if len(changes) > 0 {
            return changes[0], true
        }
    case map[string]any:
        for _, key := range []string{"change", "data", "result"} {
            if candidate, ok := typed[key]; ok {
                if change, found := findChange(candidate); found {
                    if testCases := findTestCases(typed); len(testCases) > 0 {
                        change.TestCases = testCases
                    }
                    return change, true
                }
            }
        }
        change := changeFromMap(typed)
        if change.ID != "" || change.Title != "" {
            if testCases := findTestCases(typed); len(testCases) > 0 {
                change.TestCases = testCases
            }
            return change, true
        }
        for _, candidate := range typed {
            if change, found := findChange(candidate); found {
                return change, true
            }
        }
    }
    return dto.Change{}, false
}

func findTestCases(value any) []dto.TestCase {
    switch typed := value.(type) {
    case []any:
        return testCasesFromArray(typed)
    case map[string]any:
        for key, candidate := range typed {
            if key == "test_cases" {
                if list, ok := candidate.([]any); ok {
                    return testCasesFromArray(list)
                }
            }
        }
    }
    return nil
}

func findProject(value any) (dto.Project, bool) {
    switch typed := value.(type) {
    case []any:
        projects := projectsFromArray(typed)
        if len(projects) > 0 {
            return projects[0], true
        }
    case map[string]any:
        for _, key := range []string{"project", "data", "result"} {
            if candidate, ok := typed[key]; ok {
                if project, found := findProject(candidate); found {
                    return project, true
                }
            }
        }
        project := projectFromMap(typed)
        if project.ID != "" || project.Name != "" {
            return project, true
        }
        for _, candidate := range typed {
            if project, found := findProject(candidate); found {
                return project, true
            }
        }
    }
    return dto.Project{}, false
}

func optionsFromArray(values []any) []dto.Option {
    options := make([]dto.Option, 0, len(values))
    for _, value := range values {
        switch typed := value.(type) {
        case string:
            options = append(options, dto.Option{ID: typed, Label: typed})
        case map[string]any:
            option := dto.Option{
                ID:    firstString(typed, "id", "project_id", "epic_id", "slug", "value"),
                Label: firstString(typed, "name", "title", "slug", "label", "value", "id"),
                Color: firstString(typed, "color", "colour", "lip_gloss_color", "lipgloss_color"),
            }
            if option.Label == "" {
                continue
            }
            if option.ID == "" {
                option.ID = option.Label
            }
            options = append(options, option)
        }
    }
    return options
}

func projectsFromArray(values []any) []dto.Project {
    projects := make([]dto.Project, 0, len(values))
    for _, value := range values {
        typed, ok := value.(map[string]any)
        if !ok {
            continue
        }
        project := projectFromMap(typed)
        if project.ID == "" && project.Name == "" {
            continue
        }
        projects = append(projects, project)
    }
    return projects
}

func changesFromArray(values []any) []dto.Change {
    changes := make([]dto.Change, 0, len(values))
    for _, value := range values {
        typed, ok := value.(map[string]any)
        if !ok {
            continue
        }
        change := changeFromMap(typed)
        if change.ID == "" && change.Title == "" {
            continue
        }
        changes = append(changes, change)
    }
    return changes
}

func testCasesFromArray(values []any) []dto.TestCase {
    testCases := make([]dto.TestCase, 0, len(values))
    for _, value := range values {
        typed, ok := value.(map[string]any)
        if !ok {
            continue
        }
        testCase := dto.TestCase{
            ID:       firstString(typed, "id", "test_case_id"),
            Scenario: firstString(typed, "scenario"),
            Done:     firstBool(typed, "done"),
            ChangeID: firstString(typed, "change_id"),
        }
        if testCase.ID == "" && testCase.Scenario == "" {
            continue
        }
        testCases = append(testCases, testCase)
    }
    return testCases
}

func projectFromMap(values map[string]any) dto.Project {
    return dto.Project{
        ID:          firstString(values, "id", "project_id"),
        Name:        firstString(values, "name", "title"),
        LastRef:     firstInt(values, "last_ref"),
        ChangeCount: firstInt(values, "change_count"),
        Created:     firstString(values, "created", "created_at"),
        Modified:    firstString(values, "modified", "updated", "updated_at"),
    }
}

func changeFromMap(values map[string]any) dto.Change {
    return dto.Change{
        ID:          firstString(values, "id", "change_id"),
        RefUUID:     firstString(values, "ref_uuid"),
        Ref:         firstString(values, "ref"),
        Slug:        firstString(values, "slug"),
        ProjectID:   firstString(values, "project_id"),
        EpicID:      firstString(values, "epic_id"),
        EpicName:    firstString(values, "epic_name", "epic_title", "epic"),
        ChangePhase: firstString(values, "change_phase", "phase"),
        ChangeTypes: firstStringSlice(values, "change_types", "types"),
        Title:       firstString(values, "title", "name"),
        Def:         firstString(values, "def"),
        Spec:        firstString(values, "spec"),
        PR:          firstString(values, "pr"),
        PRUrl:       firstString(values, "pr_url"),
        AgentEdit:   firstBool(values, "agent_edit"),
        Open:        firstBool(values, "open"),
        Done:        firstInt(values, "done_tc", "done"),
        Total:       firstInt(values, "total_tc", "total"),
        Completed:   firstInt(values, "completed", "completed_pct"),
        Created:     firstString(values, "created", "created_at"),
        Modified:    firstString(values, "modified", "updated", "updated_at"),
    }
}

func numericCurrentProjectID(projectID string) (int, error) {
    projectID = strings.TrimSpace(projectID)
    if projectID == "" {
        return 0, fmt.Errorf("current project is required")
    }
    numericProjectID, err := strconv.Atoi(projectID)
    if err != nil || numericProjectID <= 0 {
        return 0, fmt.Errorf("current project must be numeric")
    }
    return numericProjectID, nil
}

func firstString(values map[string]any, keys ...string) string {
    for _, key := range keys {
        value, ok := values[key]
        if !ok {
            continue
        }
        switch typed := value.(type) {
        case string:
            return typed
        case float64:
            return strconv.FormatInt(int64(typed), 10)
        case int:
            return strconv.Itoa(typed)
        }
    }
    return ""
}

func firstInt(values map[string]any, keys ...string) int {
    for _, key := range keys {
        value, ok := values[key]
        if !ok {
            continue
        }
        switch typed := value.(type) {
        case float64:
            return int(typed)
        case int:
            return typed
        case string:
            parsed, err := strconv.Atoi(strings.TrimSpace(typed))
            if err == nil {
                return parsed
            }
        }
    }
    return 0
}

func firstStringSlice(values map[string]any, keys ...string) []string {
    for _, key := range keys {
        value, ok := values[key]
        if !ok {
            continue
        }
        switch typed := value.(type) {
        case []any:
            items := make([]string, 0, len(typed))
            for _, item := range typed {
                switch value := item.(type) {
                case string:
                    items = append(items, value)
                case float64:
                    items = append(items, strconv.FormatInt(int64(value), 10))
                }
            }
            return items
        case []string:
            return append([]string(nil), typed...)
        case string:
            if strings.TrimSpace(typed) == "" {
                return nil
            }
            return strings.Split(typed, "|")
        }
    }
    return nil
}

func firstBool(values map[string]any, keys ...string) bool {
    for _, key := range keys {
        value, ok := values[key]
        if !ok {
            continue
        }
        switch typed := value.(type) {
        case bool:
            return typed
        case string:
            return typed == "true"
        }
    }
    return false
}
