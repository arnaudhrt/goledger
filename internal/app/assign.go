package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/arnaudhrt/goledger/internal/db"
	"github.com/arnaudhrt/goledger/internal/parser"
)

type assignState struct {
	input  textinput.Model
	idx    int      // index into bulk.parsed
	cats   []string // filtered category suggestions
	cursor int      // cursor in suggestions
}

func (m *Model) startAssign() bool {
	for i, p := range m.bulk.parsed {
		if p.Status == parser.StatusNeedCategory {
			m.assign.idx = i
			m.initAssignInput()
			return true
		}
	}
	return false
}

func (m *Model) initAssignInput() {
	ti := textinput.New()
	ti.Placeholder = "Type category..."
	ti.Focus()
	ti.CharLimit = 50
	m.assign.input = ti
	m.assign.cursor = 0
	m.refreshAssignFilter()
}

func (m *Model) refreshAssignFilter() {
	entry := m.bulk.parsed[m.assign.idx].Entry
	cats := m.categoriesForType(entry.Type)
	input := strings.ToLower(m.assign.input.Value())
	if input == "" {
		m.assign.cats = cats
	} else {
		m.assign.cats = nil
		for _, c := range cats {
			if strings.Contains(strings.ToLower(c), input) {
				m.assign.cats = append(m.assign.cats, c)
			}
		}
	}
	if m.assign.cursor >= len(m.assign.cats) {
		m.assign.cursor = max(0, len(m.assign.cats)-1)
	}
}

func (m *Model) categoriesForType(t db.EntryType) []string {
	switch t {
	case db.Expense:
		return m.Config.Categories.Exp
	case db.Income:
		return m.Config.Categories.Inc
	case db.Investment:
		return m.Config.Categories.Inv
	}
	return nil
}

func (m *Model) advanceAssign() bool {
	for i := m.assign.idx + 1; i < len(m.bulk.parsed); i++ {
		if m.bulk.parsed[i].Status == parser.StatusNeedCategory {
			m.assign.idx = i
			m.initAssignInput()
			return true
		}
	}
	return false
}

func (m Model) updateAssign(msg tea.Msg) (Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.Screen = ScreenBulkPaste
			return m, nil
		case "enter":
			var cat string
			if len(m.assign.cats) > 0 && m.assign.cursor < len(m.assign.cats) {
				cat = m.assign.cats[m.assign.cursor]
			} else if v := m.assign.input.Value(); v != "" {
				cat = v // new category typed by user
			}
			if cat != "" {
				m.bulk.parsed[m.assign.idx].Entry.Category = cat
				m.bulk.parsed[m.assign.idx].Status = parser.StatusOK
			}
			if !m.advanceAssign() {
				m.Screen = ScreenBulkPaste
			}
			return m, nil
		case "tab":
			if !m.advanceAssign() {
				m.Screen = ScreenBulkPaste
			}
			return m, nil
		case "up":
			if m.assign.cursor > 0 {
				m.assign.cursor--
			}
			return m, nil
		case "down":
			if m.assign.cursor < len(m.assign.cats)-1 {
				m.assign.cursor++
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.assign.input, cmd = m.assign.input.Update(msg)
	m.refreshAssignFilter()
	return m, cmd
}

func (m Model) assignView() string {
	if m.assign.idx >= len(m.bulk.parsed) {
		return "  No entry to assign.\n"
	}

	e := m.bulk.parsed[m.assign.idx].Entry

	// Count remaining uncategorized
	remaining := 0
	for _, pp := range m.bulk.parsed {
		if pp.Status == parser.StatusNeedCategory {
			remaining++
		}
	}

	w := m.viewWidth()
	pad := "  "

	var s strings.Builder
	s.WriteString("\n")
	s.WriteString(pad + styleBold.Render("Assign Category") +
		styleDim.Render(fmt.Sprintf("  (%d remaining)", remaining)) + "\n")
	s.WriteString(pad + styleDim.Render(strings.Repeat("─", min(w-4, 50))) + "\n\n")

	// Entry details
	typeStyle := styleDim
	switch e.Type {
	case db.Income:
		typeStyle = styleIncome
	case db.Expense:
		typeStyle = styleExpense
	case db.Investment:
		typeStyle = styleInvestment
	}
	s.WriteString(fmt.Sprintf("%s  %s  %s  %s  %s %s\n\n",
		pad,
		e.Date.Format("02/01"),
		typeStyle.Render(string(e.Type)),
		e.Note,
		fmtAmount(e.Amount),
		e.Currency))

	// Text input
	s.WriteString(pad + "Category: " + m.assign.input.View() + "\n\n")

	// Suggestion list
	maxShow := 10
	if len(m.assign.cats) > 0 {
		show := m.assign.cats
		if len(show) > maxShow {
			show = show[:maxShow]
		}
		for i, cat := range show {
			if i == m.assign.cursor {
				s.WriteString(pad + styleHighlight.Render("  > "+cat) + "\n")
			} else {
				s.WriteString(fmt.Sprintf("%s    %s\n", pad, cat))
			}
		}
		if len(m.assign.cats) > maxShow {
			s.WriteString(pad + styleDim.Render(fmt.Sprintf("    ... %d more", len(m.assign.cats)-maxShow)) + "\n")
		}
	} else if v := m.assign.input.Value(); v != "" {
		s.WriteString(pad + styleDim.Render("  (new category: "+v+")") + "\n")
	}

	s.WriteString("\n")
	s.WriteString(pad + styleDim.Render("enter confirm  tab skip  ↑↓ navigate  esc back") + "\n\n")
	return s.String()
}
