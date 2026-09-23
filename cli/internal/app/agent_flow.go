package app

import (
    "context"
    "fmt"
    "strings"
    "time"

    "cli/internal/agent"
    "cli/internal/changes"
    "cli/internal/dto"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/gofrs/uuid/v5"
)

func (m Model) beginAgentNewChange() (tea.Model, tea.Cmd) {
    if _, err := currentProjectNumericID(m.currentProject.ID); err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, nil
    }
    newUUID := m.newChangeUUID
    if newUUID == nil {
        newUUID = uuid.NewV7
    }
    refUUID, err := newUUID()
    if err != nil {
        m.err = err.Error()
        m.status = "agent failed"
        return m, nil
    }
    m.agentFlow = agent.NewChangeModel(m.agentWorkspace, refUUID.String())
    if err := m.agentFlow.Workspace.InitializeChange(); err != nil {
        m.err = err.Error()
        m.status = "agent failed"
        return m, nil
    }
    return m.openAgentDefEditor(false, "")
}

func (m Model) openAgentDefEditor(replace bool, initialContent string) (tea.Model, tea.Cmd) {
    if replace && m.agentFlow.Workspace.RootDir == "" {
        if err := m.agentFlow.Workspace.ResetDef(); err != nil {
            m.err = err.Error()
            m.status = "agent failed"
            return m, nil
        }
    }
    m.agentFlow.DefEntryContent = initialContent
    m.agentFlow.Stage = agent.StageDefEntry
    m.state = CreateDefState
    m.status = "agent def"
    return m.openPersistentEditor(CreateDefState, m.agentFlow.Workspace.DefPath())
}

func (m Model) handleAgentEditorFinished(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
    if msg.err != nil {
        m.err = msg.err.Error()
        m.status = "editor failed"
        return m, tea.ClearScreen
    }
    switch m.agentFlow.Stage {
    case agent.StageDefEntry:
        if m.agentFlow.Workspace.RootDir == "" {
            return m.handleLegacyAgentEditorFinished(msg.content)
        }
        equal, err := m.agentFlow.Workspace.EqualDefFiles()
        if err != nil {
            m.err = err.Error()
            m.status = "agent failed"
            return m, tea.ClearScreen
        }
        if equal {
            updated, _ := m.discardAgentDef(ChangesListState, "cancel", true)
            return updated, tea.ClearScreen
        }
        m.agentFlow.DefEntryContent = msg.content
        m.status = "agent def"
        m.agentFlow.Stage = agent.StageCreateConfirmation
        m.openDropdown(CreateDefState, dropdownDef, CreateDefState, CreateDefState, "Create Change?", []dto.Option{
            {ID: "/yes", Label: "/yes"},
            {ID: "/no", Label: "/no"},
        }, false)
        return m, tea.ClearScreen
    default:
        return m, tea.ClearScreen
    }
}

func (m Model) handleLegacyAgentEditorFinished(content string) (tea.Model, tea.Cmd) {
    if err := m.agentFlow.Workspace.WriteDef(content); err != nil {
        m.err = err.Error()
        m.status = "agent failed"
        return m, tea.ClearScreen
    }
    m.agentFlow.DefEntryContent = content
    m.agentFlow.Stage = agent.StageCreateConfirmation
    m.openDropdown(CreateDefState, dropdownDef, CreateDefState, CreateDefState, "Create Change?", []dto.Option{
        {ID: "/yes", Label: "/yes"},
        {ID: "/no", Label: "/no"},
    }, false)
    return m, tea.ClearScreen
}

func (m Model) showNewChangeError(err error) Model {
    m.err = err.Error()
    m.status = "agent failed"
    m.openDropdown(CreateDefState, dropdownDef, CreateDefState, CreateDefState, err.Error(), []dto.Option{
        {ID: "/fix", Label: "/fix"},
        {ID: "/cancel", Label: "/cancel"},
    }, false)
    return m
}

func (m Model) showPersistedChangeError(err error) Model {
    m.err = err.Error()
    m.status = "save failed"
    m.state = ChangeDetailsState
    m.agentFlow.Stage = agent.StageIdle
    m.openDropdown(ChangeDetailsState, dropdownPersistedDef, ChangeDetailsState, ChangeDetailsState, err.Error(), []dto.Option{
        {ID: "/fix", Label: "/fix"},
        {ID: "/cancel", Label: "/cancel"},
    }, false)
    return m
}

func changeArtifactEditLoadCommand(client appClient, id int, field detailEditField) tea.Cmd {
    return func() tea.Msg {
        change, err := client.GetChange(id)
        return changeArtifactEditLoadedMsg{id: id, field: field, change: change, err: err}
    }
}

func (m Model) handleArtifactEditLoaded(msg changeArtifactEditLoadedMsg) (tea.Model, tea.Cmd) {
    if m.state != ChangeDetailsState || m.detailEditField != msg.field {
        return m, nil
    }
    currentID, err := changeNumericID(m.changeList.Detail)
    if err != nil || currentID != msg.id {
        return m, nil
    }
    if msg.err != nil {
        m.err = msg.err.Error()
        m.status = "load failed"
        return m, nil
    }
    loadedID, err := changeNumericID(msg.change)
    if err != nil || loadedID != msg.id {
        m.err = "loaded Change does not match the selected Change"
        m.status = "load failed"
        return m, nil
    }
    if _, err := uuid.FromString(strings.TrimSpace(msg.change.RefUUID)); err != nil {
        m.err = "change ref UUID must be a valid UUID"
        m.status = "validation failed"
        return m, nil
    }
    operation, content, err := artifactWriteSelection(msg.field, msg.change)
    if err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, nil
    }
    m.changeList = m.changeList.WithDetail(msg.change)
    m.agentFlow = agent.NewArtifactModel(m.agentWorkspace, msg.change.RefUUID, operation)
    if err := m.agentFlow.Workspace.PrepareArtifact(content); err != nil {
        m.err = err.Error()
        m.status = "agent failed"
        return m, nil
    }
    sessionID, err := m.agentFlow.Workspace.ReadSessionID()
    if err != nil {
        m.err = err.Error()
        m.status = "agent failed"
        return m, nil
    }
    m.agentFlow.SessionID = sessionID
    m.agentFlow.Stage = agent.StageArtifactEntry
    m = m.setPromptValue(content)
    m.status = "editor"
    return m.openPersistentEditor(ChangeDetailsState, m.agentFlow.Workspace.OutputPath())
}

func (m Model) handleArtifactEditorFinished(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
    if msg.err != nil {
        m.err = msg.err.Error()
        m.status = "editor failed"
        return m, tea.ClearScreen
    }
    equal, err := m.agentFlow.Workspace.EqualDefFiles()
    if err != nil {
        m.err = err.Error()
        m.status = "agent failed"
        return m, tea.ClearScreen
    }
    if equal {
        m.agentFlow = agent.NewModelWithWorkspace(m.agentWorkspace)
        m.detailEditField = ""
        m = m.setPromptValue("")
        m.status = "cancel"
        return m, tea.ClearScreen
    }
    if err := validateArtifactWrite(m.detailEditField, msg.content); err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, tea.ClearScreen
    }
    id, err := changeNumericID(m.changeList.Detail)
    if err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, tea.ClearScreen
    }
    m.status = "saving " + string(m.detailEditField)
    return m, tea.Batch(tea.ClearScreen, changeArtifactUpdateForWriteCommand(m.client, id, m.detailEditField, msg.content))
}

func artifactWriteSelection(field detailEditField, change dto.Change) (agent.WriteOperation, string, error) {
    switch field {
    case detailEditDef:
        return agent.DefWriteOperation, change.Def, nil
    case detailEditSpec:
        return agent.SpecWriteOperation, change.Spec, nil
    case detailEditPullRequest:
        return agent.PRWriteOperation, change.PR, nil
    default:
        return "", "", fmt.Errorf("unsupported artifact write field: %s", field)
    }
}

func validateArtifactWrite(field detailEditField, content string) error {
    switch field {
    case detailEditDef:
        _, err := changes.ParseDefStructure(content)
        return err
    case detailEditSpec:
        if strings.TrimSpace(content) == "" {
            return fmt.Errorf("spec is required")
        }
    case detailEditPullRequest:
        if strings.TrimSpace(content) == "" {
            return fmt.Errorf("PR is required")
        }
    default:
        return fmt.Errorf("unsupported artifact write field: %s", field)
    }
    return nil
}

func (m Model) handleUpdateDefEditorFinished(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
    if msg.err != nil {
        m.err = msg.err.Error()
        m.status = "editor failed"
        return m, tea.ClearScreen
    }
    if m.agentFlow.Stage == agent.StagePersistedDefEntry {
        equal, err := m.agentFlow.Workspace.EqualDefFiles()
        if err != nil {
            m = m.showPersistedChangeError(err)
            return m, tea.ClearScreen
        }
        if equal {
            m.agentFlow.Stage = agent.StageIdle
            m.state = ChangeDetailsState
            m.status = "cancel"
            return m, tea.ClearScreen
        }
    }
    if err := m.agentFlow.Workspace.WriteDef(msg.content); err != nil {
        m.err = err.Error()
        m.status = "agent failed"
        return m, tea.ClearScreen
    }
    m.agentFlow.DefEntryContent = msg.content
    if _, err := changes.ParseDefStructure(msg.content); err != nil {
        if m.agentFlow.Stage == agent.StagePersistedDefEntry {
            m = m.showPersistedChangeError(err)
            return m, tea.ClearScreen
        }
        m.err = "error parsing title"
        m.openDropdown(UpdateDefState, dropdownDef, UpdateDefState, UpdateDefState, "error parsing title:", []dto.Option{
            {ID: "/edit", Label: "/edit"},
            {ID: "/cancel", Label: "/cancel"},
        }, false)
        return m, tea.ClearScreen
    }
    id, err := changeNumericID(m.changeList.Detail)
    if err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, tea.ClearScreen
    }
    m.state = UpdateDefState
    m.status = "saving def"
    return m, tea.Sequence(tea.ClearScreen, changeDefUpdateForRewriteCommand(m.client, id, msg.content))
}

func (m Model) discardAgentDef(target State, status string, clear bool) (tea.Model, tea.Cmd) {
    if clear {
        var err error
        if m.agentFlow.Workspace.RootDir != "" {
            err = m.agentFlow.Workspace.RemoveChange()
        } else {
            err = m.agentFlow.Workspace.RemoveDef()
        }
        if err != nil {
            m.err = err.Error()
            m.status = "agent failed"
            return m, nil
        }
    }
    m.agentFlow = agent.NewModelWithWorkspace(m.agentWorkspace)
    m.state = target
    m.status = status
    return m, nil
}

func (m Model) startAgentRewrite(sessionID string) (tea.Model, tea.Cmd) {
    m.state = RewriteDefState
    m.agentFlow.Stage = agent.StageAIRunning
    m.agentFlow.SessionID = strings.TrimSpace(sessionID)
    m.agentFlow.CommandOutput = ""
    m.agentElapsed = 0
    m.agentMessageCount = 0
    m.agentSessionResumed = m.agentFlow.SessionID != ""
    m = m.setPromptValue("")
    m.refreshAgentViewport(true)
    m.status = string(agent.StageAIRunning)
    updates := make(chan string)
    return m, tea.Batch(
        tea.ClearScreen,
        agentRewriteCommand(m.agentRunner, m.agentFlow.Workspace, sessionID, updates),
        agentCommandOutputCommand(updates),
        m.agentSpinner.Tick,
        agentElapsedTick(),
    )
}

func (m Model) handleAgentRewriteFinished(msg agentRewriteFinishedMsg) (tea.Model, tea.Cmd) {
    m.agentFlow.CommandOutput = agentRewriteDisplayOutput(msg.result)
    m.refreshAgentViewport(true)
    if msg.err != nil {
        m.agentFlow.Stage = agent.StageAIRunning
        m.err = msg.err.Error()
        m.status = "agent failed"
        return m, nil
    }
    if strings.TrimSpace(msg.result.SessionID) == "" || msg.result.Output != "Done." {
        m.agentFlow.Stage = agent.StageAIRunning
        m.err = agentRewriteContractError(msg.result)
        m.status = "agent failed"
        return m, nil
    }
    m.agentFlow.SessionID = msg.result.SessionID
    m.agentFlow.RepoRoot = msg.result.RepoRoot
    if err := m.agentFlow.Workspace.WriteSessionID(msg.result.SessionID); err != nil {
        m.agentFlow.Stage = agent.StageAIRunning
        m.err = err.Error()
        m.status = "agent failed"
        return m, nil
    }
    id, err := changeNumericID(m.changeList.Detail)
    if err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, nil
    }
    m.status = "saving artifact"
    return m, changeArtifactAgentEditSaveCommand(m.client, id, m.detailEditField, m.agentFlow.Workspace)
}

//lint:ignore U1000 preserved for the future Write Spec with Agent flow.
func (m Model) openAgentSpecInit() (tea.Model, tea.Cmd) {
    m.state = RewriteDefState
    m.agentFlow.Stage = agent.StageAIRunning
    m.agentFlow.CommandOutput = ""
    m.agentElapsed = 0
    m.status = string(agent.StageAIRunning)
    return m, tea.Batch(
        tea.ClearScreen,
        agentSpecInitCommand(m.agentRunner, m.agentFlow.RepoRoot, m.agentFlow.SessionID),
        m.agentSpinner.Tick,
        agentElapsedTick(),
    )
}

func (m Model) handleAgentInitFinished(msg agentInitFinishedMsg) (tea.Model, tea.Cmd) {
    if msg.err != nil {
        m.agentFlow.Stage = agent.StageAIRunning
        m.err = msg.err.Error()
        m.status = "agent failed"
        return m, nil
    }
    m.agentFlow.RepoRoot = msg.repoRoot
    projectID, err := currentProjectNumericID(m.currentProject.ID)
    if err != nil {
        m.err = err.Error()
        m.status = "validation failed"
        return m, nil
    }
    m.status = "saving change"
    return m, agentSpecCreateCommand(m.client, projectID, m.agentFlow.Workspace)
}

func (m Model) handleAgentCommandOutput(msg agentCommandOutputMsg) (tea.Model, tea.Cmd) {
    if msg.output != "" {
        m.agentMessageCount++
        if m.agentFlow.SessionID == "" {
            m.agentFlow.SessionID = agent.ExtractSessionID(msg.output)
        }
        followOutput := m.agentViewport.AtBottom()
        output := agent.FormatCommandOutput(msg.output)
        if strings.TrimSpace(output) != "" {
            m.agentFlow.CommandOutput = appendCommandOutput(m.agentFlow.CommandOutput, output)
            m.refreshAgentViewport(followOutput)
        }
    }
    if msg.done {
        return m, nil
    }
    return m, agentCommandOutputCommand(msg.updates)
}

func agentRewriteCommand(runner agent.Runner, workspace agent.Workspace, sessionID string, updates chan<- string) tea.Cmd {
    return func() tea.Msg {
        defer close(updates)
        if runner == nil {
            runner = agent.NewProcessRunner()
        }
        ctx := context.Background()
        repoRoot, err := runner.ResolveRepoRoot(ctx)
        if err != nil {
            return agentRewriteFinishedMsg{err: err}
        }
        result, err := runner.Rewrite(ctx, repoRoot, sessionID, workspace, func(output string) {
            updates <- output
        })
        if err != nil {
            return agentRewriteFinishedMsg{result: result, err: err}
        }
        return agentRewriteFinishedMsg{result: result}
    }
}

//lint:ignore U1000 preserved for the future Write Spec with Agent flow.
func agentSpecInitCommand(runner agent.Runner, repoRoot string, sessionID string) tea.Cmd {
    return func() tea.Msg {
        if runner == nil {
            runner = agent.NewProcessRunner()
        }
        if strings.TrimSpace(repoRoot) == "" {
            var err error
            repoRoot, err = runner.ResolveRepoRoot(context.Background())
            if err != nil {
                return agentInitFinishedMsg{err: err}
            }
        }
        cmd := runner.InitCommand(repoRoot, sessionID)
        if err := cmd.Run(); err != nil {
            return agentInitFinishedMsg{repoRoot: repoRoot, err: err}
        }
        return agentInitFinishedMsg{repoRoot: repoRoot}
    }
}

func agentCommandOutputCommand(updates <-chan string) tea.Cmd {
    return func() tea.Msg {
        output, ok := <-updates
        if !ok {
            return agentCommandOutputMsg{updates: updates, done: true}
        }
        return agentCommandOutputMsg{output: output, updates: updates}
    }
}

func agentSpecCreateCommand(client appClient, projectID int, workspace agent.Workspace) tea.Cmd {
    return func() tea.Msg {
        spec, err := workspace.ReadGenerated()
        if err != nil {
            return agentSpecCreatedMsg{err: err}
        }
        parsed, err := agent.ParseGeneratedChange(spec)
        if err != nil {
            return agentSpecCreatedMsg{err: err}
        }
        created, err := client.CreateChange(dto.ChangeCreateInput{
            ProjectID: projectID,
            Title:     parsed.Title,
            Def:       parsed.Spec,
        })
        if err != nil {
            return agentSpecCreatedMsg{err: err}
        }
        id, err := changeNumericID(created)
        if err != nil {
            return agentSpecCreatedMsg{err: err}
        }
        if _, err := client.UpdateChangeSpec(id, parsed.Spec, true); err != nil {
            return agentSpecCreatedMsg{err: err}
        }
        if err := updateArtifactTypes(client, id, parsed.ChangeTypes, parsed.ChangeTypesPresent); err != nil {
            return agentSpecCreatedMsg{err: err}
        }
        for _, scenario := range parsed.TestCases {
            if _, err := client.CreateTestCase(id, scenario); err != nil {
                return agentSpecCreatedMsg{err: err}
            }
        }
        change, err := client.GetChange(id)
        if err != nil {
            return agentSpecCreatedMsg{change: created, reloadErr: err}
        }
        return agentSpecCreatedMsg{change: change}
    }
}

func agentRewriteDisplayOutput(result agent.RewriteResult) string {
    parts := make([]string, 0, 2)
    if strings.TrimSpace(result.CommandOutput) != "" {
        parts = append(parts, agent.FormatCommandOutput(result.CommandOutput))
    }
    if strings.TrimSpace(result.Output) != "" {
        parts = append(parts, "final output:\n"+agent.FormatCommandOutput(result.Output))
    }
    return strings.Join(parts, "\n")
}

func appendCommandOutput(existing string, next string) string {
    if strings.TrimSpace(existing) == "" {
        return next
    }
    if strings.TrimSpace(next) == "" {
        return existing
    }
    return strings.TrimRight(existing, "\n") + "\n" + strings.TrimLeft(next, "\n")
}

func agentRewriteContractError(result agent.RewriteResult) string {
    return agent.GenericError
}

func (m Model) agentRunningActive() bool {
    return m.agentFlow.Stage == agent.StageAIRunning && m.status != "agent failed"
}

func agentElapsedTick() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return agentElapsedMsg(t)
    })
}

func agentChangeCreateForRewriteCommand(client appClient, projectID int, workspace agent.Workspace) tea.Cmd {
    return func() tea.Msg {
        def, err := workspace.ReadDef()
        if err != nil {
            return changeCreatedForRewriteMsg{err: err}
        }
        parsed, err := changes.ParseDefStructure(def)
        if err != nil {
            return changeCreatedForRewriteMsg{err: err}
        }
        created, err := client.CreateChange(dto.ChangeCreateInput{
            ProjectID: projectID,
            RefUUID:   workspace.RefUUID,
            Title:     parsed.Title,
            Def:       parsed.Def,
        })
        if err != nil {
            return changeCreatedForRewriteMsg{err: err}
        }
        return changeCreatedForRewriteMsg{
            change:             created,
            changeTypes:        parsed.ChangeTypes,
            changeTypesPresent: parsed.ChangeTypesPresent,
        }
    }
}

func changeTypesUpdateForRewriteCommand(client appClient, change dto.Change, changeTypes []string) tea.Cmd {
    return func() tea.Msg {
        id, err := changeNumericID(change)
        if err != nil {
            return changeTypesUpdatedForRewriteMsg{change: change, err: err}
        }
        updated, err := client.UpdateChangeTypes(id, changeTypes)
        if err != nil {
            return changeTypesUpdatedForRewriteMsg{change: change, err: err}
        }
        return changeTypesUpdatedForRewriteMsg{change: updated}
    }
}

func changeDefUpdateForRewriteCommand(client appClient, id int, def string) tea.Cmd {
    return func() tea.Msg {
        change, err := client.UpdateChangeDef(id, def, false)
        if err != nil {
            return changeDefUpdatedForRewriteMsg{err: err}
        }
        changeTypes, present := changes.ParseArtifactTypes(def)
        if err := updateArtifactTypes(client, id, changeTypes, present); err != nil {
            return changeDefUpdatedForRewriteMsg{change: change, err: err}
        }
        if present {
            change, err = client.GetChange(id)
            if err != nil {
                return changeDefUpdatedForRewriteMsg{change: change, err: err}
            }
        }
        return changeDefUpdatedForRewriteMsg{change: change}
    }
}

func changeArtifactUpdateForWriteCommand(client appClient, id int, field detailEditField, content string) tea.Cmd {
    return func() tea.Msg {
        change, err := updateArtifactAndReload(client, id, field, content, false)
        if err != nil {
            return changeArtifactUpdatedForWriteMsg{change: change, err: err}
        }
        return changeArtifactUpdatedForWriteMsg{change: change}
    }
}

func changeArtifactAgentEditSaveCommand(client appClient, id int, field detailEditField, workspace agent.Workspace) tea.Cmd {
    return func() tea.Msg {
        content, err := workspace.ReadDef()
        if err != nil {
            return changeArtifactAgentEditSavedMsg{err: err}
        }
        if field == "" {
            field, err = artifactFieldForOperation(workspace.Operation)
            if err != nil {
                return changeArtifactAgentEditSavedMsg{err: err}
            }
        }
        change, err := updateArtifactAndReload(client, id, field, content, true)
        if err != nil {
            return changeArtifactAgentEditSavedMsg{change: change, err: err}
        }
        if workspace.RootDir == "" {
            if err := workspace.RemoveDef(); err != nil {
                return changeArtifactAgentEditSavedMsg{change: change, err: err}
            }
        }
        return changeArtifactAgentEditSavedMsg{change: change}
    }
}

func artifactFieldForOperation(operation agent.WriteOperation) (detailEditField, error) {
    switch operation {
    case "", agent.DefWriteOperation:
        return detailEditDef, nil
    case agent.SpecWriteOperation:
        return detailEditSpec, nil
    case agent.PRWriteOperation:
        return detailEditPullRequest, nil
    default:
        return "", fmt.Errorf("unsupported artifact write operation: %s", operation)
    }
}
