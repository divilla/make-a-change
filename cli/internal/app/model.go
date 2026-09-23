package app

import (
    "path/filepath"
    "strconv"
    "strings"
    "time"

    "cli/internal/agent"
    "cli/internal/changes"
    "cli/internal/dto"
    "cli/internal/epics"
    "cli/internal/projects"
    "cli/internal/styles"
    httpclient "cli/pkg/client"

    "github.com/charmbracelet/bubbles/cursor"
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/textarea"
    "github.com/charmbracelet/bubbles/viewport"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/gofrs/uuid/v5"
)

const defaultBackendURL = "http://localhost:8080"
const noProjectsToSelectError = "No projects to select from. Please create new project and select it on Main Screen."
const defaultInputPlaceholder = "Type / for commands"

type dropdownKind string

const (
    dropdownCommand      dropdownKind = "command"
    dropdownList         dropdownKind = "list"
    dropdownSelect       dropdownKind = "select"
    dropdownConfirm      dropdownKind = "confirm"
    dropdownAgent        dropdownKind = "agent"
    dropdownDef          dropdownKind = "def"
    dropdownPersistedDef dropdownKind = "persisted def"
)

type selectorSource string

const (
    selectorProjects selectorSource = "projects"
    selectorPhases   selectorSource = "phases"
    selectorEpics    selectorSource = "epics"
    selectorTypes    selectorSource = "types"
)

type filterField string

const (
    filterPhase filterField = "phase"
    filterEpic  filterField = "epic"
    filterType  filterField = "type"
)

type detailEditField string

const (
    detailEditTitle       detailEditField = "title"
    detailEditPhase       detailEditField = "phase"
    detailEditEpic        detailEditField = "epic"
    detailEditTypes       detailEditField = "types"
    detailEditDef         detailEditField = "def"
    detailEditSpec        detailEditField = "spec"
    detailEditPullRequest detailEditField = "pull request"
    detailEditPRUrl       detailEditField = "pr url"
    detailEditTestCase    detailEditField = "test case"
)

type changesFilters struct {
    phase dto.Option
    epic  dto.Option
    typ   dto.Option
    find  string
}

type optionCatalog struct {
    phases []dto.Option
    types  []dto.Option
    loaded bool
    err    error
}

type dropdownModel struct {
    kind         dropdownKind
    state        State
    previous     State
    onSelect     State
    source       selectorSource
    filterField  filterField
    editField    detailEditField
    label        string
    options      []dto.Option
    filter       string
    highlighted  int
    loading      bool
    pendingTypes []string
    typesChanged bool
}

type selectorLoadedMsg struct {
    source  selectorSource
    options []dto.Option
    err     error
}

type projectListLoadedMsg struct {
    projects []dto.Project
    err      error
}

type projectSavedMsg struct {
    source  State
    project dto.Project
    err     error
}

type projectLoadedMsg struct {
    id      int
    project dto.Project
    err     error
}

type changeListLoadedMsg struct {
    changes []dto.Change
    err     error
}

type changeLoadedMsg struct {
    id     int
    change dto.Change
    err    error
}

type changeSavedMsg struct {
    source    State
    change    dto.Change
    err       error
    reloadErr error
}

type changeCreatedForRewriteMsg struct {
    change             dto.Change
    changeTypes        []string
    changeTypesPresent bool
    err                error
}

type changeTypesUpdatedForRewriteMsg struct {
    change dto.Change
    err    error
}

type changeDefUpdatedForRewriteMsg struct {
    change dto.Change
    err    error
}

type changeArtifactEditLoadedMsg struct {
    id     int
    field  detailEditField
    change dto.Change
    err    error
}

type changeArtifactUpdatedForWriteMsg struct {
    change dto.Change
    err    error
}

type changeArtifactAgentEditSavedMsg struct {
    change    dto.Change
    err       error
    reloadErr error
}

type agentSpecCreatedMsg struct {
    change    dto.Change
    err       error
    reloadErr error
}

type changeDeletedMsg struct {
    target State
    err    error
}

type agentRewriteFinishedMsg struct {
    result agent.RewriteResult
    err    error
}

type agentInitFinishedMsg struct {
    repoRoot string
    err      error
}

type agentCommandOutputMsg struct {
    output  string
    updates <-chan string
    done    bool
}

type optionCatalogLoadedMsg struct {
    phases []dto.Option
    types  []dto.Option
    err    error
}

type agentElapsedMsg time.Time

type currentProjectLoadedMsg struct {
    id      int
    project dto.Project
    err     error
}

type editorFinishedMsg struct {
    source  State
    content string
    err     error
}

type startupProjectSelectionMsg struct{}

type appClient interface {
    projects.API
    changes.API
    epics.API
    ListTypes() ([]dto.Option, error)
}

// Model is the root Bubble Tea model for the mch application shell.
type Model struct {
    input               textarea.Model
    state               State
    previousState       State
    width               int
    height              int
    quitting            bool
    err                 string
    status              string
    helpQuery           string
    promptCursorRow     int
    promptCursorCol     int
    pendingAltO         bool
    changesFilters      changesFilters
    optionCatalog       optionCatalog
    changeList          changes.Model
    agentFlow           agent.Model
    agentRunner         agent.Runner
    agentWorkspace      string
    agentSpinner        spinner.Model
    agentViewport       viewport.Model
    agentElapsed        int
    agentMessageCount   int
    agentSessionResumed bool
    newChangeUUID       func() (uuid.UUID, error)
    currentProject      dto.Option
    projectList         projects.Model
    client              appClient
    appConfig           appConfig
    configPath          string
    dropdown            dropdownModel
    detailEditField     detailEditField
    activeTestCase      dto.TestCase
}

// NewModel creates the default mch model using local config and HTTP backend access.
func NewModel() Model {
    cfg, err := loadRepositoryConfig()
    m := newModelWithConfig(httpclient.NewHTTPClient(cfg.BackendURL), cfg)
    if err != nil {
        m.err = err.Error()
    }
    return m
}

// NewModelWithClient creates a model with an injected backend client for tests.
func NewModelWithClient(client appClient) Model {
    m := newModelWithConfig(client, appConfig{BackendURL: defaultBackendURL})
    m.agentWorkspace = "configured-test-temp"
    m.agentFlow = agent.NewModelWithWorkspace(m.agentWorkspace)
    return m
}

func newModelWithConfig(client appClient, cfg appConfig) Model {
    input := textarea.New()
    input.Placeholder = defaultInputPlaceholder
    input.Prompt = "> "
    input.ShowLineNumbers = false
    input.EndOfBufferCharacter = ' '
    input.CharLimit = defaultPromptCharLimit
    input.SetWidth(0)
    input.SetHeight(1)
    input.FocusedStyle.Base = styles.Default.InputBand
    input.FocusedStyle.Prompt = styles.Default.InputBand.Foreground(lipgloss.Color("183"))
    input.FocusedStyle.Text = styles.Default.InputBand.Foreground(lipgloss.Color("15"))
    input.FocusedStyle.CursorLine = styles.Default.InputBand.Foreground(lipgloss.Color("15"))
    input.FocusedStyle.Placeholder = styles.Default.InputBand.Foreground(lipgloss.Color("0"))
    input.FocusedStyle.EndOfBuffer = styles.Default.InputBand.Foreground(lipgloss.Color("240"))
    input.BlurredStyle = input.FocusedStyle
    input.Cursor.Style = styles.Default.InputBand.Foreground(lipgloss.Color("15"))
    input.Cursor.TextStyle = input.FocusedStyle.Text
    input.Cursor.SetMode(cursor.CursorStatic)
    input.Focus()
    spin := spinner.New(
        spinner.WithSpinner(spinner.Spinner{
            Frames: []string{"·", "•", "●", "•"},
            FPS:    250 * time.Millisecond,
        }),
    )
    agentViewport := viewport.New(80, agentViewportHeight(24))
    agentViewport.Style = agentViewportStyle()

    currentProject := dto.Option{}
    if cfg.ProjectID > 0 {
        currentProject = dto.Option{
            ID: strconv.Itoa(cfg.ProjectID),
        }
    }
    agentWorkspace := ""
    if strings.TrimSpace(cfg.RepositoryRoot) != "" {
        agentWorkspace = filepath.Join(cfg.RepositoryRoot, agent.TempDir)
    }

    return Model{
        input:          input,
        state:          MainState,
        width:          80,
        height:         24,
        agentFlow:      agent.NewModelWithWorkspace(agentWorkspace),
        agentRunner:    agent.NewProcessRunner(),
        agentWorkspace: agentWorkspace,
        agentSpinner:   spin,
        agentViewport:  agentViewport,
        newChangeUUID:  uuid.NewV7,
        currentProject: currentProject,
        client:         client,
        appConfig:      cfg,
        configPath:     cfg.ConfigPath,
        status:         "MainState",
    }
}

var _ tea.Model = Model{}
