package navigation

// ReturnTargets maps returnable states to their target state.
func ReturnTargets() map[State]State {
	return map[State]State{
		ChangesListState:     MainState,
		ChangeDetailsState:   ChangesListState,
		TestCaseDetailsState: ChangeDetailsState,
		EpicsListState:       MainState,
		EpicDetailsState:     EpicsListState,
		ProjectsListState:    MainState,
		ProjectDetailsState:  ProjectsListState,
		MainHelpState:        MainState,
		ChangesHelpState:     ChangesListState,
		EpicsHelpState:       EpicsListState,
		ProjectsHelpState:    ProjectsListState,
		ConfigState:          MainState,
	}
}

// CreateTarget returns the create state for a source state.
func CreateTarget(state State) State {
	switch state {
	case MainState, ChangesListState:
		return ChangeCreateState
	case ChangeDetailsState, TestCaseDetailsState:
		return TestCaseCreateState
	case EpicsListState:
		return EpicCreateState
	case ProjectsListState:
		return ProjectCreateState
	default:
		return state
	}
}

// UpdateTarget returns the edit state for a source state.
func UpdateTarget(state State) State {
	switch state {
	case ChangeDetailsState:
		return ChangeUpdateState
	case TestCaseDetailsState:
		return TestCaseUpdateState
	case EpicDetailsState:
		return EpicUpdateState
	case ProjectDetailsState:
		return ProjectUpdateState
	default:
		return state
	}
}

// SaveTarget returns the state reached after a navigation-only save.
func SaveTarget(state State) State {
	switch state {
	case CreateDefState, UpdateDefState, RewriteDefState, ChangeCreateState, ChangeUpdateState:
		return ChangeDetailsState
	case TestCaseCreateState, TestCaseUpdateState:
		return ChangeDetailsState
	case EpicCreateState, EpicUpdateState:
		return EpicDetailsState
	case ProjectCreateState, ProjectUpdateState:
		return ProjectDetailsState
	default:
		return state
	}
}

// CancelTarget returns the state reached after canceling an edit/create flow.
func CancelTarget(state State) State {
	switch state {
	case CreateDefState, ChangeCreateState:
		return ChangesListState
	case UpdateDefState, ChangeUpdateState:
		return ChangeDetailsState
	case TestCaseCreateState, TestCaseUpdateState:
		return ChangeDetailsState
	case EpicCreateState:
		return EpicsListState
	case EpicUpdateState:
		return EpicDetailsState
	case ProjectCreateState:
		return ProjectsListState
	case ProjectUpdateState:
		return ProjectDetailsState
	default:
		return state
	}
}

// DeleteConfirmationState returns the confirmation state for a delete source.
func DeleteConfirmationState(state State) State {
	switch state {
	case ChangeDetailsState:
		return ChangeDeleteConfirmation
	case TestCaseDetailsState:
		return TestCaseDeleteConfirmation
	case EpicDetailsState:
		return EpicDeleteConfirmation
	case ProjectDetailsState:
		return ProjectDeleteConfirmation
	default:
		return state
	}
}

// DeleteReturnState returns the target state after confirming a delete.
func DeleteReturnState(state State) State {
	switch state {
	case ChangeDetailsState:
		return ChangesListState
	case TestCaseDetailsState:
		return ChangeDetailsState
	case EpicDetailsState:
		return EpicsListState
	case ProjectDetailsState:
		return ProjectsListState
	default:
		return state
	}
}
