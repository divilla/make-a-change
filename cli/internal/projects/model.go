package projects

import (
	"cli/internal/dto"
	"context"
)

// NoSelectableError is shown when enter is pressed without a selectable project.
const NoSelectableError = "no projects selectable"

// Model stores projects list and detail state.
type Model struct {
	Generation uint64
	Operation  Operation
	EntityID   int
	Busy       bool
	Draft      string
	Status     string
	Err        error
	Catalog    dto.ProjectConfig
	ShowConfig bool
	cancel     context.CancelFunc
	Rows       []dto.Project
	Selected   int
	Detail     dto.Project
	Loading    bool
}
