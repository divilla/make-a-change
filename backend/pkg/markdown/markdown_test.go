package markdown

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"

	"github.com/stretchr/testify/assert"
)

func TestGoldmarkParserAndBluemondaySanitizer(t *testing.T) {
	parser := NewGoldmarkParser()
	sanitizer := NewBluemondaySanitizer()

	html := sanitizer.Parse(parser.Parse("| Done |\n| --- |\n| <script>alert(1)</script> |\n"))

	assert.Contains(t, html, "<table>")
	assert.NotContains(t, strings.ToLower(html), "<script")
}

type failingMarkdown struct {
	goldmark.Markdown
	err error
}

func (m failingMarkdown) Convert(_ []byte, _ io.Writer, _ ...parser.ParseOption) error { return m.err }

func TestMarkdownFailureKeepsEmptyOutputAndLogsCause(t *testing.T) {
	cause := errors.New("converter failed")
	var output bytes.Buffer
	previous := log.Logger
	log.Logger = zerolog.New(&output)
	t.Cleanup(func() { log.Logger = previous })
	p := &GoldmarkParser{parser: failingMarkdown{err: cause}}
	assert.Empty(t, p.Parse("text"))
	assert.Contains(t, output.String(), "render markdown: converter failed")
}
