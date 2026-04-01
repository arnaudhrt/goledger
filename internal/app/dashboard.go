package app

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/arnaudhrt/goledger/internal/db"
)

// Styles
var (
	green = lipgloss.Color("#4ade80")
	red   = lipgloss.Color("#f87171")
	blue  = lipgloss.Color("#60a5fa")
	dim   = lipgloss.Color("#6b7280")

	styleIncome     = lipgloss.NewStyle().Foreground(green)
	styleExpense    = lipgloss.NewStyle().Foreground(red)
	styleInvestment = lipgloss.NewStyle().Foreground(blue)
	styleDim        = lipgloss.NewStyle().Foreground(dim)
	styleBold       = lipgloss.NewStyle().Bold(true)
	styleHighlight  = lipgloss.NewStyle().Reverse(true)
	styleWarning    = lipgloss.NewStyle().Foreground(lipgloss.Color("#fbbf24"))
	styleSaved      = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b7280"))

	catColors = []lipgloss.Color{
		"#fb7185", // coral
		"#fbbf24", // amber
		"#60a5fa", // blue
		"#f472b6", // pink
		"#a78bfa", // purple
		"#9ca3af", // gray
	}
)

// catSummary holds aggregated data for a category.
type catSummary struct {
	Name    string
	Total   float64
	Percent float64
	Subs    []catSummary
}

func (m Model) dashboardView() string {
	if m.err != nil {
		return fmt.Sprintf("  Error: %v\n", m.err)
	}

	w := m.viewWidth()

	saved := m.totalInc - m.totalExp - m.totalInv
	pad := "  "

	var s strings.Builder
	s.WriteString("\n")

	// ── Header ──
	monthLabel := fmt.Sprintf("◂ %s %d ▸", m.Month.String(), m.Year)
	title := styleBold.Render("GoLedger")
	gap := w - 4 - lipgloss.Width(title) - len(monthLabel)
	if gap < 2 {
		gap = 2
	}
	s.WriteString(pad + title + strings.Repeat(" ", gap) + monthLabel + "\n")
	s.WriteString(pad + styleDim.Render(fmt.Sprintf("Currency: %s", m.Config.DisplayCurrency)) + "\n\n")

	// ── Three-lane bar ──
	// Bar is relative to income: expenses + investments + saved = income
	barW := w - 4
	if barW < 20 {
		barW = 20
	}

	total := m.totalInc + m.totalExp + m.totalInv
	if total > 0 {
		spending := m.totalExp + m.totalInv
		overspent := m.totalInc > 0 && spending > m.totalInc

		if overspent {
			// Bar scaled to spending, with green income marker
			expW := int(math.Round(float64(barW) * m.totalExp / spending))
			incomePos := int(math.Round(float64(barW) * m.totalInc / spending))
			if incomePos >= barW {
				incomePos = barW - 1
			}

			// Build bar with marker splitting the segments
			beforeExp := min(incomePos, expW)
			beforeInv := max(0, incomePos-expW)
			afterStart := incomePos + 1
			afterExp := max(0, expW-afterStart)
			afterInv := barW - incomePos - 1 - afterExp

			bar := styleExpense.Render(strings.Repeat("█", beforeExp)) +
				styleInvestment.Render(strings.Repeat("█", beforeInv)) +
				styleIncome.Render("┃") +
				styleExpense.Render(strings.Repeat("█", afterExp)) +
				styleInvestment.Render(strings.Repeat("█", max(0, afterInv)))
			s.WriteString(pad + bar + "\n\n")
		} else if m.totalInc > 0 {
			// Normal bar: expenses + investments + saved
			expW := int(math.Round(float64(barW) * m.totalExp / m.totalInc))
			invW := int(math.Round(float64(barW) * m.totalInv / m.totalInc))
			savedW := barW - expW - invW
			if savedW < 0 {
				savedW = 0
			}

			bar := styleExpense.Render(strings.Repeat("█", expW)) +
				styleInvestment.Render(strings.Repeat("█", invW)) +
				styleSaved.Render(strings.Repeat("█", savedW))
			s.WriteString(pad + bar + "\n\n")
		} else {
			// No income: bar = expenses + investments
			expW := int(math.Round(float64(barW) * m.totalExp / spending))
			invW := barW - expW

			bar := styleExpense.Render(strings.Repeat("█", expW)) +
				styleInvestment.Render(strings.Repeat("█", invW))
			s.WriteString(pad + bar + "\n\n")
		}

		// Breakdown lines
		if m.totalInc > 0 {
			expPct := m.totalExp / m.totalInc * 100
			invPct := m.totalInv / m.totalInc * 100
			s.WriteString(pad + styleIncome.Render(fmt.Sprintf("income     %8s %s", fmtAmount(m.totalInc), m.Config.DisplayCurrency)) + "\n")
			s.WriteString(pad + styleExpense.Render(fmt.Sprintf("expenses   %8s %s  %3.0f%%", fmtAmount(m.totalExp), m.Config.DisplayCurrency, expPct)) + "\n")
			s.WriteString(pad + styleInvestment.Render(fmt.Sprintf("invest     %8s %s  %3.0f%%", fmtAmount(m.totalInv), m.Config.DisplayCurrency, invPct)) + "\n")
			if !overspent {
				savedPct := saved / m.totalInc * 100
				s.WriteString(pad + styleSaved.Render(fmt.Sprintf("saved      %8s %s  %3.0f%%", fmtAmount(saved), m.Config.DisplayCurrency, savedPct)) + "\n")
			}
			s.WriteString("\n")
		} else {
			s.WriteString(pad + styleIncome.Render(fmt.Sprintf("income     %8s %s", fmtAmount(0.0), m.Config.DisplayCurrency)) + "\n")
			s.WriteString(pad + styleExpense.Render(fmt.Sprintf("expenses   %8s %s", fmtAmount(m.totalExp), m.Config.DisplayCurrency)) + "\n")
			s.WriteString(pad + styleInvestment.Render(fmt.Sprintf("invest     %8s %s", fmtAmount(m.totalInv), m.Config.DisplayCurrency)) + "\n\n")
		}
	} else {
		s.WriteString(pad + styleDim.Render("No data this month") + "\n\n")
	}

	// ── Category breakdown ──
	if len(m.categories) > 0 {
		s.WriteString(pad + styleBold.Render("Expenses by category") + "\n")
		s.WriteString(pad + styleDim.Render(strings.Repeat("─", min(w-4, 50))) + "\n")

		maxBarW := w - 4 - 15 - 22
		if maxBarW < 10 {
			maxBarW = 10
		}

		for i, cat := range m.categories {
			color := catColors[i%len(catColors)]
			barLen := int(math.Round(cat.Percent / 100 * float64(maxBarW)))
			if barLen < 1 && cat.Total > 0 {
				barLen = 1
			}

			bar := lipgloss.NewStyle().Foreground(color).Render(strings.Repeat("█", barLen))
			line := fmt.Sprintf("  %-13s %s %8s %s  %3.0f%%",
				cat.Name, bar, fmtAmount(cat.Total), m.Config.DisplayCurrency, cat.Percent)

			if i == m.cursor {
				s.WriteString(pad + styleHighlight.Render(line) + "\n")
			} else {
				s.WriteString(pad + line + "\n")
			}

			if m.showSubs && len(cat.Subs) > 0 {
				for _, sub := range cat.Subs {
					subLine := fmt.Sprintf("    %-13s %8s %s",
						sub.Name, fmtAmount(sub.Total), m.Config.DisplayCurrency)
					s.WriteString(pad + styleDim.Render(subLine) + "\n")
				}
			}
		}
	}
	s.WriteString("\n")

	// ── Recent entries ──
	if len(m.entries) == 0 {
		s.WriteString(pad + styleDim.Render("No entries this month. Press 'b' to bulk paste or 'a' to add one.") + "\n")
	} else {
		s.WriteString(pad + styleBold.Render(fmt.Sprintf("Recent entries (%d total)", len(m.entries))) + "\n")
		s.WriteString(pad + styleDim.Render(strings.Repeat("─", min(w-4, 50))) + "\n")

		show := m.entries
		if len(show) > 5 {
			show = show[len(show)-5:]
		}
		for _, e := range show {
			typeStyle := styleDim
			switch e.Type {
			case db.Income:
				typeStyle = styleIncome
			case db.Expense:
				typeStyle = styleExpense
			case db.Investment:
				typeStyle = styleInvestment
			}

			note := e.Note
			if len(note) > 20 {
				note = note[:17] + "..."
			}

			s.WriteString(fmt.Sprintf("%s  %s  %s  %-20s %8s %s  %s\n",
				pad,
				e.Date.Format("02/01"),
				typeStyle.Render(string(e.Type)),
				note,
				fmtAmount(e.Amount),
				e.Currency,
				styleDim.Render(e.Category)))
		}
	}
	s.WriteString("\n")

	// ── Footer ──
	s.WriteString(pad + styleDim.Render("← → month  ↑↓ scroll  t subs  enter drill  a add  b bulk  s settings  q quit") + "\n\n")

	return s.String()
}

func aggregateCategories(entries []db.Entry, totalExp float64) []catSummary {
	parentTotals := make(map[string]float64)
	subTotals := make(map[string]map[string]float64)

	for _, e := range entries {
		if e.Type != db.Expense {
			continue
		}
		cat := e.Category
		if cat == "" {
			cat = "other"
		}
		parent := cat
		sub := ""
		if idx := strings.Index(cat, ":"); idx >= 0 {
			parent = cat[:idx]
			sub = cat
		}
		parentTotals[parent] += e.Amount
		if sub != "" {
			if subTotals[parent] == nil {
				subTotals[parent] = make(map[string]float64)
			}
			subTotals[parent][sub] += e.Amount
		}
	}

	var cats []catSummary
	for name, total := range parentTotals {
		pct := 0.0
		if totalExp > 0 {
			pct = total / totalExp * 100
		}
		cat := catSummary{Name: name, Total: total, Percent: pct}

		if subs, ok := subTotals[name]; ok {
			for subName, subTotal := range subs {
				cat.Subs = append(cat.Subs, catSummary{Name: subName, Total: subTotal})
			}
			sort.Slice(cat.Subs, func(i, j int) bool {
				return cat.Subs[i].Total > cat.Subs[j].Total
			})
		}
		cats = append(cats, cat)
	}

	sort.Slice(cats, func(i, j int) bool {
		return cats[i].Total > cats[j].Total
	})

	return cats
}

func fmtAmount(n float64) string {
	neg := n < 0
	abs := math.Abs(n)
	s := fmt.Sprintf("%.0f", abs)
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var buf []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && (len(s)-i)%3 == 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, s[i])
	}
	if neg {
		return "-" + string(buf)
	}
	return string(buf)
}
