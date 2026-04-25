// Package search provides asset indexing and search functionality.
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
			},
			detail: &service.AssetDetail{
				ID:          pe.ID,
				Name:        pe.Name,
				Description: pe.Description,
				AssetType:     pe.AssetType,
				Tags:        pe.Tags,
				State:       pe.State,
				Snapshots:   pe.Snapshots,
				Labels:      nil,
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
// It scans the filesystem for .md files and updates the index.
func (i *Indexer) Reconcile(ctx context.Context) (service.ReconcileReport, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	report := service.ReconcileReport{}
	if i.gitBridge == nil {
		return report, nil
	}

	added, modified, deleted, err := i.gitBridge.Status(ctx)
	if err != nil {
		report.Errors = append(report.Errors, err.Error())
		return report, nil
	}

	// Process deleted files
	for _, filePath := range deleted {
		if strings.HasSuffix(filePath, ".md") {
			id := extractIDFromPath(filePath)
			if id != "" {
				delete(i.assets, id)
				report.Deleted++
			}
		}
	}

	// Process added and modified files - read frontmatter from disk
	allFiles := append(added, modified...)
	for _, filePath := range allFiles {
		if strings.HasSuffix(filePath, ".md") {
			if err := i.reconcileFile(ctx, filePath, &report); err != nil {
				report.Errors = append(report.Errors, err.Error())
			}
		}
	}

	// Persist index to disk for cross-process sharing
	if err := i.persist(); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("persist index: %v", err))
	}

	return report, nil
}

// reconcileFile reads a .md file, parses frontmatter, and saves to index.
func (i *Indexer) reconcileFile(ctx context.Context, filePath string, report *service.ReconcileReport) error {
	// Resolve relative file path against repo root
	absPath := filePath
	if i.gitBridge != nil && i.gitBridge.RepoPath() != "" {
		absPath = filepath.Join(i.gitBridge.RepoPath(), filePath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read file %s: %w", filePath, err)
	}

	// Parse frontmatter
	lines := strings.Split(string(content), "\n")
	if len(lines) < 3 || lines[0] != "---" {
		return nil // No frontmatter, skip
	}

	// Find end of frontmatter
	endIdx := -1
	for idx := 1; idx < len(lines); idx++ {
		if lines[idx] == "---" {
			endIdx = idx
			break
		}
	}
	if endIdx < 0 {
		return nil
	}

	frontmatter := strings.Join(lines[1:endIdx], "\n")
	// ParseFrontMatter expects full markdown with --- delimiters
	fullContent := "---\n" + frontmatter + "\n---"
	fm, _, err := yamlutil.ParseFrontMatter(fullContent)
	if err != nil {
		return fmt.Errorf("parse frontmatter %s: %w", filePath, err)
	}

	// Check if asset already exists (update) or is new (add)
	_, existed := i.assets[fm.ID]
	repoPath := ""
	if i.gitBridge != nil {
		repoPath = i.gitBridge.RepoPath()
	}
	asset := service.Asset{
		ID:          fm.ID,
		Name:        fm.Name,
		Description: fm.Description,
		AssetType:     fm.AssetType,
		Tags:        fm.Tags,
		State:       fm.State,
		ContentHash: fm.ContentHash,
		RepoPath:    repoPath,
	}

	// Build snapshots from eval history
	snapshots := make([]service.SnapshotSummary, 0, len(fm.EvalHistory))
	for _, entry := range fm.EvalHistory {
		createdAt, _ := time.Parse("2006-01-02", entry.Date)
		score := float64(entry.Score) / 100.0 // Convert 0-100 to 0.0-1.0
		snapshots = append(snapshots, service.SnapshotSummary{
			Version:    entry.EvalCaseVersion,
			CommitHash: "",
			Author:     entry.By,
			Reason:     "",
			EvalScore:  &score,
			CreatedAt:  createdAt,
		})
	}
	// Sort by CreatedAt descending (newest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	i.assets[fm.ID] = &assetEntry{asset: asset, detail: &service.AssetDetail{
		ID:          asset.ID,
		Name:        asset.Name,
		Description: asset.Description,
		AssetType:     asset.AssetType,
		Tags:        asset.Tags,
		State:       asset.State,
		Snapshots:   snapshots,
	}}
	if existed {
		report.Updated++
	} else {
		report.Added++
	}
	return nil
}

// extractIDFromPath extracts the asset ID from a file path like "prompts/01ARZ3NDEKTSV4RRFFQ69G5FAV.md"
func extractIDFromPath(filePath string) string {
	base := filepath.Base(filePath)
	return strings.TrimSuffix(base, ".md")
}

// Search searches for assets matching the query and filters.
func (i *Indexer) Search(ctx context.Context, query string, filters service.SearchFilters) ([]service.AssetSummary, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var results []service.AssetSummary
	for _, entry := range i.assets {
		if matchAsset(entry.asset, query, filters) {
			// Get latest score from snapshots
			var latestScore *float64
			if len(entry.detail.Snapshots) > 0 {
				latestScore = entry.detail.Snapshots[0].EvalScore
			}
			results = append(results, service.AssetSummary{
				ID:          entry.asset.ID,
				Name:        entry.asset.Name,
				Description: entry.asset.Description,
				AssetType:     entry.asset.AssetType,
				Tags:        entry.asset.Tags,
				State:       entry.asset.State,
				LatestScore: latestScore,
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
			AssetType:     asset.AssetType,
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

// GetFileContent reads the raw content of a prompt file from disk.
func (i *Indexer) GetFileContent(ctx context.Context, id string) (string, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	repoPath := ""
	if i.gitBridge != nil {
		repoPath = i.gitBridge.RepoPath()
	}
	if repoPath == "" {
		return "", fmt.Errorf("repository not initialized")
	}

	filePath := filepath.Join(repoPath, "prompts", id+".md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", filePath, err)
	}
	return string(content), nil
}

// SaveFileContent writes the full file content (including frontmatter) to a prompt file and commits it to Git.
func (i *Indexer) SaveFileContent(ctx context.Context, id, fullContent, commitMessage string) (string, error) {
	repoPath := ""
	if i.gitBridge != nil {
		repoPath = i.gitBridge.RepoPath()
	}
	if repoPath == "" {
		return "", fmt.Errorf("repository not initialized")
	}

	promptsDir := filepath.Join(repoPath, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return "", fmt.Errorf("create prompts directory: %w", err)
	}

	filePath := filepath.Join(promptsDir, id+".md")
	if err := os.WriteFile(filePath, []byte(fullContent), 0644); err != nil {
		return "", fmt.Errorf("write file %s: %w", filePath, err)
	}

	// Stage and commit via GitBridge
	relativePath := filepath.Join("prompts", id+".md")
	hash, err := i.gitBridge.StageAndCommit(ctx, relativePath, commitMessage)
	if err != nil {
		return "", fmt.Errorf("git commit: %w", err)
	}

	return hash, nil
}

// CreatePlaceholder creates a draft placeholder file and commits it to Git.
func (i *Indexer) CreatePlaceholder(ctx context.Context, id, name, bizLine string, tags []string) error {
	repoPath := ""
	if i.gitBridge != nil {
		repoPath = i.gitBridge.RepoPath()
	}
	if repoPath == "" {
		return fmt.Errorf("repository not initialized")
	}

	promptsDir := filepath.Join(repoPath, "prompts")
	if err := os.MkdirAll(promptsDir, 0755); err != nil {
		return fmt.Errorf("create prompts directory: %w", err)
	}

	fm := &domain.FrontMatter{
		ID:      id,
		Name:    name,
		AssetType: bizLine,
		Tags:    tags,
		State:   "draft",
	}

	fullContent, err := yamlutil.FormatMarkdown(fm, `
# Draft

This is a placeholder. Content will be added in a future commit.
`)
	if err != nil {
		return fmt.Errorf("format placeholder: %w", err)
	}

	filePath := filepath.Join(promptsDir, id+".md")
	if err := os.WriteFile(filePath, []byte(fullContent), 0644); err != nil {
		return fmt.Errorf("write placeholder file %s: %w", filePath, err)
	}

	relativePath := filepath.Join("prompts", id+".md")
	_, err = i.gitBridge.StageAndCommit(ctx, relativePath, fmt.Sprintf("Create placeholder for %s (%s draft)", id, name))
	if err != nil {
		return fmt.Errorf("git commit placeholder: %w", err)
	}

	return nil
}

// Ensure Indexer implements service.AssetFileManager.
var _ service.AssetFileManager = (*Indexer)(nil)

// GetFrontmatter reads and parses the frontmatter from a prompt file.
func (i *Indexer) GetFrontmatter(ctx context.Context, id string) (*domain.FrontMatter, error) {
	fullContent, err := i.GetFileContent(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get file content: %w", err)
	}

	fm, _, err := yamlutil.ParseFrontMatter(fullContent)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, nil
}

// UpdateFrontmatter reads the existing file, applies the updater to frontmatter,
// writes back and commits to Git. Returns the commit hash. Body is preserved.
func (i *Indexer) UpdateFrontmatter(ctx context.Context, id string, updater func(*domain.FrontMatter) error, commitMsg string) (string, error) {
	fullContent, err := i.GetFileContent(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get file content: %w", err)
	}

	fm, body, err := yamlutil.ParseFrontMatter(fullContent)
	if err != nil {
		return "", fmt.Errorf("parse frontmatter: %w", err)
	}

	if err := updater(fm); err != nil {
		return "", fmt.Errorf("updater rejected: %w", err)
	}

	newFullContent, err := yamlutil.FormatMarkdown(fm, body)
	if err != nil {
		return "", fmt.Errorf("format markdown: %w", err)
	}

	hash, err := i.SaveFileContent(ctx, id, newFullContent, commitMsg)
	if err != nil {
		return "", fmt.Errorf("save file content: %w", err)
	}

	return hash, nil
}

// WriteContent reads the existing file, applies the updater to frontmatter,
// replaces the body with newBody, then writes back and commits to Git.
// Returns the commit hash.
func (i *Indexer) WriteContent(ctx context.Context, id string, updater func(*domain.FrontMatter) error, newBody string, commitMsg string) (string, error) {
	fullContent, err := i.GetFileContent(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get file content: %w", err)
	}

	fm, _, err := yamlutil.ParseFrontMatter(fullContent)
	if err != nil {
		return "", fmt.Errorf("parse frontmatter: %w", err)
	}

	if err := updater(fm); err != nil {
		return "", fmt.Errorf("updater rejected: %w", err)
	}

	newFullContent, err := yamlutil.FormatMarkdown(fm, newBody)
	if err != nil {
		return "", fmt.Errorf("format markdown: %w", err)
	}

	hash, err := i.SaveFileContent(ctx, id, newFullContent, commitMsg)
	if err != nil {
		return "", fmt.Errorf("save file content: %w", err)
	}

	return hash, nil
}

// GetBody reads the file, strips frontmatter, returns only the body.
func (i *Indexer) GetBody(ctx context.Context, id string) (string, error) {
	fullContent, err := i.GetFileContent(ctx, id)
	if err != nil {
		return "", fmt.Errorf("get file content: %w", err)
	}

	lines := strings.Split(fullContent, "\n")
	frontmatterEnd := -1
	inFrontmatter := false
	for idx, line := range lines {
		if idx == 0 && strings.HasPrefix(line, "---") {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.HasPrefix(line, "---") {
			frontmatterEnd = idx
			break
		}
	}

	if frontmatterEnd >= 0 {
		return strings.TrimSpace(strings.Join(lines[frontmatterEnd+1:], "\n")), nil
	}
	return fullContent, nil
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

	// AssetType filter
	if filters.AssetType != "" && filters.AssetType != asset.AssetType {
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
