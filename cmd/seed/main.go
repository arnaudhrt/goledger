package main

import (
	"fmt"
	"os"
	"time"

	"github.com/arnaudhrt/goledger/internal/db"
)

func main() {
	database, err := db.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	now := time.Now()
	y, m := now.Year(), now.Month()
	d := func(day int) time.Time {
		return time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	}

	entries := []db.Entry{
		// Income
		{Date: d(1), Type: db.Income, Note: "Salary", Amount: 55000, Currency: "THB", Category: "salary"},
		{Date: d(15), Type: db.Income, Note: "Freelance project", Amount: 12000, Currency: "THB", Category: "freelance"},

		// Investments
		{Date: d(1), Type: db.Investment, Note: "Monthly BTC", Amount: 10000, Currency: "THB", Category: "crypto"},
		{Date: d(1), Type: db.Investment, Note: "VWRA ETF", Amount: 5000, Currency: "THB", Category: "stocks"},

		// Expenses - housing
		{Date: d(1), Type: db.Expense, Note: "Rent", Amount: 15000, Currency: "THB", Category: "housing:rent"},
		{Date: d(5), Type: db.Expense, Note: "Electric bill", Amount: 1200, Currency: "THB", Category: "housing:electric"},
		{Date: d(5), Type: db.Expense, Note: "Water bill", Amount: 180, Currency: "THB", Category: "housing:water"},

		// Expenses - food
		{Date: d(2), Type: db.Expense, Note: "Grab Food dinner", Amount: 250, Currency: "THB", Category: "food:dining"},
		{Date: d(4), Type: db.Expense, Note: "Tops grocery", Amount: 1800, Currency: "THB", Category: "food:grocery"},
		{Date: d(7), Type: db.Expense, Note: "Starbucks", Amount: 165, Currency: "THB", Category: "food:coffee"},
		{Date: d(12), Type: db.Expense, Note: "Sushi lunch", Amount: 450, Currency: "THB", Category: "food:dining"},
		{Date: d(18), Type: db.Expense, Note: "7-11 snacks", Amount: 85, Currency: "THB", Category: "food:grocery"},

		// Expenses - transport
		{Date: d(3), Type: db.Expense, Note: "Grab to office", Amount: 120, Currency: "THB", Category: "transport:grab"},
		{Date: d(10), Type: db.Expense, Note: "PTT fuel", Amount: 1500, Currency: "THB", Category: "transport:fuel"},

		// Expenses - fun
		{Date: d(8), Type: db.Expense, Note: "Netflix sub", Amount: 419, Currency: "THB", Category: "fun:sub"},
		{Date: d(14), Type: db.Expense, Note: "Drinks with friends", Amount: 650, Currency: "THB", Category: "fun:social"},

		// Expenses - bills
		{Date: d(5), Type: db.Expense, Note: "TRUE Internet", Amount: 599, Currency: "THB", Category: "bills:internet"},
		{Date: d(5), Type: db.Expense, Note: "AIS mobile", Amount: 399, Currency: "THB", Category: "bills:mobile"},
		{Date: d(20), Type: db.Expense, Note: "Annual health insurance premium payment", Amount: 8500, Currency: "THB", Category: "bills:insurance"},
	}

	if err := database.InsertEntries(entries); err != nil {
		fmt.Fprintf(os.Stderr, "Error inserting: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Seeded %d entries for %s %d\n", len(entries), m.String(), y)
}
