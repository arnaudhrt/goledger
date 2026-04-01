package db

import (
	"fmt"
	"time"
)

// EntryType represents the type of a financial entry.
type EntryType string

const (
	Expense    EntryType = "EXP"
	Income     EntryType = "INC"
	Investment EntryType = "INV"
)

// Entry represents a single financial entry.
type Entry struct {
	ID       int64
	Date     time.Time
	Type     EntryType
	Category string
	Note     string
	Amount   float64
	Currency string
}

const dateFmt = "2006-01-02"

// InsertEntry adds a new entry to the database.
func (db *DB) InsertEntry(e Entry) (int64, error) {
	res, err := db.conn.Exec(
		`INSERT INTO entries (date, type, category, note, amount, currency)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		e.Date.Format(dateFmt), string(e.Type), e.Category, e.Note, e.Amount, e.Currency,
	)
	if err != nil {
		return 0, fmt.Errorf("insert entry: %w", err)
	}
	return res.LastInsertId()
}

// InsertEntries adds multiple entries in a single transaction.
func (db *DB) InsertEntries(entries []Entry) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO entries (date, type, category, note, amount, currency)
		 VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, e := range entries {
		if _, err := stmt.Exec(
			e.Date.Format(dateFmt), string(e.Type), e.Category, e.Note, e.Amount, e.Currency,
		); err != nil {
			return fmt.Errorf("insert entry: %w", err)
		}
	}
	return tx.Commit()
}

// UpdateEntry updates an existing entry.
func (db *DB) UpdateEntry(e Entry) error {
	_, err := db.conn.Exec(
		`UPDATE entries SET date=?, type=?, category=?, note=?, amount=?, currency=?
		 WHERE id=?`,
		e.Date.Format(dateFmt), string(e.Type), e.Category, e.Note, e.Amount, e.Currency, e.ID,
	)
	return err
}

// DeleteEntry removes an entry by ID.
func (db *DB) DeleteEntry(id int64) error {
	_, err := db.conn.Exec(`DELETE FROM entries WHERE id=?`, id)
	return err
}

// EntriesByMonth returns all entries for a given year/month, ordered by date.
func (db *DB) EntriesByMonth(year int, month time.Month) ([]Entry, error) {
	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	rows, err := db.conn.Query(
		`SELECT id, date, type, category, note, amount, currency
		 FROM entries
		 WHERE date >= ? AND date < ?
		 ORDER BY date, id`,
		start.Format(dateFmt), end.Format(dateFmt),
	)
	if err != nil {
		return nil, fmt.Errorf("query entries: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var dateStr string
		if err := rows.Scan(&e.ID, &dateStr, &e.Type, &e.Category, &e.Note, &e.Amount, &e.Currency); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		e.Date, err = time.Parse(dateFmt, dateStr)
		if err != nil {
			return nil, fmt.Errorf("parse date %q: %w", dateStr, err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
