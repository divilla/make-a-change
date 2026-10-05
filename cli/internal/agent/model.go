// Package agent owns the brief rewrite and spec writing sequence.
package agent

import (
	"cli/internal/dto"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// API supplies document persistence and change creation without refresh side effects.
type API interface {
	GetProjectConfig(context.Context, int) (dto.ProjectConfig, error)
	GetChange(context.Context, int) (dto.Change, error)
	CreateChange(context.Context, dto.ChangeCreateInput) (int, error)
	UpdateChangeTypes(context.Context, int, []string) error
	ActiveDocuments(context.Context, int, string) ([]dto.Document, error)
	InsertDocument(context.Context, dto.DocumentInput) (int, error)
}

// Workspace supplies operation-owned draft files and repository prompts.
type Workspace interface {
	Prepare(string, string) (string, error)
	Read(string) (string, error)
	Stamp(string) (dto.FileStamp, error)
	Prompt(string, string, string) (string, error)
	Cleanup(string) error
}

// Runner supplies interactive handoff and streamed non-interactive execution.
type Runner interface {
	Interactive(context.Context, ...string) *exec.Cmd
	Exec(context.Context, string, string, string, chan<- string) (dto.AgentOutput, error)
}

// Request fixes the project, change and scratch file for one user save.
type Request struct {
	Root, UUID, Path, Title, Brief string
	ProjectID, ChangeID            int
	ChangeTypes                    []string
	ChangeTypesPresent             bool
}

// Result retains successful persistence even if a later operation fails.
type Result struct {
	ChangeID         int
	Path, Status     string
	Brief, Spec      *dto.Document
	Err              error
	ChangeTypes      []string
	ChangeTypesSaved bool
}

// Run executes the fixed sequence. The shell hands interactive commands to the terminal.
func Run(ctx context.Context, api API, files Workspace, runner Runner, req Request, interactive func(*exec.Cmd) error, progress chan<- string, syncCases func(context.Context, int, string) error) (result Result) {
	result.ChangeID, result.Path = req.ChangeID, req.Path
	defer func() {
		if result.Err != nil {
			result.Err = fmt.Errorf("%w; %s; draft retained at %s", result.Err, result.Status, result.Path)
		}
	}()
	if err := ctx.Err(); err != nil {
		result.Err = err
		return
	}
	cfg, err := api.GetProjectConfig(ctx, req.ProjectID)
	if err != nil {
		result.Err = err
		return
	}
	if !slices.Contains(cfg.ChangeDocs, "brief") || !slices.Contains(cfg.ChangeDocs, "spec") {
		result.Err = fmt.Errorf("project requires configured brief and spec document types")
		return
	}
	if req.ChangeID == 0 {
		for _, kind := range req.ChangeTypes {
			if !slices.Contains(cfg.ChangeTypes, kind) {
				result.Err = fmt.Errorf("type %q is not configured for this project", kind)
				return
			}
		}
		if !slices.Contains(cfg.ChangePhases, "backlog") {
			result.Err = fmt.Errorf("project requires configured backlog phase")
			return
		}
		result.ChangeID, result.Err = api.CreateChange(ctx, dto.ChangeCreateInput{ProjectID: req.ProjectID, RefUUID: req.UUID, Title: req.Title, Brief: req.Brief})
		if result.Err != nil {
			return
		}
		result.Status = fmt.Sprintf("created change #%d", result.ChangeID)
		if req.ChangeTypesPresent {
			if err := api.UpdateChangeTypes(ctx, result.ChangeID, req.ChangeTypes); err != nil {
				result.Err = fmt.Errorf("type update failed: %w", err)
				return
			}
			result.ChangeTypes, result.ChangeTypesSaved = req.ChangeTypes, true
			result.Status += "; saved types " + strings.Join(req.ChangeTypes, "|")
		}
	} else {
		change, err := api.GetChange(ctx, req.ChangeID)
		if err != nil {
			result.Err = err
			return
		}
		if change.ID != req.ChangeID || change.ProjectID != req.ProjectID {
			result.Err = fmt.Errorf("change does not belong to selected project")
			return
		}
		docs, err := api.ActiveDocuments(ctx, req.ChangeID, "change")
		if err != nil {
			result.Err = err
			return
		}
		var brief *dto.Document
		for _, doc := range docs {
			if doc.RefID != req.ChangeID || doc.RefTable != "change" {
				result.Err = fmt.Errorf("document owner mismatch")
				return
			}
			if doc.DocType == "brief" {
				if brief != nil {
					result.Err = fmt.Errorf("conflicting active briefs")
					return
				}
				savedBrief := doc
				brief = &savedBrief
			}
		}
		if brief == nil {
			result.Err = fmt.Errorf("saved brief is missing")
			return
		}
		result.Path, result.Err = files.Prepare(change.RefUUID, brief.Body)
		if result.Err != nil {
			return
		}
		result.Status = "human brief saved"
	}
	before, err := files.Stamp(result.Path)
	if err != nil || !before.Exists {
		result.Err = fmt.Errorf("read brief modification time: %v", err)
		return
	}
	prompt, err := files.Prompt(req.Root, "brief-rewrite", result.Path)
	if err != nil {
		result.Err = err
		return
	}
	if err := interactive(runner.Interactive(ctx, "-C", req.Root, prompt)); err != nil {
		result.Err = fmt.Errorf("brief rewrite: %w", err)
		return
	}
	if err := ctx.Err(); err != nil {
		result.Err = err
		return
	}
	after, err := files.Stamp(result.Path)
	if err != nil || !after.Exists {
		result.Err = fmt.Errorf("read rewritten brief modification time: %v", err)
		return
	}
	if after.Time.Equal(before.Time) {
		result.Status += "; brief unchanged"
		result.Err = files.Cleanup(result.Path)
		return
	}
	brief, err := files.Read(result.Path)
	if err != nil {
		result.Err = err
		return
	}
	result.Brief, result.Err = save(ctx, api, result.ChangeID, "brief", brief)
	if result.Err != nil {
		return
	}
	result.Status += fmt.Sprintf("; agent brief document #%d saved", result.Brief.ID)
	prompt, err = files.Prompt(req.Root, "spec-write", result.Path)
	if err != nil {
		result.Err = err
		return
	}
	output, err := runner.Exec(ctx, req.Root, result.Path, prompt, progress)
	if err != nil {
		result.Err = fmt.Errorf("spec writing: %w", err)
		return
	}
	if err := ctx.Err(); err != nil {
		result.Err = err
		return
	}
	specPath := filepath.Join(filepath.Dir(result.Path), "spec.md")
	if strings.TrimSpace(output.Final) != "Done." {
		if strings.TrimSpace(output.SessionID) == "" {
			result.Err = fmt.Errorf("spec writing session ID is missing")
			return
		}
		if err := interactive(runner.Interactive(ctx, "resume", output.SessionID)); err != nil {
			result.Err = fmt.Errorf("spec resume: %w", err)
			return
		}
		if err := ctx.Err(); err != nil {
			result.Err = err
			return
		}
	}
	stamp, err := files.Stamp(specPath)
	if err != nil {
		result.Err = err
		return
	}
	if !stamp.Exists {
		result.Err = fmt.Errorf("error generating `spec`")
		return
	}
	spec, err := files.Read(specPath)
	if err != nil {
		result.Err = err
		return
	}
	docs, err := api.ActiveDocuments(ctx, result.ChangeID, "change")
	if err != nil {
		result.Err = fmt.Errorf("read active spec: %w", err)
		return
	}
	var active *dto.Document
	for _, doc := range docs {
		if doc.RefID != result.ChangeID || doc.RefTable != "change" {
			result.Err = fmt.Errorf("document owner mismatch")
			return
		}
		if doc.DocType == "spec" {
			if active != nil {
				result.Err = fmt.Errorf("conflicting active specs")
				return
			}
			active = &doc
		}
	}
	if active != nil && strings.TrimSpace(active.Body) == strings.TrimSpace(spec) {
		result.Spec = active
		result.Status += "; spec unchanged"
	} else {
		result.Spec, result.Err = save(ctx, api, result.ChangeID, "spec", spec)
		if result.Err != nil {
			return
		}
		result.Status += fmt.Sprintf("; agent spec document #%d saved", result.Spec.ID)
	}
	result.Err = syncCases(ctx, result.ChangeID, strings.TrimSpace(result.Spec.Body))
	if result.Err != nil {
		return
	}
	result.Status += "; testcases synchronized"
	result.Err = files.Cleanup(result.Path)
	return
}

func save(ctx context.Context, api API, change int, kind, body string) (*dto.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	id, err := api.InsertDocument(ctx, dto.DocumentInput{RefID: change, RefTable: "change", DocType: kind, Body: body, AgentEdit: true})
	if err != nil {
		return nil, err
	}
	return &dto.Document{ID: id, RefID: change, RefTable: "change", DocType: kind, Body: strings.TrimSpace(body), AgentEdit: true}, nil
}
