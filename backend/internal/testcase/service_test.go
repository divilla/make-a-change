package testcase

import (
	"context"
	"errors"
	"fmt"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeTestCaseRepository struct {
	err      error
	calls    []any
	contexts []context.Context
	cases    []domain.TestCase
}

func (r *fakeTestCaseRepository) record(ctx context.Context, req any) {
	r.calls = append(r.calls, req)
	r.contexts = append(r.contexts, ctx)
}

func (r *fakeTestCaseRepository) List(ctx context.Context, req domain.TestCaseListRequest) ([]domain.TestCase, error) {
	r.record(ctx, req)
	if r.cases == nil {
		return []domain.TestCase{}, r.err
	}
	return r.cases, r.err
}

func (r *fakeTestCaseRepository) Create(ctx context.Context, req domain.TestCaseCreateRequest) (domain.TestCaseIDRequest, error) {
	r.record(ctx, req)
	return domain.TestCaseIDRequest{ID: 3}, r.err
}

func (r *fakeTestCaseRepository) Update(ctx context.Context, req domain.TestCaseUpdateRequest) error {
	r.record(ctx, req)
	return r.err
}

func (r *fakeTestCaseRepository) UpdateDone(ctx context.Context, req domain.TestCaseUpdateDoneRequest) error {
	r.record(ctx, req)
	return r.err
}

func (r *fakeTestCaseRepository) Delete(ctx context.Context, req domain.TestCaseIDRequest) error {
	r.record(ctx, req)
	return r.err
}

func TestServiceRejectsInvalidTestCaseInput(t *testing.T) {
	for _, id := range []int{0, -1} {
		t.Run(fmt.Sprint(id), func(t *testing.T) {
			r := &fakeTestCaseRepository{}
			s := NewService(r)
			ctx := context.Background()
			_, err := s.List(ctx, domain.TestCaseListRequest{ChangeID: id})
			require.ErrorIs(t, err, app.ErrTestCaseInvalidInput)
			_, err = s.Create(ctx, domain.TestCaseCreateRequest{ChangeID: id, Scenario: "valid"})
			require.ErrorIs(t, err, app.ErrTestCaseInvalidInput)
			require.ErrorIs(t, s.UpdateTestCase(ctx, domain.TestCaseUpdateRequest{ID: id, Scenario: "valid"}), app.ErrTestCaseInvalidInput)
			require.ErrorIs(t, s.UpdateTestCaseDone(ctx, domain.TestCaseUpdateDoneRequest{ID: id}), app.ErrTestCaseInvalidInput)
			require.ErrorIs(t, s.Delete(ctx, domain.TestCaseIDRequest{ID: id}), app.ErrTestCaseInvalidInput)
			require.Empty(t, r.calls)
		})
	}
	for _, blank := range []string{"", " \t\n "} {
		s := NewService(nil)
		_, err := s.Create(context.Background(), domain.TestCaseCreateRequest{ChangeID: 1, Scenario: blank})
		require.ErrorIs(t, err, app.ErrTestCaseInvalidInput)
		require.ErrorIs(t, s.UpdateTestCase(context.Background(), domain.TestCaseUpdateRequest{ID: 1, Scenario: blank}), app.ErrTestCaseInvalidInput)
	}
}

func TestServiceNormalizesAndDelegatesOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, id := range []int{1, 1 << 40} {
		for _, cause := range []error{nil, errors.New("repository failure")} {
			r := &fakeTestCaseRepository{err: cause, cases: []domain.TestCase{{ID: id, ChangeID: id}}}
			s := NewService(r)
			got, err := s.List(ctx, domain.TestCaseListRequest{ChangeID: id})
			require.ErrorIs(t, err, cause)
			require.Equal(t, r.cases, got)
			createdAt, err := s.Create(ctx, domain.TestCaseCreateRequest{ChangeID: id, Scenario: " \tfirst\n "})
			require.ErrorIs(t, err, cause)
			require.Equal(t, 3, createdAt.ID)
			require.ErrorIs(t, s.UpdateTestCase(ctx, domain.TestCaseUpdateRequest{ID: id, Scenario: " \tsecond\n "}), cause)
			for _, done := range []bool{true, false} {
				require.ErrorIs(t, s.UpdateTestCaseDone(ctx, domain.TestCaseUpdateDoneRequest{ID: id, Done: done}), cause)
			}
			require.ErrorIs(t, s.Delete(ctx, domain.TestCaseIDRequest{ID: id}), cause)
			require.Equal(t, []any{domain.TestCaseListRequest{ChangeID: id}, domain.TestCaseCreateRequest{ChangeID: id, Scenario: "first"}, domain.TestCaseUpdateRequest{ID: id, Scenario: "second"}, domain.TestCaseUpdateDoneRequest{ID: id, Done: true}, domain.TestCaseUpdateDoneRequest{ID: id}, domain.TestCaseIDRequest{ID: id}}, r.calls)
			for _, actual := range r.contexts {
				require.Same(t, ctx, actual)
			}
		}
	}
}
