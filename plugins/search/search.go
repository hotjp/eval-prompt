// Package search provides asset indexing and search functionality.
package search

import (
	"context"
	"fmt"
	"sync"

	"github.com/eval-prompt/internal/service"
)

// Indexer implements service.AssetIndexer using in-memory storage.
// For production, replace with Meilisearch or other search engine.
type Indexer struct {
	mu      sync.RWMutex
	assets  map[string]*assetEntry
	summaries []service.AssetSummary
}

type assetEntry struct {
	asset   service.Asset
	detail  *service.AssetDetail
}

// NewIndexer creates a new in-memory Indexer.
func NewIndexer() *Indexer {
	return &Indexer{
		assets: make(map[string]*assetEntry),
	}
}

// Ensure Indexer implements AssetIndexer.
var _ service.AssetIndexer = (*Indexer)(nil)

// Reconcile synchronizes the index with the Git repository.
// For in-memory implementation, this is a no-op.
func (i *Indexer) Reconcile(ctx context.Context) (service.ReconcileReport, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// In-memory index is always in sync - no-op
	return service.ReconcileReport{
		Added:   0,
		Updated: 0,
		Deleted: 0,
		Errors:  nil,
	}, nil
}

// Search searches for assets matching the query and filters.
func (i *Indexer) Search(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var results []service.AssetSummary
	for _, entry := range i.assets {
		if matchAsset(entry.asset, query, filters) {
			results = append(results, service.AssetSummary{
				ID:          entry.asset.ID,
				Name:        entry.asset.Name,
				Description: entry.asset.Description,
				BizLine:     entry.asset.BizLine,
				Tags:        entry.asset.Tags,
				State:       entry.asset.State,
				LatestScore: nil,
			})
		}
	}
	return results, nil
}

// GetByID retrieves an asset by its ID.
func (i *Indexer) GetByID(ctx context.Context, id string) (*service.AssetDetail, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	entry, ok := i.assets[id]
	if !ok {
		return nil, fmt.Errorf("asset not found: %s", id)
	}
	return entry.detail, nil
}

// Save saves an asset to the index.
func (i *Indexer) Save(ctx context.Context, asset service.Asset) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	entry := &assetEntry{
		asset: asset,
		detail: &service.AssetDetail{
			ID:          asset.ID,
			Name:        asset.Name,
			Description: asset.Description,
			BizLine:     asset.BizLine,
			Tags:        asset.Tags,
			State:       asset.State,
			Snapshots:   nil,
			Labels:      nil,
		},
	}
	i.assets[asset.ID] = entry
	return nil
}

// Delete removes an asset from the index.
func (i *Indexer) Delete(ctx context.Context, id string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	delete(i.assets, id)
	return nil
}

// matchAsset returns true if the asset matches the query and filters.
func matchAsset(asset service.Asset, query string, filters service.SearchFilters) bool {
	// Query match (case-insensitive substring in name or description)
	if query != "" {
		q := query
		match := false
		if containsIgnoreCase(asset.Name, q) {
			match = true
		}
		if containsIgnoreCase(asset.Description, q) {
			match = true
		}
		if !match {
			return false
		}
	}

	// BizLine filter
	if filters.BizLine != "" && filters.BizLine != asset.BizLine {
		return false
	}

	// State filter
	if filters.State != "" && filters.State != asset.State {
		return false
	}

	// Label filter (not applicable to Asset)
	if filters.Label != "" {
		// Skip label filter for in-memory implementation
	}

	// Tags filter
	if len(filters.Tags) > 0 {
		hasTag := false
		for _, filterTag := range filters.Tags {
			for _, assetTag := range asset.Tags {
				if filterTag == assetTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	return true
}

// containsIgnoreCase returns true if s contains substr (case-insensitive).
func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	// Simple case-insensitive contains
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalIgnoreCase(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

// equalIgnoreCase returns true if a == b (case-insensitive).
func equalIgnoreCase(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
