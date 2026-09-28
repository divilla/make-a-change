// Command validate-apih-suite rejects APIHydra breakpoints before coverage runs.
package main

import (
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
				return fmt.Errorf("Debug directives are prohibited in the full campaign (line %d)", node.Content[i].Line)
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

func validateDirectory(directory string) error {
	return filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || (filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml") {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		if err := validate(file); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		return nil
	})
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: validate-apih-suite DIRECTORY")
		os.Exit(2)
	}
	if err := validateDirectory(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
