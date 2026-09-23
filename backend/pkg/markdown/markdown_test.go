package markdown

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoldmarkParserAndBluemondaySanitizer(t *testing.T) {
	parser := NewGoldmarkParser()
	sanitizer := NewBluemondaySanitizer()

	html := sanitizer.Parse(parser.Parse("| Done |\n| --- |\n| <script>alert(1)</script> |\n"))

	assert.Contains(t, html, "<table>")
	assert.NotContains(t, strings.ToLower(html), "<script")
}
