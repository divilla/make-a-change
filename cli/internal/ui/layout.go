package ui

// NormalizeWidth returns a usable terminal width for rendering.
func NormalizeWidth(width int) int {
	if width < 20 {
		return 100
	}
	return width
}
