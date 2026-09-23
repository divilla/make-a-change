package markdown

import (
	"bytes"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// Parser defines Parser values.
type Parser interface {
	Parse(source string) string
}

// Sanitizer defines Sanitizer values.
type Sanitizer interface {
	Parse(source string) string
}

// GoldmarkParser defines GoldmarkParser values.
type GoldmarkParser struct {
	parser goldmark.Markdown
}

// NewGoldmarkParser initializes or executes NewGoldmarkParser behavior.
func NewGoldmarkParser() *GoldmarkParser {
	return &GoldmarkParser{
		parser: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
	}
}

// Parse executes Parse behavior.
func (p *GoldmarkParser) Parse(source string) string {
	var out bytes.Buffer
	if err := p.parser.Convert([]byte(source), &out); err != nil {
		return ""
	}
	return out.String()
}

// BluemondaySanitizer defines BluemondaySanitizer values.
type BluemondaySanitizer struct {
	policy *bluemonday.Policy
}

// NewBluemondaySanitizer initializes or executes NewBluemondaySanitizer behavior.
func NewBluemondaySanitizer() *BluemondaySanitizer {
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("type", "checked", "disabled").OnElements("input")
	return &BluemondaySanitizer{policy: policy}
}

// Parse executes Parse behavior.
func (s *BluemondaySanitizer) Parse(source string) string {
	return s.policy.Sanitize(source)
}
