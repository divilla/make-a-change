package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestRejectDebugYAMLForms(t *testing.T) {
	for _, input := range []string{
		"spec:\n  steps:\n    - debug: true",
		`spec: {steps: [{debug: true}]}`,
		`spec: {steps: [{"debug": true}]}`,
		`spec: {steps: [{'debug': true}]}`,
		`spec: {steps: [{"de\u0062ug": true}]}`,
		"base: &breakpoint {debug: true}\nspec: {steps: [*breakpoint]}",
		"key: &key debug\nspec: {steps: [{*key: true}]}",
		"spec: {}\n---\nspec: {steps: [{debug: false}]}",
	} {
		t.Run(input, func(t *testing.T) {
			if err := validate(strings.NewReader(input)); err == nil || !strings.Contains(err.Error(), "Debug") {
				t.Fatalf("expected Debug rejection, got %v", err)
			}
		})
	}
}

func TestAcceptBodiesAndRejectMalformedYAML(t *testing.T) {
	if err := validate(strings.NewReader(`spec: {steps: [{response: {expected_body: '{"debug": true}'}}]}`)); err != nil {
		t.Fatal(err)
	}
	if err := validate(strings.NewReader("spec: [")); err == nil {
		t.Fatal("malformed YAML accepted")
	}
}

const executable = `app: apihydra
kind: steps
spec:
  steps:
    - request: {method: GET, path: /api/health}
      response: {expected_status: 200}
`

func writeSuite(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	write := func(path, data string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, path)), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, path), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("root.yaml", "app: apihydra\nkind: root\nspec: {base_url: 'http://127.0.0.1:8080', timeout: 5}")
	write("notes.md", "not YAML: [")
	write("change/02-main.yaml", executable)
	write("change/nested/defaults.yml", "app: apihydra\nkind: defaults\nspec: {timeout: 2, headers: {Content-Type: application/json}}")
	write("change/nested/03-post.yml", executable)
	return dir
}

func TestCompleteDecodedManifest(t *testing.T) {
	dir := writeSuite(t)
	result, err := suiteManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(result.Phases, ",") != "standalone" || len(result.Documents) != 4 {
		t.Fatalf("%+v", result)
	}
	steps := 0
	for _, doc := range result.Documents {
		data, err := os.ReadFile(filepath.Join(dir, doc.Path))
		if err != nil {
			t.Fatal(err)
		}
		if doc.Hash != fmt.Sprintf("%x", sha256.Sum256(data)) {
			t.Fatal("hash mismatch")
		}
		if doc.Kind == "steps" {
			steps++
			selection := strings.Split(doc.Path, "/")[0]
			phase := selection
			if selection == "change" {
				phase = "standalone"
			}
			if doc.Phase != phase || doc.Selection != selection || doc.Steps != 1 {
				t.Fatalf("%+v", doc)
			}
			if doc.Path == "change/nested/03-post.yml" && doc.Stage != 2 {
				t.Fatal("lost ancestor stage")
			}
		} else if doc.Phase != "" || doc.Steps != 0 {
			t.Fatal("nonexecutable classified as steps")
		}
	}
	if steps != 2 {
		t.Fatal("omitted executable")
	}
	if _, err := suiteManifest(filepath.Join(dir, "missing")); err == nil {
		t.Fatal("missing suite accepted")
	}
}

func TestRejectIncompleteOrMalformedWholeSuite(t *testing.T) {
	for _, tc := range []struct{ name, path, data string }{
		{"root executable", "extra.yaml", executable},
		{"extra phase", "extra/steps.yaml", executable},
		{"missing root", "root.yaml", "REMOVE"},
		{"empty phase", "change/02-main.yaml", "app: apihydra\nkind: steps\nspec: {steps: []}"},
		{"malformed unselected", "unused.yaml", "spec: ["},
		{"unknown app", "unused.yaml", "app: typo\nkind: defaults\nspec: {}"},
		{"unknown kind", "unused.yaml", "app: apihydra\nkind: stepps\nspec: {}"},
		{"missing kind", "unused.yaml", "app: apihydra\nspec: {}"},
		{"bad defaults field", "unused.yaml", "app: apihydra\nkind: defaults\nspec: {timeout: wrong}"},
		{"unknown defaults field", "unused.yaml", "app: apihydra\nkind: defaults\nspec: {timeuot: 1}"},
		{"bad steps field", "change/02-main.yaml", strings.Replace(executable, "expected_status: 200", "expected_status: broken", 1)},
		{"missing step request", "change/02-main.yaml", "app: apihydra\nkind: steps\nspec: {steps: [{}]}"},
		{"duplicate key", "change/02-main.yaml", strings.Replace(executable, "kind: steps", "kind: steps\nkind: steps", 1)},
		{"multiple documents", "change/02-main.yaml", executable + "---\n" + executable},
		{"nested root", "change/root.yaml", "app: apihydra\nkind: root\nspec: {}"},
		{"duplicate root", "another.yaml", "app: apihydra\nkind: root\nspec: {}"},
		{"invalid spec", "unused.yaml", "app: apihydra\nkind: defaults\nspec: null"},
		{"escaped debug", "unused.yaml", `app: apihydra
kind: defaults
spec: {"de\u0062ug": false}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeSuite(t)
			path := filepath.Join(dir, tc.path)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if tc.data == "REMOVE" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			} else if err := os.WriteFile(path, []byte(tc.data), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := suiteManifest(dir); err == nil {
				t.Fatal("invalid suite accepted")
			}
		})
	}
}

func TestRepositoryManifestIncludesEveryParallelGroup(t *testing.T) {
	manifest, err := suiteManifest("../../apih-tests")
	if err != nil {
		t.Fatal(err)
	}
	normal := 0
	groups := map[string]bool{}
	for _, doc := range manifest.Documents {
		if doc.Phase != "standalone" {
			continue
		}
		normal += doc.Steps
		groups[doc.Selection] = true
		if doc.Stage != 1 {
			t.Fatalf("normal group must execute at the same stage: %+v", doc)
		}
		switch filepath.Base(doc.Path) {
		case "01-init.yaml", "02-main.yaml", "03-post.yaml":
		default:
			t.Fatalf("unexpected group file: %s", doc.Path)
		}
	}
	if normal == 0 {
		t.Fatal("standalone suite has no scenarios")
	}
	for _, group := range []string{"change", "config", "doc", "epic", "health", "project", "testcase"} {
		if !groups[group] {
			t.Errorf("missing parallel group: %s", group)
		}
	}
	if len(groups) != 7 {
		t.Fatalf("unexpected normal groups: %v", groups)
	}
}

// Parallel groups must never consume another group's capture: scheduling between
// groups is deliberately unspecified. Numbered files within a group are serial.
func TestParallelGroupsHaveLocalOrderedCaptures(t *testing.T) {
	manifest, err := suiteManifest("../../apih-tests")
	if err != nil {
		t.Fatal(err)
	}
	references := regexp.MustCompile(`\$\{([^}]+)\}`)
	available := map[string]map[string]bool{}
	owners := map[string]string{}
	for _, entry := range manifest.Documents {
		if entry.Phase != "standalone" {
			continue
		}
		group := entry.Selection
		if available[group] == nil {
			available[group] = map[string]bool{}
		}
		data, err := os.ReadFile(filepath.Join("../../apih-tests", entry.Path))
		if err != nil {
			t.Fatal(err)
		}
		var doc document
		if err = decodeStrict(data, &doc); err != nil {
			t.Fatal(err)
		}
		var spec struct {
			Steps []step `yaml:"steps"`
		}
		if err = doc.Spec.Decode(&spec); err != nil {
			t.Fatal(err)
		}
		for index, step := range spec.Steps {
			for name := range step.Vars {
				available[group][name] = true
			}
			for _, ref := range references.FindAllStringSubmatch(step.Request.Body+step.Response.Body, -1) {
				if !available[group][ref[1]] {
					t.Errorf("%s step %d consumes %s before a local producer", entry.Path, index, ref[1])
				}
			}
			for name := range step.Response.Capture {
				if previous, ok := owners[name]; ok {
					t.Errorf("capture %s written twice (%s, %s)", name, previous, entry.Path)
				}
				owners[name] = entry.Path
				available[group][name] = true
			}
		}
	}
}

func TestStandaloneSuiteUsesCapturedMutationTargets(t *testing.T) {
	root := "../../apih-tests"
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".sql" {
			t.Errorf("standalone suite must not contain SQL fixtures: %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	manifest, err := suiteManifest(root)
	if err != nil {
		t.Fatal(err)
	}
	literalID := regexp.MustCompile(`"(?:id|ref_id|project_id|change_id|epic_id|after_change_id)"\s*:\s*[1-9][0-9]*`)
	for _, entry := range manifest.Documents {
		if entry.Kind != "steps" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, entry.Path))
		if err != nil {
			t.Fatal(err)
		}
		var doc document
		if err := decodeStrict(data, &doc); err != nil {
			t.Fatal(err)
		}
		var spec struct {
			Steps []step `yaml:"steps"`
		}
		if err := doc.Spec.Decode(&spec); err != nil {
			t.Fatal(err)
		}
		for i, step := range spec.Steps {
			// Only read-only not-found boundary checks may use a literal positive ID.
			read := strings.HasSuffix(step.Request.Path, "/details") || strings.HasSuffix(step.Request.Path, "/list")
			if literalID.MatchString(step.Request.Body) && (!read || step.Response.Status != 404) {
				t.Errorf("%s step %d targets a fixed positive ID; capture an API-created ID", entry.Path, i)
			}
		}
	}
}

func TestStandaloneSuiteCoversEveryRegisteredOperation(t *testing.T) {
	manifest, err := suiteManifest("../../apih-tests")
	if err != nil {
		t.Fatal(err)
	}
	covered := map[string]bool{}
	for _, entry := range manifest.Documents {
		if entry.Kind != "steps" {
			continue
		}
		data, err := os.ReadFile(filepath.Join("../../apih-tests", entry.Path))
		if err != nil {
			t.Fatal(err)
		}
		var doc document
		if err := decodeStrict(data, &doc); err != nil {
			t.Fatal(err)
		}
		var spec struct {
			Steps []step `yaml:"steps"`
		}
		if err := doc.Spec.Decode(&spec); err != nil {
			t.Fatal(err)
		}
		for _, step := range spec.Steps {
			if step.Response.Status >= 200 && step.Response.Status < 300 {
				covered[step.Request.Method+" "+step.Request.Path] = true
			}
		}
	}
	files, err := filepath.Glob("../../internal/*/api.go")
	if err != nil {
		t.Fatal(err)
	}
	group := regexp.MustCompile(`\.Group\("([^"]*)"\)`)
	route := regexp.MustCompile(`(a\.g|e)\.(GET|POST)\("([^"]*)"`)
	registered := map[string]bool{}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		prefix := ""
		for _, match := range group.FindAllStringSubmatch(string(data), -1) {
			prefix += match[1]
		}
		for _, match := range route.FindAllStringSubmatch(string(data), -1) {
			path := match[3]
			if match[1] == "a.g" {
				path = prefix + path
			}
			operation := match[2] + " " + path
			registered[operation] = true
			if !covered[operation] {
				t.Errorf("no standalone success case for %s", operation)
			}
		}
	}
	if len(registered) == 0 || len(covered) != len(registered) {
		t.Fatalf("route inventory mismatch: %d registered, %d covered", len(registered), len(covered))
	}
}
