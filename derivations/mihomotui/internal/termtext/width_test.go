package termtext

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

func TestTruncateUsesTerminalCellsAndGraphemes(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		width int
		want  string
	}{
		{"ASCII", "abcdef", 4, "abc…"},
		{"CJK", "東京abc", 4, "東…"},
		{"combining grapheme", "e\u0301clair", 2, "e\u0301…"},
		{"emoji ZWJ", "👩‍💻dev", 3, "👩‍💻…"},
		{"control sequences", terminalControls("safe"), 100, "safe"},
		{"zero width", "東京", 0, ""},
		{"negative width", "東京", -1, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Truncate(test.input, test.width)
			if got != test.want {
				t.Fatalf("Truncate(%q, %d) = %q, want %q", test.input, test.width, got, test.want)
			}
			assertSafeCellText(t, got, test.width)
		})
	}
}

func TestPadRightUsesTerminalCellsAndSanitizes(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		width int
		want  string
	}{
		{"ASCII", "ab", 4, "ab  "},
		{"CJK", "東京", 6, "東京  "},
		{"combining grapheme", "e\u0301", 3, "e\u0301  "},
		{"emoji ZWJ", "👩‍💻", 4, "👩‍💻  "},
		{"truncated CJK", "東京abc", 4, "東… "},
		{"control sequences", terminalControls("safe"), 6, "safe  "},
		{"zero width", "東京", 0, ""},
		{"negative width", "東京", -1, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := PadRight(test.input, test.width)
			if got != test.want {
				t.Fatalf("PadRight(%q, %d) = %q, want %q", test.input, test.width, got, test.want)
			}
			assertSafeCellText(t, got, test.width)
			if test.width > 0 && testCellWidth(got) != test.width {
				t.Fatalf("PadRight(%q, %d) has cell width %d, want %d", test.input, test.width, testCellWidth(got), test.width)
			}
		})
	}
}

func terminalControls(prefix string) string {
	return prefix +
		"\x1b[31m\x1b[0m" +
		"\x1b]52;c;OSC\x07" +
		"\x1bPDCS\x1b\\" +
		"\x1bXSOS\x1b\\" +
		"\x1b^PM\x1b\\" +
		"\x1b_APC\x1b\\" +
		"\x00\r\n\t" +
		"\u009b31m\u009c" +
		"\u0090C1DCS\u009c" +
		"\u0098C1SOS\u009c" +
		"\u009eC1PM\u009c" +
		"\u009fC1APC\u009c"
}

func assertSafeCellText(t *testing.T, text string, limit int) {
	t.Helper()
	if !utf8.ValidString(text) {
		t.Fatalf("text is not valid UTF-8: %q", text)
	}
	if text != SingleLine(text) {
		t.Fatalf("text retains terminal controls: %q", text)
	}
	if strings.ContainsRune(text, '\x1b') {
		t.Fatalf("text retains ESC: %q", text)
	}
	if limit >= 0 && testCellWidth(text) > limit {
		t.Fatalf("text cell width %d exceeds %d: %q", testCellWidth(text), limit, text)
	}
}

func testCellWidth(text string) int {
	width := 0
	for graphemes := uniseg.NewGraphemes(text); graphemes.Next(); {
		width += graphemes.Width()
	}
	return width
}
