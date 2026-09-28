package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
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
	write("root.yaml", "app: apihydra\nkind: root\nspec: {base_url: 'http://127.0.0.1:19080', timeout: 5}")
	write("notes.md", "not YAML: [")
	write("fixtures.sql", "not YAML: [")
	write("normal/steps.yaml", executable)
	write("normal/nested/defaults.yml", "app: apihydra\nkind: defaults\nspec: {timeout: 2, headers: {Content-Type: application/json}}")
	write("normal/nested/steps.yml", executable)
	write("outage/steps.yaml", executable)
	write("recovery/steps.yaml", executable)
	return dir
}

func TestCompleteDecodedManifest(t *testing.T) {
	dir := writeSuite(t)
	result, err := suiteManifest(dir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(result.Phases, ",") != "normal,outage,recovery" || len(result.Documents) != 6 {
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
			if doc.Phase != strings.Split(doc.Path, "/")[0] || doc.Selection != doc.Phase || doc.Steps != 1 {
				t.Fatalf("%+v", doc)
			}
			if doc.Path == "normal/nested/steps.yml" && doc.Stage != 2 {
				t.Fatal("lost ancestor stage")
			}
		} else if doc.Phase != "" || doc.Steps != 0 {
			t.Fatal("nonexecutable classified as steps")
		}
	}
	if steps != 4 {
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
		{"missing phase", "outage/steps.yaml", "REMOVE"},
		{"empty phase", "outage/steps.yaml", "app: apihydra\nkind: steps\nspec: {steps: []}"},
		{"malformed unselected", "unused.yaml", "spec: ["},
		{"unknown app", "unused.yaml", "app: typo\nkind: defaults\nspec: {}"},
		{"unknown kind", "unused.yaml", "app: apihydra\nkind: stepps\nspec: {}"},
		{"missing kind", "unused.yaml", "app: apihydra\nspec: {}"},
		{"bad defaults field", "unused.yaml", "app: apihydra\nkind: defaults\nspec: {timeout: wrong}"},
		{"unknown defaults field", "unused.yaml", "app: apihydra\nkind: defaults\nspec: {timeuot: 1}"},
		{"bad steps field", "outage/steps.yaml", strings.Replace(executable, "expected_status: 200", "expected_status: broken", 1)},
		{"missing step request", "outage/steps.yaml", "app: apihydra\nkind: steps\nspec: {steps: [{}]}"},
		{"duplicate key", "outage/steps.yaml", strings.Replace(executable, "kind: steps", "kind: steps\nkind: steps", 1)},
		{"multiple documents", "outage/steps.yaml", executable + "---\n" + executable},
		{"nested root", "outage/root.yaml", "app: apihydra\nkind: root\nspec: {}"},
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

func TestRepositoryManifestRetainsNormalAndAllPhases(t *testing.T) {
	manifest, err := suiteManifest("../../apih-tests")
	if err != nil {
		t.Fatal(err)
	}
	normal, normalFiles := 0, 0
	for _, doc := range manifest.Documents {
		if doc.Phase == "normal" {
			normal += doc.Steps
			normalFiles++
		}
	}
	if normal != 408 || normalFiles != 5 {
		t.Fatalf("normal scenarios lost: %d requests in %d files", normal, normalFiles)
	}
}
