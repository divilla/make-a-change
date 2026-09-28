package config

import (
	"context"
	"errors"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	err   error
	calls []string
	req   any
}

func (r *fakeRepo) record(op string, q any) error {
	r.calls = append(r.calls, op)
	r.req = q
	return r.err
}

func (r *fakeRepo) List(context.Context) ([]domain.Config, error) {
	return []domain.Config{}, r.record("List", nil)
}

func (r *fakeRepo) Details(_ context.Context, q domain.ConfigSlugRequest) (domain.Config, error) {
	return domain.Config{Slug: q.Slug}, r.record("Details", q)
}

func (r *fakeRepo) Insert(_ context.Context, q domain.ConfigWriteRequest) (domain.ConfigSlugRequest, error) {
	return domain.ConfigSlugRequest{Slug: q.Slug}, r.record("Insert", q)
}

func (r *fakeRepo) Update(_ context.Context, q domain.ConfigWriteRequest) error {
	return r.record("Update", q)
}

func (r *fakeRepo) Delete(_ context.Context, q domain.ConfigSlugRequest) error {
	return r.record("Delete", q)
}

func validConfig() domain.ConfigWriteRequest {
	return domain.ConfigWriteRequest{Slug: "custom", ProjectDocs: []string{"prd"}, EpicDocs: []string{"brief"}, ChangeDocs: []string{"spec"}, ChangePhases: []string{"todo"}, ChangeColors: []string{"blue"}, ChangeTypes: []string{"fix"}}
}

func TestConfigCompleteReplacementAndErrors(t *testing.T) {
	ctx := context.Background()
	failure := errors.New("failure")
	for _, err := range []error{nil, failure} {
		r := &fakeRepo{err: err}
		s := NewService(r)
		q := validConfig()
		slug := domain.ConfigSlugRequest{Slug: q.Slug}
		_, got := s.List(ctx)
		require.ErrorIs(t, got, err)
		d, got := s.Details(ctx, slug)
		require.ErrorIs(t, got, err)
		require.Equal(t, q.Slug, d.Slug)
		id, got := s.Insert(ctx, q)
		require.ErrorIs(t, got, err)
		require.Equal(t, slug, id)
		require.Equal(t, q, r.req)
		require.ErrorIs(t, s.Update(ctx, q), err)
		require.Equal(t, q, r.req)
		require.ErrorIs(t, s.Delete(ctx, slug), err)
	}
	for _, slug := range []string{"", " \t"} {
		s := &Service{}
		_, err := s.Details(ctx, domain.ConfigSlugRequest{Slug: slug})
		require.ErrorIs(t, err, app.ErrConfigInvalidInput)
		require.ErrorIs(t, s.Delete(ctx, domain.ConfigSlugRequest{Slug: slug}), app.ErrConfigInvalidInput)
	}
	for field := 0; field < 7; field++ {
		for _, blank := range []bool{false, true} {
			q := validConfig()
			arrays := []*[]string{&q.ProjectDocs, &q.EpicDocs, &q.ChangeDocs, &q.ChangePhases, &q.ChangeColors, &q.ChangeTypes}
			if field == 6 {
				q.Slug = " "
			} else if blank {
				*arrays[field] = []string{" "}
			} else {
				*arrays[field] = nil
			}
			s := &Service{}
			_, err := s.Insert(ctx, q)
			require.ErrorIs(t, err, app.ErrConfigInvalidInput)
			require.ErrorIs(t, s.Update(ctx, q), app.ErrConfigInvalidInput)
		}
	}
	q := domain.ConfigWriteRequest{Slug: "empty", ProjectDocs: []string{}, EpicDocs: []string{}, ChangeDocs: []string{}, ChangePhases: []string{}, ChangeColors: []string{}, ChangeTypes: []string{}}
	require.NoError(t, NewService(&fakeRepo{}).Update(ctx, q))
}
