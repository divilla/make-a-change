// Command validate-apih-suite checks standalone APIHydra suites.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func validate(document io.Reader) error {
	decoder := yaml.NewDecoder(document)
	for {
		var root yaml.Node
		if err := decoder.Decode(&root); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := inspect(&root, make(map[*yaml.Node]bool)); err != nil {
			return err
		}
	}
}

func inspect(node *yaml.Node, seen map[*yaml.Node]bool) error {
	if node == nil || seen[node] {
		return nil
	}
	seen[node] = true
	if node.Kind == yaml.AliasNode {
		return inspect(node.Alias, seen)
	}
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind == yaml.AliasNode {
				key = key.Alias
			}
			if key != nil && strings.EqualFold(key.Value, "debug") {
				return fmt.Errorf("Debug directives are prohibited in the full suite (line %d)", node.Content[i].Line)
			}
		}
	}
	for _, child := range node.Content {
		if err := inspect(child, seen); err != nil {
			return err
		}
	}
	return nil
}

// These local decoding types mirror the installed APIHydra document shapes.
// KnownFields rejects malformed unselected defaults just as executable files.
type defaults struct {
	BaseURL        string            `yaml:"base_url"`
	BasePath       string            `yaml:"base_path"`
	Headers        map[string]string `yaml:"headers"`
	Timeout        int               `yaml:"timeout"`
	Retries        int               `yaml:"retries"`
	DisableCookies *bool             `yaml:"disable_cookies"`
}
type step struct {
	Vars    map[string]string `yaml:"vars"`
	Request struct {
		Method   string   `yaml:"method"`
		Path     string   `yaml:"path"`
		Query    string   `yaml:"query"`
		Body     string   `yaml:"body"`
		Defaults defaults `yaml:"defaults"`
	} `yaml:"request"`
	Response struct {
		Status  int                 `yaml:"expected_status"`
		Body    string              `yaml:"expected_body"`
		Types   map[string][]string `yaml:"expected_types"`
		Capture map[string]string   `yaml:"capture"`
	} `yaml:"response"`
}
type document struct {
	App      string `yaml:"app"`
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name   string   `yaml:"name"`
		Labels []string `yaml:"labels"`
	} `yaml:"metadata"`
	Spec yaml.Node `yaml:"spec"`
}
type manifestEntry struct {
	Path      string `json:"path"`
	Hash      string `json:"sha256"`
	Kind      string `json:"kind"`
	Phase     string `json:"phase,omitempty"`
	Selection string `json:"selection,omitempty"`
	Stage     int    `json:"stage"`
	Steps     int    `json:"steps"`
}
type manifest struct {
	Phases    []string        `json:"phases"`
	Documents []manifestEntry `json:"documents"`
}

func decodeStrict(data []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("expected exactly one YAML document")
	}
	return nil
}

func suiteManifest(directory string) (manifest, error) {
	result := manifest{Phases: []string{"standalone"}}
	counts := map[string]int{"standalone": 0}
	rootCount := 0
	defaultsDirs := map[string]bool{}
	err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("suite symlinks prohibited: %s", path)
		}
		if entry.IsDir() || (filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := validate(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		var doc document
		if err := decodeStrict(data, &doc); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if doc.App != "apihydra" {
			return fmt.Errorf("%s: app must be apihydra", path)
		}
		if doc.Spec.Kind != yaml.MappingNode {
			return fmt.Errorf("%s: spec must be a mapping", path)
		}
		rel, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		item := manifestEntry{Path: rel, Hash: fmt.Sprintf("%x", sha256.Sum256(data)), Kind: doc.Kind, Stage: strings.Count(rel, "/")}
		spec, err := yaml.Marshal(&doc.Spec)
		if err != nil {
			return err
		}
		switch doc.Kind {
		case "root", "defaults":
			var value defaults
			if err := decodeStrict(spec, &value); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			if doc.Kind == "root" {
				rootCount++
				if filepath.Dir(rel) != "." {
					return fmt.Errorf("nested root: %s", rel)
				}
			} else {
				dir := filepath.Dir(rel)
				if defaultsDirs[dir] {
					return fmt.Errorf("duplicate defaults: %s", dir)
				}
				defaultsDirs[dir] = true
			}
		case "steps":
			var value struct {
				Defaults defaults `yaml:"defaults"`
				Steps    []step   `yaml:"steps"`
			}
			if err := decodeStrict(spec, &value); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			selection, _, found := strings.Cut(rel, "/")
			phase := selection
			switch selection {
			case "change", "config", "doc", "epic", "health", "project", "testcase":
				phase = "standalone"
			}
			if _, ok := counts[phase]; !found || !ok {
				return fmt.Errorf("unclassified executable: %s", rel)
			}
			if len(value.Steps) == 0 {
				return fmt.Errorf("empty executable: %s", rel)
			}
			for _, step := range value.Steps {
				if step.Request.Method == "" || step.Request.Path == "" || step.Response.Status < 100 || step.Response.Status > 599 {
					return fmt.Errorf("%s: each step needs method, path and explicit HTTP status", rel)
				}
			}
			item.Phase, item.Selection, item.Steps = phase, selection, len(value.Steps)
			counts[phase] += len(value.Steps)
		default:
			return fmt.Errorf("%s: unknown document kind %q", path, doc.Kind)
		}
		result.Documents = append(result.Documents, item)
		return nil
	})
	if err != nil {
		return result, err
	}
	if rootCount != 1 {
		return result, fmt.Errorf("expected one root document, got %d", rootCount)
	}
	for _, phase := range result.Phases {
		if counts[phase] == 0 {
			return result, fmt.Errorf("missing/empty phase: %s", phase)
		}
	}
	return result, nil
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: validate-apih-suite DIRECTORY")
		os.Exit(2)
	}
	result, err := suiteManifest(os.Args[1])
	if err == nil {
		err = json.NewEncoder(os.Stdout).Encode(result)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
