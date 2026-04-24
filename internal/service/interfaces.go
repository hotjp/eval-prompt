// Package service implements L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"time"
)

// CommitInfo represents git commit information.
type CommitInfo struct {
	Hash      string
	ShortHash string
	Subject   string
	Body      string
	Author    string
	Timestamp time.Time
}

// GitBridger is the interface for Git operations on prompt assets.
// Implemented by plugins/gitbridge.
type GitBridger interface {
	// InitRepo initializes a new Git repository at the given path.
	InitRepo(ctx context.Context, path string) error

	// StageAndCommit stages the file at filePath and creates a commit with the given message.
	// Returns the commit hash.
	StageAndCommit(ctx context.Context, filePath, message string) (string, error)

	// Diff returns the diff output between two commits (commit1 and commit2).
	Diff(ctx context.Context, commit1, commit2 string) (string, error)

	// Log returns the commit log for a file, limited to the specified number of entries.
	Log(ctx context.Context, filePath string, limit int) ([]CommitInfo, error)

	// Status returns the current working tree status: added, modified, and deleted files.
	Status(ctx context.Context) (added, modified, deleted []string, err error)
}

// AssetIndexer is the interface for indexing and searching prompt assets.
// Implemented by plugins/search.
type AssetIndexer interface {
	// Reconcile synchronizes the index with the Git repository.
	Reconcile(ctx context.Context) (ReconcileReport, error)

	// Search searches for assets matching the query and filters.
	Search(ctx context.Context, query string, filters SearchFilters) ([]AssetSummary, error)

	// GetByID retrieves an asset by its ID.
	GetByID(ctx context.Context, id string) (*AssetDetail, error)

	// Save saves an asset to the index.
	Save(ctx context.Context, asset Asset) error

	// Delete removes an asset from the index.
	Delete(ctx context.Context, id string) error
}

// SearchFilters contains filter criteria for asset search.
type SearchFilters struct {
	BizLine string
	Tags    []string
	State   string
	Label   string
}

// AssetSummary is a condensed asset representation for search results.
type AssetSummary struct {
	ID          string
	Name        string
	Description string
	BizLine     string
	Tags        []string
	State       string
	LatestScore *float64
}

// AssetDetail is a full asset representation.
type AssetDetail struct {
	ID          string
	Name        string
	Description string
	BizLine     string
	Tags        []string
	State       string
	Snapshots   []SnapshotSummary
	Labels      []LabelInfo
}

// SnapshotSummary is a condensed snapshot representation.
type SnapshotSummary struct {
	Version    string
	CommitHash string
	Author     string
	Reason     string
	EvalScore  *float64
	CreatedAt  time.Time
}

// LabelInfo represents a label on an asset.
type LabelInfo struct {
	Name       string
	SnapshotID string
	UpdatedAt  time.Time
}

// ReconcileReport contains the results of a reconciliation.
type ReconcileReport struct {
	Added   int
	Updated int
	Deleted int
	Errors  []string
}

// Asset represents a prompt asset (mirrors domain.Asset for plugin use).
type Asset struct {
	ID          string
	Name        string
	Description string
	BizLine     string
	Tags        []string
	ContentHash string
	FilePath    string
	State       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// LLMResponse contains the LLM output and metadata.
type LLMResponse struct {
	Content     string
	Model       string
	TokensIn    int
	TokensOut   int
	StopReason  string
	RawResponse []byte
}

// LLMInvoker abstracts LLM provider calls.
// Implemented by plugins/llm.
type LLMInvoker interface {
	// Invoke calls the LLM with a prompt, model, and temperature.
	// Returns the text response and usage metadata.
	Invoke(ctx context.Context, prompt string, model string, temperature float64) (*LLMResponse, error)

	// InvokeWithSchema calls the LLM and enforces a JSON output schema.
	// The schema is a JSON Schema (draft-07) describing the expected output structure.
	// Returns the parsed JSON response.
	InvokeWithSchema(ctx context.Context, prompt string, schema []byte) ([]byte, error)
}
