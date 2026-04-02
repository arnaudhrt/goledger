package app

import (
	"fmt"
	"time"

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
	if m.cursor >= len(m.categories) {
		m.cursor = max(0, len(m.categories)-1)
	}
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
	switch keyMsg.String() {
	case "q":
		return m, tea.Quit
	case "left":
		m.Month--
		if m.Month < time.January {
			m.Month = time.December
			m.Year--
		}
		m.cursor = 0
		m.loadMonth()
		return m, nil
	case "right":
		m.Month++
		if m.Month > time.December {
			m.Month = time.January
			m.Year++
		}
		m.cursor = 0
		m.loadMonth()
		return m, nil
	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down":
		if len(m.categories) > 0 && m.cursor < len(m.categories)-1 {
			m.cursor++
		}
		return m, nil
	case "t":
		m.showSubs = !m.showSubs
		return m, nil
	case "b":
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
