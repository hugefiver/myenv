package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNormalizeSingleInputPasteIMEAndRunes(t *testing.T) {
	input := appendKeyRunes("", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("https://例.example/配置.yaml\r\n"), Paste: true})
	input = appendKeyRunes(input, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("東京")})
	if got, want := input, "https://例.example/配置.yaml\r\n東京"; got != want {
		t.Fatalf("appendKeyRunes() = %q, want %q", got, want)
	}
	if got, want := removeLastRune(input), "https://例.example/配置.yaml\r\n東"; got != want {
		t.Fatalf("removeLastRune() = %q, want %q", got, want)
	}
	if got := removeLastRune(""); got != "" {
		t.Fatalf("removeLastRune(empty) = %q", got)
	}
}

func TestNormalizeSingleInputTrimsAndRejectsLines(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{input: "\t https://example.test/profile.yaml\r\n", want: "https://example.test/profile.yaml"},
		{input: "  C:\\配置\\東京.yaml\t", want: "C:\\配置\\東京.yaml"},
	} {
		got, err := normalizeSingleInput(tc.input)
		if err != nil || got != tc.want {
			t.Fatalf("normalizeSingleInput(%q) = %q, %v; want %q", tc.input, got, err, tc.want)
		}
	}
	for _, input := range []string{"", " \r\n\t", "first\nsecond", "first\rsecond"} {
		if got, err := normalizeSingleInput(input); err == nil {
			t.Fatalf("normalizeSingleInput(%q) = %q, nil", input, got)
		}
	}
}
