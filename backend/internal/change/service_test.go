package change

import (
	"context"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"strconv"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceResolvesChangeCreateIdentity(t *testing.T) {
	repo := &fakeChangeRepository{}
	service := NewService(repo, NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	generated, err := service.CreateChange(context.Background(), domain.ChangeCreateRequest{ProjectID: 1, Title: "Generated", Brief: "Brief"})
	require.NoError(t, err)
	require.NotNil(t, repo.createReq.RefUUID)
	assert.Equal(t, byte(7), repo.createReq.RefUUID.Version())
	assert.Equal(t, repo.createReq.RefUUID.String(), generated.RefUUID)

	supplied := uuid.Must(uuid.FromString("0198a86f-9b8a-7d89-ae5b-6f25b528b04c"))
	preserved, err := service.CreateChange(context.Background(), domain.ChangeCreateRequest{ProjectID: 1, RefUUID: &supplied, Title: "Supplied", Brief: "Brief"})
	require.NoError(t, err)
	require.NotNil(t, repo.createReq.RefUUID)
	assert.Equal(t, supplied, *repo.createReq.RefUUID)
	assert.Equal(t, supplied.String(), preserved.RefUUID)
}

func TestServiceRejectsInvalidChangeInput(t *testing.T) {
	service := &Service{}
	_, err := service.ListChanges(context.Background(), domain.ChangeListRequest{})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.GetChange(context.Background(), domain.ChangeIDRequest{})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.CreateChange(context.Background(), domain.ChangeCreateRequest{
		ProjectID: 1, Title: "   ",
	})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.UpdateTitle(context.Background(), domain.ChangeUpdateTitleRequest{ID: 2, Title: "   "})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.UpdatePhase(context.Background(), domain.ChangeUpdatePhaseRequest{ID: 2, ChangePhase: "   "})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.UpdateBrief(context.Background(), domain.ChangeUpdateBriefRequest{ID: 2, Brief: "brief"})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.UpdateSpec(context.Background(), domain.ChangeUpdateSpecRequest{ID: 2, Spec: "   "})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.UpdateSpec(context.Background(), domain.ChangeUpdateSpecRequest{ID: 2, Spec: "spec"})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.UpdatePR(context.Background(), domain.ChangeUpdatePRRequest{ID: 2, PR: "   "})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.UpdatePR(context.Background(), domain.ChangeUpdatePRRequest{ID: 2, PR: "pr"})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.UpdatePRUrl(context.Background(), domain.ChangeUpdatePRUrlRequest{ID: 2, PRUrl: "   "})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	_, err = service.UpdateOpen(context.Background(), domain.ChangeUpdateOpenRequest{ID: 2})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	badURL := "javascript:alert(1)"
	_, err = service.UpdatePRUrl(context.Background(), domain.ChangeUpdatePRUrlRequest{ID: 2, PRUrl: badURL})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	missingHostURL := "https:///missing-host"
	_, err = service.UpdatePRUrl(context.Background(), domain.ChangeUpdatePRUrlRequest{ID: 2, PRUrl: missingHostURL})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
	err = service.DeleteChange(context.Background(), domain.ChangeIDRequest{})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
}

func TestServiceNormalizesChangeRequests(t *testing.T) {
	repo := &fakeChangeRepository{availableTypes: []string{"fix"}}
	service := NewService(repo, NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))
	epicID := 4
	agentEdit := false

	_, err := service.ListChanges(context.Background(), domain.ChangeListRequest{ProjectID: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.projectID)
	_, err = service.GetChange(context.Background(), domain.ChangeIDRequest{ID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.id)

	_, err = service.CreateChange(context.Background(), domain.ChangeCreateRequest{ProjectID: 1, Title: " Change Title ", Brief: " Brief "})
	require.NoError(t, err)
	assert.Equal(t, "Change Title", repo.createReq.Title)
	assert.Equal(t, "Brief", repo.createReq.Brief)

	err = service.UpdateChangeTypes(context.Background(), domain.ChangeUpdateChangeTypesRequest{ID: 2, ChangeTypes: []string{" fix ", "missing", "fix "}})
	require.NoError(t, err)
	assert.Equal(t, []string{"fix"}, repo.updateTypesReq.ChangeTypes)
	err = service.UpdateChangeTypes(context.Background(), domain.ChangeUpdateChangeTypesRequest{ID: 2, ChangeTypes: []string{"missing"}})
	require.NoError(t, err)
	assert.Empty(t, repo.updateTypesReq.ChangeTypes)
	_, err = service.UpdateTitle(context.Background(), domain.ChangeUpdateTitleRequest{ID: 2, Title: " Focused Title "})
	require.NoError(t, err)
	assert.Equal(t, "Focused Title", repo.updateTitleReq.Title)
	_, err = service.UpdateBrief(context.Background(), domain.ChangeUpdateBriefRequest{ID: 2, Brief: " Focused Brief ", AgentEdit: &agentEdit})
	require.NoError(t, err)
	assert.Equal(t, "Focused Brief", repo.updateBriefReq.Brief)
	spec := " Focused Spec "
	_, err = service.UpdateSpec(context.Background(), domain.ChangeUpdateSpecRequest{ID: 2, Spec: spec, AgentEdit: &agentEdit})
	require.NoError(t, err)
	assert.Equal(t, "Focused Spec", repo.updateSpecReq.Spec)
	pr := " PR Body "
	_, err = service.UpdatePR(context.Background(), domain.ChangeUpdatePRRequest{ID: 2, PR: pr, AgentEdit: &agentEdit})
	require.NoError(t, err)
	assert.Equal(t, "PR Body", repo.updatePRReq.PR)
	prURL := " https://example.test/pr "
	_, err = service.UpdatePRUrl(context.Background(), domain.ChangeUpdatePRUrlRequest{ID: 2, PRUrl: prURL})
	require.NoError(t, err)
	assert.Equal(t, "https://example.test/pr", repo.updatePRUrlReq.PRUrl)
	_, err = service.UpdateEpic(context.Background(), domain.ChangeUpdateEpicRequest{ID: 2, EpicID: &epicID})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.id)
	_, err = service.UpdatePhase(context.Background(), domain.ChangeUpdatePhaseRequest{ID: 2, ChangePhase: " review "})
	require.NoError(t, err)
	assert.Equal(t, "review", repo.phase)
	open := true
	_, err = service.UpdateOpen(context.Background(), domain.ChangeUpdateOpenRequest{ID: 2, Open: &open})
	require.NoError(t, err)
	require.NotNil(t, repo.open)
	assert.True(t, *repo.open)
	err = service.DeleteChange(context.Background(), domain.ChangeIDRequest{ID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.id)
}

func TestServiceReturnsCurrentChangeDetails(t *testing.T) {
	repo := &fakeChangeRepository{}
	service := NewService(repo, NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	detail, err := service.GetChange(context.Background(), domain.ChangeIDRequest{ID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, detail.ID)
}

func TestServiceRendersBatchChangeSpecs(t *testing.T) {
	repo := &fakeChangeRepository{}
	service := NewService(repo, NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	response, err := service.RenderedArtifacts(context.Background(), domain.ChangeRenderedArtifactsRequest{
		IDs: []int{3, 2, 3},
	})
	require.NoError(t, err)
	assert.Equal(t, []int{3, 2}, repo.specIDs)
	require.Equal(t, 2, len(response.Artifacts))
	assert.Equal(t, 3, response.Artifacts[0].ID)
	assert.Equal(t, "clean(parsed(**Change 3**))", response.Artifacts[0].SpecHTML)
	assert.Equal(t, 2, response.Artifacts[1].ID)
	assert.Equal(t, "clean(parsed(**Change 2**))", response.Artifacts[1].SpecHTML)
}

func TestServiceRejectsInvalidRenderedSpecIDs(t *testing.T) {
	service := &Service{}
	_, err := service.RenderedArtifacts(context.Background(), domain.ChangeRenderedArtifactsRequest{IDs: []int{1, 0}})
	require.ErrorIs(t, err, apperror.ErrChangeInvalidInput)
}

type fakeMarkdownParser struct{}

func (fakeMarkdownParser) Parse(source string) string {
	return "parsed(" + source + ")"
}

type fakeMarkdownSanitizer struct{}

func (fakeMarkdownSanitizer) Parse(source string) string {
	return "clean(" + source + ")"
}

type fakeChangeRepository struct {
	projectID      int
	id             int
	availableTypes []string
	phase          string
	open           *bool
	specIDs        []int
	createReq      domain.ChangeCreateRequest
	updateTypesReq domain.ChangeUpdateChangeTypesRequest
	updateTitleReq domain.ChangeUpdateTitleRequest
	updateBriefReq domain.ChangeUpdateBriefRequest
	updateSpecReq  domain.ChangeUpdateSpecRequest
	updatePRReq    domain.ChangeUpdatePRRequest
	updatePRUrlReq domain.ChangeUpdatePRUrlRequest
	err            error
}

func (r *fakeChangeRepository) AvailableChangeTypes(_ context.Context) ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]string(nil), r.availableTypes...), nil
}

func (r *fakeChangeRepository) List(_ context.Context, projectID int) ([]domain.ChangeListItem, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.projectID = projectID
	return []domain.ChangeListItem{}, nil
}

func (r *fakeChangeRepository) Details(_ context.Context, id int) (domain.ChangeDetails, error) {
	if r.err != nil {
		return domain.ChangeDetails{}, r.err
	}
	r.id = id
	return domain.ChangeDetails{ChangeListItem: domain.ChangeListItem{ID: id}}, nil
}

func (r *fakeChangeRepository) Artifacts(_ context.Context, ids []int) ([]domain.Change, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.specIDs = ids
	changes := make([]domain.Change, 0, len(ids))
	for _, id := range ids {
		changes = append(changes, domain.Change{ID: id, Spec: "**Change " + strconv.Itoa(id) + "**"})
	}
	return changes, nil
}

func (r *fakeChangeRepository) Create(_ context.Context, req domain.ChangeCreateRequest) (domain.Change, error) {
	if r.err != nil {
		return domain.Change{}, r.err
	}
	r.createReq = req
	change := domain.Change{ID: 2, ProjectID: req.ProjectID, Title: req.Title, Brief: req.Brief}
	if req.RefUUID != nil {
		change.RefUUID = req.RefUUID.String()
	}
	return change, nil
}

func (r *fakeChangeRepository) UpdateChangeTypes(_ context.Context, req domain.ChangeUpdateChangeTypesRequest) error {
	if r.err != nil {
		return r.err
	}
	r.updateTypesReq = req
	return nil
}

func (r *fakeChangeRepository) UpdateTitle(_ context.Context, req domain.ChangeUpdateTitleRequest) (domain.Change, error) {
	if r.err != nil {
		return domain.Change{}, r.err
	}
	r.updateTitleReq = req
	return domain.Change{ID: req.ID, Title: req.Title}, nil
}

func (r *fakeChangeRepository) UpdateBrief(_ context.Context, req domain.ChangeUpdateBriefRequest) (domain.Change, error) {
	if r.err != nil {
		return domain.Change{}, r.err
	}
	r.updateBriefReq = req
	return domain.Change{ID: req.ID, Brief: req.Brief}, nil
}

func (r *fakeChangeRepository) UpdateSpec(_ context.Context, req domain.ChangeUpdateSpecRequest) (domain.Change, error) {
	if r.err != nil {
		return domain.Change{}, r.err
	}
	r.updateSpecReq = req
	return domain.Change{ID: req.ID, Spec: req.Spec}, nil
}

func (r *fakeChangeRepository) UpdatePR(_ context.Context, req domain.ChangeUpdatePRRequest) (domain.Change, error) {
	if r.err != nil {
		return domain.Change{}, r.err
	}
	r.updatePRReq = req
	return domain.Change{ID: req.ID, PR: req.PR}, nil
}

func (r *fakeChangeRepository) UpdatePRUrl(_ context.Context, req domain.ChangeUpdatePRUrlRequest) (domain.Change, error) {
	if r.err != nil {
		return domain.Change{}, r.err
	}
	r.updatePRUrlReq = req
	return domain.Change{ID: req.ID, PRUrl: req.PRUrl}, nil
}

func (r *fakeChangeRepository) UpdateEpic(_ context.Context, req domain.ChangeUpdateEpicRequest) (domain.Change, error) {
	if r.err != nil {
		return domain.Change{}, r.err
	}
	r.id = req.ID
	return domain.Change{ID: req.ID, EpicID: req.EpicID}, nil
}

func (r *fakeChangeRepository) UpdatePhase(_ context.Context, req domain.ChangeUpdatePhaseRequest) (domain.Change, error) {
	if r.err != nil {
		return domain.Change{}, r.err
	}
	r.id, r.phase = req.ID, req.ChangePhase
	return domain.Change{ID: req.ID, ChangePhase: req.ChangePhase}, nil
}

func (r *fakeChangeRepository) UpdateOpen(_ context.Context, req domain.ChangeUpdateOpenRequest) (domain.Change, error) {
	if r.err != nil {
		return domain.Change{}, r.err
	}
	r.id, r.open = req.ID, req.Open
	return domain.Change{ID: req.ID, Open: *req.Open}, nil
}

func (r *fakeChangeRepository) Delete(_ context.Context, req domain.ChangeIDRequest) error {
	if r.err != nil {
		return r.err
	}
	r.id = req.ID
	return nil
}
