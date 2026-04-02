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
	green     = lipgloss.Color("#73E2A7")
	red       = lipgloss.Color("#F28B82")
	blue      = lipgloss.Color("#7EC8E3")
	yellow    = lipgloss.Color("#DBAB79")
	dim       = lipgloss.Color("#B9BFCA")
	gray      = lipgloss.Color("#5a616c")
	muted     = lipgloss.Color("#393d44")
	selPurple = lipgloss.Color("#224044")

	styleIncome     = lipgloss.NewStyle().Foreground(green)
	styleExpense    = lipgloss.NewStyle().Foreground(red)
	styleInvestment = lipgloss.NewStyle().Foreground(blue)
	styleDim        = lipgloss.NewStyle().Foreground(dim)
	styleMuted      = lipgloss.NewStyle().Foreground(muted)
	styleGray       = lipgloss.NewStyle().Foreground(gray)
	styleBold       = lipgloss.NewStyle().Bold(true)
	styleHighlight  = lipgloss.NewStyle().Reverse(true)
	styleSel        = lipgloss.NewStyle().Background(selPurple)
	styleWarning    = lipgloss.NewStyle().Foreground(yellow)
	styleSaved      = lipgloss.NewStyle().Foreground(dim)

	catColors = []lipgloss.Color{
		"#E88388", // red
		"#DBAB79", // yellow
		"#71BEF2", // blue
		"#D290E4", // magenta
		"#66C2CD", // cyan
		"#A8CC8C", // green
		"#E8A0BF", // pink
		"#B5CEA8", // sage
		"#CF9EF2", // violet
		"#F2C97D", // gold
		"#7DC4E4", // sky
		"#E89B73", // orange
		"#A6D8D2", // teal
		"#C4A7E7", // lavender
		"#B9BFCA", // gray
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

	// ── Header (bordered box) ──
	boxW := w - 4
	if boxW < 20 {
		boxW = 20
	}
	innerW := boxW - 2

	titleText := " GoLedger "
	topAfter := innerW - len(titleText)
	if topAfter < 0 {
		topAfter = 0
	}
	topBorder := styleDim.Render("╭") + styleBold.Render(titleText) + styleDim.Render(strings.Repeat("─", topAfter)+"╮")
	botBorder := styleDim.Render("╰" + strings.Repeat("─", innerW) + "╯")

	monthLabel := fmt.Sprintf("◂  %s %d  ▸", m.Month.String(), m.Year)
	currLabel := fmt.Sprintf("Currency: %s", m.Config.DisplayCurrency)

	centerLine := func(text string) string {
		textW := lipgloss.Width(text)
		leftPad := (innerW - textW) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		rightPad := innerW - textW - leftPad
		if rightPad < 0 {
			rightPad = 0
		}
		return styleDim.Render("│") + strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad) + styleDim.Render("│")
	}

	s.WriteString(pad + topBorder + "\n")
	s.WriteString(pad + centerLine("") + "\n")
	s.WriteString(pad + centerLine(styleBold.Render(monthLabel)) + "\n")
	s.WriteString(pad + centerLine(styleDim.Render(currLabel)) + "\n")
	s.WriteString(pad + centerLine("") + "\n")
	s.WriteString(pad + botBorder + "\n\n")

	// ── Progress bars ──
	total := m.totalInc + m.totalExp + m.totalInv
	lineW := min(w-4, 70)
	if total > 0 {
		spending := m.totalExp + m.totalInv
		overspent := m.totalInc > 0 && spending > m.totalInc
		ref := math.Max(m.totalInc, spending)

		if m.totalInc > 0 {
			s.WriteString(pad + m.progIncome.ViewAs(m.totalInc/ref) + "\n")
		}
		if m.totalExp > 0 {
			s.WriteString(pad + m.progExpense.ViewAs(m.totalExp/ref) + "\n")
		}
		if m.totalInv > 0 {
			s.WriteString(pad + m.progInvest.ViewAs(m.totalInv/ref) + "\n")
		}
		s.WriteString("\n")

		// Breakdown lines

		breakdownLine := func(style lipgloss.Style, name string, amount float64, pct float64, showPct bool) string {
			left := "█  " + name
			right := fmt.Sprintf("%s %s", fmtAmount(amount), m.Config.DisplayCurrency)
			if showPct {
				right += fmt.Sprintf("  %3.0f%%", pct)
			}
			gap := lineW - len(left) - len(right)
			if gap < 2 {
				gap = 2
			}
			return style.Render(left) + styleMuted.Render(strings.Repeat("·", gap)) + style.Render(right)
		}

		if m.totalInc > 0 {
			expPct := m.totalExp / m.totalInc * 100
			invPct := m.totalInv / m.totalInc * 100
			s.WriteString(pad + breakdownLine(styleIncome, "Income", m.totalInc, 0, false) + "\n")
			s.WriteString(pad + styleMuted.Render(strings.Repeat("─", lineW)) + "\n")
			s.WriteString(pad + breakdownLine(styleExpense, "Expenses", m.totalExp, expPct, true) + "\n")
			s.WriteString(pad + breakdownLine(styleInvestment, "Invest", m.totalInv, invPct, true) + "\n")
			if !overspent {
				savedPct := saved / m.totalInc * 100
				s.WriteString(pad + breakdownLine(styleSaved, "Saved", saved, savedPct, true) + "\n")
			}
			s.WriteString("\n")
		} else {
			s.WriteString(pad + breakdownLine(styleIncome, "Income", 0, 0, false) + "\n")
			s.WriteString(pad + styleMuted.Render(strings.Repeat("─", lineW)) + "\n")
			s.WriteString(pad + breakdownLine(styleExpense, "Expenses", m.totalExp, 0, false) + "\n")
			s.WriteString(pad + breakdownLine(styleInvestment, "Invest", m.totalInv, 0, false) + "\n\n")
		}
	} else {
		s.WriteString(pad + styleDim.Render("No data this month") + "\n\n")
	}

	// ── Category breakdown ──
	if len(m.categories) > 0 {
		s.WriteString(pad + styleBold.Render("Expenses by category") + "\n")
		s.WriteString(pad + styleMuted.Render(strings.Repeat("─", lineW)) + "\n")

		selCatIdx, selSubIdx := m.cursorPos()

		for i, cat := range m.categories {
			color := catColors[i%len(catColors)]
			catStyle := lipgloss.NewStyle().Foreground(color)

			name := capitalize(truncateName(cat.Name, 13))
			left := fmt.Sprintf("█  %-13s", name)
			right := fmt.Sprintf("%8s %s  %3.0f%%", fmtAmount(cat.Total), m.Config.DisplayCurrency, cat.Percent)

			barMaxW := lineW - lipgloss.Width(left) - lipgloss.Width(right) - 2
			if barMaxW < 1 {
				barMaxW = 1
			}
			barLen := int(math.Round(cat.Percent / 100 * float64(barMaxW)))
			if barLen < 1 && cat.Total > 0 {
				barLen = 1
			}
			barSpace := barMaxW - barLen
			if barSpace < 0 {
				barSpace = 0
			}

			if i == selCatIdx && selSubIdx == -1 {
				selCat := catStyle.Background(selPurple)
				bar := selCat.Render(strings.Repeat("█", barLen)) + styleSel.Render(strings.Repeat(" ", barSpace))
				line := selCat.Render(left) + styleSel.Render(" ") + bar + styleSel.Render(" ") + selCat.Render(right)
				s.WriteString(pad + line + "\n")
			} else {
				bar := catStyle.Render(strings.Repeat("█", barLen)) + strings.Repeat(" ", barSpace)
				line := catStyle.Render(left) + " " + bar + " " + catStyle.Render(right)
				s.WriteString(pad + line + "\n")
			}

			if m.showSubs && len(cat.Subs) > 0 {
				connStyle := lipgloss.NewStyle().Foreground(color)

				for j, sub := range cat.Subs {
					connector := "├── "
					if j == len(cat.Subs)-1 {
						connector = "╰── "
					}

					subPct := 0.0
					if m.totalExp > 0 {
						subPct = sub.Total / m.totalExp * 100
					}
					subName := sub.Name
					if idx := strings.Index(subName, ":"); idx >= 0 {
						subName = subName[idx+1:]
					}
					subName = capitalize(truncateName(subName, 13))
					subRight := fmt.Sprintf("%8s %s  %3.0f%%", fmtAmount(sub.Total), m.Config.DisplayCurrency, subPct)
					gap := lineW - 6 - lipgloss.Width(subName) - lipgloss.Width(subRight)
					if gap < 2 {
						gap = 2
					}

					if i == selCatIdx && j == selSubIdx {
						selConn := connStyle.Background(selPurple)
						subLine := styleSel.Render("  ") + selConn.Render(connector) +
							lipgloss.NewStyle().Foreground(dim).Background(selPurple).Render(subName) +
							lipgloss.NewStyle().Foreground(muted).Background(selPurple).Render(strings.Repeat("·", gap)) +
							lipgloss.NewStyle().Foreground(dim).Background(selPurple).Render(subRight)
						s.WriteString(pad + subLine + "\n")
					} else {
						subLine := "  " + connStyle.Render(connector) +
							styleDim.Render(subName) +
							styleMuted.Render(strings.Repeat("·", gap)) +
							styleDim.Render(subRight)
						s.WriteString(pad + subLine + "\n")
					}
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

	// ── Footer (help) ──
	s.WriteString(m.helpView(w-4) + "\n\n")

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

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[:1])) + string(r[1:])
}

func truncateName(s string, maxW int) string {
	if lipgloss.Width(s) <= maxW {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > maxW-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
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
