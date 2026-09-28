package doc

import (
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

// Render returns sanitized HTML without modifying its raw source.
func (r Renderer) Render(source string) string {
	if source == "" || r.parser == nil || r.sanitizer == nil {
		return ""
	}
	return r.sanitizer.Parse(r.parser.Parse(source))
}
