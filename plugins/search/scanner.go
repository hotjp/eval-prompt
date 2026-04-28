// Package search provides asset indexing and search functionality.
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

// Scan scans the source directory for assets to import.
// It detects asset types, generates asset.yaml files, and moves files
// to the appropriate type directories.
func (i *Indexer) Scan(ctx context.Context, source string) (*service.ScanResult, error) {
	result := &service.ScanResult{
		ScannedDirs: []string{},
		CreatedAssets: []string{},
		UpdatedAssets: []string{},
		Errors: []string{},
		Commits: make(map[string]string),
	}

	repoPath := i.gitBridge.RepoPath()
	if repoPath == "" {
		return nil, fmt.Errorf("repository not initialized")
	}

	// Resolve source path relative to repo root
	sourcePath := source
	if !filepath.IsAbs(sourcePath) {
		sourcePath = filepath.Join(repoPath, sourcePath)
	}

	// Check if source exists
	info, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("source directory does not exist: %s", source)
		}
		return nil, fmt.Errorf("stat source directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("source is not a directory: %s", source)
	}

	// Scan all subdirectories in the source
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("read source directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Process directory
			if err := i.importDirectory(ctx, sourcePath, repoPath, entry.Name(), result); err != nil {
				result.Errors = append(result.Errors, err.Error())
			}
			continue
		}

		// Process single file directly placed in .import/
		if err := i.importSingleFile(ctx, sourcePath, repoPath, entry.Name(), result); err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	return result, nil
}

// assetTypeDir returns the plural directory name for an asset type.
func assetTypeDir(assetType string) string {
	switch assetType {
	case "prompt":
		return "prompts"
	case "skill":
		return "skills"
	case "workflow":
		return "workflows"
	case "agent":
		return "agents"
	case "knowledge":
		return "knowledges"
	case "system":
		return "systems"
	case "tool":
		return "tools"
	default:
		return assetType + "s"
	}
}

// importDirectory imports a single directory from the source path.
func (i *Indexer) importDirectory(ctx context.Context, sourcePath, repoPath, dirName string, result *service.ScanResult) error {
	result.ScannedDirs = append(result.ScannedDirs, dirName)

	// Detect asset type from directory contents
	assetType, mainFile, err := i.detectAssetType(filepath.Join(sourcePath, dirName))
	if err != nil {
		return fmt.Errorf("detect type for %s: %v", dirName, err)
	}

	// Generate asset.yaml
	ay := i.generateAssetYAML(dirName, assetType, mainFile)

	// Determine destination paths
	assetDir := fmt.Sprintf("%s/%s", assetTypeDir(assetType), dirName) // e.g., "prompts/my-prompt"
	assetYAMLPath := filepath.Join("assets", assetTypeDir(assetType), fmt.Sprintf("%s.yaml", dirName))

	// Check if asset.yaml already exists (update) or is new (create)
	existingAY, err := i.GetAssetYAML(ctx, assetYAMLPath)
	if err == nil && existingAY != nil {
		// Update existing - merge with new data but preserve metadata
		ay.Metadata = existingAY.Metadata
		result.UpdatedAssets = append(result.UpdatedAssets, dirName)
	} else {
		result.CreatedAssets = append(result.CreatedAssets, dirName)
	}

	// Move files from source to destination
	sourceDir := filepath.Join(sourcePath, dirName)
	destDir := filepath.Join(repoPath, assetDir)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dest dir %s: %v", destDir, err)
	}

	// Move all files from source to destination
	if err := i.moveDirectoryContents(sourceDir, destDir); err != nil {
		return fmt.Errorf("move files for %s: %v", dirName, err)
	}

	// Remove empty source directory
	os.Remove(sourceDir)

	// Update main path in asset.yaml to reflect new location
	ay.Main = filepath.Join(assetDir, filepath.Base(mainFile))

	// Save asset.yaml
	commitMsg := fmt.Sprintf("Import %s %s", ay.AssetType, ay.Name)
	commitHash, err := i.SaveAssetYAML(ctx, assetYAMLPath, ay, commitMsg)
	if err != nil {
		return fmt.Errorf("save asset.yaml for %s: %v", dirName, err)
	}

	result.Commits[dirName] = commitHash
	return nil
}

// importSingleFile imports a single file directly placed in the import directory.
func (i *Indexer) importSingleFile(ctx context.Context, sourcePath, repoPath, fileName string, result *service.ScanResult) error {
	assetType, mainFile, assetID, err := i.detectAssetTypeFromFile(fileName)
	if err != nil {
		return fmt.Errorf("detect type for %s: %v", fileName, err)
	}

	result.ScannedDirs = append(result.ScannedDirs, fileName)

	// Generate asset.yaml
	ay := i.generateAssetYAML(assetID, assetType, mainFile)

	// Determine destination paths
	assetDir := fmt.Sprintf("%s/%s", assetTypeDir(assetType), assetID)
	assetYAMLPath := filepath.Join("assets", assetTypeDir(assetType), fmt.Sprintf("%s.yaml", assetID))

	// Check if asset.yaml already exists (update) or is new (create)
	existingAY, err := i.GetAssetYAML(ctx, assetYAMLPath)
	if err == nil && existingAY != nil {
		ay.Metadata = existingAY.Metadata
		result.UpdatedAssets = append(result.UpdatedAssets, assetID)
	} else {
		result.CreatedAssets = append(result.CreatedAssets, assetID)
	}

	// Create destination directory and move the single file
	destDir := filepath.Join(repoPath, assetDir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create dest dir %s: %v", destDir, err)
	}

	srcFile := filepath.Join(sourcePath, fileName)
	dstFile := filepath.Join(destDir, fileName)
	if err := i.moveFile(srcFile, dstFile); err != nil {
		return fmt.Errorf("move file %s: %v", fileName, err)
	}

	// Update main path in asset.yaml to reflect new location
	ay.Main = filepath.Join(assetDir, fileName)

	// Save asset.yaml
	commitMsg := fmt.Sprintf("Import %s %s", ay.AssetType, ay.Name)
	commitHash, err := i.SaveAssetYAML(ctx, assetYAMLPath, ay, commitMsg)
	if err != nil {
		return fmt.Errorf("save asset.yaml for %s: %v", fileName, err)
	}

	result.Commits[assetID] = commitHash
	return nil
}

// detectAssetTypeFromFile detects asset type from a single file name/extension.
func (i *Indexer) detectAssetTypeFromFile(fileName string) (assetType, mainFile, assetID string, err error) {
	ext := strings.ToLower(filepath.Ext(fileName))
	base := strings.TrimSuffix(fileName, ext)

	switch ext {
	case ".py":
		return "skill", fileName, base, nil
	case ".yaml", ".yml":
		return "workflow", fileName, base, nil
	case ".md":
		return "prompt", fileName, base, nil
	default:
		return "", "", "", fmt.Errorf("unsupported file extension '%s' for file: %s", ext, fileName)
	}
}

// moveFile moves a single file from src to dst using copy+delete.
func (i *Indexer) moveFile(srcPath, destPath string) error {
	if _, err := os.Stat(destPath); err == nil {
		if err := os.Remove(destPath); err != nil {
			return fmt.Errorf("remove existing dest: %w", err)
		}
	}

	input, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	if err := os.WriteFile(destPath, input, 0644); err != nil {
		return fmt.Errorf("write dest: %w", err)
	}
	if err := os.Remove(srcPath); err != nil {
		return fmt.Errorf("remove source: %w", err)
	}
	return nil
}

// detectAssetType detects the asset type from the directory contents.
// Returns the asset type, main file path, and error.
func (i *Indexer) detectAssetType(dirPath string) (string, string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", "", fmt.Errorf("read directory: %w", err)
	}

	// Track files to find main
	var pyFiles []string
	var mdFiles []string
	var yamlFiles []string
	var hasInitPy bool

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		if name == "__init__.py" {
			hasInitPy = true
		} else if strings.HasSuffix(name, ".py") {
			pyFiles = append(pyFiles, name)
		} else if strings.HasSuffix(name, ".md") {
			mdFiles = append(mdFiles, name)
		} else if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			yamlFiles = append(yamlFiles, name)
		}
	}

	// Determine type based on patterns
	// skill: contains __init__.py or handler.py or other .py files
	if hasInitPy || hasFile(pyFiles, "handler.py") || len(pyFiles) > 0 {
		// If has __init__.py, use it as main file
		if hasInitPy {
			return "skill", "__init__.py", nil
		}
		mainFile := findMainFile(pyFiles, "handler.py", "process.py", "run.py", "__main__.py")
		return "skill", mainFile, nil
	}

	// workflow/mcp: contains .yaml files
	if len(yamlFiles) > 0 {
		mainFile := findMainFile(yamlFiles, "workflow.yaml", "mcp.yaml", "config.yaml")
		return "workflow", mainFile, nil
	}

	// agent/prompt: contains .md files
	if len(mdFiles) > 0 {
		mainFile := findMainFile(mdFiles, "agent.md", "prompt.md", "main.md", "overview.md")
		return "prompt", mainFile, nil
	}

	// Default to prompt with first .md file
	if len(mdFiles) > 0 {
		return "prompt", mdFiles[0], nil
	}

	return "", "", fmt.Errorf("could not detect asset type for directory: %s", dirPath)
}

// hasFile checks if a file with the given name exists in the list.
func hasFile(files []string, name string) bool {
	for _, f := range files {
		if f == name {
			return true
		}
	}
	return false
}

// findMainFile finds the main file from a list of files.
// It prefers files with specific names in order.
func findMainFile(files []string, preferredNames ...string) string {
	// First, try to find by preferred names
	for _, preferred := range preferredNames {
		for _, f := range files {
			if f == preferred {
				return f
			}
		}
	}
	// Default to first file
	if len(files) > 0 {
		return files[0]
	}
	return ""
}

// generateAssetYAML creates a new AssetYAML from detected information.
func (i *Indexer) generateAssetYAML(id, assetType, mainFile string) *domain.AssetYAML {
	ay := domain.NewAssetYAML(assetType, formatName(id), mainFile)

	// Set default state and category
	ay.State = "draft"
	ay.Category = "content"

	// Build files list
	ay.Files = []domain.FileEntry{
		{Path: mainFile, Role: "main"},
	}

	return ay
}

// formatName formats an ID into a human-readable name.
func formatName(id string) string {
	// Replace hyphens and underscores with spaces
	name := strings.ReplaceAll(id, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	// Capitalize words
	words := strings.Split(name, " ")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// moveDirectoryContents moves all files from srcDir to destDir.
func (i *Indexer) moveDirectoryContents(srcDir, destDir string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read src directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Recursively move subdirectories
			srcPath := filepath.Join(srcDir, entry.Name())
			destPath := filepath.Join(destDir, entry.Name())
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("create subdirectory: %w", err)
			}
			if err := i.moveDirectoryContents(srcPath, destPath); err != nil {
				return err
			}
			os.Remove(srcPath)
		} else {
			// Move file using copy+delete (rename may fail across filesystems)
			srcPath := filepath.Join(srcDir, entry.Name())
			destPath := filepath.Join(destDir, entry.Name())
			if err := i.moveFile(srcPath, destPath); err != nil {
				return fmt.Errorf("move file %s: %w", entry.Name(), err)
			}
		}
	}

	// Remove the source directory itself after all contents have been moved
	if err := os.Remove(srcDir); err != nil {
		return fmt.Errorf("remove source directory: %w", err)
	}

	return nil
}

// GetAssetYAML reads and parses an asset.yaml file.
func (i *Indexer) GetAssetYAML(ctx context.Context, assetPath string) (*domain.AssetYAML, error) {
	repoPath := i.gitBridge.RepoPath()
	if repoPath == "" {
		return nil, fmt.Errorf("repository not initialized")
	}

	fullPath := filepath.Join(repoPath, assetPath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read asset.yaml: %w", err)
	}

	return domain.ParseAssetYAML(string(content))
}

// SaveAssetYAML writes an AssetYAML to disk and commits it to Git.
func (i *Indexer) SaveAssetYAML(ctx context.Context, assetPath string, ay *domain.AssetYAML, commitMsg string, extraFiles ...string) (string, error) {
	if i.gitBridge == nil {
		return "", fmt.Errorf("git bridge not configured")
	}
	repoPath := i.gitBridge.RepoPath()
	if repoPath == "" {
		return "", fmt.Errorf("repository not initialized")
	}

	// Ensure directory exists
	fullPath := filepath.Join(repoPath, assetPath)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	// Serialize AssetYAML
	yamlContent, err := domain.SerializeAssetYAML(ay)
	if err != nil {
		return "", fmt.Errorf("serialize asset yaml: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(yamlContent), 0644); err != nil {
		return "", fmt.Errorf("write asset.yaml: %w", err)
	}

	// Git add + commit
	var commitHash string
	if len(extraFiles) > 0 {
		files := append([]string{assetPath}, extraFiles...)
		var commitErr error
		commitHash, commitErr = i.gitBridge.StageAndCommitFiles(ctx, files, commitMsg)
		if commitErr != nil {
			return "", fmt.Errorf("git commit: %w", commitErr)
		}
	} else {
		var commitErr error
		commitHash, commitErr = i.gitBridge.StageAndCommit(ctx, assetPath, commitMsg)
		if commitErr != nil {
			return "", fmt.Errorf("git commit: %w", commitErr)
		}
	}

	return commitHash, nil
}

// MoveAssetFiles moves files from sourceDir to destDir and stages them in Git.
// Both paths are relative to the repo root.
func (i *Indexer) MoveAssetFiles(ctx context.Context, sourceDir, destDir string) error {
	repoPath := i.gitBridge.RepoPath()
	if repoPath == "" {
		return fmt.Errorf("repository not initialized")
	}

	srcPath := filepath.Join(repoPath, sourceDir)
	dstPath := filepath.Join(repoPath, destDir)

	// Create destination directory
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		return fmt.Errorf("create destination directory: %w", err)
	}

	// Move contents
	if err := i.moveDirectoryContents(srcPath, dstPath); err != nil {
		return fmt.Errorf("move directory contents: %w", err)
	}

	// Remove empty source directory
	os.Remove(srcPath)

	return nil
}

// GetRepoPath returns the repository path.
func (i *Indexer) GetRepoPath() string {
	if i.gitBridge == nil {
		return ""
	}
	return i.gitBridge.RepoPath()
}
