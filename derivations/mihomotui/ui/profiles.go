package ui

import (
	"context"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"mihomotui/api"
)

type profileEntry struct {
	kind  string    // "FILE" or "URL"
	name  string    // display name
	path  string    // local file path
	url   string    // original URL (empty for FILE)
	added time.Time // import timestamp
}

type profileAppliedMsg struct {
	path string
	err  error
}

type profileDownloadedMsg struct {
	url       string
	path      string
	autoApply bool
	err       error
}

type profileImportErrMsg struct {
	msg string
}

type profilesModel struct {
	cli       *api.Client
	entries   []profileEntry
	cursor    int
	scroll    int
	width     int
	height    int
	importing bool
	input     string
}

func newProfilesModel(cli *api.Client) *profilesModel {
	return &profilesModel{cli: cli}
}

func (m *profilesModel) load() tea.Cmd {
	return nil
}

func (m *profilesModel) applyProfile(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.entries) {
		return nil
	}
	e := m.entries[idx]
	cli := m.cli
	p := e.path
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*reqTimeout)
		defer cancel()
		err := cli.PutConfig(ctx, p)
		return profileAppliedMsg{path: p, err: err}
	}
}

func (m *profilesModel) importEntry(input string) tea.Cmd {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	cli := m.cli
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		rawURL := input
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 30*reqTimeout)
			defer cancel()
			path, err := cli.DownloadConfig(ctx, rawURL)
			return profileDownloadedMsg{url: rawURL, path: path, err: err}
		}
	}
	path := input
	if _, err := os.Stat(path); err != nil {
		return func() tea.Msg { return profileImportErrMsg{msg: "file not found: " + path} }
	}
	m.entries = append(m.entries, profileEntry{
		kind:  "FILE",
		name:  pathFileName(path),
		path:  path,
		added: time.Now(),
	})
	return nil
}

func pathFileName(p string) string {
	parts := strings.Split(strings.ReplaceAll(p, "\\", "/"), "/")
	if len(parts) == 0 {
		return p
	}
	return parts[len(parts)-1]
}

func (m *profilesModel) refreshURL(idx int) tea.Cmd {
	if idx < 0 || idx >= len(m.entries) {
		return nil
	}
	e := m.entries[idx]
	if e.url == "" {
		return nil
	}
	cli := m.cli
	rawURL := e.url
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*reqTimeout)
		defer cancel()
		path, err := cli.DownloadConfig(ctx, rawURL)
		return profileDownloadedMsg{url: rawURL, path: path, autoApply: true, err: err}
	}
}

func (m *profilesModel) removeEntry(idx int) {
	if idx < 0 || idx >= len(m.entries) {
		return
	}
	m.entries = append(m.entries[:idx], m.entries[idx+1:]...)
	if m.cursor >= len(m.entries) {
		m.cursor = len(m.entries) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *profilesModel) Update(msg tea.Msg) (tea.Cmd, string, bool) {
	switch v := msg.(type) {
	case profileAppliedMsg:
		if v.err != nil {
			return nil, "apply failed: " + v.err.Error(), true
		}
		return nil, "config applied: " + v.path, false
	case profileImportErrMsg:
		return nil, v.msg, true
	case profileDownloadedMsg:
		if v.err != nil {
			return nil, "download failed: " + v.err.Error(), true
		}
		m.entries = append(m.entries, profileEntry{
			kind:  "URL",
			name:  pathFileName(v.path),
			path:  v.path,
			url:   v.url,
			added: time.Now(),
		})
		if v.autoApply {
			return m.applyProfile(len(m.entries)-1), "downloaded, applying", false
		}
		return nil, "downloaded: " + v.url, false
	case tea.KeyMsg:
		return m.handleKey(v)
	}
	return nil, "", false
}

func (m *profilesModel) handleKey(k tea.KeyMsg) (tea.Cmd, string, bool) {
	if m.importing {
		return m.handleImportKey(k)
	}
	switch k.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	case "home":
		m.cursor = 0
	case "end":
		m.cursor = len(m.entries) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	case "i":
		m.importing = true
		m.input = ""
		return nil, "import: enter file path or URL", false
	case "enter":
		return m.applyProfile(m.cursor), "applying profile", false
	case "d":
		m.removeEntry(m.cursor)
		return nil, "profile removed", false
	case "R":
		return m.refreshURL(m.cursor), "refreshing URL", false
	}
	return nil, "", false
}

func (m *profilesModel) handleImportKey(k tea.KeyMsg) (tea.Cmd, string, bool) {
	switch k.String() {
	case "esc":
		m.importing = false
		m.input = ""
		return nil, "", false
	case "enter":
		m.importing = false
		cmd := m.importEntry(m.input)
		return cmd, "importing", false
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	default:
		s := k.String()
		if len(s) == 1 {
			m.input += s
		}
	}
	return nil, "", false
}

func (m *profilesModel) View() string {
	var b strings.Builder
	b.WriteString(stTitle.Render("Profiles") + "\n\n")

	if m.importing {
		b.WriteString(stMuted.Render("Import: ") + m.input + "\u2588\n\n")
		b.WriteString(stMuted.Render("enter confirm  esc cancel") + "\n")
		return b.String()
	}

	if len(m.entries) == 0 {
		b.WriteString(stMuted.Render("no profiles imported") + "\n\n")
		b.WriteString(stMuted.Render("[i] import from file or URL"))
		return b.String()
	}

	// List
	listH := m.height - 6
	if listH < 3 {
		listH = 3
	}
	m.scroll = clampScroll(m.cursor, m.scroll, listH, len(m.entries))
	end := m.scroll + listH
	if end > len(m.entries) {
		end = len(m.entries)
	}

	nameW := m.width - 20
	if nameW < 10 {
		nameW = 10
	}

	for i := m.scroll; i < end; i++ {
		e := m.entries[i]
		cur := "  "
		if i == m.cursor {
			cur = stMark.Render("\u25b8 ")
		}
		badge := stCyan.Render("FILE")
		if e.kind == "URL" {
			badge = stWarn.Render(" URL")
		}
		display := e.name
		if i == m.cursor {
			display = lipgloss.NewStyle().Bold(true).Render(display)
		}
		line := cur + badge + " " + padR(trunc(display, nameW), nameW) + " " + stMuted.Render(e.added.Format("2006-01-02"))
		b.WriteString(truncWide(line, m.width) + "\n")
	}

	// Detail
	b.WriteString("\n")
	if m.cursor >= 0 && m.cursor < len(m.entries) {
		e := m.entries[m.cursor]
		b.WriteString(stMuted.Render("path: ") + stBase.Render(e.path) + "\n")
		if e.url != "" {
			b.WriteString(stMuted.Render("url:  ") + stBase.Render(e.url) + "\n")
		}
		b.WriteString(stMuted.Render("time: ") + stMuted.Render(e.added.Format("2006-01-02 15:04:05")) + "\n")
	}

	return b.String()
}
