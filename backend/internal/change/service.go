package change

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"aipm/internal/dto"

	"github.com/gofrs/uuid/v5"
)

var (
	// ErrInvalidInput is a package-level value.
	ErrInvalidInput = errors.New("invalid change input")
	// ErrInvalidReference is returned when a change reference is invalid.
	ErrInvalidReference = errors.New("invalid change reference")
	// ErrNotFound is returned when a change cannot be found.
	ErrNotFound = errors.New("change not found")
)

// Service defines Service values.
type Service struct {
	repo     Repository
	renderer Renderer
}

// NewService initializes or executes NewService behavior.
func NewService(changeRepository Repository, renderer Renderer) *Service {
	return &Service{repo: changeRepository, renderer: renderer}
}

// ListChanges executes ListChanges behavior.
func (s *Service) ListChanges(ctx context.Context, req dto.ChangeListRequest) ([]dto.ChangeListItem, error) {
	if req.ProjectID <= 0 {
		return nil, ErrInvalidInput
	}
	return s.repo.List(ctx, req.ProjectID)
}

// GetChange executes GetChange behavior.
func (s *Service) GetChange(ctx context.Context, req dto.ChangeIDRequest) (dto.ChangeDetail, error) {
	if req.ID <= 0 {
		return dto.ChangeDetail{}, ErrInvalidInput
	}
	detail, err := s.repo.Get(ctx, req.ID)
	if err != nil {
		return dto.ChangeDetail{}, err
	}
	detail.Change = s.renderer.RenderChange(detail.Change)
	return detail, nil
}

// RenderedArtifacts executes RenderedArtifacts behavior.
func (s *Service) RenderedArtifacts(ctx context.Context, req dto.ChangeRenderedArtifactsRequest) (dto.ChangeRenderedArtifactsResponse, error) {
	ids, err := normalizeIDs(req.IDs)
	if err != nil {
		return dto.ChangeRenderedArtifactsResponse{}, err
	}
	if len(ids) == 0 {
		return dto.ChangeRenderedArtifactsResponse{Artifacts: []dto.ChangeRenderedArtifact{}}, nil
	}
	changes, err := s.repo.Artifacts(ctx, ids)
	if err != nil {
		return dto.ChangeRenderedArtifactsResponse{}, err
	}
	artifacts := make([]dto.ChangeRenderedArtifact, 0, len(changes))
	for _, item := range changes {
		item = s.renderer.RenderChange(item)
		artifacts = append(artifacts, dto.ChangeRenderedArtifact{
			ID:       item.ID,
			SpecHTML: item.SpecHTML,
			PRHtml:   item.PRHtml,
		})
	}
	return dto.ChangeRenderedArtifactsResponse{Artifacts: artifacts}, nil
}

// CreateChange executes CreateChange behavior.
func (s *Service) CreateChange(ctx context.Context, req dto.ChangeCreateRequest) (dto.Change, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Def = strings.TrimSpace(req.Def)
	if req.ProjectID <= 0 || req.Title == "" || req.Def == "" {
		return dto.Change{}, ErrInvalidInput
	}
	if req.RefUUID == nil {
		u, err := uuid.NewV7()
		if err != nil {
			return dto.Change{}, err
		}
		req.RefUUID = &u
	}
	change, err := s.repo.Create(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// UpdateChangeTypes executes UpdateChangeTypes behavior.
func (s *Service) UpdateChangeTypes(ctx context.Context, req dto.ChangeUpdateChangeTypesRequest) (dto.Change, error) {
	req.ChangeTypes = normalizeTypes(req.ChangeTypes)
	if req.ID <= 0 {
		return dto.Change{}, ErrInvalidInput
	}
	available, err := s.repo.AvailableChangeTypes(ctx)
	if err != nil {
		return dto.Change{}, err
	}
	req.ChangeTypes = intersectTypes(req.ChangeTypes, available)
	change, err := s.repo.UpdateChangeTypes(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// UpdateTitle executes UpdateTitle behavior.
func (s *Service) UpdateTitle(ctx context.Context, req dto.ChangeUpdateTitleRequest) (dto.Change, error) {
	req.Title = strings.TrimSpace(req.Title)
	if req.ID <= 0 || req.Title == "" {
		return dto.Change{}, ErrInvalidInput
	}
	change, err := s.repo.UpdateTitle(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// UpdateDef executes UpdateDef behavior.
func (s *Service) UpdateDef(ctx context.Context, req dto.ChangeUpdateDefRequest) (dto.Change, error) {
	req.Def = strings.TrimSpace(req.Def)
	if req.ID <= 0 || req.Def == "" || req.AgentEdit == nil {
		return dto.Change{}, ErrInvalidInput
	}
	change, err := s.repo.UpdateDef(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// UpdateSpec executes UpdateSpec behavior.
func (s *Service) UpdateSpec(ctx context.Context, req dto.ChangeUpdateSpecRequest) (dto.Change, error) {
	req.Spec = strings.TrimSpace(req.Spec)
	if req.ID <= 0 || req.Spec == "" || req.AgentEdit == nil {
		return dto.Change{}, ErrInvalidInput
	}
	change, err := s.repo.UpdateSpec(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// UpdatePR executes UpdatePR behavior.
func (s *Service) UpdatePR(ctx context.Context, req dto.ChangeUpdatePRRequest) (dto.Change, error) {
	req.PR = strings.TrimSpace(req.PR)
	if req.ID <= 0 || req.PR == "" || req.AgentEdit == nil {
		return dto.Change{}, ErrInvalidInput
	}
	change, err := s.repo.UpdatePR(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// UpdatePRUrl executes UpdatePRUrl behavior.
func (s *Service) UpdatePRUrl(ctx context.Context, req dto.ChangeUpdatePRUrlRequest) (dto.Change, error) {
	req.PRUrl = strings.TrimSpace(req.PRUrl)
	if req.ID <= 0 || req.PRUrl == "" || invalidPRURL(req.PRUrl) {
		return dto.Change{}, ErrInvalidInput
	}
	change, err := s.repo.UpdatePRUrl(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// UpdateEpic executes UpdateEpic behavior.
func (s *Service) UpdateEpic(ctx context.Context, req dto.ChangeUpdateEpicRequest) (dto.Change, error) {
	if req.ID <= 0 || invalidOptionalID(req.EpicID) {
		return dto.Change{}, ErrInvalidInput
	}
	change, err := s.repo.UpdateEpic(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// UpdatePhase executes UpdatePhase behavior.
func (s *Service) UpdatePhase(ctx context.Context, req dto.ChangeUpdatePhaseRequest) (dto.Change, error) {
	req.ChangePhase = strings.TrimSpace(req.ChangePhase)
	if req.ID <= 0 || req.ChangePhase == "" {
		return dto.Change{}, ErrInvalidInput
	}
	change, err := s.repo.UpdatePhase(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// UpdateOpen executes UpdateOpen behavior.
func (s *Service) UpdateOpen(ctx context.Context, req dto.ChangeUpdateOpenRequest) (dto.Change, error) {
	if req.ID <= 0 || req.Open == nil {
		return dto.Change{}, ErrInvalidInput
	}
	change, err := s.repo.UpdateOpen(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// AssignFlow executes AssignFlow behavior.
func (s *Service) AssignFlow(ctx context.Context, req dto.ChangeIDRequest) (dto.Change, error) {
	if req.ID <= 0 {
		return dto.Change{}, ErrInvalidInput
	}
	change, err := s.repo.AssignFlow(ctx, req)
	if err != nil {
		return dto.Change{}, err
	}
	return s.renderer.RenderChange(change), nil
}

// StartRun executes StartRun behavior.
func (s *Service) StartRun(ctx context.Context, req dto.ChangeIDRequest) (dto.ChangeRunClaimResponse, error) {
	if req.ID <= 0 {
		return dto.ChangeRunClaimResponse{}, ErrInvalidInput
	}
	return s.repo.StartRun(ctx, req)
}

// UpdateRun executes UpdateRun behavior.
func (s *Service) UpdateRun(ctx context.Context, req dto.ChangeUpdateRunRequest) (dto.ChangeRunUpdateResponse, error) {
	req.RunClaimID = strings.TrimSpace(req.RunClaimID)
	req.RunFlowStage = strings.TrimSpace(req.RunFlowStage)
	req.RunTaskStep = strings.TrimSpace(req.RunTaskStep)
	req.RunTaskStatus = strings.TrimSpace(req.RunTaskStatus)
	req.RunError = strings.TrimSpace(req.RunError)
	if req.ID <= 0 || req.RunClaimID == "" {
		return dto.ChangeRunUpdateResponse{}, ErrInvalidInput
	}
	return s.repo.UpdateRun(ctx, req)
}

// ResetClaim executes ResetClaim behavior.
func (s *Service) ResetClaim(ctx context.Context, req dto.ChangeIDRequest) (dto.ChangeRunClaimResponse, error) {
	if req.ID <= 0 {
		return dto.ChangeRunClaimResponse{}, ErrInvalidInput
	}
	return s.repo.ResetClaim(ctx, req)
}

// DeleteChange executes DeleteChange behavior.
func (s *Service) DeleteChange(ctx context.Context, req dto.ChangeIDRequest) error {
	if req.ID <= 0 {
		return ErrInvalidInput
	}
	return s.repo.Delete(ctx, req)
}

func normalizeIDs(ids []int) ([]int, error) {
	normalized := make([]int, 0, len(ids))
	seen := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, ErrInvalidInput
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
	if value == "" {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return true
	}
	if parsed.Host == "" {
		return true
	}
	return !strings.EqualFold(parsed.Scheme, "https") && !strings.EqualFold(parsed.Scheme, "http")
}
