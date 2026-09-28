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
	DocumentState:             documents.Commands(),
	MainState:                 {"/changes", "/epics", "/projects", "/select-project", "/config", "/backend-configs", "/health", "/help", "/quit"},
	ChangesListState:          changes.ListCommands(),
	ChangeDetailsState:        changes.DetailCommands(),
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
