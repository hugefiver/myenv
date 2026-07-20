package termtext

import (
	"strings"

	"github.com/rivo/uniseg"
)

type state uint8

const (
	text state = iota
	escape
	csi
	osc
	oscEscape
	stringControl
	stringEscape
)

// SingleLine removes terminal control sequences and all control whitespace.
func SingleLine(input string) string {
	return sanitize(input, false)
}

// Multiline removes terminal control sequences while preserving newlines and tabs.
func Multiline(input string) string {
	return sanitize(input, true)
}

// Truncate removes terminal controls and preserves complete grapheme clusters
// within the requested terminal-cell width.
func Truncate(input string, width int) string {
	if width <= 0 {
		return ""
	}
	input = SingleLine(input)
	if cellWidth(input) <= width {
		return input
	}

	const ellipsis = "…"
	remaining := width - cellWidth(ellipsis)
	if remaining <= 0 {
		return ellipsis
	}
	var output strings.Builder
	for graphemes := uniseg.NewGraphemes(input); graphemes.Next(); {
		cluster := graphemes.Str()
		clusterWidth := graphemes.Width()
		if clusterWidth > remaining {
			break
		}
		output.WriteString(cluster)
		remaining -= clusterWidth
	}
	return output.String() + ellipsis
}

// PadRight removes terminal controls, truncates by terminal-cell width, then
// right-pads with spaces to exactly width cells.
func PadRight(input string, width int) string {
	if width <= 0 {
		return ""
	}
	output := Truncate(input, width)
	return output + strings.Repeat(" ", width-cellWidth(output))
}

// Width returns the terminal-cell width of a sanitized single-line string.
func Width(input string) int {
	return cellWidth(SingleLine(input))
}

func sanitize(input string, multiline bool) string {
	var out []rune
	current := text

	for _, r := range input {
		switch current {
		case text:
			switch r {
			case 0x1b:
				current = escape
			case 0x9b:
				current = csi
			case 0x9d:
				current = osc
			case 0x90, 0x98, 0x9e, 0x9f:
				current = stringControl
			default:
				if safeTextRune(r, multiline) {
					out = append(out, r)
				}
			}

		case escape:
			switch r {
			case '[':
				current = csi
			case ']':
				current = osc
			case 'P', 'X', '^', '_':
				current = stringControl
			case 0x1b:
				current = escape
			default:
				current = text
			}

		case csi:
			if r >= 0x40 && r <= 0x7e {
				current = text
			} else if r == 0x1b {
				current = escape
			}

		case osc:
			switch r {
			case 0x07, 0x9c:
				current = text
			case 0x1b:
				current = oscEscape
			}

		case oscEscape:
			switch r {
			case '\\', 0x9c:
				current = text
			case 0x1b:
				current = oscEscape
			default:
				current = osc
			}

		case stringControl:
			switch r {
			case 0x9c:
				current = text
			case 0x1b:
				current = stringEscape
			}

		case stringEscape:
			switch r {
			case '\\', 0x9c:
				current = text
			case 0x1b:
				current = stringEscape
			default:
				current = stringControl
			}
		}
	}

	return string(out)
}

func safeTextRune(r rune, multiline bool) bool {
	if r == '\n' || r == '\t' {
		return multiline
	}
	return r >= 0x20 && r != 0x7f && (r < 0x80 || r > 0x9f)
}

func cellWidth(input string) int {
	width := 0
	for graphemes := uniseg.NewGraphemes(input); graphemes.Next(); {
		width += graphemes.Width()
	}
	return width
}
