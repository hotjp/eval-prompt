package domain

import (
	"fmt"
	"time"
)

// Snapshot represents a version of an Asset.
// Snapshots are created when an Asset is committed to Git.
type Snapshot struct {
	ID          ID
	AssetID     ID
	Version     string // Semantic version like v1.0.0
	ContentHash string // SHA256 hash of the content
	CommitHash  string // Git commit hash
	Author      string
	Reason      string
	Model       string
	Temperature float64
	Metrics     map[string]any
	CreatedAt   time.Time
}

// Validate validates the snapshot entity.
func (s *Snapshot) Validate() error {
	if s.ID.IsEmpty() {
		return ErrInvalidID(s.ID.String())
	}
	if s.AssetID.IsEmpty() {
		return fmt.Errorf("asset_id is required")
	}
	if s.Version == "" {
		return NewDomainError(ErrSnapshotVersionInvalid, "version is required")
	}
	if s.ContentHash == "" {
		return fmt.Errorf("content_hash is required")
	}
	return nil
}

// NewSnapshot creates a new Snapshot with the given parameters.
func NewSnapshot(assetID ID, version, contentHash, author, reason string) *Snapshot {
	return &Snapshot{
		ID:          NewAutoID(),
		AssetID:     assetID,
		Version:     version,
		ContentHash: contentHash,
		Author:      author,
		Reason:      reason,
		CreatedAt:   time.Now(),
		Metrics:     make(map[string]any),
	}
}

// NewSnapshotWithCommit creates a new Snapshot with Git commit information.
func NewSnapshotWithCommit(assetID ID, version, contentHash, commitHash, author, reason string) *Snapshot {
	return &Snapshot{
		ID:          NewAutoID(),
		AssetID:     assetID,
		Version:     version,
		ContentHash: contentHash,
		CommitHash:  commitHash,
		Author:      author,
		Reason:      reason,
		CreatedAt:   time.Now(),
		Metrics:     make(map[string]any),
	}
}

// SnapshotSummary is a lightweight representation of a snapshot.
type SnapshotSummary struct {
	ID         ID
	Version    string
	CommitHash string
	Author     string
	Reason     string
	CreatedAt  time.Time
}
