package storage

import (
	"context"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/eval-prompt/internal/config"
	"github.com/eval-prompt/internal/storage/ent"
)

// Client wraps the ent client with additional functionality.
type Client struct {
	ent *ent.Client
}

// NewClient creates a new storage client with SQLite.
func NewClient(cfg config.DatabaseConfig) (*Client, error) {
	dsn := cfg.DSN
	if dsn == "" {
		dsn = "eval-prompt.db"
	}

	// Append SQLite-specific options for WAL mode and foreign keys
	dsn = fmt.Sprintf("%s?_fk=1&_journal_mode=WAL", dsn)

	// Create ent client using SQLite driver
	client, err := ent.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Auto-migrate schema
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &Client{ent: client}, nil
}

// NewClientForTest creates a new storage client wrapping an existing ent.Client.
// This is useful for testing scenarios where the ent.Client is created
// via enttest.Open().
func NewClientForTest(entClient *ent.Client) *Client {
	return &Client{ent: entClient}
}

// NewClientWithDSN creates a new storage client with a specific DSN.
func NewClientWithDSN(dsn string) (*Client, error) {
	// Append SQLite-specific options for WAL mode and foreign keys
	dsn = fmt.Sprintf("%s?_fk=1&_journal_mode=WAL", dsn)

	// Create ent client using SQLite driver
	client, err := ent.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	// Auto-migrate schema
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Schema.Create(ctx); err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &Client{ent: client}, nil
}

// Ent returns the underlying ent client.
func (c *Client) Ent() *ent.Client {
	return c.ent
}

// Close closes the database connection.
func (c *Client) Close() error {
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

// SnapshotClient returns the Snapshot client.
func (c *Client) SnapshotClient() *ent.SnapshotClient {
	return c.ent.Snapshot
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
