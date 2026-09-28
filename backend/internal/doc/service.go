package doc

import (
	"context"
	"mch_api/internal/app"
	"mch_api/internal/domain"
	"slices"
	"strings"
)

// Repository supplies stored docs and their owning project.
type Repository interface {
	List(context.Context, domain.DocListRequest) ([]domain.Doc, error)
	Current(context.Context, domain.DocListRequest) ([]domain.Doc, error)
	Details(context.Context, domain.DocIDRequest) (domain.Doc, error)
	Project(context.Context, domain.DocListRequest) (domain.ProjectIDRequest, error)
	Insert(context.Context, domain.DocInsertRequest) (domain.DocIDRequest, error)
}

// ProjectConfig resolves the selected configuration.
type ProjectConfig interface {
	Config(context.Context, domain.ProjectIDRequest) (domain.Config, error)
}

// Service validates shared doc operations and sanitizes rendered content.
type Service struct {
	repo     Repository
	renderer Renderer
	projects ProjectConfig
}

// NewService injects persistence, rendering and configuration.
func NewService(repo Repository, renderer Renderer, projects ProjectConfig) *Service {
	return &Service{repo: repo, renderer: renderer, projects: projects}
}

func validRef(req domain.DocListRequest) bool {
	return req.RefID > 0 && slices.Contains([]string{"project", "epic", "change"}, req.RefTable)
}

// List returns all retained docs in descending ID order without a parent preflight.
func (s *Service) List(ctx context.Context, req domain.DocListRequest) ([]domain.Doc, error) {
	if !validRef(req) {
		return nil, app.ErrDocInvalidInput
	}
	docs, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	for i := range docs {
		docs[i].HTML = s.renderer.Render(docs[i].Body)
	}
	return docs, nil
}

// Current returns only current docs for the supplied reference.
func (s *Service) Current(ctx context.Context, req domain.DocListRequest) ([]domain.Doc, error) {
	if !validRef(req) {
		return nil, app.ErrDocInvalidInput
	}
	docs, err := s.repo.Current(ctx, req)
	if err != nil {
		return nil, err
	}
	for i := range docs {
		docs[i].HTML = s.renderer.Render(docs[i].Body)
	}
	return docs, nil
}

// Details returns one doc by its own ID, including historical docs.
func (s *Service) Details(ctx context.Context, req domain.DocIDRequest) (domain.Doc, error) {
	if req.ID <= 0 {
		return domain.Doc{}, app.ErrDocInvalidInput
	}
	d, err := s.repo.Details(ctx, req)
	if err != nil {
		return domain.Doc{}, err
	}
	d.HTML = s.renderer.Render(d.Body)
	return d, nil
}

// Insert validates the configured doc type and delegates atomic history handling to PostgreSQL.
func (s *Service) Insert(ctx context.Context, req domain.DocInsertRequest) (domain.DocIDRequest, error) {
	ref := domain.DocListRequest{RefID: req.RefID, RefTable: req.RefTable}
	req.DocType = strings.TrimSpace(req.DocType)
	req.Body = strings.TrimSpace(req.Body)
	if !validRef(ref) || req.DocType == "" || req.Body == "" || req.AgentEdit == nil {
		return domain.DocIDRequest{}, app.ErrDocInvalidInput
	}
	project, err := s.repo.Project(ctx, ref)
	if err != nil {
		return domain.DocIDRequest{}, err
	}
	config, err := s.projects.Config(ctx, project)
	if err != nil {
		return domain.DocIDRequest{}, err
	}
	kinds := config.ChangeDocs
	switch req.RefTable {
	case "project":
		kinds = config.ProjectDocs
	case "epic":
		kinds = config.EpicDocs
	}
	if !slices.Contains(kinds, req.DocType) {
		return domain.DocIDRequest{}, app.ErrDocInvalidReference
	}
	return s.repo.Insert(ctx, req)
}
