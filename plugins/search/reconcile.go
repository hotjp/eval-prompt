package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eval-prompt/internal/domain"
	"github.com/eval-prompt/internal/service"
)

// Asset types that are scanned during reconciliation.
var assetTypes = []string{"prompt", "skill", "agent", "mcp", "workflow", "knowledge"}

// ReconcileAssetYAML synchronizes the index with asset.yaml files.
// It scans assets/{type}/*.yaml files and updates the index accordingly.
func (i *Indexer) ReconcileAssetYAML(ctx context.Context) (service.ReconcileReport, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	report := service.ReconcileReport{}
	if i.gitBridge == nil {
		return report, nil
	}

	repoPath := i.gitBridge.RepoPath()
	if repoPath == "" {
		return report, nil
	}

	// Track which asset IDs are still valid
	validAssetIDs := make(map[string]bool)

	// Scan each asset type directory
	for _, assetType := range assetTypes {
		// Use plural form for directory names (skills/, agents/, prompts/, etc.)
		assetsDir := filepath.Join(repoPath, "assets", assetType+"s")
		if err := i.scanAssetsTypeDir(ctx, assetsDir, assetType, repoPath, &report, validAssetIDs); err != nil {
			report.Errors = append(report.Errors, err.Error())
		}
	}

	// Remove assets that are no longer in the filesystem
	for id := range i.assets {
		if !validAssetIDs[id] {
			delete(i.assets, id)
			report.Deleted++
		}
	}

	// Persist index to disk for cross-process sharing
	if err := i.persist(); err != nil {
		report.Errors = append(report.Errors, fmt.Sprintf("persist index: %v", err))
	}

	return report, nil
}

// scanAssetsTypeDir scans a single asset type directory (e.g., assets/skills/).
func (i *Indexer) scanAssetsTypeDir(ctx context.Context, assetsDir, assetType, repoPath string, report *service.ReconcileReport, validAssetIDs map[string]bool) error {
	// Check if directory exists
	info, err := os.Stat(assetsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, skip
		}
		return fmt.Errorf("stat assets directory: %w", err)
	}
	if !info.IsDir() {
		return nil // Not a directory, skip
	}

	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		return fmt.Errorf("read assets directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue // Skip directories and non-yaml files
		}

		assetID := strings.TrimSuffix(entry.Name(), ".yaml")
		// Use plural form for asset type directory (assets/skills/, not assets/skill/)
		assetPath := filepath.Join("assets", assetType+"s", entry.Name())

		if err := i.reconcileAssetYAML(ctx, assetPath, assetID, assetType, repoPath, report); err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("reconcile %s: %v", assetID, err))
			continue
		}

		validAssetIDs[assetID] = true
	}

	return nil
}

// reconcileAssetYAML reads an asset.yaml file, parses it, and updates the index.
func (i *Indexer) reconcileAssetYAML(ctx context.Context, assetPath, assetID, assetType, repoPath string, report *service.ReconcileReport) error {
	if repoPath == "" {
		return fmt.Errorf("repo path is empty")
	}
	absPath := filepath.Join(repoPath, assetPath)

	// Read asset.yaml
	content, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("read asset.yaml: %w", err)
	}

	// Parse asset.yaml
	ay, err := domain.ParseAssetYAML(string(content))
	if err != nil {
		return fmt.Errorf("parse asset.yaml: %w", err)
	}

	// Resolve main path
	mainResolved, isExternal, err := ay.ResolveMain(repoPath)
	if err != nil {
		return fmt.Errorf("resolve main path: %w", err)
	}

	// Check if main file exists (for non-external assets)
	state := ay.State
	if !isExternal && state != "deleted" && state != "unavailable" {
		mainAbsPath := filepath.Join(repoPath, mainResolved)
		if _, err := os.Stat(mainAbsPath); os.IsNotExist(err) {
			state = "unavailable"
		}
	}

	// Build file tree (for future use, currently not stored)
	_ = i.buildFileTree(ay, repoPath)

	// Create asset entry
	asset := service.Asset{
		ID:          assetID,
		Name:        ay.Name,
		Description: ay.Description,
		AssetType:   ay.AssetType,
		Category:    ay.Category,
		Tags:        ay.Tags,
		State:       state,
		RepoPath:    repoPath,
		FilePath:    mainResolved,
	}

	// Check if asset already exists (update) or is new (add)
	_, existed := i.assets[assetID]

	detail := &service.AssetDetail{
		ID:          assetID,
		Name:        ay.Name,
		Description: ay.Description,
		AssetType:   ay.AssetType,
		Category:    ay.Category,
		Tags:        ay.Tags,
		State:       state,
	}

	i.assets[assetID] = &assetEntry{asset: asset, detail: detail}

	if existed {
		report.Updated++
	} else {
		report.Added++
	}

	return nil
}

// buildFileTree builds the file tree from AssetYAML.
// It includes all files in the files list and external list.
func (i *Indexer) buildFileTree(ay *domain.AssetYAML, repoPath string) []string {
	var tree []string

	// Add main file
	if ay.Main != "" {
		tree = append(tree, ay.Main)
	}

	// Add files from files list
	for _, f := range ay.Files {
		if !containsString(tree, f.Path) {
			tree = append(tree, f.Path)
		}
	}

	// Add files from external list
	for _, e := range ay.External {
		if !containsString(tree, e.Path) {
			tree = append(tree, e.Path)
		}
	}

	return tree
}

// containsString checks if a string is in a slice.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// CheckConsistency checks the consistency between filesystem, YAML, and SQLite.
// Returns a report of inconsistencies found.
func (i *Indexer) CheckConsistency(ctx context.Context) (*ConsistencyReport, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	report := &ConsistencyReport{
		OrphanFolders: []string{},
		MissingFiles:  []string{},
		OrphanYAMLs:   []string{},
	}

	if i.gitBridge == nil {
		return report, nil
	}

	repoPath := i.gitBridge.RepoPath()
	if repoPath == "" {
		return report, nil
	}

	// For each asset type, check consistency
	for _, assetType := range assetTypes {
		// Use plural form for directory names (assets/skills/, skills/, not assets/skill/, skill/)
		assetsDir := filepath.Join(repoPath, "assets", assetType+"s")
		contentDir := filepath.Join(repoPath, assetType+"s")

		// Check if directories exist
		assetsInfo, assetsErr := os.Stat(assetsDir)
		contentInfo, contentErr := os.Stat(contentDir)

		if assetsErr != nil && !os.IsNotExist(assetsErr) {
			continue // Error, skip
		}
		if contentErr != nil && !os.IsNotExist(contentErr) {
			continue // Error, skip
		}

		// Get list of YAML files and content folders
		var yamlIDs []string
		var contentFolders []string

		if assetsInfo != nil && assetsInfo.IsDir() {
			entries, _ := os.ReadDir(assetsDir)
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
					id := strings.TrimSuffix(entry.Name(), ".yaml")
					yamlIDs = append(yamlIDs, id)
				}
			}
		}

		if contentInfo != nil && contentInfo.IsDir() {
			entries, _ := os.ReadDir(contentDir)
			for _, entry := range entries {
				if entry.IsDir() {
					contentFolders = append(contentFolders, entry.Name())
				}
			}
		}

		// Check for orphan folders (in content dir but no YAML)
		yamlSet := make(map[string]bool)
		for _, id := range yamlIDs {
			yamlSet[id] = true
		}

		for _, folder := range contentFolders {
			if !yamlSet[folder] {
				// Use plural form for the path (skills/calculator, not skill/calculator)
				report.OrphanFolders = append(report.OrphanFolders, filepath.Join(assetType+"s", folder))
			}
		}

		// Check for missing files (YAML exists but main file doesn't)
		for _, id := range yamlIDs {
			ayPath := filepath.Join(assetsDir, id+".yaml")
			content, err := os.ReadFile(ayPath)
			if err != nil {
				continue
			}

			ay, err := domain.ParseAssetYAML(string(content))
			if err != nil {
				continue
			}

			mainResolved, isExternal, _ := ay.ResolveMain(repoPath)
			if !isExternal {
				mainPath := filepath.Join(repoPath, mainResolved)
				if _, err := os.Stat(mainPath); os.IsNotExist(err) {
					report.MissingFiles = append(report.MissingFiles, mainResolved)
				}
			}
		}

		// Check for orphan YAMLs (YAML exists but no content folder)
		folderSet := make(map[string]bool)
		for _, folder := range contentFolders {
			folderSet[folder] = true
		}

		for _, id := range yamlIDs {
			if !folderSet[id] {
				// Check if main file is external
				ayPath := filepath.Join(assetsDir, id+".yaml")
				content, err := os.ReadFile(ayPath)
				if err != nil {
					continue
				}

				ay, err := domain.ParseAssetYAML(string(content))
				if err != nil {
					continue
				}

				mainResolved, _, _ := ay.ResolveMain(repoPath)
				if !strings.HasPrefix(mainResolved, "/") && !strings.HasPrefix(mainResolved, "~") {
					report.OrphanYAMLs = append(report.OrphanYAMLs, filepath.Join("assets", assetType, id+".yaml"))
				}
			}
		}
	}

	return report, nil
}

// ConsistencyReport contains the results of a consistency check.
type ConsistencyReport struct {
	OrphanFolders []string // Folders in content dir but no YAML
	MissingFiles []string  // YAML exists but main file doesn't exist
	OrphanYAMLs  []string // YAML exists but content folder doesn't exist (non-external)
}
