package change

import (
	"mch_api/internal/domain"
	"mch_api/pkg/markdown"
)

// Renderer defines Renderer values.
type Renderer struct {
	parser    markdown.Parser
	sanitizer markdown.Sanitizer
}

// NewRenderer initializes or executes NewRenderer behavior.
func NewRenderer(parser markdown.Parser, sanitizer markdown.Sanitizer) Renderer {
	return Renderer{parser: parser, sanitizer: sanitizer}
}

// RenderChange is the P4 compatibility adapter consumed by testcase mutation rendering.
func (r Renderer) RenderChange(change domain.Change) domain.Change {
	if r.parser == nil || r.sanitizer == nil {
		return change
	}
	if change.Spec != "" {
		change.SpecHTML = r.sanitizer.Parse(r.parser.Parse(change.Spec))
	}
	if change.PR != "" {
		change.PRHtml = r.sanitizer.Parse(r.parser.Parse(change.PR))
	}
	return change
}

// RenderMutation preserves the unmigrated testcase response until P4.
func (r Renderer) RenderMutation(mutation domain.TestCaseMutationResponse) domain.TestCaseMutationResponse {
	mutation.Change = r.RenderChange(mutation.Change)
	return mutation
}

// Render returns sanitized HTML without modifying its raw source.
func (r Renderer) Render(source string) string {
	if source == "" || r.parser == nil || r.sanitizer == nil {
		return ""
	}
	return r.sanitizer.Parse(r.parser.Parse(source))
}
