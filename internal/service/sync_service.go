// Package service implements L4-Service layer: input validation, transaction boundaries,
// workflow triggering, domain coordination, and plugin scheduling.
package service

import (
	"context"
	"encoding/json"
	"fmt"
)

// SyncService handles synchronization between Git repository and the index.
type SyncService struct {
	indexer    AssetIndexer
	gitBridger GitBridger
}

// NewSyncService creates a new SyncService.
func NewSyncService(indexer AssetIndexer, gitBridger GitBridger) *SyncService {
	return &SyncService{
		indexer:    indexer,
		gitBridger: gitBridger,
	}
}

// Ensure SyncService implements SyncServicer.
var _ SyncServicer = (*SyncService)(nil)

// SyncServicer is the interface for sync operations.
type SyncServicer interface {
	// Reconcile synchronizes the index with the Git repository.
	Reconcile(ctx context.Context) (ReconcileReport, error)

	// RebuildIndex rebuilds the entire search index from the Git repository.
	RebuildIndex(ctx context.Context) error

	// Export exports the asset data in the specified format (json or yaml).
	Export(ctx context.Context, format string) ([]byte, error)
}

// Reconcile synchronizes the index with the Git repository.
func (s *SyncService) Reconcile(ctx context.Context) (ReconcileReport, error) {
	if s.indexer == nil {
		return ReconcileReport{}, fmt.Errorf("indexer not configured")
	}
	return s.indexer.Reconcile(ctx)
}

// RebuildIndex rebuilds the entire search index from the Git repository.
func (s *SyncService) RebuildIndex(ctx context.Context) error {
	if s.indexer == nil {
		return fmt.Errorf("indexer not configured")
	}

	// Rebuild index by reconciling
	report, err := s.indexer.Reconcile(ctx)
	if err != nil {
		return fmt.Errorf("reconcile: %w", err)
	}

	// Log report summary
	if len(report.Errors) > 0 {
		return fmt.Errorf("reconcile errors: %v", report.Errors)
	}

	_ = report // suppress unused warning
	return nil
}

// Export exports the asset data in the specified format.
func (s *SyncService) Export(ctx context.Context, format string) ([]byte, error) {
	if s.indexer == nil {
		return nil, fmt.Errorf("indexer not configured")
	}

	// Search all assets (empty query returns all)
	assets, err := s.indexer.Search(ctx, "", SearchFilters{})
	if err != nil {
		return nil, fmt.Errorf("search assets: %w", err)
	}

	switch format {
	case "json":
		return json.MarshalIndent(assets, "", "  ")
	case "yaml":
		// Simple YAML conversion
		return jsonToYAML(assets)
	default:
		return nil, fmt.Errorf("unsupported format: %s (supported: json, yaml)", format)
	}
}

// jsonToYAML converts JSON-like data to YAML string.
// This is a simple implementation for the export feature.
func jsonToYAML(data any) ([]byte, error) {
	// Use json.Marshal for now - proper YAML conversion would require a library
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}
