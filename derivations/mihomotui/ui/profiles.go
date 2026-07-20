package ui

import (
	"context"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"mihomotui/api"
	"mihomotui/internal/termtext"
	"mihomotui/profiles"
)

type profilesModel struct {
	ctx       context.Context
	cli       *api.Client
	store     *profiles.Store
	entries   []profiles.Entry
	activeID  string
	request   profileRequest
	requestNo requestID
	busy      bool
	cursor    int
	scroll    int
	width     int
	height    int
	importing bool
	input     string
}

func newProfilesModel(ctx context.Context, cli *api.Client, store *profiles.Store) *profilesModel {
	model := &profilesModel{ctx: ctx, cli: cli, store: store}
	model.syncFromStore()
	return model
}

func (m *profilesModel) syncFromStore() {
	selected := ""
	if entry, ok := m.entryAt(m.cursor); ok {
		selected = entry.ID
	}
	snapshot := m.store.Snapshot()
	m.entries = append(m.entries[:0], snapshot.Profiles...)
	sort.Slice(m.entries, func(i, j int) bool {
		if m.entries[i].CreatedAt.Equal(m.entries[j].CreatedAt) {
			return m.entries[i].ID < m.entries[j].ID
		}
		return m.entries[i].CreatedAt.Before(m.entries[j].CreatedAt)
	})
	m.activeID = snapshot.ActiveID
	if selected != "" {
		for i := range m.entries {
			if m.entries[i].ID == selected {
				m.cursor = i
				return
			}
		}
	}
	m.cursor = normalizedCursor(m.cursor, len(m.entries))
}

func normalizedCursor(cursor, length int) int {
	if length == 0 || cursor < 0 {
		return 0
	}
	if cursor >= length {
		return length - 1
	}
	return cursor
}

func (m *profilesModel) entryAt(index int) (profiles.Entry, bool) {
	if index < 0 || index >= len(m.entries) {
		return profiles.Entry{}, false
	}
	return m.entries[index], true
}

func (m *profilesModel) load() tea.Cmd {
	m.syncFromStore()
	return nil
}

func (m *profilesModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch value := msg.(type) {
	case profileResultMsg:
		return m.handleProfileResult(value)
	case tea.KeyMsg:
		return m.handleKey(value)
	}
	return nil, "", false
}

func (m *profilesModel) handleKey(key tea.KeyMsg) (tea.Cmd, string, bool) {
	if m.importing {
		return m.handleImportKey(key)
	}
	switch key.Type {
	case tea.KeyUp:
		m.cursor = normalizedCursor(m.cursor-1, len(m.entries))
	case tea.KeyDown:
		m.cursor = normalizedCursor(m.cursor+1, len(m.entries))
	case tea.KeyHome:
		m.cursor = 0
	case tea.KeyEnd:
		m.cursor = normalizedCursor(len(m.entries)-1, len(m.entries))
	case tea.KeyEnter:
		return m.applyProfile(m.cursor)
	case tea.KeyRunes:
		if key.Paste {
			return nil, "", false
		}
		switch string(key.Runes) {
		case "k":
			m.cursor = normalizedCursor(m.cursor-1, len(m.entries))
		case "j":
			m.cursor = normalizedCursor(m.cursor+1, len(m.entries))
		case "i":
			m.importing = true
			m.input = ""
			return nil, "import: enter file path or URL", false
		case "d":
			return m.deleteProfile(m.cursor)
		case "R":
			return m.refreshURL(m.cursor)
		}
	}
	return nil, "", false
}

func (m *profilesModel) handleImportKey(key tea.KeyMsg) (tea.Cmd, string, bool) {
	switch key.Type {
	case tea.KeyEsc:
		m.importing = false
		m.input = ""
		return nil, "", false
	case tea.KeyEnter:
		input, err := normalizeSingleInput(m.input)
		if err != nil {
			return nil, err.Error(), true
		}
		cmd, status, isErr := m.importEntry(input)
		if cmd != nil {
			m.importing = false
			m.input = ""
		}
		return cmd, status, isErr
	case tea.KeyBackspace:
		m.input = removeLastRune(m.input)
	case tea.KeyRunes:
		m.input = appendKeyRunes(m.input, key)
	}
	return nil, "", false
}

func (m *profilesModel) View() string {
	var output strings.Builder
	output.WriteString(renderTextLine(m.width, segment(stTitle, "Profiles")) + "\n\n")
	if m.importing {
		output.WriteString(renderTextLine(m.width,
			segment(stMuted, "Import: "),
			segment(stBase, termtext.Truncate(m.input, m.width)),
			segment(stPlain, "\u2588"),
		) + "\n\n")
		output.WriteString(renderTextLine(m.width, segment(stMuted, "enter confirm  esc cancel")) + "\n")
		return output.String()
	}
	if len(m.entries) == 0 {
		output.WriteString(renderTextLine(m.width, segment(stMuted, "no profiles imported")) + "\n\n")
		output.WriteString(renderTextLine(m.width, segment(stMuted, "[i] import from file or URL")))
		return output.String()
	}

	listHeight := m.height - 6
	if listHeight < 3 {
		listHeight = 3
	}
	m.scroll = clampScroll(m.cursor, m.scroll, listHeight, len(m.entries))
	end := m.scroll + listHeight
	if end > len(m.entries) {
		end = len(m.entries)
	}
	nameWidth := m.width - 24
	if nameWidth < 10 {
		nameWidth = 10
	}
	for i := m.scroll; i < end; i++ {
		entry := m.entries[i]
		cursor := "  "
		cursorStyle := stPlain
		if i == m.cursor {
			cursor, cursorStyle = "\u25b8 ", stMark
		}
		active, activeStyle := " ", stPlain
		if entry.ID == m.activeID {
			active, activeStyle = "●", stMark
		}
		name := termtext.PadRight(entry.Name, nameWidth)
		nameStyle := stPlain
		if i == m.cursor {
			nameStyle = lipgloss.NewStyle().Bold(true)
		}
		output.WriteString(renderTextLine(m.width,
			segment(cursorStyle, cursor),
			segment(activeStyle, active),
			segment(stPlain, " "),
			segment(stCyan, strings.ToUpper(string(entry.Kind))),
			segment(stPlain, " "),
			segment(nameStyle, name),
			segment(stPlain, " "),
			segment(stMuted, entry.UpdatedAt.Format("2006-01-02")),
		) + "\n")
	}

	output.WriteString("\n")
	if entry, ok := m.entryAt(m.cursor); ok {
		output.WriteString(renderTextLine(m.width, segment(stMuted, "file: "), segment(stBase, entry.File)) + "\n")
		if entry.Kind == profiles.KindURL {
			output.WriteString(renderTextLine(m.width, segment(stMuted, "url:  "), segment(stBase, entry.URL)) + "\n")
		}
		output.WriteString(renderTextLine(m.width, segment(stMuted, "time: "), segment(stMuted, entry.UpdatedAt.Format("2006-01-02 15:04:05"))) + "\n")
	}
	return output.String()
}
