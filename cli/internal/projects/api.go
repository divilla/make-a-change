package projects

import "cli/internal/dto"

// API defines backend operations needed by project screens.
type API interface {
    ListProjects() ([]dto.Option, error)
    ListProjectRows() ([]dto.Project, error)
    GetProject(id int) (dto.Project, error)
    CreateProject(name string) (dto.Project, error)
    UpdateProject(id int, name string) (dto.Project, error)
}
