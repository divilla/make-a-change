package agent

import (
	"cli/internal/dto"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type briefAPI struct {
	config                        dto.ProjectConfig
	configErr                     error
	change                        dto.Change
	docs                          []dto.Document
	creates                       []dto.ChangeCreateInput
	inserts                       []dto.DocumentInput
	createErr, insertErr, readErr error
	reads                         int
}

func (a *briefAPI) GetProjectConfig(context.Context, int) (dto.ProjectConfig, error) {
	if a.configErr != nil {
		return dto.ProjectConfig{}, a.configErr
	}
	return a.config, nil
}
func (a *briefAPI) GetChange(context.Context, int) (dto.Change, error) { return a.change, nil }
func (a *briefAPI) CreateChange(_ context.Context, in dto.ChangeCreateInput) (int, error) {
	a.creates = append(a.creates, in)
	if a.createErr != nil {
		return 0, a.createErr
	}
	a.change = dto.Change{ID: 21, ProjectID: in.ProjectID}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: strings.TrimSpace(in.Brief), Current: true}}
	return 21, nil
}

func (a *briefAPI) CurrentDocuments(context.Context, int, string) ([]dto.Document, error) {
	a.reads++
	if a.readErr != nil {
		return nil, a.readErr
	}
	return append([]dto.Document(nil), a.docs...), nil
}

func (a *briefAPI) InsertDocument(_ context.Context, in dto.DocumentInput) (int, error) {
	a.inserts = append(a.inserts, in)
	if a.insertErr != nil {
		return 0, a.insertErr
	}
	id := 31 + len(a.inserts)
	a.docs = []dto.Document{{ID: id, RefID: in.RefID, RefTable: "change", DocType: in.DocType, Body: strings.TrimSpace(in.Body), Current: true}}
	return id, nil
}

type briefRunner struct {
	results  []Output
	requests []Request
	err      error
	path     string
}

func (r *briefRunner) Run(_ context.Context, req Request) (Output, string, error) {
	r.requests = append(r.requests, req)
	if r.err != nil {
		return Output{}, filepath.Dir(req.InputPath), r.err
	}
	out := r.results[0]
	r.results = r.results[1:]
	out.InputRevision = req.Revision
	if r.path != "" {
		return out, r.path, nil
	}
	return out, filepath.Dir(req.InputPath), nil
}

func TestP803ControllerForwardsProgressChannel(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Text", Current: true}}
	m := Existing(7, 21)
	m.Draft, m.BackendBrief, m.DocumentID = "Text", "Text", 31
	progress := make(chan string, 1)
	runner := &briefRunner{results: []Output{{RewrittenBrief: "Clear text", Questions: []Question{}, Unresolved: []string{}, ReadyForSpec: true}}}
	_, run := m.Begin(context.Background(), Run, progress)
	require.NotNil(t, run)
	result := run(a, runner, briefTestRoot(t))
	require.NoError(t, result.Err)
	require.Len(t, runner.requests, 1)
	require.Equal(t, (chan<- string)(progress), runner.requests[0].Progress)
}

func TestP803ControllerOwnsRunnerPathsAndRejectsMissingArrays(t *testing.T) {
	for _, tc := range []struct {
		name string
		out  Output
		want string
	}{
		{"missing questions", Output{RewrittenBrief: "Ready", Unresolved: []string{}, ReadyForSpec: true}, "arrays"},
		{"missing unresolved", Output{RewrittenBrief: "Ready", Questions: []Question{}, ReadyForSpec: true}, "arrays"},
		{"wrong output directory", Output{RewrittenBrief: "Ready", Questions: []Question{}, Unresolved: []string{}, ReadyForSpec: true}, "outside its operation directory"},
		{"valid empty arrays", Output{RewrittenBrief: "Ready", Questions: []Question{}, Unresolved: []string{}, ReadyForSpec: true}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := validAPI()
			a.change = dto.Change{ID: 21, ProjectID: 7}
			a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Original", Current: true}}
			root := briefTestRoot(t)
			m, _ := doStep(t, Existing(7, 21), a, nil, Preflight)
			runner := &briefRunner{results: []Output{tc.out}}
			if tc.name == "wrong output directory" {
				runner.path = t.TempDir()
			}
			next, run := m.Begin(context.Background(), Run)
			require.NotNil(t, run)
			result := run(a, runner, root)
			require.Len(t, runner.requests, 1)
			req := runner.requests[0]
			dir := filepath.Dir(req.InputPath)
			require.Equal(t, filepath.Join(root, ".mch", "tmp"), filepath.Dir(dir))
			require.DirExists(t, dir)
			require.Equal(t, filepath.Join(dir, "original.md"), req.OriginalPath)
			require.Equal(t, filepath.Join(dir, "context.json"), req.ContextPath)
			require.Equal(t, filepath.Join(dir, "questions.json"), req.QuestionsPath)
			require.Equal(t, filepath.Join(dir, "answers.json"), req.AnswersPath)
			require.Equal(t, filepath.Join(dir, "result.json"), req.OutputPath)
			next, _, ok := next.Apply(result)
			require.True(t, ok)
			if tc.want != "" {
				require.ErrorContains(t, next.Err, tc.want)
				require.Equal(t, Failed, next.Phase)
				require.NotEqual(t, Ready, next.Phase)
			} else {
				require.NoError(t, next.Err)
				require.Equal(t, Review, next.Phase)
			}
		})
	}
}

func TestP803ControllerRefusesUnownedScratchPath(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Original", Current: true}}
	root := briefTestRoot(t)
	base := filepath.Join(root, ".mch", "tmp")
	require.NoError(t, os.WriteFile(base, []byte("user file"), 0o600))
	m, _ := doStep(t, Existing(7, 21), a, nil, Preflight)
	runner := &briefRunner{results: []Output{{RewrittenBrief: "Ready", Questions: []Question{}, Unresolved: []string{}, ReadyForSpec: true}}}
	next, run := m.Begin(context.Background(), Run)
	result := run(a, runner, root)
	next, _, ok := next.Apply(result)
	require.True(t, ok)
	require.Equal(t, Failed, next.Phase)
	require.ErrorContains(t, next.Err, "not an owned directory")
	require.Empty(t, runner.requests)
	content, err := os.ReadFile(base)
	require.NoError(t, err)
	require.Equal(t, "user file", string(content))
}

func validAPI() *briefAPI {
	return &briefAPI{config: dto.ProjectConfig{ChangeDocs: []string{"brief", "spec"}, ChangePhases: []string{"backlog"}}}
}

func briefTestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, ".mch"), 0o700))
	return root
}

func doStep(t *testing.T, m Model, a *briefAPI, r *briefRunner, step Step) (Model, Step) {
	t.Helper()
	next, run := m.Begin(context.Background(), step)
	require.NotNil(t, run)
	result := run(a, r, briefTestRoot(t))
	var ok bool
	next, step, ok = next.Apply(result)
	require.True(t, ok)
	return next, step
}

func TestP801ControllerPhasesAndOriginalInput(t *testing.T) {
	m := New(7).EditIdentity("A title", "").EditBrief("\t# Brief\n```sh\nrm -rf example\n```\n")
	require.Equal(t, "\t# Brief\n```sh\nrm -rf example\n```\n", m.Original)
	a := validAPI()
	m, step := doStep(t, m, a, nil, Preflight)
	require.Empty(t, step)
	require.Equal(t, Draft, m.Phase)
	m, step = doStep(t, m, a, nil, Write)
	require.Equal(t, Refresh, step)
	require.Equal(t, 21, m.ChangeID)
	require.Equal(t, 21, m.CommittedID)
	require.Equal(t, "\t# Brief\n```sh\nrm -rf example\n```\n", a.creates[0].Brief)
	m, step = doStep(t, m, a, nil, step)
	require.Equal(t, Run, step)
	require.Equal(t, 31, m.DocumentID)
	require.Equal(t, "\t# Brief\n```sh\nrm -rf example\n```\n", m.Original)
}

func TestP802CommittedIdentitySurvivesReadAndAgentFailure(t *testing.T) {
	a := validAPI()
	m := New(7).EditIdentity("Title", "").EditBrief(" brief ")
	m, _ = doStep(t, m, a, nil, Write)
	a.readErr = errors.New("read down")
	m, _ = doStep(t, m, a, nil, Refresh)
	require.Equal(t, 21, m.CommittedID)
	require.Equal(t, 21, m.ChangeID)
	require.Len(t, a.creates, 1)
	a.readErr = nil
	m, step := doStep(t, m, a, nil, Write)
	require.Equal(t, Run, step)
	r := &briefRunner{err: errors.New("process failed")}
	m, _ = doStep(t, m, a, r, Run)
	require.Equal(t, Failed, m.Phase)
	require.Contains(t, m.ScratchPath, filepath.Join(".mch", "tmp", "brief-"))
	require.Len(t, a.creates, 1)
}

func TestP802CreatedBriefConflictBeforeRunner(t *testing.T) {
	a := validAPI()
	m := New(7).EditIdentity("Title", "").EditBrief(" brief ")
	m, step := doStep(t, m, a, nil, Write)
	require.Equal(t, Refresh, step)
	a.docs[0].Body = "Another client's brief"
	m, step = doStep(t, m, a, nil, Refresh)
	require.Empty(t, step)
	require.Equal(t, Failed, m.Phase)
	require.Equal(t, Refresh, m.FailedStep)
	require.ErrorContains(t, m.Err, "created brief is not the current document")
	require.Equal(t, 21, m.CommittedID)
	require.Equal(t, " brief ", m.Draft)
	require.Len(t, a.creates, 1)
	m, step = doStep(t, m, a, nil, Refresh)
	require.Empty(t, step)
	require.Len(t, a.creates, 1)
}

func TestP804MultipleQuestionsAnswersAndResolveLoops(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Original", Current: true}}
	m := Existing(7, 21)
	m, _ = doStep(t, m, a, nil, Preflight)
	r := &briefRunner{results: []Output{
		{RewrittenBrief: "Rewrite one", Questions: []Question{{ID: "Q1", Text: "Which?", Context: "section A"}, {ID: "Q2", Text: "When?", Context: "section B"}}, Unresolved: []string{"Q1", "Q2"}},
		{RewrittenBrief: "Rewrite two", Questions: []Question{{ID: "Q1", Text: "Which?", Context: "section A"}, {ID: "Q2", Text: "When?", Context: "section B"}}, Unresolved: []string{"Q2"}},
		{RewrittenBrief: "Rewrite three", Questions: []Question{{ID: "Q1", Text: "Which?", Context: "section A"}, {ID: "Q2", Text: "When?", Context: "section B"}}, Unresolved: []string{}, ReadyForSpec: true},
	}}
	m, _ = doStep(t, m, a, r, Run)
	require.Equal(t, Review, m.Phase)
	var run func(API, Runner, string) Result
	m, run = m.Approve(context.Background())
	require.NotNil(t, run)
	result := run(a, r, briefTestRoot(t))
	m, step, ok := m.Apply(result)
	require.True(t, ok)
	require.Equal(t, Refresh, step)
	require.True(t, a.inserts[0].AgentEdit)
	m, _ = doStep(t, m, a, r, Refresh)
	require.Equal(t, Answering, m.Phase)
	m = m.Answer("Q1", "Alpha")
	m, run = m.Resolve(context.Background())
	require.Nil(t, run)
	require.ErrorContains(t, m.Err, "Q2")
	m = m.Answer("Q2", "Tomorrow")
	m, run = m.Resolve(context.Background())
	require.NotNil(t, run)
	result = run(a, r, briefTestRoot(t))
	m, _, ok = m.Apply(result)
	require.True(t, ok)
	require.Equal(t, "brief-resolve", r.requests[1].Prompt)
	require.Equal(t, "Alpha", r.requests[1].Questions[0].Answer)
	require.Equal(t, "Tomorrow", r.requests[1].Questions[1].Answer)
	m, run = m.Approve(context.Background())
	require.NotNil(t, run)
	result = run(a, r, briefTestRoot(t))
	m, _, ok = m.Apply(result)
	require.True(t, ok)
	m, _ = doStep(t, m, a, r, Refresh)
	require.Equal(t, Answering, m.Phase)
	m, run = m.Resolve(context.Background())
	require.Nil(t, run)
	require.ErrorContains(t, m.Err, "Q2")
	m = m.Answer("Q2", "Confirmed tomorrow")
	m, run = m.Resolve(context.Background())
	require.NotNil(t, run)
	result = run(a, r, briefTestRoot(t))
	m, _, ok = m.Apply(result)
	require.True(t, ok)
	m, run = m.Approve(context.Background())
	require.NotNil(t, run)
	result = run(a, r, briefTestRoot(t))
	m, _, ok = m.Apply(result)
	require.True(t, ok)
	m, _ = doStep(t, m, a, r, Refresh)
	require.Equal(t, Ready, m.Phase)
	require.Equal(t, "Original", m.Original)
}

func TestP804UnansweredAndConflictingBlockersCannotAdvance(t *testing.T) {
	base := Output{InputRevision: 4, RewrittenBrief: "Draft", Questions: []Question{{ID: "Q", Text: "What?", Context: "line 1"}}, Unresolved: []string{"Q"}}
	require.NoError(t, validateOutput(base, 4, nil))
	bad := base
	bad.ReadyForSpec = true
	require.Error(t, validateOutput(bad, 4, nil))
	bad = base
	bad.Unresolved = nil
	bad.ReadyForSpec = true
	require.Error(t, validateOutput(bad, 4, nil))
	bad = base
	bad.Questions = append(bad.Questions, bad.Questions[0])
	require.Error(t, validateOutput(bad, 4, nil))
	bad = base
	bad.InputRevision = 3
	require.Error(t, validateOutput(bad, 4, nil))
	bad = base
	bad.Unresolved = []string{"unknown"}
	require.Error(t, validateOutput(bad, 4, nil))
	require.ErrorContains(t, validateOutput(Output{InputRevision: 4, RewrittenBrief: "Draft", Questions: []Question{}, Unresolved: []string{}}, 4, nil), "no unresolved blockers")
}

func TestP804QuestionIDsMatchAnswerInputSyntax(t *testing.T) {
	base := Output{InputRevision: 4, RewrittenBrief: "Draft", Unresolved: []string{"Q1"}}
	for _, id := range []string{" Q1 ", "phase:date", "Q\t1", "Q\n1"} {
		t.Run(id, func(t *testing.T) {
			out := base
			out.Questions = []Question{{ID: id, Text: "When?", Context: "date"}}
			out.Unresolved = []string{id}
			require.ErrorContains(t, validateOutput(out, 4, nil), "incomplete question")
		})
	}
	base.Questions = []Question{{ID: "phase date", Text: "When?", Context: "date"}}
	base.Unresolved = []string{"phase date"}
	require.NoError(t, validateOutput(base, 4, nil))
}

func TestP804NonReadyWithoutBlockersStaysRecoverable(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Original", Current: true}}
	m := Existing(7, 21)
	m, _ = doStep(t, m, a, nil, Preflight)
	r := &briefRunner{results: []Output{{RewrittenBrief: "Original", Questions: []Question{}, Unresolved: []string{}}}}
	m, _ = doStep(t, m, a, r, Run)
	require.Equal(t, Failed, m.Phase)
	require.Equal(t, Run, m.FailedStep)
	require.ErrorContains(t, m.Err, "no unresolved blockers")
	require.Empty(t, m.Questions)
	require.Empty(t, m.Candidate.RewrittenBrief)
}

func TestP804AnsweredPairsSurviveAgentOmissionAndFollowUp(t *testing.T) {
	m := Existing(7, 21)
	m.Phase = Review
	m.Questions = []Question{{ID: "Q1", Text: "Which?", Context: "intro", Answer: "Alpha"}, {ID: "Q2", Text: "When?", Context: "timeline", Answer: "Tomorrow"}}
	m.Candidate = Output{InputRevision: m.Revision, RewrittenBrief: "Draft", Questions: []Question{{ID: "Q2", Text: "When exactly?", Context: "timeline"}}, Unresolved: []string{"Q2"}}
	m = m.finishCandidate()
	require.Len(t, m.Questions, 2)
	require.Equal(t, "Alpha", m.Questions[1].Answer)
	require.True(t, m.NeedsAnswer["Q2"])
	m, run := m.Resolve(context.Background())
	require.Nil(t, run)
	require.ErrorContains(t, m.Err, "Q2")
	m = m.Answer("Q2", "Next Tuesday")
	require.False(t, m.NeedsAnswer["Q2"])
	require.Equal(t, "Alpha", m.Questions[1].Answer)
}

func TestP805StaleSelectionRevisionAndShutdownCancellation(t *testing.T) {
	a := validAPI()
	m := New(7).EditIdentity("Title", "").EditBrief("Original")
	next, run := m.Begin(context.Background(), Preflight)
	require.NotNil(t, run)
	result := run(a, nil, briefTestRoot(t))
	next = next.EditBrief("changed")
	ignored, _, ok := next.Apply(result)
	require.False(t, ok)
	require.Equal(t, "changed", ignored.Draft)
	ignored = ignored.Invalidate()
	require.False(t, ignored.Busy)
}

func TestP806MissingCatalogAndCurrentBriefRecovery(t *testing.T) {
	a := validAPI()
	a.configErr = errors.New("catalog unavailable")
	m := New(7)
	m, _ = doStep(t, m, a, nil, Preflight)
	require.ErrorContains(t, m.Err, "catalog unavailable")
	require.Empty(t, a.creates)
	a.configErr = nil
	a.config.ChangeDocs = []string{"brief"}
	m = New(7)
	m, _ = doStep(t, m, a, nil, Preflight)
	require.Equal(t, Failed, m.Phase)
	require.ErrorContains(t, m.Err, "spec")
	require.Empty(t, a.creates)
	a.config.ChangeDocs = []string{"spec"}
	m = New(7)
	m, _ = doStep(t, m, a, nil, Preflight)
	require.ErrorContains(t, m.Err, "brief")
	a = validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	m = Existing(7, 21)
	m, _ = doStep(t, m, a, nil, Preflight)
	require.Equal(t, Draft, m.Phase)
	require.Empty(t, m.BackendBrief)
	m = m.EditBrief("Supplied missing brief")
	m, step := doStep(t, m, a, nil, Write)
	require.Equal(t, Refresh, step)
	require.False(t, a.inserts[0].AgentEdit)
	require.Equal(t, "Supplied missing brief", a.inserts[0].Body)
	m, _ = doStep(t, m, a, nil, Refresh)
	require.Equal(t, "Supplied missing brief", m.BackendBrief)
	a.docs = []dto.Document{{ID: 31, RefID: 99, RefTable: "change", DocType: "brief", Current: true}}
	m, _ = doStep(t, m, a, nil, Preflight)
	require.Equal(t, Failed, m.Phase)
	require.ErrorContains(t, m.Err, "owner")
}

func TestP802NewCreateAndExistingHumanInsertPayloads(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Existing", Current: true}}
	m := Existing(7, 21)
	m, _ = doStep(t, m, a, nil, Preflight)
	require.Equal(t, "Existing", m.Original)
	m = m.EditBrief("  edited\t\n```sh\nprint hi\n```\n")
	require.Equal(t, "  edited\t\n```sh\nprint hi\n```\n", m.Original)
	m, step := doStep(t, m, a, nil, Write)
	require.Equal(t, Refresh, step)
	require.Len(t, a.inserts, 1)
	require.Equal(t, dto.DocumentInput{RefID: 21, RefTable: "change", DocType: "brief", Body: "  edited\t\n```sh\nprint hi\n```\n", AgentEdit: false}, a.inserts[0])
	require.Equal(t, 32, m.CommittedID)
	m, step = doStep(t, m, a, nil, Refresh)
	require.Equal(t, Run, step)
	require.Equal(t, "  edited\t\n```sh\nprint hi\n```\n", m.Original)
	require.Equal(t, 32, m.DocumentID)
}

func TestP802OriginalBriefExactEditorAndNoOp(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Tabs\tand\nlines", Current: true}}
	m := Existing(7, 21)
	m, _ = doStep(t, m, a, nil, Preflight)
	m, run := m.Begin(context.Background(), Write)
	require.NotNil(t, run)
	require.Equal(t, Run, m.Pending)
	require.Empty(t, a.inserts)
	m = m.Invalidate()
	m = m.EditBrief("manual change")
	require.Equal(t, "manual change", m.Original)
	require.Equal(t, "manual change", m.Draft)
}

func TestP805WhitespaceOnlyHumanAndAgentChangesDoNotAppend(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "text", Current: true}}
	m, _ := doStep(t, Existing(7, 21), a, nil, Preflight)
	m = m.EditBrief("text\n")
	require.Equal(t, "text\n", m.Original)
	m, run := m.Begin(context.Background(), Write)
	require.NotNil(t, run)
	require.Equal(t, Run, m.Pending)
	require.Empty(t, a.inserts)
	runner := &briefRunner{results: []Output{{RewrittenBrief: " \ttext\n", Questions: []Question{}, Unresolved: []string{}, ReadyForSpec: true}}}
	result := run(a, runner, briefTestRoot(t))
	var ok bool
	m, _, ok = m.Apply(result)
	require.True(t, ok)
	require.Equal(t, Review, m.Phase)
	require.Equal(t, "text\n", runner.requests[0].Brief)
	require.Equal(t, "text\n", runner.requests[0].Original)
	m, verify := m.Approve(context.Background())
	require.NotNil(t, verify)
	require.Equal(t, Verify, m.Pending)
	result = verify(a, runner, briefTestRoot(t))
	m, _, ok = m.Apply(result)
	require.True(t, ok)
	require.Equal(t, Ready, m.Phase)
	require.Equal(t, 31, m.DocumentID)
	require.Empty(t, a.inserts)
}

func TestP802ExistingFirstUserEditRemainsOriginalAfterAgentRewrite(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Backend brief", Current: true}}
	m, _ := doStep(t, Existing(7, 21), a, nil, Preflight)
	userBrief := "\tUser brief\n```sh\nprint example\n```\n"
	m = m.EditBrief(userBrief)
	require.Equal(t, "Backend brief", m.BackendBrief)
	require.Equal(t, userBrief, m.Original)
	m, step := doStep(t, m, a, nil, Write)
	require.Equal(t, Refresh, step)
	m, step = doStep(t, m, a, nil, step)
	require.Equal(t, Run, step)
	r := &briefRunner{results: []Output{{RewrittenBrief: "Agent rewrite", Questions: []Question{}, Unresolved: []string{}, ReadyForSpec: true}}}
	m, _ = doStep(t, m, a, r, Run)
	require.Equal(t, userBrief, r.requests[0].Original)
	require.Equal(t, userBrief, r.requests[0].Brief)
	m, run := m.Approve(context.Background())
	require.NotNil(t, run)
	result := run(a, r, briefTestRoot(t))
	var ok bool
	m, step, ok = m.Apply(result)
	require.True(t, ok)
	require.Equal(t, Refresh, step)
	m, _ = doStep(t, m, a, r, step)
	require.Equal(t, Ready, m.Phase)
	require.Equal(t, "Agent rewrite", m.Draft)
	require.Equal(t, userBrief, m.Original)
	require.Equal(t, "Agent rewrite", m.BackendBrief)
	m = m.EditBrief("Later user revision")
	require.Equal(t, userBrief, m.Original)
}

func TestP802FailedWriteRetainsDraftAndBusyDeduplication(t *testing.T) {
	a := validAPI()
	a.createErr = errors.New("write failed")
	m := New(7).EditIdentity("Title", "").EditBrief("Draft")
	next, run := m.Begin(context.Background(), Write)
	require.NotNil(t, run)
	busy, duplicate := next.Begin(context.Background(), Write)
	require.Nil(t, duplicate)
	require.Equal(t, next.Generation, busy.Generation)
	result := run(a, nil, briefTestRoot(t))
	m, _, ok := next.Apply(result)
	require.True(t, ok)
	require.Equal(t, Failed, m.Phase)
	require.Equal(t, Write, m.FailedStep)
	require.Equal(t, "Draft", m.Draft)
	require.Zero(t, m.ChangeID)
	a.createErr = nil
	m, _ = doStep(t, m, a, nil, Write)
	require.Equal(t, 21, m.ChangeID)
}

func TestP805CommittedBriefSurvivesRepeatedFailedRefresh(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Before", Current: true}}
	m := Existing(7, 21)
	m, _ = doStep(t, m, a, nil, Preflight)
	m = m.EditBrief("After")
	m, _ = doStep(t, m, a, nil, Write)
	a.readErr = errors.New("refresh failed")
	for range 2 {
		m, _ = doStep(t, m, a, nil, Write)
		require.Equal(t, 32, m.CommittedID)
		require.Equal(t, Failed, m.Phase)
		require.Equal(t, Refresh, m.FailedStep)
	}
	require.Len(t, a.inserts, 1)
	a.readErr = nil
	m, step := doStep(t, m, a, nil, Refresh)
	require.Equal(t, Run, step)
	require.Equal(t, 32, m.DocumentID)
	require.Zero(t, m.CommittedID)
}

func TestP805RefreshRetryKeepsUnsavedEditInDraft(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Before", Current: true}}
	m := Existing(7, 21)
	m, _ = doStep(t, m, a, nil, Preflight)
	m = m.EditBrief("Committed")
	m, step := doStep(t, m, a, nil, Write)
	require.Equal(t, Refresh, step)
	a.readErr = errors.New("refresh failed")
	m, step = doStep(t, m, a, nil, Refresh)
	require.Empty(t, step)
	require.Equal(t, Failed, m.Phase)
	require.Equal(t, 32, m.CommittedID)

	m = m.EditBrief("Unsaved edit")
	a.readErr = nil
	m, step = doStep(t, m, a, nil, Write)
	require.Empty(t, step)
	require.Equal(t, Draft, m.Phase)
	require.Equal(t, "Unsaved edit", m.Draft)
	require.Equal(t, "Committed", m.BackendBrief)
	require.Equal(t, 32, m.DocumentID)
	require.Zero(t, m.CommittedID)
	require.Len(t, a.inserts, 1)

	m, step = doStep(t, m, a, nil, Write)
	require.Equal(t, Refresh, step)
	require.Len(t, a.inserts, 2)
	require.Equal(t, "Unsaved edit", a.inserts[1].Body)
	require.False(t, a.inserts[1].AgentEdit)
	m, step = doStep(t, m, a, nil, Refresh)
	require.Equal(t, Run, step)
}

func TestP804EditsInvalidateReadinessAndStaleOutput(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Original", Current: true}}
	m := Existing(7, 21)
	m, _ = doStep(t, m, a, nil, Preflight)
	r := &briefRunner{results: []Output{{RewrittenBrief: "Original", Questions: []Question{}, Unresolved: []string{}, ReadyForSpec: true}}}
	m, _ = doStep(t, m, a, r, Run)
	m, cmd := m.Approve(context.Background())
	require.NotNil(t, cmd)
	result := cmd(a, r, briefTestRoot(t))
	m, _, _ = m.Apply(result)
	require.Equal(t, Ready, m.Phase)
	priorRevision := m.Revision
	m = m.EditBrief("Edited")
	require.Equal(t, Draft, m.Phase)
	require.Greater(t, m.Revision, priorRevision)
	require.Equal(t, "Edited", m.Original)
	stale := Result{Generation: m.Generation - 1, Revision: priorRevision, ProjectID: 7, ChangeID: 21, Step: Run, Output: Output{ReadyForSpec: true}}
	updated, _, ok := m.Apply(stale)
	require.False(t, ok)
	require.Equal(t, Draft, updated.Phase)
}

func TestP804UnchangedRewriteRevalidatesBeforeReady(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		change     func(*briefAPI)
	}{
		{name: "brief changed", want: "current brief changed", change: func(a *briefAPI) { a.docs[0].Body = "Other revision" }},
		{name: "spec removed", want: "brief and spec", change: func(a *briefAPI) { a.config.ChangeDocs = []string{"brief"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := validAPI()
			a.change = dto.Change{ID: 21, ProjectID: 7}
			a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Original", Current: true}}
			m := Existing(7, 21)
			m, _ = doStep(t, m, a, nil, Preflight)
			r := &briefRunner{results: []Output{{RewrittenBrief: "Original", Questions: []Question{}, Unresolved: []string{}, ReadyForSpec: true}}}
			m, _ = doStep(t, m, a, r, Run)
			tc.change(a)
			m, cmd := m.Approve(context.Background())
			require.NotNil(t, cmd)
			require.Equal(t, Verify, m.Pending)
			result := cmd(a, r, briefTestRoot(t))
			m, step, ok := m.Apply(result)
			require.True(t, ok)
			require.Empty(t, step)
			require.Equal(t, Failed, m.Phase)
			require.Equal(t, Verify, m.FailedStep)
			require.ErrorContains(t, m.Err, tc.want)
			require.Len(t, r.requests, 1)
			require.Empty(t, a.inserts)
		})
	}
}

func TestP806MissingBacklogWrongOwnerAndConflict(t *testing.T) {
	a := validAPI()
	a.config.ChangePhases = []string{"review"}
	m := New(7)
	m, _ = doStep(t, m, a, nil, Preflight)
	require.ErrorContains(t, m.Err, "backlog")
	a = validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 8}
	m = Existing(7, 21)
	m, _ = doStep(t, m, a, nil, Preflight)
	require.ErrorContains(t, m.Err, "selected project")
	a.change.ProjectID = 7
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Before", Current: true}}
	m, _ = doStep(t, m, a, nil, Preflight)
	m = m.EditBrief("After")
	a.docs[0].ID = 99
	m, _ = doStep(t, m, a, nil, Write)
	require.ErrorContains(t, m.Err, "current brief changed")
	require.Empty(t, a.inserts)
}

func TestP806ConflictingCurrentRowsNeverStartRunner(t *testing.T) {
	a := validAPI()
	a.change = dto.Change{ID: 21, ProjectID: 7}
	a.docs = []dto.Document{{ID: 31, RefID: 21, RefTable: "change", DocType: "brief", Body: "Text", Current: true}, {ID: 32, RefID: 21, RefTable: "change", DocType: "spec", Body: "S1", Current: true}, {ID: 33, RefID: 21, RefTable: "change", DocType: "spec", Body: "S2", Current: true}}
	m := Existing(7, 21)
	m, step := doStep(t, m, a, nil, Preflight)
	require.ErrorContains(t, m.Err, "conflicting current spec")
	require.Equal(t, Failed, m.Phase)
	require.Empty(t, step)
}
