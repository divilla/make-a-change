package change

import (
	"context"
	"mch_api/internal/domain"
	apperror "mch_api/internal/error"
	"net/url"
	"slices"
	"strings"

	"github.com/gofrs/uuid/v5"
)

// ProjectConfig exposes the selected project configuration without unrelated reads.
type ProjectConfig interface {
	Config(context.Context, domain.ProjectIDRequest) (domain.Config, error)
}

// Service validates change operations and renders explicit document reads.
type Service struct {
	repo     Repository
	renderer Renderer
	projects ProjectConfig
	newUUID  func() (uuid.UUID, error)
}

// NewService injects only persistence, rendering and selected project configuration.
func NewService(repo Repository, renderer Renderer, projects ProjectConfig) *Service {
	return &Service{repo: repo, renderer: renderer, projects: projects, newUUID: uuid.NewV7}
}

// ListChanges derives completion from wide database counts.
func (s *Service) ListChanges(ctx context.Context, req domain.ChangeListRequest) ([]domain.ChangeListItem, error) {
	if req.ProjectID <= 0 {
		return nil, apperror.ErrChangeInvalidInput
	}
	changes, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	for i := range changes {
		changes[i].Completed = completion(changes[i].DoneTC, changes[i].TotalTC)
	}
	return changes, nil
}

// GetChange returns current stored fields and derived completion.
func (s *Service) GetChange(ctx context.Context, req domain.ChangeIDRequest) (domain.ChangeDetails, error) {
	if req.ID <= 0 {
		return domain.ChangeDetails{}, apperror.ErrChangeInvalidInput
	}
	change, err := s.repo.Details(ctx, req)
	if err != nil {
		return domain.ChangeDetails{}, err
	}
	change.Completed = completion(change.DoneTC, change.TotalTC)
	return change, nil
}

func completion(done, total int64) int64 {
	if total == 0 {
		return 0
	}
	return 100 * done / total
}

// RenderedArtifacts renders only current spec/PR documents, preserving request order.
func (s *Service) RenderedArtifacts(ctx context.Context, req domain.ChangeRenderedArtifactsRequest) (domain.ChangeRenderedArtifactsResponse, error) {
	ids, err := normalizeIDs(req.IDs)
	if err != nil {
		return domain.ChangeRenderedArtifactsResponse{}, err
	}
	result := domain.ChangeRenderedArtifactsResponse{Artifacts: []domain.ChangeRenderedArtifact{}}
	if len(ids) == 0 {
		return result, nil
	}
	sources, err := s.repo.Artifacts(ctx, domain.ChangeRenderedArtifactsRequest{IDs: ids})
	if err != nil {
		return domain.ChangeRenderedArtifactsResponse{}, err
	}
	for _, source := range sources {
		result.Artifacts = append(result.Artifacts, domain.ChangeRenderedArtifact{ID: source.ID, SpecHTML: s.renderer.Render(source.Spec), PRHtml: s.renderer.Render(source.PR)})
	}
	return result, nil
}

// Documents checks live-parent existence and renders current raw documents separately from writes.
func (s *Service) Documents(ctx context.Context, req domain.ChangeIDRequest) ([]domain.ChangeDocument, error) {
	if req.ID <= 0 {
		return nil, apperror.ErrChangeInvalidInput
	}
	if err := s.repo.Exists(ctx, req); err != nil {
		return nil, err
	}
	docs, err := s.repo.Documents(ctx, req)
	if err != nil {
		return nil, err
	}
	for i := range docs {
		docs[i].HTML = s.renderer.Render(docs[i].Body)
	}
	return docs, nil
}

// CreateChange validates database function defaults against the selected configuration.
func (s *Service) CreateChange(ctx context.Context, req domain.ChangeCreateRequest) (domain.ChangeIDRequest, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Brief = strings.TrimSpace(req.Brief)
	if req.ProjectID <= 0 || req.Title == "" || req.Brief == "" {
		return domain.ChangeIDRequest{}, apperror.ErrChangeInvalidInput
	}
	config, err := s.projects.Config(ctx, domain.ProjectIDRequest{ID: req.ProjectID})
	if err != nil {
		return domain.ChangeIDRequest{}, err
	}
	if !slices.Contains(config.ChangePhases, "backlog") || !slices.Contains(config.ChangeDocs, "brief") {
		return domain.ChangeIDRequest{}, apperror.ErrChangeInvalidReference
	}
	if req.RefUUID == nil {
		id, err := s.newUUID()
		if err != nil {
			return domain.ChangeIDRequest{}, apperror.Wrap(err, "generate change UUID")
		}
		req.RefUUID = &id
	}
	return s.repo.Create(ctx, req)
}

func (s *Service) config(ctx context.Context, req domain.ChangeIDRequest) (domain.Config, error) {
	project, err := s.repo.Project(ctx, req)
	if err != nil {
		return domain.Config{}, err
	}
	return s.projects.Config(ctx, project)
}

// UpdateChangeTypes retains ordered filtering against this project's selected types.
func (s *Service) UpdateChangeTypes(ctx context.Context, req domain.ChangeUpdateChangeTypesRequest) error {
	if req.ID <= 0 {
		return apperror.ErrChangeInvalidInput
	}
	config, err := s.config(ctx, domain.ChangeIDRequest{ID: req.ID})
	if err != nil {
		return err
	}
	req.ChangeTypes = intersectTypes(normalizeTypes(req.ChangeTypes), config.ChangeTypes)
	return s.repo.UpdateChangeTypes(ctx, req)
}

// UpdateTitle leaves internal whitespace normalization to the database procedure.
func (s *Service) UpdateTitle(ctx context.Context, req domain.ChangeUpdateTitleRequest) error {
	req.Title = strings.TrimSpace(req.Title)
	if req.ID <= 0 || req.Title == "" {
		return apperror.ErrChangeInvalidInput
	}
	if err := s.repo.Exists(ctx, domain.ChangeIDRequest{ID: req.ID}); err != nil {
		return err
	}
	return s.repo.UpdateTitle(ctx, req)
}

// UpdatePhase rejects phases absent from the selected project configuration.
func (s *Service) UpdatePhase(ctx context.Context, req domain.ChangeUpdatePhaseRequest) error {
	req.ChangePhase = strings.TrimSpace(req.ChangePhase)
	if req.ID <= 0 || req.ChangePhase == "" {
		return apperror.ErrChangeInvalidInput
	}
	config, err := s.config(ctx, domain.ChangeIDRequest{ID: req.ID})
	if err != nil {
		return err
	}
	if !slices.Contains(config.ChangePhases, req.ChangePhase) {
		return apperror.ErrChangeInvalidReference
	}
	return s.repo.UpdatePhase(ctx, req)
}

// UpdateEpic checks same-project association; separate preflight and CALL are not atomic.
func (s *Service) UpdateEpic(ctx context.Context, req domain.ChangeUpdateEpicRequest) error {
	if req.ID <= 0 || invalidOptionalID(req.EpicID) {
		return apperror.ErrChangeInvalidInput
	}
	project, err := s.repo.Project(ctx, domain.ChangeIDRequest{ID: req.ID})
	if err != nil {
		return err
	}
	if req.EpicID != nil {
		parent, err := s.repo.EpicProject(ctx, domain.EpicIDRequest{ID: *req.EpicID})
		if err != nil {
			return err
		}
		if parent != project {
			return apperror.ErrChangeInvalidReference
		}
	}
	return s.repo.UpdateEpic(ctx, req)
}

// UpdateOpen requires an explicit boolean, including false.
func (s *Service) UpdateOpen(ctx context.Context, req domain.ChangeUpdateOpenRequest) error {
	if req.ID <= 0 || req.Open == nil {
		return apperror.ErrChangeInvalidInput
	}
	return s.repo.UpdateOpen(ctx, req)
}

// UpdatePRUrl accepts nonblank HTTP(S) URLs only.
func (s *Service) UpdatePRUrl(ctx context.Context, req domain.ChangeUpdatePRUrlRequest) error {
	req.PRUrl = strings.TrimSpace(req.PRUrl)
	if req.ID <= 0 || req.PRUrl == "" || invalidPRURL(req.PRUrl) {
		return apperror.ErrChangeInvalidInput
	}
	return s.repo.UpdatePRUrl(ctx, req)
}

// SetDocument appends a document through the database's atomic workflow.
func (s *Service) SetDocument(ctx context.Context, req domain.ChangeDocumentSetRequest) error {
	req.DocType = strings.TrimSpace(req.DocType)
	req.Body = strings.TrimSpace(req.Body)
	if req.ID <= 0 || req.DocType == "" || req.Body == "" || req.AgentEdit == nil {
		return apperror.ErrChangeInvalidInput
	}
	config, err := s.config(ctx, domain.ChangeIDRequest{ID: req.ID})
	if err != nil {
		return err
	}
	if !slices.Contains(config.ChangeDocs, req.DocType) {
		return apperror.ErrChangeInvalidReference
	}
	return s.repo.SetDocument(ctx, req)
}

// UpdateBrief maps the specialized operation to the shared document request.
func (s *Service) UpdateBrief(ctx context.Context, req domain.ChangeUpdateBriefRequest) error {
	return s.SetDocument(ctx, domain.ChangeDocumentSetRequest{ID: req.ID, DocType: "brief", Body: req.Brief, AgentEdit: req.AgentEdit})
}

// UpdateSpec maps the specialized operation to the shared document request.
func (s *Service) UpdateSpec(ctx context.Context, req domain.ChangeUpdateSpecRequest) error {
	return s.SetDocument(ctx, domain.ChangeDocumentSetRequest{ID: req.ID, DocType: "spec", Body: req.Spec, AgentEdit: req.AgentEdit})
}

// UpdatePR maps the specialized operation to the shared document request.
func (s *Service) UpdatePR(ctx context.Context, req domain.ChangeUpdatePRRequest) error {
	return s.SetDocument(ctx, domain.ChangeDocumentSetRequest{ID: req.ID, DocType: "pr", Body: req.PR, AgentEdit: req.AgentEdit})
}

// DeleteChange leaves actual FK conflicts and affected-row semantics to one DELETE.
func (s *Service) DeleteChange(ctx context.Context, req domain.ChangeIDRequest) error {
	if req.ID <= 0 {
		return apperror.ErrChangeInvalidInput
	}
	return s.repo.Delete(ctx, req)
}

func normalizeIDs(ids []int) ([]int, error) {
	normalized := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, apperror.ErrChangeInvalidInput
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized, nil
}

func normalizeTypes(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func intersectTypes(values, available []string) []string {
	availableSet := make(map[string]struct{}, len(available))
	for _, value := range available {
		availableSet[value] = struct{}{}
	}
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := availableSet[value]; ok {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func invalidOptionalID(value *int) bool {
	return value != nil && *value <= 0
}

func invalidPRURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil {
		return true
	}
	if parsed.Host == "" {
		return true
	}
	return !strings.EqualFold(parsed.Scheme, "https") && !strings.EqualFold(parsed.Scheme, "http")
}
