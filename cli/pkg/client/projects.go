package client

import (
	"bytes"
	"cli/internal/dto"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ContractError identifies an invalid backend response, retaining its cause.
type ContractError struct{ Cause error }

func (e *ContractError) Error() string { return "backend contract: " + e.Cause.Error() }
func (e *ContractError) Unwrap() error { return e.Cause }

// HTTPError retains the operation, response status (zero before a response), and cause.
type HTTPError struct {
	Path   string
	Status int
	Cause  error
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%s: %d %s: %v", e.Path, e.Status, http.StatusText(e.Status), e.Cause)
}
func (e *HTTPError) Unwrap() error { return e.Cause }

func (c HTTPClient) projectRequest(ctx context.Context, path string, input any, status int, output any) (err error) {
	wrap := func(cause error, code int) error { return &HTTPError{Path: path, Status: code, Cause: cause} }
	body, err := json.Marshal(input)
	if err != nil {
		return wrap(err, 0)
	}
	// Enforce a finite deadline even for a supplied client with no Timeout.
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return wrap(err, 0)
	}
	req.Header.Set("Content-Type", "application/json")
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	// A redirect is a different operation; never silently follow alternate routes.
	bounded := *client
	bounded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := bounded.Do(req)
	if err != nil {
		return wrap(err, 0)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = errors.Join(err, wrap(closeErr, resp.StatusCode))
		}
	}()
	if resp.StatusCode != status {
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return wrap(errors.Join(fmt.Errorf("unexpected status: %s", data), readErr), resp.StatusCode)
	}
	if status == http.StatusNoContent {
		if !strings.HasPrefix(path, "/api/v1/test-case/") && !strings.HasPrefix(path, "/api/v1/config/") {
			return nil
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 1))
		if len(data) != 0 || readErr != nil {
			return wrap(&ContractError{errors.Join(errors.New("unexpected response body"), readErr)}, status)
		}
		return nil
	}
	decoder := json.NewDecoder(resp.Body)
	if strings.HasPrefix(path, "/api/v1/doc/") || strings.HasPrefix(path, "/api/v1/config/") {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(output); err != nil {
		return wrap(&ContractError{err}, status)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return wrap(&ContractError{err}, status)
	}
	return nil
}

// Pointers distinguish absent/null required fields from valid zero values.
type projectWire struct {
	ID          *int       `json:"id"`
	Name        *string    `json:"name"`
	ConfigSlug  *string    `json:"config_slug"`
	Active      *bool      `json:"active"`
	LastRef     *int32     `json:"last_ref"`
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
	ChangeCount *int       `json:"change_count"`
}

func (p projectWire) value() (dto.Project, error) {
	if p.ID == nil || *p.ID <= 0 || p.Active == nil || p.Name == nil || p.ConfigSlug == nil || p.LastRef == nil || p.CreatedAt == nil || p.UpdatedAt == nil || p.ChangeCount == nil {
		return dto.Project{}, &ContractError{errors.New("missing or invalid project fields")}
	}
	return dto.Project{ID: *p.ID, Name: *p.Name, Active: *p.Active, ConfigSlug: *p.ConfigSlug, LastRef: *p.LastRef, CreatedAt: *p.CreatedAt, UpdatedAt: *p.UpdatedAt, ChangeCount: *p.ChangeCount}, nil
}

func contractStatus(path string, err error) error {
	if err == nil {
		return nil
	}
	return &HTTPError{Path: path, Status: http.StatusOK, Cause: err}
}

type projectID struct {
	ID int `json:"id"`
}

// ListProjectRows performs exactly one project/list operation.
func (c HTTPClient) ListProjectRows(ctx context.Context) ([]dto.Project, error) {
	const path = "/api/v1/project/list"
	var wire []projectWire
	if err := c.projectRequest(ctx, path, struct{}{}, http.StatusOK, &wire); err != nil {
		return nil, err
	}
	if wire == nil {
		return nil, contractStatus(path, &ContractError{errors.New("expected project array")})
	}
	rows := make([]dto.Project, 0, len(wire))
	for _, p := range wire {
		row, err := p.value()
		if err != nil {
			return nil, contractStatus(path, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// GetProject performs exactly one project/details operation.
func (c HTTPClient) GetProject(ctx context.Context, id int) (dto.Project, error) {
	const path = "/api/v1/project/details"
	if id <= 0 {
		return dto.Project{}, errors.New("project ID must be a valid positive number")
	}
	var wire projectWire
	if err := c.projectRequest(ctx, path, projectID{id}, http.StatusOK, &wire); err != nil {
		return dto.Project{}, err
	}
	p, err := wire.value()
	if err == nil && p.ID != id {
		err = &ContractError{errors.New("project ID differs from request")}
	}
	return p, contractStatus(path, err)
}

// CreateProject returns only the committed ID; callers explicitly refresh reads.
func (c HTTPClient) CreateProject(ctx context.Context, name string) (int, error) {
	const path = "/api/v1/project/create"
	var result projectID
	err := c.projectRequest(ctx, path, struct {
		Name string `json:"name"`
	}{name}, http.StatusCreated, &result)
	if err == nil && result.ID <= 0 {
		err = &HTTPError{Path: path, Status: 201, Cause: &ContractError{errors.New("missing created ID")}}
	}
	return result.ID, err
}

// UpdateProject performs one update and accepts its empty 204 response.
func (c HTTPClient) UpdateProject(ctx context.Context, id int, name string) error {
	if id <= 0 {
		return errors.New("project ID must be a valid positive number")
	}
	return c.projectRequest(ctx, "/api/v1/project/update", struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}{id, name}, 204, nil)
}

// DeleteProject performs one delete and accepts its empty 204 response.
func (c HTTPClient) DeleteProject(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("project ID must be a valid positive number")
	}
	return c.projectRequest(ctx, "/api/v1/project/delete", projectID{id}, 204, nil)
}

// GetProjectConfig reads all catalogs without defaults or global fallback.
func (c HTTPClient) GetProjectConfig(ctx context.Context, id int) (dto.ProjectConfig, error) {
	const path = "/api/v1/project/config"
	if id <= 0 {
		return dto.ProjectConfig{}, errors.New("project ID must be a valid positive number")
	}
	var cfg configWire
	err := c.projectRequest(ctx, path, projectID{id}, 200, &cfg)
	if err == nil && (cfg.Slug == "" || cfg.ProjectDocs == nil || cfg.EpicDocs == nil || cfg.ChangeDocs == nil || cfg.ChangePhases == nil || cfg.ChangeColors == nil || cfg.ChangeTypes == nil) {
		err = contractStatus(path, &ContractError{errors.New("missing project configuration fields")})
	}
	return dto.ProjectConfig{Slug: cfg.Slug, ProjectDocs: cfg.ProjectDocs, EpicDocs: cfg.EpicDocs, ChangeDocs: cfg.ChangeDocs, ChangePhases: cfg.ChangePhases, ChangeColors: cfg.ChangeColors, ChangeTypes: cfg.ChangeTypes}, err
}

// stringArray rejects null members, which encoding/json otherwise turns into empty strings.
type stringArray []string

func (a *stringArray) UnmarshalJSON(data []byte) error {
	var values []*string
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	if values == nil {
		return errors.New("expected string array")
	}
	result := make([]string, len(values))
	for i, value := range values {
		if value == nil {
			return errors.New("expected string array member")
		}
		result[i] = *value
	}
	*a = result
	return nil
}

type configWire struct {
	Slug         string      `json:"slug"`
	ProjectDocs  stringArray `json:"project_docs"`
	EpicDocs     stringArray `json:"epic_docs"`
	ChangeDocs   stringArray `json:"change_docs"`
	ChangePhases stringArray `json:"change_phases"`
	ChangeColors stringArray `json:"change_colors"`
	ChangeTypes  stringArray `json:"change_types"`
}
