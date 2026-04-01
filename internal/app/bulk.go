package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/arnaudhrt/goledger/internal/db"
	"github.com/arnaudhrt/goledger/internal/parser"
)

type bulkState struct {
	textarea textarea.Model
	parsed   []parser.ParseResult
}

func (m *Model) enterBulk() {
	ta := textarea.New()
	ta.Placeholder = "Paste entries here... (DD/MM TYPE description amount [currency] [category])"
	ta.Focus()
	ta.CharLimit = 0
	w := m.viewWidth() - 4
	ta.SetWidth(w)
	ta.SetHeight(8)
	m.bulk.textarea = ta
	m.bulk.parsed = nil
}

func (m Model) updateBulk(msg tea.Msg) (Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.Screen = ScreenDashboard
			m.loadMonth()
			return m, nil
		case "ctrl+s":
			m.saveBulkEntries()
			m.Screen = ScreenDashboard
			m.loadMonth()
			return m, nil
		case "enter":
			if m.startAssign() {
				m.Screen = ScreenAssign
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.bulk.textarea, cmd = m.bulk.textarea.Update(msg)
	m.bulk.parsed = parser.ParseBulk(
		m.bulk.textarea.Value(),
		m.Config.DisplayCurrency,
		time.Now().Year(),
	)
	return m, cmd
}

func (m *Model) saveBulkEntries() {
	var entries []db.Entry
	for _, p := range m.bulk.parsed {
		if p.Status != parser.StatusError {
			entries = append(entries, p.Entry)
		}
	}
	if len(entries) > 0 {
		_ = m.DB.InsertEntries(entries)
	}
}

func (m Model) bulkView() string {
	w := m.viewWidth()
	pad := "  "
	sep := styleDim.Render(strings.Repeat("─", min(w-4, 50)))

	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(pad + styleBold.Render("Bulk Paste") + "\n")
	s.WriteString(pad + sep + "\n\n")
	s.WriteString(pad + m.bulk.textarea.View() + "\n\n")

	if len(m.bulk.parsed) > 0 {
		s.WriteString(pad + styleBold.Render("Preview") + "\n")
		s.WriteString(pad + sep + "\n")

		for _, p := range m.bulk.parsed {
			switch p.Status {
			case parser.StatusOK:
				e := p.Entry
				note := e.Note
				if len(note) > 20 {
					note = note[:17] + "..."
				}
				s.WriteString(fmt.Sprintf("%s  %s  %s  %-3s  %-20s %8s %s  %s\n",
					pad,
					styleIncome.Render("✓"),
					e.Date.Format("02/01"),
					string(e.Type),
					note,
					fmtAmount(e.Amount),
					e.Currency,
					styleDim.Render(e.Category)))

			case parser.StatusNeedCategory:
				e := p.Entry
				note := e.Note
				if len(note) > 20 {
					note = note[:17] + "..."
				}
				s.WriteString(fmt.Sprintf("%s  %s  %s  %-3s  %-20s %8s %s  %s\n",
					pad,
					styleWarning.Render("?"),
					e.Date.Format("02/01"),
					string(e.Type),
					note,
					fmtAmount(e.Amount),
					e.Currency,
					styleDim.Render("—")))

			case parser.StatusError:
				s.WriteString(fmt.Sprintf("%s  %s  %s\n",
					pad,
					styleExpense.Render("✗"),
					styleExpense.Render(p.Error)))
			}
		}
	}

	s.WriteString("\n")
	s.WriteString(pad + styleDim.Render("enter assign categories  ctrl+s save  esc cancel") + "\n\n")
	return s.String()
}
