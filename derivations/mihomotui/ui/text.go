package ui

import (
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func appendKeyRunes(input string, key tea.KeyMsg) string {
	if key.Type != tea.KeyRunes {
		return input
	}
	return input + string(key.Runes)
}

func removeLastRune(input string) string {
	runes := []rune(input)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func normalizeSingleInput(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", errors.New("profile source is empty")
	}
	if strings.ContainsAny(input, "\r\n") {
		return "", errors.New("profile source must be a single line")
	}
	return input, nil
}
