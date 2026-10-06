package app

import (
	"cli/internal/agent"
	"cli/internal/changes"
	"cli/internal/configurations"
	"cli/internal/documents"
	"cli/internal/dto"
	"cli/internal/epics"
	"cli/internal/health"
	"cli/internal/projects"
	"cli/internal/styles"
	"cli/internal/testcases"
	"cli/pkg/briefprocess"
	"cli/pkg/documentprocess"
	"context"
	"os"
	"strconv"

	httpclient "cli/pkg/client"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultBackendURL       = "http://localhost:8080"
	noProjectsToSelectError = "No projects to select from. Please create new project and select it on Main Screen."
	defaultInputPlaceholder = "Type / for commands"
)

type dropdownKind string

const (
	dropdownCommand dropdownKind = "command"
	dropdownList    dropdownKind = "list"
	dropdownSelect  dropdownKind = "select"
	dropdownConfirm dropdownKind = "confirm"
)

type selectorSource string

const (
	selectorProjects  selectorSource = "projects"
	selectorPhases    selectorSource = "phases"
	selectorEpics     selectorSource = "epics"
	selectorTypes     selectorSource = "types"
	selectorDocuments selectorSource = "documents"
)

type filterField string

const (
	filterPhase filterField = "phase"
	filterEpic  filterField = "epic"
	filterTypes filterField = "types"
)

type detailEditField string

const (
	detailEditComment     detailEditField = "comment"
	detailEditTitle       detailEditField = "title"
	detailEditSlug        detailEditField = "slug"
	detailEditDocument    detailEditField = "document"
	detailEditAfterChange detailEditField = "after-change"
	detailEditPhase       detailEditField = "phase"
	detailEditEpic        detailEditField = "epic"
	detailEditTypes       detailEditField = "types"
	detailEditBrief       detailEditField = "brief"
	detailEditSpec        detailEditField = "spec"
	detailEditPullRequest detailEditField = "pull request"
	detailEditPRUrl       detailEditField = "pr url"
	detailEditTestCase    detailEditField = "test case"
)

type changesFilters struct {
	phase    dto.Option
	epic     dto.Option
	typ      dto.Option
	find     string
	inactive bool
}

type optionCatalog struct {
	config dto.ProjectConfig
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
	generation uint64
	projectID  string
	source     selectorSource
	options    []dto.Option
	err        error
}

type changeSavedMsg struct {
	source    State
	change    dto.ChangeView
	err       error
	reloadErr error
}

type optionCatalogLoadedMsg struct {
	id         int
	generation uint64
	config     dto.ProjectConfig
	phases     []dto.Option
	types      []dto.Option
	err        error
}

type currentProjectLoadedMsg struct {
	generation uint64
	id         int
	project    dto.Project
	err        error
}

type editorScratch struct {
	path      string
	identity  os.FileInfo
	projectID int
	document  dto.Document
}

type editorFinishedMsg struct {
	scratch            *editorScratch
	generation         uint64
	projectID, ownerID string
	field              detailEditField
	documentRevision   uint64
	documentOwner      int
	documentTable      string
	documentType       string
	commentID          int
	source             State
	original           string
	content            string
	err                error
}

type startupProjectSelectionMsg struct{}

type appClient interface {
	agent.API
	projects.API
	changes.API
	epics.API
	documents.MutationAPI
	testcases.API
}

// Model is the root Bubble Tea model for the mch application shell.
type Model struct {
	ctx                 context.Context
	selectionGeneration uint64
	catalogGeneration   uint64
	configCatalogCancel context.CancelFunc
	selectorGeneration  uint64
	input               textarea.Model
	editorDraft         *string
	editorScratch       *editorScratch
	editorGeneration    uint64
	state               State
	previousState       State
	width               int
	height              int
	quitting            bool
	quitRequested       bool
	err                 string
	status              string
	helpQuery           string
	promptCursorRow     int
	promptCursorCol     int
	pendingAltO         bool
	changesFilters      changesFilters
	changeSelectionID   string
	optionCatalog       optionCatalog
	selectedConfigSlug  string
	configurations      configurations.Model
	health              health.Model
	changeList          changes.Model
	changeDetailLoaded  bool
	currentProject      dto.Option
	projectList         projects.Model
	epicList            epics.Model
	client              appClient
	appConfig           appConfig
	configSaveInFlight  bool
	configSavePending   bool
	configPath          string
	dropdown            dropdownModel
	detailEditField     detailEditField
	slugPrefix          string
	testCase            testcases.Model
	history             documents.History
	historyOpen         bool
	historyReturning    bool
	historyDetailReload bool
	historyPrinter      documents.Printer
	changeDocuments     documents.ChangeModel
	commentID           int
	deleteDocumentID    int
	document            documents.Model
	documentReturn      State
	documentForm        bool
	briefRunner         agent.Runner
	agentOperation      *agentOperation
	specSyncCancel      context.CancelFunc
	agentOutput         string
	agentOffset         int
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
	input.FocusedStyle.Prompt = styles.Default.InputBand.Foreground(styles.AccentPurple)
	input.FocusedStyle.Text = styles.Default.InputBand.Foreground(styles.Foreground)
	input.FocusedStyle.CursorLine = styles.Default.InputBand.Foreground(styles.Foreground)
	input.FocusedStyle.Placeholder = styles.Default.InputBand.Foreground(styles.Gray)
	input.FocusedStyle.EndOfBuffer = styles.Default.InputBand.Foreground(lipgloss.Color("240"))
	input.BlurredStyle = input.FocusedStyle
	input.Cursor.Style = styles.Default.InputBand.Foreground(styles.Foreground)
	input.Cursor.TextStyle = input.FocusedStyle.Text
	input.Cursor.SetMode(cursor.CursorStatic)
	input.Focus()
	currentProject := dto.Option{}
	if cfg.ProjectID > 0 {
		currentProject = dto.Option{
			ID: strconv.Itoa(cfg.ProjectID),
		}
	}
	return Model{
		ctx:            context.Background(),
		input:          input,
		state:          MainState,
		width:          80,
		height:         24,
		currentProject: currentProject,
		client:         client,
		appConfig:      cfg,
		configPath:     cfg.ConfigPath,
		status:         "MainState",
		briefRunner:    briefprocess.Runner{},
		historyPrinter: documentprocess.Bat{},
	}
}

var _ tea.Model = Model{}
