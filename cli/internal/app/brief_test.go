package app

import (
	"cli/internal/agent"
	"cli/internal/changes"
	"cli/internal/dto"
	"cli/internal/testcases"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func Test032NewChangeEditorUsesEmptyUUIDPathAndInvalidBriefStaysOnList(t *testing.T) {
	for _, brief := range []string{"", "\n ", "## Wrong level", "# ", "text\n# Title"} {
		t.Run(brief, func(t *testing.T) {
			t.Setenv("TMPDIR", t.TempDir())
			client := &fakeClient{}
			m := newChangeTestModel(client)
			next, cmd := m.newChangeBrief()
			require.NotNil(t, cmd)
			m = next.(Model)
			require.Equal(t, ChangesListState, m.state)
			require.NotNil(t, m.agentOperation)
			path := m.agentOperation.path
			require.Equal(t, filepath.Join(os.TempDir(), "mch", m.agentOperation.uuid, "brief.md"), path)
			require.Regexp(t, `^[0-9a-f-]{36}$`, m.agentOperation.uuid)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Empty(t, data)
			require.NoError(t, os.WriteFile(path, []byte(brief), 0o600))
			next, cmd = m.Update(newBriefEditedMsg{operation: m.agentOperation})
			m = next.(Model)
			require.Nil(t, cmd)
			require.Equal(t, ChangesListState, m.state)
			require.Contains(t, m.err, "brief title is required")
			require.Empty(t, client.changeCreateInputs)
			require.Nil(t, m.agentOperation)
			require.Equal(t, brief, readTestFile(t, path))
		})
	}
}

type partialSyncClient struct {
	*fakeClient
	cancel context.CancelFunc
}

func (a *partialSyncClient) DeleteTestCase(_ context.Context, id int) error {
	a.testCaseDeleteIDs = append(a.testCaseDeleteIDs, id)
	a.gotChange.TestCases = nil
	a.gotChange.Done, a.gotChange.Total, a.gotChange.Completed = 0, 0, 0
	if a.cancel != nil {
		a.cancel()
	}
	return nil
}

func Test032PartialSynchronizationInvalidatesManualAndAgentCachesUntilReadSucceeds(t *testing.T) {
	for _, source := range []string{"manual", "agent"} {
		for _, cancelled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/cancelled=%t", source, cancelled), func(t *testing.T) {
				spec := "# Saved spec\n## Testcases\n- New action → new result"
				old := dto.TestCase{ID: 31, ChangeID: 12, Scenario: "Old action → old result", Done: true}
				a := &partialSyncClient{fakeClient: &fakeClient{changeUpdateErr: errors.New("create failed"), gotChange: dto.ChangeView{ID: "12", ProjectID: "7", Spec: spec, TestCases: []dto.TestCase{old}, Done: 1, Total: 1, Completed: 100}}}
				m := testcaseScreen(a.fakeClient)
				m.client = a
				m.changeList.Detail.Spec = spec
				m.changeList.Detail.Documents = []dto.Document{{ID: 91, DocType: "spec", Body: spec}}
				m = selectTestcase(m, "31")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if cancelled {
					a.cancel = cancel
				}
				var msg tea.Msg
				if source == "manual" {
					m.ctx = ctx
					next, cmd := m.syncSavedSpec(12, spec)
					m = next.(Model)
					msg = cmd()
				} else {
					err := testcases.Synchronize(ctx, a, 12, spec)
					m.agentOperation = &agentOperation{ctx: ctx, cancel: cancel, projectID: "7"}
					msg = agentFinishedMsg{operation: m.agentOperation, result: agent.Result{ChangeID: 12, Spec: &dto.Document{ID: 91, DocType: "spec", Body: spec, AgentEdit: true}, Status: "spec saved", Err: fmt.Errorf("draft retained: %w", err)}}
				}
				next, cmd := m.Update(msg)
				m = next.(Model)
				require.Nil(t, cmd)
				require.Equal(t, []int{31}, a.testCaseDeleteIDs)
				require.False(t, m.changeDetailLoaded)
				require.False(t, m.changeList.DetailLoaded)
				require.False(t, m.testCase.Loaded)
				require.Equal(t, spec, m.changeList.Detail.Spec)
				require.Contains(t, m.err, "1 successful mutations (partial persistence)")
				require.Contains(t, m.err, "/retry")
				if source == "agent" {
					require.Equal(t, 91, m.changeList.Detail.Documents[0].ID)
					require.True(t, m.changeList.Detail.Documents[0].AgentEdit)
				}
				for _, key := range []tea.KeyType{tea.KeyEnter, tea.KeySpace, tea.KeyDelete} {
					next, cmd = m.Update(tea.KeyMsg{Type: key})
					require.Nil(t, cmd)
					require.Equal(t, ChangeDetailsState, next.(Model).state)
					require.Contains(t, next.(Model).err, "/retry")
				}
				// Refresh has its own context, and never repeats synchronization writes.
				m.ctx = context.Background()
				a.changeGetErr = errors.New("read unavailable")
				next, cmd = m.executeCommandFrom(ChangeDetailsState, "/retry")
				m = applyMsg(next.(Model), cmd())
				require.False(t, m.changeDetailLoaded)
				require.False(t, m.testCase.Loaded)
				a.changeGetErr = nil
				next, cmd = m.executeCommandFrom(ChangeDetailsState, "/retry")
				m = applyMsg(next.(Model), cmd())
				require.True(t, m.changeDetailLoaded)
				require.True(t, m.testCase.Loaded)
				require.Empty(t, m.changeList.Detail.TestCases)
				require.Empty(t, m.testCase.Rows)
				require.Zero(t, m.changeList.Detail.Total)
				require.Equal(t, spec, m.changeList.Detail.Spec)
				require.Equal(t, []int{31}, a.testCaseDeleteIDs)
				require.LessOrEqual(t, a.testCaseCreateCalls, 1)
				require.Zero(t, a.testCaseUpdateCalls)
				require.Zero(t, a.testCaseDoneCalls)
			})
		}
	}
}

func Test032TitleOnlyAndLeadingBlankBriefStartsImmediately(t *testing.T) {
	for _, brief := range []string{"# Add DB migrations tool to the project", "\n\n# Valid title", "# Title\n\nBody\twith data"} {
		t.Run(brief, func(t *testing.T) {
			t.Setenv("TMPDIR", t.TempDir())
			m := newChangeTestModel(&fakeClient{})
			next, _ := m.newChangeBrief()
			m = next.(Model)
			op := m.agentOperation
			require.NoError(t, os.WriteFile(op.path, []byte(brief), 0o600))
			next, cmd := m.applyNewBrief(newBriefEditedMsg{operation: op})
			m = next.(Model)
			require.NotNil(t, cmd)
			require.Equal(t, AgentExecState, m.state)
			require.Same(t, op, m.agentOperation)
			require.Equal(t, brief, readTestFile(t, op.path))
			op.cancel()
		})
	}
}

func Test032EditorErrorKeepsLocalDraftAndStops(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	m := newChangeTestModel(&fakeClient{})
	next, _ := m.newChangeBrief()
	m = next.(Model)
	op := m.agentOperation
	require.NoError(t, os.WriteFile(op.path, []byte("# Kept draft"), 0o600))
	next, cmd := m.applyNewBrief(newBriefEditedMsg{operation: op, err: context.Canceled})
	m = next.(Model)
	require.Nil(t, cmd)
	require.Contains(t, m.err, "editor failed; draft retained")
	require.Equal(t, "# Kept draft", readTestFile(t, op.path))
	require.Equal(t, ChangesListState, m.state)
}

func Test032ConfiguredEditorAndFallbackPersistWithSelection(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "configured-editor")
	require.NoError(t, os.WriteFile(script, []byte("#!/bin/sh\nprintf '# Edited' > \"$1\"\n"), 0o700))
	path := filepath.Join(dir, "brief.md")
	t.Setenv("EDITOR", "false")
	require.NoError(t, configuredEditorCommand(script, path).Run())
	require.Equal(t, "# Edited", readTestFile(t, path))
	t.Setenv("EDITOR", script)
	require.NoError(t, editorCommand(path).Run())
	t.Setenv("EDITOR", "")
	require.Equal(t, []string{"nano", path}, editorCommand(path).Args)
	cfgPath := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("backend_url: http://example.test\nproject_id: 7\neditor: "+script+"\n"), 0o600))
	cfg, err := loadConfigFile(cfgPath)
	require.NoError(t, err)
	require.Equal(t, script, cfg.Editor)
	cfg.ProjectID = 8
	require.NoError(t, saveAppConfig(cfgPath, cfg))
	again, err := loadConfigFile(cfgPath)
	require.NoError(t, err)
	require.Equal(t, script, again.Editor)
	require.Equal(t, 8, again.ProjectID)
}

func Test032ManualSpecSavesSynchronizeAndHumanBriefSavesLaunch(t *testing.T) {
	for _, kind := range []string{"spec", "brief"} {
		t.Run(kind, func(t *testing.T) {
			client := &fakeClient{gotChange: dto.ChangeView{ID: "12", ProjectID: "7"}}
			m := newChangeTestModel(client)
			m.state = ChangeDetailsState
			m.changeList.Detail = dto.ChangeView{ID: "12", ProjectID: "7"}
			m.changeDetailLoaded = true
			m.detailEditField = detailEditSpec
			if kind == "brief" {
				m.detailEditField = detailEditBrief
			}
			body := "# Title\n## Testcases\n- Click → saved"
			next, cmd := m.saveChangeDetailTextValue(body)
			require.NotNil(t, cmd)
			m = next.(Model)
			result := cmd().(changes.Result)
			require.NotNil(t, result.SavedDocument)
			require.False(t, result.SavedDocument.AgentEdit)
			next, cmd = m.Update(result)
			m = next.(Model)
			require.NotNil(t, cmd)
			if kind == "brief" {
				require.Equal(t, AgentExecState, m.state)
				m.agentOperation.cancel()
			} else {
				require.Equal(t, ChangeDetailsState, m.state)
				msg := cmd().(savedDocumentSyncMsg)
				require.NoError(t, msg.err)
				require.Equal(t, []dto.TestCase{{ChangeID: 12, Scenario: "Click → saved"}}, client.testCaseCreateInputs)
			}
		})
	}
	// Invalid sections retain the saved spec, without listing or mutating cases.
	client := &fakeClient{}
	m := newChangeTestModel(client)
	m.changeList.Detail = dto.ChangeView{ID: "12", Spec: "# Saved without cases"}
	next, cmd := m.syncSavedSpec(12, "# Saved without cases")
	m = next.(Model)
	next, cmd = m.Update(cmd())
	m = next.(Model)
	require.Nil(t, cmd)
	require.Equal(t, ChangeDetailsState, m.state)
	require.Contains(t, m.err, "testcase error")
	require.Equal(t, "# Saved without cases", m.changeList.Detail.Spec)
	require.Empty(t, client.testCaseCreateInputs)
}

func Test032GenericChangeDocumentSavesRouteToFlowAndSync(t *testing.T) {
	for _, kind := range []string{"brief", "spec"} {
		t.Run(kind, func(t *testing.T) {
			a := &appDocs{config: dto.ProjectConfig{ChangeDocs: []string{"brief", "spec"}}, rows: []dto.Document{}, current: []dto.Document{}}
			m := NewModelWithClient(a)
			m.currentProject.ID = "7"
			m.state = ChangeDetailsState
			m.changeDetailLoaded = true
			m.changeList.Detail.ID = "12"
			m.changeList.Detail.ProjectID = "7"
			next, cmd := m.openDocuments(ChangeDetailsState)
			m = next.(Model)
			m = applyMsg(m, cmd())
			m.document = m.document.SetType(kind)
			m.documentForm = true
			next, cmd = m.beginDocumentInsert("# Manual\n## Testcases\n- Q → R")
			m = next.(Model)
			next, cmd = m.Update(cmd())
			m = next.(Model)
			require.NotNil(t, cmd)
			require.False(t, a.insert[0].AgentEdit)
			if kind == "brief" {
				require.Equal(t, AgentExecState, m.state)
				m.agentOperation.cancel()
			} else {
				require.Equal(t, ChangeDetailsState, m.state)
				msg := cmd().(savedDocumentSyncMsg)
				require.NoError(t, msg.err)
				require.Equal(t, []dto.TestCase{{ChangeID: 12, Scenario: "Q → R"}}, a.testCaseCreateInputs)
				require.Equal(t, "# Manual\n## Testcases\n- Q → R", m.changeList.Detail.Spec)
			}
		})
	}
}

func Test032StreamingColorsStaleResultsAndExactFooterError(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	op := &agentOperation{ctx: ctx, cancel: cancel, projectID: "7", progress: make(chan string, 1), events: make(chan tea.Msg, 1)}
	m.agentOperation = op
	m.state = AgentExecState
	next, cmd := m.Update(agentProgressMsg{operation: op, text: "\x1b[32mColored progress\x1b[0m\n"})
	m = next.(Model)
	require.NotNil(t, cmd)
	require.Contains(t, m.View(), "\x1b[32mColored progress")
	old := &agentOperation{}
	next, cmd = m.Update(agentFinishedMsg{operation: old, result: agent.Result{ChangeID: 99}})
	require.Nil(t, cmd)
	require.Same(t, op, next.(Model).agentOperation)
	next, cmd = m.Update(agentFinishedMsg{operation: op, result: agent.Result{ChangeID: 12, Err: errors.New("draft retained: error generating `spec`")}})
	m = next.(Model)
	require.Nil(t, cmd)
	require.Equal(t, ChangeDetailsState, m.state)
	require.Equal(t, "Error generating `spec`", m.err)
	lines := strings.Split(stripANSI(m.View()), "\n")
	index := -1
	for i, line := range lines {
		if strings.Contains(line, "Error generating `spec`") {
			index = i
		}
	}
	require.GreaterOrEqual(t, index, 0)
	require.Equal(t, "Error generating `spec`", strings.TrimSpace(lines[index]))
	require.NotContains(t, lines[index+1], "Error:")
	require.NotContains(t, commandsByState[ChangeDetailsState], "/brief-clarify")
	require.NotContains(t, commandsByState[MainState], "/brief-new")
}

func Test032DetailRefreshFeedbackIgnoresStaleGeneration(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	m.changeList.Generation = 2
	m.changeList.EntityID = 12
	m.changeList.Operation = changes.Details
	m.status, m.err = "current screen", "current error"
	next, cmd := m.Update(agentDetailMsg{
		project: "7", status: "old spec saved", err: "old error",
		result: changes.Result{Generation: 1, ProjectID: 7, ID: 12, Operation: changes.Details},
	})
	require.Nil(t, cmd)
	require.Equal(t, "current screen", next.(Model).status)
	require.Equal(t, "current error", next.(Model).err)
}

func Test032EditorDraftCleanupRequiresOwnedIdentity(t *testing.T) {
	for _, replaced := range []bool{false, true} {
		t.Run(fmt.Sprint(replaced), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "draft.md")
			require.NoError(t, os.WriteFile(path, []byte("recoverable"), 0o600))
			info, err := os.Lstat(path)
			require.NoError(t, err)
			m := newChangeTestModel(&fakeClient{})
			saved := dto.Document{RefID: 12, RefTable: "change", DocType: "spec", Body: "recoverable"}
			m.editorScratch = &editorScratch{path: path, identity: info, projectID: 7, document: saved}
			if replaced {
				require.NoError(t, os.Rename(path, path+".original"))
				require.NoError(t, os.WriteFile(path, []byte("unrelated"), 0o600))
			}
			m = m.cleanupEditorDraft(7, saved)
			if replaced {
				require.Equal(t, "unrelated", readTestFile(t, path))
				require.Contains(t, m.err, "ownership changed")
			} else {
				require.NoFileExists(t, path)
				require.Empty(t, m.err)
			}
			require.Nil(t, m.editorScratch)
		})
	}
}

func Test032ManualSyncCanCancelWithoutNavigatingAway(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	m.changeList.Detail = dto.ChangeView{ID: "12", Spec: "## Testcases\n- Q → R"}
	next, cmd := m.syncSavedSpec(12, m.changeList.Detail.Spec)
	m = next.(Model)
	require.NotNil(t, m.specSyncCancel)
	next, blocked := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = next.(Model)
	require.Nil(t, blocked)
	require.Equal(t, ChangeDetailsState, m.state)
	next, blocked = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(Model)
	require.Nil(t, blocked)
	require.Equal(t, ChangeDetailsState, m.state)
	next, blocked = m.Update(cmd())
	m = next.(Model)
	require.Nil(t, blocked)
	require.Nil(t, m.specSyncCancel)
	require.Contains(t, m.err, "cancelled")
	require.Equal(t, "## Testcases\n- Q → R", m.changeList.Detail.Spec)
}

func Test032EditorContextCancellationRetainsDraft(t *testing.T) {
	path := filepath.Join(t.TempDir(), "brief.md")
	require.NoError(t, os.WriteFile(path, []byte("# Retained"), 0o600))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Error(t, editorProcess(ctx, "cat", path).Run())
	require.Equal(t, "# Retained", readTestFile(t, path))
}

func Test032DocumentSaveRetainedAfterFollowupFailure(t *testing.T) {
	for _, kind := range []string{"spec", "brief"} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv("TMPDIR", t.TempDir())
			a := &appDocs{config: dto.ProjectConfig{ChangeDocs: []string{"brief", "spec"}}}
			m := NewModelWithClient(a)
			m.currentProject.ID = "7"
			m.state, m.changeDetailLoaded = ChangeDetailsState, true
			m.changeList.Detail = dto.ChangeView{ID: "12", ProjectID: "7", Brief: "# Old brief", Spec: "# Old spec", DocumentTypes: []string{"brief", "spec"}, Documents: []dto.Document{{ID: 8, DocType: kind, Body: "# Old", AgentEdit: true}}}
			next, cmd := m.openDocuments(ChangeDetailsState)
			m = applyMsg(next.(Model), cmd())
			m.document = m.document.SetType(kind)
			m.documentForm = true
			body := " \n# Committed " + kind + "\nnew text\t\n "
			next, cmd = m.beginDocumentInsert(body)
			m = next.(Model)
			next, cmd = m.Update(cmd())
			m = next.(Model)
			require.False(t, m.documentForm)
			require.Empty(t, m.document.DraftBody)
			require.Equal(t, []dto.Document{{ID: 91, RefID: 12, RefTable: "change", DocType: kind, Body: strings.TrimSpace(body)}}, m.changeList.Detail.Documents)
			if kind == "spec" {
				next, cmd = m.Update(cmd())
				require.Nil(t, cmd)
				m = next.(Model)
				require.Contains(t, m.err, "testcase error")
				require.Equal(t, strings.TrimSpace(body), m.changeList.Detail.Spec)
				require.Empty(t, a.testCaseCreateInputs)
				require.Empty(t, a.testCaseDeleteIDs)
			} else {
				a.configErr = errors.New("config unavailable")
				next, cmd = m.Update(cmd())
				require.Nil(t, cmd)
				m = next.(Model)
				require.Contains(t, m.err, "config unavailable")
				require.Equal(t, strings.TrimSpace(body), m.changeList.Detail.Brief)
			}
			require.Equal(t, ChangeDetailsState, m.state)
			require.Contains(t, stripANSI(m.View()), "[✓] "+kind)
			m.changeList.Draft.DocumentType = kind
			next, cmd = m.beginDetailTextEditor(detailEditDocument)
			require.NotNil(t, cmd)
			require.Equal(t, strings.TrimSpace(body), next.(Model).promptValue())
		})
	}
}

func Test032CreatedTypesRemainVisibleAfterWorkflowError(t *testing.T) {
	m := newChangeTestModel(&fakeClient{})
	ctx, cancel := context.WithCancel(context.Background())
	op := &agentOperation{ctx: ctx, cancel: cancel, projectID: "7", title: "New", brief: "# New"}
	m.agentOperation = op
	next, cmd := m.applyAgentFinished(agentFinishedMsg{operation: op, result: agent.Result{ChangeID: 12, ChangeTypes: []string{"feature"}, ChangeTypesSaved: true, Err: errors.New("rewrite failed")}})
	require.Nil(t, cmd)
	require.Equal(t, []string{"feature"}, next.(Model).changeList.Detail.ChangeTypes)
	require.Contains(t, stripANSI(next.(Model).View()), "feature")
}

func Test032FailedEditorSaveSurvivesAnotherChangesCompletion(t *testing.T) {
	for _, completion := range []string{"agent", "manual-spec"} {
		t.Run(completion, func(t *testing.T) {
			client := &fakeClient{changeUpdateErr: errors.New("save refused")}
			m := newChangeTestModel(client)
			m.state, m.detailEditField = ChangeDetailsState, detailEditSpec
			m.changeList.Detail = dto.ChangeView{ID: "12", ProjectID: "7"}
			path := filepath.Join(t.TempDir(), "spec.md")
			body := "# Recoverable spec\n## Testcases\n- Q → R"
			require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
			info, err := os.Lstat(path)
			require.NoError(t, err)
			next, cmd := m.Update(editorFinishedMsg{source: ChangeDetailsState, content: body, scratch: &editorScratch{path: path, identity: info, projectID: 7, document: dto.Document{RefID: 12, RefTable: "change", DocType: "spec", Body: body}}})
			m = next.(Model)
			// The editor save command is batched with a screen clear.
			batch := cmd().(tea.BatchMsg)
			m = applyMsg(m, batch[1]())
			require.Contains(t, m.err, "save refused")
			require.Equal(t, body, readTestFile(t, path))
			m.changeList.Detail = dto.ChangeView{ID: "13", ProjectID: "7"}
			if completion == "agent" {
				ctx, cancel := context.WithCancel(context.Background())
				m.agentOperation = &agentOperation{ctx: ctx, cancel: cancel, projectID: "7"}
				next, _ = m.Update(agentFinishedMsg{operation: m.agentOperation, result: agent.Result{ChangeID: 13}})
			} else {
				next, _ = m.Update(savedDocumentSyncMsg{project: "7", change: 13, generation: m.selectionGeneration})
			}
			require.Equal(t, body, readTestFile(t, path))
			require.NotNil(t, next.(Model).editorScratch)
		})
	}
}

func Test032NewChangeFailureRequiresDetailReloadBeforeEditing(t *testing.T) {
	for _, failure := range []string{"rewrite failed", "spec writing failed"} {
		t.Run(failure, func(t *testing.T) {
			body := "# Initial saved brief"
			client := &fakeClient{gotChange: dto.ChangeView{ID: "13", ProjectID: "7", Brief: body}}
			m := newChangeTestModel(client)
			m.changeList.Detail = dto.ChangeView{ID: "12", ProjectID: "7", Brief: "# Previous change"}
			m.changeList.DetailLoaded, m.changeDetailLoaded = true, true
			m.testCase.Loaded = true
			ctx, cancel := context.WithCancel(context.Background())
			op := &agentOperation{ctx: ctx, cancel: cancel, projectID: "7", uuid: "new-ref", title: "Initial saved brief", brief: body}
			m.agentOperation = op
			next, cmd := m.Update(agentFinishedMsg{operation: op, result: agent.Result{ChangeID: 13, Status: "change created; brief saved", Err: errors.New(failure)}})
			m = next.(Model)
			require.Nil(t, cmd)
			require.Equal(t, "13", m.changeList.Detail.ID)
			require.Equal(t, body, m.changeList.Detail.Brief)
			require.Contains(t, m.err, failure)
			require.False(t, m.changeDetailLoaded)
			require.False(t, m.changeList.DetailLoaded)
			require.False(t, m.testCase.Loaded)
			m.changeList.Draft.DocumentType = "brief"
			for _, field := range []detailEditField{detailEditDocument, detailEditBrief, detailEditSpec} {
				next, cmd = m.beginDetailTextEditor(field)
				require.Nil(t, cmd, "incomplete snapshots must not reach the editor")
				require.Contains(t, next.(Model).err, "Load change details")
			}
			client.changeGetErr = errors.New("read unavailable")
			next, cmd = m.executeCommandFrom(ChangeDetailsState, "/retry")
			m = applyMsg(next.(Model), cmd())
			require.False(t, m.changeDetailLoaded)
			require.False(t, m.changeList.DetailLoaded)
			client.changeGetErr = nil
			next, cmd = m.executeCommandFrom(ChangeDetailsState, "/retry")
			m = applyMsg(next.(Model), cmd())
			require.True(t, m.changeDetailLoaded)
			require.True(t, m.changeList.DetailLoaded)
			m.changeList.Draft.DocumentType = "brief"
			next, cmd = m.beginDetailTextEditor(detailEditDocument)
			require.NotNil(t, cmd)
			require.Equal(t, body, next.(Model).promptValue())
		})
	}
}

func Test032EditorDraftCleanupMatchesOnlyThePersistedDocument(t *testing.T) {
	for _, mismatch := range []string{"project", "owner", "table", "type", "body", "agent"} {
		t.Run(mismatch, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "draft.md")
			require.NoError(t, os.WriteFile(path, []byte(" recoverable\n"), 0o600))
			info, err := os.Lstat(path)
			require.NoError(t, err)
			draft := dto.Document{RefID: 12, RefTable: "change", DocType: "brief", Body: " recoverable\n"}
			m := newChangeTestModel(&fakeClient{})
			m.editorScratch = &editorScratch{path: path, identity: info, projectID: 7, document: draft}
			saved := draft
			saved.Body = strings.TrimSpace(draft.Body)
			project := 7
			switch mismatch {
			case "project":
				project = 8
			case "owner":
				saved.RefID = 13
			case "table":
				saved.RefTable = "epic"
			case "type":
				saved.DocType = "spec"
			case "body":
				saved.Body = "another draft"
			case "agent":
				saved.AgentEdit = true
			}
			m = m.cleanupEditorDraft(project, saved)
			require.Equal(t, draft.Body, readTestFile(t, path))
			require.NotNil(t, m.editorScratch)
			saved = draft
			saved.Body = strings.TrimSpace(draft.Body)
			m = m.cleanupEditorDraft(7, saved)
			require.NoFileExists(t, path)
			require.Nil(t, m.editorScratch)
		})
	}
}

func Test032MatchingEditorDraftRemovedOnlyAfterItsSuccessfulFollowup(t *testing.T) {
	for _, source := range []string{"detail", "documents"} {
		for _, kind := range []string{"brief", "spec"} {
			t.Run(source+"/"+kind, func(t *testing.T) {
				path := filepath.Join(t.TempDir(), "draft.md")
				body := " \n# Saved body\n## Testcases\n- Q → R\n "
				require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
				info, err := os.Lstat(path)
				require.NoError(t, err)
				a := &appDocs{config: dto.ProjectConfig{ChangeDocs: []string{"brief", "spec"}}}
				a.gotChange = dto.ChangeView{ID: "12", ProjectID: "7"}
				m := newChangeTestModel(&a.fakeClient)
				m.client = a
				m.state, m.changeList.Detail = ChangeDetailsState, a.gotChange
				m.editorScratch = &editorScratch{path: path, identity: info, projectID: 7, document: dto.Document{RefID: 12, RefTable: "change", DocType: kind, Body: body}}
				var next tea.Model
				var cmd tea.Cmd
				if source == "documents" {
					next, cmd = m.openDocuments(ChangeDetailsState)
					m = applyMsg(next.(Model), cmd())
					m.document = m.document.SetType(kind)
					m.documentForm = true
					next, cmd = m.beginDocumentInsert(body)
				} else {
					m.detailEditField = detailEditBrief
					if kind == "spec" {
						m.detailEditField = detailEditSpec
					}
					next, cmd = m.saveChangeDetailTextValue(body)
				}
				m = next.(Model)
				next, followup := m.Update(cmd())
				m = next.(Model)
				require.Equal(t, body, readTestFile(t, path))
				require.NotNil(t, m.editorScratch)
				require.NotNil(t, followup)
				if m.agentOperation != nil {
					op := m.agentOperation
					next, _ = m.Update(agentFinishedMsg{operation: op, result: agent.Result{ChangeID: 12}})
				} else {
					next, _ = m.Update(followup())
				}
				require.NoFileExists(t, path)
				require.Nil(t, next.(Model).editorScratch)
			})
		}
	}
}

func Test032OrdinaryDocumentSaveCleansMatchingEditorDraft(t *testing.T) {
	for _, owner := range []string{"project", "epic"} {
		for _, kind := range []string{"brief", "spec"} {
			for _, failed := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/failed=%t", owner, kind, failed), func(t *testing.T) {
					a := &appDocs{config: dto.ProjectConfig{ProjectDocs: []string{kind}, EpicDocs: []string{kind}}, rows: []dto.Document{}, current: []dto.Document{}}
					m := NewModelWithClient(a)
					m.currentProject.ID = "7"
					m.projectList.Detail, m.projectList.DetailLoaded = dto.Project{ID: 7}, true
					m.epicList.Detail, m.epicList.DetailLoaded = dto.Epic{ID: 12, ProjectID: 7}, true
					source, id := ProjectDetailsState, 7
					if owner == "epic" {
						source, id = EpicDetailsState, 12
					}
					next, cmd := m.openDocuments(source)
					m = applyMsg(next.(Model), cmd())
					m.documentForm = true
					body := " \n# Ordinary draft\n "
					path := filepath.Join(t.TempDir(), "mch-project-draft.md")
					require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
					info, err := os.Lstat(path)
					require.NoError(t, err)
					m = applyMsg(m, editorFinishedMsg{source: DocumentState, content: body, scratch: &editorScratch{path: path, identity: info, projectID: 7, document: dto.Document{RefID: id, RefTable: owner, DocType: kind, Body: body}}})
					require.FileExists(t, path, "editing alone must retain the unsaved draft")
					if failed {
						a.insertErr = errors.New("save refused")
					}
					next, cmd = m.beginDocumentInsert(body)
					next, refresh := next.(Model).Update(cmd())
					m = next.(Model)
					require.Equal(t, []dto.DocumentInput{{RefID: id, RefTable: owner, DocType: kind, Body: body}}, a.insert)
					if failed {
						// Retry the same draft after the API becomes available.
						require.Nil(t, refresh)
						require.Contains(t, m.err, "save refused")
						require.Equal(t, body, readTestFile(t, path))
						require.NotNil(t, m.editorScratch)
						a.insertErr = nil
						next, cmd = m.beginDocumentInsert(body)
						next, refresh = next.(Model).Update(cmd())
						m = next.(Model)
					}
					require.NoFileExists(t, path)
					require.Nil(t, m.editorScratch)
					require.Nil(t, m.agentOperation)
					require.Nil(t, m.specSyncCancel)
					require.Equal(t, DocumentState, m.state)
					require.NotNil(t, refresh, "ordinary save still refreshes document history")
					a.rows = []dto.Document{{ID: 91, RefID: id, RefTable: owner, DocType: kind, Body: strings.TrimSpace(body)}}
					a.current = a.rows
					a.detail = a.rows[0]
					m = applyMsg(m, refresh())
					require.Empty(t, m.err)
				})
			}
		}
	}
}
