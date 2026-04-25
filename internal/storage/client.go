package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/storage/ent"
)

// Client wraps the ent client with additional functionality.
type Client struct {
	ent *ent.Client
	db  *sql.DB
}

// NewClient creates a new storage client with SQLite.
func NewClient(cfg config.DatabaseConfig) (*Client, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = "eval-prompt.db"
	}

	// Append SQLite-specific options for WAL mode and foreign keys
	dsnFull := fmt.Sprintf("%s?_fk=1&_journal_mode=WAL", dsn)

	// Create ent client using SQLite driver
	entClient, err := ent.Open("sqlite3", dsnFull)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Open a separate sql.DB for raw SQL access (shares the same SQLite connection via WAL mode)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		entClient.Close()
		return nil, fmt.Errorf("failed to open raw sqlite: %w", err)
	}

	// Auto-migrate schema
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := entClient.Schema.Create(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &Client{ent: entClient, db: db}, nil
}

// NewClientForTest creates a new storage client wrapping an existing ent.Client.
// This is useful for testing scenarios where the ent.Client is created
// via enttest.Open().
func NewClientForTest(entClient *ent.Client) *Client {
	return &Client{ent: entClient}
}

// NewClientWithDSN creates a new storage client with a specific DSN.
func NewClientWithDSN(dsn string) (*Client, error) {
	// Use default database if DSN is empty
	if dsn == "" {
		dsn = "eval.db"
	}
	// Append SQLite-specific options for WAL mode and foreign keys
	dsnFull := fmt.Sprintf("%s?_fk=1&_journal_mode=WAL", dsn)

	// Create ent client using SQLite driver
	entClient, err := ent.Open("sqlite3", dsnFull)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Open a separate sql.DB for raw SQL access
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		entClient.Close()
		return nil, fmt.Errorf("failed to open raw sqlite: %w", err)
	}

	// Auto-migrate schema
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := entClient.Schema.Create(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &Client{ent: entClient, db: db}, nil
}

// Ent returns the underlying ent client.
func (c *Client) Ent() *ent.Client {
	return c.ent
}

// DB returns the underlying database/sql.DB for raw SQL access.
func (c *Client) DB() *sql.DB {
	return c.db
}

// Close closes the database connections.
func (c *Client) Close() error {
	if c.db != nil {
		c.db.Close()
	}
	return c.ent.Close()
}

// Tx starts a new transaction.
func (c *Client) Tx(ctx context.Context) (*ent.Tx, error) {
	return c.ent.Tx(ctx)
}

// AssetClient returns the Asset client.
func (c *Client) AssetClient() *ent.AssetClient {
	return c.ent.Asset
}

// LabelClient returns the Label client.
func (c *Client) LabelClient() *ent.LabelClient {
	return c.ent.Label
}

// EvalCaseClient returns the EvalCase client.
func (c *Client) EvalCaseClient() *ent.EvalCaseClient {
	return c.ent.EvalCase
}

// EvalRunClient returns the EvalRun client.
func (c *Client) EvalRunClient() *ent.EvalRunClient {
	return c.ent.EvalRun
}

// OutboxEventClient returns the OutboxEvent client.
func (c *Client) OutboxEventClient() *ent.OutboxEventClient {
	return c.ent.OutboxEvent
}

// AuditLogClient returns the AuditLog client.
func (c *Client) AuditLogClient() *ent.AuditLogClient {
	return c.ent.AuditLog
}

// ModelAdaptationClient returns the ModelAdaptation client.
func (c *Client) ModelAdaptationClient() *ent.ModelAdaptationClient {
	return c.ent.ModelAdaptation
}

// EvalExecutionClient returns the EvalExecution client.
func (c *Client) EvalExecutionClient() *ent.EvalExecutionClient {
	return c.ent.EvalExecution
}

// EvalWorkItemClient returns the EvalWorkItem client.
func (c *Client) EvalWorkItemClient() *ent.EvalWorkItemClient {
	return c.ent.EvalWorkItem
}
