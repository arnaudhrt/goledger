package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite connection.
type DB struct {
	conn *sql.DB
}

// Open creates the data directory if needed and opens the SQLite database.
func Open() (*DB, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dbPath := filepath.Join(dir, "spend.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Enable WAL mode for better concurrent access.
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the underlying sql.DB for direct queries if needed.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

func (db *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS entries (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		date     TEXT NOT NULL,
		type     TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT '',
		note     TEXT NOT NULL DEFAULT '',
		amount   REAL NOT NULL,
		currency TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_entries_date     ON entries(date);
	CREATE INDEX IF NOT EXISTS idx_entries_type     ON entries(type);
	CREATE INDEX IF NOT EXISTS idx_entries_category ON entries(category);

	CREATE TABLE IF NOT EXISTS rates (
		base       TEXT NOT NULL,
		target     TEXT NOT NULL,
		rate       REAL NOT NULL,
		fetched_at TEXT NOT NULL,
		PRIMARY KEY (base, target)
	);
	`
	_, err := db.conn.Exec(schema)
	return err
}

func dataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".spend"), nil
}
