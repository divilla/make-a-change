package app

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

type FinalRequirement struct {
	Title     string
	Types     []string
	EpicName  *string
	EpicID    *int
	Body      string
	Source    string
	BodyLines []string
}

var typeSlugPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

func ParseRequirementMarkdown(source string) (FinalRequirement, error) {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	h1Count := 0
	h1Index := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "# ") {
			h1Count++
			if h1Index == -1 {
				h1Index = i
			}
		}
	}
	if h1Count == 0 {
		return FinalRequirement{}, errors.New("final markdown must contain a single H1 title")
	}
	if h1Count > 1 {
		return FinalRequirement{}, errors.New("final markdown must not contain multiple H1 titles")
	}
	for i := 0; i < h1Index; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return FinalRequirement{}, errors.New("final markdown must not include content before the H1 title")
		}
	}
	title := strings.TrimSpace(strings.TrimPrefix(lines[h1Index], "# "))
	if title == "" {
		return FinalRequirement{}, errors.New("H1 title cannot be empty")
	}

	typeIndex := nextNonBlank(lines, h1Index+1)
	if typeIndex == -1 || !strings.HasPrefix(strings.TrimSpace(lines[typeIndex]), "Types: ") {
		return FinalRequirement{}, errors.New("final markdown must include a Types metadata line after the H1 title")
	}
	types, err := parseTypes(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[typeIndex]), "Types: ")))
	if err != nil {
		return FinalRequirement{}, err
	}

	metadataIndexes := map[int]struct{}{h1Index: {}, typeIndex: {}}
	var epicName *string
	nextIndex := nextNonBlank(lines, typeIndex+1)
	if nextIndex != -1 && strings.HasPrefix(strings.TrimSpace(lines[nextIndex]), "Epic: ") {
		name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[nextIndex]), "Epic: "))
		if name == "" {
			return FinalRequirement{}, errors.New("Epic metadata line cannot be empty")
		}
		epicName = &name
		metadataIndexes[nextIndex] = struct{}{}
	}

	bodyLines := make([]string, 0, len(lines))
	for i, line := range lines {
		if _, ok := metadataIndexes[i]; ok {
			continue
		}
		bodyLines = append(bodyLines, line)
	}
	body := strings.TrimSpace(strings.Join(bodyLines, "\n"))
	if body == "" {
		return FinalRequirement{}, errors.New("final markdown body cannot be empty")
	}

	return FinalRequirement{
		Title:     title,
		Types:     types,
		EpicName:  epicName,
		Body:      body,
		Source:    source,
		BodyLines: strings.Split(body, "\n"),
	}, nil
}

func ValidateRequirementReferences(req FinalRequirement, refs ChangeReferences, epics []Epic) (FinalRequirement, error) {
	validTypes := make([]string, 0, len(refs.Types))
	for _, item := range refs.Types {
		validTypes = append(validTypes, item.Slug)
	}
	for _, slug := range req.Types {
		if !slices.Contains(validTypes, slug) {
			return FinalRequirement{}, fmt.Errorf("unknown type slug %q", slug)
		}
	}
	if req.EpicName != nil {
		for _, epic := range epics {
			if epic.Name == *req.EpicName {
				req.EpicID = &epic.ID
				return req, nil
			}
		}
		return FinalRequirement{}, fmt.Errorf("unknown epic %q", *req.EpicName)
	}
	return req, nil
}

func parseTypes(value string) ([]string, error) {
	if value == "" {
		return nil, errors.New("Types metadata line cannot be empty")
	}
	if strings.ContainsAny(value, " \t") {
		return nil, errors.New("Types metadata line must not contain spaces")
	}
	parts := strings.Split(value, "|")
	types := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" || !typeSlugPattern.MatchString(part) {
			return nil, fmt.Errorf("invalid type slug %q", part)
		}
		types = append(types, part)
	}
	return types, nil
}

func nextNonBlank(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "" {
			return i
		}
	}
	return -1
}
