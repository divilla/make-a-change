package app

import (
	"cli/internal/configurations"
	"cli/internal/dto"
	"cli/internal/styles"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	httpclient "cli/pkg/client"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func p7Row() dto.BackendConfig {
	return dto.BackendConfig{Slug: "program", ProjectDocs: []string{"readme"}, EpicDocs: []string{"brief"}, ChangeDocs: []string{"brief", "spec"}, ChangePhases: []string{"todo"}, ChangeColors: []string{"12"}, ChangeTypes: []string{"feature"}}
}

func TestP702ConfigScreenNavigationAndAllCatalogs(t *testing.T) {
	row := p7Row()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/config/list":
			_ = json.NewEncoder(w).Encode([]dto.BackendConfig{row})
		case "/api/v1/config/details":
			_ = json.NewEncoder(w).Encode(row)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	m := NewModelWithClient(httpclient.NewHTTPClient(server.URL))
	var cmd tea.Cmd
	m, cmd = sendCommand(m, "/backend-configs")
	m = applyCommand(m, cmd)
	assert.Equal(t, BackendConfigListState, m.state)
	assert.Contains(t, stripANSI(m.View()), "program")
	m, cmd = sendKey(m, tea.KeyEnter)
	m = applyCommand(m, cmd)
	assert.Equal(t, BackendConfigDetailsState, m.state)
	view := stripANSI(m.View())
	assert.Contains(t, view, "Configuration: program")
	for _, field := range configurations.FieldNames[1:] {
		assert.Contains(t, view, field)
	}
	assert.Contains(t, view, "[\"brief\",\"spec\"]")
	m, _ = sendCommand(m, "/return")
	assert.Equal(t, BackendConfigListState, m.state)
}

func TestP703SelectedProjectCatalogRefreshAndScope(t *testing.T) {
	row := p7Row()
	row.ChangePhases = []string{"newphase"}
	reads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/config/update":
			w.WriteHeader(204)
		case "/api/v1/config/list":
			_ = json.NewEncoder(w).Encode([]dto.BackendConfig{row})
		case "/api/v1/config/details":
			_ = json.NewEncoder(w).Encode(row)
		case "/api/v1/project/config":
			reads++
			_ = json.NewEncoder(w).Encode(row)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	m := newModelWithConfig(httpclient.NewHTTPClient(server.URL), appConfig{ProjectID: 7})
	m.selectedConfigSlug = "program"
	m.optionCatalog = optionCatalog{config: dto.ProjectConfig{Slug: "program", ChangePhases: []string{"old"}}, loaded: true}
	m.state = BackendConfigFormState
	m.configurations = configurations.Model{Detail: p7Row(), DetailLoaded: true}.OpenForm(true)
	m.configurations.Raw[4] = `["newphase"]`
	m = m.setPromptValue(m.configurations.Raw[m.configurations.Field])
	var cmd tea.Cmd
	var next tea.Model
	next, cmd = m.beginConfiguration(configurations.Update, "program")
	m = next.(Model)
	r := cmd().(configurations.Result)
	updated, follow := m.Update(r)
	m = updated.(Model)
	require.NotNil(t, follow)
	m = applyCommand(m, follow)
	assert.Equal(t, 1, reads)
	assert.Equal(t, []string{"newphase"}, m.optionCatalog.config.ChangePhases)
	old := optionCatalogLoadedMsg{id: 7, generation: m.catalogGeneration - 1, config: dto.ProjectConfig{Slug: "program", ChangePhases: []string{"stale"}}}
	m = applyMsg(m, old)
	assert.Equal(t, []string{"newphase"}, m.optionCatalog.config.ChangePhases)
}

func TestP703ConfigurationExitCancelsCatalogRefreshAndIgnoresLateResult(t *testing.T) {
	for _, exit := range []string{"return", "quit"} {
		t.Run(exit, func(t *testing.T) {
			started := make(chan struct{})
			canceled := make(chan struct{})
			release := make(chan struct{})
			var reads atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/project/config" {
					t.Errorf("unexpected route %s", r.URL.Path)
					return
				}
				var body map[string]any
				_ = json.NewDecoder(r.Body).Decode(&body)
				if reads.Add(1) > 1 {
					row := p7Row()
					row.ChangeDocs = []string{"new-doc"}
					row.ChangePhases = []string{"new-phase"}
					row.ChangeTypes = []string{"new-type"}
					_ = json.NewEncoder(w).Encode(row)
					return
				}
				close(started)
				select {
				case <-r.Context().Done():
					close(canceled)
				case <-release:
				}
			}))
			defer server.Close()
			defer close(release)
			m := newModelWithConfig(httpclient.NewHTTPClient(server.URL), appConfig{ProjectID: 7})
			m.state = BackendConfigListState
			m.selectedConfigSlug = "program"
			m, cmd := m.beginConfigurationCatalogRefresh()
			result := make(chan optionCatalogLoadedMsg, 1)
			go func() { result <- cmd().(optionCatalogLoadedMsg) }()
			select {
			case <-started:
			case <-time.After(2 * time.Second):
				t.Fatal("catalog refresh did not start")
			}
			generation := m.catalogGeneration
			var navigation tea.Cmd
			if exit == "return" {
				m, navigation = sendKey(m, tea.KeyEsc)
			} else {
				next, cmd := m.requestQuit()
				m = next.(Model)
				navigation = cmd
			}
			assert.Nil(t, m.configCatalogCancel)
			assert.Greater(t, m.catalogGeneration, generation)
			select {
			case <-canceled:
			case <-time.After(2 * time.Second):
				t.Fatal("catalog refresh was not canceled")
			}
			beforeStatus, beforeErr := m.status, m.err
			m = applyMsg(m, <-result)
			m = applyMsg(m, optionCatalogLoadedMsg{id: 7, generation: generation, config: dto.ProjectConfig{Slug: "program", ChangePhases: []string{"stale"}}})
			assert.Equal(t, beforeStatus, m.status)
			assert.Equal(t, beforeErr, m.err)
			assert.False(t, m.optionCatalog.loaded)
			if exit == "return" {
				require.NotNil(t, navigation)
				m = applyCommand(m, navigation)
				assert.Equal(t, MainState, m.state)
				assert.Equal(t, int32(2), reads.Load())
				assert.True(t, m.optionCatalog.loaded)
				assert.Equal(t, []string{"new-doc"}, m.optionCatalog.config.ChangeDocs)
				assert.Equal(t, []string{"new-phase"}, m.optionCatalog.config.ChangePhases)
				assert.Equal(t, []string{"new-type"}, m.optionCatalog.config.ChangeTypes)
			} else {
				assert.Equal(t, int32(1), reads.Load())
			}
		})
	}
}

func TestP703UpdateRefreshesCatalogWhileProjectIdentityLoads(t *testing.T) {
	row := p7Row()
	row.ChangePhases = []string{"old"}
	reads := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/project/details":
			_ = json.NewEncoder(w).Encode(dto.Project{ID: 7, Config: "program"})
		case "/api/v1/project/config":
			reads++
			_ = json.NewEncoder(w).Encode(row)
		case "/api/v1/config/update":
			row.ChangePhases = []string{"new"}
			w.WriteHeader(http.StatusNoContent)
		case "/api/v1/config/list":
			_ = json.NewEncoder(w).Encode([]dto.BackendConfig{row})
		case "/api/v1/config/details":
			_ = json.NewEncoder(w).Encode(row)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	m := newModelWithConfig(httpclient.NewHTTPClient(server.URL), appConfig{ProjectID: 7})
	startupCatalog := optionCatalogCommand(m.ctx, m.client, 7, m.catalogGeneration)().(optionCatalogLoadedMsg)
	startupDetails := currentProjectCommand(m.ctx, m.client, 7, m.selectionGeneration)().(currentProjectLoadedMsg)
	require.Empty(t, m.selectedConfigSlug)
	m.state = BackendConfigFormState
	m.configurations = configurations.Model{Detail: p7Row(), DetailLoaded: true}.OpenForm(true)
	m.configurations.Raw[4] = `["new"]`
	next, write := m.beginConfiguration(configurations.Update, "program")
	m = next.(Model)
	updated, follow := m.Update(write())
	m = updated.(Model)
	require.NotNil(t, follow)
	m = applyCommand(m, follow)
	require.Equal(t, 2, reads)
	assert.Equal(t, []string{"new"}, m.optionCatalog.config.ChangePhases)
	m = applyMsg(m, startupCatalog)
	assert.Equal(t, []string{"new"}, m.optionCatalog.config.ChangePhases)
	m = applyMsg(m, startupDetails)
	assert.Equal(t, "program", m.selectedConfigSlug)
	assert.Equal(t, "updated configuration program", m.configurations.Committed)
	m.configurations.Committed = ""
	m.selectedConfigSlug = ""
	m = applyMsg(m, optionCatalogLoadedMsg{id: 7, generation: m.catalogGeneration, err: errors.New("later catalog failure")})
	assert.Equal(t, "option catalog failed", m.status)
}

func TestP702ReentryClearsCanceledConfigurationModals(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/config/list" {
			t.Errorf("unexpected route %s", r.URL.Path)
			return
		}
		_ = json.NewEncoder(w).Encode([]dto.BackendConfig{p7Row()})
	}))
	defer server.Close()
	for _, tc := range []struct {
		name  string
		state State
		op    configurations.Operation
	}{
		{"save", BackendConfigFormState, configurations.Update},
		{"delete", BackendConfigDeleteState, configurations.Delete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModelWithClient(httpclient.NewHTTPClient(server.URL))
			m.state = tc.state
			m.configurations = configurations.Model{Detail: p7Row(), DetailLoaded: true}
			if tc.op == configurations.Update {
				m.configurations = m.configurations.OpenForm(true)
			} else {
				m.configurations.Confirm = true
			}
			next, pending := m.beginConfiguration(tc.op, "program")
			m = next.(Model)
			require.NotNil(t, pending)
			m, _ = sendKey(m, tea.KeyEsc)
			require.Equal(t, MainState, m.state)
			m, list := sendCommand(m, "/backend-configs")
			require.Equal(t, BackendConfigListState, m.state)
			assert.False(t, m.configurations.Form)
			assert.False(t, m.configurations.Confirm)
			assert.False(t, m.configurations.Editing)
			m = applyCommand(m, list)
			m = applyMsg(m, pending())
			view := stripANSI(m.View())
			assert.Contains(t, view, "program")
			assert.NotContains(t, view, "Delete configuration program?")
			assert.NotContains(t, view, "project_docs:")
		})
	}
}

func TestP702CreateSlugAcceptsLeadingSlashAndKeepsCommands(t *testing.T) {
	m := NewModelWithClient(httpclient.NewHTTPClient("http://backend.test"))
	m.state = BackendConfigListState
	m, _ = sendCommand(m, "/new-config")
	m, _ = sendRune(m, '/')
	assert.Equal(t, "/", m.input.Value())
	assert.Empty(t, m.dropdown.kind)
	for _, r := range "team" {
		m, _ = sendRune(m, r)
	}
	m, _ = sendKey(m, tea.KeyTab)
	assert.Equal(t, "/team", m.configurations.Raw[0])
	row, err := m.configurations.ParseDraft()
	require.NoError(t, err)
	assert.Equal(t, "/team", row.Slug)
	m, _ = sendKey(m, tea.KeyCtrlG)
	assert.Equal(t, dropdownCommand, m.dropdown.kind)
	assert.Contains(t, stripANSI(m.View()), "    save")
}

func TestP703FailedWriteDraftAndBusyDeduplication(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls++; http.Error(w, "write rejected", 500) }))
	defer server.Close()
	m := NewModelWithClient(httpclient.NewHTTPClient(server.URL))
	m.state = BackendConfigFormState
	m.configurations = configurations.Model{}.OpenForm(false)
	m.configurations.Raw[0] = "draft"
	m = m.setPromptValue("draft")
	var cmd tea.Cmd
	m, cmd = sendKey(m, tea.KeyCtrlS)
	require.NotNil(t, cmd)
	_, extra := sendKey(m, tea.KeyCtrlS)
	assert.Nil(t, extra)
	m = applyMsg(m, cmd())
	assert.Equal(t, 1, calls)
	assert.Equal(t, "draft", m.configurations.Raw[0])
	assert.Equal(t, BackendConfigFormState, m.state)
	assert.Contains(t, m.err, "write rejected")
}

func TestP705LocalConfigAndProjectConfigRemainDistinct(t *testing.T) {
	m := newModelWithConfig(httpclient.NewHTTPClient("http://backend.test"), appConfig{BackendURL: "http://backend.test", ConfigPath: "/repo/.mch/config.yaml", ProjectID: 7})
	m, _ = sendCommand(m, "/config")
	assert.Equal(t, ConfigState, m.state)
	assert.Contains(t, stripANSI(m.View()), "config_path: /repo/.mch/config.yaml")
	assert.NotContains(t, stripANSI(m.View()), "project_docs:")
	assert.True(t, commandAllowed(MainState, "/backend-configs"))
	assert.False(t, commandAllowed(ConfigState, "/backend-configs"))
	assert.True(t, commandAllowed(ProjectDetailsState, "/project-config"))
}

func TestP706ScenarioManifestIncludesConfigHealthAndPTY(t *testing.T) {
	data, err := os.ReadFile("../../scripts/terminal-scenarios.json")
	require.NoError(t, err)
	var manifest struct {
		Program struct {
			Tests []string `json:"tests"`
		} `json:"program"`
		PTY struct {
			Tests []string `json:"tests"`
		} `json:"pty"`
	}
	require.NoError(t, json.Unmarshal(data, &manifest))
	for _, name := range []string{"TestCLIProgramHealthRoutesAndDegradedStatus", "TestCLIProgramConfigurationCRUDAndCatalogRefresh", "TestCLIProgramConfigurationSelectedProjectCatalogRefresh", "TestCLIProgramConfigurationStaleResponseIsolation"} {
		assert.Contains(t, manifest.Program.Tests, name)
	}
	assert.Contains(t, manifest.PTY.Tests, "TestShellNavigationEditorAndScrolling")
	assert.NotContains(t, strings.Join(manifest.Program.Tests, ","), "TestP701")
}

func TestP703SelectedProjectCatalogFailureAndRetry(t *testing.T) {
	row := p7Row()
	row.ChangePhases = []string{"restored"}
	row.ChangeTypes = []string{"newtype"}
	row.ChangeDocs = []string{"custom"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/config/details", "/api/v1/project/config":
			_ = json.NewEncoder(w).Encode(row)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	m := newModelWithConfig(httpclient.NewHTTPClient(server.URL), appConfig{ProjectID: 7})
	m.state = BackendConfigDetailsState
	m.selectedConfigSlug = "program"
	m.configurations = configurations.Model{Detail: p7Row(), DetailLoaded: true, RequestedSlug: "program", Committed: "updated configuration program", CommittedSlug: "program", CommittedOperation: configurations.Update}
	m.optionCatalog = optionCatalog{config: dto.ProjectConfig{Slug: "program", ChangePhases: []string{"obsolete"}}, loaded: true}
	m = applyMsg(m, optionCatalogLoadedMsg{id: 7, generation: m.catalogGeneration, err: errors.New("catalog unavailable")})
	assert.False(t, m.optionCatalog.loaded)
	assert.Error(t, m.optionCatalog.err)
	assert.Contains(t, m.err, "project catalog refresh failed")
	next, cmd := m.configurationCommand(BackendConfigDetailsState, "/retry")
	m = next.(Model)
	m = applyCommand(m, cmd)
	assert.True(t, m.optionCatalog.loaded)
	assert.Nil(t, m.optionCatalog.err)
	assert.Equal(t, []string{"restored"}, m.optionCatalog.config.ChangePhases)
	assert.Equal(t, []string{"newtype"}, m.optionCatalog.config.ChangeTypes)
	assert.Equal(t, []string{"custom"}, m.optionCatalog.config.ChangeDocs)
	assert.Empty(t, m.err)
}

func TestP702DetailRetryUsesSelectedSlug(t *testing.T) {
	first := p7Row()
	first.Slug = "a"
	second := p7Row()
	second.Slug = "b"
	requested := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/config/list":
			_ = json.NewEncoder(w).Encode([]dto.BackendConfig{first, second})
		case "/api/v1/config/details":
			var body struct {
				Slug string `json:"slug"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			requested = append(requested, body.Slug)
			if len(requested) == 1 {
				http.Error(w, "try again", http.StatusServiceUnavailable)
			} else {
				_ = json.NewEncoder(w).Encode(second)
			}
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	m := NewModelWithClient(httpclient.NewHTTPClient(server.URL))
	var cmd tea.Cmd
	m, cmd = sendCommand(m, "/backend-configs")
	m = applyCommand(m, cmd)
	m, _ = sendKey(m, tea.KeyDown)
	m, cmd = sendKey(m, tea.KeyEnter)
	m = applyCommand(m, cmd)
	assert.False(t, m.configurations.DetailLoaded)
	assert.Equal(t, "b", m.configurations.RequestedSlug)
	m, cmd = sendCommand(m, "/retry")
	m = applyCommand(m, cmd)
	assert.True(t, m.configurations.DetailLoaded)
	assert.Equal(t, "b", m.configurations.Detail.Slug)
	assert.Equal(t, []string{"b", "b"}, requested)
}

func TestP704HealthRouteSwitchClearsPriorFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/health", r.URL.Path)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"degraded","api":"ok","database":"error","error":"database unavailable"}`))
	}))
	defer server.Close()
	m := NewModelWithClient(httpclient.NewHTTPClient(server.URL))
	m.state = HealthState
	m.health = m.health.SelectRoute("/api/v1/health")
	m.err = "old route failed"
	next, cmd := m.healthCommand("/health-legacy")
	m = next.(Model)
	assert.Empty(t, m.err)
	m = applyCommand(m, cmd)
	assert.Equal(t, "/api/health", m.health.Route)
	assert.Empty(t, m.err)
	assert.NotContains(t, stripANSI(m.View()), "old route failed")
	assert.Contains(t, stripANSI(m.View()), "HTTP 503")
}

func TestP702ConfigurationDropdownKeysReachActions(t *testing.T) {
	for _, tc := range []struct {
		name      string
		start     State
		down      int
		wantState State
		wantLabel string
	}{
		{"new", BackendConfigListState, 0, BackendConfigFormState, "new backend configuration"},
		{"edit", BackendConfigDetailsState, 0, BackendConfigFormState, "edit backend configuration"},
		{"delete", BackendConfigDetailsState, 1, BackendConfigDeleteState, "confirm delete program"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModelWithClient(httpclient.NewHTTPClient("http://backend.test"))
			m.state = tc.start
			m.configurations = configurations.Model{Detail: p7Row(), DetailLoaded: true}
			m, _ = sendRune(m, '/')
			require.Equal(t, dropdownCommand, m.dropdown.kind)
			for range tc.down {
				m, _ = sendKey(m, tea.KeyDown)
			}
			m, _ = sendKey(m, tea.KeyEnter)
			assert.Equal(t, tc.wantState, m.state)
			assert.Equal(t, tc.wantLabel, m.status)
			assert.Empty(t, m.dropdown.kind)
		})
	}
}

func TestP704HealthDropdownKeysSelectBothRoutes(t *testing.T) {
	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.Path)
		_, _ = w.Write([]byte(`{"status":"ok","api":"ok","database":"ok"}`))
	}))
	defer server.Close()
	m := NewModelWithClient(httpclient.NewHTTPClient(server.URL))
	m.state = HealthState
	for _, tc := range []struct {
		down  int
		route string
	}{
		{1, "/api/health"},
		{0, "/api/v1/health"},
	} {
		var cmd tea.Cmd
		m, _ = sendRune(m, '/')
		require.Equal(t, dropdownCommand, m.dropdown.kind)
		for range tc.down {
			m, _ = sendKey(m, tea.KeyDown)
		}
		m, cmd = sendKey(m, tea.KeyEnter)
		m = applyCommand(m, cmd)
		assert.Equal(t, tc.route, m.health.Route)
		assert.Contains(t, stripANSI(m.View()), "HTTP 200")
	}
	assert.Equal(t, []string{"/api/health", "/api/v1/health"}, requested)
}

func TestP702ConfigurationDraftCanRevisitEarlierFields(t *testing.T) {
	m := NewModelWithClient(httpclient.NewHTTPClient("http://backend.test"))
	m.state = BackendConfigFormState
	m.configurations = configurations.Model{}.OpenForm(false)
	m.configurations.Raw[0] = "draft"
	m.configurations.Raw[1] = "invalid JSON"
	m.configurations.Field = 6
	m = m.setPromptValue(`["saved last field"]`)
	m, _ = sendKey(m, tea.KeyCtrlS)
	assert.Contains(t, m.err, "project_docs")
	assert.Equal(t, 6, m.configurations.Field)
	m, _ = sendKey(m, tea.KeyTab)
	assert.Equal(t, 0, m.configurations.Field)
	assert.Equal(t, "draft", m.input.Value())
	m, _ = sendKey(m, tea.KeyTab)
	assert.Equal(t, 1, m.configurations.Field)
	assert.Equal(t, "invalid JSON", m.input.Value())
	m = m.setPromptValue(`[]`)
	m, _ = sendKey(m, tea.KeyShiftTab)
	assert.Equal(t, 0, m.configurations.Field)
	assert.Equal(t, "[]", m.configurations.Raw[1])
	m, _ = sendKey(m, tea.KeyShiftTab)
	assert.Equal(t, 6, m.configurations.Field)
	assert.Equal(t, `["saved last field"]`, m.input.Value())
	row, err := m.configurations.ParseDraft()
	require.NoError(t, err)
	assert.Equal(t, []string{}, row.ProjectDocs)
	assert.Equal(t, []string{"saved last field"}, row.ChangeTypes)

	m.configurations = configurations.Model{Detail: p7Row(), DetailLoaded: true}.OpenForm(true)
	m = m.setPromptValue(m.configurations.Raw[1])
	m, _ = sendKey(m, tea.KeyShiftTab)
	assert.Equal(t, 6, m.configurations.Field)
	m, _ = sendKey(m, tea.KeyTab)
	assert.Equal(t, 1, m.configurations.Field)
	assert.Equal(t, "program", m.configurations.Raw[0])
}

func TestP702EditorSlugKeepsExactBytesThroughSaveAndFieldNavigation(t *testing.T) {
	const slug = "part\tone"
	for _, revisit := range []bool{false, true} {
		t.Run(map[bool]string{false: "direct save", true: "revisit slug"}[revisit], func(t *testing.T) {
			var inserted dto.BackendConfig
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/api/v1/config/insert", r.URL.Path)
				require.NoError(t, json.NewDecoder(r.Body).Decode(&inserted))
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]string{"slug": inserted.Slug})
			}))
			defer server.Close()
			m := NewModelWithClient(httpclient.NewHTTPClient(server.URL))
			m.state = BackendConfigFormState
			m.configurations = configurations.Model{}.OpenForm(false)
			m = applyMsg(m, editorFinishedMsg{source: BackendConfigFormState, content: slug})
			require.Equal(t, slug, m.promptValue())
			assert.NotEqual(t, slug, m.input.Value(), "textarea preview is lossy")
			if revisit {
				m, _ = sendKey(m, tea.KeyTab)
				m, _ = sendKey(m, tea.KeyShiftTab)
				require.Equal(t, slug, m.promptValue())
			}
			m, cmd := sendKey(m, tea.KeyCtrlS)
			require.NotNil(t, cmd)
			result := cmd().(configurations.Result)
			require.NoError(t, result.Err)
			assert.Equal(t, slug, inserted.Slug)
			assert.Equal(t, slug, m.configurations.Raw[0])
		})
	}
}

func TestP703ReturnFromFailedDetailRefreshRetriesList(t *testing.T) {
	row := p7Row()
	var listReads, detailReads int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/config/list":
			listReads++
			_ = json.NewEncoder(w).Encode([]dto.BackendConfig{row})
		case "/api/v1/config/details":
			detailReads++
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		default:
			t.Errorf("unexpected route %s", r.URL.Path)
		}
	}))
	defer server.Close()
	m := NewModelWithClient(httpclient.NewHTTPClient(server.URL))
	m.state = BackendConfigDetailsState
	m.configurations = configurations.Model{
		Rows: []dto.BackendConfig{row}, Committed: "updated configuration program",
		CommittedSlug: row.Slug, CommittedOperation: configurations.Update,
		Refresh: configurations.Details, RequestedSlug: row.Slug,
	}
	m, _ = sendCommand(m, "/return")
	require.Equal(t, BackendConfigListState, m.state)
	require.Empty(t, m.configurations.Refresh)
	m, cmd := sendCommand(m, "/retry")
	m = applyCommand(m, cmd)
	assert.Equal(t, 1, listReads)
	assert.Zero(t, detailReads)
	assert.Equal(t, BackendConfigListState, m.state)
	assert.False(t, m.configurations.DetailLoaded)
	assert.Contains(t, stripANSI(m.View()), "> program")
}

func TestP702ConfigurationSelectionStaysVisibleAcrossNavigation(t *testing.T) {
	rows := make([]dto.BackendConfig, 18)
	for i := range rows {
		rows[i] = p7Row()
		rows[i].Slug = "row-" + strconv.Itoa(i)
	}
	rows[2].Slug = strings.Repeat("wide", 40)
	var opened string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/config/details", r.URL.Path)
		var body struct {
			Slug string `json:"slug"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		opened = body.Slug
		_ = json.NewEncoder(w).Encode(rows[12])
	}))
	defer server.Close()
	m := NewModelWithClient(httpclient.NewHTTPClient(server.URL))
	m.state = BackendConfigListState
	m.height = 12
	m.width = 30
	m.configurations.Rows = rows
	for range 12 {
		m, _ = sendKey(m, tea.KeyDown)
		marker := "> row-" + strconv.Itoa(m.configurations.Selected)
		if m.configurations.Selected == 2 {
			marker = "> wide"
		}
		assert.Contains(t, stripANSI(m.View()), marker)
	}
	assert.Positive(t, m.configurations.Offset)
	m, _ = sendKey(m, tea.KeyUp)
	assert.Contains(t, stripANSI(m.View()), "> row-11")
	m, _ = sendKey(m, tea.KeyPgUp)
	assert.Contains(t, stripANSI(m.View()), "> row-5")
	m, _ = sendKey(m, tea.KeyPgDown)
	assert.Contains(t, stripANSI(m.View()), "> row-11")
	m, _ = sendKey(m, tea.KeyDown)
	assert.Contains(t, stripANSI(m.View()), "> row-12")
	m, cmd := sendKey(m, tea.KeyEnter)
	assert.Zero(t, m.configurations.Offset, "details begin at their heading")
	m = applyCommand(m, cmd)
	assert.Equal(t, "row-12", opened)
	assert.Equal(t, BackendConfigDetailsState, m.state)
	m, _ = sendCommand(m, "/return")
	assert.Equal(t, BackendConfigListState, m.state)
	assert.Contains(t, stripANSI(m.View()), "> row-12")
}

func TestP702ConfigurationFormKeepsActiveFieldVisibleAndPages(t *testing.T) {
	m := NewModelWithClient(httpclient.NewHTTPClient("http://backend.test"))
	m.state = BackendConfigListState
	m.height, m.width = 12, 30
	m, _ = sendCommand(m, "/new-config")
	m.configurations.Raw[1] = `["` + strings.Repeat("wide", 80) + `"]`
	for field := 1; field < len(configurations.FieldNames); field++ {
		m, _ = sendKey(m, tea.KeyTab)
		require.Equal(t, field, m.configurations.Field)
		assert.Contains(t, stripANSI(m.View()), "> "+configurations.FieldNames[field]+":")
	}
	assert.Positive(t, m.configurations.Offset)
	m, _ = sendKey(m, tea.KeyPgUp)
	assert.NotContains(t, stripANSI(m.View()), "> change_types:")
	m, _ = sendKey(m, tea.KeyPgDown)
	assert.Contains(t, stripANSI(m.View()), "> change_types:")
	m, _ = sendKey(m, tea.KeyShiftTab)
	assert.Contains(t, stripANSI(m.View()), "> change_colors:")
	m = applyMsg(m, tea.WindowSizeMsg{Width: 24, Height: 10})
	assert.Contains(t, stripANSI(m.View()), "> change_colors:")

	m.state = BackendConfigDetailsState
	m.configurations = configurations.Model{Detail: p7Row(), DetailLoaded: true}
	m.configurations.Detail.Slug = strings.Repeat("slug", 80)
	m, _ = sendCommand(m, "/edit")
	assert.Contains(t, stripANSI(m.View()), "> project_docs:")
}

func TestP702ConfigurationInputFollowsCursorInLongArray(t *testing.T) {
	m := NewModelWithClient(httpclient.NewHTTPClient("http://backend.test"))
	m.state = BackendConfigFormState
	m.height, m.width = 12, 30
	m.configurations = configurations.Model{}.OpenForm(false)
	m.configurations.Field = 1
	m = m.setConfigurationPrompt(`["` + strings.Repeat("a", 70) + `tail"]`)

	m, _ = sendKey(m, tea.KeyLeft)
	m, _ = sendKey(m, tea.KeyLeft)
	m, _ = sendKey(m, tea.KeyLeft)
	m, _ = sendKey(m, tea.KeyLeft)
	m, _ = sendKey(m, tea.KeyLeft)
	m, _ = sendKey(m, tea.KeyLeft)
	m, _ = sendRune(m, 'Z')
	assert.Contains(t, m.configurationInputBand(m.width), styles.Default.InputBand.Foreground(styles.AccentPurple).Render(" project docs > "))
	assert.Contains(t, m.configurationInputBand(m.width), styles.Default.InputBand.Foreground(styles.AccentGreen).Render("…aaaaaZ▏tail\"]"))
	assert.Contains(t, stripANSI(m.configurationInputBand(m.width)), "Z▏tail")
	assert.NotContains(t, stripANSI(m.configurationInputBand(m.width)), strings.Repeat("a", 30))
	assert.Equal(t, `["`+strings.Repeat("a", 70)+`Ztail"]`, m.promptValue())
	assert.Contains(t, stripANSI(m.View()), "Z▏tail")

	m, _ = sendKey(m, tea.KeyHome)
	assert.Contains(t, stripANSI(m.configurationInputBand(m.width)), `▏["aaaa`)
	m, _ = sendKey(m, tea.KeyEnd)
	assert.Contains(t, stripANSI(m.configurationInputBand(m.width)), `Ztail"]▏`)
	assert.LessOrEqual(t, ansi.StringWidth(stripANSI(m.configurationInputBand(m.width))), m.width)
}

func TestP702ConfigurationPageUpRespondsAfterRepeatedPageDown(t *testing.T) {
	row := p7Row()
	row.ProjectDocs = []string{strings.Repeat("long", 100)}
	for name, setup := range map[string]func(*Model){
		"details": func(m *Model) {
			m.state = BackendConfigDetailsState
			m.configurations = configurations.Model{Detail: row, DetailLoaded: true}
		},
		"form": func(m *Model) {
			m.state = BackendConfigFormState
			m.configurations = (configurations.Model{Detail: row}).OpenForm(true)
		},
	} {
		t.Run(name, func(t *testing.T) {
			m := NewModelWithClient(httpclient.NewHTTPClient("http://backend.test"))
			m.height, m.width = 12, 30
			setup(&m)
			for range 30 {
				m, _ = sendKey(m, tea.KeyPgDown)
			}
			bottom := m.configurations.Offset
			require.Positive(t, bottom)
			bottomView := stripANSI(m.View())
			m, _ = sendKey(m, tea.KeyPgDown)
			assert.Equal(t, bottom, m.configurations.Offset)
			m, _ = sendKey(m, tea.KeyPgUp)
			assert.Equal(t, bottom-max(1, m.height/2), m.configurations.Offset)
			assert.NotEqual(t, bottomView, stripANSI(m.View()))
		})
	}
}
