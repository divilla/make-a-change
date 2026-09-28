package projects

import (
	"cli/internal/dto"
	"context"
)

// API is the project capability injected by the shell.
type API interface {
	ListProjectRows(context.Context) ([]dto.Project, error)
	GetProject(context.Context, int) (dto.Project, error)
	CreateProject(context.Context, string) (int, error)
	UpdateProject(context.Context, int, string) error
	DeleteProject(context.Context, int) error
	GetProjectConfig(context.Context, int) (dto.ProjectConfig, error)
}
