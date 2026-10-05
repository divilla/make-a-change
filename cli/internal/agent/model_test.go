package agent

import (
	"cli/internal/dto"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type flowAPI struct {
	cfg     dto.ProjectConfig
	change  dto.Change
	docs    []dto.Document
	creates []dto.ChangeCreateInput
	inserts []dto.DocumentInput
	fail    string
	calls   []string
	types   []string
	typesID int
}

func (a *flowAPI) check(name string) error {
	a.calls = append(a.calls, name)
	if a.fail == name {
		return errors.New(name + " failed")
	}
	return nil
}

func (a *flowAPI) GetProjectConfig(context.Context, int) (dto.ProjectConfig, error) {
	return a.cfg, a.check("config")
}

func (a *flowAPI) GetChange(context.Context, int) (dto.Change, error) {
	return a.change, a.check("change")
}

func (a *flowAPI) ActiveDocuments(context.Context, int, string) ([]dto.Document, error) {
	return a.docs, a.check("active")
}

func (a *flowAPI) CreateChange(_ context.Context, in dto.ChangeCreateInput) (int, error) {
	if err := a.check("create"); err != nil {
		return 0, err
	}
	a.creates = append(a.creates, in)
	return 12, nil
}

func (a *flowAPI) InsertDocument(_ context.Context, in dto.DocumentInput) (int, error) {
	if err := a.check("insert-" + in.DocType); err != nil {
		return 0, err
	}
	a.inserts = append(a.inserts, in)
	return 40 + len(a.inserts), nil
}

func (a *flowAPI) UpdateChangeTypes(_ context.Context, id int, values []string) error {
	if err := a.check("types"); err != nil {
		return err
	}
	a.typesID, a.types = id, values
	return nil
}

type flowFiles struct {
	body     map[string]string
	stamps   map[string]dto.FileStamp
	fail     string
	cleaned  bool
	prepared string
}

func (f *flowFiles) check(name string) error {
	if f.fail == name {
		return errors.New(name + " failed")
	}
	return nil
}

func (f *flowFiles) Prepare(ref, body string) (string, error) {
	if err := f.check("prepare"); err != nil {
		return "", err
	}
	f.prepared = ref
	f.body["/tmp/mch/ref/brief.md"] = body
	f.stamps["/tmp/mch/ref/brief.md"] = dto.FileStamp{Exists: true, Time: time.Unix(1, 0)}
	return "/tmp/mch/ref/brief.md", nil
}

func (f *flowFiles) Read(path string) (string, error) {
	return f.body[path], f.check("read-" + filepath.Base(path))
}

func (f *flowFiles) Stamp(path string) (dto.FileStamp, error) {
	return f.stamps[path], f.check("stamp-" + filepath.Base(path))
}

func (f *flowFiles) Prompt(root, name, path string) (string, error) {
	return root + " " + name + " " + path, f.check(name)
}
func (f *flowFiles) Cleanup(string) error { f.cleaned = true; return f.check("cleanup") }

type flowRunner struct {
	files                                               *flowFiles
	output                                              dto.AgentOutput
	err                                                 error
	execCalls                                           int
	interactive                                         [][]string
	changeBrief, touchBrief, resumeCreate, resumeModify bool
	interactiveErr                                      string
	execSpec                                            bool
}

func (r *flowRunner) Interactive(ctx context.Context, args ...string) *exec.Cmd {
	r.interactive = append(r.interactive, args)
	return exec.CommandContext(ctx, "fake", args...)
}

func (r *flowRunner) Exec(_ context.Context, _, path, _ string, _ chan<- string) (dto.AgentOutput, error) {
	r.execCalls++
	if r.execSpec {
		r.files.body[filepath.Join(filepath.Dir(path), "spec.md")] = "# Spec\n## Testcases\n- Q → R"
		r.files.stamps[filepath.Join(filepath.Dir(path), "spec.md")] = dto.FileStamp{Exists: true, Time: time.Unix(2, 0)}
	}
	return r.output, r.err
}

func (r *flowRunner) run(cmd *exec.Cmd) error {
	if cmd.Args[1] == "resume" {
		if r.resumeCreate || r.resumeModify {
			r.files.body["/tmp/mch/ref/spec.md"] = "# Resumed\n## Testcases\n- Q → R"
			r.files.stamps["/tmp/mch/ref/spec.md"] = dto.FileStamp{Exists: true, Time: time.Unix(3, 0)}
		}
		if r.interactiveErr == "resume" {
			return errors.New("resume failed")
		}
	} else {
		if r.changeBrief {
			r.files.body["/tmp/mch/ref/brief.md"] = "# Rewrite"
		}
		if r.changeBrief || r.touchBrief {
			r.files.stamps["/tmp/mch/ref/brief.md"] = dto.FileStamp{Exists: true, Time: time.Unix(2, 0)}
		}
		if r.interactiveErr == "rewrite" {
			return context.Canceled
		}
	}
	return nil
}

func flowFixture() (*flowAPI, *flowFiles, *flowRunner, Request) {
	a := &flowAPI{cfg: dto.ProjectConfig{ChangeDocs: []string{"brief", "spec"}, ChangePhases: []string{"backlog"}}, change: dto.Change{ID: 12, ProjectID: 7, RefUUID: "ref"}, docs: []dto.Document{{ID: 31, RefID: 12, RefTable: "change", DocType: "brief", Body: "# Saved brief"}}}
	f := &flowFiles{body: map[string]string{"/tmp/mch/ref/brief.md": "# User"}, stamps: map[string]dto.FileStamp{"/tmp/mch/ref/brief.md": {Exists: true, Time: time.Unix(1, 0)}}}
	r := &flowRunner{files: f, output: dto.AgentOutput{Final: "Done.", SessionID: "run-session"}, changeBrief: true, execSpec: true}
	req := Request{Root: "/repo", ProjectID: 7, Path: "/tmp/mch/ref/brief.md", Title: "User", Brief: "# User", UUID: "ref"}
	return a, f, r, req
}

func Test032NewAndExistingSequenceAndProvenance(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			a, f, r, req := flowFixture()
			if existing {
				req.ChangeID = 12
			}
			synced := 0
			result := Run(context.Background(), a, f, r, req, r.run, nil, func(_ context.Context, id int, spec string) error {
				require.Equal(t, 12, id)
				require.Equal(t, "# Spec\n## Testcases\n- Q → R", spec)
				require.Len(t, a.inserts, 2)
				synced++
				return nil
			})
			require.NoError(t, result.Err)
			require.Equal(t, 1, synced)
			require.True(t, f.cleaned)
			require.Equal(t, 12, result.ChangeID)
			require.Equal(t, [][]string{{"-C", "/repo", "/repo brief-rewrite /tmp/mch/ref/brief.md"}}, r.interactive)
			require.Equal(t, []dto.DocumentInput{{RefID: 12, RefTable: "change", DocType: "brief", Body: "# Rewrite", AgentEdit: true}, {RefID: 12, RefTable: "change", DocType: "spec", Body: "# Spec\n## Testcases\n- Q → R", AgentEdit: true}}, a.inserts)
			if existing {
				require.Empty(t, a.creates)
				require.Equal(t, "ref", f.prepared)
			} else {
				require.Equal(t, []dto.ChangeCreateInput{{ProjectID: 7, RefUUID: "ref", Title: "User", Brief: "# User"}}, a.creates)
			}
			require.Contains(t, result.Status, "testcases synchronized")
		})
	}
}

func Test032ModificationTimeControlsRewrite(t *testing.T) {
	for _, touch := range []bool{false, true} {
		t.Run(fmt.Sprint(touch), func(t *testing.T) {
			a, f, r, req := flowFixture()
			r.changeBrief = false
			r.touchBrief = touch
			result := Run(context.Background(), a, f, r, req, r.run, nil, func(context.Context, int, string) error { return nil })
			require.NoError(t, result.Err)
			if touch {
				require.Len(t, a.inserts, 2)
				require.Equal(t, "# User", a.inserts[0].Body)
				require.Equal(t, 1, r.execCalls)
			} else {
				require.Empty(t, a.inserts)
				require.Zero(t, r.execCalls)
				require.Contains(t, result.Status, "brief unchanged")
			}
		})
	}
}

func Test032ResumeUsesThisSessionAndSpecFile(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		prior, create, modify bool
		want                  string
	}{
		{name: "created", create: true}, {name: "modified", prior: true, modify: true}, {name: "missing", want: "error generating `spec`"}, {name: "unchanged file time", prior: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, f, r, req := flowFixture()
			r.output.Final = "Question?"
			r.execSpec = tc.prior
			r.resumeCreate = tc.create
			r.resumeModify = tc.modify
			syncCalls := 0
			result := Run(context.Background(), a, f, r, req, r.run, nil, func(context.Context, int, string) error { syncCalls++; return nil })
			require.Equal(t, []string{"resume", "run-session"}, r.interactive[1])
			if tc.want != "" {
				require.ErrorContains(t, result.Err, tc.want)
				require.Len(t, a.inserts, 1)
				require.Zero(t, syncCalls)
				require.False(t, f.cleaned)
			} else {
				require.NoError(t, result.Err)
				require.Len(t, a.inserts, 2)
				require.Equal(t, f.body["/tmp/mch/ref/spec.md"], a.inserts[1].Body)
				require.Equal(t, 1, syncCalls)
			}
		})
	}
}

func TestSpecComparedWithActiveDocumentAfterAgentExit(t *testing.T) {
	for _, resumed := range []bool{false, true} {
		for _, state := range []string{"missing", "different", "identical", "outer whitespace", "read failure", "wrong owner", "duplicate"} {
			t.Run(fmt.Sprintf("resumed=%t/%s", resumed, state), func(t *testing.T) {
				a, f, r, req := flowFixture()
				req.ChangeID = 12
				body := "# Spec\n## Testcases\n- Q → R"
				if resumed {
					r.output.Final = "Please confirm."
				}
				interactive := func(cmd *exec.Cmd) error {
					if err := r.run(cmd); err != nil {
						return err
					}
					// Change the API result after preflight to require a fresh read.
					doc := dto.Document{ID: 99, RefID: 12, RefTable: "change", DocType: "spec", Body: body}
					switch state {
					case "missing":
						return nil
					case "different":
						doc.Body = "# Old spec"
					case "outer whitespace":
						doc.Body = "\n " + body + "\n\t"
					case "read failure":
						a.fail = "active"
						return nil
					case "wrong owner":
						doc.RefID = 13
					}
					a.docs = []dto.Document{doc}
					if state == "duplicate" {
						a.docs = append(a.docs, doc)
					}
					return nil
				}
				syncCalls := 0
				result := Run(context.Background(), a, f, r, req, interactive, nil, func(_ context.Context, id int, spec string) error {
					require.Equal(t, 12, id)
					require.Equal(t, body, spec)
					syncCalls++
					return nil
				})
				switch state {
				case "read failure", "wrong owner", "duplicate":
					require.Error(t, result.Err)
					require.Len(t, a.inserts, 1)
					require.Zero(t, syncCalls)
					require.False(t, f.cleaned)
				default:
					require.NoError(t, result.Err)
					require.Equal(t, 1, syncCalls)
					require.True(t, f.cleaned)
					if state == "identical" || state == "outer whitespace" {
						require.Len(t, a.inserts, 1)
						require.Equal(t, 99, result.Spec.ID)
						require.Contains(t, result.Status, "spec unchanged")
					} else {
						require.Len(t, a.inserts, 2)
						require.Equal(t, body, a.inserts[1].Body)
					}
				}
			})
		}
	}
}

func Test032FailuresStopDependentEffectsAndKeepDrafts(t *testing.T) {
	for _, tc := range []struct {
		name, api, files, interactive          string
		execErr                                bool
		final, id                              string
		wantInserts, wantExec, wantInteractive int
	}{
		{name: "config", api: "config"},
		{name: "create", api: "create"},
		{name: "brief stamp", files: "stamp-brief.md"},
		{name: "rewrite prompt", files: "brief-rewrite"},
		{name: "cancel rewrite", interactive: "rewrite", wantInteractive: 1},
		{name: "read rewritten brief", files: "read-brief.md", wantInteractive: 1},
		{name: "insert brief", api: "insert-brief", wantInteractive: 1},
		{name: "spec prompt", files: "spec-write", wantInserts: 1, wantInteractive: 1},
		{name: "exec error with session", execErr: true, wantInserts: 1, wantExec: 1, wantInteractive: 1},
		{name: "missing session", final: "Question", id: "", wantInserts: 1, wantExec: 1, wantInteractive: 1},
		{name: "spec time", files: "stamp-spec.md", final: "Question", id: "session", wantInserts: 1, wantExec: 1, wantInteractive: 2},
		{name: "resume failure", interactive: "resume", final: "Question", id: "session", wantInserts: 1, wantExec: 1, wantInteractive: 2},
		{name: "read spec", files: "read-spec.md", wantInserts: 1, wantExec: 1, wantInteractive: 1},
		{name: "insert spec", api: "insert-spec", wantInserts: 1, wantExec: 1, wantInteractive: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, f, r, req := flowFixture()
			a.fail = tc.api
			f.fail = tc.files
			r.interactiveErr = tc.interactive
			if tc.execErr {
				r.err = errors.New("exec failed")
			}
			if tc.final != "" {
				r.output = dto.AgentOutput{Final: tc.final, SessionID: tc.id}
			}
			syncCalls := 0
			result := Run(context.Background(), a, f, r, req, r.run, nil, func(context.Context, int, string) error { syncCalls++; return nil })
			require.Error(t, result.Err)
			require.False(t, f.cleaned)
			require.Len(t, a.inserts, tc.wantInserts)
			require.Equal(t, tc.wantExec, r.execCalls)
			require.Len(t, r.interactive, tc.wantInteractive)
			require.Zero(t, syncCalls)
			if len(a.creates) > 0 {
				require.Equal(t, 12, result.ChangeID)
				require.Contains(t, result.Err.Error(), "created change #12")
			}
			require.Contains(t, result.Err.Error(), "draft retained")
		})
	}
}

func Test032ExistingPreflightAndUnsupportedCatalogs(t *testing.T) {
	for _, tc := range []string{"brief missing", "spec missing", "backlog missing", "change read", "active read", "owner", "doc owner", "no brief", "duplicate", "prepare"} {
		t.Run(tc, func(t *testing.T) {
			a, f, r, req := flowFixture()
			req.ChangeID = 12
			switch tc {
			case "brief missing":
				a.cfg.ChangeDocs = []string{"spec"}
			case "spec missing":
				a.cfg.ChangeDocs = []string{"brief"}
			case "backlog missing":
				req.ChangeID = 0
				a.cfg.ChangePhases = nil
			case "change read":
				a.fail = "change"
			case "active read":
				a.fail = "active"
			case "owner":
				a.change.ProjectID = 8
			case "doc owner":
				a.docs[0].RefID = 99
			case "no brief":
				a.docs = nil
			case "duplicate":
				a.docs = append(a.docs, a.docs[0])
			case "prepare":
				f.fail = "prepare"
			}
			result := Run(context.Background(), a, f, r, req, r.run, nil, func(context.Context, int, string) error { return nil })
			require.Error(t, result.Err)
			require.Empty(t, a.creates)
			require.Empty(t, a.inserts)
			require.Empty(t, r.interactive)
		})
	}
}

func Test032SyncFailureKeepsSavedSpecAndStopsCleanup(t *testing.T) {
	a, f, r, req := flowFixture()
	result := Run(context.Background(), a, f, r, req, r.run, nil, func(context.Context, int, string) error { return errors.New("partial persistence") })
	require.ErrorContains(t, result.Err, "partial persistence")
	require.NotNil(t, result.Spec)
	require.Equal(t, 42, result.Spec.ID)
	require.Len(t, a.inserts, 2)
	require.False(t, f.cleaned)
	require.Contains(t, result.Status, "agent spec document #42 saved")
	require.False(t, strings.Contains(result.Status, "synchronized"))
}

func Test032CanceledWorkflowCannotPersistAfterSuccessfulProcessExit(t *testing.T) {
	a, f, r, req := flowFixture()
	ctx, cancel := context.WithCancel(context.Background())
	result := Run(ctx, a, f, r, req, func(cmd *exec.Cmd) error { err := r.run(cmd); cancel(); return err }, nil, func(context.Context, int, string) error { return nil })
	require.ErrorIs(t, result.Err, context.Canceled)
	require.Len(t, a.creates, 1)
	require.Empty(t, a.inserts)
	require.Zero(t, r.execCalls)
	require.False(t, f.cleaned)
	a, f, r, req = flowFixture()
	result = Run(ctx, a, f, r, req, r.run, nil, func(context.Context, int, string) error { return nil })
	require.ErrorIs(t, result.Err, context.Canceled)
	require.Empty(t, a.calls)
}

func Test032CreationTypesValidateAndPersistBeforeRewrite(t *testing.T) {
	for _, tc := range []struct {
		name    string
		present bool
		types   []string
		fail    string
		want    string
	}{
		{name: "absent"},
		{name: "supported", present: true, types: []string{"feature", "fix"}},
		{name: "explicit empty", present: true, types: []string{}},
		{name: "unsupported", present: true, types: []string{"feature", "unknown"}, want: `type "unknown" is not configured`},
		{name: "update failure", present: true, types: []string{"feature"}, fail: "types", want: "type update failed"},
		{name: "later failure", present: true, types: []string{"feature"}, fail: "insert-brief", want: "insert-brief failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, f, r, req := flowFixture()
			a.cfg.ChangeTypes = []string{"feature", "fix"}
			a.fail = tc.fail
			req.ChangeTypes, req.ChangeTypesPresent = tc.types, tc.present
			result := Run(context.Background(), a, f, r, req, func(cmd *exec.Cmd) error {
				if tc.present {
					require.Equal(t, 12, a.typesID)
					require.Equal(t, tc.types, a.types)
					require.Equal(t, []string{"config", "create", "types"}, a.calls)
				}
				return r.run(cmd)
			}, nil, func(context.Context, int, string) error { return nil })
			if tc.want == "" {
				require.NoError(t, result.Err)
			} else {
				require.ErrorContains(t, result.Err, tc.want)
				require.False(t, f.cleaned)
			}
			switch tc.name {
			case "unsupported":
				require.Empty(t, a.creates)
				require.Equal(t, []string{"config"}, a.calls)
				require.Empty(t, r.interactive)
			case "update failure":
				require.Equal(t, 12, result.ChangeID)
				require.Contains(t, result.Err.Error(), "created change #12")
				require.Empty(t, a.inserts)
				require.Empty(t, r.interactive)
				require.False(t, result.ChangeTypesSaved)
			default:
				require.Equal(t, tc.present, result.ChangeTypesSaved)
				require.Equal(t, tc.types, result.ChangeTypes)
			}
		})
	}
}
