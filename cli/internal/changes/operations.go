package changes

import (
	"cli/internal/dto"
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// Operation identifies a read or a single change-field mutation.
type Operation string

// Supported operations; Retry always schedules List or Details.
const (
	List        Operation = "list"
	Details     Operation = "details"
	Create      Operation = "create"
	Delete      Operation = "delete"
	Title       Operation = "title"
	Slug        Operation = "slug"
	Phase       Operation = "phase"
	Types       Operation = "types"
	Epic        Operation = "epic"
	AfterChange Operation = "after-change"
	Open        Operation = "open"
	PRURL       Operation = "pr-url"
	Document    Operation = "document"
)

// Input is feature-owned form data, separate from the backend DTO.
type Input struct {
	Value        string
	Types        []string
	Association  *int
	Open         bool
	Title        string
	UUID         string
	DocumentType string
}

// Result records every committed step before a dependent effect is attempted.
type Result struct {
	Generation      uint64
	ProjectID, ID   int
	Operation       Operation
	Detail          dto.ChangeView
	Rows            []dto.ChangeView
	Err, RefreshErr error
	Steps           []string
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// Begin validates and captures one operation. Commands never mutate a model.
func (m Model) Begin(ctx context.Context, api API, docs Documents, op Operation, project, id int, in Input, catalog dto.ProjectConfig) (Model, tea.Cmd) {
	if m.Busy {
		return m, nil
	}
	if m.ProjectID == 0 {
		m.ProjectID = project
	} else if m.ProjectID != project {
		m = m.Scope(project)
	}
	if op != m.refreshOp || id != m.refreshID {
		m.Outcome = ""
	}
	if err := validate(op, project, id, in, catalog); err != nil {
		m.Err = err
		m.Status = "validation failed"
		return m, nil
	}
	if unchanged(m.Detail, op, in) {
		m.Err = nil
		m.Status = "unchanged"
		return m, nil
	}
	outcome := m.Outcome
	m = m.Invalidate()
	m.Outcome = outcome
	m.Operation = op
	m.EntityID = id
	m.Err = nil
	m.Loading = true
	m.Status = "loading change"
	if op == Details {
		m.DetailLoaded = false
	}
	if op == List {
		m.Rows = nil
		m.Selected = 0
		m.Offset = 0
		m.Status = "loading changes"
	}
	if op != List && op != Details {
		m.Busy = true
		m.Outcome = ""
		m.Status = "saving"
	}
	if op == Delete {
		m.Status = "deleting change"
	}
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	generation := m.Generation
	prior := m.Detail
	prior.Documents = append([]dto.Document(nil), m.Detail.Documents...)
	priorRows := append([]dto.ChangeView(nil), m.Rows...)
	return m, func() tea.Msg {
		defer cancel()
		r := Result{Generation: generation, ProjectID: project, ID: id, Operation: op, Detail: prior}
		switch op {
		case List:
			r.Rows, r.Err = readRows(ctx, api, project)
			return r
		case Details:
			r.Detail, r.Err = readDetail(ctx, api, docs, project, id)
			return r
		case Create:
			r.ID, r.Err = api.CreateChange(ctx, dto.ChangeCreateInput{ProjectID: project, Title: in.Title, Brief: in.Value, RefUUID: in.UUID})
			if r.Err == nil {
				r.Detail = dto.ChangeView{ID: strconv.Itoa(r.ID), ProjectID: strconv.Itoa(project), Title: in.Title, Brief: in.Value}
				r.Steps = append(r.Steps, fmt.Sprintf("created change #%d", r.ID))
				types, present := ParseArtifactTypes(in.Value)
				if present {
					if err := api.UpdateChangeTypes(ctx, r.ID, types); err != nil {
						r.RefreshErr = fmt.Errorf("type update failed: %w", err)
						return r
					}
					r.Detail.ChangeTypes = types
					r.Steps = append(r.Steps, "saved types "+strings.Join(types, "|"))
				}
			}
		case Delete:
			r.Err = api.DeleteChange(ctx, id)
		case Title:
			r.Err = api.UpdateChangeTitle(ctx, id, in.Value)
			r.Detail.Title = in.Value
		case Slug:
			r.Err = api.UpdateChangeSlug(ctx, id, in.Value)
			if prefix, _, ok := strings.Cut(r.Detail.RefSlug, "-"); ok {
				r.Detail.RefSlug = prefix + "-" + in.Value
			}
		case Phase:
			r.Err = api.UpdateChangePhase(ctx, id, in.Value)
			r.Detail.ChangePhase = in.Value
		case Types:
			r.Err = api.UpdateChangeTypes(ctx, id, in.Types)
			r.Detail.ChangeTypes = in.Types
		case Epic:
			r.Err = api.UpdateChangeEpic(ctx, id, in.Association)
			r.Detail.EpicID = optionalInt(in.Association)
			r.Detail.EpicName = ""
		case AfterChange:
			r.Err = api.UpdateChangeAfterChange(ctx, id, in.Association)
			r.Detail.AfterChangeID = optionalInt(in.Association)
			r.Detail.AfterChangeName = "null"
		case Open:
			r.Err = api.UpdateChangeOpen(ctx, id, in.Open)
			r.Detail.Open = in.Open
		case PRURL:
			r.Err = api.UpdateChangePRUrl(ctx, id, in.Value)
			r.Detail.PRUrl = in.Value
		case Document:
			var docID int
			docID, r.Err = docs.Save(ctx, id, in.DocumentType, in.Value)
			if r.Err == nil {
				r.Steps = append(r.Steps, fmt.Sprintf("saved %s document #%d", in.DocumentType, docID))
				found := false
				for i := range r.Detail.Documents {
					if r.Detail.Documents[i].DocType == in.DocumentType {
						r.Detail.Documents[i].Body = in.Value
						r.Detail.Documents[i].ID = docID
						found = true
					}
				}
				if !found {
					r.Detail.Documents = append(r.Detail.Documents, dto.Document{ID: docID, RefID: id, RefTable: "change", DocType: in.DocumentType, Body: in.Value, Current: true})
				}
				switch in.DocumentType {
				case "brief":
					r.Detail.Brief = in.Value
				case "spec":
					r.Detail.Spec = in.Value
				case "pr":
					r.Detail.PR = in.Value
				}
				// Optional existing artifact metadata remains a distinct, recorded write.
				types, present := ParseArtifactTypes(in.Value)
				if present {
					if err := api.UpdateChangeTypes(ctx, id, types); err != nil {
						r.RefreshErr = fmt.Errorf("type update failed: %w", err)
						return r
					}
					r.Detail.ChangeTypes = types
					r.Steps = append(r.Steps, "saved types "+strings.Join(types, "|"))
				}
			}
		}
		if r.Err != nil {
			return r
		}
		if len(r.Steps) == 0 {
			r.Steps = append(r.Steps, completedStep(op, id, in))
		}
		if op == Delete {
			r.Rows, r.RefreshErr = readRows(ctx, api, project)
			if r.RefreshErr != nil {
				for _, v := range priorRows {
					if v.ID != strconv.Itoa(id) {
						r.Rows = append(r.Rows, v)
					}
				}
			}
			return r
		}
		refreshed, err := readDetail(ctx, api, docs, project, r.ID)
		r.RefreshErr = err
		if err == nil {
			r.Detail = refreshed
		}
		return r
	}
}

func completedStep(op Operation, id int, in Input) string {
	if op == Delete {
		return fmt.Sprintf("deleted change #%d", id)
	}
	value := in.Value
	switch op {
	case Types:
		value = fmt.Sprint(in.Types)
	case Epic, AfterChange:
		value = optionalInt(in.Association)
	case Open:
		value = strconv.FormatBool(in.Open)
	}
	// Bound feedback after escaping control characters; Detail retains the full value.
	quoted := ansi.Truncate(strconv.Quote(value), 42, "…\"")
	return fmt.Sprintf("saved %s %s on change #%d", op, quoted, id)
}

func validate(op Operation, project, id int, in Input, catalog dto.ProjectConfig) error {
	if project <= 0 {
		return errors.New("select a valid project first")
	}
	if op != List && op != Create && id <= 0 {
		return errors.New("change ID must be a valid positive number")
	}
	switch op {
	case Create:
		if strings.TrimSpace(in.Title) == "" {
			return errors.New("change title is required")
		}
		if strings.TrimSpace(in.Value) == "" {
			return errors.New("brief is required")
		}
		if in.UUID != "" && !uuidPattern.MatchString(in.UUID) {
			return errors.New("reference UUID must be a UUID or omitted")
		}
		if !slices.Contains(catalog.ChangePhases, "backlog") || !slices.Contains(catalog.ChangeDocs, "brief") {
			return errors.New("create requires configured backlog phase and brief document type; reload /project-config")
		}
	case Title:
		if strings.TrimSpace(in.Value) == "" {
			return errors.New("change title is required")
		}
	case Slug:
		if !regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(in.Value) {
			return errors.New("change slug requires a lowercase suffix")
		}
	case Phase:
		if !slices.Contains(catalog.ChangePhases, in.Value) {
			return errors.New("phase is not configured for this project")
		}
	case Types:
		for _, v := range in.Types {
			if !slices.Contains(catalog.ChangeTypes, v) {
				return fmt.Errorf("type %q is not configured for this project", v)
			}
		}
	case Epic, AfterChange:
		if in.Association != nil && *in.Association <= 0 {
			return errors.New("association ID must be positive or null")
		}
	case PRURL:
		u, err := url.Parse(strings.TrimSpace(in.Value))
		if err != nil || u.Host == "" || (!strings.EqualFold(u.Scheme, "http") && !strings.EqualFold(u.Scheme, "https")) {
			return errors.New("PR URL requires a nonblank HTTP(S) URL; clearing is unsupported")
		}
	case Document:
		if !slices.Contains(catalog.ChangeDocs, in.DocumentType) {
			return fmt.Errorf("change document type %q is not configured for this project", in.DocumentType)
		}
	case List, Details, Delete, Open:
	default:
		return errors.New("unsupported change operation")
	}
	return nil
}

func unchanged(v dto.ChangeView, op Operation, in Input) bool {
	if v.ID == "" {
		return false
	}
	switch op {
	case Title:
		return v.Title == in.Value
	case Slug:
		_, suffix, ok := strings.Cut(v.RefSlug, "-")
		return ok && suffix == in.Value
	case PRURL:
		return v.PRUrl == in.Value
	case AfterChange:
		return v.AfterChangeID == optionalInt(in.Association)
	case Document:
		switch in.DocumentType {
		case "brief":
			return v.Brief == in.Value
		case "spec":
			return v.Spec == in.Value
		case "pr":
			return v.PR == in.Value
		default:
			for _, d := range v.Documents {
				if d.DocType == in.DocumentType {
					return d.Body == in.Value
				}
			}
		}
	}
	return false
}

// Invalidate cancels obsolete reads and discards outcome attribution on navigation.
func (m Model) Invalidate() Model {
	if m.cancel != nil {
		m.cancel()
	}
	m.cancel = nil
	m.Generation++
	m.Loading = false
	m.Outcome = ""
	return m
}

// Scope clears state from the previous project.
func (m Model) Scope(project int) Model {
	m = m.Invalidate()
	return Model{ProjectID: project, Generation: m.Generation}
}

// Apply rejects stale project/entity/operation/revision results.
func (m Model) Apply(r Result) (Model, bool) {
	if r.Generation != m.Generation || r.ProjectID != m.ProjectID || r.Operation != m.Operation || (r.Operation != Create && r.ID != m.EntityID) {
		return m, false
	}
	m.Generation++
	m.cancel = nil
	m.Busy = false
	m.Loading = false
	m.Err = r.Err
	if r.Err != nil {
		m.Status = "load failed"
		if r.Operation != List && r.Operation != Details {
			m.Status = "save failed"
		}
		if m.Outcome != "" {
			m.Status = m.Outcome + "; refresh failed — /retry reads only"
		}
		return m, true
	}
	switch r.Operation {
	case List:
		m = m.WithRows(r.Rows)
		m.Status = "changes loaded"
		if len(r.Rows) == 0 {
			m.Status = "no changes"
		}
	case Delete:
		m = m.WithRows(r.Rows)
		m.Detail = dto.ChangeView{}
		m.DetailLoaded = false
	default:
		m = m.WithDetail(r.Detail)
		m.DetailLoaded = r.RefreshErr == nil
		m.Status = "loaded change"
	}
	m.Outcome = ""
	if len(r.Steps) > 0 {
		m.Status = strings.Join(r.Steps, "; ")
	}
	if r.RefreshErr != nil {
		m.Err = r.RefreshErr
		m.Outcome = m.Status
		m.refreshOp, m.refreshID = Details, r.ID
		if r.Operation == Delete {
			m.refreshOp, m.refreshID = List, 0
		}
		m.Status += "; remaining step failed — /retry reads only"
	}
	return m, true
}

func readRows(ctx context.Context, api API, project int) ([]dto.ChangeView, error) {
	rows, err := api.ListChangeRows(ctx, project)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ChangeView, 0, len(rows))
	for _, r := range rows {
		out = append(out, Present(r))
	}
	return out, nil
}

func readDetail(ctx context.Context, api API, docs Documents, project, id int) (dto.ChangeView, error) {
	c, err := api.GetChange(ctx, id)
	if err != nil {
		return dto.ChangeView{}, err
	}
	if c.ID != id || c.ProjectID != project {
		return dto.ChangeView{}, errors.New("change does not belong to selected project/entity")
	}
	view := Present(c)
	rows, err := docs.Load(ctx, id)
	if err != nil {
		return view, err
	}
	view.Documents = rows
	for _, d := range rows {
		switch d.DocType {
		case "brief":
			view.Brief = d.Body
		case "spec":
			view.Spec = d.Body
		case "pr":
			view.PR = d.Body
		}
	}
	view.TestCases, err = api.ListTestCases(ctx, id)
	return view, err
}

// Present formats persisted identity and server completion without deriving either.
func Present(c dto.Change) dto.ChangeView {
	v := dto.ChangeView{ID: strconv.Itoa(c.ID), ProjectID: strconv.Itoa(c.ProjectID), RefUUID: c.RefUUID, Ref: "null", RefSlug: "null", EpicID: optionalInt(c.EpicID), EpicName: "null", AfterChangeID: optionalInt(c.AfterChangeID), AfterChangeName: "null", Title: c.Title, ChangePhase: c.ChangePhase, ChangeTypes: append([]string(nil), c.ChangeTypes...), Open: c.Open, Done: c.DoneTC, Total: c.TotalTC, Completed: c.Completed, PRUrl: c.PRUrl, Created: c.CreatedAt.Format(time.RFC3339Nano), Modified: c.UpdatedAt.Format(time.RFC3339Nano)}
	if c.RefSlug != nil {
		v.RefSlug = *c.RefSlug
		if ref, _, ok := strings.Cut(*c.RefSlug, "-"); ok {
			v.Ref = ref
		}
	}
	if c.EpicName != nil {
		v.EpicName = *c.EpicName
	}
	if c.AfterChangeName != nil {
		v.AfterChangeName = *c.AfterChangeName
	}
	return v
}

func optionalInt(v *int) string {
	if v == nil {
		return "null"
	}
	return strconv.Itoa(*v)
}

// PrepareCreate keeps explicit form values and optionally extracts a Markdown title.
func (m Model) PrepareCreate(brief string) Model {
	m.Draft.Value = brief
	if m.Draft.Title == "" {
		if p, err := ParseBriefStructure(brief); err == nil {
			m.Draft.Title = p.Title
		}
	}
	return m
}

// AssociationInput validates a nullable association entered in a text form.
func AssociationInput(value string) (*int, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "null" {
		return nil, nil
	}
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return nil, errors.New("prerequisite must be a positive change ID or null")
	}
	return &id, nil
}
