package app

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/arnaudhrt/goledger/internal/config"
	"github.com/arnaudhrt/goledger/internal/db"
)

const maxWidth = 120

// Screen identifies which screen is currently active.
type Screen int

const (
	ScreenDashboard Screen = iota
	ScreenBulkPaste
	ScreenAssign
	ScreenDrilldown
	ScreenAddEntry
	ScreenSettings
)

// Model is the top-level Bubbletea model.
type Model struct {
	DB     *db.DB
	Config config.Config
	Screen Screen

	// Current month context
	Year  int
	Month time.Month

	// Cached month data
	entries    []db.Entry
	categories []catSummary
	totalInc   float64
	totalExp   float64
	totalInv   float64
	err        error

	// Dashboard state
	cursor   int
	showSubs bool

	// Bulk paste & category assignment state
	bulk   bulkState
	assign assignState

	// Progress bars
	progIncome  progress.Model
	progExpense progress.Model
	progInvest  progress.Model

	// Help
	keys    dashboardKeyMap
	helpAll bool

	width  int
	height int
}

func (m Model) viewWidth() int {
	w := m.width
	if w < 40 {
		w = 80
	}
	if w > maxWidth {
		w = maxWidth
	}
	return w
}

// New creates the initial app model.
func New(database *db.DB, cfg config.Config) Model {
	now := time.Now()
	m := Model{
		DB:          database,
		Config:      cfg,
		Screen:      ScreenDashboard,
		Year:        now.Year(),
		Month:       now.Month(),
		progIncome:  progress.New(progress.WithScaledGradient("#73E2A7", "#1B9E5C"), progress.WithoutPercentage()),
		progExpense: progress.New(progress.WithScaledGradient("#F28B82", "#C0392B"), progress.WithoutPercentage()),
		progInvest:  progress.New(progress.WithScaledGradient("#7EC8E3", "#2E86AB"), progress.WithoutPercentage()),
		keys: dashboardKeys,
	}
	m.loadMonth()
	return m
}

func (m *Model) loadMonth() {
	entries, err := m.DB.EntriesByMonth(m.Year, m.Month)
	if err != nil {
		m.err = err
		return
	}
	m.err = nil
	m.entries = entries

	m.totalInc, m.totalExp, m.totalInv = 0, 0, 0
	for _, e := range entries {
		switch e.Type {
		case db.Income:
			m.totalInc += e.Amount
		case db.Expense:
			m.totalExp += e.Amount
		case db.Investment:
			m.totalInv += e.Amount
		}
	}
	m.categories = aggregateCategories(entries, m.totalExp)
	if maxC := m.cursorMax(); maxC == 0 {
		m.cursor = 0
	} else if m.cursor >= maxC {
		m.cursor = maxC - 1
	}
}

// cursorMax returns the total number of selectable items in the category list.
func (m Model) cursorMax() int {
	n := len(m.categories)
	if m.showSubs {
		for _, cat := range m.categories {
			n += len(cat.Subs)
		}
	}
	return n
}

// cursorPos maps the flat cursor index to (category index, sub index).
// subIdx is -1 when the cursor is on a parent category.
func (m Model) cursorPos() (catIdx, subIdx int) {
	pos := 0
	for i, cat := range m.categories {
		if pos == m.cursor {
			return i, -1
		}
		pos++
		if m.showSubs {
			for j := range cat.Subs {
				if pos == m.cursor {
					return i, j
				}
				pos++
			}
		}
	}
	return 0, -1
}

func (m Model) Init() tea.Cmd {
	return tea.WindowSize()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		barW := m.viewWidth() - 4
		if barW < 20 {
			barW = 20
		}
		m.progIncome.Width = barW
		m.progExpense.Width = barW
		m.progInvest.Width = barW
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	switch m.Screen {
	case ScreenDashboard:
		return m.updateDashboard(msg)
	case ScreenBulkPaste:
		return m.updateBulk(msg)
	case ScreenAssign:
		return m.updateAssign(msg)
	}
	return m, nil
}

func (m Model) updateDashboard(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(keyMsg, m.keys.Left):
		m.Month--
		if m.Month < time.January {
			m.Month = time.December
			m.Year--
		}
		m.cursor = 0
		m.loadMonth()
		return m, nil
	case key.Matches(keyMsg, m.keys.Right):
		m.Month++
		if m.Month > time.December {
			m.Month = time.January
			m.Year++
		}
		m.cursor = 0
		m.loadMonth()
		return m, nil
	case key.Matches(keyMsg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case key.Matches(keyMsg, m.keys.Down):
		if maxC := m.cursorMax(); m.cursor < maxC-1 {
			m.cursor++
		}
		return m, nil
	case key.Matches(keyMsg, m.keys.Subs):
		catIdx, _ := m.cursorPos()
		m.showSubs = !m.showSubs
		pos := 0
		for i := 0; i < catIdx; i++ {
			pos++
			if m.showSubs {
				pos += len(m.categories[i].Subs)
			}
		}
		m.cursor = pos
		return m, nil
	case key.Matches(keyMsg, m.keys.Help):
		m.helpAll = !m.helpAll
		return m, nil
	case key.Matches(keyMsg, m.keys.Bulk):
		m.enterBulk()
		m.Screen = ScreenBulkPaste
		return m, nil
	}
	return m, nil
}

func (m Model) View() string {
	switch m.Screen {
	case ScreenDashboard:
		return m.dashboardView()
	case ScreenBulkPaste:
		return m.bulkView()
	case ScreenAssign:
		return m.assignView()
	default:
		return fmt.Sprintf("\n  Screen %d — not implemented yet. Press esc to go back.\n\n", m.Screen)
	}
}
