package epics

import (
	"cli/internal/dto"
	"context"
)

// NoSelectableError is shown when enter is pressed without a selectable epic.
const NoSelectableError = "no epics selectable"

// Model stores epics list and detail state.
type Model struct {
	Generation   uint64
	Operation    Operation
	EntityID     int
	Busy         bool
	Draft        string
	Status       string
	Outcome      string
	refreshOp    Operation
	refreshID    int
	Err          error
	ProjectID    int
	DetailOffset int
	DetailLoaded bool
	cancel       context.CancelFunc
	Rows         []dto.Epic
	Selected     int
	Detail       dto.Epic
	Loading      bool
}
