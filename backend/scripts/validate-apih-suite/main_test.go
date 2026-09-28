package main

import (
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

func TestDirectoryValidation(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "root.yaml"), []byte("app: apihydra"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "notes.md"), []byte("not YAML: ["), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateDirectory(directory); err != nil {
		t.Fatal(err)
	}
	if err := validateDirectory(filepath.Join(directory, "missing")); err == nil {
		t.Fatal("missing suite accepted")
	}
}
