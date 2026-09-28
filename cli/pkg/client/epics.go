package client

import (
	"cli/internal/dto"
	"context"
	"errors"
	"time"
)

type epicWire struct {
	ID          *int       `json:"id"`
	ProjectID   *int       `json:"project_id"`
	Name        *string    `json:"name"`
	DoneTC      *int64     `json:"done_tc"`
	TotalTC     *int64     `json:"total_tc"`
	Completed   *int64     `json:"completed"`
	ChangeCount *int       `json:"change_count"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

func (e epicWire) value() (dto.Epic, error) {
	if e.ID == nil || *e.ID <= 0 || e.ProjectID == nil || *e.ProjectID <= 0 || e.Name == nil || e.DoneTC == nil || e.TotalTC == nil || e.Completed == nil || e.ChangeCount == nil || e.CreatedAt == nil || e.UpdatedAt == nil {
		return dto.Epic{}, &ContractError{errors.New("missing or invalid epic fields")}
	}
	return dto.Epic{ID: *e.ID, ProjectID: *e.ProjectID, Name: *e.Name, DoneTC: *e.DoneTC, TotalTC: *e.TotalTC, Completed: *e.Completed, ChangeCount: *e.ChangeCount, CreatedAt: *e.CreatedAt, UpdatedAt: *e.UpdatedAt}, nil
}

type epicProjectID struct {
	ProjectID int `json:"project_id"`
}

// ListEpics performs one scoped epic/list operation.
func (c HTTPClient) ListEpics(ctx context.Context, projectID int) ([]dto.Epic, error) {
	const path = "/api/v1/epic/list"
	if projectID <= 0 {
		return nil, errors.New("select a valid project with /select-project first")
	}
	var wire []epicWire
	if err := c.projectRequest(ctx, path, epicProjectID{projectID}, 200, &wire); err != nil {
		return nil, err
	}
	if wire == nil {
		return nil, contractStatus(path, &ContractError{errors.New("expected epic array")})
	}
	rows := make([]dto.Epic, 0, len(wire))
	for _, e := range wire {
		row, err := e.value()
		if err == nil && row.ProjectID != projectID {
			err = &ContractError{errors.New("epic project ID differs from request")}
		}
		if err != nil {
			return nil, contractStatus(path, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// GetEpic performs one epic/details operation.
func (c HTTPClient) GetEpic(ctx context.Context, id int) (dto.Epic, error) {
	const path = "/api/v1/epic/details"
	if id <= 0 {
		return dto.Epic{}, errors.New("epic ID must be a valid positive number")
	}
	var wire epicWire
	if err := c.projectRequest(ctx, path, projectID{id}, 200, &wire); err != nil {
		return dto.Epic{}, err
	}
	e, err := wire.value()
	if err == nil && e.ID != id {
		err = &ContractError{errors.New("epic ID differs from request")}
	}
	return e, contractStatus(path, err)
}

// CreateEpic returns only the committed ID; refresh is an explicit feature operation.
func (c HTTPClient) CreateEpic(ctx context.Context, projectID int, name string) (int, error) {
	const path = "/api/v1/epic/create"
	if projectID <= 0 {
		return 0, errors.New("select a valid project with /select-project first")
	}
	var result struct {
		ID int `json:"id"`
	}
	err := c.projectRequest(ctx, path, struct {
		ProjectID int    `json:"project_id"`
		Name      string `json:"name"`
	}{projectID, name}, 201, &result)
	if err == nil && result.ID <= 0 {
		err = &HTTPError{Path: path, Status: 201, Cause: &ContractError{errors.New("missing created ID")}}
	}
	return result.ID, err
}

// UpdateEpic accepts the empty 204 response without a hidden read.
func (c HTTPClient) UpdateEpic(ctx context.Context, id int, name string) error {
	if id <= 0 {
		return errors.New("epic ID must be a valid positive number")
	}
	return c.projectRequest(ctx, "/api/v1/epic/update", struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{id, name}, 204, nil)
}

// DeleteEpic performs exactly one delete.
func (c HTTPClient) DeleteEpic(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("epic ID must be a valid positive number")
	}
	return c.projectRequest(ctx, "/api/v1/epic/delete", projectID{id}, 204, nil)
}
