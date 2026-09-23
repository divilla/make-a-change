package agent

import (
    "encoding/json"
    "fmt"
    "os/exec"
    "strings"

    "cli/internal/changes"
)

// ParsedChange stores the backend fields parsed from generated Change markdown.
type ParsedChange struct {
    Title              string
    Spec               string
    ChangeTypes        []string
    ChangeTypesPresent bool
    TestCases          []string
}

// ParseGeneratedChange extracts the title and Types metadata from a generated Change spec.
func ParseGeneratedChange(spec string) (ParsedChange, error) {
    parsed, err := changes.ParseSpecStructure(spec)
    if err != nil {
        return ParsedChange{}, err
    }
    return ParsedChange{
        Title:              parsed.Title,
        Spec:               parsed.Spec,
        ChangeTypes:        parsed.ChangeTypes,
        ChangeTypesPresent: parsed.ChangeTypesPresent,
        TestCases:          ExtractQATestCases(parsed.Spec),
    }, nil
}

// ExtractQATestCases extracts QA scenarios listed under the generated Change QA section.
func ExtractQATestCases(spec string) []string {
    normalized := strings.ReplaceAll(strings.ReplaceAll(spec, "\r\n", "\n"), "\r", "\n")
    lines := strings.Split(normalized, "\n")
    sectionIndex := -1
    for i, line := range lines {
        if isQATestCasesHeading(line) {
            sectionIndex = i
            break
        }
    }
    if sectionIndex < 0 {
        return nil
    }

    testCases := []string{}
    current := ""
    for _, line := range lines[sectionIndex+1:] {
        trimmed := strings.TrimSpace(line)
        if strings.HasPrefix(trimmed, "## ") {
            break
        }
        if trimmed == "" {
            continue
        }
        if item, ok := markdownListItemText(trimmed); ok {
            if current != "" {
                testCases = appendQATestCase(testCases, current)
            }
            current = item
            continue
        }
        if current != "" {
            current += " " + trimmed
        }
    }
    if current != "" {
        testCases = appendQATestCase(testCases, current)
    }
    return testCases
}

func isQATestCasesHeading(line string) bool {
    line = strings.TrimSpace(line)
    line = strings.TrimSuffix(line, ":")
    return strings.EqualFold(line, "## QA Test Cases")
}

func appendQATestCase(testCases []string, value string) []string {
    value = strings.TrimSpace(value)
    if value == "" || strings.EqualFold(value, "none.") || strings.EqualFold(value, "none") {
        return testCases
    }
    return append(testCases, value)
}

func markdownListItemText(line string) (string, bool) {
    for _, prefix := range []string{"- ", "* ", "+ "} {
        if strings.HasPrefix(line, prefix) {
            return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
        }
    }
    for i, r := range line {
        if r < '0' || r > '9' {
            if i > 0 && (r == '.' || r == ')') && len(line) > i+1 && line[i+1] == ' ' {
                return strings.TrimSpace(line[i+2:]), true
            }
            return "", false
        }
    }
    return "", false
}

// ExtractSessionID returns the Codex thread/session ID from JSON event output.
func ExtractSessionID(jsonLines string) string {
    if sessionID := extractThreadStartedIDWithJQ(jsonLines); sessionID != "" {
        return sessionID
    }
    events := parseJSONLineEvents(jsonLines)
    for _, event := range events {
        if stringField(event, "type") == "thread.started" {
            if sessionID := stringField(event, "thread_id"); sessionID != "" {
                return sessionID
            }
        }
    }
    for _, event := range events {
        if sessionID := stringField(event, "session_id"); sessionID != "" {
            return sessionID
        }
        if session, ok := event["session"].(map[string]any); ok {
            if sessionID := stringField(session, "id"); sessionID != "" {
                return sessionID
            }
        }
        if sessionID := stringField(event, "id"); sessionID != "" {
            return sessionID
        }
    }
    return ""
}

func extractThreadStartedIDWithJQ(jsonLines string) string {
    cmd := exec.Command("jq", "-r", `select(.type=="thread.started") | .thread_id`)
    cmd.Stdin = strings.NewReader(jsonLines)
    output, err := cmd.Output()
    if err != nil {
        return ""
    }
    for _, line := range strings.Split(string(output), "\n") {
        line = strings.TrimSpace(line)
        if line != "" && line != "null" {
            return line
        }
    }
    return ""
}

func parseJSONLineEvents(jsonLines string) []map[string]any {
    events := []map[string]any{}
    for _, line := range strings.Split(jsonLines, "\n") {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }
        var event map[string]any
        if err := json.Unmarshal([]byte(line), &event); err != nil {
            continue
        }
        events = append(events, event)
    }
    return events
}

func stringField(values map[string]any, key string) string {
    value, ok := values[key]
    if !ok {
        return ""
    }
    switch typed := value.(type) {
    case string:
        return typed
    default:
        return fmt.Sprint(typed)
    }
}
