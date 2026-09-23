package app

import (
    "fmt"
    "sort"
    "strconv"
    "strings"

    "cli/internal/changes"
    "cli/internal/dto"

    tea "github.com/charmbracelet/bubbletea"
)

func (m Model) saveChangeCreate() (tea.Model, tea.Cmd) {
    return m.saveChangeCreateValue(m.input.Value())
}

func (m Model) saveChangeCreateValue(def string) (tea.Model, tea.Cmd) {
    projectID, err := currentProjectNumericID(m.currentProject.ID)
    if err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, nil
    }
    m.status = "saving"
    return m, changeCreateCommand(m.client, projectID, def)
}

func (m Model) saveChangeUpdate() (tea.Model, tea.Cmd) {
    return m.saveChangeUpdateValue(m.input.Value())
}

func (m Model) saveChangeUpdateValue(spec string) (tea.Model, tea.Cmd) {
    id, err := changeNumericID(m.changeList.Detail)
    if err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, nil
    }
    m.status = "saving"
    return m, changeUpdateCommand(m.client, id, m.changeList.Detail, spec)
}

func (m Model) saveTestCaseCreateValue(scenario string) (tea.Model, tea.Cmd) {
    changeID, err := changeNumericID(m.changeList.Detail)
    if err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, nil
    }
    if strings.TrimSpace(scenario) == "" {
        m.err = "test case scenario is required"
        m.status = "validation failed"
        return m, nil
    }
    m.status = "saving test case"
    return m, testCaseCreateCommand(m.client, changeID, scenario)
}

func (m Model) saveTestCaseUpdateValue(scenario string) (tea.Model, tea.Cmd) {
    testCaseID, err := testCaseNumericID(m.activeTestCase.ID)
    if err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, nil
    }
    if strings.TrimSpace(scenario) == "" {
        m.err = "test case scenario is required"
        m.status = "validation failed"
        return m, nil
    }
    m.status = "saving test case"
    return m, testCaseUpdateCommand(m.client, testCaseID, scenario)
}

func changeCreateCommand(client appClient, projectID int, def string) tea.Cmd {
    return func() tea.Msg {
        parsed, err := changes.ParseDefStructure(def)
        if err != nil {
            return changeSavedMsg{source: ChangeCreateState, err: err}
        }
        created, err := client.CreateChange(dto.ChangeCreateInput{
            ProjectID: projectID,
            Title:     parsed.Title,
            Def:       parsed.Def,
        })
        if err != nil {
            return changeSavedMsg{source: ChangeCreateState, err: err}
        }
        id, err := changeNumericID(created)
        if err != nil {
            return changeSavedMsg{source: ChangeCreateState, err: err}
        }
        if err := updateArtifactTypes(client, id, parsed.ChangeTypes, parsed.ChangeTypesPresent); err != nil {
            return changeSavedMsg{source: ChangeCreateState, change: created, err: err}
        }
        change, err := client.GetChange(id)
        if err != nil {
            return changeSavedMsg{source: ChangeCreateState, change: created, reloadErr: err}
        }
        return changeSavedMsg{source: ChangeCreateState, change: change}
    }
}

func changeUpdateCommand(client appClient, id int, original dto.Change, spec string) tea.Cmd {
    return func() tea.Msg {
        parsed, err := parseChangeArtifact(spec)
        if err != nil {
            return changeSavedMsg{source: ChangeUpdateState, err: err}
        }
        if parsed.Title != original.Title {
            if _, err := client.UpdateChangeTitle(id, parsed.Title); err != nil {
                return changeSavedMsg{source: ChangeUpdateState, err: err}
            }
        }
        if parsed.Spec != original.Spec {
            if _, err := client.UpdateChangeSpec(id, parsed.Spec, false); err != nil {
                return changeSavedMsg{source: ChangeUpdateState, err: err}
            }
        }
        if err := updateArtifactTypes(client, id, parsed.ChangeTypes, parsed.ChangeTypesPresent); err != nil {
            return changeSavedMsg{source: ChangeUpdateState, err: err}
        }
        change, err := client.GetChange(id)
        return changeSavedMsg{source: ChangeUpdateState, change: change, err: err}
    }
}

func changeGetCommand(client appClient, id int) tea.Cmd {
    return func() tea.Msg {
        change, err := client.GetChange(id)
        return changeLoadedMsg{id: id, change: change, err: err}
    }
}

func changeDeleteCommand(client appClient, change dto.Change, target State) tea.Cmd {
    return func() tea.Msg {
        id, err := changeNumericID(change)
        if err != nil {
            return changeDeletedMsg{target: target, err: err}
        }
        return changeDeletedMsg{target: target, err: client.DeleteChange(id)}
    }
}

func changeReferenceCommand(client appClient, change dto.Change) tea.Cmd {
    return func() tea.Msg {
        id, err := changeNumericID(change)
        if err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        referenced, err := client.ReferenceChange(id)
        if err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        if strings.TrimSpace(referenced.Ref) == "" || strings.TrimSpace(referenced.Slug) == "" {
            return changeSavedMsg{source: ChangeDetailsState, change: referenced, err: fmt.Errorf("referenced change must include ref and slug")}
        }
        refreshed, err := client.GetChange(id)
        if err != nil {
            return changeSavedMsg{source: ChangeDetailsState, change: referenced, err: err}
        }
        if strings.TrimSpace(refreshed.Ref) == "" || strings.TrimSpace(refreshed.Slug) == "" {
            return changeSavedMsg{source: ChangeDetailsState, change: refreshed, err: fmt.Errorf("referenced change must include ref and slug")}
        }
        if err := reconcileChangeBranch(refreshed.Ref, refreshed.Slug); err != nil {
            return changeSavedMsg{source: ChangeDetailsState, change: refreshed, err: err}
        }
        return changeSavedMsg{source: ChangeDetailsState, change: refreshed}
    }
}

func testCaseCreateCommand(client appClient, changeID int, scenario string) tea.Cmd {
    return func() tea.Msg {
        change, err := client.CreateTestCase(changeID, scenario)
        return changeSavedMsg{source: TestCaseCreateState, change: change, err: err}
    }
}

func testCaseUpdateCommand(client appClient, testCaseID int, scenario string) tea.Cmd {
    return func() tea.Msg {
        change, err := client.UpdateTestCase(testCaseID, scenario)
        return changeSavedMsg{source: TestCaseUpdateState, change: change, err: err}
    }
}

func testCaseDeleteCommand(client appClient, testCase dto.TestCase) tea.Cmd {
    return func() tea.Msg {
        testCaseID, err := testCaseNumericID(testCase.ID)
        if err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        change, err := client.DeleteTestCase(testCaseID)
        return changeSavedMsg{source: ChangeDetailsState, change: change, err: err}
    }
}

func changeDetailFieldUpdateCommand(client appClient, change dto.Change, field detailEditField, selected dto.Option) tea.Cmd {
    return func() tea.Msg {
        id, err := changeNumericID(change)
        if err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        switch field {
        case detailEditPhase:
            if _, err := client.UpdateChangePhase(id, selected.ID); err != nil {
                return changeSavedMsg{source: ChangeDetailsState, err: err}
            }
        case detailEditTypes:
            changeTypes := toggleChangeType(change.ChangeTypes, selected)
            if _, err := client.UpdateChangeTypes(id, changeTypes); err != nil {
                return changeSavedMsg{source: ChangeDetailsState, err: err}
            }
        case detailEditEpic:
            epicID, err := selectedEpicID(selected)
            if err != nil {
                return changeSavedMsg{source: ChangeDetailsState, err: err}
            }
            if _, err := client.UpdateChangeEpic(id, epicID); err != nil {
                return changeSavedMsg{source: ChangeDetailsState, err: err}
            }
        default:
            return changeSavedMsg{source: ChangeDetailsState, err: fmt.Errorf("unsupported change detail field: %s", field)}
        }
        change, err := client.GetChange(id)
        return changeSavedMsg{source: ChangeDetailsState, change: change, err: err}
    }
}

func changeDetailTypesUpdateCommand(client appClient, change dto.Change, changeTypes []string) tea.Cmd {
    return func() tea.Msg {
        id, err := changeNumericID(change)
        if err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        if _, err := client.UpdateChangeTypes(id, normalizeTypeSet(changeTypes)); err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        change, err := client.GetChange(id)
        return changeSavedMsg{source: ChangeDetailsState, change: change, err: err}
    }
}

func changeDetailOpenUpdateCommand(client appClient, change dto.Change) tea.Cmd {
    return func() tea.Msg {
        id, err := changeNumericID(change)
        if err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        if _, err := client.UpdateChangeOpen(id, !change.Open); err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        change, err := client.GetChange(id)
        return changeSavedMsg{source: ChangeDetailsState, change: change, err: err}
    }
}

func changeDetailTestCaseDoneUpdateCommand(client appClient, change dto.Change, row changes.DetailRow) tea.Cmd {
    return func() tea.Msg {
        changeID, err := changeNumericID(change)
        if err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        testCaseID, err := testCaseNumericID(row.TestCaseID)
        if err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        if _, err := client.UpdateTestCaseDone(testCaseID, !row.TestCaseDone); err != nil {
            return changeSavedMsg{source: ChangeDetailsState, err: err}
        }
        change, err := client.GetChange(changeID)
        return changeSavedMsg{source: ChangeDetailsState, change: change, err: err}
    }
}

func changeDetailTextUpdateCommand(client appClient, source State, change dto.Change, field detailEditField, value string) tea.Cmd {
    return func() tea.Msg {
        id, err := changeNumericID(change)
        if err != nil {
            return changeSavedMsg{source: source, err: err}
        }
        artifactTypes := []string(nil)
        artifactTypesPresent := false
        switch field {
        case detailEditTitle:
            if _, err := client.UpdateChangeTitle(id, value); err != nil {
                return changeSavedMsg{source: source, err: err}
            }
        case detailEditSpec:
            if strings.TrimSpace(value) == "" {
                return changeSavedMsg{source: source, err: fmt.Errorf("spec is required")}
            }
            if _, err := client.UpdateChangeSpec(id, value, false); err != nil {
                return changeSavedMsg{source: source, err: err}
            }
            artifactTypes, artifactTypesPresent = changes.ParseArtifactTypes(value)
        case detailEditDef:
            if _, err := client.UpdateChangeDef(id, value, false); err != nil {
                return changeSavedMsg{source: source, err: err}
            }
            artifactTypes, artifactTypesPresent = changes.ParseArtifactTypes(value)
        case detailEditPullRequest:
            if strings.TrimSpace(value) == "" {
                return changeSavedMsg{source: source, err: fmt.Errorf("PR is required")}
            }
            if _, err := client.UpdateChangePR(id, value, false); err != nil {
                return changeSavedMsg{source: source, err: err}
            }
            artifactTypes, artifactTypesPresent = changes.ParseArtifactTypes(value)
        case detailEditPRUrl:
            if strings.TrimSpace(value) == "" {
                return changeSavedMsg{source: source, err: fmt.Errorf("PR URL is required")}
            }
            if _, err := client.UpdateChangePRUrl(id, value); err != nil {
                return changeSavedMsg{source: source, err: err}
            }
        default:
            return changeSavedMsg{source: source, err: fmt.Errorf("unsupported change detail text field: %s", field)}
        }
        if err := updateArtifactTypes(client, id, artifactTypes, artifactTypesPresent); err != nil {
            return changeSavedMsg{source: source, err: err}
        }
        change, err := client.GetChange(id)
        return changeSavedMsg{source: source, change: change, err: err}
    }
}

func updateArtifactAndReload(client appClient, id int, field detailEditField, value string, agentEdit bool) (dto.Change, error) {
    if err := validateArtifactWrite(field, value); err != nil {
        return dto.Change{}, err
    }
    var updated dto.Change
    var err error
    switch field {
    case detailEditDef:
        updated, err = client.UpdateChangeDef(id, value, agentEdit)
    case detailEditSpec:
        updated, err = client.UpdateChangeSpec(id, value, agentEdit)
    case detailEditPullRequest:
        updated, err = client.UpdateChangePR(id, value, agentEdit)
    default:
        return dto.Change{}, fmt.Errorf("unsupported artifact write field: %s", field)
    }
    if err != nil {
        return updated, err
    }
    changeTypes, present := changes.ParseArtifactTypes(value)
    if err := updateArtifactTypes(client, id, changeTypes, present); err != nil {
        return updated, err
    }
    change, err := client.GetChange(id)
    if err != nil {
        return updated, err
    }
    return change, nil
}

func parseChangeArtifact(spec string) (changes.ParsedSpec, error) {
    return changes.ParseSpecStructure(spec)
}

func updateArtifactTypes(client appClient, id int, changeTypes []string, present bool) error {
    if !present {
        return nil
    }
    _, err := client.UpdateChangeTypes(id, changeTypes)
    return err
}

func changeNumericID(change dto.Change) (int, error) {
    id, err := strconv.Atoi(strings.TrimSpace(change.ID))
    if err != nil || id <= 0 {
        return 0, fmt.Errorf("change ID must be a valid positive number")
    }
    return id, nil
}

func testCaseNumericID(idValue string) (int, error) {
    id, err := strconv.Atoi(strings.TrimSpace(idValue))
    if err != nil || id <= 0 {
        return 0, fmt.Errorf("test case ID must be a valid positive number")
    }
    return id, nil
}

func currentProjectNumericID(projectID string) (int, error) {
    id, err := strconv.Atoi(strings.TrimSpace(projectID))
    if err != nil || id <= 0 {
        return 0, fmt.Errorf("current project must be numeric")
    }
    return id, nil
}

func toggleChangeType(current []string, selected dto.Option) []string {
    selectedID := strings.TrimSpace(selected.ID)
    if selectedID == "" {
        selectedID = strings.TrimSpace(selected.Label)
    }
    next := make([]string, 0, len(current)+1)
    removed := false
    for _, changeType := range current {
        if changeType == selectedID || changeType == selected.Label {
            removed = true
            continue
        }
        next = append(next, changeType)
    }
    if !removed && selectedID != "" {
        next = append(next, selectedID)
    }
    sort.Strings(next)
    return next
}

func normalizeTypeSet(values []string) []string {
    seen := make(map[string]struct{}, len(values))
    next := make([]string, 0, len(values))
    for _, value := range values {
        value = strings.TrimSpace(value)
        if value == "" {
            continue
        }
        if _, ok := seen[value]; ok {
            continue
        }
        seen[value] = struct{}{}
        next = append(next, value)
    }
    sort.Strings(next)
    return next
}

func selectedEpicID(selected dto.Option) (*int, error) {
    if selected.ID == "@none" {
        return nil, nil
    }
    epicID, err := strconv.Atoi(strings.TrimSpace(selected.ID))
    if err != nil || epicID <= 0 {
        return nil, fmt.Errorf("epic ID must be numeric")
    }
    return &epicID, nil
}
