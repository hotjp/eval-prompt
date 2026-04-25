// Package handlers contains HTTP handlers for the gateway layer.
package handlers

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteChecker checks if a SQLite database file is accessible.
type SQLiteChecker struct {
	dsn string
}

// NewSQLiteChecker creates a new SQLiteChecker.
func NewSQLiteChecker(dsn string) *SQLiteChecker {
	return &SQLiteChecker{dsn: dsn}
}

// Ping attempts to open a connection and execute a simple query.
func (c *SQLiteChecker) Ping(ctx context.Context) error {
	if c.dsn == "" {
		return errors.New("sqlite: no DSN configured")
	}

	// Try to open a connection and ping
	db, err := sql.Open("sqlite3", c.dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	return db.PingContext(ctx)
}
