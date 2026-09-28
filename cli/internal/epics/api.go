package epics

import (
	"cli/internal/dto"
	"context"
)

// API is the epic capability injected by the shell.
type API interface {
	ListEpics(context.Context, int) ([]dto.Epic, error)
	GetEpic(context.Context, int) (dto.Epic, error)
	CreateEpic(context.Context, int, string) (int, error)
	UpdateEpic(context.Context, int, string) error
	DeleteEpic(context.Context, int) error
}
