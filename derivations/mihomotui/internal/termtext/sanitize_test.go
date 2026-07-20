package termtext

import "testing"

func TestSingleLineRemovesTerminalControls(t *testing.T) {
	input := "\x1b[31mred\x1b[0m\x1b]0;owned\x07\x1bP1;2|dcs\x1b\\\x1b_apc\x1b\\\u009b32mgreen\u009c\x00\r\n\tend"
	if got, want := SingleLine(input), "redgreenend"; got != want {
		t.Fatalf("SingleLine() = %q, want %q", got, want)
	}
}

func TestMultilinePreservesOnlySafeWhitespace(t *testing.T) {
	input := "a\n\tb\x1b]52;c;secret\x07\x00"
	if got, want := Multiline(input), "a\n\tb"; got != want {
		t.Fatalf("Multiline() = %q, want %q", got, want)
	}
}

func TestSanitizersRemoveC1StringControls(t *testing.T) {
	input := "before\u0090secret\u009cafter\u009dtitle\u009cend"
	if got, want := SingleLine(input), "beforeafterend"; got != want {
		t.Fatalf("SingleLine() = %q, want %q", got, want)
	}
}
