package db

import (
	"fmt"
	"time"
)

// Rate represents an exchange rate between two currencies.
type Rate struct {
	Base      string
	Target    string
	Rate      float64
	FetchedAt time.Time
}

// UpsertRate inserts or updates an exchange rate.
func (db *DB) UpsertRate(r Rate) error {
	_, err := db.conn.Exec(
		`INSERT INTO rates (base, target, rate, fetched_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(base, target) DO UPDATE SET rate=excluded.rate, fetched_at=excluded.fetched_at`,
		r.Base, r.Target, r.Rate, r.FetchedAt.Format(time.RFC3339),
	)
	return err
}

// GetRate returns the cached rate for a currency pair, or false if not found.
func (db *DB) GetRate(base, target string) (Rate, bool, error) {
	var r Rate
	var fetchedStr string
	err := db.conn.QueryRow(
		`SELECT base, target, rate, fetched_at FROM rates WHERE base=? AND target=?`,
		base, target,
	).Scan(&r.Base, &r.Target, &r.Rate, &fetchedStr)

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return Rate{}, false, nil
		}
		return Rate{}, false, fmt.Errorf("get rate: %w", err)
	}

	r.FetchedAt, err = time.Parse(time.RFC3339, fetchedStr)
	if err != nil {
		return Rate{}, false, fmt.Errorf("parse fetched_at: %w", err)
	}
	return r, true, nil
}

// AllRates returns all cached rates.
func (db *DB) AllRates() ([]Rate, error) {
	rows, err := db.conn.Query(`SELECT base, target, rate, fetched_at FROM rates`)
	if err != nil {
		return nil, fmt.Errorf("query rates: %w", err)
	}
	defer rows.Close()

	var rates []Rate
	for rows.Next() {
		var r Rate
		var fetchedStr string
		if err := rows.Scan(&r.Base, &r.Target, &r.Rate, &fetchedStr); err != nil {
			return nil, fmt.Errorf("scan rate: %w", err)
		}
		r.FetchedAt, err = time.Parse(time.RFC3339, fetchedStr)
		if err != nil {
			return nil, fmt.Errorf("parse fetched_at: %w", err)
		}
		rates = append(rates, r)
	}
	return rates, rows.Err()
}
