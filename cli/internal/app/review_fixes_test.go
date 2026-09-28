package app

import (
	"cli/internal/dto"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArtifactEditorSeedsOriginalDocument(t *testing.T) {
	for _, field := range []detailEditField{detailEditDef, detailEditSpec, detailEditPullRequest} {
		for name, original := range map[string]string{
			"empty":      "",
			"tabs":       "```make\nbuild:\n\tgo build ./...\n```\n",
			"many lines": strings.Repeat("line\n", 10001) + "final line\n",
		} {
			t.Run(string(field)+"/"+name, func(t *testing.T) {
				dir := t.TempDir()
				t.Setenv("TMPDIR", dir)
				m := NewModelWithClient(&fakeClient{})
				m.state = ChangeDetailsState
				m = applyMsg(m, changeLoadedMsg{id: 12, change: dto.Change{ID: "12", Def: original, Spec: original, PR: original}})
				next, cmd := m.beginDetailTextEditor(field)
				require.NotNil(t, cmd)
				require.Empty(t, next.(Model).err)
				files, err := filepath.Glob(filepath.Join(dir, "mch-project-*.md"))
				require.NoError(t, err)
				require.Len(t, files, 1)
				content, err := os.ReadFile(files[0])
				require.NoError(t, err)
				assert.True(t, original == string(content), "editor seed differs: original %d bytes, seed %d bytes", len(original), len(content))
			})
		}
	}
}

func TestArtifactEditorUnchangedExitSkipsPersistence(t *testing.T) {
	for _, field := range []detailEditField{detailEditDef, detailEditSpec, detailEditPullRequest} {
		for _, original := range []string{"", "Types: feature\n\n```make\nbuild:\n\tgo build ./...\n```\n"} {
			t.Run(string(field)+"/"+original, func(t *testing.T) {
				client := &fakeClient{}
				m := NewModelWithClient(client)
				m.state = ChangeDetailsState
				m.detailEditField = field
				m.changeList.Detail = dto.Change{ID: "12", Def: original, Spec: original, PR: original, ChangeTypes: []string{"bugfix"}}
				m = m.setPromptValue(original)
				next, cmd := m.Update(editorFinishedMsg{source: ChangeDetailsState, original: original, content: original})
				require.NotNil(t, cmd)
				assert.IsType(t, tea.ClearScreen(), cmd(), "unchanged editor exit must only redraw")
				got := next.(Model)
				assert.Equal(t, ChangeDetailsState, got.state)
				assert.Equal(t, m.changeList.Detail, got.changeList.Detail)
				assert.Empty(t, got.detailEditField)
				assert.Empty(t, got.input.Value())
				assert.Empty(t, got.err)
				assert.Equal(t, "unchanged", got.status)
				assert.Empty(t, client.requestOrder)
			})
		}
	}
}

func TestChangeCreateRetainsCommittedChangeAfterTypeFailure(t *testing.T) {
	cause := errors.New("type rejected")
	created := dto.Change{ID: "12", Title: "Created", Def: "Body"}
	client := &fakeClient{createdChange: created, changeTypesUpdateErr: cause}
	m := NewModelWithClient(client)
	m.currentProject = dto.Option{ID: "7"}
	m.state = ChangeCreateState
	m = m.setPromptValue("# Created\n\nTypes: feature\n\nBody")
	next, cmd := m.submitPrompt()
	require.NotNil(t, cmd)
	msg := cmd().(changeSavedMsg)
	require.NoError(t, msg.err)
	require.ErrorIs(t, msg.reloadErr, cause)
	m = applyMsg(next.(Model), msg)
	assert.Equal(t, ChangeDetailsState, m.state)
	assert.Equal(t, created, m.changeList.Detail)
	assert.Contains(t, m.err, "change created; type update failed: type rejected")
	assert.Empty(t, m.input.Value())
	assert.Equal(t, []string{"change/create", "change/update-change-types"}, client.requestOrder)

	// A repeated save cannot replay creation after the committed result.
	next, cmd = m.executeCommand("/save")
	require.Nil(t, cmd)
	assert.Equal(t, ChangeDetailsState, next.(Model).state)
	assert.Equal(t, 1, client.changeCreateCalls)
	assert.Zero(t, client.changeGetCalls)
}

func TestPromptSubmissionPreservesSlashPrefixedData(t *testing.T) {
	for _, state := range []State{TestCaseCreateState, TestCaseUpdateState, ProjectCreateState, ProjectUpdateState, ChangeDetailsState} {
		t.Run(string(state), func(t *testing.T) {
			for _, content := range []string{"/api/v1/health returns 200", "/cancel returns to the previous screen", "/unknown", "/quit"} {
				t.Run(content, func(t *testing.T) {
					client := &fakeClient{gotChange: dto.Change{ID: "12"}}
					m := NewModelWithClient(client)
					m.state = state
					m.changeList.Detail = dto.Change{ID: "12"}
					m.projectList.Detail = dto.Project{ID: "7"}
					m.activeTestCase = dto.TestCase{ID: "31"}
					if state == ChangeDetailsState {
						m.detailEditField = detailEditTitle
					}
					m = m.setPromptValue(content)
					next, save := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
					require.Empty(t, next.(Model).err)
					require.NotNil(t, save)
					_ = save()
					switch state {
					case TestCaseCreateState:
						assert.Equal(t, []dto.TestCase{{ChangeID: "12", Scenario: content}}, client.testCaseCreateInputs)
					case TestCaseUpdateState:
						assert.Equal(t, []dto.TestCase{{ID: "31", Scenario: content}}, client.testCaseUpdateInputs)
					case ProjectCreateState:
						assert.Equal(t, []string{content}, client.createNames)
					case ProjectUpdateState:
						assert.Equal(t, []string{content}, client.updateNames)
					case ChangeDetailsState:
						assert.Equal(t, []string{content}, client.changeTitleUpdates)
					}
				})
			}
		})
	}
}

func TestPromptSubmissionDispatchesRecognizedFormCommands(t *testing.T) {
	for _, state := range []State{ChangeCreateState, ChangeUpdateState, TestCaseCreateState, TestCaseUpdateState, ProjectCreateState, ProjectUpdateState} {
		t.Run(string(state), func(t *testing.T) {
			client := &fakeClient{}
			m := NewModelWithClient(client)
			m.state = state
			m.changeList.Detail = dto.Change{ID: "12"}
			m.projectList.Detail = dto.Project{ID: "7"}
			m = m.setPromptValue(" /cancel ")
			next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			assert.Empty(t, next.(Model).err)
			assert.NotEqual(t, state, next.(Model).state)
			assert.Empty(t, next.(Model).input.Value())
			assert.Empty(t, client.requestOrder)
		})
	}
	m := NewModelWithClient(&fakeClient{})
	m = m.setPromptValue("/unknown")
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
	assert.Equal(t, "unknown command: /unknown", next.(Model).err)
}

func TestEditorSubmissionPreservesSlashPrefixedData(t *testing.T) {
	for _, state := range []State{TestCaseCreateState, TestCaseUpdateState, ProjectCreateState, ProjectUpdateState, ChangeDetailsState} {
		t.Run(string(state), func(t *testing.T) {
			for _, content := range []string{"/api/v1/health returns 200\nsecond line", "/cancel"} {
				client := &fakeClient{gotChange: dto.Change{ID: "12"}}
				m := NewModelWithClient(client)
				m.state = state
				m.changeList.Detail = dto.Change{ID: "12"}
				m.projectList.Detail = dto.Project{ID: "7"}
				m.activeTestCase = dto.TestCase{ID: "31"}
				if state == ChangeDetailsState {
					m.detailEditField = detailEditTitle
				}
				updated, cmd := m.Update(editorFinishedMsg{source: state, content: content})
				require.NotNil(t, cmd)
				got := updated.(Model)
				assert.Empty(t, got.err)
				assert.Equal(t, state, got.state)
				assert.Equal(t, content, got.input.Value())

				// Execute the data submission directly; the program suite covers
				// Bubble Tea's editor/redraw command sequence.
				_, save := m.submitPromptValue(content)
				require.NotNil(t, save)
				_ = save()
				switch state {
				case TestCaseCreateState:
					assert.Equal(t, []dto.TestCase{{ChangeID: "12", Scenario: content}}, client.testCaseCreateInputs)
				case TestCaseUpdateState:
					assert.Equal(t, []dto.TestCase{{ID: "31", Scenario: content}}, client.testCaseUpdateInputs)
				case ProjectCreateState:
					assert.Equal(t, []string{content}, client.createNames)
				case ProjectUpdateState:
					assert.Equal(t, []string{content}, client.updateNames)
				case ChangeDetailsState:
					assert.Equal(t, []string{content}, client.changeTitleUpdates)
				}
			}
		})
	}
}

func TestArtifactDraftSurvivesFailedSaveAndRetry(t *testing.T) {
	for _, field := range []detailEditField{detailEditDef, detailEditSpec, detailEditPullRequest} {
		for name, draft := range map[string]string{
			"tabs":       "```make\nbuild:\n\tgo build ./...\n```\n",
			"many lines": strings.Repeat("line\n", 10001) + "final line\n",
		} {
			t.Run(string(field)+"/"+name, func(t *testing.T) {
				t.Setenv("TMPDIR", t.TempDir())
				client := &fakeClient{changeUpdateErr: errors.New("offline"), gotChange: dto.Change{ID: "12"}}
				m := NewModelWithClient(client)
				m.state, m.detailEditField = ChangeDetailsState, field
				m.changeList.Detail = dto.Change{ID: "12", Def: "persisted", Spec: "persisted", PR: "persisted"}
				next, cmd := m.Update(editorFinishedMsg{source: m.state, original: "persisted", content: draft})
				m = applyCommand(next.(Model), cmd)
				require.Equal(t, "save failed", m.status)
				// Cursor movement must not turn the textarea preview into the draft.
				m, _ = sendKey(m, tea.KeyLeft)
				next, cmd = m.openPromptEditor(m.state)
				require.NotNil(t, cmd)
				files, err := filepath.Glob(filepath.Join(os.Getenv("TMPDIR"), "mch-project-*.md"))
				require.NoError(t, err)
				require.Len(t, files, 1)
				assert.True(t, draft == readTestFile(t, files[0]), "reopened editor must receive exact draft bytes")
				// Exiting the reopened editor unchanged must retry, not discard.
				next, cmd = next.(Model).Update(editorFinishedMsg{source: m.state, original: draft, content: draft})
				m = applyCommand(next.(Model), cmd)
				require.Equal(t, "save failed", m.status)
				require.Equal(t, field, m.detailEditField)
				client.changeUpdateErr = nil
				m, cmd = sendKey(m, tea.KeyEnter)
				require.NotNil(t, cmd)
				m = applyCommand(m, cmd)
				require.Equal(t, "save", m.status)
				assert.Empty(t, m.input.Value())
				var submissions []string
				switch field {
				case detailEditDef:
					submissions = client.changeDefUpdates
				case detailEditSpec:
					submissions = client.changeSpecUpdates
				case detailEditPullRequest:
					submissions = client.changePRUpdates
				}
				require.Len(t, submissions, 3)
				for _, submitted := range submissions {
					assert.True(t, draft == submitted, "every attempt must preserve raw draft bytes")
				}
			})
		}
	}
}

func TestEditorRetryKeepsLiteralData(t *testing.T) {
	for _, state := range []State{TestCaseCreateState, TestCaseUpdateState, ProjectCreateState, ProjectUpdateState, ChangeDetailsState} {
		for _, content := range []string{"/api/v1/health returns 200", "/cancel"} {
			t.Run(string(state)+content, func(t *testing.T) {
				failure := errors.New("offline")
				client := &fakeClient{changeUpdateErr: failure, createErr: failure, updateErr: failure, gotChange: dto.Change{ID: "12"}, createdProject: dto.Project{ID: "7"}}
				m := NewModelWithClient(client)
				m.state = state
				m.changeList.Detail = dto.Change{ID: "12"}
				m.projectList.Detail = dto.Project{ID: "7"}
				m.activeTestCase = dto.TestCase{ID: "31"}
				if state == ChangeDetailsState {
					m.detailEditField = detailEditTitle
				}
				next, _ := m.Update(editorFinishedMsg{source: state, content: content})
				m = next.(Model)
				// Execute the save separately from the Bubble Tea redraw sequence.
				_, cmd := m.submitPromptValue(content)
				m = applyCommand(m, cmd)
				require.Equal(t, "save failed", m.status)
				client.changeUpdateErr, client.createErr, client.updateErr = nil, nil, nil
				m, cmd = sendKey(m, tea.KeyEnter)
				require.NotNil(t, cmd, "Enter retries literal editor data")
				m = applyCommand(m, cmd)
				require.Equal(t, "save", m.status)
				switch state {
				case TestCaseCreateState:
					require.Len(t, client.testCaseCreateInputs, 2)
					assert.Equal(t, content, client.testCaseCreateInputs[1].Scenario)
				case TestCaseUpdateState:
					require.Len(t, client.testCaseUpdateInputs, 2)
					assert.Equal(t, content, client.testCaseUpdateInputs[1].Scenario)
				case ProjectCreateState:
					assert.Equal(t, []string{content, content}, client.createNames)
				case ProjectUpdateState:
					assert.Equal(t, []string{content, content}, client.updateNames)
				case ChangeDetailsState:
					assert.Equal(t, []string{content, content}, client.changeTitleUpdates)
				}
			})
		}
	}
}

func TestEditorDraftEditingAndDiscard(t *testing.T) {
	for _, draft := range []string{"/cancel", "", "\tkeep raw\n", strings.Repeat("x", 300)} {
		t.Run(draft[:min(len(draft), 10)], func(t *testing.T) {
			m := NewModelWithClient(&fakeClient{})
			m.state = TestCaseCreateState
			m.changeList.Detail = dto.Change{ID: "12"}
			next, _ := m.Update(editorFinishedMsg{source: m.state, content: draft})
			m = next.(Model)
			lossless := m.input.Value() == draft
			m, _ = sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
			m = m.insertPromptNewline()
			m = m.insertPromptLiteral("more")
			if lossless {
				assert.Equal(t, draft+"/\nmore", m.promptValue())
			} else {
				assert.Equal(t, draft, m.promptValue())
				assert.Contains(t, m.err, "Ctrl+E")
			}
			m, _ = sendKey(m, tea.KeyCtrlC)
			assert.Nil(t, m.editorDraft)
			assert.Empty(t, m.input.Value())
			// After explicit discard, slash commands regain their usual meaning.
			m = m.setPromptValue("/cancel")
			m, _ = sendKey(m, tea.KeyEnter)
			assert.Equal(t, ChangeDetailsState, m.state)
		})
	}
}

func TestEditorDraftAsyncPasteRetry(t *testing.T) {
	// Exercise the textarea's real asynchronous paste message with an owned
	// clipboard command, without reading or changing the desktop clipboard.
	bin := t.TempDir()
	for _, name := range []string{"xclip", "xsel", "wl-paste", "termux-clipboard-get", "powershell.exe", "pbpaste"} {
		output := "pasted"
		if name == "powershell.exe" {
			output += "\r\n"
		}
		require.NoError(t, os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nprintf '"+output+"'\n"), 0o755))
	}
	t.Setenv("PATH", bin)
	unsupported := clipboard.Unsupported
	clipboard.Unsupported = false
	t.Cleanup(func() { clipboard.Unsupported = unsupported })

	for _, draft := range []string{"original", "/cancel", "\tkeep raw"} {
		t.Run(draft, func(t *testing.T) {
			client := &fakeClient{changeUpdateErr: errors.New("offline"), gotChange: dto.Change{ID: "12"}}
			m := NewModelWithClient(client)
			m.state = TestCaseCreateState
			m.changeList.Detail = dto.Change{ID: "12"}
			// A paste requested before editor completion may arrive afterward.
			_, paste := sendKey(m, tea.KeyCtrlV)
			require.NotNil(t, paste)
			next, _ := m.Update(editorFinishedMsg{source: m.state, content: draft})
			m = next.(Model)
			_, save := m.submitPromptValue(draft)
			m = applyCommand(m, save)
			require.Equal(t, "save failed", m.status)
			representable := m.input.Value() == draft
			if representable {
				m, paste = sendKey(m, tea.KeyCtrlV)
				require.NotNil(t, paste)
			}
			m = applyMsg(m, paste())
			want := draft
			if representable {
				want += "pasted"
				require.Equal(t, want, m.input.Value())
				require.Equal(t, want, m.promptValue())
				m, _ = sendKeyMsg(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("!")})
				want += "!"
				require.Empty(t, m.err)
			}
			require.Equal(t, want, m.promptValue())
			client.changeUpdateErr = nil
			m, save = sendKey(m, tea.KeyEnter)
			m = applyCommand(m, save)
			require.Equal(t, "save", m.status)
			require.Len(t, client.testCaseCreateInputs, 2)
			assert.Equal(t, want, client.testCaseCreateInputs[1].Scenario)
		})
	}
}

func TestDocumentEditorRequiresSuccessfulDetailLoad(t *testing.T) {
	for _, field := range []detailEditField{detailEditDef, detailEditSpec, detailEditPullRequest, detailEditPRUrl} {
		t.Run(string(field), func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("TMPDIR", dir)
			original := "Existing document\n\tkeep all bytes\n"
			client := &fakeClient{gotChange: dto.Change{ID: "12", Def: original, Spec: original, PR: original, PRUrl: original}}
			m := NewModelWithClient(client)
			m.state = ChangesListState
			m.changeList = m.changeList.WithRows([]dto.Change{{ID: "12", Title: "List row"}, {ID: "13", Title: "Other row"}})
			m, load := sendKey(m, tea.KeyEnter)
			require.NotNil(t, load)
			assertBlocked := func() {
				t.Helper()
				next, cmd := m.beginDetailTextEditor(field)
				require.Nil(t, cmd, "must not open an editor before successful detail loading")
				got := next.(Model)
				assert.Contains(t, got.err, "Load change details before editing")
				assert.Empty(t, got.detailEditField)
				files, err := os.ReadDir(dir)
				require.NoError(t, err)
				assert.Empty(t, files, "blocked edits must not create editor files")
				assert.Zero(t, client.changeSpecUpdateCalls)
				assert.Zero(t, client.changeDefUpdateCalls)
				assert.Zero(t, client.changePRUpdateCalls)
				assert.Zero(t, client.changePRUrlUpdateCalls)
			}
			// Hold the initial asynchronous result, then fail it.
			assertBlocked()
			m = applyMsg(m, changeLoadedMsg{id: 12, err: errors.New("detail unavailable")})
			assert.Equal(t, "detail unavailable", m.err)
			assertBlocked()
			// A successful retry unlocks editing with the exact loaded bytes.
			m = applyMsg(m, load())
			next, cmd := m.beginDetailTextEditor(field)
			require.NotNil(t, cmd)
			assert.Equal(t, field, next.(Model).detailEditField)
			files, err := filepath.Glob(filepath.Join(dir, "mch-project-*.md"))
			require.NoError(t, err)
			require.Len(t, files, 1)
			content, err := os.ReadFile(files[0])
			require.NoError(t, err)
			assert.Equal(t, original, string(content))
			require.NoError(t, os.Remove(files[0]))
			// Readiness must not carry over to the next selected change.
			m.state = ChangesListState
			m.changeList.Selected = 1
			m, load = sendKey(m, tea.KeyEnter)
			require.NotNil(t, load)
			m = applyMsg(m, changeLoadedMsg{id: 12, change: client.gotChange})
			assertBlocked()
		})
	}
}
