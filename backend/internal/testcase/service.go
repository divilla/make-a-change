package testcase

import (
	"context"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"strings"
)

type (
	// Service defines Service values.
	Service struct {
		repo Repository
	}

	// Repository defines Repository values.
	Repository interface {
		List(ctx context.Context, req domain.TestCaseListRequest) ([]domain.TestCase, error)
		Create(ctx context.Context, req domain.TestCaseCreateRequest) (domain.TestCaseIDRequest, error)
		Update(ctx context.Context, req domain.TestCaseUpdateRequest) error
		UpdateDone(ctx context.Context, req domain.TestCaseUpdateDoneRequest) error
		Delete(ctx context.Context, req domain.TestCaseIDRequest) error
	}
)

// NewService initializes or executes NewService behavior.
func NewService(testCaseRepository Repository) *Service {
	return &Service{repo: testCaseRepository}
}

// ListTestCases executes ListTestCases behavior.
func (s *Service) ListTestCases(ctx context.Context, req domain.TestCaseListRequest) ([]domain.TestCase, error) {
	if req.ChangeID <= 0 {
		return nil, apperror.ErrTestCaseInvalidInput
	}
	return s.repo.List(ctx, req)
}

// CreateTestCase executes CreateTestCase behavior.
func (s *Service) CreateTestCase(ctx context.Context, req domain.TestCaseCreateRequest) (domain.TestCaseIDRequest, error) {
	req.Scenario = strings.TrimSpace(req.Scenario)
	if req.ChangeID <= 0 || req.Scenario == "" {
		return domain.TestCaseIDRequest{}, apperror.ErrTestCaseInvalidInput
	}
	return s.repo.Create(ctx, req)
}

// UpdateTestCase executes UpdateTestCase behavior.
func (s *Service) UpdateTestCase(ctx context.Context, req domain.TestCaseUpdateRequest) error {
	req.Scenario = strings.TrimSpace(req.Scenario)
	if req.ID <= 0 || req.Scenario == "" {
		return apperror.ErrTestCaseInvalidInput
	}
	return s.repo.Update(ctx, req)
}

// UpdateTestCaseDone executes UpdateTestCaseDone behavior.
func (s *Service) UpdateTestCaseDone(ctx context.Context, req domain.TestCaseUpdateDoneRequest) error {
	if req.ID <= 0 {
		return apperror.ErrTestCaseInvalidInput
	}
	return s.repo.UpdateDone(ctx, req)
}

// DeleteTestCase executes DeleteTestCase behavior.
func (s *Service) DeleteTestCase(ctx context.Context, req domain.TestCaseIDRequest) error {
	if req.ID <= 0 {
		return apperror.ErrTestCaseInvalidInput
	}
	return s.repo.Delete(ctx, req)
}
