package testcase

import (
	"context"
	"mch_api/internal/change"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceRejectsInvalidTestCaseInput(t *testing.T) {
	service := &Service{}
	_, err := service.ListTestCases(context.Background(), domain.TestCaseListRequest{})
	require.ErrorIs(t, err, apperror.ErrTestCaseInvalidInput)
	_, err = service.CreateTestCase(context.Background(), domain.TestCaseCreateRequest{ChangeID: 2, Scenario: "   "})
	require.ErrorIs(t, err, apperror.ErrTestCaseInvalidInput)
	_, err = service.UpdateTestCase(context.Background(), domain.TestCaseUpdateRequest{ID: 3, Scenario: "   "})
	require.ErrorIs(t, err, apperror.ErrTestCaseInvalidInput)
	_, err = service.DeleteTestCase(context.Background(), domain.TestCaseIDRequest{})
	require.ErrorIs(t, err, apperror.ErrTestCaseInvalidInput)
}

func TestServiceNormalizesTestCaseRequests(t *testing.T) {
	repo := &fakeTestCaseRepository{}
	service := NewService(repo, change.NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	_, err := service.ListTestCases(context.Background(), domain.TestCaseListRequest{ChangeID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.changeID)
	_, err = service.CreateTestCase(context.Background(), domain.TestCaseCreateRequest{ChangeID: 2, Scenario: " Add API test "})
	require.NoError(t, err)
	assert.Equal(t, "Add API test", repo.createReq.Scenario)
	_, err = service.UpdateTestCase(context.Background(), domain.TestCaseUpdateRequest{
		ID: 3, Scenario: " Mark test green ",
	})
	require.NoError(t, err)
	assert.Equal(t, "Mark test green", repo.updateReq.Scenario)
	_, err = service.DeleteTestCase(context.Background(), domain.TestCaseIDRequest{ID: 3})
	require.NoError(t, err)
	assert.Equal(t, 3, repo.id)
}

func TestServiceRendersMutationChangeSpecHTML(t *testing.T) {
	repo := &fakeTestCaseRepository{}
	service := NewService(repo, change.NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	mutation, err := service.CreateTestCase(context.Background(), domain.TestCaseCreateRequest{
		ChangeID: 2,
		Scenario: "TestCase",
	})
	require.NoError(t, err)
	assert.Equal(t, "clean(parsed(**Change**))", mutation.Change.SpecHTML)
}

type fakeMarkdownParser struct{}

func (fakeMarkdownParser) Parse(source string) string {
	return "parsed(" + source + ")"
}

type fakeMarkdownSanitizer struct{}

func (fakeMarkdownSanitizer) Parse(source string) string {
	return "clean(" + source + ")"
}

type fakeTestCaseRepository struct {
	err       error
	id        int
	changeID  int
	createReq domain.TestCaseCreateRequest
	updateReq domain.TestCaseUpdateRequest
}

func (r *fakeTestCaseRepository) List(_ context.Context, changeID int) ([]domain.TestCase, error) {
	r.changeID = changeID
	return []domain.TestCase{}, r.err
}

func (r *fakeTestCaseRepository) Create(_ context.Context, req domain.TestCaseCreateRequest) (domain.TestCaseMutationResponse, error) {
	r.createReq = req
	testCase := domain.TestCase{ID: 3, ChangeID: req.ChangeID, Scenario: req.Scenario}
	return domain.TestCaseMutationResponse{
		TestCase: &testCase,
		Change:   domain.Change{ID: req.ChangeID, Spec: "**Change**"},
	}, r.err
}

func (r *fakeTestCaseRepository) Update(_ context.Context, req domain.TestCaseUpdateRequest) (domain.TestCaseMutationResponse, error) {
	r.updateReq = req
	return domain.TestCaseMutationResponse{}, r.err
}

func (r *fakeTestCaseRepository) UpdateDone(_ context.Context, req domain.TestCaseUpdateDoneRequest) (domain.TestCaseMutationResponse, error) {
	r.id = req.ID
	return domain.TestCaseMutationResponse{}, r.err
}

func (r *fakeTestCaseRepository) Delete(_ context.Context, req domain.TestCaseIDRequest) (domain.TestCaseMutationResponse, error) {
	r.id = req.ID
	return domain.TestCaseMutationResponse{}, r.err
}
