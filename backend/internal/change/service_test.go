package change

import (
	"context"
	"strconv"
	"testing"

	"aipm/internal/dto"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceResolvesChangeCreateIdentity(t *testing.T) {
	repo := &fakeChangeRepository{}
	service := NewService(repo, NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	generated, err := service.CreateChange(context.Background(), dto.ChangeCreateRequest{ProjectID: 1, Title: "Generated", Def: "Def"})
	require.NoError(t, err)
	require.NotNil(t, repo.createReq.RefUUID)
	assert.Equal(t, byte(7), repo.createReq.RefUUID.Version())
	assert.Equal(t, repo.createReq.RefUUID.String(), generated.RefUUID)

	supplied := uuid.Must(uuid.FromString("0198a86f-9b8a-7d89-ae5b-6f25b528b04c"))
	preserved, err := service.CreateChange(context.Background(), dto.ChangeCreateRequest{ProjectID: 1, RefUUID: &supplied, Title: "Supplied", Def: "Def"})
	require.NoError(t, err)
	require.NotNil(t, repo.createReq.RefUUID)
	assert.Equal(t, supplied, *repo.createReq.RefUUID)
	assert.Equal(t, supplied.String(), preserved.RefUUID)
}

func TestServiceRejectsInvalidChangeInput(t *testing.T) {
	service := &Service{}
	_, err := service.ListChanges(context.Background(), dto.ChangeListRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.GetChange(context.Background(), dto.ChangeIDRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.CreateChange(context.Background(), dto.ChangeCreateRequest{
		ProjectID: 1, Title: "   ",
	})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateTitle(context.Background(), dto.ChangeUpdateTitleRequest{ID: 2, Title: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdatePhase(context.Background(), dto.ChangeUpdatePhaseRequest{ID: 2, ChangePhase: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateDef(context.Background(), dto.ChangeUpdateDefRequest{ID: 2, Def: "def"})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateSpec(context.Background(), dto.ChangeUpdateSpecRequest{ID: 2, Spec: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateSpec(context.Background(), dto.ChangeUpdateSpecRequest{ID: 2, Spec: "spec"})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdatePR(context.Background(), dto.ChangeUpdatePRRequest{ID: 2, PR: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdatePR(context.Background(), dto.ChangeUpdatePRRequest{ID: 2, PR: "pr"})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdatePRUrl(context.Background(), dto.ChangeUpdatePRUrlRequest{ID: 2, PRUrl: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateOpen(context.Background(), dto.ChangeUpdateOpenRequest{ID: 2})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.AssignFlow(context.Background(), dto.ChangeIDRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.StartRun(context.Background(), dto.ChangeIDRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateRun(context.Background(), dto.ChangeUpdateRunRequest{ID: 2, RunClaimID: "   "})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.UpdateRun(context.Background(), dto.ChangeUpdateRunRequest{RunClaimID: "00000000-0000-0000-0000-000000000001"})
	require.ErrorIs(t, err, ErrInvalidInput)
	_, err = service.ResetClaim(context.Background(), dto.ChangeIDRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
	badURL := "javascript:alert(1)"
	_, err = service.UpdatePRUrl(context.Background(), dto.ChangeUpdatePRUrlRequest{ID: 2, PRUrl: badURL})
	require.ErrorIs(t, err, ErrInvalidInput)
	missingHostURL := "https:///missing-host"
	_, err = service.UpdatePRUrl(context.Background(), dto.ChangeUpdatePRUrlRequest{ID: 2, PRUrl: missingHostURL})
	require.ErrorIs(t, err, ErrInvalidInput)
	err = service.DeleteChange(context.Background(), dto.ChangeIDRequest{})
	require.ErrorIs(t, err, ErrInvalidInput)
}

func TestServiceNormalizesChangeRequests(t *testing.T) {
	repo := &fakeChangeRepository{availableTypes: []string{"fix"}}
	service := NewService(repo, NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))
	epicID := 4
	agentEdit := false

	_, err := service.ListChanges(context.Background(), dto.ChangeListRequest{ProjectID: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, repo.projectID)
	_, err = service.GetChange(context.Background(), dto.ChangeIDRequest{ID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.id)

	_, err = service.CreateChange(context.Background(), dto.ChangeCreateRequest{ProjectID: 1, Title: " Change Title ", Def: " Def "})
	require.NoError(t, err)
	assert.Equal(t, "Change Title", repo.createReq.Title)
	assert.Equal(t, "Def", repo.createReq.Def)

	_, err = service.UpdateChangeTypes(context.Background(), dto.ChangeUpdateChangeTypesRequest{ID: 2, ChangeTypes: []string{" fix ", "missing", "fix "}})
	require.NoError(t, err)
	assert.Equal(t, []string{"fix"}, repo.updateTypesReq.ChangeTypes)
	_, err = service.UpdateChangeTypes(context.Background(), dto.ChangeUpdateChangeTypesRequest{ID: 2, ChangeTypes: []string{"missing"}})
	require.NoError(t, err)
	assert.Empty(t, repo.updateTypesReq.ChangeTypes)
	_, err = service.UpdateTitle(context.Background(), dto.ChangeUpdateTitleRequest{ID: 2, Title: " Focused Title "})
	require.NoError(t, err)
	assert.Equal(t, "Focused Title", repo.updateTitleReq.Title)
	_, err = service.UpdateDef(context.Background(), dto.ChangeUpdateDefRequest{ID: 2, Def: " Focused Def ", AgentEdit: &agentEdit})
	require.NoError(t, err)
	assert.Equal(t, "Focused Def", repo.updateDefReq.Def)
	spec := " Focused Spec "
	_, err = service.UpdateSpec(context.Background(), dto.ChangeUpdateSpecRequest{ID: 2, Spec: spec, AgentEdit: &agentEdit})
	require.NoError(t, err)
	assert.Equal(t, "Focused Spec", repo.updateSpecReq.Spec)
	pr := " PR Body "
	_, err = service.UpdatePR(context.Background(), dto.ChangeUpdatePRRequest{ID: 2, PR: pr, AgentEdit: &agentEdit})
	require.NoError(t, err)
	assert.Equal(t, "PR Body", repo.updatePRReq.PR)
	prURL := " https://example.test/pr "
	_, err = service.UpdatePRUrl(context.Background(), dto.ChangeUpdatePRUrlRequest{ID: 2, PRUrl: prURL})
	require.NoError(t, err)
	assert.Equal(t, "https://example.test/pr", repo.updatePRUrlReq.PRUrl)
	_, err = service.UpdateEpic(context.Background(), dto.ChangeUpdateEpicRequest{ID: 2, EpicID: &epicID})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.id)
	_, err = service.UpdatePhase(context.Background(), dto.ChangeUpdatePhaseRequest{ID: 2, ChangePhase: " review "})
	require.NoError(t, err)
	assert.Equal(t, "review", repo.phase)
	open := true
	_, err = service.UpdateOpen(context.Background(), dto.ChangeUpdateOpenRequest{ID: 2, Open: &open})
	require.NoError(t, err)
	require.NotNil(t, repo.open)
	assert.True(t, *repo.open)
	_, err = service.AssignFlow(context.Background(), dto.ChangeIDRequest{ID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.id)
	_, err = service.StartRun(context.Background(), dto.ChangeIDRequest{ID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.id)
	_, err = service.UpdateRun(context.Background(), dto.ChangeUpdateRunRequest{
		ID:             2,
		RunClaimID:     " 00000000-0000-0000-0000-000000000001 ",
		RunFlowStage:   " docs ",
		RunTaskStep:    " agent ",
		RunTaskStatus:  " running ",
		RunError:       " latest error ",
		RunIsCompleted: true,
	})
	require.NoError(t, err)
	assert.Equal(t, dto.ChangeUpdateRunRequest{
		ID:             2,
		RunClaimID:     "00000000-0000-0000-0000-000000000001",
		RunFlowStage:   "docs",
		RunTaskStep:    "agent",
		RunTaskStatus:  "running",
		RunError:       "latest error",
		RunIsCompleted: true,
	}, repo.updateRunReq)
	_, err = service.ResetClaim(context.Background(), dto.ChangeIDRequest{ID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.id)
	err = service.DeleteChange(context.Background(), dto.ChangeIDRequest{ID: 2})
	require.NoError(t, err)
	assert.Equal(t, 2, repo.id)
}

func TestServicePropagatesRunOperationErrors(t *testing.T) {
	repo := &fakeChangeRepository{err: ErrNotFound}
	service := NewService(repo, NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	_, err := service.AssignFlow(context.Background(), dto.ChangeIDRequest{ID: 2})
	require.ErrorIs(t, err, ErrNotFound)
	_, err = service.StartRun(context.Background(), dto.ChangeIDRequest{ID: 2})
	require.ErrorIs(t, err, ErrNotFound)
	_, err = service.UpdateRun(context.Background(), dto.ChangeUpdateRunRequest{
		ID:             2,
		RunClaimID:     "00000000-0000-0000-0000-000000000001",
		RunFlowStage:   "docs",
		RunTaskStep:    "agent",
		RunTaskStatus:  "running",
		RunIsCompleted: false,
	})
	require.ErrorIs(t, err, ErrNotFound)
	_, err = service.ResetClaim(context.Background(), dto.ChangeIDRequest{ID: 2})
	require.ErrorIs(t, err, ErrNotFound)
}

func TestServiceRendersChangeSpecHTML(t *testing.T) {
	repo := &fakeChangeRepository{}
	service := NewService(repo, NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	detail, err := service.GetChange(context.Background(), dto.ChangeIDRequest{ID: 2})
	require.NoError(t, err)
	assert.Equal(t, "clean(parsed(**Change**))", detail.Change.SpecHTML)
}

func TestServiceRendersBatchChangeSpecs(t *testing.T) {
	repo := &fakeChangeRepository{}
	service := NewService(repo, NewRenderer(fakeMarkdownParser{}, fakeMarkdownSanitizer{}))

	response, err := service.RenderedArtifacts(context.Background(), dto.ChangeRenderedArtifactsRequest{
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
	_, err := service.RenderedArtifacts(context.Background(), dto.ChangeRenderedArtifactsRequest{IDs: []int{1, 0}})
	require.ErrorIs(t, err, ErrInvalidInput)
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
	createReq      dto.ChangeCreateRequest
	updateTypesReq dto.ChangeUpdateChangeTypesRequest
	updateTitleReq dto.ChangeUpdateTitleRequest
	updateDefReq   dto.ChangeUpdateDefRequest
	updateSpecReq  dto.ChangeUpdateSpecRequest
	updatePRReq    dto.ChangeUpdatePRRequest
	updatePRUrlReq dto.ChangeUpdatePRUrlRequest
	updateRunReq   dto.ChangeUpdateRunRequest
	err            error
}

func (r *fakeChangeRepository) AvailableChangeTypes(_ context.Context) ([]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	return append([]string(nil), r.availableTypes...), nil
}

func (r *fakeChangeRepository) List(_ context.Context, projectID int) ([]dto.ChangeListItem, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.projectID = projectID
	return []dto.ChangeListItem{}, nil
}

func (r *fakeChangeRepository) Get(_ context.Context, id int) (dto.ChangeDetail, error) {
	if r.err != nil {
		return dto.ChangeDetail{}, r.err
	}
	r.id = id
	return dto.ChangeDetail{Change: dto.Change{ID: id, Spec: "**Change**"}}, nil
}

func (r *fakeChangeRepository) Artifacts(_ context.Context, ids []int) ([]dto.Change, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.specIDs = ids
	changes := make([]dto.Change, 0, len(ids))
	for _, id := range ids {
		changes = append(changes, dto.Change{ID: id, Spec: "**Change " + strconv.Itoa(id) + "**"})
	}
	return changes, nil
}

func (r *fakeChangeRepository) Create(_ context.Context, req dto.ChangeCreateRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.createReq = req
	change := dto.Change{ID: 2, ProjectID: req.ProjectID, Title: req.Title, Def: req.Def}
	if req.RefUUID != nil {
		change.RefUUID = req.RefUUID.String()
	}
	return change, nil
}

func (r *fakeChangeRepository) UpdateChangeTypes(_ context.Context, req dto.ChangeUpdateChangeTypesRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.updateTypesReq = req
	return dto.Change{ID: req.ID, ChangeTypes: req.ChangeTypes}, nil
}

func (r *fakeChangeRepository) UpdateTitle(_ context.Context, req dto.ChangeUpdateTitleRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.updateTitleReq = req
	return dto.Change{ID: req.ID, Title: req.Title}, nil
}

func (r *fakeChangeRepository) UpdateDef(_ context.Context, req dto.ChangeUpdateDefRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.updateDefReq = req
	return dto.Change{ID: req.ID, Def: req.Def}, nil
}

func (r *fakeChangeRepository) UpdateSpec(_ context.Context, req dto.ChangeUpdateSpecRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.updateSpecReq = req
	return dto.Change{ID: req.ID, Spec: req.Spec}, nil
}

func (r *fakeChangeRepository) UpdatePR(_ context.Context, req dto.ChangeUpdatePRRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.updatePRReq = req
	return dto.Change{ID: req.ID, PR: req.PR}, nil
}

func (r *fakeChangeRepository) UpdatePRUrl(_ context.Context, req dto.ChangeUpdatePRUrlRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.updatePRUrlReq = req
	return dto.Change{ID: req.ID, PRUrl: req.PRUrl}, nil
}

func (r *fakeChangeRepository) UpdateEpic(_ context.Context, req dto.ChangeUpdateEpicRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.id = req.ID
	return dto.Change{ID: req.ID, EpicID: req.EpicID}, nil
}

func (r *fakeChangeRepository) UpdatePhase(_ context.Context, req dto.ChangeUpdatePhaseRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.id, r.phase = req.ID, req.ChangePhase
	return dto.Change{ID: req.ID, ChangePhase: req.ChangePhase}, nil
}

func (r *fakeChangeRepository) UpdateOpen(_ context.Context, req dto.ChangeUpdateOpenRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.id, r.open = req.ID, req.Open
	return dto.Change{ID: req.ID, Open: *req.Open}, nil
}

func (r *fakeChangeRepository) Delete(_ context.Context, req dto.ChangeIDRequest) error {
	if r.err != nil {
		return r.err
	}
	r.id = req.ID
	return nil
}

func (r *fakeChangeRepository) AssignFlow(_ context.Context, req dto.ChangeIDRequest) (dto.Change, error) {
	if r.err != nil {
		return dto.Change{}, r.err
	}
	r.id = req.ID
	return dto.Change{ID: req.ID}, nil
}

func (r *fakeChangeRepository) StartRun(_ context.Context, req dto.ChangeIDRequest) (dto.ChangeRunClaimResponse, error) {
	if r.err != nil {
		return dto.ChangeRunClaimResponse{}, r.err
	}
	r.id = req.ID
	claimID := "00000000-0000-0000-0000-000000000001"
	return dto.ChangeRunClaimResponse{ClaimID: &claimID}, nil
}

func (r *fakeChangeRepository) UpdateRun(_ context.Context, req dto.ChangeUpdateRunRequest) (dto.ChangeRunUpdateResponse, error) {
	if r.err != nil {
		return dto.ChangeRunUpdateResponse{}, r.err
	}
	r.updateRunReq = req
	changeID := req.ID
	return dto.ChangeRunUpdateResponse{ChangeID: &changeID}, nil
}

func (r *fakeChangeRepository) ResetClaim(_ context.Context, req dto.ChangeIDRequest) (dto.ChangeRunClaimResponse, error) {
	if r.err != nil {
		return dto.ChangeRunClaimResponse{}, r.err
	}
	r.id = req.ID
	claimID := "00000000-0000-0000-0000-000000000002"
	return dto.ChangeRunClaimResponse{ClaimID: &claimID}, nil
}
