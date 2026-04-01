package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/arnaudhrt/goledger/internal/db"
)

// LineStatus indicates the parse result status.
type LineStatus int

const (
	StatusOK           LineStatus = iota // fully parsed with category
	StatusNeedCategory                   // parsed but missing category
	StatusError                          // parse failed
)

// ParseResult holds the result of parsing a single line.
type ParseResult struct {
	Entry   db.Entry
	Status  LineStatus
	Error   string
	RawLine string
}

// ParseBulk parses multiple lines of text, skipping blank lines.
func ParseBulk(text, defaultCurrency string, defaultYear int) []ParseResult {
	lines := strings.Split(text, "\n")
	var results []ParseResult
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		results = append(results, ParseLine(line, defaultCurrency, defaultYear))
	}
	return results
}

// ParseLine parses a single entry line.
// Format: DD/MM[/YYYY] TYPE description amount [currency] [category]
func ParseLine(line, defaultCurrency string, defaultYear int) ParseResult {
	line = strings.TrimSpace(line)
	if line == "" {
		return ParseResult{RawLine: line, Status: StatusError, Error: "empty line"}
	}

	tokens := strings.Fields(line)
	if len(tokens) < 4 {
		return ParseResult{RawLine: line, Status: StatusError, Error: "need at least: date type description amount"}
	}

	date, err := parseDate(tokens[0], defaultYear)
	if err != nil {
		return ParseResult{RawLine: line, Status: StatusError, Error: err.Error()}
	}

	entryType, err := parseType(tokens[1])
	if err != nil {
		return ParseResult{RawLine: line, Status: StatusError, Error: err.Error()}
	}

	// Remaining tokens after date and type
	rest := tokens[2:]

	// Find amount: scan from right for the rightmost numeric token.
	// This handles descriptions containing numbers like "7-11 Snack".
	amountIdx := -1
	for i := len(rest) - 1; i >= 0; i-- {
		if isAmount(rest[i]) {
			amountIdx = i
			break
		}
	}
	if amountIdx < 0 {
		return ParseResult{RawLine: line, Status: StatusError, Error: "no amount found"}
	}
	if amountIdx == 0 {
		return ParseResult{RawLine: line, Status: StatusError, Error: "no description found"}
	}

	amount := parseAmount(rest[amountIdx])
	description := strings.Join(rest[:amountIdx], " ")

	// Parse optional currency and category after the amount
	var currency, category string
	after := rest[amountIdx+1:]

	switch len(after) {
	case 0:
		// nothing after amount
	case 1:
		if isCurrency(after[0]) {
			currency = strings.ToUpper(after[0])
		} else {
			category = after[0]
		}
	default:
		if isCurrency(after[0]) {
			currency = strings.ToUpper(after[0])
			category = strings.Join(after[1:], " ")
		} else {
			category = strings.Join(after, " ")
		}
	}

	if currency == "" {
		currency = defaultCurrency
	}

	status := StatusOK
	if category == "" {
		status = StatusNeedCategory
	}

	return ParseResult{
		Entry: db.Entry{
			Date:     date,
			Type:     entryType,
			Note:     description,
			Amount:   amount,
			Currency: currency,
			Category: category,
		},
		Status:  status,
		RawLine: line,
	}
}

func parseDate(s string, defaultYear int) (time.Time, error) {
	parts := strings.Split(s, "/")
	switch len(parts) {
	case 2:
		// DD/MM → use defaultYear
		day, err := strconv.Atoi(parts[0])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid day: %s", parts[0])
		}
		month, err := strconv.Atoi(parts[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid month: %s", parts[1])
		}
		return time.Date(defaultYear, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
	case 3:
		// DD/MM/YYYY
		day, err := strconv.Atoi(parts[0])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid day: %s", parts[0])
		}
		month, err := strconv.Atoi(parts[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid month: %s", parts[1])
		}
		year, err := strconv.Atoi(parts[2])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid year: %s", parts[2])
		}
		return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
	default:
		return time.Time{}, fmt.Errorf("invalid date: %s (use DD/MM or DD/MM/YYYY)", s)
	}
}

func parseType(s string) (db.EntryType, error) {
	switch strings.ToUpper(s) {
	case "EXP":
		return db.Expense, nil
	case "INC":
		return db.Income, nil
	case "INV":
		return db.Investment, nil
	default:
		return "", fmt.Errorf("unknown type: %s (use EXP, INC, or INV)", s)
	}
}

// isAmount returns true if the token looks like a number (digits, commas, optional dot).
func isAmount(s string) bool {
	if s == "" || s[0] < '0' || s[0] > '9' {
		return false
	}
	hasDot := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			continue
		case c == ',':
			continue
		case c == '.' && !hasDot && i > 0:
			hasDot = true
		default:
			return false
		}
	}
	return true
}

func parseAmount(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// isCurrency returns true if the token is exactly 3 uppercase letters.
func isCurrency(s string) bool {
	if len(s) != 3 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	return true
}
