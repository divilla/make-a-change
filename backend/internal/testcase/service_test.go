package testcase

import (
	"aipm/internal/change"
	"context"
	"testing"

	"aipm/internal/dto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceRejectsInvalidTestCaseInput(t *testing.T) {
	service := &Service{}
	_, err := service.ListTestCases(context.Background(), dto.TestCaseListRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.CreateTestCase(context.Background(), dto.TestCaseCreateRequest{ChangeID: 2, Scenario: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateTestCase(context.Background(), dto.TestCaseUpdateRequest{ID: 3, Scenario: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateTestCaseChange(context.Background(), dto.TestCaseUpdateChangeRequest{ID: 3})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.DeleteTestCase(context.Background(), dto.TestCaseIDRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestServiceNormalizesTestCaseRequests(t *testing.T) {
	repo := &fakeTestCaseRepository{}
	service := NewService(repo, change.NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	_, err := service.ListTestCases(context.Background(), dto.TestCaseListRequest{ChangeID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.changeID)
	_, err = service.CreateTestCase(context.Background(), dto.TestCaseCreateRequest{ChangeID: 2, Scenario: " Add API test "})
	require.NoError(t, err)
	assert.Equal(t, "Add API test", repo.createReq.Scenario)
	_, err = service.UpdateTestCase(context.Background(), dto.TestCaseUpdateRequest{
		ID: 3, Scenario: " Mark test green ",
	})
	require.NoError(t, err)
	assert.Equal(t, "Mark test green", repo.updateReq.Scenario)
	_, err = service.UpdateTestCaseChange(context.Background(), dto.TestCaseUpdateChangeRequest{ID: 3, ChangeID: 4})
	require.NoError(t, err)
	assert.Equal(t, 4, repo.changeID)
	_, err = service.DeleteTestCase(context.Background(), dto.TestCaseIDRequest{ID: 3})
	require.NoError(t, err)
	assert.Equal(t, 3, repo.id)
}

func TestServiceRendersMutationChangeSpecHTML(t *testing.T) {
	repo := &fakeTestCaseRepository{}
	service := NewService(repo, change.NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	mutation, err := service.CreateTestCase(context.Background(), dto.TestCaseCreateRequest{
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
	id        int
	changeID  int
	createReq dto.TestCaseCreateRequest
	updateReq dto.TestCaseUpdateRequest
}

func (r *fakeTestCaseRepository) List(_ context.Context, changeID int) ([]dto.TestCase, error) {
	r.changeID = changeID
	return []dto.TestCase{}, nil
}

func (r *fakeTestCaseRepository) Create(_ context.Context, req dto.TestCaseCreateRequest) (dto.TestCaseMutationResponse, error) {
	r.createReq = req
	testCase := dto.TestCase{ID: 3, ChangeID: req.ChangeID, Scenario: req.Scenario}
	return dto.TestCaseMutationResponse{
		TestCase: &testCase,
		Change:   dto.Change{ID: req.ChangeID, Spec: "**Change**"},
	}, nil
}

func (r *fakeTestCaseRepository) Update(_ context.Context, req dto.TestCaseUpdateRequest) (dto.TestCaseMutationResponse, error) {
	r.updateReq = req
	return dto.TestCaseMutationResponse{}, nil
}

func (r *fakeTestCaseRepository) UpdateDone(_ context.Context, req dto.TestCaseUpdateDoneRequest) (dto.TestCaseMutationResponse, error) {
	r.id = req.ID
	return dto.TestCaseMutationResponse{}, nil
}

func (r *fakeTestCaseRepository) UpdateChange(_ context.Context, req dto.TestCaseUpdateChangeRequest) (dto.TestCaseMutationResponse, error) {
	r.id, r.changeID = req.ID, req.ChangeID
	return dto.TestCaseMutationResponse{}, nil
}

func (r *fakeTestCaseRepository) Delete(_ context.Context, req dto.TestCaseIDRequest) (dto.TestCaseMutationResponse, error) {
	r.id = req.ID
	return dto.TestCaseMutationResponse{}, nil
}
