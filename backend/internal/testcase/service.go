package testcase

import (
	"context"
	"errors"
	"mch_api/internal/change"
	"strings"

	"mch_api/internal/domain"
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
		List(ctx context.Context, changeID int) ([]domain.TestCase, error)
		Create(ctx context.Context, req domain.TestCaseCreateRequest) (domain.TestCaseMutationResponse, error)
		Update(ctx context.Context, req domain.TestCaseUpdateRequest) (domain.TestCaseMutationResponse, error)
		UpdateDone(ctx context.Context, req domain.TestCaseUpdateDoneRequest) (domain.TestCaseMutationResponse, error)
		Delete(ctx context.Context, req domain.TestCaseIDRequest) (domain.TestCaseMutationResponse, error)
	}
)

// NewService initializes or executes NewService behavior.
func NewService(testCaseRepository Repository, renderer change.Renderer) *Service {
	return &Service{repo: testCaseRepository, renderer: renderer}
}

// ListTestCases executes ListTestCases behavior.
func (s *Service) ListTestCases(ctx context.Context, req domain.TestCaseListRequest) ([]domain.TestCase, error) {
	if req.ChangeID <= 0 {
		return nil, ErrInvalidInput
	}
	return s.repo.List(ctx, req.ChangeID)
}

// CreateTestCase executes CreateTestCase behavior.
func (s *Service) CreateTestCase(ctx context.Context, req domain.TestCaseCreateRequest) (domain.TestCaseMutationResponse, error) {
	req.Scenario = strings.TrimSpace(req.Scenario)
	if req.ChangeID <= 0 || req.Scenario == "" {
		return domain.TestCaseMutationResponse{}, ErrInvalidInput
	}
	mutation, err := s.repo.Create(ctx, req)
	if err != nil {
		return domain.TestCaseMutationResponse{}, err
	}
	return s.renderMutation(mutation), nil
}

// UpdateTestCase executes UpdateTestCase behavior.
func (s *Service) UpdateTestCase(ctx context.Context, req domain.TestCaseUpdateRequest) (domain.TestCaseMutationResponse, error) {
	req.Scenario = strings.TrimSpace(req.Scenario)
	if req.ID <= 0 || req.Scenario == "" {
		return domain.TestCaseMutationResponse{}, ErrInvalidInput
	}
	mutation, err := s.repo.Update(ctx, req)
	if err != nil {
		return domain.TestCaseMutationResponse{}, err
	}
	return s.renderMutation(mutation), nil
}

// UpdateTestCaseDone executes UpdateTestCaseDone behavior.
func (s *Service) UpdateTestCaseDone(ctx context.Context, req domain.TestCaseUpdateDoneRequest) (domain.TestCaseMutationResponse, error) {
	if req.ID <= 0 {
		return domain.TestCaseMutationResponse{}, ErrInvalidInput
	}
	mutation, err := s.repo.UpdateDone(ctx, req)
	if err != nil {
		return domain.TestCaseMutationResponse{}, err
	}
	return s.renderMutation(mutation), nil
}

// DeleteTestCase executes DeleteTestCase behavior.
func (s *Service) DeleteTestCase(ctx context.Context, req domain.TestCaseIDRequest) (domain.TestCaseMutationResponse, error) {
	if req.ID <= 0 {
		return domain.TestCaseMutationResponse{}, ErrInvalidInput
	}
	mutation, err := s.repo.Delete(ctx, req)
	if err != nil {
		return domain.TestCaseMutationResponse{}, err
	}
	return s.renderMutation(mutation), nil
}

func (s *Service) renderMutation(mutation domain.TestCaseMutationResponse) domain.TestCaseMutationResponse {
	return s.renderer.RenderMutation(mutation)
}
