package changes

import (
	"cli/internal/dto"
	"context"
)

// API is the change feature's single-operation backend capability.
type API interface {
	ListChangeRows(context.Context, int) ([]dto.Change, error)
	ListInactiveChanges(context.Context, int) ([]dto.Change, error)
	GetChange(context.Context, int) (dto.Change, error)
	UpdateChangeTitle(context.Context, int, string) error
	UpdateChangeSlug(context.Context, int, string) error
	UpdateChangePRUrl(context.Context, int, string) error
	UpdateChangeTypes(context.Context, int, []string) error
	UpdateChangePhase(context.Context, int, string) error
	UpdateChangeActive(context.Context, int, bool) error
	UpdateChangeEpic(context.Context, int, *int) error
	UpdateChangeAfterChange(context.Context, int, *int) error
	DeleteChange(context.Context, int) error
	ListTestCases(context.Context, int) ([]dto.TestCase, error)
}

// Documents is injected by the shell; document ownership stays outside changes.
type Documents interface {
	Load(context.Context, int) ([]dto.Document, error)
	Save(context.Context, int, string, string) (int, error)
}
