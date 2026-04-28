// Package search provides asset indexing and search functionality.
package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/service"
	"github.com/eval-prompt/internal/yamlutil"
)

// Indexer implements service.AssetIndexer using in-memory storage.
// For production, replace with Meilisearch or other search engine.
type Indexer struct {
	mu         sync.RWMutex
	assets     map[string]*assetEntry
	summaries  []service.AssetSummary
	gitBridge  service.GitBridger
	persistDir string
}

type assetEntry struct {
	asset  service.Asset
	detail *service.AssetDetail
}

// SetGitBridge sets the Git bridger for filesystem scanning.
func (i *Indexer) SetGitBridge(bridge service.GitBridger) {
	i.gitBridge = bridge
}

// SetPersistDir sets the directory for index persistence.
// The index will be saved to {persistDir}/index.json after each Reconcile.
func (i *Indexer) SetPersistDir(dir string) {
	i.persistDir = dir
}

// persist saves the index to disk if persistDir is set.
func (i *Indexer) persist() error {
	if i.persistDir == "" {
		return nil
	}
	if err := os.MkdirAll(i.persistDir, 0755); err != nil {
		return fmt.Errorf("create persist dir: %w", err)
	}
	path := filepath.Join(i.persistDir, "index.json")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create index file: %w", err)
	}
	defer f.Close()
	// Only persist AssetDetail (which includes snapshots) for version history
	type persistEntry struct {
		ID          string                    `json:"id"`
		Name        string                    `json:"name"`
		Description string                    `json:"description"`
		AssetType     string                    `json:"asset_type"`
		Tags        []string                  `json:"tags"`
		State       string                    `json:"state"`
		Snapshots   []service.SnapshotSummary `json:"snapshots"`
		Category    string                    `json:"category"`
		EvalHistory []domain.EvalHistoryEntry `json:"eval_history"`
		EvalStats   domain.EvalStats         `json:"eval_stats"`
		Triggers    []domain.TriggerEntry    `json:"triggers"`
		TestCases   []domain.TestCase         `json:"test_cases"`
		RecommendedSnapshotID string           `json:"recommended_snapshot_id"`
		Labels      []service.LabelInfo       `json:"labels"`
		AssetPath   string                    `json:"asset_path"`
	}
	data := make([]persistEntry, 0, len(i.assets))
	for _, entry := range i.assets {
		data = append(data, persistEntry{
			ID:          entry.detail.ID,
			Name:        entry.detail.Name,
			Description: entry.detail.Description,
			AssetType:     entry.detail.AssetType,
			Tags:        entry.detail.Tags,
			State:       entry.detail.State,
			Snapshots:   entry.detail.Snapshots,
			Category:    entry.detail.Category,
			EvalHistory: entry.detail.EvalHistory,
			EvalStats:   entry.detail.EvalStats,
			Triggers:    entry.detail.Triggers,
			TestCases:   entry.detail.TestCases,
			RecommendedSnapshotID: entry.detail.RecommendedSnapshotID,
			Labels:      entry.detail.Labels,
			AssetPath:   entry.detail.AssetPath,
		})
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// Load restores the index from disk if a persistence file exists.
func (i *Indexer) Load() error {
	if i.persistDir == "" {
		return nil
	}
	path := filepath.Join(i.persistDir, "index.json")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no saved index, start fresh
		}
		return fmt.Errorf("open index file: %w", err)
	}
	defer f.Close()
	type persistEntry struct {
		ID          string                    `json:"id"`
		Name        string                    `json:"name"`
		Description string                    `json:"description"`
		AssetType     string                    `json:"asset_type"`
		Tags        []string                  `json:"tags"`
		State       string                    `json:"state"`
		Snapshots   []service.SnapshotSummary `json:"snapshots"`
		Category    string                    `json:"category"`
		EvalHistory []domain.EvalHistoryEntry `json:"eval_history"`
		EvalStats   domain.EvalStats         `json:"eval_stats"`
		Triggers    []domain.TriggerEntry    `json:"triggers"`
		TestCases   []domain.TestCase         `json:"test_cases"`
		RecommendedSnapshotID string           `json:"recommended_snapshot_id"`
		Labels      []service.LabelInfo       `json:"labels"`
		AssetPath   string                    `json:"asset_path"`
	}
	var data []persistEntry
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return fmt.Errorf("decode index: %w", err)
	}
	for _, pe := range data {
		// Sort snapshots by CreatedAt descending
		sort.Slice(pe.Snapshots, func(i, j int) bool {
			return pe.Snapshots[i].CreatedAt.After(pe.Snapshots[j].CreatedAt)
		})
		i.assets[pe.ID] = &assetEntry{
			asset: service.Asset{
				ID:          pe.ID,
				Name:        pe.Name,
				Description: pe.Description,
				AssetType:     pe.AssetType,
				Tags:        pe.Tags,
				State:       pe.State,
				AssetPath:   pe.AssetPath,
			},
			detail: &service.AssetDetail{
				ID:          pe.ID,
				Name:        pe.Name,
				Description: pe.Description,
				AssetType:     pe.AssetType,
				Tags:        pe.Tags,
				State:       pe.State,
				Snapshots:   pe.Snapshots,
				Category:    pe.Category,
				EvalHistory: pe.EvalHistory,
				EvalStats:   pe.EvalStats,
				Triggers:    pe.Triggers,
				TestCases:   pe.TestCases,
				RecommendedSnapshotID: pe.RecommendedSnapshotID,
				Labels:      pe.Labels,
				AssetPath:   pe.AssetPath,
			},
		}
	}
	return nil
}

// NewIndexer creates a new in-memory Indexer.
func NewIndexer() *Indexer {
	return &Indexer{
		assets: make(map[string]*assetEntry),
	}
}

// Default returns a package-level singleton Indexer that persists to disk.
// This allows CLI commands to share state across invocations.
func Default() *Indexer {
	defaultOnce.Do(func() {
		defaultIndexer = NewIndexer()
	})
	return defaultIndexer
}

var (
	defaultOnce     sync.Once
	defaultIndexer *Indexer
)

// Ensure Indexer implements AssetIndexer.
var _ service.AssetIndexer = (*Indexer)(nil)

// Reconcile synchronizes the index with the Git repository.
// It scans assets/{type}/*.yaml files and updates the index.
func (i *Indexer) Reconcile(ctx context.Context) (service.ReconcileReport, error) {
	return i.ReconcileAssetYAML(ctx)
}

// Search searches for assets matching the query and filters.
func (i *Indexer) Search(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var results []service.AssetSummary
	for _, entry := range i.assets {
		if matchAsset(entry.detail, query, filters) {
			// Get latest score from snapshots
			var latestScore *float64
			if len(entry.detail.Snapshots) > 0 {
				latestScore = entry.detail.Snapshots[0].EvalScore
			}
			results = append(results, service.AssetSummary{
				ID:          entry.asset.ID,
				Name:        entry.asset.Name,
				Description: entry.asset.Description,
				AssetType:   entry.asset.AssetType,
				Category:    entry.asset.Category,
				Tags:        entry.asset.Tags,
				State:       entry.asset.State,
				LatestScore: latestScore,
				Keywords:    entry.detail.Keywords,
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
			AssetType:   asset.AssetType,
			Category:    asset.Category,
			Tags:        asset.Tags,
			State:       asset.State,
			Snapshots:   []service.SnapshotSummary{},
			Labels:      []service.LabelInfo{},
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

// Ensure Indexer implements service.AssetFileManager.
var _ service.AssetFileManager = (*Indexer)(nil)

// CommitFile stages and commits an asset without modifying its content.
// For folder-based assets, commits both asset.yaml and main file.
// Returns the commit hash.
func (i *Indexer) CommitFile(ctx context.Context, id string, commitMsg string) (string, error) {
	repoPath := ""
	if i.gitBridge != nil {
		repoPath = i.gitBridge.RepoPath()
	}
	if repoPath == "" {
		return "", fmt.Errorf("repository not initialized")
	}

	// Lookup asset to find asset.yaml path
	detail, err := i.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("asset not found: %w", err)
	}

	if detail.AssetPath == "" {
		return "", fmt.Errorf("legacy .md assets are not supported")
	}

	// Read asset.yaml to resolve main path
	ay, err := i.GetAssetYAML(ctx, detail.AssetPath)
	if err != nil {
		return "", fmt.Errorf("read asset.yaml: %w", err)
	}

	mainResolved, isExternal, err := ay.ResolveMain(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve main path: %w", err)
	}

	// Collect files to commit
	filesToCommit := []string{detail.AssetPath}
	if !isExternal {
		filesToCommit = append(filesToCommit, mainResolved)
	}

	hash, err := i.gitBridge.StageAndCommitFiles(ctx, filesToCommit, commitMsg)
	if err != nil {
		return "", fmt.Errorf("git commit: %w", err)
	}

	return hash, nil
}

// CommitFiles stages and commits multiple assets in batch.
// Returns a map of asset ID to commit hash.
func (i *Indexer) CommitFiles(ctx context.Context, ids []string, commitMsg string) (map[string]string, error) {
	results := make(map[string]string)
	for _, id := range ids {
		hash, err := i.CommitFile(ctx, id, commitMsg)
		if err != nil {
			// Log error but continue with other files
			continue
		}
		results[id] = hash
	}
	return results, nil
}

// matchAsset returns true if the asset matches the query and filters.
// It searches in name, description, tags, and keywords (if available).
func matchAsset(detail *service.AssetDetail, query string, filters service.SearchFilters) bool {
	// Query match (case-insensitive substring in name, description, tags, or keywords)
	if query != "" {
		q := query
		match := false
		if containsIgnoreCase(detail.Name, q) {
			match = true
		}
		if containsIgnoreCase(detail.Description, q) {
			match = true
		}
		// Search in tags
		for _, tag := range detail.Tags {
			if containsIgnoreCase(tag, q) {
				match = true
				break
			}
		}
		// Search in keywords (LLM-generated)
		for _, kw := range detail.Keywords {
			if containsIgnoreCase(kw, q) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}

	// AssetType filter
	if filters.AssetType != "" && filters.AssetType != detail.AssetType {
		return false
	}

	// State filter
	if filters.State != "" && filters.State != detail.State {
		return false
	}

	// Category filter
	if filters.Category != "" && filters.Category != detail.Category {
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
			for _, assetTag := range detail.Tags {
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

// ReInit reinitializes the indexer with a new repository path.
// It clears the current index and sets the new gitBridge path.
func (i *Indexer) ReInit(ctx context.Context, path string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Clear the current index
	i.assets = make(map[string]*assetEntry)
	i.summaries = nil

	// Update gitBridge path if it implements path setting
	if i.gitBridge != nil {
		i.gitBridge.SetPath(path)
	}

	return nil
}

// GetMainFileContent reads the main file content from an asset.yaml.
// It resolves the main path from asset.yaml and reads the actual file.
func (i *Indexer) GetMainFileContent(ctx context.Context, assetPath string) (content string, mainPath string, isExternal bool, err error) {
	repoPath := i.gitBridge.RepoPath()
	if repoPath == "" {
		return "", "", false, fmt.Errorf("repository not initialized")
	}

	// Read asset.yaml
	fullAssetPath := filepath.Join(repoPath, assetPath)
	yamlContent, err := os.ReadFile(fullAssetPath)
	if err != nil {
		return "", "", false, fmt.Errorf("read asset.yaml %s: %w", assetPath, err)
	}

	// Parse asset.yaml
	ay, err := domain.ParseAssetYAML(string(yamlContent))
	if err != nil {
		return "", "", false, fmt.Errorf("parse asset.yaml: %w", err)
	}

	// Resolve main path
	resolvedMain, isExt, err := ay.ResolveMain(repoPath)
	if err != nil {
		return "", "", false, fmt.Errorf("resolve main path: %w", err)
	}

	// Read main file content
	if isExt {
		// External asset - read from absolute path
		data, err := os.ReadFile(resolvedMain)
		if err != nil {
			return "", resolvedMain, true, fmt.Errorf("read external file %s: %w", resolvedMain, err)
		}
		content = string(data)
	} else {
		// Local asset - read from repo relative path
		mainFullPath := filepath.Join(repoPath, resolvedMain)
		data, err := os.ReadFile(mainFullPath)
		if err != nil {
			return "", resolvedMain, false, fmt.Errorf("read main file %s: %w", resolvedMain, err)
		}
		content = string(data)
	}

	return string(content), resolvedMain, isExt, nil
}

// WriteMainFileContent writes content to the main file specified in asset.yaml.
// It updates the content_hash in asset.yaml after writing.
func (i *Indexer) WriteMainFileContent(ctx context.Context, assetPath string, content string) (newContentHash string, err error) {
	repoPath := i.gitBridge.RepoPath()
	if repoPath == "" {
		return "", fmt.Errorf("repository not initialized")
	}

	// Read asset.yaml
	fullAssetPath := filepath.Join(repoPath, assetPath)
	yamlContent, err := os.ReadFile(fullAssetPath)
	if err != nil {
		return "", fmt.Errorf("read asset.yaml: %w", err)
	}

	ay, err := domain.ParseAssetYAML(string(yamlContent))
	if err != nil {
		return "", fmt.Errorf("parse asset.yaml: %w", err)
	}

	// Check if external
	_, isExt, err := ay.ResolveMain(repoPath)
	if err != nil {
		return "", fmt.Errorf("resolve main path: %w", err)
	}
	if isExt {
		return "", fmt.Errorf("external assets are read-only")
	}

	// Compute content hash
	hashed := sha256.Sum256([]byte(content))
	newContentHash = hex.EncodeToString(hashed[:8])

	// Write main file
	mainFullPath := filepath.Join(repoPath, ay.Main)
	if err := os.WriteFile(mainFullPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write main file: %w", err)
	}

	// Update asset.yaml metadata
	ay.Metadata.UpdatedAt = time.Now()

	// Write updated asset.yaml
	updatedYaml, err := domain.SerializeAssetYAML(ay)
	if err != nil {
		return "", fmt.Errorf("serialize asset.yaml: %w", err)
	}
	if err := os.WriteFile(fullAssetPath, []byte(updatedYaml), 0644); err != nil {
		return "", fmt.Errorf("write asset.yaml: %w", err)
	}

	return newContentHash, nil
}

// GetAssetFiles returns the files and external file lists from an asset.yaml.
func (i *Indexer) GetAssetFiles(ctx context.Context, assetPath string) (files []service.FileInfo, external []service.FileInfo, err error) {
	repoPath := i.gitBridge.RepoPath()
	if repoPath == "" {
		return nil, nil, fmt.Errorf("repository not initialized")
	}

	// Read asset.yaml
	fullAssetPath := filepath.Join(repoPath, assetPath)
	yamlContent, err := os.ReadFile(fullAssetPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read asset.yaml: %w", err)
	}

	ay, err := domain.ParseAssetYAML(string(yamlContent))
	if err != nil {
		return nil, nil, fmt.Errorf("parse asset.yaml: %w", err)
	}

	// Convert files
	files = make([]service.FileInfo, len(ay.Files))
	for idx, f := range ay.Files {
		files[idx] = service.FileInfo{
			Path: f.Path,
			Role: f.Role,
		}
	}

	// Convert external
	external = make([]service.FileInfo, len(ay.External))
	for idx, e := range ay.External {
		external[idx] = service.FileInfo{
			Path: e.Path,
			Role: e.Role,
		}
	}

	return files, external, nil
}

// GetFrontmatter reads and parses the frontmatter from an asset's main file.
func (i *Indexer) GetFrontmatter(ctx context.Context, id string) (*domain.FrontMatter, error) {
	detail, err := i.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("asset not found: %s: %w", id, err)
	}
	if detail.AssetPath == "" {
		return nil, fmt.Errorf("asset path is empty for: %s", id)
	}
	content, _, _, err := i.GetMainFileContent(ctx, detail.AssetPath)
	if err != nil {
		return nil, fmt.Errorf("read main file: %w", err)
	}
	fm, _, err := yamlutil.ParseFrontMatter(content)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, nil
}

// UpdateFrontmatter reads the frontmatter, applies the updater function, and writes it back.
// Then creates a Git commit.
func (i *Indexer) UpdateFrontmatter(ctx context.Context, id string, updater func(*domain.FrontMatter) error, commitMsg string) (string, error) {
	detail, err := i.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("asset not found: %s: %w", id, err)
	}
	if detail.AssetPath == "" {
		return "", fmt.Errorf("asset path is empty for: %s", id)
	}

	// Read current content
	content, mainPath, isExt, err := i.GetMainFileContent(ctx, detail.AssetPath)
	if err != nil {
		return "", fmt.Errorf("read main file: %w", err)
	}

	// Parse frontmatter
	fm, body, err := yamlutil.ParseFrontMatter(content)
	if err != nil {
		return "", fmt.Errorf("parse frontmatter: %w", err)
	}

	// Apply updater
	if err := updater(fm); err != nil {
		return "", fmt.Errorf("updater failed: %w", err)
	}

	// Serialize frontmatter back
	yamlStr, err := yamlutil.SerializeFrontMatter(fm)
	if err != nil {
		return "", fmt.Errorf("serialize frontmatter: %w", err)
	}

	// Reconstruct file content: --- + yaml + --- + body
	newContent := fmt.Sprintf("---\n%s---\n%s", yamlStr, body)

	// Write back
	repoPath := i.gitBridge.RepoPath()
	if !isExt {
		fullPath := filepath.Join(repoPath, mainPath)
		if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
			return "", fmt.Errorf("write file: %w", err)
		}
	}

	// Git commit
	if i.gitBridge != nil && commitMsg != "" {
		commitHash, err := i.gitBridge.StageAndCommit(ctx, mainPath, commitMsg)
		if err != nil {
			return "", fmt.Errorf("git commit: %w", err)
		}
		return commitHash, nil
	}
	return "", nil
}

// UpdateFrontmatterFileOnly updates frontmatter without creating a Git commit.
func (i *Indexer) UpdateFrontmatterFileOnly(ctx context.Context, id string, updater func(*domain.FrontMatter) error) error {
	_, err := i.UpdateFrontmatter(ctx, id, updater, "")
	return err
}

// GetBody returns the body content (markdown) without frontmatter.
func (i *Indexer) GetBody(ctx context.Context, id string) (string, error) {
	detail, err := i.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("asset not found: %s: %w", id, err)
	}
	if detail.AssetPath == "" {
		return "", fmt.Errorf("asset path is empty for: %s", id)
	}
	content, _, _, err := i.GetMainFileContent(ctx, detail.AssetPath)
	if err != nil {
		return "", fmt.Errorf("read main file: %w", err)
	}
	_, body, err := yamlutil.ParseFrontMatter(content)
	if err != nil {
		// If no frontmatter, return the whole content as body
		return content, nil
	}
	return body, nil
}

// WriteFileOnly writes the body content without updating frontmatter metadata.
func (i *Indexer) WriteFileOnly(ctx context.Context, id string, updater func(*domain.FrontMatter) error, newBody string) error {
	detail, err := i.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("asset not found: %s: %w", id, err)
	}
	if detail.AssetPath == "" {
		return fmt.Errorf("asset path is empty for: %s", id)
	}

	// Read current content
	content, mainPath, isExt, err := i.GetMainFileContent(ctx, detail.AssetPath)
	if err != nil {
		return fmt.Errorf("read main file: %w", err)
	}

	// Parse frontmatter
	fm, _, err := yamlutil.ParseFrontMatter(content)
	if err != nil {
		return fmt.Errorf("parse frontmatter: %w", err)
	}

	// Apply updater to frontmatter
	if err := updater(fm); err != nil {
		return fmt.Errorf("updater failed: %w", err)
	}

	// Serialize frontmatter
	yamlStr, err := yamlutil.SerializeFrontMatter(fm)
	if err != nil {
		return fmt.Errorf("serialize frontmatter: %w", err)
	}

	// Reconstruct: --- + yaml + --- + newBody
	newContent := fmt.Sprintf("---\n%s---\n%s", yamlStr, newBody)

	// Write back (skip for external assets)
	if !isExt {
		repoPath := i.gitBridge.RepoPath()
		fullPath := filepath.Join(repoPath, mainPath)
		if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("write file: %w", err)
		}
	}
	return nil
}

// GetFileContent reads the raw file content (including frontmatter) for an asset.
func (i *Indexer) GetFileContent(ctx context.Context, id string) (string, error) {
	detail, err := i.GetByID(ctx, id)
	if err != nil {
		return "", fmt.Errorf("asset not found: %s: %w", id, err)
	}
	if detail.AssetPath == "" {
		return "", fmt.Errorf("asset path is empty for: %s", id)
	}
	content, _, _, err := i.GetMainFileContent(ctx, detail.AssetPath)
	if err != nil {
		return "", fmt.Errorf("read main file: %w", err)
	}
	return content, nil
}

// CreatePlaceholder is intentionally removed.
// Asset creation is handled by AssetHandler.CreateAsset which follows RFC_FOLDER_STRUCTURE.md:
// - Index: assets/{type}s/{id}.yaml
// - Content: {type}s/{id}/overview.md (or type-specific main file)
// The old implementation incorrectly placed both asset.yaml and main.md under assets/{type}s/{id}/.
func (i *Indexer) CreatePlaceholder(ctx context.Context, id, name, assetType string, tags []string, category string) error {
	return fmt.Errorf("CreatePlaceholder is deprecated: use CreateAsset handler instead")
}
