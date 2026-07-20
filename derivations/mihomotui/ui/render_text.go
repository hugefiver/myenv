package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"mihomotui/internal/termtext"
)

type textSegment struct {
	style lipgloss.Style
	text  string
}

func segment(style lipgloss.Style, text string) textSegment {
	return textSegment{style: style, text: text}
}

// renderTextLine sanitizes and truncates each unstyled segment before it is
// rendered, keeping the resulting line within width terminal cells.
func renderTextLine(width int, segments ...textSegment) string {
	if width < 0 {
		return ""
	}
	if width == 0 {
		width = 80
	}
	var output strings.Builder
	remaining := width
	for _, part := range segments {
		text := termtext.Truncate(part.text, remaining)
		if text == "" {
			continue
		}
		output.WriteString(part.style.Render(text))
		remaining -= termtext.Width(text)
	}
	return output.String()
}
