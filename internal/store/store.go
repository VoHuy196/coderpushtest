// Package store saves the catalog and update runs in SQLite.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure Go SQLite driver, no CGO needed
)

// Errors returned by the store.
var (
	ErrNotFound    = errors.New("not found")
	ErrOpenRun     = errors.New("another update is still running or interrupted; resume or abandon it first")
	ErrWrongStatus = errors.New("run is not in the expected status")
)

const schema = `
CREATE TABLE IF NOT EXISTS products (
	sku                 TEXT PRIMARY KEY,
	name                TEXT NOT NULL DEFAULT '',
	description         TEXT NOT NULL DEFAULT '',
	price_cents         INTEGER NOT NULL DEFAULT 0,
	currency            TEXT NOT NULL DEFAULT '',
	status              TEXT NOT NULL,
	supplier_updated_at TEXT NOT NULL,
	last_run_id         INTEGER NOT NULL,
	local_updated_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS update_runs (
	id                 INTEGER PRIMARY KEY AUTOINCREMENT,
	supplier_update_id TEXT NOT NULL,
	status             TEXT NOT NULL,
	next_batch         INTEGER NOT NULL DEFAULT 1,
	total_batches      INTEGER NOT NULL DEFAULT 0,
	created_count      INTEGER NOT NULL DEFAULT 0,
	updated_count      INTEGER NOT NULL DEFAULT 0,
	withdrawn_count    INTEGER NOT NULL DEFAULT 0,
	skipped_count      INTEGER NOT NULL DEFAULT 0,
	invalid_count      INTEGER NOT NULL DEFAULT 0,
	last_error         TEXT NOT NULL DEFAULT '',
	started_at         TEXT NOT NULL,
	finished_at        TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS run_changes (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	run_id     INTEGER NOT NULL REFERENCES update_runs(id),
	batch      INTEGER NOT NULL,
	sku        TEXT NOT NULL,
	action     TEXT NOT NULL,
	detail     TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_run_changes_run ON run_changes(run_id);

CREATE TABLE IF NOT EXISTS problems (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	run_id      INTEGER NOT NULL REFERENCES update_runs(id),
	batch       INTEGER NOT NULL DEFAULT 0,
	sku         TEXT NOT NULL DEFAULT '',
	kind        TEXT NOT NULL,
	message     TEXT NOT NULL,
	status      TEXT NOT NULL DEFAULT 'open',
	created_at  TEXT NOT NULL,
	resolved_at TEXT NOT NULL DEFAULT '',
	resolution  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_problems_status ON problems(status);
`

// querier is the part of *sql.DB and *sql.Tx that we use, so the same
// helper functions work inside and outside a transaction.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store is the SQLite-backed storage.
type Store struct {
	db *sql.DB
}

// Open opens (or creates) the database file and makes sure the tables exist.
func Open(path string) (*Store, error) {
	// WAL + synchronous=FULL: a committed transaction survives a crash or power loss.
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// One connection keeps things simple: SQLite allows only one writer anyway.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Tx is a database transaction. All writes for one batch go through one Tx,
// so a batch is saved completely or not at all.
type Tx struct {
	q querier
}

// InTx runs fn inside a transaction. If fn returns an error, nothing is saved.
func (s *Store) InTx(ctx context.Context, fn func(tx *Tx) error) error {
	sqlTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(&Tx{q: sqlTx}); err != nil {
		sqlTx.Rollback()
		return err
	}
	return sqlTx.Commit()
}

// Times are stored as RFC 3339 text in UTC so the database file is easy to read.

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

func parseOptionalTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t := parseTime(s)
	return &t
}
