package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// helpView renders keybindings with wrapping when the window is narrow.
func (m Model) helpView(maxW int) string {
	var bindings []key.Binding
	if m.helpAll {
		for _, group := range m.keys.FullHelp() {
			bindings = append(bindings, group...)
		}
	} else {
		bindings = m.keys.ShortHelp()
	}

	sep := styleGray.Render(" • ")
	sepW := lipgloss.Width(sep)
	pad := "  "

	var lines []string
	var line strings.Builder
	lineW := 0

	for i, b := range bindings {
		item := styleGray.Render(b.Help().Key + " " + b.Help().Desc)
		itemW := lipgloss.Width(item)

		addSep := i > 0 && lineW > 0
		needed := itemW
		if addSep {
			needed += sepW
		}

		if lineW > 0 && lineW+needed > maxW {
			lines = append(lines, pad+line.String())
			line.Reset()
			lineW = 0
			addSep = false
		}

		if addSep {
			line.WriteString(sep)
			lineW += sepW
		}
		line.WriteString(item)
		lineW += itemW
	}
	if lineW > 0 {
		lines = append(lines, pad+line.String())
	}

	return strings.Join(lines, "\n")
}

// dashboardKeyMap defines the keybindings shown on the dashboard screen.
type dashboardKeyMap struct {
	Left  key.Binding
	Right key.Binding
	Up    key.Binding
	Down  key.Binding
	Subs  key.Binding
	Drill key.Binding
	Add   key.Binding
	Bulk  key.Binding
	Set   key.Binding
	Help  key.Binding
	Quit  key.Binding
}

// ShortHelp returns keybindings for the compact help line.
func (k dashboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Left, k.Up, k.Subs, k.Bulk, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view.
func (k dashboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Left, k.Right},
		{k.Up, k.Down},
		{k.Subs, k.Drill},
		{k.Add, k.Bulk, k.Set},
		{k.Help, k.Quit},
	}
}

var dashboardKeys = dashboardKeyMap{
	Left: key.NewBinding(
		key.WithKeys("left"),
		key.WithHelp("←/→", "month"),
	),
	Right: key.NewBinding(
		key.WithKeys("right"),
		key.WithHelp("→", "next month"),
	),
	Up: key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑/↓", "scroll"),
	),
	Down: key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "scroll down"),
	),
	Subs: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "subs"),
	),
	Drill: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "drill"),
	),
	Add: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "add"),
	),
	Bulk: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "bulk"),
	),
	Set: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "settings"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
}
