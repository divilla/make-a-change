package app

import (
	"cli/internal/changes"
	"cli/internal/documents"
	"cli/internal/dto"
	"cli/internal/epics"
	"cli/internal/help"
	"cli/internal/projects"
	"cli/internal/testcases"
)

var commandsByState = map[State][]string{
	BriefState:                {"/title", "/uuid", "/brief", "/confirm", "/approve", "/resolve", "/retry", "/reload", "/return"},
	DocumentState:             documents.Commands(),
	MainState:                 {"/changes", "/epics", "/projects", "/select-project", "/config", "/backend-configs", "/health", "/help", "/quit", "/brief-new"},
	ChangesListState:          changes.ListCommands(),
	ChangeDetailsState:        append(changes.DetailCommands(), "/brief-clarify"),
	TestCaseDetailsState:      testcases.DetailCommands(),
	ChangeCreateState:         {"/title", "/uuid", "/save", "/cancel"},
	ChangeUpdateState:         {"/save", "/cancel"},
	TestCaseCreateState:       testcases.EditCommands(),
	TestCaseUpdateState:       testcases.EditCommands(),
	EpicsListState:            epics.ListCommands(),
	EpicDetailsState:          epics.DetailCommands(),
	EpicCreateState:           {"/editor", "/save", "/cancel"},
	EpicUpdateState:           {"/editor", "/save", "/cancel"},
	ProjectsListState:         projects.ListCommands(),
	ProjectDetailsState:       projects.DetailCommands(),
	ProjectCreateState:        {"/editor", "/save", "/cancel"},
	ProjectUpdateState:        {"/editor", "/save", "/cancel"},
	ConfigState:               {"/return"},
	BackendConfigListState:    {"/new-config", "/retry", "/return"},
	BackendConfigDetailsState: {"/edit", "/delete", "/retry", "/return"},
	BackendConfigFormState:    {"/save", "/cancel"},
	BackendConfigDeleteState:  {"/confirm", "/cancel"},
	HealthState:               {"/health-v1", "/health-legacy", "/retry", "/return"},
	MainHelpState:             help.Commands(),
	ChangesHelpState:          help.Commands(),
	EpicsHelpState:            help.Commands(),
	ProjectsHelpState:         help.Commands(),
}

var commandDescriptions = map[string]string{
	"/after-change":    "Set the prerequisite change",
	"/approve":         "Approve the brief",
	"/backend-configs": "Manage backend configurations",
	"/brief":           "Edit the brief",
	"/brief-clarify":   "Clarify this change's brief",
	"/brief-new":       "Start a new brief clarification",
	"/cancel":          "Discard and return",
	"/changes":         "Browse changes",
	"/clear-filters":   "Clear all change filters",
	"/config":          "Show the current configuration",
	"/confirm":         "Confirm the current step",
	"/delete":          "Delete after confirmation",
	"/document":        "Create a document of a selected type",
	"/documents":       "Browse document history",
	"/edit":            "Edit this item",
	"/edit-spec":       "Edit the change specification",
	"/editor":          "Open the external editor",
	"/epic":            "Choose an epic",
	"/epic-filter":     "Filter changes by epic",
	"/epics":           "Browse epics",
	"/find":            "Find text on this screen",
	"/find-filter":     "Filter changes by text",
	"/health":          "Check backend health",
	"/health-legacy":   "Check the legacy health route",
	"/health-v1":       "Check the v1 health route",
	"/help":            "Show help for this screen",
	"/new-comment":     "Create an independent comment",
	"/new-change":      "Create a change",
	"/new-config":      "Create a backend configuration",
	"/new-document":    "Append a document",
	"/new-epic":        "Create an epic",
	"/new-project":     "Create a project",
	"/new-testcase":    "Create a test case",
	"/active":          "Toggle the change's active state",
	"/phase":           "Choose a phase",
	"/phase-filter":    "Filter changes by phase",
	"/pr-url":          "Set the pull request URL",
	"/project-config":  "Show this project's configuration",
	"/projects":        "Browse projects",
	"/quit":            "Quit the application",
	"/reload":          "Reload the current brief",
	"/resolve":         "Resolve brief questions",
	"/retry":           "Reload this screen",
	"/return":          "Return to the previous screen",
	"/save":            "Save the current draft",
	"/select-project":  "Choose the current project",
	"/title":           "Edit the title",
	"/type":            "Choose a document type",
	"/types-filter":    "Filter changes by change types",
	"/types":           "Choose change types",
	"/uuid":            "Edit the optional UUID",
}

func commandOptions(state State) []dto.Option {
	commands := commandsByState[state]
	options := make([]dto.Option, 0, len(commands))
	for _, command := range commands {
		options = append(options, dto.Option{ID: command, Label: command})
	}
	return options
}

func commandAllowed(state State, command string) bool {
	for _, allowed := range commandsByState[state] {
		if allowed == command {
			return true
		}
	}
	return false
}

func helpStateFor(state State) State {
	switch state {
	case MainState:
		return MainHelpState
	case ChangesListState, ChangeDetailsState, TestCaseDetailsState:
		return ChangesHelpState
	case EpicsListState, EpicDetailsState:
		return EpicsHelpState
	case ProjectsListState, ProjectDetailsState:
		return ProjectsHelpState
	default:
		return state
	}
}
