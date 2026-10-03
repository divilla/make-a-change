package ui

import (
	"strconv"
	"strings"
	"unicode"
)

// SafeText escapes terminal controls for display without changing stored data.
func SafeText(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r == '\n' {
			b.WriteRune(r)
			continue
		}
		if r == '\t' {
			b.WriteString("    ")
			continue
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			b.WriteString(strconv.QuoteRune(r)[1 : len(strconv.QuoteRune(r))-1])
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
