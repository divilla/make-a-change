package client

import (
	"cli/internal/dto"
	"context"
	"encoding/json"
	"errors"
	"time"
)

// nullableWire distinguishes an explicit null from a missing field.
type nullableWire[T any] struct {
	Present bool
	Value   *T
}

func (n *nullableWire[T]) UnmarshalJSON(data []byte) error {
	n.Present = true
	return json.Unmarshal(data, &n.Value)
}

type changeWire struct {
	ID            *int                 `json:"id"`
	ProjectID     *int                 `json:"project_id"`
	RefUUID       *string              `json:"ref_uuid"`
	ChangePhase   *string              `json:"change_phase"`
	Title         *string              `json:"title"`
	Open          *bool                `json:"open"`
	DoneTC        *int64               `json:"done_tc"`
	TotalTC       *int64               `json:"total_tc"`
	Completed     *int64               `json:"completed"`
	UpdatedAt     *time.Time           `json:"updated_at"`
	Ref           nullableWire[int32]  `json:"ref"`
	Slug          nullableWire[string] `json:"slug"`
	EpicID        nullableWire[int]    `json:"epic_id"`
	EpicName      nullableWire[string] `json:"epic_name"`
	AfterChangeID nullableWire[int]    `json:"after_change_id"`
	ChangeTypes   stringArray          `json:"change_types"`
	PRUrl         *string              `json:"pr_url"`
	CreatedAt     *time.Time           `json:"created_at"`
}

func (w changeWire) value(details bool) (dto.Change, error) {
	if w.ID == nil || w.ProjectID == nil || w.RefUUID == nil || w.ChangePhase == nil || w.Title == nil || w.Open == nil || w.DoneTC == nil || w.TotalTC == nil || w.Completed == nil || w.UpdatedAt == nil || !w.Ref.Present || !w.Slug.Present || !w.EpicID.Present || !w.EpicName.Present || w.ChangeTypes == nil || *w.ID <= 0 || *w.ProjectID <= 0 {
		return dto.Change{}, &ContractError{errors.New("missing or invalid change fields")}
	}
	if details && (!w.AfterChangeID.Present || w.PRUrl == nil || w.CreatedAt == nil) {
		return dto.Change{}, &ContractError{errors.New("missing change detail fields")}
	}
	c := dto.Change{ID: *w.ID, ProjectID: *w.ProjectID, RefUUID: *w.RefUUID, ChangePhase: *w.ChangePhase, Title: *w.Title, Open: *w.Open, DoneTC: *w.DoneTC, TotalTC: *w.TotalTC, Completed: *w.Completed, UpdatedAt: *w.UpdatedAt, Ref: w.Ref.Value, Slug: w.Slug.Value, EpicID: w.EpicID.Value, EpicName: w.EpicName.Value, ChangeTypes: w.ChangeTypes, AfterChangeID: w.AfterChangeID.Value}
	if details {
		c.PRUrl = *w.PRUrl
		c.CreatedAt = *w.CreatedAt
	}
	return c, nil
}

// ListChangeRows performs exactly one cancellable list operation.
func (c HTTPClient) ListChangeRows(ctx context.Context, project int) ([]dto.Change, error) {
	const path = "/api/v1/change/list"
	if project <= 0 {
		return nil, errors.New("select a valid project first")
	}
	var wire []changeWire
	if err := c.projectRequest(ctx, path, epicProjectID{project}, 200, &wire); err != nil {
		return nil, err
	}
	if wire == nil {
		return nil, contractStatus(path, &ContractError{errors.New("expected change array")})
	}
	rows := make([]dto.Change, 0, len(wire))
	for _, w := range wire {
		v, err := w.value(false)
		if err == nil && v.ProjectID != project {
			err = &ContractError{errors.New("change project differs from request")}
		}
		if err != nil {
			return nil, contractStatus(path, err)
		}
		rows = append(rows, v)
	}
	return rows, nil
}

// GetChange performs exactly one details operation.
func (c HTTPClient) GetChange(ctx context.Context, id int) (dto.Change, error) {
	const path = "/api/v1/change/details"
	if id <= 0 {
		return dto.Change{}, errors.New("change ID must be a valid positive number")
	}
	var w changeWire
	if err := c.projectRequest(ctx, path, projectID{id}, 200, &w); err != nil {
		return dto.Change{}, err
	}
	v, err := w.value(true)
	if err == nil && v.ID != id {
		err = &ContractError{errors.New("change ID differs from request")}
	}
	return v, contractStatus(path, err)
}

// CreateChange returns only the committed ID, without a refresh.
func (c HTTPClient) CreateChange(ctx context.Context, input dto.ChangeCreateInput) (int, error) {
	if input.ProjectID <= 0 {
		return 0, errors.New("select a valid project first")
	}
	return c.insertID(ctx, "/api/v1/change/create", input)
}

func (c HTTPClient) insertID(ctx context.Context, path string, input any) (int, error) {
	var result projectID
	err := c.projectRequest(ctx, path, input, 201, &result)
	if err == nil && result.ID <= 0 {
		err = &HTTPError{Path: path, Status: 201, Cause: &ContractError{errors.New("missing created ID")}}
	}
	return result.ID, err
}

// UpdateChangeTitle performs one write, preserving explicit empty/false/null values.
func (c HTTPClient) UpdateChangeTitle(ctx context.Context, id int, value string) error {
	if id <= 0 {
		return errors.New("change ID must be a valid positive number")
	}
	return c.projectRequest(ctx, "/api/v1/change/update-title", struct {
		ID    int    `json:"id"`
		Value string `json:"title"`
	}{id, value}, 204, nil)
}

// UpdateChangeTypes performs one write, preserving explicit empty/false/null values.
func (c HTTPClient) UpdateChangeTypes(ctx context.Context, id int, value []string) error {
	if id <= 0 {
		return errors.New("change ID must be a valid positive number")
	}
	if value == nil {
		value = []string{}
	}
	return c.projectRequest(ctx, "/api/v1/change/update-types", struct {
		ID    int      `json:"id"`
		Value []string `json:"change_types"`
	}{id, value}, 204, nil)
}

// UpdateChangePhase performs one write, preserving explicit empty/false/null values.
func (c HTTPClient) UpdateChangePhase(ctx context.Context, id int, value string) error {
	if id <= 0 {
		return errors.New("change ID must be a valid positive number")
	}
	return c.projectRequest(ctx, "/api/v1/change/update-phase", struct {
		ID    int    `json:"id"`
		Value string `json:"change_phase"`
	}{id, value}, 204, nil)
}

// UpdateChangeOpen performs one write, preserving explicit empty/false/null values.
func (c HTTPClient) UpdateChangeOpen(ctx context.Context, id int, value bool) error {
	if id <= 0 {
		return errors.New("change ID must be a valid positive number")
	}
	return c.projectRequest(ctx, "/api/v1/change/update-open", struct {
		ID    int  `json:"id"`
		Value bool `json:"open"`
	}{id, value}, 204, nil)
}

// UpdateChangeEpic performs one write, preserving explicit empty/false/null values.
func (c HTTPClient) UpdateChangeEpic(ctx context.Context, id int, value *int) error {
	if id <= 0 {
		return errors.New("change ID must be a valid positive number")
	}
	if value != nil && *value <= 0 {
		return errors.New("association ID must be positive or null")
	}
	return c.projectRequest(ctx, "/api/v1/change/update-epic", struct {
		ID    int  `json:"id"`
		Value *int `json:"epic_id"`
	}{id, value}, 204, nil)
}

// UpdateChangeAfterChange performs one write, preserving explicit empty/false/null values.
func (c HTTPClient) UpdateChangeAfterChange(ctx context.Context, id int, value *int) error {
	if id <= 0 {
		return errors.New("change ID must be a valid positive number")
	}
	if value != nil && *value <= 0 {
		return errors.New("association ID must be positive or null")
	}
	return c.projectRequest(ctx, "/api/v1/change/update-after-change", struct {
		ID    int  `json:"id"`
		Value *int `json:"after_change_id"`
	}{id, value}, 204, nil)
}

// UpdateChangePRUrl performs one write, preserving explicit empty/false/null values.
func (c HTTPClient) UpdateChangePRUrl(ctx context.Context, id int, value string) error {
	if id <= 0 {
		return errors.New("change ID must be a valid positive number")
	}
	return c.projectRequest(ctx, "/api/v1/change/update-pr-url", struct {
		ID    int    `json:"id"`
		Value string `json:"pr_url"`
	}{id, value}, 204, nil)
}

// DeleteChange performs exactly one delete.
func (c HTTPClient) DeleteChange(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("change ID must be a valid positive number")
	}
	return c.projectRequest(ctx, "/api/v1/change/delete", projectID{id}, 204, nil)
}
