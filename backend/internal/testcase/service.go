package testcase

import (
	"aipm/internal/change"
	"context"
	"errors"
	"strings"

	"aipm/internal/dto"
)

var (
	// ErrInvalidInput is a package-level value.
	ErrInvalidInput = errors.New("invalid test case input")
	// ErrNotFound is returned when a test case cannot be found.
	ErrNotFound = errors.New("test case not found")
)

type (
	// Service defines Service values.
	Service struct {
		repo     Repository
		renderer change.Renderer
	}

	// Repository defines Repository values.
	Repository interface {
		List(ctx context.Context, changeID int) ([]dto.TestCase, error)
		Create(ctx context.Context, req dto.TestCaseCreateRequest) (dto.TestCaseMutationResponse, error)
		Update(ctx context.Context, req dto.TestCaseUpdateRequest) (dto.TestCaseMutationResponse, error)
		UpdateDone(ctx context.Context, req dto.TestCaseUpdateDoneRequest) (dto.TestCaseMutationResponse, error)
		UpdateChange(ctx context.Context, req dto.TestCaseUpdateChangeRequest) (dto.TestCaseMutationResponse, error)
		Delete(ctx context.Context, req dto.TestCaseIDRequest) (dto.TestCaseMutationResponse, error)
	}
)

// NewService initializes or executes NewService behavior.
func NewService(testCaseRepository Repository, renderer change.Renderer) *Service {
	return &Service{repo: testCaseRepository, renderer: renderer}
}

// ListTestCases executes ListTestCases behavior.
func (s *Service) ListTestCases(ctx context.Context, req dto.TestCaseListRequest) ([]dto.TestCase, error) {
	if req.ChangeID <= 0 {
		return nil, ErrInvalidInput
	}
	return s.repo.List(ctx, req.ChangeID)
}

// CreateTestCase executes CreateTestCase behavior.
func (s *Service) CreateTestCase(ctx context.Context, req dto.TestCaseCreateRequest) (dto.TestCaseMutationResponse, error) {
	req.Scenario = strings.TrimSpace(req.Scenario)
	if req.ChangeID <= 0 || req.Scenario == "" {
		return dto.TestCaseMutationResponse{}, ErrInvalidInput
	}
	mutation, err := s.repo.Create(ctx, req)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	return s.renderMutation(mutation), nil
}

// UpdateTestCase executes UpdateTestCase behavior.
func (s *Service) UpdateTestCase(ctx context.Context, req dto.TestCaseUpdateRequest) (dto.TestCaseMutationResponse, error) {
	req.Scenario = strings.TrimSpace(req.Scenario)
	if req.ID <= 0 || req.Scenario == "" {
		return dto.TestCaseMutationResponse{}, ErrInvalidInput
	}
	mutation, err := s.repo.Update(ctx, req)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	return s.renderMutation(mutation), nil
}

// UpdateTestCaseDone executes UpdateTestCaseDone behavior.
func (s *Service) UpdateTestCaseDone(ctx context.Context, req dto.TestCaseUpdateDoneRequest) (dto.TestCaseMutationResponse, error) {
	if req.ID <= 0 {
		return dto.TestCaseMutationResponse{}, ErrInvalidInput
	}
	mutation, err := s.repo.UpdateDone(ctx, req)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	return s.renderMutation(mutation), nil
}

// UpdateTestCaseChange executes UpdateTestCaseChange behavior.
func (s *Service) UpdateTestCaseChange(ctx context.Context, req dto.TestCaseUpdateChangeRequest) (dto.TestCaseMutationResponse, error) {
	if req.ID <= 0 || req.ChangeID <= 0 {
		return dto.TestCaseMutationResponse{}, ErrInvalidInput
	}
	mutation, err := s.repo.UpdateChange(ctx, req)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	return s.renderMutation(mutation), nil
}

// DeleteTestCase executes DeleteTestCase behavior.
func (s *Service) DeleteTestCase(ctx context.Context, req dto.TestCaseIDRequest) (dto.TestCaseMutationResponse, error) {
	if req.ID <= 0 {
		return dto.TestCaseMutationResponse{}, ErrInvalidInput
	}
	mutation, err := s.repo.Delete(ctx, req)
	if err != nil {
		return dto.TestCaseMutationResponse{}, err
	}
	return s.renderMutation(mutation), nil
}

func (s *Service) renderMutation(mutation dto.TestCaseMutationResponse) dto.TestCaseMutationResponse {
	return s.renderer.RenderMutation(mutation)
}
