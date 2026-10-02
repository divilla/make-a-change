package change

import (
	"context"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/gofrs/uuid/v5"
)

// ProjectConfig exposes the selected project configuration without unrelated reads.
type ProjectConfig interface {
	Config(context.Context, domain.ProjectIDRequest) (domain.Config, error)
}

// Service validates change operations.
type Service struct {
	repo     Repository
	projects ProjectConfig
	newUUID  func() (uuid.UUID, error)
}

// NewService injects only persistence and selected project configuration.
func NewService(repo Repository, projects ProjectConfig) *Service {
	return &Service{repo: repo, projects: projects, newUUID: uuid.NewV7}
}

// List derives completion from wide database counts.
func (s *Service) List(ctx context.Context, req domain.ChangeListRequest) ([]domain.ChangeListItem, error) {
	if req.ProjectID <= 0 {
		return nil, app.ErrChangeInvalidInput
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

// ListInactive returns inactive entries with the same derived counters.
func (s *Service) ListInactive(ctx context.Context, req domain.ChangeListRequest) ([]domain.ChangeListItem, error) {
	if req.ProjectID <= 0 {
		return nil, app.ErrChangeInvalidInput
	}
	changes, err := s.repo.ListInactive(ctx, req)
	if err != nil {
		return nil, err
	}
	for i := range changes {
		changes[i].Completed = completion(changes[i].DoneTC, changes[i].TotalTC)
	}
	return changes, nil
}

// Details returns current stored fields and derived completion.
func (s *Service) Details(ctx context.Context, req domain.ChangeIDRequest) (domain.ChangeDetails, error) {
	if req.ID <= 0 {
		return domain.ChangeDetails{}, app.ErrChangeInvalidInput
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

// Create validates database function defaults against the selected configuration.
func (s *Service) Create(ctx context.Context, req domain.ChangeCreateRequest) (domain.ChangeIDRequest, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Brief = strings.TrimSpace(req.Brief)
	if req.ProjectID <= 0 || req.Title == "" || req.Brief == "" {
		return domain.ChangeIDRequest{}, app.ErrChangeInvalidInput
	}
	config, err := s.projects.Config(ctx, domain.ProjectIDRequest{ID: req.ProjectID})
	if err != nil {
		return domain.ChangeIDRequest{}, err
	}
	if !slices.Contains(config.ChangePhases, "backlog") || !slices.Contains(config.ChangeDocs, "brief") {
		return domain.ChangeIDRequest{}, app.ErrChangeInvalidReference
	}
	if req.RefUUID == nil {
		id, err := s.newUUID()
		if err != nil {
			return domain.ChangeIDRequest{}, app.WrapError(err, "generate change UUID")
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

// UpdateTypes validates normalized types against this project's selected configuration.
func (s *Service) UpdateTypes(ctx context.Context, req domain.ChangeUpdateTypesRequest) error {
	if req.ID <= 0 {
		return app.ErrChangeInvalidInput
	}
	config, err := s.config(ctx, domain.ChangeIDRequest{ID: req.ID})
	if err != nil {
		return err
	}
	req.ChangeTypes = normalizeTypes(req.ChangeTypes)
	for _, kind := range req.ChangeTypes {
		if !slices.Contains(config.ChangeTypes, kind) {
			return app.ErrChangeInvalidReference
		}
	}
	return s.repo.UpdateTypes(ctx, req)
}

// UpdateTitle leaves internal whitespace normalization to the database procedure.
func (s *Service) UpdateTitle(ctx context.Context, req domain.ChangeUpdateTitleRequest) error {
	req.Title = strings.TrimSpace(req.Title)
	if req.ID <= 0 || req.Title == "" {
		return app.ErrChangeInvalidInput
	}
	if err := s.repo.Exists(ctx, domain.ChangeIDRequest{ID: req.ID}); err != nil {
		return err
	}
	return s.repo.UpdateTitle(ctx, req)
}

var slugPattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

// UpdateSlug validates the editable suffix; the view adds the reference prefix.
func (s *Service) UpdateSlug(ctx context.Context, req domain.ChangeUpdateSlugRequest) error {
	if req.ID <= 0 {
		return app.ErrChangeInvalidInput
	}
	if !slugPattern.MatchString(req.Slug) {
		return app.ErrChangeInvalidInput
	}
	return s.repo.UpdateSlug(ctx, req)
}

// UpdatePhase rejects phases absent from the selected project configuration.
func (s *Service) UpdatePhase(ctx context.Context, req domain.ChangeUpdatePhaseRequest) error {
	req.ChangePhase = strings.TrimSpace(req.ChangePhase)
	if req.ID <= 0 || req.ChangePhase == "" {
		return app.ErrChangeInvalidInput
	}
	config, err := s.config(ctx, domain.ChangeIDRequest{ID: req.ID})
	if err != nil {
		return err
	}
	if !slices.Contains(config.ChangePhases, req.ChangePhase) {
		return app.ErrChangeInvalidReference
	}
	return s.repo.UpdatePhase(ctx, req)
}

// UpdateEpic checks same-project association; separate preflight and CALL are not atomic.
func (s *Service) UpdateEpic(ctx context.Context, req domain.ChangeUpdateEpicRequest) error {
	if req.ID <= 0 || invalidOptionalID(req.EpicID) {
		return app.ErrChangeInvalidInput
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
			return app.ErrChangeInvalidReference
		}
	}
	return s.repo.UpdateEpic(ctx, req)
}

// UpdateActive requires an explicit boolean, including false.
func (s *Service) UpdateActive(ctx context.Context, req domain.ChangeUpdateActiveRequest) error {
	if req.ID <= 0 || req.Active == nil {
		return app.ErrChangeInvalidInput
	}
	return s.repo.UpdateActive(ctx, req)
}

// UpdatePRUrl accepts nonblank HTTP(S) URLs only.
func (s *Service) UpdatePRUrl(ctx context.Context, req domain.ChangeUpdatePRUrlRequest) error {
	req.PRUrl = strings.TrimSpace(req.PRUrl)
	if req.ID <= 0 || req.PRUrl == "" {
		return app.ErrChangeInvalidInput
	}
	if err := validatePRURL(req.PRUrl); err != nil {
		return err
	}
	return s.repo.UpdatePRUrl(ctx, req)
}

// Delete leaves actual FK conflicts and affected-row semantics to one DELETE.
func (s *Service) Delete(ctx context.Context, req domain.ChangeIDRequest) error {
	if req.ID <= 0 {
		return app.ErrChangeInvalidInput
	}
	return s.repo.Delete(ctx, req)
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

func invalidOptionalID(value *int) bool {
	return value != nil && *value <= 0
}

func validatePRURL(value string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return app.ValidationError(err, app.ErrChangeInvalidInput)
	}
	if parsed.Host == "" || (!strings.EqualFold(parsed.Scheme, "https") && !strings.EqualFold(parsed.Scheme, "http")) {
		return app.ErrChangeInvalidInput
	}
	return nil
}

// UpdateAfterChange stores the nullable prerequisite; the database enforces the reference.
func (s *Service) UpdateAfterChange(ctx context.Context, req domain.ChangeUpdateAfterChangeRequest) error {
	if req.ID <= 0 || invalidOptionalID(req.AfterChangeID) {
		return app.ErrChangeInvalidInput
	}
	return s.repo.UpdateAfterChange(ctx, req)
}
